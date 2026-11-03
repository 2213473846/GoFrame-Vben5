package v1

import (
	"github.com/gogf/gf/v2/frame/g"

	"demo/api/menu/v1"
)

// GrantInfoReq 查询某角色的可授权菜单与已授权菜单
// 超管: 可授权全部启用菜单; 管理员: 仅超管下发给自己的菜单
type GrantInfoReq struct {
	g.Meta `path:"/role/grant" method:"get" tags:"Role" summary:"查询角色菜单授权详情"`
	Id     uint64 `json:"id" v:"required#请指定角色ID"`
}

// GrantInfoRes 授权详情
type GrantInfoRes struct {
	RoleId     uint64         `json:"roleId"`
	Name       string         `json:"name"`
	Items      []*v1.MenuItem `json:"items"`      // 当前操作人可授权的菜单树
	CheckedIds []uint64       `json:"checkedIds"` // 目标角色已授权的菜单ID
}

// GrantReq 保存角色菜单授权(全量覆盖)
// 管理员保存时会保留超管直接下发、自己无权管理的菜单
type GrantReq struct {
	g.Meta  `path:"/role/grant" method:"put" tags:"Role" summary:"保存角色菜单授权"`
	RoleId  uint64   `json:"roleId" v:"required#请选择角色"`
	MenuIds []uint64 `json:"menuIds"`
}

// GrantRes 授权结果
type GrantRes struct{}
