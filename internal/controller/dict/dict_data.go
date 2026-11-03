package dict

import (
	"context"
	"strings"

	"demo/api/dict/v1"
	dictsvc "demo/internal/service/dict"
)

// DataList 字典项列表
func (c *ControllerV1) DataList(ctx context.Context, req *v1.DataListReq) (res *v1.DataListRes, err error) {
	items, total, err := c.dictSvc.DataList(ctx, dictsvc.DataListInput{
		DictType: req.DictType,
		Keyword:  req.Keyword,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.DataListRes{
		Items: make([]v1.DictDataItem, 0, len(items)),
		Total: total,
	}
	for _, d := range items {
		res.Items = append(res.Items, v1.DictDataItem{
			Id:       d.Id,
			DictType: d.DictType,
			Label:    d.Label,
			Value:    d.Value,
			Sort:     d.Sort,
			Status:   d.Status,
			TagColor: d.TagColor,
			Remark:   d.Remark,
		})
	}
	return
}

// DataCreate 新增字典项
func (c *ControllerV1) DataCreate(ctx context.Context, req *v1.DataCreateReq) (res *v1.DataCreateRes, err error) {
	id, err := c.dictSvc.DataCreate(ctx, dictsvc.DataCreateInput{
		DictType: req.DictType,
		Label:    req.Label,
		Value:    req.Value,
		Sort:     req.Sort,
		Status:   req.Status,
		TagColor: req.TagColor,
		Remark:   req.Remark,
	})
	if err != nil {
		return nil, err
	}
	return &v1.DataCreateRes{Id: id}, nil
}

// DataUpdate 修改字典项
func (c *ControllerV1) DataUpdate(ctx context.Context, req *v1.DataUpdateReq) (res *v1.DataUpdateRes, err error) {
	if err = c.dictSvc.DataUpdate(ctx, dictsvc.DataUpdateInput{
		Id:       req.Id,
		DictType: req.DictType,
		Label:    req.Label,
		Value:    req.Value,
		Sort:     req.Sort,
		Status:   req.Status,
		TagColor: req.TagColor,
		Remark:   req.Remark,
	}); err != nil {
		return nil, err
	}
	return &v1.DataUpdateRes{}, nil
}

// DataDelete 删除字典项
func (c *ControllerV1) DataDelete(ctx context.Context, req *v1.DataDeleteReq) (res *v1.DataDeleteRes, err error) {
	if err = c.dictSvc.DataDelete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DataDeleteRes{}, nil
}

// DataStatus 启用/停用字典项
func (c *ControllerV1) DataStatus(ctx context.Context, req *v1.DataStatusReq) (res *v1.DataStatusRes, err error) {
	if err = c.dictSvc.DataStatus(ctx, req.Id, req.Status); err != nil {
		return nil, err
	}
	return &v1.DataStatusRes{}, nil
}

// DataBatch 按 type 批量取值
func (c *ControllerV1) DataBatch(ctx context.Context, req *v1.DataBatchReq) (res *v1.DataBatchRes, err error) {
	types := splitTypes(req.Types)
	data, err := c.dictSvc.Options(ctx, types)
	if err != nil {
		return nil, err
	}
	res = &v1.DataBatchRes{Data: make(map[string][]v1.DictDataItem, len(data))}
	for t, items := range data {
		list := make([]v1.DictDataItem, 0, len(items))
		for _, d := range items {
			list = append(list, v1.DictDataItem{
				Id:       d.Id,
				DictType: d.DictType,
				Label:    d.Label,
				Value:    d.Value,
				Sort:     d.Sort,
				Status:   d.Status,
				TagColor: d.TagColor,
				Remark:   d.Remark,
			})
		}
		res.Data[t] = list
	}
	return
}

// splitTypes 支持逗号分隔的批量类型参数
func splitTypes(raw string) []string {
	parts := strings.Split(raw, ",")
	types := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			types = append(types, p)
		}
	}
	return types
}
