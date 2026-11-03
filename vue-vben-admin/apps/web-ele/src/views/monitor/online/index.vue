<script lang="ts" setup>
import type { OnlineUserItem } from '#/api/system/online';

import { computed } from 'vue';

import { useUserStore } from '@vben/stores';

import {
  ElButton,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElPagination,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import {
  getOnlineUserListApi,
  kickOnlineUserApi,
} from '#/api/system/online';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'OnlineUser' });

const userStore = useUserStore();
// 当前登录用户, 强退自己没有意义, 前端先行禁用
const currentUserId = computed(() => Number(userStore.userInfo?.userId ?? 0));

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
} = useTable<OnlineUserItem, { username: string }>({
  async fetch(q) {
    return getOnlineUserListApi({
      page: q.page,
      pageSize: q.pageSize,
      username: q.username || undefined,
    });
  },
  defaultQuery: { username: '' },
});

const roleText = (role: string) => {
  if (role === 'super_admin') return '超级管理员';
  if (role === 'admin') return '管理员';
  return '普通用户';
};
const roleTagType = (role: string) => {
  if (role === 'super_admin') return 'danger';
  if (role === 'admin') return 'warning';
  return 'info';
};

async function handleKick(row: OnlineUserItem) {
  const name = row.realName || row.username;
  const confirmed = await ElMessageBox.confirm(
    `确定强退用户「${name}」吗？强退后该用户的全部登录会话将失效，需要重新登录。`,
    '强退确认',
    {
      confirmButtonText: '强退',
      cancelButtonText: '取消',
      type: 'warning',
    },
  ).catch(() => false);
  if (!confirmed) return;
  await kickOnlineUserApi(row.userId);
  ElMessage.success(`已强退用户「${name}」`);
  fetchList();
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
        empty-text="暂无在线用户"
      >
        <ElTableColumn prop="username" label="用户名称" min-width="120">
          <template #default="{ row }">
            {{ row.username }}
            <ElTag
              v-if="Number(row.userId) === currentUserId"
              size="small"
              class="ml-1"
              >当前</ElTag
            >
          </template>
        </ElTableColumn>
        <ElTableColumn label="角色" width="110">
          <template #default="{ row }">
            <ElTag :type="roleTagType(row.role)" size="small" effect="plain">
              {{ roleText(row.role) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="ip" label="登录地址(IP)" min-width="130" />
        <ElTableColumn prop="location" label="登录地点" min-width="140" />
        <ElTableColumn prop="browser" label="浏览器" min-width="110" />
        <ElTableColumn prop="os" label="操作系统" min-width="110" />
        <ElTableColumn prop="loginTime" label="登录时间" min-width="160" />
        <ElTableColumn label="状态" width="90">
          <template #default="{ row }">
            <ElTag
              :type="row.status === 1 ? 'success' : 'info'"
              size="small"
            >
              {{ row.status === 1 ? '在线' : '已下线' }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="下线原因" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.status === 0" class="text-gray-500">
              {{ row.offlineReason || '会话已失效' }}
              <span v-if="row.offlineAt" class="text-gray-400">
                ({{ row.offlineAt }})
              </span>
            </span>
            <span v-else class="text-gray-400">—</span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <ElButton
              link
              type="danger"
              :disabled="
                Number(row.userId) === currentUserId || row.status === 0
              "
              @click="handleKick(row as OnlineUserItem)"
            >
              强退
            </ElButton>
          </template>
        </ElTableColumn>
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
