// 一次性清理脚本：清空日志测试数据
// 用法: go run ./utility/clearlog
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()
	for _, table := range []string{"sys_login_log", "sys_operation_log"} {
		if _, err := g.DB().Exec(ctx, "TRUNCATE TABLE `"+table+"`"); err != nil {
			panic(fmt.Errorf("清空 %s 失败: %w", table, err))
		}
		fmt.Printf("%s cleared\n", table)
	}
}
