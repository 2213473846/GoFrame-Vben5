package v1

import "github.com/gogf/gf/v2/frame/g"

// PurgeReq 彻底删除(物理删除, 不可恢复, 级联清除其已删子孙)
type PurgeReq struct {
	g.Meta `path:"/user/purge" method:"delete" tags:"User" summary:"彻底删除用户"`
	Id     uint64 `json:"id" v:"required#请指定用户ID"`
}

// PurgeRes 彻底删除结果
type PurgeRes struct{}
