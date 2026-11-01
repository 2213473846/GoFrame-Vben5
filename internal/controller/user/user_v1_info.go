package user

import (
	"context"

	"demo/api/user/v1"
	"demo/internal/consts"
	"demo/internal/service/identity"
	menusvc "demo/internal/service/menu"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/gconv"
)

var roleNames = map[string]string{
	consts.RoleSuperAdmin: "超级管理员",
	consts.RoleAdmin:      "管理员",
	consts.RoleUser:       "普通用户",
}

// Info 当前登录用户信息
func (c *ControllerV1) Info(ctx context.Context, req *v1.InfoReq) (res *v1.InfoRes, err error) {
	id := identity.Get(ctx)
	if id == nil {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
	}
	res = &v1.InfoRes{
		UserId:   gconv.String(id.Id),
		Username: id.Username,
		RealName: id.RealName,
		Avatar:   id.Avatar,
		Desc:     roleNames[id.Role],
		HomePath: homePath(ctx),
		Roles:    id.Roles(),
	}
	return
}

// homePath 按当前用户实际可见的菜单计算登录后的默认首页
func homePath(ctx context.Context) string {
	path, err := menusvc.New().FirstPath(ctx)
	if err != nil || path == "" {
		// 计算失败时回退到个人中心, 避免跳转到不存在的路由
		return "/profile"
	}
	return path
}
