// Package testutil 提供基于本机 MySQL 的集成测试基础设施。
// 仅被各服务包的 _test.go 引用, 不会编译进正式二进制。
// 约定:
//   - 连接本机 MySQL(127.0.0.1:3306, root/123456, 与 manifest/config/config.example.yaml 一致)
//   - 使用独立的 vben_admin_test 库, 每次测试运行前 DROP + CREATE, 不影响开发库
//   - MySQL 不可用时 Prepare 返回 false, 集成测试自动跳过
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

// TestDBName 集成测试专用数据库名前缀
const TestDBName = "vben_admin_test"

// serverLink 不指定库的连接串, 用于建库
const serverLink = "mysql:root:123456@tcp(127.0.0.1:3306)/"

// testDBName 按测试包隔离数据库(取当前工作目录的包名后缀), 避免 `go test ./...`
// 并行执行多个包时互相 DROP/CREATE 同一个库。
func testDBName() string {
	wd, err := os.Getwd()
	if err != nil {
		return TestDBName
	}
	base := filepath.Base(wd)
	const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	base = strings.Map(func(r rune) rune {
		if strings.ContainsRune(allowed, r) {
			return r
		}
		return -1
	}, base)
	if base == "" {
		return TestDBName
	}
	return TestDBName + "_" + base
}

// ddl 测试所需表的建表语句(与 manifest/sql/create.sql 保持一致)
var ddl = []string{
	"CREATE TABLE IF NOT EXISTS `sys_user` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '用户ID'," +
		"`pid` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '父级管理员ID'," +
		"`username` varchar(64) NOT NULL COMMENT '登录用户名'," +
		"`password` varchar(100) NOT NULL COMMENT '密码(bcrypt)'," +
		"`real_name` varchar(64) NOT NULL DEFAULT '' COMMENT '昵称'," +
		"`avatar` varchar(255) NOT NULL DEFAULT '' COMMENT '头像地址'," +
		"`role` varchar(32) NOT NULL DEFAULT 'user' COMMENT '角色'," +
		"`status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '状态'," +
		"`remark` varchar(255) NOT NULL DEFAULT '' COMMENT '备注'," +
		"`max_devices` int(11) NULL DEFAULT NULL COMMENT '最大同时在线设备数'," +
		"`must_change_password` tinyint(4) NOT NULL DEFAULT 0 COMMENT '是否需强制修改密码'," +
		"`created_at` datetime DEFAULT NULL," +
		"`updated_at` datetime DEFAULT NULL," +
		"`deleted_at` datetime DEFAULT NULL," +
		"`deleted_by` bigint(20) unsigned NULL DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_username_deleted` (`username`, `deleted_at`), KEY `idx_pid` (`pid`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_menu` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`pid` bigint(20) unsigned NOT NULL DEFAULT 0," +
		"`name` varchar(64) NOT NULL," +
		"`path` varchar(128) NOT NULL," +
		"`component` varchar(255) NOT NULL DEFAULT ''," +
		"`redirect` varchar(255) NOT NULL DEFAULT ''," +
		"`type` varchar(16) NOT NULL DEFAULT 'menu'," +
		"`title` varchar(64) NOT NULL," +
		"`icon` varchar(64) NOT NULL DEFAULT ''," +
		"`sort` int(11) NOT NULL DEFAULT 0," +
		"`status` tinyint(4) NOT NULL DEFAULT 1," +
		"`hide_in_menu` tinyint(4) NOT NULL DEFAULT 0," +
		"`keep_alive` tinyint(4) NOT NULL DEFAULT 0," +
		"`affix_tab` tinyint(4) NOT NULL DEFAULT 0," +
		"`is_public` tinyint(4) NOT NULL DEFAULT 0," +
		"`remark` varchar(255) NOT NULL DEFAULT ''," +
		"`created_by` bigint(20) unsigned NOT NULL DEFAULT 0," +
		"`created_at` datetime DEFAULT NULL," +
		"`updated_at` datetime DEFAULT NULL," +
		"`deleted_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_name_deleted` (`name`, `deleted_at`), KEY `idx_pid` (`pid`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_user_menu` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`user_id` bigint(20) unsigned NOT NULL," +
		"`menu_id` bigint(20) unsigned NOT NULL," +
		"`created_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_user_menu` (`user_id`, `menu_id`), KEY `idx_menu` (`menu_id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_role` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`code` varchar(64) NOT NULL," +
		"`name` varchar(64) NOT NULL," +
		"`remark` varchar(255) NOT NULL DEFAULT ''," +
		"`status` tinyint(4) NOT NULL DEFAULT 1," +
		"`created_by` bigint(20) unsigned NOT NULL DEFAULT 0," +
		"`created_at` datetime DEFAULT NULL," +
		"`updated_at` datetime DEFAULT NULL," +
		"`deleted_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_code_deleted` (`code`, `deleted_at`), KEY `idx_name` (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_user_role` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`user_id` bigint(20) unsigned NOT NULL," +
		"`role_id` bigint(20) unsigned NOT NULL," +
		"`created_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_user_role` (`user_id`, `role_id`), KEY `idx_role` (`role_id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_role_menu` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`role_id` bigint(20) unsigned NOT NULL," +
		"`menu_id` bigint(20) unsigned NOT NULL," +
		"`created_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_role_menu` (`role_id`, `menu_id`), KEY `idx_menu` (`menu_id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_login_log` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`user_id` bigint(20) unsigned NOT NULL DEFAULT 0," +
		"`username` varchar(64) NOT NULL DEFAULT ''," +
		"`ip` varchar(64) NOT NULL DEFAULT ''," +
		"`location` varchar(128) NOT NULL DEFAULT ''," +
		"`browser` varchar(64) NOT NULL DEFAULT ''," +
		"`os` varchar(64) NOT NULL DEFAULT ''," +
		"`status` tinyint(4) NOT NULL DEFAULT 1," +
		"`message` varchar(255) NOT NULL DEFAULT ''," +
		"`created_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), KEY `idx_user_id` (`user_id`), KEY `idx_created_at` (`created_at`), KEY `idx_username` (`username`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_operation_log` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`user_id` bigint(20) unsigned NOT NULL DEFAULT 0," +
		"`username` varchar(64) NOT NULL DEFAULT ''," +
		"`module` varchar(64) NOT NULL DEFAULT ''," +
		"`action` varchar(64) NOT NULL DEFAULT ''," +
		"`method` varchar(16) NOT NULL DEFAULT ''," +
		"`path` varchar(255) NOT NULL DEFAULT ''," +
		"`ip` varchar(64) NOT NULL DEFAULT ''," +
		"`location` varchar(128) NOT NULL DEFAULT ''," +
		"`browser` varchar(64) NOT NULL DEFAULT ''," +
		"`os` varchar(64) NOT NULL DEFAULT ''," +
		"`status` tinyint(4) NOT NULL DEFAULT 1," +
		"`message` varchar(255) NOT NULL DEFAULT ''," +
		"`created_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), KEY `idx_user_id` (`user_id`), KEY `idx_created_at` (`created_at`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_dict_type` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`type` varchar(64) NOT NULL," +
		"`name` varchar(64) NOT NULL," +
		"`status` tinyint(4) NOT NULL DEFAULT 1," +
		"`remark` varchar(255) NOT NULL DEFAULT ''," +
		"`created_by` bigint(20) unsigned NOT NULL DEFAULT 0," +
		"`created_at` datetime DEFAULT NULL," +
		"`updated_at` datetime DEFAULT NULL," +
		"`deleted_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_type_deleted` (`type`, `deleted_at`), KEY `idx_name` (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_dict_data` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`dict_type` varchar(64) NOT NULL," +
		"`label` varchar(64) NOT NULL," +
		"`value` varchar(64) NOT NULL," +
		"`sort` int(11) NOT NULL DEFAULT 0," +
		"`status` tinyint(4) NOT NULL DEFAULT 1," +
		"`tag_color` varchar(32) NOT NULL DEFAULT ''," +
		"`remark` varchar(255) NOT NULL DEFAULT ''," +
		"`created_at` datetime DEFAULT NULL," +
		"`updated_at` datetime DEFAULT NULL," +
		"`deleted_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_dict_type_value_deleted` (`dict_type`, `value`, `deleted_at`), KEY `idx_dict_type` (`dict_type`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	"CREATE TABLE IF NOT EXISTS `sys_config` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`config_key` varchar(128) NOT NULL," +
		"`config_value` text NOT NULL," +
		"`name` varchar(64) NOT NULL DEFAULT ''," +
		"`remark` varchar(255) NOT NULL DEFAULT ''," +
		"`status` tinyint(4) NOT NULL DEFAULT 1," +
		"`builtin` tinyint(4) NOT NULL DEFAULT 0," +
		"`created_by` bigint(20) unsigned NOT NULL DEFAULT 0," +
		"`created_at` datetime DEFAULT NULL," +
		"`updated_at` datetime DEFAULT NULL," +
		"`deleted_at` datetime DEFAULT NULL," +
		"PRIMARY KEY (`id`), UNIQUE KEY `uk_config_key_deleted` (`config_key`, `deleted_at`), KEY `idx_name` (`name`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
}

// Prepare 准备集成测试数据库: 重建当前包专属测试库并建表, 注册为默认数据库组。
// MySQL 不可用时返回 false(调用方应跳过集成测试)。
func Prepare() bool {
	ctx := gctx.New()
	name := testDBName()
	// 建库(服务器级连接)
	server, err := gdb.New(gdb.ConfigNode{Type: "mysql", Link: serverLink})
	if err != nil {
		return false
	}
	if _, err = server.Exec(ctx, "DROP DATABASE IF EXISTS `"+name+"`"); err != nil {
		return false
	}
	if _, err = server.Exec(ctx, "CREATE DATABASE `"+name+"` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		return false
	}
	// 注册默认组并建表
	gdb.SetConfig(gdb.Config{
		gdb.DefaultGroupName: gdb.ConfigGroup{
			{Type: "mysql", Link: "mysql:root:123456@tcp(127.0.0.1:3306)/" + name},
		},
	})
	db, err := gdb.New(gdb.ConfigNode{Type: "mysql", Link: "mysql:root:123456@tcp(127.0.0.1:3306)/" + name})
	if err != nil {
		return false
	}
	for _, sql := range ddl {
		if _, err = db.Exec(ctx, sql); err != nil {
			return false
		}
	}
	return true
}

// Ctx 返回测试上下文
func Ctx() context.Context {
	return gctx.New()
}

// PrepareRedis 配置会话测试专用的 Redis DB(15) 并清空, 避免污染开发库(DB 0)。
func PrepareRedis() bool {
	return PrepareRedisDB(15)
}

// PrepareRedisDB 按测试包选择独立 Redis DB 并清空, 避免 `go test ./...` 并行时互相 FLUSHDB。
func PrepareRedisDB(db int) bool {
	gredis.SetConfig(&gredis.Config{
		Address: "127.0.0.1:6379",
		Db:      db,
	})
	ctx := gctx.New()
	redis := g.Redis()
	if redis == nil {
		return false
	}
	if _, err := redis.Do(ctx, "PING"); err != nil {
		return false
	}
	if _, err := redis.Do(ctx, "FLUSHDB"); err != nil {
		return false
	}
	return true
}
