package menu

import (
	"context"

	"demo/api/menu/v1"
	menusvc "demo/internal/service/menu"
)

// All 获取当前用户可见路由(后端下发)
func (c *ControllerV1) All(ctx context.Context, req *v1.AllReq) (res *v1.AllRes, err error) {
	routes, err := c.menuSvc.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make(v1.AllRes, 0, len(routes))
	for _, r := range routes {
		out = append(out, toRouteNode(r))
	}
	res = &out
	return
}

// toRouteNode 服务层路由节点转 API 节点
func toRouteNode(in *menusvc.RouteNode) *v1.RouteNode {
	node := &v1.RouteNode{
		Name:      in.Name,
		Path:      in.Path,
		Component: in.Component,
		Redirect:  in.Redirect,
		Meta: &v1.RouteMeta{
			Title:      in.Meta.Title,
			Icon:       in.Meta.Icon,
			Order:      in.Meta.Order,
			HideInMenu: in.Meta.HideInMenu,
			KeepAlive:  in.Meta.KeepAlive,
			AffixTab:   in.Meta.AffixTab,
		},
	}
	if len(in.Children) > 0 {
		node.Children = make([]*v1.RouteNode, 0, len(in.Children))
		for _, child := range in.Children {
			node.Children = append(node.Children, toRouteNode(child))
		}
	}
	return node
}
