<script lang="ts" setup>
import type { ConfigItem } from '#/api/system/config';

import { computed, reactive, ref } from 'vue';

import { useUserStore } from '@vben/stores';

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
  getConfigListApi,
  resetConfigApi,
  updateConfigApi,
} from '#/api/system/config';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'SystemConfig' });

const userStore = useUserStore();
// 写操作仅超级管理员(服务层同样强制限制, 管理员只读)
const isSuperAdmin = computed(
  () => userStore.userInfo?.roles?.includes('super_admin') ?? false,
);

// 列表状态(本页无分页, total 恒为 0 仅用于满足 useTable 的返回结构)
const {
  fetchList,
  handleReset: handleResetQuery,
  handleSearch,
  items,
  loading,
  query,
} = useTable<ConfigItem, { keyword: string; status: number }>({
  async fetch(q) {
    const res = await getConfigListApi({
      keyword: q.keyword || undefined,
      status: q.status,
    });
    return { items: res?.items ?? [], total: 0 };
  },
  defaultQuery: { keyword: '', status: -1 },
});

// 编辑参数值(内置项 key 只读, 不可删除)
const dialogVisible = ref(false);
const saving = ref(false);
const current = ref<ConfigItem>();
const form = reactive({
  key: '',
  name: '',
  remark: '',
  value: '',
});

function openEdit(row: unknown) {
  const item = row as ConfigItem;
  current.value = item;
  Object.assign(form, {
    key: item.configKey,
    name: item.name,
    remark: item.remark,
    value: item.configValue,
  });
  dialogVisible.value = true;
}

async function handleSave() {
  if (!form.key) return;
  saving.value = true;
  try {
    await updateConfigApi({ key: form.key, value: form.value });
    ElMessage.success('已保存, 下次读取立即生效');
    dialogVisible.value = false;
    await fetchList();
  } finally {
    saving.value = false;
  }
}

async function handleResetDefault(row: unknown) {
  const item = row as ConfigItem;
  const confirmed = await ElMessageBox.confirm(
    `将参数「${item.configKey}」恢复为内置默认值?`,
    '重置确认',
    { type: 'warning' },
  ).catch(() => false);
  if (!confirmed) return;
  const res = await resetConfigApi(item.configKey);
  ElMessage.success(`已重置为默认值: ${res?.configValue ?? ''}`);
  await fetchList();
}
</script>

<template>
  <div class="p-4">
    <div class="rounded-md border border-border bg-card p-4">
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <ElInput
          v-model="query.keyword"
          placeholder="参数键/名称"
          clearable
          class="w-52"
          @keyup.enter="handleSearch"
        />
        <ElSelect v-model="query.status" class="w-32">
          <ElOption :value="-1" label="全部状态" />
          <ElOption :value="1" label="启用" />
          <ElOption :value="0" label="停用" />
        </ElSelect>
        <ElButton type="primary" @click="handleSearch">查询</ElButton>
        <ElButton @click="handleResetQuery">重置</ElButton>
        <div class="flex-1"></div>
      </div>

      <ElTable
        v-loading="loading"
        :data="items"
        border
        stripe
        empty-text="暂无系统参数"
      >
        <ElTableColumn label="参数键" min-width="220">
          <template #default="{ row }">
            <span class="font-mono">{{ row.configKey }}</span>
            <ElTag v-if="row.builtin === 1" size="small" class="ml-1" type="info">
              内置
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="name" label="名称" min-width="150" />
        <ElTableColumn prop="configValue" label="当前值" min-width="160" />
        <ElTableColumn label="状态" width="90">
          <template #default="{ row }">
            <ElTag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '停用' }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="remark"
          label="说明"
          min-width="200"
          show-overflow-tooltip
        />
        <ElTableColumn prop="updatedAt" label="更新时间" width="170" />
        <ElTableColumn label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <ElButton
              link
              type="primary"
              :disabled="!isSuperAdmin"
              @click="openEdit(row)"
            >
              编辑值
            </ElButton>
            <ElButton
              link
              type="warning"
              :disabled="!isSuperAdmin || row.builtin !== 1"
              @click="handleResetDefault(row)"
            >
              重置默认
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>

      <div class="mt-3 text-xs text-gray-400">
        <p>
          内置项(标记"内置")的参数键不可修改、不可删除, 仅可调整参数值或重置为默认值;
          写操作仅超级管理员可执行。
        </p>
        <p>
          登录失败阈值、锁定窗口、验证码阈值、密码复杂度、会话上限等安全参数仍保存在
          config.yaml, 不在此表内, 避免"可被管理员修改的值"反过来约束管理员自身登录安全。
        </p>
      </div>
    </div>

    <ElDialog
      v-model="dialogVisible"
      :title="`编辑参数: ${form.key}`"
      width="520px"
      destroy-on-close
    >
      <ElForm label-width="90px">
        <ElFormItem label="参数键">
          <ElInput v-model="form.key" disabled />
        </ElFormItem>
        <ElFormItem label="名称">
          <ElInput :model-value="form.name" disabled />
        </ElFormItem>
        <ElFormItem label="参数值" required>
          <ElInput
            v-model="form.value"
            placeholder="请输入参数值"
            maxlength="512"
          />
        </ElFormItem>
        <ElFormItem label="说明">
          <span class="text-sm text-gray-500">{{ form.remark || '—' }}</span>
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
