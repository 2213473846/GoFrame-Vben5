package v1

import "github.com/gogf/gf/v2/frame/g"

// UpdateReq 修改定时任务(仅超级管理员)
type UpdateReq struct {
	g.Meta            `path:"/job/update" method:"put" tags:"Job" summary:"修改定时任务"`
	Id                uint64 `json:"id" v:"required#请指定任务"`
	Name              string `json:"name" v:"required#请填写任务名称"`
	JobGroup          string `json:"jobGroup" d:"default"`
	InvokeTarget      string `json:"invokeTarget" v:"required#请填写任务标识"`
	CronExpression    string `json:"cronExpression" v:"required#请填写 cron 表达式"`
	Status            int    `json:"status" d:"1"`
	ConcurrencyPolicy int    `json:"concurrencyPolicy" d:"0"`
	TimeoutSeconds    int    `json:"timeoutSeconds" d:"60"`
	Remark            string `json:"remark"`
}

// UpdateRes 修改结果
type UpdateRes struct{}
