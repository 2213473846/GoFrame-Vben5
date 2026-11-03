import type { PageParams, PageResult } from '#/api/types';

import { requestClient } from '#/api/request';

export interface OnlineUserItem {
  browser: string;
  ip: string;
  location: string;
  loginTime: string;
  os: string;
  offlineAt?: string;
  offlineReason?: string;
  realName: string;
  role: string;
  sessionId: string;
  /** 1=在线 0=已下线(被顶替/超出设备数上限/管理员强退/退出) */
  status: number;
  userId: number;
  username: string;
}

export interface OnlineUserListParams extends PageParams {
  username?: string;
}

export type OnlineUserListResult = PageResult<OnlineUserItem>;

/**
 * 在线用户列表(数据范围: 超管全部, 管理员自己+新增用户, 用户仅自己)
 */
export async function getOnlineUserListApi(params: OnlineUserListParams) {
  return requestClient.get<OnlineUserListResult>('/online/list', {
    params,
  });
}

/**
 * 强退用户(删除其全部登录会话, 被强退后需重新登录)
 */
export async function kickOnlineUserApi(userId: number) {
  return requestClient.post('/online/kick', { userId });
}
