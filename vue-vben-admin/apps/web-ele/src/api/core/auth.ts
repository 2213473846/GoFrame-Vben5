import { baseRequestClient, requestClient } from '#/api/request';

export namespace AuthApi {
  /** 登录接口参数(验证码字段仅在失败次数达到阈值后由后端要求) */
  export interface LoginParams {
    captchaCode?: string;
    captchaId?: string;
    password?: string;
    username?: string;
  }

  /** 登录接口返回值 */
  export interface LoginResult {
    accessToken: string;
    /** true 表示首次登录/弱密码, 需先修改密码 */
    mustChangePassword?: boolean;
  }

  /** 图形验证码 */
  export interface CaptchaResult {
    captchaId: string;
    expireSeconds: number;
    /** data:image/svg+xml;base64,... 可直接作为 img src */
    image: string;
  }

  /** 修改密码参数 */
  export interface ChangePasswordParams {
    newPassword: string;
    oldPassword: string;
  }

  export interface RefreshTokenResult {
    data: string;
    status: number;
  }
}

/**
 * 登录
 */
export async function loginApi(data: AuthApi.LoginParams) {
  return requestClient.post<AuthApi.LoginResult>('/auth/login', data);
}

/**
 * 获取图形验证码(登录失败达到阈值后使用)
 */
export async function getCaptchaApi() {
  return requestClient.get<AuthApi.CaptchaResult>('/auth/captcha');
}

/**
 * 修改当前用户密码(首次登录强制改密也走这里)
 */
export async function changePasswordApi(data: AuthApi.ChangePasswordParams) {
  return requestClient.post<null>('/auth/change-password', data);
}

/**
 * 刷新accessToken
 */
export async function refreshTokenApi() {
  return baseRequestClient.post<AuthApi.RefreshTokenResult>(
    '/auth/refresh',
    undefined,
    {
      withCredentials: true,
    },
  );
}

/**
 * 退出登录
 */
export async function logoutApi() {
  return baseRequestClient.post('/auth/logout', undefined, {
    withCredentials: true,
  });
}

/**
 * 获取用户权限码
 */
export async function getAccessCodesApi() {
  return requestClient.get<string[]>('/auth/codes');
}
