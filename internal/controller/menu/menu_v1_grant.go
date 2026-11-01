package menu

import (
	"context"

	"demo/api/menu/v1"
)

// Grant 保存用户菜单授权(全量覆盖)
func (c *ControllerV1) Grant(ctx context.Context, req *v1.GrantReq) (res *v1.GrantRes, err error) {
	if err = c.menuSvc.Grant(ctx, req.UserId, req.MenuIds); err != nil {
		return nil, err
	}
	res = &v1.GrantRes{}
	return
}
