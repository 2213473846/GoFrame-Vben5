import { silentRequestClient } from '#/api/request';

/** CPU 指标 */
export interface ServerCpu {
  /** 采集是否成功, false 时页面展示"暂无数据" */
  available: boolean;
  /** 逻辑核数 */
  cores: number;
  /** 使用率(百分比) */
  usagePercent: number;
}

/** 物理内存指标(单位: 字节) */
export interface ServerMemory {
  available: boolean;
  total: number;
  used: number;
  usagePercent: number;
}

/** 磁盘分区指标(单位: 字节) */
export interface ServerDisk {
  mountpoint: string;
  total: number;
  used: number;
  usagePercent: number;
}

/** Go 运行时指标 */
export interface ServerRuntime {
  goroutines: number;
  heapAllocBytes: number;
  heapSysBytes: number;
  numGC: number;
}

/** 单个依赖健康状态(后端只下发 ok/fail 与延迟, 不含错误详情) */
export interface ServerDep {
  status: string;
  latencyMs: number;
}

/** 依赖健康状态 */
export interface ServerDeps {
  mysql: ServerDep;
  redis: ServerDep;
}

/** 服务监控结果 */
export interface ServerInfo {
  cpu: ServerCpu;
  deps: ServerDeps;
  disks: ServerDisk[];
  /** 磁盘分区枚举是否成功, false 时页面展示"暂无数据" */
  disksAvailable: boolean;
  memory: ServerMemory;
  /** 自动刷新间隔(秒), 由后端配置下发 */
  refreshSeconds: number;
  runtime: ServerRuntime;
  /** 进程已运行时长(秒) */
  uptimeSeconds: number;
}

/** 错误对象中与 HTTP 响应相关的结构 */
interface ErrorResponse {
  data?: {
    code?: number;
    message?: string;
  };
  status?: number;
}

/**
 * 收窄并取出错误对象中的 response, 非 HTTP 错误(如代码异常)返回 undefined
 */
function pickResponse(error: unknown): ErrorResponse | undefined {
  if (typeof error !== 'object' || error === null) {
    return undefined;
  }
  const response = (error as { response?: unknown }).response;
  if (typeof response !== 'object' || response === null) {
    return undefined;
  }
  return response as ErrorResponse;
}

/**
 * 是否为禁止访问(403): 后端业务权限错误统一返回 HTTP 403 + body code -1,
 * 4030 为强制改密(由全局拦截器处理, 不算本页的权限拒绝)
 */
export function isForbiddenError(error: unknown): boolean {
  const response = pickResponse(error);
  return response?.status === 403 && response.data?.code !== 4030;
}

/**
 * 服务监控(仅超级管理员; 其他角色后端返回 403, 页面展示空状态)
 */
export async function getServerInfoApi() {
  return silentRequestClient.get<ServerInfo>('/monitor/server');
}
