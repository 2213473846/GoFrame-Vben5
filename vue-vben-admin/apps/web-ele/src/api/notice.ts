import type { PageParams, PageResult } from '#/api/types';

import { requestClient, silentRequestClient } from '#/api/request';

/**
 * 用户端公告条目(任意已登录用户可见范围由后端统一收口)
 */
export interface MyNoticeItem {
  content: string;
  createdAt: string;
  id: number;
  /** 当前用户是否已读 */
  isRead: boolean;
  /** 级别(字典 notice_level: normal/important/urgent) */
  level: string;
  publishAt: string;
  title: string;
  /** 类型(字典 notice_type: notice/announcement) */
  type: string;
}

export type MyNoticeListResult = PageResult<MyNoticeItem>;

/**
 * 我的通知列表(仅返回当前用户可见且未过期的公告)
 */
export async function getMyNoticeListApi(params: PageParams) {
  return requestClient.get<MyNoticeListResult>('/notice/my', { params });
}

export interface NoticeUnreadCountResult {
  count: number;
  /** 轮询间隔(秒), 后端配置下发, 前端缺省按 60 处理 */
  pollSeconds: number;
}

/**
 * 未读公告数(顶栏轮询使用 silent 客户端, 避免网络抖动时反复弹全局错误提示)
 */
export async function getNoticeUnreadCountApi() {
  return silentRequestClient.get<NoticeUnreadCountResult>(
    '/notice/unread-count',
  );
}

/**
 * 标记已读: id 指定单条, all=true 全部已读(后端幂等)
 */
export async function readNoticeApi(data: { all?: boolean; id?: number }) {
  return requestClient.post('/notice/read', data);
}
