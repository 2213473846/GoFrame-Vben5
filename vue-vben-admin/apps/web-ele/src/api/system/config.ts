import { requestClient } from '#/api/request';

export interface ConfigItem {
  builtin: number;
  configKey: string;
  configValue: string;
  createdAt: string;
  id: number;
  name: string;
  remark: string;
  status: number;
  updatedAt: string;
}

export interface ConfigListParams {
  keyword?: string;
  status?: number;
}

/**
 * 系统参数列表(任意已登录用户可读)
 */
export async function getConfigListApi(params?: ConfigListParams) {
  return requestClient.get<{ items: ConfigItem[] }>('/config/list', { params });
}

/**
 * 按 key 读取单个参数(任意已登录用户可读)
 */
export async function getConfigApi(key: string) {
  return requestClient.get<{ configValue: string }>('/config/get', {
    params: { key },
  });
}

/**
 * 批量读取常用参数(任意已登录用户可读)
 */
export async function getConfigBatchApi(keys: string[]) {
  return requestClient.get<{ values: Record<string, string> }>(
    '/config/batch',
    { params: { keys: keys.join(',') } },
  );
}

/**
 * 修改参数值(仅超级管理员)
 */
export async function updateConfigApi(data: { key: string; value: string }) {
  return requestClient.put('/config/update', data);
}

/**
 * 重置内置参数为默认值(仅超级管理员)
 */
export async function resetConfigApi(key: string) {
  return requestClient.put<{ configValue: string }>('/config/reset', { key });
}
