package auth

import (
	"context"
	"net/http"

	"demo/api/auth/v1"
	"demo/internal/consts"
	"demo/internal/service/identity"
	logsvc "demo/internal/service/log"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
)

// ChangePassword 修改当前用户密码; 成功后该用户全部旧会话失效, 需重新登录
func (c *ControllerV1) ChangePassword(ctx context.Context, req *v1.ChangePasswordReq) (res *v1.ChangePasswordRes, err error) {
	id := identity.Get(ctx)
	if id == nil {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
	}
	r := g.RequestFromCtx(ctx)
	if err = c.authSvc.ChangePassword(ctx, id.Id, req.OldPassword, req.NewPassword); err != nil {
		logsvc.RecordLoginFromRequest(r, id.Id, id.Username, false, "修改密码失败: "+err.Error())
		return nil, err
	}
	logsvc.RecordLoginFromRequest(r, id.Id, id.Username, true, "密码修改成功, 旧会话已失效")
	// 旧会话已全部作废, 清理 refreshToken Cookie
	r.Cookie.SetHttpCookie(&http.Cookie{
		Name:     consts.CookieRefreshToken,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return &v1.ChangePasswordRes{}, nil
}
