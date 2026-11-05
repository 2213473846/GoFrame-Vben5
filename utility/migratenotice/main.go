// 可重复执行的迁移脚本：创建通知公告相关表(sys_notice / sys_notice_user / sys_notice_read)
// 用法: go run ./utility/migratenotice
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var ddl = []struct {
	table string
	sql   string
}{
	{
		table: "sys_notice",
		sql: "CREATE TABLE IF NOT EXISTS `sys_notice` (" +
			"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '公告ID'," +
			"`title` varchar(128) NOT NULL COMMENT '标题'," +
			"`content` text NOT NULL COMMENT '正文(纯文本, 不含HTML)'," +
			"`notice_type` varchar(32) NOT NULL DEFAULT '' COMMENT '类型(字典 notice_type 的值)'," +
			"`notice_level` varchar(32) NOT NULL DEFAULT '' COMMENT '级别(字典 notice_level 的值)'," +
			"`status` tinyint(4) NOT NULL DEFAULT 0 COMMENT '状态: 0=草稿 1=已发布 2=已撤回'," +
			"`target_scope` tinyint(4) NOT NULL DEFAULT 1 COMMENT '可见范围: 1=全员 2=指定用户'," +
			"`publisher_id` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '发布人ID'," +
			"`publish_at` datetime DEFAULT NULL COMMENT '发布时间'," +
			"`expire_at` datetime DEFAULT NULL COMMENT '失效时间, 空表示永不过期'," +
			"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
			"`updated_at` datetime DEFAULT NULL COMMENT '更新时间'," +
			"`deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)'," +
			"PRIMARY KEY (`id`)," +
			"KEY `idx_status` (`status`, `deleted_at`)," +
			"KEY `idx_publish_at` (`publish_at`)," +
			"KEY `idx_publisher` (`publisher_id`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知公告表'",
	},
	{
		table: "sys_notice_user",
		sql: "CREATE TABLE IF NOT EXISTS `sys_notice_user` (" +
			"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键'," +
			"`notice_id` bigint(20) unsigned NOT NULL COMMENT '公告ID'," +
			"`user_id` bigint(20) unsigned NOT NULL COMMENT '接收用户ID'," +
			"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `uk_notice_user` (`notice_id`, `user_id`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='公告定向用户表'",
	},
	{
		table: "sys_notice_read",
		sql: "CREATE TABLE IF NOT EXISTS `sys_notice_read` (" +
			"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键'," +
			"`notice_id` bigint(20) unsigned NOT NULL COMMENT '公告ID'," +
			"`user_id` bigint(20) unsigned NOT NULL COMMENT '用户ID'," +
			"`read_at` datetime DEFAULT NULL COMMENT '阅读时间'," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE KEY `uk_notice_user` (`notice_id`, `user_id`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='公告阅读记录表'",
	},
}

func main() {
	ctx := gctx.New()
	for _, item := range ddl {
		if _, err := g.DB().Exec(ctx, item.sql); err != nil {
			panic(fmt.Errorf("执行 %s 失败: %w", item.table, err))
		}
	}
	dbName, err := g.DB().GetValue(ctx, "SELECT DATABASE()")
	if err != nil {
		panic(err)
	}
	for _, item := range ddl {
		count, err := g.DB().GetCount(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", dbName.String(), item.table)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s => %v\n", item.table, count > 0)
	}
	fmt.Println("migration done")
}
