// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysJobLog is the golang structure for table sys_job_log.
type SysJobLog struct {
	Id          uint64      `json:"id"           orm:"id"            description:"主键"`
	JobId       uint64      `json:"jobId"        orm:"job_id"        description:"任务ID"`
	JobName     string      `json:"jobName"      orm:"job_name"      description:"任务名称"`
	TriggerType int         `json:"triggerType"  orm:"trigger_type"  description:"触发方式: 1=定时 2=手动"`
	StartedAt   *gtime.Time `json:"startedAt"    orm:"started_at"    description:"开始时间"`
	FinishedAt  *gtime.Time `json:"finishedAt"   orm:"finished_at"   description:"结束时间"`
	DurationMs  int64       `json:"durationMs"   orm:"duration_ms"   description:"执行耗时(毫秒)"`
	Status      int         `json:"status"       orm:"status"        description:"执行状态: 1=成功 0=失败"`
	Message     string      `json:"message" orm:"message" description:"执行结果摘要/失败原因"`
	CreatedAt   *gtime.Time `json:"createdAt"    orm:"created_at"    description:"创建时间"`
}
