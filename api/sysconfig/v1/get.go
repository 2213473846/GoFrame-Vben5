package v1

import "github.com/gogf/gf/v2/frame/g"

// GetReq 按 key 读取参数(任意已登录用户可用)
type GetReq struct {
	g.Meta `path:"/config/get" method:"get" tags:"SysConfig" summary:"按key获取系统参数"`
	Key    string `json:"key" v:"required#请指定参数键"`
}

// GetRes 参数值
type GetRes struct {
	ConfigValue string `json:"configValue"`
}
