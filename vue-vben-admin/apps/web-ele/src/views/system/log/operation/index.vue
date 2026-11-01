<script lang="ts" setup>
import type { OperationLogItem } from '#/api/system/log';

import { computed, onMounted, reactive, ref } from 'vue';

import { useUserStore } from '@vben/stores';

import {
  ElButton,
  ElInput,
  ElOption,
  ElPagination,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import { getOperationLogListApi } from '#/api/system/log';

defineOptions({ name: 'SystemOperationLog' });

const userStore = useUserStore();
// 请求方法列仅超级管理员可见
const isSuperAdmin = computed(
  () => userStore.userInfo?.roles?.includes('super_admin') ?? false,
);

const loading = ref(false);
const items = ref<OperationLogItem[]>([]);
const total = ref(0);
const query = reactive({
  page: 1,
  pageSize: 10,
  status: -1 as number,
  username: '',
});

async function fetchList() {
  loading.value = true;
  try {
    const res = await getOperationLogListApi({
      page: query.page,
      pageSize: query.pageSize,
      status: query.status,
      username: query.username || undefined,
    });
    items.value = res.items;
    total.value = res.total;
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  query.page = 1;
  fetchList();
}

function handleReset() {
  query.username = '';
  query.status = -1;
  handleSearch();
}

function handlePageSizeChange(size: number) {
  query.pageSize = size;
  query.page = 1;
  fetchList();
}

function handlePageChange(page: number) {
  query.page = page;
  fetchList();
}

const methodTagType = (method: string) => {
  if (method === 'POST') return 'success';
  if (method === 'PUT') return 'warning';
  if (method === 'DELETE') return 'danger';
  return 'info';
};

onMounted(fetchList);
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
            <ElTag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '成功' : '失败' }}
            </ElTag>
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
