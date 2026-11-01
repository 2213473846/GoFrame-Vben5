// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysLoginLog is the golang structure of table sys_login_log for DAO operations like Where/Data.
type SysLoginLog struct {
	g.Meta    `orm:"table:sys_login_log, do:true"`
	Id        any         // 主键
	UserId    any         // 用户ID, 登录失败且用户不存在时为0
	Username  any         // 登录用户名
	Ip        any         // 登录地址(IP)
	Location  any         // 登录地点
	Browser   any         // 浏览器
	Os        any         // 操作系统
	Status    any         // 登录状态: 1=成功 0=失败
	Message   any         // 登录信息
	CreatedAt *gtime.Time // 登录时间
}
