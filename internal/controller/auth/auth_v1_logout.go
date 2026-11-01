package auth

import (
	"context"
	"net/http"

	"demo/api/auth/v1"
	"demo/internal/consts"
	"demo/internal/service/middleware"
	onlinesvc "demo/internal/service/online"

	"github.com/gogf/gf/v2/frame/g"
)

// Logout 退出登录, 作废 token 并清除 Cookie
func (c *ControllerV1) Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error) {
	r := g.RequestFromCtx(ctx)
	accessToken := middleware.GetBearerToken(r)
	c.authSvc.Logout(ctx, accessToken, r.Cookie.Get(consts.CookieRefreshToken).String())
	// 移除在线用户会话
	onlinesvc.Drop(ctx, accessToken)
	r.Cookie.SetHttpCookie(&http.Cookie{
		Name:     consts.CookieRefreshToken,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return &v1.LogoutRes{}, nil
}
