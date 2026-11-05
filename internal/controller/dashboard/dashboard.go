package dashboard

import (
	"context"

	v1 "demo/api/dashboard/v1"
	dashboardsvc "demo/internal/service/dashboard"
)

// ControllerV1 工作台控制器
type ControllerV1 struct {
	svc *dashboardsvc.Service
}

// NewV1 创建工作台控制器
func NewV1() *ControllerV1 {
	return &ControllerV1{svc: dashboardsvc.New()}
}

// Overview 工作台概览
func (c *ControllerV1) Overview(ctx context.Context, req *v1.OverviewReq) (res *v1.OverviewRes, err error) {
	data, err := c.svc.Overview(ctx)
	if err != nil {
		return nil, err
	}
	out := &v1.OverviewRes{
		Scope:             data.Scope,
		UserTotal:         data.UserTotal,
		OnlineCount:       data.OnlineCount,
		LoginSuccessToday: data.LoginSuccessToday,
		LoginFailedToday:  data.LoginFailedToday,
		TrendDays:         data.TrendDays,
		Trend:             make([]v1.TrendPoint, 0, len(data.Trend)),
		Recent:            make([]v1.RecentItem, 0, len(data.Recent)),
		UnreadNoticeCount: data.UnreadNoticeCount,
		RecentNotices:     make([]v1.NoticeBrief, 0, len(data.RecentNotices)),
	}
	for _, p := range data.Trend {
		out.Trend = append(out.Trend, v1.TrendPoint{Date: p.Date, Success: p.Success, Failed: p.Failed})
	}
	for _, item := range data.Recent {
		out.Recent = append(out.Recent, v1.RecentItem{
			Id: item.Id, Username: item.Username, Module: item.Module,
			Action: item.Action, Status: item.Status, Ip: item.Ip, CreatedAt: item.CreatedAt,
		})
	}
	for _, item := range data.RecentNotices {
		out.RecentNotices = append(out.RecentNotices, v1.NoticeBrief{
			Id: item.Id, Title: item.Title, Level: item.Level,
			PublishAt: item.PublishAt, IsRead: item.IsRead,
		})
	}
	return out, nil
}
