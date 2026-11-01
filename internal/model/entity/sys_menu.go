// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// SysMenu is the golang structure for table sys_menu.
type SysMenu struct {
	Id         uint64      `json:"id"         orm:"id"           description:"菜单ID"`                        // 菜单ID
	Pid        uint64      `json:"pid"        orm:"pid"          description:"父级ID, 0为顶级"`                  // 父级ID, 0为顶级
	Name       string      `json:"name"       orm:"name"         description:"路由名称(vue-router name, 全局唯一)"` // 路由名称(vue-router name, 全局唯一)
	Path       string      `json:"path"       orm:"path"         description:"路由路径, 顶级以/开头, 子级相对路径"`        // 路由路径, 顶级以/开头, 子级相对路径
	Component  string      `json:"component"  orm:"component"    description:"组件路径, 目录填BasicLayout"`        // 组件路径, 目录填BasicLayout
	Redirect   string      `json:"redirect"   orm:"redirect"     description:"重定向地址"`                       // 重定向地址
	Type       string      `json:"type"       orm:"type"         description:"类型: catalog=目录 menu=菜单"`      // 类型: catalog=目录 menu=菜单
	Title      string      `json:"title"      orm:"title"        description:"菜单标题"`                        // 菜单标题
	Icon       string      `json:"icon"       orm:"icon"         description:"图标"`                          // 图标
	Sort       int         `json:"sort"       orm:"sort"         description:"排序(升序)"`                      // 排序(升序)
	Status     int         `json:"status"     orm:"status"       description:"状态: 1=启用 0=停用"`               // 状态: 1=启用 0=停用
	HideInMenu int         `json:"hideInMenu" orm:"hide_in_menu" description:"是否隐藏菜单"`                      // 是否隐藏菜单
	KeepAlive  int         `json:"keepAlive"  orm:"keep_alive"   description:"是否缓存页面"`                      // 是否缓存页面
	AffixTab   int         `json:"affixTab"   orm:"affix_tab"    description:"是否固定标签页"`                     // 是否固定标签页
	IsPublic   int         `json:"isPublic"   orm:"is_public"    description:"是否公共菜单(所有登录用户可见, 无需授权)"`      // 是否公共菜单(所有登录用户可见, 无需授权)
	Remark     string      `json:"remark"     orm:"remark"       description:"备注"`                          // 备注
	CreatedBy  uint64      `json:"createdBy"  orm:"created_by"   description:"创建人ID"`                       // 创建人ID
	CreatedAt  *gtime.Time `json:"createdAt"  orm:"created_at"   description:"创建时间"`                        // 创建时间
	UpdatedAt  *gtime.Time `json:"updatedAt"  orm:"updated_at"   description:"更新时间"`                        // 更新时间
	DeletedAt  *gtime.Time `json:"deletedAt"  orm:"deleted_at"   description:"删除时间(软删除)"`                   // 删除时间(软删除)
}
