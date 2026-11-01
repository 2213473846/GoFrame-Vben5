package v1

import "github.com/gogf/gf/v2/frame/g"

// OperationLogListReq 操作日志列表
type OperationLogListReq struct {
	g.Meta   `path:"/operationlog/list" method:"get" tags:"OperationLog" summary:"操作日志列表"`
	Page     int    `p:"page" d:"1"`
	PageSize int    `p:"pageSize" d:"10"`
	Username string `p:"username"` // 用户名模糊查询
	Status   int    `p:"status" d:"-1"` // -1=全部 1=成功 0=失败
}

// OperationLogItem 操作日志条目
type OperationLogItem struct {
	Id        int64  `json:"id"`
	Username  string `json:"username"`
	Module    string `json:"module"`
	Action    string `json:"action"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Ip        string `json:"ip"`
	Location  string `json:"location"`
	Browser   string `json:"browser"`
	Os        string `json:"os"`
	Status    int    `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

// OperationLogListRes 操作日志列表结果
type OperationLogListRes struct {
	Items []*OperationLogItem `json:"items"`
	Total int                 `json:"total"`
}
