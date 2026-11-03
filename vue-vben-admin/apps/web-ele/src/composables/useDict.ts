import type { DictDataItem } from '#/api/system/dict';

import { onMounted, ref, watch } from 'vue';

import { useAccessStore } from '@vben/stores';

import { getDictBatchApi } from '#/api/system/dict';

export type DictTagType = 'danger' | 'info' | 'primary' | 'success' | 'warning';

export interface DictFallbackItem {
  label: string;
  tagColor?: DictTagType;
}

/**
 * 字典内存缓存:
 * - 同一 type 只请求一次
 * - 切换账号/登出(accessToken 变化)自动失效
 * - 字典管理保存后调用 invalidateDictCache 立即失效并触发已挂载组件重新拉取
 */
const cache = new Map<string, DictDataItem[]>();
const inflight = new Map<string, Promise<DictDataItem[]>>();
let cachedToken: null | string | undefined;
const cacheVersion = ref(0);

function currentToken(): null | string {
  return useAccessStore().accessToken ?? null;
}

function ensureTokenFresh() {
  const token = currentToken();
  if (cachedToken !== undefined && cachedToken !== token) {
    cache.clear();
    inflight.clear();
    cacheVersion.value += 1;
  }
  cachedToken = token;
}

async function loadDict(type: string, force = false): Promise<DictDataItem[]> {
  ensureTokenFresh();
  if (!force && cache.has(type)) {
    return cache.get(type) as DictDataItem[];
  }
  // 强制刷新时若已有相同类型的请求在途(如多行 DictTag 同时收到缓存失效通知), 合并为同一个请求
  if (inflight.has(type)) {
    return inflight.get(type) as Promise<DictDataItem[]>;
  }
  const request = getDictBatchApi([type])
    .then((res) => {
      const items = res?.data?.[type] ?? [];
      cache.set(type, items);
      return items;
    })
    .finally(() => {
      inflight.delete(type);
    });
  inflight.set(type, request);
  return request;
}

/**
 * 清空字典缓存并触发已挂载的 useDict 重新加载
 */
export function invalidateDictCache() {
  cache.clear();
  inflight.clear();
  cacheVersion.value += 1;
}

/**
 * 字典组合式函数
 * @param type 字典类型编码
 * @param fallback 取不到字典时的兜底文案, 避免页面白屏
 */
export function useDict(
  type: string,
  fallback: Record<string, DictFallbackItem> = {},
) {
  const items = ref<DictDataItem[]>(cache.get(type) ?? []);
  const loading = ref(false);

  async function reload(force = false) {
    loading.value = true;
    try {
      items.value = await loadDict(type, force);
    } catch {
      // 接口异常时不阻塞页面, 使用兜底文案
      items.value = [];
    } finally {
      loading.value = false;
    }
  }

  function findItem(value: number | string): DictDataItem | undefined {
    const target = String(value);
    return items.value.find((item) => item.value === target);
  }

  function labelOf(value: number | string, defaultLabel = ''): string {
    return (
      findItem(value)?.label ??
      fallback[String(value)]?.label ??
      defaultLabel ??
      String(value)
    );
  }

  function tagOf(value: number | string): DictTagType {
    const color = findItem(value)?.tagColor || fallback[String(value)]?.tagColor;
    const allowed: DictTagType[] = [
      'danger',
      'info',
      'primary',
      'success',
      'warning',
    ];
    return allowed.includes(color as DictTagType)
      ? (color as DictTagType)
      : 'info';
  }

  onMounted(() => {
    reload();
  });

  // 缓存被清除(管理员改字典/切换账号)时, 已挂载组件自动刷新
  watch(cacheVersion, () => reload(true));

  return {
    items,
    labelOf,
    loading,
    reload,
    tagOf,
  };
}
