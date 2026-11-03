import { onMounted, reactive, ref } from 'vue';

/**
 * 通用分页查询参数(与 #/api/types 的 PageParams 结构一致, 避免组合式函数反向依赖 api 层)
 */
interface PageParams {
  page?: number;
  pageSize?: number;
}

/**
 * useTable 的查询条件类型: 业务查询字段 + 必填分页字段
 */
export type TableQuery<Q extends object> = Q & Required<PageParams>;

/**
 * useTable 配置项
 */
export interface UseTableOptions<T, Q extends object> {
  /**
   * 拉取列表数据的请求函数: 入参为查询条件(含 page/pageSize), 返回分页结果
   */
  fetch: (query: TableQuery<Q>) => Promise<{ items: T[]; total: number }>;
  /**
   * 默认查询条件: 初始化 query 与 handleReset 恢复时使用;
   * 可携带初始 page/pageSize(如 pageSize: 20), 但重置时仅恢复筛选字段、不动分页状态
   */
  defaultQuery?: Partial<Q & PageParams>;
  /**
   * 是否在 onMounted 时自动拉取一次, 默认 true
   */
  immediate?: boolean;
  /**
   * 搜索(handleSearch/handleReset)时是否重置回第 1 页, 默认 true
   */
  autoResetPage?: boolean;
  /**
   * 请求前回调: 返回 false 可跳过本次拉取(用于"未选中主表行不发请求"等主从联动场景)
   */
  beforeFetch?: (query: TableQuery<Q>) => false | void | Promise<false | void>;
  /**
   * 请求成功后回调: 此时 items/total 已更新, 可做主从表联动等收尾处理
   */
  afterFetch?: (result: { items: T[]; total: number }) => void;
}

/**
 * 通用表格分页组合式函数(不依赖具体 UI 库)
 *
 * 统一收敛列表页重复的 loading/items/total/query/fetchList/搜索/重置逻辑:
 * - fetchList(): 按当前查询条件拉取
 * - handleSearch(): 重置回第 1 页后拉取
 * - handleReset(): 查询条件恢复默认值后拉取(等价于"清空筛选再搜索")
 * - handlePageChange(page) / handlePageSizeChange(size): 翻页与改每页条数
 *
 * @example
 * const { items, loading, query, total, fetchList, handleSearch, handleReset } =
 *   useTable<UserItem, { username: string }>({
 *     fetch: (q) => getUserListApi({ ...q, username: q.username || undefined }),
 *     defaultQuery: { username: '' },
 *   });
 */
export function useTable<T, Q extends object>(options: UseTableOptions<T, Q>) {
  const {
    afterFetch,
    autoResetPage = true,
    beforeFetch,
    defaultQuery = {},
    fetch,
    immediate = true,
  } = options;

  // 初始查询条件: page/pageSize 在前, 允许 defaultQuery 覆盖初始分页值
  const query = reactive({
    page: 1,
    pageSize: 10,
    ...defaultQuery,
  }) as unknown as TableQuery<Q>;

  const loading = ref(false);
  const items = ref<T[]>([]);
  const total = ref(0);

  async function fetchList() {
    if ((await beforeFetch?.(query)) === false) {
      return;
    }
    loading.value = true;
    try {
      const result = await fetch(query);
      items.value = result.items;
      total.value = result.total;
      afterFetch?.(result);
    } finally {
      loading.value = false;
    }
  }

  function handleSearch() {
    if (autoResetPage) {
      query.page = 1;
    }
    fetchList();
  }

  function handleReset() {
    // 仅恢复筛选字段; page/pageSize 不随重置恢复, 与各页面原有重置语义一致
    const defaults: Partial<Q & PageParams> = { ...defaultQuery };
    delete defaults.page;
    delete defaults.pageSize;
    Object.assign(query, defaults);
    handleSearch();
  }

  function handlePageChange(page: number) {
    query.page = page;
    fetchList();
  }

  function handlePageSizeChange(size: number) {
    query.pageSize = size;
    query.page = 1;
    fetchList();
  }

  if (immediate) {
    onMounted(() => {
      fetchList();
    });
  }

  return {
    fetchList,
    handlePageChange,
    handlePageSizeChange,
    handleReset,
    handleSearch,
    items,
    loading,
    query,
    total,
  };
}
