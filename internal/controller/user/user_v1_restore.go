package user

import (
	"context"

	v1 "demo/api/user/v1"
)

func (c *ControllerV1) Restore(ctx context.Context, req *v1.RestoreReq) (res *v1.RestoreRes, err error) {
	if err = c.userSvc.Restore(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.RestoreRes{}, nil
}
