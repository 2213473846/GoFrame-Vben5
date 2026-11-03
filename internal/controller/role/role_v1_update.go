package role

import (
	"context"

	"demo/api/role/v1"
	rolesvc "demo/internal/service/role"
)

// Update 修改角色
func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	err = c.roleSvc.Update(ctx, rolesvc.UpdateInput{
		Id: req.Id,
		CreateInput: rolesvc.CreateInput{
			Code:   req.Code,
			Name:   req.Name,
			Remark: req.Remark,
			Status: req.Status,
		},
	})
	if err != nil {
		return nil, err
	}
	res = &v1.UpdateRes{}
	return
}
