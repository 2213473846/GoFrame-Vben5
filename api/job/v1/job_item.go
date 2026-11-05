package v1

// JobItem 定时任务条目
type JobItem struct {
	Id                uint64 `json:"id"`
	Name              string `json:"name"`
	JobGroup          string `json:"jobGroup"`
	InvokeTarget      string `json:"invokeTarget"`
	CronExpression    string `json:"cronExpression"`
	Status            int    `json:"status"`
	ConcurrencyPolicy int    `json:"concurrencyPolicy"`
	TimeoutSeconds    int    `json:"timeoutSeconds"`
	Remark            string `json:"remark"`
	LastRunAt         string `json:"lastRunAt"`
	LastResult        string `json:"lastResult"`
	NextRunAt         string `json:"nextRunAt"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}
