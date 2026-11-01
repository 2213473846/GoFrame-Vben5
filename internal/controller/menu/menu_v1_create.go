package menu

import (
	"context"

	"demo/api/menu/v1"
	menusvc "demo/internal/service/menu"
)

// Create 新增菜单/路由, 仅超级管理员
func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	id, err := c.menuSvc.Create(ctx, menusvc.CreateInput{
		Pid:        req.Pid,
		Name:       req.Name,
		Path:       req.Path,
		Component:  req.Component,
		Redirect:   req.Redirect,
		Type:       req.Type,
		Title:      req.Title,
		Icon:       req.Icon,
		Sort:       req.Sort,
		Status:     req.Status,
		HideInMenu: req.HideInMenu,
		KeepAlive:  req.KeepAlive,
		AffixTab:   req.AffixTab,
		IsPublic:   req.IsPublic,
		Remark:     req.Remark,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.CreateRes{Id: id}
	return
}
