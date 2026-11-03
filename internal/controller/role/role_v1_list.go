package role

import (
	"context"

	"demo/api/role/v1"
)

// List 功能角色列表
func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	roles, err := c.roleSvc.List(ctx)
	if err != nil {
		return nil, err
	}
	res = &v1.ListRes{
		Items: make([]v1.RoleItem, 0, len(roles)),
	}
	for _, r := range roles {
		item := v1.RoleItem{
			Id:     r.Id,
			Code:   r.Code,
			Name:   r.Name,
			Remark: r.Remark,
			Status: r.Status,
		}
		if r.CreatedAt != nil {
			item.CreatedAt = r.CreatedAt.String()
		}
		res.Items = append(res.Items, item)
	}
	return
}
