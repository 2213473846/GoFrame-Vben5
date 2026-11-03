package sysconfig

import (
	"context"
	"strings"

	v1 "demo/api/sysconfig/v1"
	"demo/internal/model/entity"
	sysconfigsvc "demo/internal/service/sysconfig"
)

// ControllerV1 系统参数配置控制器
type ControllerV1 struct {
	svc *sysconfigsvc.Service
}

// NewV1 创建系统参数配置控制器
func NewV1() *ControllerV1 {
	return &ControllerV1{svc: sysconfigsvc.New()}
}

// List 参数列表
func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	list, err := c.svc.List(ctx, sysconfigsvc.ListInput{Keyword: req.Keyword, Status: req.Status})
	if err != nil {
		return nil, err
	}
	items := make([]v1.ConfigItem, 0, len(list))
	for _, row := range list {
		items = append(items, toConfigItem(row))
	}
	return &v1.ListRes{Items: items}, nil
}

// Get 按 key 读取
func (c *ControllerV1) Get(ctx context.Context, req *v1.GetReq) (res *v1.GetRes, err error) {
	value, err := c.svc.Get(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	return &v1.GetRes{ConfigValue: value}, nil
}

// Batch 批量读取
func (c *ControllerV1) Batch(ctx context.Context, req *v1.BatchReq) (res *v1.BatchRes, err error) {
	values, err := c.svc.Batch(ctx, splitKeys(req.Keys))
	if err != nil {
		return nil, err
	}
	return &v1.BatchRes{Values: values}, nil
}

// Update 修改参数值
func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	if err = c.svc.Update(ctx, req.Key, req.Value); err != nil {
		return nil, err
	}
	return &v1.UpdateRes{}, nil
}

// Reset 重置为默认值
func (c *ControllerV1) Reset(ctx context.Context, req *v1.ResetReq) (res *v1.ResetRes, err error) {
	value, err := c.svc.Reset(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	return &v1.ResetRes{ConfigValue: value}, nil
}

func toConfigItem(row *entity.SysConfig) v1.ConfigItem {
	item := v1.ConfigItem{
		Id:          row.Id,
		ConfigKey:   row.ConfigKey,
		ConfigValue: row.ConfigValue,
		Name:        row.Name,
		Remark:      row.Remark,
		Status:      row.Status,
		Builtin:     row.Builtin,
	}
	if row.CreatedAt != nil {
		item.CreatedAt = row.CreatedAt.Format("Y-m-d H:i:s")
	}
	if row.UpdatedAt != nil {
		item.UpdatedAt = row.UpdatedAt.Format("Y-m-d H:i:s")
	}
	return item
}

func splitKeys(raw string) []string {
	parts := strings.Split(raw, ",")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			keys = append(keys, p)
		}
	}
	return keys
}
