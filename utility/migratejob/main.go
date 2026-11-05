// 可重复执行的迁移脚本：创建定时任务表(sys_job / sys_job_log)
// 用法: go run ./utility/migratejob
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
		table: "sys_job",
		sql: "CREATE TABLE IF NOT EXISTS `sys_job` (" +
			"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '任务ID'," +
			"`name` varchar(64) NOT NULL COMMENT '任务名称'," +
			"`job_group` varchar(64) NOT NULL DEFAULT 'default' COMMENT '任务分组'," +
			"`invoke_target` varchar(128) NOT NULL COMMENT '白名单任务标识'," +
			"`cron_expression` varchar(64) NOT NULL COMMENT 'cron 表达式(5 段或 6 段)'," +
			"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用'," +
			"`concurrency_policy` tinyint(4) NOT NULL DEFAULT 0 COMMENT '并发策略: 1=允许并发 0=禁止并发'," +
			"`timeout_seconds` int(11) NOT NULL DEFAULT 60 COMMENT '执行超时秒数, 0=不限制'," +
			"`remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注'," +
			"`last_run_at` datetime DEFAULT NULL COMMENT '最近执行时间'," +
			"`last_result` varchar(128) NOT NULL DEFAULT '' COMMENT '最近执行结果文案'," +
			"`next_run_at` datetime DEFAULT NULL COMMENT '下次执行时间'," +
			"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
			"`updated_at` datetime DEFAULT NULL COMMENT '更新时间'," +
			"`deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)'," +
			"PRIMARY KEY (`id`)," +
			"KEY `idx_name` (`name`)," +
			"KEY `idx_status` (`status`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务表'",
	},
	{
		table: "sys_job_log",
		sql: "CREATE TABLE IF NOT EXISTS `sys_job_log` (" +
			"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键'," +
			"`job_id` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '任务ID'," +
			"`job_name` varchar(64) NOT NULL DEFAULT '' COMMENT '任务名称'," +
			"`trigger_type` tinyint(4) NOT NULL DEFAULT 1 COMMENT '触发方式: 1=定时 2=手动'," +
			"`started_at` datetime DEFAULT NULL COMMENT '开始时间'," +
			"`finished_at` datetime DEFAULT NULL COMMENT '结束时间'," +
			"`duration_ms` bigint(20) NOT NULL DEFAULT 0 COMMENT '执行耗时(毫秒)'," +
			"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '执行状态: 1=成功 0=失败'," +
			"`message` text NULL COMMENT '执行结果摘要/失败原因'," +
			"`created_at` datetime DEFAULT NULL COMMENT '创建时间'," +
			"PRIMARY KEY (`id`)," +
			"KEY `idx_job_id` (`job_id`)," +
			"KEY `idx_started_at` (`started_at`)" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务执行日志表'",
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
	// 兼容早期草稿版本: 历史表可能只有 error_message 列, 缺少 message 列时补齐(可重复执行)
	columnCount, err := g.DB().GetCount(ctx,
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = ? AND table_name = 'sys_job_log' AND column_name = 'message'",
		dbName.String())
	if err != nil {
		panic(err)
	}
	if columnCount == 0 {
		if _, err = g.DB().Exec(ctx, "ALTER TABLE `sys_job_log` ADD COLUMN `message` text NULL COMMENT '执行结果摘要/失败原因'"); err != nil {
			panic(fmt.Errorf("补齐 sys_job_log.message 列失败: %w", err))
		}
		fmt.Println("sys_job_log.message => added")
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
