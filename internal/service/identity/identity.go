package identity

import (
	"context"

	"demo/internal/consts"
)

// Identity 当前登录用户身份
type Identity struct {
	Id       uint64 `json:"id"`
	Pid      uint64 `json:"pid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	RealName string `json:"realName"`
	Avatar   string `json:"avatar"`
}

// Roles 返回角色数组(对齐前端 UserInfo.roles)
func (i Identity) Roles() []string {
	return []string{i.Role}
}

// IsSuperAdmin 是否超级管理员
func (i Identity) IsSuperAdmin() bool {
	return i.Role == consts.RoleSuperAdmin
}

type ctxKey struct{}

// With 将身份写入上下文
func With(ctx context.Context, in Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, in)
}

// Get 从上下文获取身份, 未登录返回 nil
func Get(ctx context.Context) *Identity {
	if v, ok := ctx.Value(ctxKey{}).(Identity); ok && v.Id > 0 {
		return &v
	}
	return nil
}
