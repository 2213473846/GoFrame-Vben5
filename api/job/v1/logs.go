package v1

import "github.com/gogf/gf/v2/frame/g"

// LogsReq 定时任务执行日志(超管/管理员可读)
type LogsReq struct {
	g.Meta   `path:"/job/logs" method:"get" tags:"Job" summary:"定时任务执行日志"`
	Page     int    `p:"page" d:"1"`
	PageSize int    `p:"pageSize" d:"10"`
	JobId    uint64 `p:"jobId"`
	Status   int    `p:"status" d:"-1"` // -1=全部 1=成功 0=失败
}

// JobLogItem 执行日志条目
type JobLogItem struct {
	Id          uint64 `json:"id"`
	JobId       uint64 `json:"jobId"`
	JobName     string `json:"jobName"`
	TriggerType int    `json:"triggerType"`
	StartedAt   string `json:"startedAt"`
	FinishedAt  string `json:"finishedAt"`
	DurationMs  int64  `json:"durationMs"`
	Status      int    `json:"status"`
	Message     string `json:"message"`
	CreatedAt   string `json:"createdAt"`
}

// LogsRes 执行日志结果
type LogsRes struct {
	Items []*JobLogItem `json:"items"`
	Total int           `json:"total"`
}
