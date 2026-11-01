// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysOperationLog is the golang structure for table sys_operation_log.
type SysOperationLog struct {
	Id        uint64      `json:"id"        orm:"id"         description:"主键"`              // 主键
	UserId    uint64      `json:"userId"    orm:"user_id"    description:"用户ID"`            // 用户ID
	Username  string      `json:"username"  orm:"username"   description:"用户名"`             // 用户名
	Module    string      `json:"module"    orm:"module"     description:"操作模块"`            // 操作模块
	Action    string      `json:"action"    orm:"action"     description:"操作类型"`            // 操作类型
	Method    string      `json:"method"    orm:"method"     description:"请求方法"`            // 请求方法
	Path      string      `json:"path"      orm:"path"       description:"请求地址"`            // 请求地址
	Ip        string      `json:"ip"        orm:"ip"         description:"操作地址(IP)"`        // 操作地址(IP)
	Location  string      `json:"location"  orm:"location"   description:"操作地点"`            // 操作地点
	Browser   string      `json:"browser"   orm:"browser"    description:"浏览器"`             // 浏览器
	Os        string      `json:"os"        orm:"os"         description:"操作系统"`            // 操作系统
	Status    int         `json:"status"    orm:"status"     description:"操作状态: 1=成功 0=失败"` // 操作状态: 1=成功 0=失败
	Message   string      `json:"message"   orm:"message"    description:"操作信息"`            // 操作信息
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"操作时间"`            // 操作时间
}
