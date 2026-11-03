package v1

import "github.com/gogf/gf/v2/frame/g"

// LoginLogExportReq 导出登录日志, 条件与列表一致
type LoginLogExportReq struct {
	g.Meta   `path:"/loginlog/export" method:"get" tags:"LoginLog" summary:"导出登录日志"`
	Username string `json:"username"`
	Status   int    `json:"status" d:"-1"`
	Format   string `json:"format" d:"xlsx" v:"in:xlsx,csv#导出格式仅支持 xlsx/csv"`
}

// LoginLogExportRes 导出结果
type LoginLogExportRes struct{}
