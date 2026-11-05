package v1

import "github.com/gogf/gf/v2/frame/g"

// CreateReq 新增定时任务(仅超级管理员)
type CreateReq struct {
	g.Meta            `path:"/job/create" method:"post" tags:"Job" summary:"新增定时任务"`
	Name              string `json:"name" v:"required#请填写任务名称"`
	JobGroup          string `json:"jobGroup" d:"default"`
	InvokeTarget      string `json:"invokeTarget" v:"required#请填写任务标识"`
	CronExpression    string `json:"cronExpression" v:"required#请填写 cron 表达式"`
	Status            int    `json:"status" d:"1"`
	ConcurrencyPolicy int    `json:"concurrencyPolicy" d:"0"`
	TimeoutSeconds    int    `json:"timeoutSeconds" d:"60"`
	Remark            string `json:"remark"`
}

// CreateRes 新增结果
type CreateRes struct {
	Id uint64 `json:"id"`
}
