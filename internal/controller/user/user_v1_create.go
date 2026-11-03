package user

import (
	"context"

	"demo/api/user/v1"
	usersvc "demo/internal/service/user"
)

// Create 创建代理/用户
func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	id, err := c.userSvc.Create(ctx, usersvc.CreateInput{
		Username:   req.Username,
		Password:   req.Password,
		RealName:   req.RealName,
		Role:       req.Role,
		Remark:     req.Remark,
		RoleIds:    req.RoleIds,
		MaxDevices: req.MaxDevices,
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateRes{Id: id}, nil
}
