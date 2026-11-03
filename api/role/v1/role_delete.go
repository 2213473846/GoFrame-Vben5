package v1

import "github.com/gogf/gf/v2/frame/g"

// DeleteReq 删除角色(仅超级管理员, 级联清理角色菜单授权与用户关联)
type DeleteReq struct {
	g.Meta `path:"/role/delete" method:"delete" tags:"Role" summary:"删除角色"`
	Id     uint64 `json:"id" v:"required#请指定角色ID"`
}

// DeleteRes 删除结果
type DeleteRes struct{}
