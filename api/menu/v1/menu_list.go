package v1

import "github.com/gogf/gf/v2/frame/g"

// ListReq 菜单管理列表(树形)
// 超级管理员返回全部菜单; 管理员仅返回自己可下发的菜单(只读)
type ListReq struct {
	g.Meta `path:"/menu/list" method:"get" tags:"Menu" summary:"菜单/路由管理列表"`
}

// MenuItem 菜单树节点
type MenuItem struct {
	Id         uint64      `json:"id"`
	Pid        uint64      `json:"pid"`
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Component  string      `json:"component"`
	Redirect   string      `json:"redirect"`
	Type       string      `json:"type"`
	Title      string      `json:"title"`
	Icon       string      `json:"icon"`
	Sort       int         `json:"sort"`
	Status     int         `json:"status"`
	HideInMenu bool        `json:"hideInMenu"`
	KeepAlive  bool        `json:"keepAlive"`
	AffixTab   bool        `json:"affixTab"`
	IsPublic   bool        `json:"isPublic"`
	Remark     string      `json:"remark"`
	CreatedAt  string      `json:"createdAt"`
	Children   []*MenuItem `json:"children,omitempty"`
}

// ListRes 菜单树
type ListRes struct {
	Items []*MenuItem `json:"items"`
}
