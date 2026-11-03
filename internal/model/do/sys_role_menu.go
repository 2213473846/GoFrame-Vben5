// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysRoleMenu is the golang structure of table sys_role_menu for DAO operations like Where/Data.
type SysRoleMenu struct {
	g.Meta    `orm:"table:sys_role_menu, do:true"`
	Id        any         // 主键
	RoleId    any         // 角色ID
	MenuId    any         // 菜单ID
	CreatedAt *gtime.Time // 授权时间
}
