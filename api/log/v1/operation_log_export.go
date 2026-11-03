package v1

import "github.com/gogf/gf/v2/frame/g"

// OperationLogExportReq 导出操作日志, 条件与列表一致
type OperationLogExportReq struct {
	g.Meta   `path:"/operationlog/export" method:"get" tags:"OperationLog" summary:"导出操作日志"`
	Username string `json:"username"`
	Status   int    `json:"status" d:"-1"`
	Format   string `json:"format" d:"xlsx" v:"in:xlsx,csv#导出格式仅支持 xlsx/csv"`
}

// OperationLogExportRes 导出结果
type OperationLogExportRes struct{}
