// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysRoleDao is the data access object for the table sys_role.
type SysRoleDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysRoleColumns     // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysRoleColumns defines and stores column names for the table sys_role.
type SysRoleColumns struct {
	Id        string // 角色ID
	Code      string // 角色编码(全局唯一, 如 ops)
	Name      string // 角色名称(显示用)
	Remark    string // 备注
	Status    string // 状态: 1=启用 0=停用
	CreatedBy string // 创建人ID
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删除)
}

// sysRoleColumns holds the columns for the table sys_role.
var sysRoleColumns = SysRoleColumns{
	Id:        "id",
	Code:      "code",
	Name:      "name",
	Remark:    "remark",
	Status:    "status",
	CreatedBy: "created_by",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewSysRoleDao creates and returns a new DAO object for table data access.
func NewSysRoleDao(handlers ...gdb.ModelHandler) *SysRoleDao {
	return &SysRoleDao{
		group:    "default",
		table:    "sys_role",
		columns:  sysRoleColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysRoleDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the DAO.
func (dao *SysRoleDao) Table() string {
	return dao.table
}

// Columns returns all column names of the DAO.
func (dao *SysRoleDao) Columns() SysRoleColumns {
	return dao.columns
}

// Group returns the configuration group name of database of the DAO.
func (dao *SysRoleDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the DAO. It automatically sets the context for the current operation.
func (dao *SysRoleDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysRoleDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
