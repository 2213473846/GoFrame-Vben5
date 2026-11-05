// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysNoticeUser is the golang structure for table sys_notice_user.
type SysNoticeUser struct {
	Id        uint64      `json:"id"        orm:"id"         description:"主键"`
	NoticeId  uint64      `json:"noticeId"  orm:"notice_id"  description:"公告ID"`
	UserId    uint64      `json:"userId"    orm:"user_id"    description:"接收用户ID"`
	CreatedAt *gtime.Time `json:"createdAt" orm:"created_at" description:"创建时间"`
}
