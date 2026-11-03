package auth

import (
	"context"

	"demo/api/auth/v1"
	"demo/internal/consts"

	"github.com/gogf/gf/v2/frame/g"
)

// Refresh 刷新 accessToken, 响应体直接为新的 token 字符串(与前端 baseRequestClient 约定)
func (c *ControllerV1) Refresh(ctx context.Context, req *v1.RefreshReq) (res *v1.RefreshRes, err error) {
	r := g.RequestFromCtx(ctx)
	out, err := c.authSvc.Refresh(ctx, r.Cookie.Get(consts.CookieRefreshToken).String())
	if err != nil {
		return nil, err
	}
	// refreshToken 不轮换, 同一浏览器多标签共享 Cookie 并发刷新时保持稳定
	setRefreshTokenCookie(r, out.RefreshToken)
	// 在线会话按 sessionId 维护, 刷新只续期, 不新建会话记录
	r.Response.Write(out.AccessToken)
	return nil, nil
}
