package job

import (
	"context"

	"demo/api/job/v1"
)

// IJobV1 定时任务管理接口
type IJobV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	ChangeStatus(ctx context.Context, req *v1.StatusReq) (res *v1.StatusRes, err error)
	Run(ctx context.Context, req *v1.RunReq) (res *v1.RunRes, err error)
	Logs(ctx context.Context, req *v1.LogsReq) (res *v1.LogsRes, err error)
}
