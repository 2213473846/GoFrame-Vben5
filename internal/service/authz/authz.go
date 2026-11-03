// Package authz 统一的服务层权限守卫: 收口登录态/角色校验, 错误码与文案只有一处真相。
package authz

import (
	"context"

	"demo/internal/consts"
	"demo/internal/service/identity"

	"github.com/gogf/gf/v2/errors/gerror"
)

// Login 校验登录态: 未登录返回 401, 已登录返回身份。
func Login(ctx context.Context) (*identity.Identity, error) {
	id := identity.Get(ctx)
	if id == nil {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
	}
	return id, nil
}

// Manager 校验登录且为超管或管理员; msgs[0] 可覆盖默认拒绝文案。
func Manager(ctx context.Context, msgs ...string) (*identity.Identity, error) {
	id, err := Login(ctx)
	if err != nil {
		return nil, err
	}
	if !id.IsSuperAdmin() && id.Role != consts.RoleAdmin {
		return nil, gerror.NewCode(consts.CodeForbidden, defaultMsg(msgs, "无权限执行此操作"))
	}
	return id, nil
}

// SuperAdmin 校验登录且为超级管理员; msgs[0] 可覆盖默认拒绝文案。
func SuperAdmin(ctx context.Context, msgs ...string) (*identity.Identity, error) {
	id, err := Login(ctx)
	if err != nil {
		return nil, err
	}
	if !id.IsSuperAdmin() {
		return nil, gerror.NewCode(consts.CodeForbidden, defaultMsg(msgs, "仅超级管理员可执行此操作"))
	}
	return id, nil
}

func defaultMsg(msgs []string, def string) string {
	if len(msgs) > 0 && msgs[0] != "" {
		return msgs[0]
	}
	return def
}
