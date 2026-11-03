package v1

import "github.com/gogf/gf/v2/frame/g"

// ResetReq 重置内置参数为默认值(仅超级管理员)
type ResetReq struct {
	g.Meta `path:"/config/reset" method:"put" tags:"SysConfig" summary:"重置系统参数为默认值"`
	Key    string `json:"key" v:"required#请指定参数键"`
}

// ResetRes 重置后的参数值
type ResetRes struct {
	ConfigValue string `json:"configValue"`
}
