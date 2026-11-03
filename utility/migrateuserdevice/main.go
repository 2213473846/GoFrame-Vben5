// 一次性迁移脚本：为 sys_user 增加 max_devices 字段(每用户最大在线设备数)
// 用法: go run ./utility/migrateuserdevice
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

const (
	dbName   = "vben_admin"
	table    = "sys_user"
	column   = "max_devices"
	alterDDL = "ALTER TABLE `sys_user` ADD COLUMN `max_devices` int(11) NULL DEFAULT NULL COMMENT '最大在线设备数, NULL=使用全局默认'"
)

func main() {
	ctx := gctx.New()

	count, err := g.DB().GetCount(ctx,
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?",
		dbName, table, column,
	)
	if err != nil {
		panic(err)
	}
	if count > 0 {
		fmt.Printf("%s.%s 已存在, 跳过\n", table, column)
	} else {
		if _, err = g.DB().Exec(ctx, alterDDL); err != nil {
			panic(fmt.Errorf("新增字段失败: %w", err))
		}
		fmt.Printf("%s.%s 已新增\n", table, column)
	}
	fmt.Println("migration done")
}
