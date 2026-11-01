// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package online

import (
	"context"

	"demo/api/online/v1"
)

type IOnlineV1 interface {
	OnlineUserKick(ctx context.Context, req *v1.OnlineUserKickReq) (res *v1.OnlineUserKickRes, err error)
	OnlineUserList(ctx context.Context, req *v1.OnlineUserListReq) (res *v1.OnlineUserListRes, err error)
}
