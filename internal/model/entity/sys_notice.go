// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysNotice is the golang structure for table sys_notice.
type SysNotice struct {
	Id          uint64      `json:"id"          orm:"id"           description:"公告ID"`
	Title       string      `json:"title"       orm:"title"        description:"标题"`
	Content     string      `json:"content"     orm:"content"      description:"正文(纯文本)"`
	NoticeType  string      `json:"noticeType"  orm:"notice_type"  description:"类型(字典 notice_type)"`
	NoticeLevel string      `json:"noticeLevel" orm:"notice_level" description:"级别(字典 notice_level)"`
	Status      int         `json:"status"      orm:"status"       description:"状态: 0=草稿 1=已发布 2=已撤回"`
	TargetScope int         `json:"targetScope" orm:"target_scope" description:"可见范围: 1=全员 2=指定用户"`
	PublisherId uint64      `json:"publisherId" orm:"publisher_id" description:"发布人ID"`
	PublishAt   *gtime.Time `json:"publishAt"   orm:"publish_at"   description:"发布时间"`
	ExpireAt    *gtime.Time `json:"expireAt"    orm:"expire_at"    description:"失效时间"`
	CreatedAt   *gtime.Time `json:"createdAt"   orm:"created_at"   description:"创建时间"`
	UpdatedAt   *gtime.Time `json:"updatedAt"   orm:"updated_at"   description:"更新时间"`
	DeletedAt   *gtime.Time `json:"deletedAt"   orm:"deleted_at"   description:"删除时间(软删除)"`
}
