import type { PageParams, PageResult } from '#/api/types';

import { silentRequestClient } from '#/api/request';

/** 定时任务条目 */
export interface JobItem {
  concurrencyPolicy: number;
  createdAt: string;
  cronExpression: string;
  id: number;
  /** 任务标识(仅内置已注册任务, 如 clearlog) */
  invokeTarget: string;
  jobGroup: string;
  lastResult: string;
  lastRunAt: string;
  name: string;
  nextRunAt: string;
  remark: string;
  /** 1=启用 0=停用 */
  status: number;
  timeoutSeconds: number;
  updatedAt: string;
}

/** 新增/修改任务的提交字段 */
export interface JobPayload {
  concurrencyPolicy: number;
  cronExpression: string;
  invokeTarget: string;
  jobGroup: string;
  name: string;
  remark: string;
  status: number;
  timeoutSeconds: number;
}

export interface JobListParams extends PageParams {
  name?: string;
  /** -1=全部 1=启用 0=停用 */
  status?: number;
}

export type JobListResult = PageResult<JobItem>;

/** 执行日志条目 */
export interface JobLogItem {
  createdAt: string;
  durationMs: number;
  finishedAt: string;
  id: number;
  jobId: number;
  jobName: string;
  message: string;
  startedAt: string;
  /** 1=成功 0=失败 */
  status: number;
  /** 1=定时触发 2=手动触发 */
  triggerType: number;
}

export interface JobLogListParams extends PageParams {
  jobId?: number;
  /** -1=全部 1=成功 0=失败 */
  status?: number;
}

export type JobLogListResult = PageResult<JobLogItem>;

/**
 * 定时任务列表(超管全部, 管理员只读)
 */
export async function listJobsApi(params: JobListParams) {
  return silentRequestClient.get<JobListResult>('/job/list', { params });
}

/**
 * 新增定时任务(仅超级管理员)
 */
export async function createJobApi(data: JobPayload) {
  return silentRequestClient.post<{ id: number }>('/job/create', data);
}

/**
 * 修改定时任务(仅超级管理员)
 */
export async function updateJobApi(data: JobPayload & { id: number }) {
  return silentRequestClient.put('/job/update', data);
}

/**
 * 删除定时任务(仅超级管理员, 软删除)
 */
export async function deleteJobApi(id: number) {
  return silentRequestClient.delete(`/job/delete?id=${id}`);
}

/**
 * 启停定时任务(仅超级管理员)
 */
export async function changeJobStatusApi(data: { id: number; status: number }) {
  return silentRequestClient.put('/job/status', data);
}

/**
 * 手动执行一次(仅超级管理员, 异步受理后立即返回)
 */
export async function runJobApi(id: number) {
  return silentRequestClient.post<{ accepted: boolean }>('/job/run', {
    id,
  });
}

/**
 * 执行日志列表(超管/管理员可读)
 */
export async function listJobLogsApi(params: JobLogListParams) {
  return silentRequestClient.get<JobLogListResult>('/job/logs', { params });
}
