package v1

import "github.com/gogf/gf/v2/frame/g"

// BatchReq 批量读取常用参数(任意已登录用户可用)
type BatchReq struct {
	g.Meta `path:"/config/batch" method:"get" tags:"SysConfig" summary:"批量获取系统参数"`
	Keys   string `json:"keys" v:"required#请指定参数键(逗号分隔)"`
}

// BatchRes 批量参数值: key 为参数键
type BatchRes struct {
	Values map[string]string `json:"values"`
}
