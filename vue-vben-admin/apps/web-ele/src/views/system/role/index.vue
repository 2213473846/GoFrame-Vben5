<script setup lang="ts">
import type { SystemRoleItem } from '#/api/system/role';

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
  ElTree,
} from 'element-plus';

import {
  createRoleApi,
  deleteRoleApi,
  getRoleGrantInfoApi,
  getRoleListApi,
  grantRoleApi,
  updateRoleApi,
} from '#/api/system/role';
import type { SystemMenuItem } from '#/api/system/menu';

defineOptions({ name: 'SystemRole' });

// 列表状态
const loading = ref(false);
const items = ref<SystemRoleItem[]>([]);

async function fetchList() {
  loading.value = true;
  try {
    const res = await getRoleListApi();
    items.value = (res as { items: SystemRoleItem[] }).items ?? [];
  } finally {
    loading.value = false;
  }
}

// 新增/编辑对话框
const dialogVisible = ref(false);
const saving = ref(false);
const isEdit = ref(false);
const form = reactive({
  code: '',
  id: 0,
  name: '',
  remark: '',
  status: 1,
});

function openCreate() {
  isEdit.value = false;
  Object.assign(form, { code: '', id: 0, name: '', remark: '', status: 1 });
  dialogVisible.value = true;
}

function openEdit(row: SystemRoleItem) {
  isEdit.value = true;
  Object.assign(form, {
    code: row.code,
    id: row.id,
    name: row.name,
    remark: row.remark,
    status: row.status,
  });
  dialogVisible.value = true;
}

async function handleSave() {
  if (!form.name) {
    ElMessage.warning('请输入角色名称');
    return;
  }
  saving.value = true;
  try {
    if (isEdit.value) {
      await updateRoleApi({
        code: form.code,
        id: form.id,
        name: form.name,
        remark: form.remark,
        status: form.status,
      });
      ElMessage.success('更新成功');
    } else {
      if (!form.code) {
        ElMessage.warning('请输入角色编码');
        return;
      }
      await createRoleApi({
        code: form.code,
        name: form.name,
        remark: form.remark,
        status: form.status,
      });
      ElMessage.success('创建成功');
    }
    dialogVisible.value = false;
    fetchList();
  } finally {
    saving.value = false;
  }
}

async function handleDelete(row: SystemRoleItem) {
  const confirmed = await ElMessageBox.confirm(
    `删除角色「${row.name}」将同时清理其菜单授权与用户关联, 是否继续?`,
    '删除确认',
    { type: 'warning' },
  ).catch(() => false);
  if (!confirmed) return;
  await deleteRoleApi(row.id);
  ElMessage.success('已删除');
  fetchList();
}

// 菜单授权对话框
const grantVisible = ref(false);
const grantLoading = ref(false);
const grantSaving = ref(false);
const grantRoleId = ref<number>();
const grantRoleName = ref('');
const grantTree = ref();
const grantMenus = ref<SystemMenuItem[]>([]);
const grantChecked = ref<number[]>([]);

async function openGrant(row: SystemRoleItem) {
  grantVisible.value = true;
  grantRoleId.value = row.id;
  grantRoleName.value = row.name;
  grantMenus.value = [];
  grantChecked.value = [];
  grantLoading.value = true;
  try {
    const res = await getRoleGrantInfoApi(row.id);
    grantMenus.value = res.items ?? [];
    grantChecked.value = res.checkedIds ?? [];
    grantTree.value?.setCheckedKeys(grantChecked.value, false);
  } finally {
    grantLoading.value = false;
  }
}

async function handleGrantSave() {
  if (!grantRoleId.value) return;
  grantSaving.value = true;
  try {
    const checked: number[] = grantTree.value?.getCheckedKeys() ?? [];
    await grantRoleApi(grantRoleId.value, checked);
    ElMessage.success(`已保存「${grantRoleName.value}」的菜单授权`);
    grantVisible.value = false;
  } finally {
    grantSaving.value = false;
  }
}

onMounted(fetchList);
</script>

<template>
  <div class="p-4">
    <div class="mb-4 flex flex-wrap items-center gap-2">
      <div class="text-sm text-gray-400">
        功能角色决定"能用哪些菜单", 与账号的数据权限分级(超管/管理员/用户)相互独立
      </div>
      <div class="flex-1"></div>
      <ElButton type="primary" @click="openCreate">新增</ElButton>
    </div>

    <ElTable v-loading="loading" :data="items" border stripe>
      <ElTableColumn prop="code" label="角色编码" min-width="120" />
      <ElTableColumn prop="name" label="角色名称" min-width="140" />
      <ElTableColumn label="状态" width="90">
        <template #default="{ row }">
          <ElTag :type="row.status === 1 ? 'success' : 'danger'">
            {{ row.status === 1 ? '启用' : '停用' }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn prop="remark" label="备注" min-width="160" show-overflow-tooltip />
      <ElTableColumn prop="createdAt" label="创建时间" width="170" />
      <ElTableColumn label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <ElButton link type="primary" @click="openGrant(row as SystemRoleItem)">
            授权菜单
          </ElButton>
          <ElButton link type="primary" @click="openEdit(row as SystemRoleItem)">
            编辑
          </ElButton>
          <ElButton link type="danger" @click="handleDelete(row as SystemRoleItem)">
            删除
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- 新增/编辑 -->
    <ElDialog
      v-model="dialogVisible"
      :title="isEdit ? `编辑角色: ${form.name}` : '新增角色'"
      width="480px"
      destroy-on-close
    >
      <ElForm label-width="80px">
        <ElFormItem label="角色编码" :required="!isEdit">
          <ElInput
            v-model="form.code"
            :placeholder="isEdit ? '可修改, 2-64位, 全局唯一' : '2-64位, 全局唯一, 如 ops'"
            maxlength="64"
          />
        </ElFormItem>
        <ElFormItem label="角色名称" required>
          <ElInput v-model="form.name" maxlength="64" />
        </ElFormItem>
        <ElFormItem label="状态">
          <ElSelect v-model="form.status" class="w-full">
            <ElOption :value="1" label="启用" />
            <ElOption :value="0" label="停用" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="备注">
          <ElInput v-model="form.remark" maxlength="255" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="handleSave">保存</ElButton>
      </template>
    </ElDialog>

    <!-- 菜单授权 -->
    <ElDialog
      v-model="grantVisible"
      :title="`菜单授权: ${grantRoleName}`"
      width="620px"
      destroy-on-close
    >
      <div class="mb-2 text-xs text-gray-400">
        超级管理员可将任意路由授予角色; 管理员只能将自己可授权的路由转发给角色,
        保存时会保留超管直授、自己无权管理的菜单
      </div>
      <div v-loading="grantLoading" class="h-[360px] overflow-auto border rounded p-2">
        <ElTree
          v-if="grantMenus.length > 0"
          ref="grantTree"
          :data="grantMenus"
          node-key="id"
          show-checkbox
          default-expand-all
          :props="{ label: 'title', children: 'children' }"
        />
        <div v-else class="text-gray-400">暂无可授权菜单</div>
      </div>
      <template #footer>
        <ElButton @click="grantVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="grantSaving"
          :disabled="!grantRoleId"
          @click="handleGrantSave"
          >保存授权</ElButton
        >
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
