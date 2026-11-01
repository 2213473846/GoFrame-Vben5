// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysOperationLogDao is the data access object for the table sys_operation_log.
type SysOperationLogDao struct {
	table    string                 // table is the underlying table name of the DAO.
	group    string                 // group is the database configuration group name of the current DAO.
	columns  SysOperationLogColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler     // handlers for customized model modification.
}

// SysOperationLogColumns defines and stores column names for the table sys_operation_log.
type SysOperationLogColumns struct {
	Id        string // 主键
	UserId    string // 用户ID
	Username  string // 用户名
	Module    string // 操作模块
	Action    string // 操作类型
	Method    string // 请求方法
	Path      string // 请求地址
	Ip        string // 操作地址(IP)
	Location  string // 操作地点
	Browser   string // 浏览器
	Os        string // 操作系统
	Status    string // 操作状态: 1=成功 0=失败
	Message   string // 操作信息
	CreatedAt string // 操作时间
}

// sysOperationLogColumns holds the columns for the table sys_operation_log.
var sysOperationLogColumns = SysOperationLogColumns{
	Id:        "id",
	UserId:    "user_id",
	Username:  "username",
	Module:    "module",
	Action:    "action",
	Method:    "method",
	Path:      "path",
	Ip:        "ip",
	Location:  "location",
	Browser:   "browser",
	Os:        "os",
	Status:    "status",
	Message:   "message",
	CreatedAt: "created_at",
}

// NewSysOperationLogDao creates and returns a new DAO object for table data access.
func NewSysOperationLogDao(handlers ...gdb.ModelHandler) *SysOperationLogDao {
	return &SysOperationLogDao{
		group:    "default",
		table:    "sys_operation_log",
		columns:  sysOperationLogColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysOperationLogDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysOperationLogDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysOperationLogDao) Columns() SysOperationLogColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysOperationLogDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysOperationLogDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *SysOperationLogDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
