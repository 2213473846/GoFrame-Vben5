package v1

import "github.com/gogf/gf/v2/frame/g"

// ListReq 系统参数列表(任意已登录用户可读)
type ListReq struct {
	g.Meta  `path:"/config/list" method:"get" tags:"SysConfig" summary:"系统参数列表"`
	Keyword string `json:"keyword"`
	Status  int    `json:"status" d:"-1"`
}

// ConfigItem 系统参数
type ConfigItem struct {
	Id          uint64 `json:"id"`
	ConfigKey   string `json:"configKey"`
	ConfigValue string `json:"configValue"`
	Name        string `json:"name"`
	Remark      string `json:"remark"`
	Status      int    `json:"status"`
	Builtin     int    `json:"builtin"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// ListRes 系统参数列表
type ListRes struct {
	Items []ConfigItem `json:"items"`
}
