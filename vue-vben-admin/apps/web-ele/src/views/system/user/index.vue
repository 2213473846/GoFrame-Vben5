<script lang="ts" setup>
import type { SystemUserListItem } from '#/api/system/user';
import type { DictFallbackItem } from '#/composables/useDict';

import { computed, onMounted, reactive, ref } from 'vue';

import { useUserStore } from '@vben/stores';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElSelect,
  ElSwitch,
  ElTable,
  ElTableColumn,
  ElTag,
  ElUpload,
} from 'element-plus';

import {
  createUserApi,
  deleteUserApi,
  getRecycleListApi,
  getUserListApi,
  purgeUserApi,
  restoreUserApi,
  updateUserApi,
} from '#/api/system/user';
import type { RecycleItem } from '#/api/system/user';
import { getRoleListApi } from '#/api/system/role';
import type { SystemRoleItem } from '#/api/system/role';
import { downloadUserImportTemplateApi, importUsersApi } from '#/api/system/import';
import type { ImportResult } from '#/api/system/import';
import { downloadExportApi } from '#/api/system/export';
import DictTag from '#/components/dict-tag/index.vue';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'SystemUser' });

// 用户状态字典兜底文案(字典未配置时回退), 展示交给 DictTag 组件
const userStatusFallback: Record<string, DictFallbackItem> = {
  '0': { label: '禁用', tagColor: 'danger' },
  '1': { label: '启用', tagColor: 'success' },
};

const userStore = useUserStore();
// 最大在线设备数仅超级管理员可查看/编辑; 接口层同样校验, 前端隐藏只是体验优化
const isSuperAdmin = computed(
  () => userStore.userInfo?.roles?.includes('super_admin') ?? false,
);

const roleOptions = [
  { label: '管理员', value: 'admin' },
  { label: '用户', value: 'user' },
];

const roleTagType = (role: string) => {
  if (role === 'super_admin') return 'danger';
  if (role === 'admin') return 'warning';
  return 'info';
};

const roleText = (role: string) => {
  if (role === 'super_admin') return '超级管理员';
  if (role === 'admin') return '管理员';
  return '用户';
};

// 列表状态(分页/加载/查询条件由 useTable 统一管理)
const {
  fetchList,
  handlePageChange,
  handleReset,
  handleSearch,
  items,
  loading,
  query,
  total,
} = useTable<SystemUserListItem, { role: string; username: string }>({
  async fetch(q) {
    return getUserListApi({
      page: q.page,
      pageSize: q.pageSize,
      role: q.role || undefined,
      username: q.username || undefined,
    });
  },
  defaultQuery: { role: '', username: '' },
});

// 导出当前筛选条件下的用户列表(后端复用列表查询与数据权限)
const exporting = ref(false);
async function handleExport(format: 'csv' | 'xlsx') {
  exporting.value = true;
  try {
    await downloadExportApi('/user/export', {
      format,
      params: {
        role: query.role || undefined,
        username: query.username || undefined,
      },
    });
    ElMessage.success('导出已开始');
  } finally {
    exporting.value = false;
  }
}

// Excel 导入(部分成功; 默认不回滚, 可勾选全量回滚)
const importVisible = ref(false);
const importLoading = ref(false);
const importFile = ref<File>();
const importAtomic = ref(false);
const importResult = ref<ImportResult>();

function openImport() {
  importFile.value = undefined;
  importAtomic.value = false;
  importResult.value = undefined;
  importVisible.value = true;
}

function handleImportFileChange(uploadFile: any) {
  importFile.value = uploadFile?.raw;
  importResult.value = undefined;
}

function handleImportFileRemove() {
  importFile.value = undefined;
}

async function handleDownloadTemplate() {
  await downloadUserImportTemplateApi();
  ElMessage.success('模板已开始下载');
}

async function handleImport() {
  if (!importFile.value) {
    ElMessage.warning('请先选择 Excel 文件');
    return;
  }
  importLoading.value = true;
  try {
    const res = await importUsersApi(importFile.value, importAtomic.value);
    importResult.value = res;
    if (res.failed === 0) {
      ElMessage.success(`导入完成: 成功 ${res.success} 条`);
    } else {
      ElMessage.warning(
        `导入完成: 成功 ${res.success} 条, 失败 ${res.failed} 条`,
      );
    }
    fetchList();
  } finally {
    importLoading.value = false;
  }
}

// 新增/编辑对话框
const dialogVisible = ref(false);
const saving = ref(false);
const isEdit = ref(false);
const form = reactive({
  id: 0,
  maxDevices: 0,
  password: '',
  realName: '',
  remark: '',
  role: 'user',
  roleIds: [] as number[],
  status: 1,
  username: '',
});

// 功能角色(RBAC)选项
const rbacRoleOptions = ref<SystemRoleItem[]>([]);

async function fetchRoles() {
  try {
    const res = await getRoleListApi();
    rbacRoleOptions.value = (res as { items: SystemRoleItem[] }).items ?? [];
  } catch {
    // 无权限或加载失败时保持空选项, 保存时后端会校验
    rbacRoleOptions.value = [];
  }
}

function openCreate() {
  isEdit.value = false;
  Object.assign(form, {
    id: 0,
    maxDevices: 0,
    password: '',
    realName: '',
    remark: '',
    role: 'user',
    roleIds: [],
    status: 1,
    username: '',
  });
  dialogVisible.value = true;
}

function openEdit(row: SystemUserListItem) {
  isEdit.value = true;
  Object.assign(form, {
    id: row.id,
    maxDevices: row.maxDevices ?? 0,
    password: '',
    realName: row.realName,
    remark: row.remark,
    role: row.role,
    roleIds: [...(row.roleIds ?? [])],
    status: row.status,
    username: row.username,
  });
  // 用户已有但不在当前可选列表中的角色(如超管分配的), 补充为不可选展示项, 避免下拉框显示原始ID
  const known = new Set(rbacRoleOptions.value.map((r) => r.id));
  const extraIds = form.roleIds.filter((id) => !known.has(id));
  const extraNames = row.roleNames ?? [];
  for (const id of extraIds) {
    const idx = row.roleIds?.indexOf(id) ?? -1;
    const name = extraNames[idx] || `角色#${id}`;
    rbacRoleOptions.value.push({
      code: '',
      createdAt: '',
      id,
      name: `${name} (不可分配)`,
      remark: '',
      status: 1,
    });
    known.add(id);
  }
  dialogVisible.value = true;
}

async function handleSave() {
  saving.value = true;
  try {
    if (isEdit.value) {
      await updateUserApi({
        id: form.id,
        // 仅超管提交该字段; 其他角色由后端强制拦截
        ...(isSuperAdmin.value ? { maxDevices: form.maxDevices } : {}),
        password: form.password || undefined,
        realName: form.realName,
        remark: form.remark,
        roleIds: form.roleIds,
        status: form.status,
        username: form.username || undefined,
      });
      ElMessage.success('更新成功');
    } else {
      if (!form.username) {
        ElMessage.warning('请输入用户名');
        return;
      }
      if (!form.password) {
        ElMessage.warning('请输入初始密码');
        return;
      }
      await createUserApi({
        ...(isSuperAdmin.value ? { maxDevices: form.maxDevices } : {}),
        password: form.password,
        realName: form.realName,
        remark: form.remark,
        role: form.role,
        roleIds: form.roleIds,
        username: form.username,
      });
      ElMessage.success('创建成功');
    }
    dialogVisible.value = false;
    fetchList();
  } finally {
    saving.value = false;
  }
}

async function handleDelete(row: SystemUserListItem) {
  const confirmed = await ElMessageBox.confirm(
    row.role === 'admin'
      ? `删除管理员「${row.username}」将同时删除其所有下级, 是否继续?`
      : `确认删除用户「${row.username}」吗?`,
    '删除确认',
    { type: 'warning' },
  ).catch(() => false);
  if (!confirmed) return;
  await deleteUserApi(row.id);
  ElMessage.success('已移入回收站');
  fetchList();
}

// 回收站
const recycleVisible = ref(false);
const recycleLoading = ref(false);
const recycleItems = ref<RecycleItem[]>([]);
const recycleSelection = ref<RecycleItem[]>([]);
const recycleQuery = reactive({ username: '' });

function handleRecycleSelect(rows: RecycleItem[]) {
  recycleSelection.value = rows;
}

async function fetchRecycle() {
  recycleLoading.value = true;
  try {
    const res = await getRecycleListApi({
      page: 1,
      pageSize: 100,
      username: recycleQuery.username || undefined,
    });
    recycleItems.value = (res as { items: RecycleItem[] }).items;
  } finally {
    recycleLoading.value = false;
  }
}

function openRecycle() {
  recycleVisible.value = true;
  recycleQuery.username = '';
  fetchRecycle();
}

async function handleRestore(row: RecycleItem) {
  await restoreUserApi(row.id);
  ElMessage.success(`已恢复「${row.username}」`);
  fetchRecycle();
  fetchList();
}

async function handlePurge(row: RecycleItem) {
  const confirmed = await ElMessageBox.confirm(
    `彻底删除「${row.username}」后数据将从数据库物理移除, 不可恢复! 是否继续?`,
    '彻底删除确认',
    { type: 'error', confirmButtonText: '彻底删除' },
  ).catch(() => false);
  if (!confirmed) return;
  await purgeUserApi(row.id);
  ElMessage.success('已彻底删除');
  fetchRecycle();
}

async function handleBatchRestore() {
  if (recycleSelection.value.length === 0) return;
  let ok = 0;
  const failed: string[] = [];
  for (const row of recycleSelection.value) {
    try {
      await restoreUserApi(row.id);
      ok += 1;
    } catch (error: any) {
      failed.push(`${row.username}(${error?.message || '失败'})`);
    }
  }
  if (ok > 0) ElMessage.success(`成功恢复 ${ok} 个账号`);
  if (failed.length > 0) ElMessage.warning(`恢复失败: ${failed.join(', ')}`);
  fetchRecycle();
  fetchList();
}

async function handleBatchPurge() {
  if (recycleSelection.value.length === 0) return;
  const count = recycleSelection.value.length;
  const confirmed = await ElMessageBox.confirm(
    `彻底删除选中的 ${count} 个账号, 数据将从数据库物理移除且不可恢复! 是否继续?`,
    '批量彻底删除确认',
    { type: 'error', confirmButtonText: '彻底删除' },
  ).catch(() => false);
  if (!confirmed) return;
  let ok = 0;
  const failed: string[] = [];
  for (const row of recycleSelection.value) {
    try {
      await purgeUserApi(row.id);
      ok += 1;
    } catch (error: any) {
      failed.push(`${row.username}(${error?.message || '失败'})`);
    }
  }
  if (ok > 0) ElMessage.success(`已彻底删除 ${ok} 个账号`);
  if (failed.length > 0) ElMessage.warning(`删除失败: ${failed.join(', ')}`);
  fetchRecycle();
}

onMounted(() => {
  fetchRoles();
});
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex flex-wrap items-center gap-2">
      <ElInput
        v-model="query.username"
        placeholder="用户名"
        clearable
        class="w-44"
        @keyup.enter="handleSearch"
      />
      <ElSelect
        v-model="query.role"
        placeholder="角色"
        clearable
        class="w-36"
      >
        <ElOption
          v-for="opt in roleOptions"
          :key="opt.value"
          :label="opt.label"
          :value="opt.value"
        />
      </ElSelect>
      <ElButton type="primary" @click="handleSearch">查询</ElButton>
      <ElButton @click="handleReset">重置</ElButton>
      <div class="flex-1"></div>
      <ElButton :loading="exporting" @click="handleExport('xlsx')">
        导出 Excel
      </ElButton>
      <ElButton :loading="exporting" @click="handleExport('csv')">
        导出 CSV
      </ElButton>
      <ElButton @click="openRecycle">回收站</ElButton>
      <ElButton @click="openImport">导入</ElButton>
      <ElButton type="primary" @click="openCreate">新增</ElButton>
    </div>

    <ElTable v-loading="loading" :data="items" border stripe>
      <ElTableColumn prop="username" label="用户名" min-width="120" />
      <ElTableColumn prop="realName" label="昵称" min-width="120" />
      <ElTableColumn label="上级" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.parentUsername">
            {{ row.parentUsername }}
            <span v-if="row.parentRealName" class="text-gray-400">
              ({{ row.parentRealName }})
            </span>
          </span>
          <span v-else class="text-gray-400">顶级</span>
        </template>
      </ElTableColumn>
      <ElTableColumn label="角色" width="110">
        <template #default="{ row }">
          <ElTag :type="roleTagType(row.role)">{{ roleText(row.role) }}</ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn label="功能角色" min-width="160">
        <template #default="{ row }">
          <template v-if="row.roleNames && row.roleNames.length > 0">
            <ElTag
              v-for="name in row.roleNames"
              :key="name"
              size="small"
              class="mr-1"
            >
              {{ name }}
            </ElTag>
          </template>
          <span v-else class="text-gray-400">—</span>
        </template>
      </ElTableColumn>
      <ElTableColumn label="状态" width="90">
        <template #default="{ row }">
          <DictTag
            type="sys_user_status"
            :value="row.status"
            :fallback="userStatusFallback"
          />
        </template>
      </ElTableColumn>
      <ElTableColumn v-if="isSuperAdmin" label="最大设备数" width="110">
        <template #default="{ row }">
          <span v-if="row.maxDevices">{{ row.maxDevices }}</span>
          <span v-else class="text-gray-400">全局默认</span>
        </template>
      </ElTableColumn>
      <ElTableColumn prop="createdAt" label="创建时间" width="170" />
      <ElTableColumn label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <ElButton link type="primary" @click="openEdit(row as SystemUserListItem)">
            编辑
          </ElButton>
          <ElButton link type="danger" @click="handleDelete(row as SystemUserListItem)">
            删除
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <div class="mt-4 flex justify-end">
      <ElButton
        :disabled="query.page <= 1"
        @click="handlePageChange(query.page - 1)"
        >上一页</ElButton
      >
      <span class="mx-3 leading-[32px]">
        第 {{ query.page }} / {{ Math.max(1, Math.ceil(total / query.pageSize)) }} 页, 共
        {{ total }} 条
      </span>
      <ElButton
        :disabled="query.page >= Math.ceil(total / query.pageSize)"
        @click="handlePageChange(query.page + 1)"
        >下一页</ElButton
      >
    </div>

    <ElDialog
      v-model="recycleVisible"
      title="回收站 (已删除账号)"
      width="760px"
      destroy-on-close
    >
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <ElInput
          v-model="recycleQuery.username"
          placeholder="用户名搜索"
          clearable
          class="w-44"
          @keyup.enter="fetchRecycle"
        />
        <ElButton type="primary" @click="fetchRecycle">搜索</ElButton>
        <div class="flex-1"></div>
        <ElButton
          :disabled="recycleSelection.length === 0"
          @click="handleBatchRestore"
          >批量恢复</ElButton
        >
        <ElButton
          type="danger"
          :disabled="recycleSelection.length === 0"
          @click="handleBatchPurge"
          >批量彻底删除</ElButton
        >
      </div>
      <ElTable
        v-loading="recycleLoading"
        :data="recycleItems"
        border
        size="small"
        @selection-change="handleRecycleSelect"
      >
        <ElTableColumn type="selection" width="45" />
        <ElTableColumn prop="username" label="用户名" min-width="110" />
        <ElTableColumn prop="realName" label="昵称" min-width="110" />
        <ElTableColumn label="角色" width="100">
          <template #default="{ row }">
            <ElTag :type="roleTagType(row.role)" size="small">
              {{ roleText(row.role) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作人" width="110">
          <template #default="{ row }">
            {{ row.operator || '—' }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="deletedAt" label="删除时间" width="155" />
        <ElTableColumn label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <ElButton link type="primary" @click="handleRestore(row as RecycleItem)">
              恢复
            </ElButton>
            <ElButton link type="danger" @click="handlePurge(row as RecycleItem)">
              彻底删除
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
      <div class="mt-2 text-xs text-gray-400">
        删除的账号在此保留, 可恢复; 彻底删除将从数据库物理移除且不可恢复。
      </div>
    </ElDialog>

    <ElDialog
      v-model="dialogVisible"
      :title="isEdit ? `编辑: ${form.username}` : '新增用户'"
      width="480px"
      destroy-on-close
    >
      <ElForm label-width="80px">
        <ElFormItem label="用户名" :required="!isEdit">
          <ElInput
            v-model="form.username"
            :placeholder="isEdit ? '可修改, 3-32位' : '3-32位'"
            maxlength="32"
          />
        </ElFormItem>
        <ElFormItem v-if="!isEdit" label="角色" required>
          <ElSelect v-model="form.role" class="w-full">
            <ElOption
              v-for="opt in roleOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="功能角色">
          <ElSelect
            v-model="form.roleIds"
            multiple
            clearable
            placeholder="可选, 决定用户可用的菜单"
            class="w-full"
          >
            <ElOption
              v-for="r in rbacRoleOptions"
              :key="r.id"
              :label="r.name"
              :value="r.id"
              :disabled="r.code === ''"
            />
          </ElSelect>
          <div class="w-full text-xs text-gray-400">
            功能角色决定"能用哪些菜单", 与上方数据权限角色相互独立; 灰色选项为超管分配,
            不可在此移除
          </div>
        </ElFormItem>
        <ElFormItem label="昵称">
          <ElInput v-model="form.realName" maxlength="64" />
        </ElFormItem>
        <ElFormItem
          :label="isEdit ? '重置密码' : '密码'"
          :required="!isEdit"
        >
          <ElInput
            v-model="form.password"
            type="password"
            show-password
            :placeholder="isEdit ? '留空则不修改' : '6-32位'"
            maxlength="32"
          />
        </ElFormItem>
        <ElFormItem v-if="isEdit" label="状态">
          <ElSelect v-model="form.status" class="w-full">
            <ElOption :value="1" label="启用" />
            <ElOption :value="0" label="禁用" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem v-if="isSuperAdmin" label="最大设备数">
          <ElInputNumber
            v-model="form.maxDevices"
            :min="0"
            :max="999"
            class="w-full"
          />
          <div class="w-full text-xs text-gray-400">
            该账号允许同时在线的设备数, 0 表示使用全局默认值; 超出上限时
            最早登录的设备会被顶下线
          </div>
        </ElFormItem>
        <ElFormItem label="备注">
          <ElInput v-model="form.remark" maxlength="255" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="handleSave">
          保存
        </ElButton>
      </template>
    </ElDialog>

    <!-- Excel 导入 -->
    <ElDialog
      v-model="importVisible"
      title="Excel 导入用户"
      width="720px"
      destroy-on-close
    >
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <ElButton @click="handleDownloadTemplate">下载模板</ElButton>
        <span class="text-xs text-gray-400">
          模板含表头、说明行与示例行; 需填用户名、密码、角色(管理员/用户)、状态(启用/禁用),
          {{ isSuperAdmin ? '可选填最大设备数' : '最大设备数仅超级管理员可设置' }}
        </span>
      </div>

      <ElUpload
        drag
        :auto-upload="false"
        :limit="1"
        accept=".xlsx"
        :on-change="handleImportFileChange"
        :on-remove="handleImportFileRemove"
      >
        <div class="p-4 text-sm">
          将 .xlsx 文件拖到此处, 或
          <span class="text-primary">点击选择</span>
        </div>
      </ElUpload>

      <div class="mt-3 flex flex-wrap items-center gap-2">
        <ElSwitch v-model="importAtomic" />
        <span class="text-sm">全量回滚(勾选后任一行失败则全部不写入)</span>
        <span class="text-xs text-gray-400">默认部分成功</span>
      </div>

      <div v-if="importResult" class="mt-4">
        <div class="mb-2 text-sm font-medium">
          导入结果: 总 {{ importResult.total }} 条, 成功
          {{ importResult.success }} 条, 失败 {{ importResult.failed }} 条
          <ElTag
            v-if="importResult.atomic"
            size="small"
            type="warning"
            class="ml-1"
          >
            全量回滚
          </ElTag>
        </div>
        <ElTable
          v-if="importResult.errors && importResult.errors.length > 0"
          :data="importResult.errors"
          border
          size="small"
          max-height="260"
        >
          <ElTableColumn prop="row" label="行号" width="80" />
          <ElTableColumn prop="username" label="用户名" width="140" />
          <ElTableColumn prop="reason" label="失败原因" min-width="260" />
        </ElTable>
        <div v-else class="text-sm text-emerald-600">全部导入成功</div>
      </div>

      <template #footer>
        <ElButton @click="importVisible = false">关闭</ElButton>
        <ElButton type="primary" :loading="importLoading" @click="handleImport">
          开始导入
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<style scoped>
:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  text-align: center;
}
</style>
