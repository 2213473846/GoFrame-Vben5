import type { PageParams, PageResult } from '#/api/types';

import { requestClient } from '#/api/request';

export interface LogListParams extends PageParams {
  status?: number; // -1=全部 1=成功 0=失败
  username?: string;
}

export interface LoginLogItem {
  browser: string;
  createdAt: string;
  id: number;
  ip: string;
  location: string;
  message: string;
  os: string;
  status: number;
  username: string;
}

export interface OperationLogItem {
  action: string;
  browser: string;
  createdAt: string;
  id: number;
  ip: string;
  location: string;
  message: string;
  method: string;
  module: string;
  os: string;
  path: string;
  status: number;
  username: string;
}

export type LogListResult<T> = PageResult<T>;

/**
 * 登录日志列表(数据范围: 超管全部, 管理员自己+新增用户, 用户仅自己)
 */
export async function getLoginLogListApi(params: LogListParams) {
  return requestClient.get<LogListResult<LoginLogItem>>('/loginlog/list', {
    params,
  });
}

/**
 * 操作日志列表(数据范围: 超管全部, 管理员自己+新增用户, 用户仅自己)
 */
export async function getOperationLogListApi(params: LogListParams) {
  return requestClient.get<LogListResult<OperationLogItem>>(
    '/operationlog/list',
    { params },
  );
}
