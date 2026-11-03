<script lang="ts" setup>
import type { OperationLogItem } from '#/api/system/log';
import type { DictFallbackItem } from '#/composables/useDict';

import { computed, ref } from 'vue';

import { useUserStore } from '@vben/stores';

import {
  ElButton,
  ElInput,
  ElMessage,
  ElOption,
  ElPagination,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import { downloadExportApi } from '#/api/system/export';
import { getOperationLogListApi } from '#/api/system/log';
import DictTag from '#/components/dict-tag/index.vue';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'SystemOperationLog' });

// 日志状态字典兜底文案(字典未配置时回退), 展示交给 DictTag 组件
const logStatusFallback: Record<string, DictFallbackItem> = {
  '0': { label: '失败', tagColor: 'danger' },
  '1': { label: '成功', tagColor: 'success' },
};

const userStore = useUserStore();
// 请求方法列仅超级管理员可见
const isSuperAdmin = computed(
  () => userStore.userInfo?.roles?.includes('super_admin') ?? false,
);

// 列表状态(分页/加载/查询条件由 useTable 统一管理)
const {
  fetchList,
  handlePageChange,
  handlePageSizeChange,
  handleReset,
  handleSearch,
  items,
  loading,
  query,
  total,
} = useTable<OperationLogItem, { status: number; username: string }>({
  async fetch(q) {
    return getOperationLogListApi({
      page: q.page,
      pageSize: q.pageSize,
      status: q.status,
      username: q.username || undefined,
    });
  },
  defaultQuery: { status: -1, username: '' },
});

// 导出当前筛选条件下的操作日志
const exporting = ref(false);
async function handleExport(format: 'csv' | 'xlsx') {
  exporting.value = true;
  try {
    await downloadExportApi('/operationlog/export', {
      format,
      params: {
        status: query.status,
        username: query.username || undefined,
      },
    });
    ElMessage.success('导出已开始');
  } finally {
    exporting.value = false;
  }
}

const methodTagType = (method: string) => {
  if (method === 'POST') return 'success';
  if (method === 'PUT') return 'warning';
  if (method === 'DELETE') return 'danger';
  return 'info';
};
</script>

<template>
  <div class="p-4">
    <div class="rounded-md border border-border bg-card p-4">
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <ElInput
          v-model="query.username"
          placeholder="用户名称"
          clearable
          class="w-44"
          @keyup.enter="handleSearch"
        />
        <ElSelect v-model="query.status" class="w-32">
          <ElOption :value="-1" label="全部状态" />
          <ElOption :value="1" label="成功" />
          <ElOption :value="0" label="失败" />
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
        <ElButton @click="fetchList">刷新</ElButton>
      </div>

      <ElTable
        v-loading="loading"
        :data="items"
        border
        stripe
        empty-text="暂无操作日志"
        class="log-table"
      >
        <ElTableColumn prop="username" label="用户名称" min-width="110" />
        <ElTableColumn prop="module" label="操作模块" min-width="100" />
        <ElTableColumn prop="action" label="操作类型" min-width="110" />
        <ElTableColumn v-if="isSuperAdmin" label="请求方法" width="95">
          <template #default="{ row }">
            <ElTag :type="methodTagType(row.method)" effect="plain">
              {{ row.method }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="path"
          label="请求地址"
          min-width="140"
          show-overflow-tooltip
        />
        <ElTableColumn prop="ip" label="操作地址(IP)" min-width="130" />
        <ElTableColumn prop="location" label="操作地点" min-width="140" />
        <ElTableColumn prop="browser" label="浏览器" min-width="100" />
        <ElTableColumn prop="os" label="操作系统" min-width="105" />
        <ElTableColumn label="操作状态" width="95">
          <template #default="{ row }">
            <DictTag
              type="sys_log_type"
              :value="row.status"
              :fallback="logStatusFallback"
            />
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="message"
          label="操作信息"
          min-width="130"
          show-overflow-tooltip
        />
        <ElTableColumn prop="createdAt" label="操作时间" min-width="160" />
      </ElTable>

      <div class="mt-4 flex justify-end">
        <ElPagination
          :current-page="query.page"
          :page-size="query.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          background
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handlePageSizeChange"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  text-align: center;
}
</style>
