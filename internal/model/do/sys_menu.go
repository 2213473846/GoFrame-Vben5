// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SysMenu is the golang structure of table sys_menu for DAO operations like Where/Data.
type SysMenu struct {
	g.Meta     `orm:"table:sys_menu, do:true"`
	Id         any         // 菜单ID
	Pid        any         // 父级ID, 0为顶级
	Name       any         // 路由名称(vue-router name, 全局唯一)
	Path       any         // 路由路径, 顶级以/开头, 子级相对路径
	Component  any         // 组件路径, 目录填BasicLayout
	Redirect   any         // 重定向地址
	Type       any         // 类型: catalog=目录 menu=菜单
	Title      any         // 菜单标题
	Icon       any         // 图标
	Sort       any         // 排序(升序)
	Status     any         // 状态: 1=启用 0=停用
	HideInMenu any         // 是否隐藏菜单
	KeepAlive  any         // 是否缓存页面
	AffixTab   any         // 是否固定标签页
	IsPublic   any         // 是否公共菜单(所有登录用户可见, 无需授权)
	Remark     any         // 备注
	CreatedBy  any         // 创建人ID
	CreatedAt  *gtime.Time // 创建时间
	UpdatedAt  *gtime.Time // 更新时间
	DeletedAt  *gtime.Time // 删除时间(软删除)
}
