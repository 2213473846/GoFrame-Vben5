package sysconfig

import (
	"context"

	v1 "demo/api/sysconfig/v1"
)

// ISysConfigV1 系统参数配置管理接口
type ISysConfigV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error)
	Batch(ctx context.Context, req *v1.BatchReq) (res *v1.BatchRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
	Reset(ctx context.Context, req *v1.ResetReq) (res *v1.ResetRes, err error)
}
