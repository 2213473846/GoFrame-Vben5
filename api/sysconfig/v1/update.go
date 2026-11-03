package v1

import "github.com/gogf/gf/v2/frame/g"

// UpdateReq 修改参数值(仅超级管理员)
type UpdateReq struct {
	g.Meta `path:"/config/update" method:"put" tags:"SysConfig" summary:"修改系统参数"`
	Key    string `json:"key" v:"required#请指定参数键"`
	Value  string `json:"value"`
}

// UpdateRes 修改结果
type UpdateRes struct{}
