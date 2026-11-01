// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysLoginLog is the golang structure for table sys_login_log.
type SysLoginLog struct {
	Id        uint64      `json:"id"        orm:"id"         description:"主键"`                  // 主键
	UserId    uint64      `json:"userId"    orm:"user_id"    description:"用户ID, 登录失败且用户不存在时为0"` // 用户ID, 登录失败且用户不存在时为0
	Username  string      `json:"username"  orm:"username"   description:"登录用户名"`               // 登录用户名
	Ip        string      `json:"ip"        orm:"ip"         description:"登录地址(IP)"`            // 登录地址(IP)
	Location  string      `json:"location"  orm:"location"   description:"登录地点"`                // 登录地点
	Browser   string      `json:"browser"   orm:"browser"    description:"浏览器"`                 // 浏览器
	Os        string      `json:"os"        orm:"os"         description:"操作系统"`                // 操作系统
	Status    int         `json:"status"    orm:"status"     description:"登录状态: 1=成功 0=失败"`     // 登录状态: 1=成功 0=失败
	Message   string      `json:"message"   orm:"message"    description:"登录信息"`                // 登录信息
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"登录时间"`                // 登录时间
}
