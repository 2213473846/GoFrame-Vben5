// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// SysMenuDao is the data access object for the table sys_menu.
type SysMenuDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  SysMenuColumns     // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// SysMenuColumns defines and stores column names for the table sys_menu.
type SysMenuColumns struct {
	Id         string // 菜单ID
	Pid        string // 父级ID, 0为顶级
	Name       string // 路由名称(vue-router name, 全局唯一)
	Path       string // 路由路径, 顶级以/开头, 子级相对路径
	Component  string // 组件路径, 目录填BasicLayout
	Redirect   string // 重定向地址
	Type       string // 类型: catalog=目录 menu=菜单
	Title      string // 菜单标题
	Icon       string // 图标
	Sort       string // 排序(升序)
	Status     string // 状态: 1=启用 0=停用
	HideInMenu string // 是否隐藏菜单
	KeepAlive  string // 是否缓存页面
	AffixTab   string // 是否固定标签页
	IsPublic   string // 是否公共菜单(所有登录用户可见, 无需授权)
	Remark     string // 备注
	CreatedBy  string // 创建人ID
	CreatedAt  string // 创建时间
	UpdatedAt  string // 更新时间
	DeletedAt  string // 删除时间(软删除)
}

// sysMenuColumns holds the columns for the table sys_menu.
var sysMenuColumns = SysMenuColumns{
	Id:         "id",
	Pid:        "pid",
	Name:       "name",
	Path:       "path",
	Component:  "component",
	Redirect:   "redirect",
	Type:       "type",
	Title:      "title",
	Icon:       "icon",
	Sort:       "sort",
	Status:     "status",
	HideInMenu: "hide_in_menu",
	KeepAlive:  "keep_alive",
	AffixTab:   "affix_tab",
	IsPublic:   "is_public",
	Remark:     "remark",
	CreatedBy:  "created_by",
	CreatedAt:  "created_at",
	UpdatedAt:  "updated_at",
	DeletedAt:  "deleted_at",
}

// NewSysMenuDao creates and returns a new DAO object for table data access.
func NewSysMenuDao(handlers ...gdb.ModelHandler) *SysMenuDao {
	return &SysMenuDao{
		group:    "default",
		table:    "sys_menu",
		columns:  sysMenuColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *SysMenuDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *SysMenuDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *SysMenuDao) Columns() SysMenuColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *SysMenuDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *SysMenuDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *SysMenuDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
