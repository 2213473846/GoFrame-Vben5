package menu

import (
	"context"

	"demo/api/menu/v1"
	menusvc "demo/internal/service/menu"
)

// List 菜单管理列表(树形), 仅超级管理员
func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	nodes, err := c.menuSvc.List(ctx)
	if err != nil {
		return nil, err
	}
	res = &v1.ListRes{Items: toMenuItems(nodes)}
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
