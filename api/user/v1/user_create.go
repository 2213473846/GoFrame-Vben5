package v1

import "github.com/gogf/gf/v2/frame/g"

// CreateReq 创建管理员或用户
// 超管: 可创建管理员(pid=0)和用户; 管理员: 可创建下级管理员/用户(pid=自己)
type CreateReq struct {
	g.Meta   `path:"/user/create" method:"post" tags:"User" summary:"创建管理员/用户"`
	Username string   `json:"username" v:"required|length:3,32#请输入用户名|用户名长度为3-32位"`
	Password string   `json:"password" v:"required#请输入密码"` // 复杂度由 service 层统一校验
	RealName string   `json:"realName"`
	Role     string   `json:"role" v:"required#请选择角色"`
	Remark   string   `json:"remark"`
	RoleIds  []uint64 `json:"roleIds"` // 功能角色(RBAC), 可为空
	// MaxDevices 最大同时在线设备数, 仅超管可设置; 不传/0 使用全局默认
	MaxDevices *int `json:"maxDevices" v:"min:0#最大在线设备数不能为负数"`
}

// CreateRes 创建结果
type CreateRes struct {
	Id uint64 `json:"id"`
}
