package menu

import (
	"context"

	"demo/api/menu/v1"
)

// GrantInfo 查询目标用户的可授权菜单与已授权菜单
func (c *ControllerV1) GrantInfo(ctx context.Context, req *v1.GrantInfoReq) (res *v1.GrantInfoRes, err error) {
	target, nodes, checked, err := c.menuSvc.GrantInfo(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if checked == nil {
		checked = []uint64{}
	}
	res = &v1.GrantInfoRes{
		UserId:     target.Id,
		Username:   target.Username,
		Items:      toMenuItems(nodes),
		CheckedIds: checked,
	}
	return
}
