package v1

import "github.com/gogf/gf/v2/frame/g"

// ImportTemplateReq 下载用户导入模板(超管模板含"最大设备数"列)
type ImportTemplateReq struct {
	g.Meta `path:"/user/import/template" method:"get" tags:"User" summary:"下载用户导入模板"`
}

// ImportTemplateRes 模板文件由 handler 直接写出
type ImportTemplateRes struct{}
