package user

import (
	"context"

	"demo/api/user/v1"
	usersvc "demo/internal/service/user"
)

// List 用户/管理员列表
func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	users, parents, total, err := c.userSvc.List(ctx, usersvc.ListInput{
		Page:     req.Page,
		PageSize: req.PageSize,
		Username: req.Username,
		RealName: req.RealName,
		Role:     req.Role,
		Status:   req.Status,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.ListRes{
		Items: make([]v1.ListItem, 0, len(users)),
		Total: total,
	}
	for _, u := range users {
		item := v1.ListItem{
			Id:       u.Id,
			Pid:      u.Pid,
			Username: u.Username,
			RealName: u.RealName,
			Avatar:   u.Avatar,
			Role:     u.Role,
			Status:   u.Status,
			Remark:   u.Remark,
		}
		if u.CreatedAt != nil {
			item.CreatedAt = u.CreatedAt.String()
		}
		if u.Pid > 0 {
			if p, ok := parents[u.Pid]; ok {
				item.ParentUsername = p.Username
				item.ParentRealName = p.RealName
			}
		}
		res.Items = append(res.Items, item)
	}
	return
}
