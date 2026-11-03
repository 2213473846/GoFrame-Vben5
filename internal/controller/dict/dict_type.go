package dict

import (
	"context"

	"demo/api/dict/v1"
	dictsvc "demo/internal/service/dict"
)

// TypeList 字典类型列表
func (c *ControllerV1) TypeList(ctx context.Context, req *v1.TypeListReq) (res *v1.TypeListRes, err error) {
	types, err := c.dictSvc.TypeList(ctx, dictsvc.TypeListInput{
		Keyword: req.Keyword,
		Status:  req.Status,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.TypeListRes{Items: make([]v1.TypeItem, 0, len(types))}
	for _, t := range types {
		item := v1.TypeItem{
			Id:     t.Id,
			Type:   t.Type,
			Name:   t.Name,
			Status: t.Status,
			Remark: t.Remark,
		}
		if t.CreatedAt != nil {
			item.CreatedAt = t.CreatedAt.String()
		}
		res.Items = append(res.Items, item)
	}
	return
}

// TypeCreate 新增字典类型
func (c *ControllerV1) TypeCreate(ctx context.Context, req *v1.TypeCreateReq) (res *v1.TypeCreateRes, err error) {
	id, err := c.dictSvc.TypeCreate(ctx, dictsvc.TypeCreateInput{
		Type:   req.Type,
		Name:   req.Name,
		Status: req.Status,
		Remark: req.Remark,
	})
	if err != nil {
		return nil, err
	}
	return &v1.TypeCreateRes{Id: id}, nil
}

// TypeUpdate 修改字典类型
func (c *ControllerV1) TypeUpdate(ctx context.Context, req *v1.TypeUpdateReq) (res *v1.TypeUpdateRes, err error) {
	if err = c.dictSvc.TypeUpdate(ctx, dictsvc.TypeUpdateInput{
		Id:     req.Id,
		Name:   req.Name,
		Status: req.Status,
		Remark: req.Remark,
	}); err != nil {
		return nil, err
	}
	return &v1.TypeUpdateRes{}, nil
}

// TypeDelete 删除字典类型
func (c *ControllerV1) TypeDelete(ctx context.Context, req *v1.TypeDeleteReq) (res *v1.TypeDeleteRes, err error) {
	if err = c.dictSvc.TypeDelete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.TypeDeleteRes{}, nil
}

// TypeStatus 启用/停用字典类型
func (c *ControllerV1) TypeStatus(ctx context.Context, req *v1.TypeStatusReq) (res *v1.TypeStatusRes, err error) {
	if err = c.dictSvc.TypeStatus(ctx, req.Id, req.Status); err != nil {
		return nil, err
	}
	return &v1.TypeStatusRes{}, nil
}
