import type { SystemMenuItem } from './menu';

import { requestClient } from '#/api/request';

/** 功能角色列表项 */
export interface SystemRoleItem {
  code: string;
  createdAt: string;
  id: number;
  name: string;
  remark: string;
  status: number;
}

/** 功能角色列表响应 */
export interface SystemRoleListResult {
  items: SystemRoleItem[];
}

/** 角色菜单授权详情 */
export interface RoleGrantInfo {
  checkedIds: number[];
  items: SystemMenuItem[];
  name: string;
  roleId: number;
}

/** 查询功能角色列表 */
export async function getRoleListApi() {
  return requestClient.get<SystemRoleListResult>('/role/list');
}

/** 新增角色 */
export async function createRoleApi(data: {
  code: string;
  name: string;
  remark?: string;
  status: number;
}) {
  return requestClient.post('/role/create', data);
}

/** 修改角色 */
export async function updateRoleApi(data: {
  code: string;
  id: number;
  name: string;
  remark?: string;
  status: number;
}) {
  return requestClient.put('/role/update', data);
}

/** 删除角色 */
export async function deleteRoleApi(id: number) {
  return requestClient.delete(`/role/delete?id=${id}`);
}

/** 查询角色菜单授权详情 */
export async function getRoleGrantInfoApi(id: number) {
  return requestClient.get<RoleGrantInfo>('/role/grant', {
    params: { id },
  });
}

/** 保存角色菜单授权 */
export async function grantRoleApi(roleId: number, menuIds: number[]) {
  return requestClient.put('/role/grant', { menuIds, roleId });
}
