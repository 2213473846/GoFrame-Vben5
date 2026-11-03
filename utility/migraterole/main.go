// 一次性迁移脚本：创建角色表、用户角色关联表与角色菜单授权表
// 用法: go run ./utility/migraterole
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var ddl = []string{
	"CREATE TABLE IF NOT EXISTS `sys_role` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '角色ID'," +
		"`code` varchar(64) NOT NULL COMMENT '角色编码(全局唯一, 如 ops)'," +
		"`name` varchar(64) NOT NULL COMMENT '角色名称(显示用)'," +
		"`remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注'," +
		"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用'," +
		"`created_by` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID'," +
		"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
		"`updated_at` datetime DEFAULT NULL COMMENT '更新时间'," +
		"`deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `uk_code_deleted` (`code`, `deleted_at`)," +
		"KEY `idx_name` (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='功能角色表'",
	"CREATE TABLE IF NOT EXISTS `sys_user_role` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键'," +
		"`user_id` bigint(20) unsigned NOT NULL COMMENT '用户ID'," +
		"`role_id` bigint(20) unsigned NOT NULL COMMENT '角色ID'," +
		"`created_at` datetime DEFAULT NULL COMMENT '分配时间'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `uk_user_role` (`user_id`, `role_id`)," +
		"KEY `idx_role` (`role_id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色关联表'",
	"CREATE TABLE IF NOT EXISTS `sys_role_menu` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键'," +
		"`role_id` bigint(20) unsigned NOT NULL COMMENT '角色ID'," +
		"`menu_id` bigint(20) unsigned NOT NULL COMMENT '菜单ID'," +
		"`created_at` datetime DEFAULT NULL COMMENT '授权时间'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `uk_role_menu` (`role_id`, `menu_id`)," +
		"KEY `idx_menu` (`menu_id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色菜单授权表'",
}

func main() {
	ctx := gctx.New()
	for _, sql := range ddl {
		if _, err := g.DB().Exec(ctx, sql); err != nil {
			panic(fmt.Errorf("执行失败: %w", err))
		}
	}
	tables := []string{"sys_role", "sys_user_role", "sys_role_menu"}
	for _, t := range tables {
		count, err := g.DB().GetCount(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'vben_admin' AND table_name = ?", t)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s => %v\n", t, count > 0)
	}
	fmt.Println("migration done")
}
