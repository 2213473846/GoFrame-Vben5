package menu

import (
	"demo/api/menu"
	menusvc "demo/internal/service/menu"
)

type ControllerV1 struct {
	menuSvc *menusvc.Service
}

func NewV1() menu.IMenuV1 {
	return &ControllerV1{
		menuSvc: menusvc.New(),
	}
}
