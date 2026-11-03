package user

import (
	"context"

	"demo/api/user/v1"
	usersvc "demo/internal/service/user"
)

// Update 更新管理员/用户
func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	err = c.userSvc.Update(ctx, usersvc.UpdateInput{
		Id:         req.Id,
		Username:   req.Username,
		RealName:   req.RealName,
		Avatar:     req.Avatar,
		Password:   req.Password,
		Status:     req.Status,
		Role:       req.Role,
		Remark:     req.Remark,
		RoleIds:    req.RoleIds,
		MaxDevices: req.MaxDevices,
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateRes{}, nil
}
