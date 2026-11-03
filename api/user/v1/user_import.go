package v1

import "github.com/gogf/gf/v2/frame/g"

// ImportReq 上传 Excel 导入用户(multipart/form-data)
type ImportReq struct {
	g.Meta `path:"/user/import" method:"post" tags:"User" summary:"Excel导入用户"`
	// Atomic 全量回滚开关: 默认 false(部分成功); true 时任一行失败则全部回滚
	Atomic bool `json:"atomic" dc:"是否全量回滚"`
}

// ImportRowError 单行失败明细
type ImportRowError struct {
	Row      int    `json:"row"`
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

// ImportRes 导入结果
type ImportRes struct {
	FileName string           `json:"fileName"`
	Atomic   bool             `json:"atomic"`
	Total    int              `json:"total"`
	Success  int              `json:"success"`
	Failed   int              `json:"failed"`
	Errors   []ImportRowError `json:"errors"`
}
