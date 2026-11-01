package v1

import "github.com/gogf/gf/v2/frame/g"

// CodesReq 获取当前用户权限码
type CodesReq struct {
	g.Meta `path:"/auth/codes" method:"get" tags:"Auth" summary:"获取用户权限码"`
}

// CodesRes 权限码数组(整体作为 data 返回)
type CodesRes []string
