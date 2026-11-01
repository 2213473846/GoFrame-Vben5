// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysOperationLog is the golang structure of table sys_operation_log for DAO operations like Where/Data.
type SysOperationLog struct {
	g.Meta    `orm:"table:sys_operation_log, do:true"`
	Id        any         // 主键
	UserId    any         // 用户ID
	Username  any         // 用户名
	Module    any         // 操作模块
	Action    any         // 操作类型
	Method    any         // 请求方法
	Path      any         // 请求地址
	Ip        any         // 操作地址(IP)
	Location  any         // 操作地点
	Browser   any         // 浏览器
	Os        any         // 操作系统
	Status    any         // 操作状态: 1=成功 0=失败
	Message   any         // 操作信息
	CreatedAt *gtime.Time // 操作时间
}
