// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package menu

import (
	"context"

	"demo/api/menu/v1"
)

type IMenuV1 interface {
	All(ctx context.Context, req *v1.AllReq) (res *v1.AllRes, err error)
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	GrantInfo(ctx context.Context, req *v1.GrantInfoReq) (res *v1.GrantInfoRes, err error)
	Grant(ctx context.Context, req *v1.GrantReq) (res *v1.GrantRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
}
