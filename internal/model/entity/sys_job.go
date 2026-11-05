// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysJob is the golang structure for table sys_job.
type SysJob struct {
	Id                uint64      `json:"id"                orm:"id"                 description:"任务ID"`
	Name              string      `json:"name"              orm:"name"               description:"任务名称"`
	JobGroup          string      `json:"jobGroup"          orm:"job_group"          description:"任务分组"`
	InvokeTarget      string      `json:"invokeTarget"      orm:"invoke_target"      description:"白名单任务标识"`
	CronExpression    string      `json:"cronExpression"    orm:"cron_expression"    description:"cron 表达式(5 段或 6 段)"`
	Status            int         `json:"status"            orm:"status"             description:"状态: 1=启用 0=停用"`
	ConcurrencyPolicy int         `json:"concurrencyPolicy" orm:"concurrency_policy" description:"并发策略: 1=允许并发 0=禁止并发"`
	TimeoutSeconds    int         `json:"timeoutSeconds"    orm:"timeout_seconds"    description:"执行超时秒数, 0=不限制"`
	Remark            string      `json:"remark"            orm:"remark"             description:"备注"`
	LastRunAt         *gtime.Time `json:"lastRunAt"         orm:"last_run_at"        description:"最近执行时间"`
	LastResult        string      `json:"lastResult"        orm:"last_result"        description:"最近执行结果文案"`
	NextRunAt         *gtime.Time `json:"nextRunAt"         orm:"next_run_at"        description:"下次执行时间"`
	CreatedAt         *gtime.Time `json:"createdAt"         orm:"created_at"         description:"创建时间"`
	UpdatedAt         *gtime.Time `json:"updatedAt"         orm:"updated_at"         description:"更新时间"`
	DeletedAt         *gtime.Time `json:"deletedAt"         orm:"deleted_at"         description:"删除时间(软删除)"`
}
