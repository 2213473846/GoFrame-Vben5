// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package role

import (
	"context"

	"demo/api/role/v1"
)

type IRoleV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	GrantInfo(ctx context.Context, req *v1.GrantInfoReq) (res *v1.GrantInfoRes, err error)
	Grant(ctx context.Context, req *v1.GrantReq) (res *v1.GrantRes, err error)
}
