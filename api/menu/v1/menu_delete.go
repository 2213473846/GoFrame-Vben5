package v1

import "github.com/gogf/gf/v2/frame/g"

// DeleteReq 删除菜单/路由(仅超级管理员)
type DeleteReq struct {
	g.Meta `path:"/menu/delete" method:"delete" tags:"Menu" summary:"删除菜单/路由"`
	Id     uint64 `json:"id" v:"required#请指定菜单ID"`
}

// DeleteRes 删除结果
type DeleteRes struct{}
