package auth

import (
	"context"

	"demo/api/auth/v1"
	"demo/internal/consts"
	"demo/internal/service/middleware"
	onlinesvc "demo/internal/service/online"

	"github.com/gogf/gf/v2/frame/g"
)

// Refresh 刷新 accessToken, 响应体直接为新的 token 字符串(与前端 baseRequestClient 约定)
func (c *ControllerV1) Refresh(ctx context.Context, req *v1.RefreshReq) (res *v1.RefreshRes, err error) {
	r := g.RequestFromCtx(ctx)
	out, err := c.authSvc.Refresh(ctx, r.Cookie.Get(consts.CookieRefreshToken).String())
	if err != nil {
		return nil, err
	}
	// 续期 refreshToken Cookie
	setRefreshTokenCookie(r, out.RefreshToken)
	// 刷新成功: 移除旧 token 的在线会话, 记录新会话
	onlinesvc.Drop(ctx, middleware.GetBearerToken(r))
	onlinesvc.RecordFromRequest(r, out.AccessToken, out.Identity)
	r.Response.Write(out.AccessToken)
	return nil, nil
}
