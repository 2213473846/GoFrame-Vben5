package v1

import "github.com/gogf/gf/v2/frame/g"

// UpdateReq 修改菜单/路由(仅超级管理员)
type UpdateReq struct {
	g.Meta     `path:"/menu/update" method:"put" tags:"Menu" summary:"修改菜单/路由"`
	Id         uint64 `json:"id" v:"required#请指定菜单ID"`
	Pid        uint64 `json:"pid"`
	Name       string `json:"name" v:"required|length:1,64#请输入路由名称|路由名称最长64位"`
	Path       string `json:"path" v:"required|length:1,128#请输入路由路径|路由路径最长128位"`
	Component  string `json:"component"`
	Redirect   string `json:"redirect"`
	Type       string `json:"type" v:"required|in:catalog,menu#请选择类型|类型只能是catalog或menu"`
	Title      string `json:"title" v:"required|length:1,64#请输入菜单标题|菜单标题最长64位"`
	Icon       string `json:"icon"`
	Sort       int    `json:"sort"`
	Status     int    `json:"status"`
	HideInMenu bool   `json:"hideInMenu"`
	KeepAlive  bool   `json:"keepAlive"`
	AffixTab   bool   `json:"affixTab"`
	IsPublic   bool   `json:"isPublic"`
	Remark     string `json:"remark"`
}

// UpdateRes 修改结果
type UpdateRes struct{}
