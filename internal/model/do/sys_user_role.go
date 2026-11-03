// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUserRole is the golang structure of table sys_user_role for DAO operations like Where/Data.
type SysUserRole struct {
	g.Meta    `orm:"table:sys_user_role, do:true"`
	Id        any         // 主键
	UserId    any         // 用户ID
	RoleId    any         // 角色ID
	CreatedAt *gtime.Time // 分配时间
}
