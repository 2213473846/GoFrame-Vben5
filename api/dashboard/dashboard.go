package dashboard

import (
	"context"

	v1 "demo/api/dashboard/v1"
)

// IDashboardV1 工作台接口
type IDashboardV1 interface {
	Overview(ctx context.Context, req *v1.OverviewReq) (res *v1.OverviewRes, err error)
}
