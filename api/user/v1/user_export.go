package v1

import "github.com/gogf/gf/v2/frame/g"

// ExportReq 导出用户列表
// 筛选条件与 /user/list 完全一致; 数据权限同样复用列表逻辑
type ExportReq struct {
	g.Meta   `path:"/user/export" method:"get" tags:"User" summary:"导出用户列表"`
	Username string `json:"username"`
	RealName string `json:"realName"`
	Role     string `json:"role"`
	Status   *int   `json:"status"`
	Format   string `json:"format" d:"xlsx" v:"in:xlsx,csv#导出格式仅支持 xlsx/csv"`
}

// ExportRes 导出结果(文件流由 handler 直接写出)
type ExportRes struct{}
