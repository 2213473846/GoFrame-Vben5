<script lang="ts" setup>
import type { MyNoticeItem } from '#/api/notice';
import type { DictFallbackItem } from '#/composables/useDict';

import { onMounted, ref } from 'vue';

import {
  ElButton,
  ElMessage,
  ElPagination,
  ElTag,
} from 'element-plus';

import {
  getMyNoticeListApi,
  getNoticeUnreadCountApi,
  readNoticeApi,
} from '#/api/notice';
import DictTag from '#/components/dict-tag/index.vue';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'MyNotice' });

// 公告级别字典兜底文案(字典未配置时回退), 展示交给 DictTag
const levelFallback: Record<string, DictFallbackItem> = {
  important: { label: '重要', tagColor: 'warning' },
  normal: { label: '普通', tagColor: 'info' },
  urgent: { label: '紧急', tagColor: 'danger' },
};

const {
  fetchList,
  handlePageChange,
  handlePageSizeChange,
  items,
  loading,
  query,
  total,
} = useTable<MyNoticeItem, Record<string, never>>({
  async fetch(q) {
    return getMyNoticeListApi({ page: q.page, pageSize: q.pageSize });
  },
  defaultQuery: {},
});

// 未读总数(跨页统计, 由后端接口返回)
const unreadCount = ref(0);
// 当前展开的公告 id; 同一条再次点击收起
const expandedId = ref<number>();

async function fetchUnreadCount() {
  try {
    const res = await getNoticeUnreadCountApi();
    unreadCount.value = res.count;
  } catch {
    // 顶栏与页面均会兜底展示, 未读数拉取失败不阻塞列表
    unreadCount.value = 0;
  }
}

async function handleToggle(item: MyNoticeItem) {
  expandedId.value = expandedId.value === item.id ? undefined : item.id;
  if (item.isRead) return;
  // 展开即标记已读(后端幂等, 重复调用无副作用)
  await readNoticeApi({ id: item.id });
  item.isRead = true;
  unreadCount.value = Math.max(0, unreadCount.value - 1);
}

async function handleReadAll() {
  if (unreadCount.value <= 0) return;
  await readNoticeApi({ all: true });
  for (const item of items.value) {
    item.isRead = true;
  }
  unreadCount.value = 0;
  ElMessage.success('已全部标记为已读');
}

onMounted(fetchUnreadCount);
</script>

<template>
  <div class="p-4">
    <div class="rounded-md border border-border bg-card p-4">
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <div class="text-base font-medium">我的通知</div>
        <ElTag v-if="unreadCount > 0" type="danger" size="small">
          未读 {{ unreadCount }} 条
        </ElTag>
        <ElTag v-else type="info" size="small">全部已读</ElTag>
        <div class="flex-1"></div>
        <ElButton @click="fetchList">刷新</ElButton>
        <ElButton
          type="primary"
          :disabled="unreadCount <= 0"
          @click="handleReadAll"
        >
          全部已读
        </ElButton>
      </div>

      <div v-loading="loading">
        <div v-if="items.length === 0" class="py-16 text-center text-sm text-gray-400">
          暂无通知
        </div>

        <div v-else class="flex flex-col gap-2">
          <div
            v-for="item in items"
            :key="item.id"
            class="cursor-pointer rounded-md border border-border p-3 transition-colors hover:bg-accent"
            @click="handleToggle(item)"
          >
            <div class="flex items-start gap-3">
              <span
                class="mt-2 size-2 shrink-0 rounded-full"
                :class="item.isRead ? 'bg-transparent' : 'bg-primary'"
              ></span>
              <div class="flex min-w-0 flex-1 flex-col gap-1">
                <div class="flex flex-wrap items-center gap-2">
                  <span
                    class="text-sm"
                    :class="item.isRead ? 'text-gray-500' : 'font-medium'"
                  >
                    {{ item.title }}
                  </span>
                  <DictTag
                    type="notice_level"
                    :value="item.level"
                    :fallback="levelFallback"
                  />
                </div>
                <div class="text-xs text-gray-400">
                  发布时间: {{ item.publishAt || item.createdAt }}
                </div>
              </div>
              <span class="shrink-0 text-xs text-gray-400">
                {{ expandedId === item.id ? '收起' : '展开' }}
              </span>
            </div>

            <!-- 纯文本渲染: 文本插值 + pre-wrap, 禁止 v-html 以防 XSS -->
            <div
              v-if="expandedId === item.id"
              class="mt-3 rounded-md bg-accent p-3 text-sm leading-6"
              style="white-space: pre-wrap; word-break: break-word"
            >
              {{ item.content }}
            </div>
          </div>
        </div>
      </div>

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
