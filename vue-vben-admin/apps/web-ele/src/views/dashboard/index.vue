<script lang="ts" setup>
import type { DashboardOverview } from '#/api/dashboard';
import type { DictFallbackItem } from '#/composables/useDict';

import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

import {
  ElButton,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import { getDashboardOverviewApi } from '#/api/dashboard';
import DictTag from '#/components/dict-tag/index.vue';

defineOptions({ name: 'Dashboard' });

// 公告级别字典兜底文案(字典未配置时回退)
const levelFallback: Record<string, DictFallbackItem> = {
  important: { label: '重要', tagColor: 'warning' },
  normal: { label: '普通', tagColor: 'info' },
  urgent: { label: '紧急', tagColor: 'danger' },
};

const router = useRouter();

const loading = ref(false);
const overview = ref<DashboardOverview>();

async function fetchOverview() {
  loading.value = true;
  try {
    overview.value = await getDashboardOverviewApi();
  } catch {
    // 普通用户无管理权限时后端返回 403, 页面保持空态
    overview.value = undefined;
  } finally {
    loading.value = false;
  }
}

interface DashboardCard {
  label: string;
  /** 有链接的卡片可点击跳转 */
  link?: string;
  value: number;
}

const cards = computed<DashboardCard[]>(() => [
  {
    label: '用户总数',
    value: overview.value?.userTotal ?? 0,
  },
  {
    label: '当前在线',
    value: overview.value?.onlineCount ?? 0,
  },
  {
    label: '今日登录成功',
    value: overview.value?.loginSuccessToday ?? 0,
  },
  {
    label: '今日登录失败',
    value: overview.value?.loginFailedToday ?? 0,
  },
  {
    label: '未读公告',
    link: '/notice',
    value: overview.value?.unreadNoticeCount ?? 0,
  },
]);

// 趋势用 CSS 柱状实现, 不额外引入图表依赖
const trend = computed(() => overview.value?.trend ?? []);
const trendMax = computed(() => {
  let max = 1;
  for (const point of trend.value) {
    max = Math.max(max, point.success, point.failed);
  }
  return max;
});

function barPercent(value: number) {
  if (value <= 0) return '0%';
  const percent = (value / trendMax.value) * 100;
  return `${Math.max(percent, 3)}%`;
}

const recent = computed(() => overview.value?.recent ?? []);

// 最近公告(后端按当前用户可见范围返回, 最多 3 条)
const recentNotices = computed(() => overview.value?.recentNotices ?? []);

const scopeText = computed(() =>
  overview.value?.scope === 'all' ? '全量数据' : '仅本人及下属范围',
);

onMounted(fetchOverview);
</script>

<template>
  <div v-loading="loading" class="p-4">
    <div
      v-if="overview"
      class="mb-3 flex flex-wrap items-center gap-2 text-sm text-gray-500"
    >
      <span>数据范围:</span>
      <ElTag :type="overview.scope === 'all' ? 'danger' : 'warning'" size="small">
        {{ scopeText }}
      </ElTag>
      <span class="text-xs text-gray-400">
        统计口径与用户/在线/登录日志列表一致
      </span>
    </div>

    <!-- 概览卡片 -->
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5">
      <div
        v-for="card in cards"
        :key="card.label"
        class="rounded-md border border-border bg-card p-4"
        :class="card.link ? 'cursor-pointer hover:border-primary' : ''"
        @click="card.link && router.push(card.link)"
      >
        <div class="text-sm text-gray-500">{{ card.label }}</div>
        <div class="mt-2 text-2xl font-semibold">{{ card.value }}</div>
      </div>
    </div>

    <!-- 登录趋势 -->
    <div class="mt-4 rounded-md border border-border bg-card p-4">
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <div class="text-sm font-medium">
          登录趋势
          <span class="ml-1 text-gray-400">
            近 {{ overview?.trendDays ?? 0 }} 天
          </span>
        </div>
        <div class="flex-1"></div>
        <div class="flex items-center gap-1 text-xs text-gray-500">
          <span class="inline-block h-2 w-2 rounded-sm bg-emerald-500"></span>
          成功
        </div>
        <div class="flex items-center gap-1 text-xs text-gray-500">
          <span class="inline-block h-2 w-2 rounded-sm bg-red-500"></span>
          失败
        </div>
      </div>

      <div v-if="trend.length === 0" class="py-8 text-center text-sm text-gray-400">
        暂无登录数据
      </div>
      <div v-else class="flex items-end gap-2 overflow-x-auto pb-2">
        <div
          v-for="point in trend"
          :key="point.date"
          class="flex min-w-10 flex-1 flex-col items-center"
          :title="`${point.date} 成功 ${point.success} / 失败 ${point.failed}`"
        >
          <div class="flex h-40 w-full items-end justify-center gap-1">
            <div
              class="w-3 rounded-t bg-emerald-500 transition-all"
              :style="{ height: barPercent(point.success) }"
            ></div>
            <div
              class="w-3 rounded-t bg-red-500 transition-all"
              :style="{ height: barPercent(point.failed) }"
            ></div>
          </div>
          <div class="mt-1 text-xs text-gray-400">
            {{ point.date.slice(5) }}
          </div>
          <div class="text-xs text-gray-400">
            {{ point.success }}/{{ point.failed }}
          </div>
        </div>
      </div>
    </div>

    <!-- 最近公告 -->
    <div class="mt-4 rounded-md border border-border bg-card p-4">
      <div class="mb-3 flex items-center">
        <div class="text-sm font-medium">最近公告</div>
        <div class="flex-1"></div>
        <ElButton link type="primary" @click="router.push('/notice')">
          查看全部
        </ElButton>
      </div>
      <div
        v-if="recentNotices.length === 0"
        class="py-6 text-center text-sm text-gray-400"
      >
        暂无公告
      </div>
      <div v-else class="flex flex-col">
        <div
          v-for="notice in recentNotices"
          :key="notice.id"
          class="flex cursor-pointer items-center gap-2 border-t border-border py-2 first:border-t-0 hover:bg-accent"
          @click="router.push('/notice')"
        >
          <span
            class="size-2 shrink-0 rounded-full"
            :class="notice.isRead ? 'bg-transparent' : 'bg-primary'"
          ></span>
          <span
            class="min-w-0 truncate text-sm"
            :class="notice.isRead ? 'text-gray-500' : 'font-medium'"
          >
            {{ notice.title }}
          </span>
          <DictTag
            type="notice_level"
            :value="notice.level"
            :fallback="levelFallback"
          />
          <div class="flex-1"></div>
          <span class="shrink-0 text-xs text-gray-400">
            {{ notice.publishAt }}
          </span>
        </div>
      </div>
    </div>

    <!-- 最近动态 -->
    <div class="mt-4 rounded-md border border-border bg-card p-4">
      <div class="mb-3 text-sm font-medium">最近动态</div>
      <ElTable :data="recent" border stripe size="small" empty-text="暂无操作记录">
        <ElTableColumn prop="createdAt" label="时间" width="170" />
        <ElTableColumn prop="username" label="操作人" width="130" />
        <ElTableColumn prop="module" label="模块" min-width="120" />
        <ElTableColumn prop="action" label="操作" min-width="160" />
        <ElTableColumn label="状态" width="90">
          <template #default="{ row }">
            <ElTag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '成功' : '失败' }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="ip" label="IP" width="140" />
      </ElTable>
    </div>
  </div>
</template>
