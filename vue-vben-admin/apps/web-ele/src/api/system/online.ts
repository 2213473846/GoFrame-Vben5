import { requestClient } from '#/api/request';

export interface OnlineUserItem {
  browser: string;
  ip: string;
  location: string;
  loginTime: string;
  os: string;
  realName: string;
  role: string;
  userId: number;
  username: string;
}

export interface OnlineUserListParams {
  page?: number;
  pageSize?: number;
  username?: string;
}

export interface OnlineUserListResult {
  items: OnlineUserItem[];
  total: number;
}

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
