package role

import (
	"context"

	"demo/api/role/v1"
)

// Grant 保存角色菜单授权(全量覆盖)
func (c *ControllerV1) Grant(ctx context.Context, req *v1.GrantReq) (res *v1.GrantRes, err error) {
	if err = c.roleSvc.Grant(ctx, req.RoleId, req.MenuIds); err != nil {
		return nil, err
	}
	res = &v1.GrantRes{}
	return
}
