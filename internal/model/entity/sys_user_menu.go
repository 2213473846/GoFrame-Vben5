// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUserMenu is the golang structure for table sys_user_menu.
type SysUserMenu struct {
	Id        uint64      `json:"id"        orm:"id"         description:"主键"`   // 主键
	UserId    uint64      `json:"userId"    orm:"user_id"    description:"用户ID"` // 用户ID
	MenuId    uint64      `json:"menuId"    orm:"menu_id"    description:"菜单ID"` // 菜单ID
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"授权时间"` // 授权时间
}
