// 日志清理脚本(薄壳)：按 sys_config 的 log.retainDays 清理过期登录/操作日志，
// 真正逻辑在 internal/service/log.ClearExpired，与定时任务 clearlog 共用同一实现。
// 用法: go run ./utility/clearlog
package main

import (
	"fmt"

	logsvc "demo/internal/service/log"
	sysconfigsvc "demo/internal/service/sysconfig"
	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"

	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()
	retainDays := sysconfigsvc.New().GetInt(ctx, sysconfigsvc.KeyLogRetainDays, 90)
	result, err := logsvc.New().ClearExpired(ctx, retainDays)
	if err != nil {
		panic(fmt.Errorf("清理日志失败: %w", err))
	}
	fmt.Printf("retainDays=%d 登录日志 %d 条 / 操作日志 %d 条\n",
		retainDays, result.LoginLogCount, result.OperationLogCount)
}
