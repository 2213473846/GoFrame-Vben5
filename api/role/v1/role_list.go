package v1

import "github.com/gogf/gf/v2/frame/g"

// ListReq 角色列表
// 超管返回全部角色; 管理员仅返回自己可分配的角色(只读)
type ListReq struct {
	g.Meta `path:"/role/list" method:"get" tags:"Role" summary:"功能角色列表"`
}

// RoleItem 角色列表项
type RoleItem struct {
	Id        uint64 `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Remark    string `json:"remark"`
	Status    int    `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// ListRes 角色列表
type ListRes struct {
	Items []RoleItem `json:"items"`
}
