package dict

import (
	"demo/api/dict"
	dictsvc "demo/internal/service/dict"
)

type ControllerV1 struct {
	dictSvc *dictsvc.Service
}

func NewV1() dict.IDictV1 {
	return &ControllerV1{
		dictSvc: dictsvc.New(),
	}
}
