import { requestClient } from '#/api/request';

export interface DashboardTrendPoint {
  date: string;
  failed: number;
  success: number;
}

export interface DashboardRecentItem {
  action: string;
  createdAt: string;
  id: number;
  ip: string;
  module: string;
  status: number;
  username: string;
}

export interface DashboardNoticeItem {
  id: number;
  isRead: boolean;
  /** 级别(字典 notice_level) */
  level: string;
  publishAt: string;
  title: string;
}

export interface DashboardOverview {
  loginFailedToday: number;
  loginSuccessToday: number;
  onlineCount: number;
  recent: DashboardRecentItem[];
  /** 最近可见公告(最多 3 条) */
  recentNotices: DashboardNoticeItem[];
  /** all=全量数据; descendants=仅当前管理员及其下属 */
  scope: string;
  trend: DashboardTrendPoint[];
  trendDays: number;
  /** 当前用户未读公告数 */
  unreadNoticeCount: number;
  userTotal: number;
}

/**
 * 工作台概览(仅超管/管理员; 普通用户 403)
 */
export async function getDashboardOverviewApi() {
  return requestClient.get<DashboardOverview>('/dashboard/overview');
}
