// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysNoticeDao is the data access object for the table sys_notice.
type SysNoticeDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysNoticeColumns   // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysNoticeColumns defines and stores column names for the table sys_notice.
type SysNoticeColumns struct {
	Id          string // 公告ID
	Title       string // 标题
	Content     string // 正文(纯文本)
	NoticeType  string // 类型(字典 notice_type)
	NoticeLevel string // 级别(字典 notice_level)
	Status      string // 状态: 0=草稿 1=已发布 2=已撤回
	TargetScope string // 可见范围: 1=全员 2=指定用户
	PublisherId string // 发布人ID
	PublishAt   string // 发布时间
	ExpireAt    string // 失效时间
	CreatedAt   string // 创建时间
	UpdatedAt   string // 更新时间
	DeletedAt   string // 删除时间(软删除)
}

// sysNoticeColumns holds the columns for the table sys_notice.
var sysNoticeColumns = SysNoticeColumns{
	Id:          "id",
	Title:       "title",
	Content:     "content",
	NoticeType:  "notice_type",
	NoticeLevel: "notice_level",
	Status:      "status",
	TargetScope: "target_scope",
	PublisherId: "publisher_id",
	PublishAt:   "publish_at",
	ExpireAt:    "expire_at",
	CreatedAt:   "created_at",
	UpdatedAt:   "updated_at",
	DeletedAt:   "deleted_at",
}

// NewSysNoticeDao creates and returns a new DAO object for table data access.
func NewSysNoticeDao(handlers ...gdb.ModelHandler) *SysNoticeDao {
	return &SysNoticeDao{
		group:    "default",
		table:    "sys_notice",
		columns:  sysNoticeColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysNoticeDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the DAO.
func (dao *SysNoticeDao) Table() string {
	return dao.table
}

// Columns returns all column names of the DAO.
func (dao *SysNoticeDao) Columns() SysNoticeColumns {
	return dao.columns
}

// Group returns the configuration group name of database of the DAO.
func (dao *SysNoticeDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the DAO. It automatically sets the context for the current operation.
func (dao *SysNoticeDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rollbacks the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
func (dao *SysNoticeDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
