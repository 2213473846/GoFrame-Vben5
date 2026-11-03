/**
 * 通用分页查询参数
 */
export interface PageParams {
  page?: number;
  pageSize?: number;
}

/**
 * 通用分页列表结果
 */
export interface PageResult<T> {
  items: T[];
  total: number;
}
