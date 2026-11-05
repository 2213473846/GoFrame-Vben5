package v1

import "github.com/gogf/gf/v2/frame/g"

// ListReq 定时任务列表(超管/管理员可读)
type ListReq struct {
	g.Meta   `path:"/job/list" method:"get" tags:"Job" summary:"定时任务列表"`
	Page     int    `p:"page" d:"1"`
	PageSize int    `p:"pageSize" d:"10"`
	Name     string `p:"name"`          // 任务名称模糊查询
	Status   int    `p:"status" d:"-1"` // -1=全部 1=启用 0=停用
}

// ListRes 定时任务列表结果
type ListRes struct {
	Items []*JobItem `json:"items"`
	Total int        `json:"total"`
}
