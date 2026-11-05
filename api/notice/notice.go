package notice

import (
	"context"

	"demo/api/notice/v1"
)

// INoticeV1 通知公告接口
type INoticeV1 interface {
	List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error)
	Detail(ctx context.Context, req *v1.DetailReq) (res *v1.DetailRes, err error)
	Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error)
	Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error)
	Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error)
	Publish(ctx context.Context, req *v1.PublishReq) (res *v1.PublishRes, err error)
	Revoke(ctx context.Context, req *v1.RevokeReq) (res *v1.RevokeRes, err error)
	MyList(ctx context.Context, req *v1.MyListReq) (res *v1.MyListRes, err error)
	UnreadCount(ctx context.Context, req *v1.UnreadCountReq) (res *v1.UnreadCountRes, err error)
	Read(ctx context.Context, req *v1.ReadReq) (res *v1.ReadRes, err error)
}
