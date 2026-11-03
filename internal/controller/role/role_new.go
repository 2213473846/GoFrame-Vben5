package role

import (
	"demo/api/role"
	rolesvc "demo/internal/service/role"
)

type ControllerV1 struct {
	roleSvc *rolesvc.Service
}

func NewV1() role.IRoleV1 {
	return &ControllerV1{
		roleSvc: rolesvc.New(),
	}
}
