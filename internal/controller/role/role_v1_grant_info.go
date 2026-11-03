package role

import (
	"context"

	"demo/api/menu/v1"
	roleapi "demo/api/role/v1"
	menusvc "demo/internal/service/menu"
)

// GrantInfo 查询目标角色的可授权菜单与已授权菜单
func (c *ControllerV1) GrantInfo(ctx context.Context, req *roleapi.GrantInfoReq) (res *roleapi.GrantInfoRes, err error) {
	target, nodes, checked, err := c.roleSvc.GrantInfo(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if checked == nil {
		checked = []uint64{}
	}
	res = &roleapi.GrantInfoRes{
		RoleId:     target.Id,
		Name:       target.Name,
		Items:      toMenuItems(nodes),
		CheckedIds: checked,
	}
	return
}

// toMenuItems 服务层菜单树转 API 菜单树
func toMenuItems(in []*menusvc.MenuNode) []*v1.MenuItem {
	if len(in) == 0 {
		return []*v1.MenuItem{}
	}
	out := make([]*v1.MenuItem, 0, len(in))
	for _, node := range in {
		out = append(out, toMenuItem(node))
	}
	return out
}

func toMenuItem(node *menusvc.MenuNode) *v1.MenuItem {
	m := node.Menu
	item := &v1.MenuItem{
		Id:         m.Id,
		Pid:        m.Pid,
		Name:       m.Name,
		Path:       m.Path,
		Component:  m.Component,
		Redirect:   m.Redirect,
		Type:       m.Type,
		Title:      m.Title,
		Icon:       m.Icon,
		Sort:       m.Sort,
		Status:     m.Status,
		HideInMenu: m.HideInMenu == 1,
		KeepAlive:  m.KeepAlive == 1,
		AffixTab:   m.AffixTab == 1,
		IsPublic:   m.IsPublic == 1,
		Remark:     m.Remark,
	}
	if m.CreatedAt != nil {
		item.CreatedAt = m.CreatedAt.String()
	}
	if len(node.Children) > 0 {
		item.Children = toMenuItems(node.Children)
	}
	return item
}
