package v1

import "github.com/gogf/gf/v2/frame/g"

// RunReq 手动执行定时任务(仅超级管理员, 异步受理)
type RunReq struct {
	g.Meta `path:"/job/run" method:"post" tags:"Job" summary:"手动执行定时任务"`
	Id     uint64 `json:"id" v:"required#请指定任务"`
}

// RunRes 受理结果
type RunRes struct {
	Accepted bool `json:"accepted"`
}
