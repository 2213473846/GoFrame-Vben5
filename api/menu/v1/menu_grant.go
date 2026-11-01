package v1

import "github.com/gogf/gf/v2/frame/g"

// GrantInfoReq 查询某用户的可授权菜单与已授权菜单
// 超管: 可查任意用户, 可授权全部启用菜单
// 管理员: 仅可查自己的下级普通用户, 只能下发超管已下发给自己的菜单
type GrantInfoReq struct {
	g.Meta `path:"/menu/grant" method:"get" tags:"Menu" summary:"查询用户菜单授权详情"`
	Id     uint64 `json:"id" v:"required#请指定用户ID"`
}

// GrantInfoRes 授权详情
type GrantInfoRes struct {
	UserId     uint64      `json:"userId"`
	Username   string      `json:"username"`
	Items      []*MenuItem `json:"items"`      // 当前操作人可授权的菜单树
	CheckedIds []uint64    `json:"checkedIds"` // 目标用户已授权的菜单ID
}

// GrantReq 保存用户菜单授权(全量覆盖)
type GrantReq struct {
	g.Meta  `path:"/menu/grant" method:"put" tags:"Menu" summary:"保存用户菜单授权"`
	UserId  uint64   `json:"userId" v:"required#请选择用户"`
	MenuIds []uint64 `json:"menuIds"`
}

// GrantRes 授权结果
type GrantRes struct{}
