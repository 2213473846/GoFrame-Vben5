package role

import (
	"context"

	"demo/api/role/v1"
)

// Delete 删除角色
func (c *ControllerV1) Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error) {
	if err = c.roleSvc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	res = &v1.DeleteRes{}
	return
}
