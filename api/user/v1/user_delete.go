package v1

import "github.com/gogf/gf/v2/frame/g"

// DeleteReq 删除用户(软删除, 代理删除时级联删除其子孙)
type DeleteReq struct {
	g.Meta `path:"/user/delete" method:"delete" tags:"User" summary:"删除代理/用户"`
	Id     uint64 `json:"id" v:"required#请指定用户ID"`
}

// DeleteRes 删除结果
type DeleteRes struct{}
