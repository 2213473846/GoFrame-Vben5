// 一次性迁移脚本：创建登录日志表与操作日志表
// 用法: go run ./utility/migratelog
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

var ddl = []string{
	"CREATE TABLE IF NOT EXISTS `sys_login_log` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键'," +
		"`user_id` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '用户ID, 登录失败且用户不存在时为0'," +
		"`username` varchar(64) NOT NULL DEFAULT '' COMMENT '登录用户名'," +
		"`ip` varchar(64) NOT NULL DEFAULT '' COMMENT '登录地址(IP)'," +
		"`location` varchar(128) NOT NULL DEFAULT '' COMMENT '登录地点'," +
		"`browser` varchar(64) NOT NULL DEFAULT '' COMMENT '浏览器'," +
		"`os` varchar(64) NOT NULL DEFAULT '' COMMENT '操作系统'," +
		"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '登录状态: 1=成功 0=失败'," +
		"`message` varchar(255) NOT NULL DEFAULT '' COMMENT '登录信息'," +
		"`created_at` datetime DEFAULT NULL COMMENT '登录时间'," +
		"PRIMARY KEY (`id`)," +
		"KEY `idx_user_id` (`user_id`)," +
		"KEY `idx_created_at` (`created_at`)," +
		"KEY `idx_username` (`username`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='登录日志表'",
	"CREATE TABLE IF NOT EXISTS `sys_operation_log` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键'," +
		"`user_id` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '用户ID'," +
		"`username` varchar(64) NOT NULL DEFAULT '' COMMENT '用户名'," +
		"`module` varchar(64) NOT NULL DEFAULT '' COMMENT '操作模块'," +
		"`action` varchar(64) NOT NULL DEFAULT '' COMMENT '操作类型'," +
		"`method` varchar(16) NOT NULL DEFAULT '' COMMENT '请求方法'," +
		"`path` varchar(255) NOT NULL DEFAULT '' COMMENT '请求地址'," +
		"`ip` varchar(64) NOT NULL DEFAULT '' COMMENT '操作地址(IP)'," +
		"`location` varchar(128) NOT NULL DEFAULT '' COMMENT '操作地点'," +
		"`browser` varchar(64) NOT NULL DEFAULT '' COMMENT '浏览器'," +
		"`os` varchar(64) NOT NULL DEFAULT '' COMMENT '操作系统'," +
		"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '操作状态: 1=成功 0=失败'," +
		"`message` varchar(255) NOT NULL DEFAULT '' COMMENT '操作信息'," +
		"`created_at` datetime DEFAULT NULL COMMENT '操作时间'," +
		"PRIMARY KEY (`id`)," +
		"KEY `idx_user_id` (`user_id`)," +
		"KEY `idx_created_at` (`created_at`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作日志表'",
}

func main() {
	ctx := gctx.New()
	for _, sql := range ddl {
		if _, err := g.DB().Exec(ctx, sql); err != nil {
			panic(fmt.Errorf("执行失败: %w", err))
		}
	}
	tables := []string{"sys_login_log", "sys_operation_log"}
	for _, t := range tables {
		count, err := g.DB().GetCount(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'vben_admin' AND table_name = ?", t)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s => %v\n", t, count > 0)
	}
	fmt.Println("migration done")
}
