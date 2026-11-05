package v1

import "github.com/gogf/gf/v2/frame/g"

// DeleteReq 删除定时任务(仅超级管理员)
type DeleteReq struct {
	g.Meta `path:"/job/delete" method:"delete" tags:"Job" summary:"删除定时任务"`
	Id     uint64 `json:"id" v:"required#请指定任务"`
}

// DeleteRes 删除结果
type DeleteRes struct{}
