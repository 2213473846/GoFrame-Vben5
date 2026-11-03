// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysDictDataDao is the data access object for the table sys_dict_data.
type SysDictDataDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysDictDataColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysDictDataColumns defines and stores column names for the table sys_dict_data.
type SysDictDataColumns struct {
	Id        string // 字典项ID
	DictType  string // 所属字典类型编码
	Label     string // 显示名称
	Value     string // 字典值
	Sort      string // 排序(升序)
	Status    string // 状态: 1=启用 0=停用
	TagColor  string // 前端标签颜色: success/warning/danger/info
	Remark    string // 备注
	CreatedAt string // 创建时间
	UpdatedAt string // 更新时间
	DeletedAt string // 删除时间(软删除)
}

// sysDictDataColumns holds the columns for the table sys_dict_data.
var sysDictDataColumns = SysDictDataColumns{
	Id:        "id",
	DictType:  "dict_type",
	Label:     "label",
	Value:     "value",
	Sort:      "sort",
	Status:    "status",
	TagColor:  "tag_color",
	Remark:    "remark",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	DeletedAt: "deleted_at",
}

// NewSysDictDataDao creates and returns a new DAO object for table data access.
func NewSysDictDataDao(handlers ...gdb.ModelHandler) *SysDictDataDao {
	return &SysDictDataDao{
		group:    "default",
		table:    "sys_dict_data",
		columns:  sysDictDataColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysDictDataDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the DAO.
func (dao *SysDictDataDao) Table() string {
	return dao.table
}

// Columns returns all column names of the DAO.
func (dao *SysDictDataDao) Columns() SysDictDataColumns {
	return dao.columns
}

// Group returns the configuration group name of database of the DAO.
func (dao *SysDictDataDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the DAO. It automatically sets the context for the current operation.
func (dao *SysDictDataDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rollbacks the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
func (dao *SysDictDataDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
