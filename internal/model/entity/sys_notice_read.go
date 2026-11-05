// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysNoticeRead is the golang structure for table sys_notice_read.
type SysNoticeRead struct {
	Id       uint64      `json:"id"       orm:"id"        description:"主键"`
	NoticeId uint64      `json:"noticeId" orm:"notice_id" description:"公告ID"`
	UserId   uint64      `json:"userId"   orm:"user_id"   description:"用户ID"`
	ReadAt   *gtime.Time `json:"readAt"   orm:"read_at"   description:"阅读时间"`
}
