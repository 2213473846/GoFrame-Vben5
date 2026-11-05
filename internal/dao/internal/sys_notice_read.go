// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysNoticeReadDao is the data access object for the table sys_notice_read.
type SysNoticeReadDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  SysNoticeReadColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// SysNoticeReadColumns defines and stores column names for the table sys_notice_read.
type SysNoticeReadColumns struct {
	Id       string // 主键
	NoticeId string // 公告ID
	UserId   string // 用户ID
	ReadAt   string // 阅读时间
}

// sysNoticeReadColumns holds the columns for the table sys_notice_read.
var sysNoticeReadColumns = SysNoticeReadColumns{
	Id:       "id",
	NoticeId: "notice_id",
	UserId:   "user_id",
	ReadAt:   "read_at",
}

// NewSysNoticeReadDao creates and returns a new DAO object for table data access.
func NewSysNoticeReadDao(handlers ...gdb.ModelHandler) *SysNoticeReadDao {
	return &SysNoticeReadDao{
		group:    "default",
		table:    "sys_notice_read",
		columns:  sysNoticeReadColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysNoticeReadDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the DAO.
func (dao *SysNoticeReadDao) Table() string {
	return dao.table
}

// Columns returns all column names of the DAO.
func (dao *SysNoticeReadDao) Columns() SysNoticeReadColumns {
	return dao.columns
}

// Group returns the configuration group name of database of the DAO.
func (dao *SysNoticeReadDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the DAO. It automatically sets the context for the current operation.
func (dao *SysNoticeReadDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rollbacks the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
func (dao *SysNoticeReadDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
