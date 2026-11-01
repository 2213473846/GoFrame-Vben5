// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUser is the golang structure of table sys_user for DAO operations like Where/Data.
type SysUser struct {
	g.Meta    `orm:"table:sys_user, do:true"`
	Id        any         // 用户ID
	Pid       any         // 父级管理员ID, 0表示顶级
	Username  any         // 登录用户名
	Password  any         // 密码(bcrypt)
	RealName  any         // 昵称
	Avatar    any         // 头像地址
	Role      any         // 角色: super_admin/admin/user
	Status    any         // 状态: 1=启用 0=禁用
	Remark    any         // 备注
	CreatedAt *gtime.Time // 创建时间
	UpdatedAt *gtime.Time // 更新时间
	DeletedAt *gtime.Time // 删除时间(软删除)
	DeletedBy any         // 删除操作人ID
}
