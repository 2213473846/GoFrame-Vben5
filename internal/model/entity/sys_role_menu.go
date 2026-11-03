// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysRoleMenu is the golang structure for table sys_role_menu.
type SysRoleMenu struct {
	Id        uint64      `json:"id"        orm:"id"         description:"主键"`   // 主键
	RoleId    uint64      `json:"roleId"    orm:"role_id"    description:"角色ID"` // 角色ID
	MenuId    uint64      `json:"menuId"    orm:"menu_id"    description:"菜单ID"` // 菜单ID
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"授权时间"` // 授权时间
}
