// 可重复执行的迁移脚本：创建数据字典类型表与字典项表
// 用法: go run ./utility/migratedict
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var ddl = []string{
	"CREATE TABLE IF NOT EXISTS `sys_dict_type` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '字典类型ID'," +
		"`type` varchar(64) NOT NULL COMMENT '字典类型编码(全局唯一)'," +
		"`name` varchar(64) NOT NULL COMMENT '字典类型名称'," +
		"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用'," +
		"`remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注'," +
		"`created_by` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID'," +
		"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
		"`updated_at` datetime DEFAULT NULL COMMENT '更新时间'," +
		"`deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `uk_type_deleted` (`type`, `deleted_at`)," +
		"KEY `idx_name` (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据字典类型表'",
	"CREATE TABLE IF NOT EXISTS `sys_dict_data` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '字典项ID'," +
		"`dict_type` varchar(64) NOT NULL COMMENT '所属字典类型编码'," +
		"`label` varchar(64) NOT NULL COMMENT '显示名称'," +
		"`value` varchar(64) NOT NULL COMMENT '字典值'," +
		"`sort` int(11) NOT NULL DEFAULT 0 COMMENT '排序(升序)'," +
		"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用'," +
		"`tag_color` varchar(32) NOT NULL DEFAULT '' COMMENT '前端标签颜色: success/warning/danger/info'," +
		"`remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注'," +
		"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
		"`updated_at` datetime DEFAULT NULL COMMENT '更新时间'," +
		"`deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)'," +
		"PRIMARY KEY (`id`)," +
		"UNIQUE KEY `uk_dict_type_value_deleted` (`dict_type`, `value`, `deleted_at`)," +
		"KEY `idx_dict_type` (`dict_type`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据字典项表'",
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
	tables := []string{"sys_dict_type", "sys_dict_data"}
	for _, t := range tables {
		count, err := g.DB().GetCount(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", dbName.String(), t)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s => %v\n", t, count > 0)
	}
	fmt.Println("migration done")
}
