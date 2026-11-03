package auth

import (
	"context"
	"net/http"

	"demo/api/auth/v1"
	"demo/internal/consts"
	authsvc "demo/internal/service/auth"
	logsvc "demo/internal/service/log"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
)

// Login 登录
func (c *ControllerV1) Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error) {
	r := g.RequestFromCtx(ctx)
	browser, osName := logsvc.ParseUserAgent(r.Header.Get("User-Agent"))
	out, err := c.authSvc.Login(ctx, authsvc.LoginInput{
		Username:    req.Username,
		Password:    req.Password,
		CaptchaId:   req.CaptchaId,
		CaptchaCode: req.CaptchaCode,
		Meta: authsvc.LoginMeta{
			Ip:      r.GetClientIp(),
			Browser: browser,
			Os:      osName,
		},
	})
	if err != nil {
		// 登录失败也记录日志(失败原因已统一为"用户名或密码错误"等, 不暴露账号是否存在)
		logsvc.RecordLoginFromRequest(r, 0, req.Username, false, err.Error())
		return nil, err
	}
	msg := "登录成功"
	if out.MustChangePassword {
		msg = "登录成功, 首次登录需修改密码"
	}
	logsvc.RecordLoginFromRequest(r, out.Identity.Id, out.Identity.Username, true, msg)
	setRefreshTokenCookie(r, out.RefreshToken)
	res = &v1.LoginRes{
		AccessToken:        out.AccessToken,
		UserId:             gconv.String(out.Identity.Id),
		Username:           out.Identity.Username,
		RealName:           out.Identity.RealName,
		Avatar:             out.Identity.Avatar,
		Roles:              out.Identity.Roles(),
		MustChangePassword: out.MustChangePassword,
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
