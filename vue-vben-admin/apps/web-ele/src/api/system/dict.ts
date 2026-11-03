import type { PageParams, PageResult } from '#/api/types';

import { requestClient } from '#/api/request';

export interface DictTypeItem {
  createdAt: string;
  id: number;
  name: string;
  remark: string;
  status: number;
  type: string;
}

export interface DictDataItem {
  dictType: string;
  id: number;
  label: string;
  remark: string;
  sort: number;
  status: number;
  tagColor: string;
  value: string;
}

export interface DictTypeListParams {
  keyword?: string;
  status?: number;
}

export interface DictDataListParams extends PageParams {
  dictType: string;
  keyword?: string;
  status?: number;
}

/**
 * 字典类型列表(超管/管理员可读)
 */
export async function getDictTypeListApi(params?: DictTypeListParams) {
  return requestClient.get<{ items: DictTypeItem[] }>('/dict/type/list', {
    params,
  });
}

/**
 * 新增字典类型(仅超级管理员)
 */
export async function createDictTypeApi(data: {
  name: string;
  remark?: string;
  status?: number;
  type: string;
}) {
  return requestClient.post<{ id: number }>('/dict/type/create', data);
}

/**
 * 修改字典类型(编码不可变更, 仅超级管理员)
 */
export async function updateDictTypeApi(data: {
  id: number;
  name: string;
  remark?: string;
  status?: number;
}) {
  return requestClient.put('/dict/type/update', data);
}

/**
 * 删除字典类型(级联删除字典项, 仅超级管理员)
 */
export async function deleteDictTypeApi(id: number) {
  return requestClient.delete(`/dict/type/delete?id=${id}`);
}

/**
 * 启用/停用字典类型(仅超级管理员)
 */
export async function updateDictTypeStatusApi(id: number, status: number) {
  return requestClient.put('/dict/type/status', { id, status });
}

/**
 * 字典项列表(管理端, 含停用项)
 */
export async function getDictDataListApi(params: DictDataListParams) {
  return requestClient.get<PageResult<DictDataItem>>('/dict/data/list', {
    params,
  });
}

/**
 * 新增字典项(仅超级管理员)
 */
export async function createDictDataApi(data: {
  dictType: string;
  label: string;
  remark?: string;
  sort?: number;
  status?: number;
  tagColor?: string;
  value: string;
}) {
  return requestClient.post<{ id: number }>('/dict/data/create', data);
}

/**
 * 修改字典项(仅超级管理员)
 */
export async function updateDictDataApi(data: {
  dictType?: string;
  id: number;
  label: string;
  remark?: string;
  sort?: number;
  status?: number;
  tagColor?: string;
  value: string;
}) {
  return requestClient.put('/dict/data/update', data);
}

/**
 * 删除字典项(仅超级管理员)
 */
export async function deleteDictDataApi(id: number) {
  return requestClient.delete(`/dict/data/delete?id=${id}`);
}

/**
 * 启用/停用字典项(仅超级管理员)
 */
export async function updateDictDataStatusApi(id: number, status: number) {
  return requestClient.put('/dict/data/status', { id, status });
}

/**
 * 按 type 批量取值(任意登录用户可用, 仅返回启用项)
 */
export async function getDictBatchApi(types: string[]) {
  return requestClient.get<{ data: Record<string, DictDataItem[]> }>(
    '/dict/data/batch',
    { params: { types: types.join(',') } },
  );
}
