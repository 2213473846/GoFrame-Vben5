import { requestClient } from '#/api/request';

export interface SystemUserListItem {
  avatar: string;
  createdAt: string;
  id: number;
  parentRealName: string;
  parentUsername: string;
  pid: number;
  realName: string;
  remark: string;
  role: string;
  status: number;
  username: string;
}

export interface SystemUserListParams {
  page?: number;
  pageSize?: number;
  realName?: string;
  role?: string;
  username?: string;
}

export interface SystemUserListResult {
  items: SystemUserListItem[];
  total: number;
}

/**
 * 用户列表(数据权限: 超管全部, 管理员仅子孙)
 */
export async function getUserListApi(params: SystemUserListParams) {
  return requestClient.get<SystemUserListResult>('/user/list', { params });
}

/**
 * 创建管理员/用户
 */
export async function createUserApi(data: {
  password: string;
  realName?: string;
  remark?: string;
  role: string;
  username: string;
}) {
  return requestClient.post<{ id: number }>('/user/create', data);
}

/**
 * 更新用户(字段为空则不修改)
 */
export async function updateUserApi(data: {
  avatar?: string;
  id: number;
  password?: string;
  realName?: string;
  remark?: string;
  role?: string;
  status?: number;
  username?: string;
}) {
  return requestClient.put('/user/update', data);
}

/**
 * 删除用户(删除管理员时级联删除其子孙)
 */
export async function deleteUserApi(id: number) {
  return requestClient.delete(`/user/delete?id=${id}`);
}

export interface RecycleItem {
  deletedAt: string;
  id: number;
  operator: string;
  pid: number;
  realName: string;
  role: string;
  username: string;
}

/**
 * 回收站列表(已软删用户)
 */
export async function getRecycleListApi(params: {
  page?: number;
  pageSize?: number;
  username?: string;
}) {
  return requestClient.get<{ items: RecycleItem[]; total: number }>(
    '/user/recycle',
    { params },
  );
}

/**
 * 恢复回收站用户
 */
export async function restoreUserApi(id: number) {
  return requestClient.put('/user/restore', { id });
}

/**
 * 彻底删除(物理删除, 不可恢复)
 */
export async function purgeUserApi(id: number) {
  return requestClient.delete(`/user/purge?id=${id}`);
}
