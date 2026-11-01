package v1

import "github.com/gogf/gf/v2/frame/g"

// LogoutReq 退出登录
type LogoutReq struct {
	g.Meta `path:"/auth/logout" method:"post" tags:"Auth" summary:"退出登录"`
}

// LogoutRes 退出结果
type LogoutRes struct{}
