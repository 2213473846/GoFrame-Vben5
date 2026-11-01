package user

import (
	"context"

	v1 "demo/api/user/v1"
)

func (c *ControllerV1) Purge(ctx context.Context, req *v1.PurgeReq) (res *v1.PurgeRes, err error) {
	if err = c.userSvc.Purge(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.PurgeRes{}, nil
}
