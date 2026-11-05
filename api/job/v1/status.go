package v1

import "github.com/gogf/gf/v2/frame/g"

// StatusReq 启停定时任务(仅超级管理员)
type StatusReq struct {
	g.Meta `path:"/job/status" method:"put" tags:"Job" summary:"启停定时任务"`
	Id     uint64 `json:"id" v:"required#请指定任务"`
	Status int    `json:"status" v:"in:0,1#状态值非法"` // 1=启用 0=停用
}

// StatusRes 启停结果
type StatusRes struct{}
