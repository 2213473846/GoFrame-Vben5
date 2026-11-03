<script lang="ts" setup>
import type { LoginLogItem } from '#/api/system/log';
import type { DictFallbackItem } from '#/composables/useDict';

import { ref } from 'vue';

import {
  ElButton,
  ElInput,
  ElMessage,
  ElOption,
  ElPagination,
  ElSelect,
  ElTable,
  ElTableColumn,
} from 'element-plus';

import { downloadExportApi } from '#/api/system/export';
import { getLoginLogListApi } from '#/api/system/log';
import DictTag from '#/components/dict-tag/index.vue';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'SystemLoginLog' });

// 日志状态字典兜底文案(字典未配置时回退), 展示交给 DictTag 组件
const logStatusFallback: Record<string, DictFallbackItem> = {
  '0': { label: '失败', tagColor: 'danger' },
  '1': { label: '成功', tagColor: 'success' },
};

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
} = useTable<LoginLogItem, { status: number; username: string }>({
  async fetch(q) {
    return getLoginLogListApi({
      page: q.page,
      pageSize: q.pageSize,
      status: q.status,
      username: q.username || undefined,
    });
  },
  defaultQuery: { status: -1, username: '' },
});

// 导出当前筛选条件下的登录日志
const exporting = ref(false);
async function handleExport(format: 'csv' | 'xlsx') {
  exporting.value = true;
  try {
    await downloadExportApi('/loginlog/export', {
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
        empty-text="暂无登录日志"
        class="log-table"
      >
        <ElTableColumn prop="username" label="用户名称" min-width="120" />
        <ElTableColumn prop="ip" label="登录地址(IP)" min-width="130" />
        <ElTableColumn prop="location" label="登录地点" min-width="150" />
        <ElTableColumn prop="browser" label="浏览器" min-width="110" />
        <ElTableColumn prop="os" label="操作系统" min-width="110" />
        <ElTableColumn label="登录状态" width="100">
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
          label="登录信息"
          min-width="150"
          show-overflow-tooltip
        />
        <ElTableColumn prop="createdAt" label="登录时间" min-width="165" />
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
