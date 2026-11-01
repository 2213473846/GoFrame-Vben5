import { requestClient } from '#/api/request';

export interface SystemMenuItem {
  affixTab: boolean;
  children?: SystemMenuItem[];
  component: string;
  createdAt?: string;
  hideInMenu: boolean;
  icon: string;
  id: number;
  isPublic: boolean;
  keepAlive: boolean;
  name: string;
  path: string;
  pid: number;
  redirect: string;
  remark: string;
  sort: number;
  status: number;
  title: string;
  type: 'catalog' | 'menu';
}

export interface SystemMenuPayload {
  affixTab?: boolean;
  component?: string;
  hideInMenu?: boolean;
  icon?: string;
  isPublic?: boolean;
  keepAlive?: boolean;
  name: string;
  path: string;
  pid?: number;
  redirect?: string;
  remark?: string;
  sort?: number;
  status?: number;
  title: string;
  type: 'catalog' | 'menu';
}

export interface MenuGrantInfo {
  checkedIds: number[];
  items: SystemMenuItem[];
  userId: number;
  username: string;
}

/**
 * 菜单管理列表(树形)
 * 超管返回全部菜单; 管理员仅返回自己可下发的菜单(只读)
 */
export async function getMenuListApi() {
  return requestClient.get<{ items: SystemMenuItem[] }>('/menu/list');
}

/**
 * 新增菜单/路由(仅超级管理员)
 */
export async function createMenuApi(data: SystemMenuPayload) {
  return requestClient.post<{ id: number }>('/menu/create', data);
}

/**
 * 修改菜单/路由(仅超级管理员)
 */
export async function updateMenuApi(data: SystemMenuPayload & { id: number }) {
  return requestClient.put('/menu/update', data);
}

/**
 * 删除菜单/路由(仅超级管理员)
 */
export async function deleteMenuApi(id: number) {
  return requestClient.delete(`/menu/delete?id=${id}`);
}

/**
 * 查询目标用户的可授权菜单与已授权菜单
 * 超管: 全部启用菜单; 管理员: 超管下发给自己的菜单
 */
export async function getMenuGrantInfoApi(id: number) {
  return requestClient.get<MenuGrantInfo>('/menu/grant', { params: { id } });
}

/**
 * 保存用户菜单授权(全量覆盖)
 * 管理员保存时会保留超管直接下发、自己无权管理的菜单
 */
export async function grantMenuApi(userId: number, menuIds: number[]) {
  return requestClient.put('/menu/grant', { menuIds, userId });
}
