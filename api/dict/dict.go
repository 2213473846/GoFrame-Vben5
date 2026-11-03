// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package dict

import (
	"context"

	"demo/api/dict/v1"
)

type IDictV1 interface {
	TypeList(ctx context.Context, req *v1.TypeListReq) (res *v1.TypeListRes, err error)
	TypeCreate(ctx context.Context, req *v1.TypeCreateReq) (res *v1.TypeCreateRes, err error)
	TypeUpdate(ctx context.Context, req *v1.TypeUpdateReq) (res *v1.TypeUpdateRes, err error)
	TypeDelete(ctx context.Context, req *v1.TypeDeleteReq) (res *v1.TypeDeleteRes, err error)
	TypeStatus(ctx context.Context, req *v1.TypeStatusReq) (res *v1.TypeStatusRes, err error)
	DataList(ctx context.Context, req *v1.DataListReq) (res *v1.DataListRes, err error)
	DataCreate(ctx context.Context, req *v1.DataCreateReq) (res *v1.DataCreateRes, err error)
	DataUpdate(ctx context.Context, req *v1.DataUpdateReq) (res *v1.DataUpdateRes, err error)
	DataDelete(ctx context.Context, req *v1.DataDeleteReq) (res *v1.DataDeleteRes, err error)
	DataStatus(ctx context.Context, req *v1.DataStatusReq) (res *v1.DataStatusRes, err error)
	DataBatch(ctx context.Context, req *v1.DataBatchReq) (res *v1.DataBatchRes, err error)
}
