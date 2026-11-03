package role

import (
	"context"

	"demo/api/role/v1"
	rolesvc "demo/internal/service/role"
)

// Create 新增角色
func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	id, err := c.roleSvc.Create(ctx, rolesvc.CreateInput{
		Code:   req.Code,
		Name:   req.Name,
		Remark: req.Remark,
		Status: req.Status,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.CreateRes{Id: id}
	return
}
