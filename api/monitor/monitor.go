package monitor

import (
	"context"

	"demo/api/monitor/v1"
)

// IMonitorV1 服务监控接口
type IMonitorV1 interface {
	Server(ctx context.Context, req *v1.ServerReq) (res *v1.ServerRes, err error)
}
