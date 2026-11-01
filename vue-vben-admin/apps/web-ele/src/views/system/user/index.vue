<script lang="ts" setup>
import type {
  SystemUserListItem,
  SystemUserListResult,
} from '#/api/system/user';

import { onMounted, reactive, ref } from 'vue';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
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

defineOptions({ name: 'SystemUser' });

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

// 列表状态
const loading = ref(false);
const items = ref<SystemUserListItem[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 10,
  username: '',
  role: '',
});

async function fetchList() {
  loading.value = true;
  try {
    const res = await getUserListApi({
      page: query.page,
      pageSize: query.pageSize,
      role: query.role || undefined,
      username: query.username || undefined,
    });
    const data = res as SystemUserListResult;
    items.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  query.page = 1;
  fetchList();
}

function handlePageChange(page: number) {
  query.page = page;
  fetchList();
}

// 新增/编辑对话框
const dialogVisible = ref(false);
const saving = ref(false);
const isEdit = ref(false);
const form = reactive({
  id: 0,
  password: '',
  realName: '',
  remark: '',
  role: 'user',
  status: 1,
  username: '',
});

function openCreate() {
  isEdit.value = false;
  Object.assign(form, {
    id: 0,
    password: '',
    realName: '',
    remark: '',
    role: 'user',
    status: 1,
    username: '',
  });
  dialogVisible.value = true;
}

function openEdit(row: SystemUserListItem) {
  isEdit.value = true;
  Object.assign(form, {
    id: row.id,
    password: '',
    realName: row.realName,
    remark: row.remark,
    role: row.role,
    status: row.status,
    username: row.username,
  });
  dialogVisible.value = true;
}

async function handleSave() {
  saving.value = true;
  try {
    if (isEdit.value) {
      await updateUserApi({
        id: form.id,
        password: form.password || undefined,
        realName: form.realName,
        remark: form.remark,
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
        password: form.password,
        realName: form.realName,
        remark: form.remark,
        role: form.role,
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

onMounted(fetchList);
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
      <ElButton @click="() => { query.username = ''; query.role = ''; handleSearch(); }"
        >重置</ElButton
      >
      <div class="flex-1"></div>
      <ElButton @click="openRecycle">回收站</ElButton>
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
      <ElTableColumn label="状态" width="90">
        <template #default="{ row }">
          <ElTag :type="row.status === 1 ? 'success' : 'danger'">
            {{ row.status === 1 ? '启用' : '禁用' }}
          </ElTag>
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
  </div>
</template>

<style scoped>
:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  text-align: center;
}
</style>
