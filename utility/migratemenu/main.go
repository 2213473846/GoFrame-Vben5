// 一次性迁移脚本：创建菜单表与用户菜单授权表
// 用法: go run ./utility/migratemenu
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var ddl = []string{
	"CREATE TABLE IF NOT EXISTS `sys_menu` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '菜单ID'," +
		"`pid` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '父级ID, 0为顶级'," +
		"`name` varchar(64) NOT NULL COMMENT '路由名称(vue-router name, 全局唯一)'," +
		"`path` varchar(128) NOT NULL COMMENT '路由路径, 顶级以/开头, 子级相对路径'," +
		"`component` varchar(255) NOT NULL DEFAULT '' COMMENT '组件路径, 目录填BasicLayout'," +
		"`redirect` varchar(255) NOT NULL DEFAULT '' COMMENT '重定向地址'," +
		"`type` varchar(16) NOT NULL DEFAULT 'menu' COMMENT '类型: catalog=目录 menu=菜单'," +
		"`title` varchar(64) NOT NULL COMMENT '菜单标题'," +
		"`icon` varchar(64) NOT NULL DEFAULT '' COMMENT '图标'," +
		"`sort` int(11) NOT NULL DEFAULT 0 COMMENT '排序(升序)'," +
		"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用'," +
		"`hide_in_menu` tinyint(4) NOT NULL DEFAULT 0 COMMENT '是否隐藏菜单'," +
		"`keep_alive` tinyint(4) NOT NULL DEFAULT 0 COMMENT '是否缓存页面'," +
		"`affix_tab` tinyint(4) NOT NULL DEFAULT 0 COMMENT '是否固定标签页'," +
		"`is_public` tinyint(4) NOT NULL DEFAULT 0 COMMENT '是否公共菜单(所有登录用户可见, 无需授权)'," +
		"`remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注'," +
		"`created_by` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID'," +
		"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
		"`updated_at` datetime DEFAULT NULL COMMENT '更新时间'," +
		"`deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `uk_name_deleted` (`name`, `deleted_at`)," +
		"KEY `idx_pid` (`pid`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统菜单/路由表'",
	"CREATE TABLE IF NOT EXISTS `sys_user_menu` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键'," +
		"`user_id` bigint(20) unsigned NOT NULL COMMENT '用户ID'," +
		"`menu_id` bigint(20) unsigned NOT NULL COMMENT '菜单ID'," +
		"`created_at` datetime DEFAULT NULL COMMENT '授权时间'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `uk_user_menu` (`user_id`, `menu_id`)," +
		"KEY `idx_menu` (`menu_id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户菜单授权表'",
}

func main() {
	ctx := gctx.New()
	for _, sql := range ddl {
		if _, err := g.DB().Exec(ctx, sql); err != nil {
			panic(fmt.Errorf("执行失败: %w", err))
		}
	}
	tables := []string{"sys_user", "sys_menu", "sys_user_menu"}
	for _, t := range tables {
		count, err := g.DB().GetCount(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'vben_admin' AND table_name = ?", t)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s => %v\n", t, count > 0)
	}
	fmt.Println("migration done")
}
