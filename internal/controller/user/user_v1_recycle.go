package user

import (
	"context"

	v1 "demo/api/user/v1"
	usersvc "demo/internal/service/user"
)

func (c *ControllerV1) Recycle(ctx context.Context, req *v1.RecycleReq) (res *v1.RecycleRes, err error) {
	items, operators, total, err := c.userSvc.Recycle(ctx, usersvc.ListInput{
		Page:     req.Page,
		PageSize: req.PageSize,
		Username: req.Username,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.RecycleRes{
		Items: make([]v1.RecycleItem, 0, len(items)),
		Total: total,
	}
	for _, u := range items {
		item := v1.RecycleItem{
			Id:       u.Id,
			Pid:      u.Pid,
			Username: u.Username,
			RealName: u.RealName,
			Role:     u.Role,
		}
		if u.DeletedAt != nil {
			item.DeletedAt = u.DeletedAt.String()
		}
		if u.DeletedBy > 0 {
			item.Operator = operators[u.DeletedBy]
		}
		res.Items = append(res.Items, item)
	}
	return
}
