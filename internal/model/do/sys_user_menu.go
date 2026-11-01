// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUserMenu is the golang structure of table sys_user_menu for DAO operations like Where/Data.
type SysUserMenu struct {
	g.Meta    `orm:"table:sys_user_menu, do:true"`
	Id        any         // 主键
	UserId    any         // 用户ID
	MenuId    any         // 菜单ID
	CreatedAt *gtime.Time // 授权时间
}
