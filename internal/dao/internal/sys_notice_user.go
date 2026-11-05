// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysNoticeUserDao is the data access object for the table sys_notice_user.
type SysNoticeUserDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  SysNoticeUserColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// SysNoticeUserColumns defines and stores column names for the table sys_notice_user.
type SysNoticeUserColumns struct {
	Id        string // 主键
	NoticeId  string // 公告ID
	UserId    string // 接收用户ID
	CreatedAt string // 创建时间
}

// sysNoticeUserColumns holds the columns for the table sys_notice_user.
var sysNoticeUserColumns = SysNoticeUserColumns{
	Id:        "id",
	NoticeId:  "notice_id",
	UserId:    "user_id",
	CreatedAt: "created_at",
}

// NewSysNoticeUserDao creates and returns a new DAO object for table data access.
func NewSysNoticeUserDao(handlers ...gdb.ModelHandler) *SysNoticeUserDao {
	return &SysNoticeUserDao{
		group:    "default",
		table:    "sys_notice_user",
		columns:  sysNoticeUserColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysNoticeUserDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the DAO.
func (dao *SysNoticeUserDao) Table() string {
	return dao.table
}

// Columns returns all column names of the DAO.
func (dao *SysNoticeUserDao) Columns() SysNoticeUserColumns {
	return dao.columns
}

// Group returns the configuration group name of database of the DAO.
func (dao *SysNoticeUserDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the DAO. It automatically sets the context for the current operation.
func (dao *SysNoticeUserDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rollbacks the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
func (dao *SysNoticeUserDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
