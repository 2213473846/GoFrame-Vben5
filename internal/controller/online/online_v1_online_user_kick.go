package online

import (
	"context"

	v1 "demo/api/online/v1"
	onlinesvc "demo/internal/service/online"
)

// OnlineUserKick 强退用户
func (c *ControllerV1) OnlineUserKick(ctx context.Context, req *v1.OnlineUserKickReq) (res *v1.OnlineUserKickRes, err error) {
	if err = onlinesvc.Kick(ctx, req.UserId); err != nil {
		return nil, err
	}
	return &v1.OnlineUserKickRes{}, nil
}
