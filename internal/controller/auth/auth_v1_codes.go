package auth

import (
	"context"

	"demo/api/auth/v1"
	"demo/internal/consts"
	"demo/internal/service/identity"

	"github.com/gogf/gf/v2/errors/gerror"
)

// Codes 获取当前用户权限码
func (c *ControllerV1) Codes(ctx context.Context, req *v1.CodesReq) (res *v1.CodesRes, err error) {
	id := identity.Get(ctx)
	if id == nil {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
	}
	res = new(v1.CodesRes)
	*res = c.authSvc.Codes(id.Role)
	return
}
