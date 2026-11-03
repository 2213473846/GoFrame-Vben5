package v1

import "github.com/gogf/gf/v2/frame/g"

// AllReq 获取当前登录用户可见的路由(由后端下发)
type AllReq struct {
	g.Meta `path:"/menu/all" method:"get" tags:"Menu" summary:"当前用户可见路由"`
}

// RouteMeta 路由元信息(与 vue-vben-admin 后端路由模式对齐)
type RouteMeta struct {
	Title      string `json:"title"`
	Icon       string `json:"icon,omitempty"`
	Order      int    `json:"order,omitempty"`
	HideInMenu bool   `json:"hideInMenu,omitempty"`
	KeepAlive  bool   `json:"keepAlive,omitempty"`
	AffixTab   bool   `json:"affixTab,omitempty"`
}

// RouteNode 路由节点
type RouteNode struct {
	Name      string       `json:"name"`
	Path      string       `json:"path"`
	Component string       `json:"component,omitempty"`
	Redirect  string       `json:"redirect,omitempty"`
	Meta      *RouteMeta   `json:"meta"`
	Children  []*RouteNode `json:"children,omitempty"`
}

// AllRes 路由树(数组直接作为 data 返回)
type AllRes []*RouteNode
