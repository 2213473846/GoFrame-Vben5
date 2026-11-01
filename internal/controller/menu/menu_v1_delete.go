package menu

import (
	"context"

	"demo/api/menu/v1"
)

// Delete 删除菜单/路由, 仅超级管理员
func (c *ControllerV1) Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error) {
	if err = c.menuSvc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	res = &v1.DeleteRes{}
	return
}
