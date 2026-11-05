/**
 * 该文件可自行根据业务逻辑进行调整
 */
import type { RequestClientOptions } from '@vben/request';

import { useAppConfig } from '@vben/hooks';
import { preferences } from '@vben/preferences';
import {
  authenticateResponseInterceptor,
  defaultResponseInterceptor,
  errorMessageResponseInterceptor,
  RequestClient,
} from '@vben/request';
import { useAccessStore } from '@vben/stores';

import { ElMessage } from 'element-plus';

import { useAuthStore } from '#/store';

import { refreshTokenApi } from './core';

const { apiURL } = useAppConfig(import.meta.env, import.meta.env.PROD);

function createRequestClient(
  baseURL: string,
  options?: RequestClientOptions,
  silent = false,
) {
  const client = new RequestClient({
    ...options,
    baseURL,
  });

  /**
   * 重新认证逻辑
   */
  async function doReAuthenticate() {
    console.warn('Access token or refresh token is invalid or expired. ');
    const accessStore = useAccessStore();
    const authStore = useAuthStore();
    accessStore.setAccessToken(null);
    if (
      preferences.app.loginExpiredMode === 'modal' &&
      accessStore.isAccessChecked
    ) {
      accessStore.setLoginExpired(true);
    } else {
      await authStore.logout();
    }
  }

  /**
   * 刷新token逻辑
   */
  async function doRefreshToken() {
    const accessStore = useAccessStore();
    const resp = await refreshTokenApi();
    const newToken = resp.data;
    accessStore.setAccessToken(newToken);
    return newToken;
  }

  function formatToken(token: null | string) {
    return token ? `Bearer ${token}` : null;
  }

  // 请求头处理
  client.addRequestInterceptor({
    fulfilled: async (config) => {
      const accessStore = useAccessStore();

      config.headers.Authorization = formatToken(accessStore.accessToken);
      config.headers['Accept-Language'] = preferences.app.locale;
      return config;
    },
  });

  // 处理返回的响应数据格式
  client.addResponseInterceptor(
    defaultResponseInterceptor({
      codeField: 'code',
      dataField: 'data',
      successCode: 0,
    }),
  );

  // 会话被顶替/超出设备数上限/管理员强制下线: HTTP 401 + body code 4001
  // 必须放在刷新拦截器之前: 直接提示业务文案并回到登录页, 不允许旧 refreshToken 再续命
  client.addResponseInterceptor({
    rejected: async (error) => {
      const response = error?.response;
      const body = response?.data ?? {};
      if (response?.status === 401 && body?.code === 4001) {
        const msg =
          body?.message || body?.error || '登录状态已失效, 请重新登录';
        ElMessage.error(msg);
        // 标记为已重试, 让后续认证拦截器跳过 refresh 流程
        if (error.config) {
          error.config.__isRetryRequest = true;
        }
        await doReAuthenticate();
      }
      throw error;
    },
  });

  // 强制改密: HTTP 403 + body code 4030(未改密前后端只放行改密与登出)
  // 登录页在拿到登录响应后会自行弹窗引导改密, 这里兜底处理会话中途被要求改密的情况
  client.addResponseInterceptor({
    rejected: async (error) => {
      const response = error?.response;
      const body = response?.data ?? {};
      if (response?.status === 403 && body?.code === 4030) {
        ElMessage.error(body?.message || '请先修改初始密码后再使用系统');
        if (error.config) {
          error.config.__isRetryRequest = true;
        }
        await doReAuthenticate();
      }
      throw error;
    },
  });

  // token过期的处理
  client.addResponseInterceptor(
    authenticateResponseInterceptor({
      client,
      doReAuthenticate,
      doRefreshToken,
      enableRefreshToken: preferences.app.enableRefreshToken,
      formatToken,
    }),
  );

  // 通用的错误处理,如果没有进入上面的错误处理逻辑，就会进入这里
  // silent 客户端不注册该拦截器: 由页面自行展示错误(如服务监控 403 需展示空状态而不弹提示)
  if (!silent) {
    client.addResponseInterceptor(
      errorMessageResponseInterceptor((msg: string, error) => {
        // 这里可以根据业务进行定制,你可以拿到 error 内的信息进行定制化处理，根据不同的 code 做不同的提示，而不是直接使用 message.error 提示 msg
        // 当前mock接口返回的错误字段是 error 或者 message
        const responseData = error?.response?.data ?? {};
        const errorMessage = responseData?.error ?? responseData?.message ?? '';
        // 如果没有错误信息，则会根据状态码进行提示
        ElMessage.error(errorMessage || msg);
      }),
    );
  }

  return client;
}

export const requestClient = createRequestClient(apiURL, {
  responseReturn: 'data',
});

/**
 * 不注册通用错误提示的客户端: 需要页面自行处理错误提示(如 403 展示空状态)的接口使用,
 * 登录态/token 刷新逻辑与 requestClient 保持一致
 */
export const silentRequestClient = createRequestClient(
  apiURL,
  { responseReturn: 'data' },
  true,
);

export const baseRequestClient = new RequestClient({ baseURL: apiURL });
