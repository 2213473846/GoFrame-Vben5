<script lang="ts" setup>
import type { MyNoticeItem } from '#/api/notice';
import type { NotificationItem } from '@vben/layouts';

import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';

import { AuthenticationLoginExpiredModal } from '@vben/common-ui';
import { VBEN_DOC_URL, VBEN_GITHUB_URL } from '@vben/constants';
import { useWatermark } from '@vben/hooks';
import { BookOpenText, CircleHelp, SvgGithubIcon } from '@vben/icons';
import {
  BasicLayout,
  LockScreen,
  Notification,
  UserDropdown,
} from '@vben/layouts';
import { preferences, usePreferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { openWindow } from '@vben/utils';

import { getMyNoticeListApi, getNoticeUnreadCountApi, readNoticeApi } from '#/api/notice';
import { $t } from '#/locales';
import { useAuthStore } from '#/store';
import LoginForm from '#/views/_core/authentication/login.vue';

// 顶栏通知: 真实数据(最近 5 条) + 未读数轮询
const notifications = ref<NotificationItem[]>([]);
const unreadCount = ref(0);
// 轮询间隔(秒), 由 /notice/unread-count 下发, 缺省 60
const pollSeconds = ref(60);
let pollTimer: ReturnType<typeof setInterval> | undefined;
// 组件是否已卸载, 避免异步首屏请求返回后继续创建定时器
let destroyed = false;

// 未读数大于 0 才显示红点
const showDot = computed(() => unreadCount.value > 0);

/** 把用户端公告映射为顶栏通知组件所需结构 */
function toNotificationItem(item: MyNoticeItem): NotificationItem {
  return {
    avatar: preferences.app.defaultAvatar,
    date: item.publishAt || item.createdAt,
    id: item.id,
    isRead: item.isRead,
    message: item.content,
    title: item.title,
    // 点击单条统一跳转「我的通知」页
    link: '/notice',
  };
}

async function loadNotifications() {
  try {
    const res = await getMyNoticeListApi({ page: 1, pageSize: 5 });
    notifications.value = res.items.map(toNotificationItem);
  } catch {
    // 列表拉取失败时保留上次结果, 不阻塞布局
  }
}

/**
 * 刷新未读数; 未读发生变化时同步刷新列表
 * @param loadListOnChange 未读数变化时是否重新拉取列表
 */
async function refreshUnread(loadListOnChange = true) {
  try {
    const res = await getNoticeUnreadCountApi();
    const nextSeconds =
      res.pollSeconds > 0 ? res.pollSeconds : 60;
    if (nextSeconds !== pollSeconds.value) {
      pollSeconds.value = nextSeconds;
      // 后端配置变化时按新间隔重启轮询
      if (pollTimer) startPolling();
    }
    if (res.count !== unreadCount.value) {
      unreadCount.value = res.count;
      if (loadListOnChange) {
        await loadNotifications();
      }
    }
  } catch {
    // 轮询失败静默处理, 由下一轮自动重试
  }
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = undefined;
  }
}

function startPolling() {
  stopPolling();
  pollTimer = setInterval(() => {
    refreshUnread();
  }, pollSeconds.value * 1000);
}

// 页面从后台切回前台时立即刷新一次
function handleVisibilityChange() {
  if (document.visibilityState === 'visible') {
    refreshUnread(false);
    loadNotifications();
  }
}

const router = useRouter();
const userStore = useUserStore();
const authStore = useAuthStore();
const accessStore = useAccessStore();
const { destroyWatermark, updateWatermark } = useWatermark();
const { isDark } = usePreferences();

const menus = computed(() => [
  {
    handler: () => {
      router.push({ name: 'Profile' });
    },
    icon: 'lucide:user',
    text: $t('page.auth.profile'),
  },
  {
    handler: () => {
      openWindow(VBEN_DOC_URL, {
        target: '_blank',
      });
    },
    icon: BookOpenText,
    text: $t('ui.widgets.document'),
  },
  {
    handler: () => {
      openWindow(VBEN_GITHUB_URL, {
        target: '_blank',
      });
    },
    icon: SvgGithubIcon,
    text: 'GitHub',
  },
  {
    handler: () => {
      openWindow(`${VBEN_GITHUB_URL}/issues`, {
        target: '_blank',
      });
    },
    icon: CircleHelp,
    text: $t('ui.widgets.qa'),
  },
]);

const avatar = computed(() => {
  return userStore.userInfo?.avatar ?? preferences.app.defaultAvatar;
});

async function handleLogout() {
  await authStore.logout(false);
}

function handleNoticeClear() {
  // 仅清空当前展示, 真实未读状态仍由后端维护
  notifications.value = [];
}

async function markRead(id: number | string) {
  const item = notifications.value.find((notice) => notice.id === id);
  if (!item || item.isRead) return;
  await readNoticeApi({ id: Number(id) });
  item.isRead = true;
  unreadCount.value = Math.max(0, unreadCount.value - 1);
}

function remove(id: number | string) {
  notifications.value = notifications.value.filter((item) => item.id !== id);
}

async function handleMakeAll() {
  if (unreadCount.value <= 0) return;
  await readNoticeApi({ all: true });
  notifications.value.forEach((item) => (item.isRead = true));
  unreadCount.value = 0;
}

const viewAll = () => {
  router.push('/notice');
};

const handleClick = (item: NotificationItem) => {
  // 点击单条: 先标记已读, 再进入「我的通知」页
  void markRead(item.id);
  router.push('/notice');
};

onMounted(async () => {
  // 首屏同时拉取未读数与最近 5 条, 之后按后端下发间隔轮询
  await Promise.all([refreshUnread(false), loadNotifications()]);
  if (destroyed) return;
  startPolling();
  document.addEventListener('visibilitychange', handleVisibilityChange);
});

onUnmounted(() => {
  destroyed = true;
  stopPolling();
  document.removeEventListener('visibilitychange', handleVisibilityChange);
});

watch(
  () => ({
    enable: preferences.app.watermark,
    content: preferences.app.watermarkContent,
    isDark: isDark.value,
  }),
  async ({ enable, content, isDark: isDarkValue }) => {
    if (enable) {
      const watermarkColor = isDarkValue
        ? 'rgba(255, 255, 255, 0.12)'
        : 'rgba(0, 0, 0, 0.12)';

      await updateWatermark({
        advancedStyle: {
          colorStops: [
            {
              color: watermarkColor,
              offset: 0,
            },
            {
              color: watermarkColor,
              offset: 1,
            },
          ],
          type: 'linear',
        },
        content:
          content ||
          `${userStore.userInfo?.username} - ${userStore.userInfo?.realName}`,
      });
    } else {
      destroyWatermark();
    }
  },
  {
    immediate: true,
  },
);
</script>

<template>
  <BasicLayout
    :avatar
    :text="userStore.userInfo?.realName"
    @clear-preferences-and-logout="handleLogout"
    @logout="handleLogout"
  >
    <template #user-dropdown>
      <UserDropdown
        :avatar
        :menus
        :text="userStore.userInfo?.realName"
        description="ann.vben@gmail.com"
        tag-text="Pro"
        @clear-preferences-and-logout="handleLogout"
        @logout="handleLogout"
      />
    </template>
    <template #notification>
      <Notification
        :dot="showDot"
        :notifications="notifications"
        @clear="handleNoticeClear"
        @read="(item) => markRead(item.id)"
        @remove="(item) => item.id && remove(item.id)"
        @make-all="handleMakeAll"
        @on-click="handleClick"
        @view-all="viewAll"
      />
    </template>
    <template #extra>
      <AuthenticationLoginExpiredModal
        v-model:open="accessStore.loginExpired"
        :avatar
      >
        <LoginForm />
      </AuthenticationLoginExpiredModal>
    </template>
    <template #lock-screen>
      <LockScreen :avatar @to-login="handleLogout" />
    </template>
  </BasicLayout>
</template>
