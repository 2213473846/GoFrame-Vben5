package v1

import "github.com/gogf/gf/v2/frame/g"

// CreateReq 新增菜单/路由(仅超级管理员)
type CreateReq struct {
	g.Meta     `path:"/menu/create" method:"post" tags:"Menu" summary:"新增菜单/路由"`
	Pid        uint64 `json:"pid"`   // 父级ID, 0为顶级
	Name       string `json:"name" v:"required|length:1,64#请输入路由名称|路由名称最长64位"`
	Path       string `json:"path" v:"required|length:1,128#请输入路由路径|路由路径最长128位"`
	Component  string `json:"component"`
	Redirect   string `json:"redirect"`
	Type       string `json:"type" v:"required|in:catalog,menu#请选择类型|类型只能是catalog或menu"`
	Title      string `json:"title" v:"required|length:1,64#请输入菜单标题|菜单标题最长64位"`
	Icon       string `json:"icon"`
	Sort       int    `json:"sort"`
	Status     int    `json:"status" d:"1"`
	HideInMenu bool   `json:"hideInMenu"`
	KeepAlive  bool   `json:"keepAlive"`
	AffixTab   bool   `json:"affixTab"`
	IsPublic   bool   `json:"isPublic"`
	Remark     string `json:"remark"`
}

// CreateRes 新增结果
type CreateRes struct {
	Id uint64 `json:"id"`
}
