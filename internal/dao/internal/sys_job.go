// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysJobDao is the data access object for the table sys_job.
type SysJobDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysJobColumns      // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysJobColumns defines and stores column names for the table sys_job.
type SysJobColumns struct {
	Id                string // 任务ID
	Name              string // 任务名称
	JobGroup          string // 任务分组
	InvokeTarget      string // 白名单任务标识
	CronExpression    string // cron 表达式(5 段或 6 段)
	Status            string // 状态: 1=启用 0=停用
	ConcurrencyPolicy string // 并发策略: 1=允许并发 0=禁止并发
	TimeoutSeconds    string // 执行超时秒数, 0=不限制
	Remark            string // 备注
	LastRunAt         string // 最近执行时间
	LastResult        string // 最近执行结果文案
	NextRunAt         string // 下次执行时间
	CreatedAt         string // 创建时间
	UpdatedAt         string // 更新时间
	DeletedAt         string // 删除时间(软删除)
}

// sysJobColumns holds the columns for the table sys_job.
var sysJobColumns = SysJobColumns{
	Id:                "id",
	Name:              "name",
	JobGroup:          "job_group",
	InvokeTarget:      "invoke_target",
	CronExpression:    "cron_expression",
	Status:            "status",
	ConcurrencyPolicy: "concurrency_policy",
	TimeoutSeconds:    "timeout_seconds",
	Remark:            "remark",
	LastRunAt:         "last_run_at",
	LastResult:        "last_result",
	NextRunAt:         "next_run_at",
	CreatedAt:         "created_at",
	UpdatedAt:         "updated_at",
	DeletedAt:         "deleted_at",
}

// NewSysJobDao creates and returns a new DAO object for table data access.
func NewSysJobDao(handlers ...gdb.ModelHandler) *SysJobDao {
	return &SysJobDao{
		group:    "default",
		table:    "sys_job",
		columns:  sysJobColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysJobDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysJobDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysJobDao) Columns() SysJobColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysJobDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysJobDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysJobDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
