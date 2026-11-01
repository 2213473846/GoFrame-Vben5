package auth

import (
	"context"
	"net/http"

	"demo/api/auth/v1"
	"demo/internal/consts"
	logsvc "demo/internal/service/log"
	onlinesvc "demo/internal/service/online"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
)

// Login 登录
func (c *ControllerV1) Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error) {
	r := g.RequestFromCtx(ctx)
	out, err := c.authSvc.Login(ctx, req.Username, req.Password)
	if err != nil {
		// 登录失败也记录日志(失败原因已统一为"用户名或密码错误"等, 不暴露账号是否存在)
		logsvc.RecordLoginFromRequest(r, 0, req.Username, false, err.Error())
		return nil, err
	}
	logsvc.RecordLoginFromRequest(r, out.Identity.Id, out.Identity.Username, true, "登录成功")
	// 写入在线用户会话(使用刚签发的 accessToken)
	onlinesvc.RecordFromRequest(r, out.AccessToken, out.Identity)
	setRefreshTokenCookie(r, out.RefreshToken)
	res = &v1.LoginRes{
		AccessToken: out.AccessToken,
		UserId:      gconv.String(out.Identity.Id),
		Username:    out.Identity.Username,
		RealName:    out.Identity.RealName,
		Avatar:      out.Identity.Avatar,
		Roles:       out.Identity.Roles(),
	}
	return
}

// setRefreshTokenCookie 将 refreshToken 写入 HttpOnly Cookie(前端刷新token时携带)
func setRefreshTokenCookie(r *ghttp.Request, value string) {
	r.Cookie.SetHttpCookie(&http.Cookie{
		Name:     consts.CookieRefreshToken,
		Value:    value,
		Path:     "/",
		MaxAge:   consts.RefreshTTL,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
