// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package user

import (
	"context"

	"demo/api/user/v1"
)

type IUserV1 interface {
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	Info(ctx context.Context, req *v1.InfoReq) (res *v1.InfoRes, err error)
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Purge(ctx context.Context, req *v1.PurgeReq) (res *v1.PurgeRes, err error)
	Recycle(ctx context.Context, req *v1.RecycleReq) (res *v1.RecycleRes, err error)
	Restore(ctx context.Context, req *v1.RestoreReq) (res *v1.RestoreRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
	Export(ctx context.Context, req *v1.ExportReq) (res *v1.ExportRes, err error)
	ImportTemplate(ctx context.Context, req *v1.ImportTemplateReq) (res *v1.ImportTemplateRes, err error)
	Import(ctx context.Context, req *v1.ImportReq) (res *v1.ImportRes, err error)
}
