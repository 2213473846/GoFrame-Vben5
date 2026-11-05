package job

import (
	"context"

	v1 "demo/api/job/v1"
	jobsvc "demo/internal/service/job"

	"github.com/gogf/gf/v2/os/gtime"
)

// List 定时任务列表
func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	out, err := jobsvc.New().List(ctx, jobsvc.ListInput{
		Page:     req.Page,
		PageSize: req.PageSize,
		Name:     req.Name,
		Status:   req.Status,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.ListRes{Total: out.Total, Items: make([]*v1.JobItem, 0, len(out.Items))}
	for _, item := range out.Items {
		res.Items = append(res.Items, &v1.JobItem{
			Id:                item.Id,
			Name:              item.Name,
			JobGroup:          item.JobGroup,
			InvokeTarget:      item.InvokeTarget,
			CronExpression:    item.CronExpression,
			Status:            item.Status,
			ConcurrencyPolicy: item.ConcurrencyPolicy,
			TimeoutSeconds:    item.TimeoutSeconds,
			Remark:            item.Remark,
			LastRunAt:         formatJobTime(item.LastRunAt),
			LastResult:        item.LastResult,
			NextRunAt:         formatJobTime(item.NextRunAt),
			CreatedAt:         formatJobTime(item.CreatedAt),
			UpdatedAt:         formatJobTime(item.UpdatedAt),
		})
	}
	return res, nil
}

// Create 新增定时任务
func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	id, err := jobsvc.New().Create(ctx, jobsvc.CreateInput{
		Name:              req.Name,
		JobGroup:          req.JobGroup,
		InvokeTarget:      req.InvokeTarget,
		CronExpression:    req.CronExpression,
		Status:            req.Status,
		ConcurrencyPolicy: req.ConcurrencyPolicy,
		TimeoutSeconds:    req.TimeoutSeconds,
		Remark:            req.Remark,
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateRes{Id: id}, nil
}

// Update 修改定时任务
func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	err = jobsvc.New().Update(ctx, jobsvc.UpdateInput{
		Id: req.Id,
		CreateInput: jobsvc.CreateInput{
			Name:              req.Name,
			JobGroup:          req.JobGroup,
			InvokeTarget:      req.InvokeTarget,
			CronExpression:    req.CronExpression,
			Status:            req.Status,
			ConcurrencyPolicy: req.ConcurrencyPolicy,
			TimeoutSeconds:    req.TimeoutSeconds,
			Remark:            req.Remark,
		},
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateRes{}, nil
}

// Delete 删除定时任务
func (c *ControllerV1) Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error) {
	if err = jobsvc.New().Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteRes{}, nil
}

// ChangeStatus 启停定时任务
func (c *ControllerV1) ChangeStatus(ctx context.Context, req *v1.StatusReq) (res *v1.StatusRes, err error) {
	if err = jobsvc.New().ChangeStatus(ctx, req.Id, req.Status); err != nil {
		return nil, err
	}
	return &v1.StatusRes{}, nil
}

// Run 手动执行定时任务(异步受理)
func (c *ControllerV1) Run(ctx context.Context, req *v1.RunReq) (res *v1.RunRes, err error) {
	if err = jobsvc.New().Run(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.RunRes{Accepted: true}, nil
}

// Logs 定时任务执行日志
func (c *ControllerV1) Logs(ctx context.Context, req *v1.LogsReq) (res *v1.LogsRes, err error) {
	out, err := jobsvc.New().Logs(ctx, jobsvc.LogListInput{
		Page:     req.Page,
		PageSize: req.PageSize,
		JobId:    req.JobId,
		Status:   req.Status,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.LogsRes{Total: out.Total, Items: make([]*v1.JobLogItem, 0, len(out.Items))}
	for _, item := range out.Items {
		res.Items = append(res.Items, &v1.JobLogItem{
			Id:          item.Id,
			JobId:       item.JobId,
			JobName:     item.JobName,
			TriggerType: item.TriggerType,
			StartedAt:   formatJobTime(item.StartedAt),
			FinishedAt:  formatJobTime(item.FinishedAt),
			DurationMs:  item.DurationMs,
			Status:      item.Status,
			Message:     item.Message,
			CreatedAt:   formatJobTime(item.CreatedAt),
		})
	}
	return res, nil
}

// formatJobTime 时间格式化
func formatJobTime(t *gtime.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("Y-m-d H:i:s")
}
