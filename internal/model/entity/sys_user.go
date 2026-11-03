// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysUser is the golang structure for table sys_user.
type SysUser struct {
	Id                 uint64      `json:"id"        orm:"id"         description:"用户ID"`                                  // 用户ID
	Pid                uint64      `json:"pid"       orm:"pid"        description:"父级管理员ID, 0表示顶级"`                        // 父级管理员ID, 0表示顶级
	Username           string      `json:"username"  orm:"username"   description:"登录用户名"`                                 // 登录用户名
	Password           string      `json:"password"  orm:"password"   description:"密码(bcrypt)"`                            // 密码(bcrypt)
	RealName           string      `json:"realName"  orm:"real_name"  description:"昵称"`                                    // 昵称
	Avatar             string      `json:"avatar"    orm:"avatar"     description:"头像地址"`                                  // 头像地址
	Role               string      `json:"role"      orm:"role"       description:"角色: super_admin/admin/user"`            // 角色: super_admin/admin/user
	Status             int         `json:"status"    orm:"status"     description:"状态: 1=启用 0=禁用"`                         // 状态: 1=启用 0=禁用
	Remark             string      `json:"remark"    orm:"remark"     description:"备注"`                                    // 备注
	MaxDevices         int         `json:"maxDevices" orm:"max_devices" description:"最大同时在线设备数, 0表示使用全局默认"`                // 最大同时在线设备数, 0表示使用全局默认
	MustChangePassword int         `json:"mustChangePassword" orm:"must_change_password" description:"是否需强制修改密码: 1=是 0=否"` // 首次登录/弱密码强制改密标记
	CreatedAt          *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`                                  // 创建时间
	UpdatedAt          *gtime.Time `json:"updatedAt" orm:"updated_at" description:"更新时间"`                                  // 更新时间
	DeletedAt          *gtime.Time `json:"deletedAt" orm:"deleted_at" description:"删除时间(软删除)"`                             // 删除时间(软删除)
	DeletedBy          uint64      `json:"deletedBy" orm:"deleted_by" description:"删除操作人ID"`                               // 删除操作人ID
}
