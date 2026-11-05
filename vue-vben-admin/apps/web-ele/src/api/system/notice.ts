import type { PageParams, PageResult } from '#/api/types';

import { requestClient } from '#/api/request';

/**
 * 管理端公告条目
 */
export interface SystemNoticeItem {
  content: string;
  createdAt: string;
  /** 失效时间, 空串表示永不过期 */
  expireAt: string;
  id: number;
  /** 级别(字典 notice_level) */
  level: string;
  /** 发布时间, 草稿可能为空 */
  publishAt: string;
  publisherId: number;
  /** 0=草稿 1=已发布 2=已撤回 */
  status: number;
  /** 1=全员 2=指定用户 */
  targetScope: number;
  /** 定向用户数, 全员时为 0 */
  targetCount: number;
  title: string;
  /** 类型(字典 notice_type) */
  type: string;
  updatedAt: string;
}

export type SystemNoticeListResult = PageResult<SystemNoticeItem>;

/**
 * 管理端列表查询条件(标题模糊查询后端字段名为 keyword)
 */
export interface SystemNoticeListParams extends PageParams {
  endTime?: string;
  keyword?: string;
  level?: string;
  startTime?: string;
  /** -1=全部 0=草稿 1=已发布 2=已撤回 */
  status?: number;
  type?: string;
}

/**
 * 公告新增/编辑提交结构
 */
export interface NoticeFormData {
  content: string;
  expireAt: string;
  level: string;
  publishAt: string;
  targetScope: number;
  targetUserIds: number[];
  title: string;
  type: string;
}

export interface SystemNoticeDetailResult {
  item: SystemNoticeItem;
  /** 定向用户 ID, 供编辑回填 */
  targetUserIds: number[];
}

/**
 * 公告列表(超管全部, 管理员仅自己创建的公告)
 */
export async function getSystemNoticeListApi(params: SystemNoticeListParams) {
  return requestClient.get<SystemNoticeListResult>('/notice/list', { params });
}

/**
 * 公告详情(管理员仅限自己创建的公告)
 */
export async function getSystemNoticeDetailApi(id: number) {
  return requestClient.get<SystemNoticeDetailResult>('/notice/detail', {
    params: { id },
  });
}

/**
 * 新增公告(后端默认落为草稿)
 */
export async function createNoticeApi(data: NoticeFormData) {
  return requestClient.post<{ id: number }>('/notice/create', data);
}

/**
 * 修改公告
 */
export async function updateNoticeApi(data: NoticeFormData & { id: number }) {
  return requestClient.put('/notice/update', data);
}

/**
 * 删除公告(后端 DELETE 接口, 同时携带 query 与 JSON body, 兼容现有删除封装与后端 json 参数绑定)
 */
export async function deleteNoticeApi(id: number) {
  return requestClient.delete('/notice/delete', {
    data: { id },
    params: { id },
  });
}

/**
 * 发布公告(仅草稿/已撤回可发布, publishAt 为空取当前时间)
 */
export async function publishNoticeApi(data: { id: number; publishAt?: string }) {
  return requestClient.put('/notice/publish', data);
}

/**
 * 撤回公告(仅已发布可撤回)
 */
export async function revokeNoticeApi(id: number) {
  return requestClient.put('/notice/revoke', { id });
}
