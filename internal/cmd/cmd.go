package cmd

import (
	"context"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"demo/internal/controller/auth"
	"demo/internal/controller/dashboard"
	"demo/internal/controller/dict"
	"demo/internal/controller/job"
	"demo/internal/controller/log"
	"demo/internal/controller/menu"
	"demo/internal/controller/monitor"
	"demo/internal/controller/notice"
	"demo/internal/controller/online"
	"demo/internal/controller/role"
	"demo/internal/controller/sysconfig"
	"demo/internal/controller/user"
	"demo/internal/service/boot"
	"demo/internal/service/middleware"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			// 启动前初始化(幂等): 确保种子账号存在
			if err = boot.Run(ctx); err != nil {
				return err
			}
			var (
				s             = g.Server()
				middlewareSvc = middleware.New()
			)
			// 统一响应包装(vben 格式)
			s.Use(middlewareSvc.Response)
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(
					ghttp.MiddlewareCORS,
					middlewareSvc.Auth,
				)
				group.Bind(
					auth.NewV1(),
					user.NewV1(),
					menu.NewV1(),
					role.NewV1(),
					log.NewV1(),
					online.NewV1(),
					dict.NewV1(),
					sysconfig.NewV1(),
					dashboard.NewV1(),
					monitor.NewV1(),
					job.NewV1(),
					notice.NewV1(),
				)
			})
			s.Run()
			return nil
		},
	}
)
