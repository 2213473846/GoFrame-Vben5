// 可重复执行的迁移脚本：创建系统参数配置表
// 用法: go run ./utility/migrateconfig
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var ddl = []string{
	"CREATE TABLE IF NOT EXISTS `sys_config` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '参数ID'," +
		"`config_key` varchar(128) NOT NULL COMMENT '参数键(唯一)'," +
		"`config_value` text NOT NULL COMMENT '参数值'," +
		"`name` varchar(64) NOT NULL DEFAULT '' COMMENT '参数名称'," +
		"`remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注'," +
		"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用'," +
		"`builtin` tinyint(4) NOT NULL DEFAULT 0 COMMENT '是否内置: 1=内置(不可删除, key不可修改)'," +
		"`created_by` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID'," +
		"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
		"`updated_at` datetime DEFAULT NULL COMMENT '更新时间'," +
		"`deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `uk_config_key_deleted` (`config_key`, `deleted_at`)," +
		"KEY `idx_name` (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统参数配置表'",
}

func main() {
	ctx := gctx.New()
	for _, sql := range ddl {
		if _, err := g.DB().Exec(ctx, sql); err != nil {
			panic(fmt.Errorf("执行失败: %w", err))
		}
	}
	dbName, err := g.DB().GetValue(ctx, "SELECT DATABASE()")
	if err != nil {
		panic(err)
	}
	count, err := g.DB().GetCount(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", dbName.String(), "sys_config")
	if err != nil {
		panic(err)
	}
	fmt.Printf("sys_config => %v\n", count > 0)
	fmt.Println("migration done")
}
