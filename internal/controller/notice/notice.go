package notice

import (
	"context"

	v1 "demo/api/notice/v1"
	"demo/internal/model/entity"
	noticesvc "demo/internal/service/notice"

	"github.com/gogf/gf/v2/os/gtime"
)

// ControllerV1 通知公告控制器
type ControllerV1 struct {
	svc *noticesvc.Service
}

// NewV1 创建通知公告控制器
func NewV1() *ControllerV1 {
	return &ControllerV1{svc: noticesvc.New()}
}

// List 管理端公告列表
func (c *ControllerV1) List(ctx context.Context, req *v1.ListReq) (res *v1.ListRes, err error) {
	out, err := c.svc.List(ctx, noticesvc.ListInput{
		Keyword:    req.Keyword,
		NoticeType: req.Type,
		Level:      req.Level,
		Status:     req.Status,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]*v1.NoticeItem, 0, len(out.Items))
	ids := make([]uint64, 0, len(out.Items))
	for _, row := range out.Items {
		items = append(items, toNoticeItem(row))
		if row.TargetScope == 2 {
			ids = append(ids, row.Id)
		}
	}
	counts, err := c.svc.TargetCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item.TargetCount = counts[item.Id]
	}
	return &v1.ListRes{Items: items, Total: out.Total}, nil
}

// Detail 管理端公告详情
func (c *ControllerV1) Detail(ctx context.Context, req *v1.DetailReq) (res *v1.DetailRes, err error) {
	row, err := c.svc.Detail(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	userIds, err := c.svc.TargetUserIds(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.DetailRes{Item: toNoticeItem(row), TargetUserIds: userIds}, nil
}

// Create 新增公告
func (c *ControllerV1) Create(ctx context.Context, req *v1.CreateReq) (res *v1.CreateRes, err error) {
	id, err := c.svc.Create(ctx, noticesvc.CreateInput{
		Title:         req.Title,
		Content:       req.Content,
		NoticeType:    req.Type,
		NoticeLevel:   req.Level,
		TargetScope:   req.TargetScope,
		TargetUserIds: req.TargetUserIds,
		PublishAt:     req.PublishAt,
		ExpireAt:      req.ExpireAt,
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateRes{Id: id}, nil
}

// Update 修改公告
func (c *ControllerV1) Update(ctx context.Context, req *v1.UpdateReq) (res *v1.UpdateRes, err error) {
	if err = c.svc.Update(ctx, noticesvc.UpdateInput{
		Id: req.Id,
		CreateInput: noticesvc.CreateInput{
			Title:         req.Title,
			Content:       req.Content,
			NoticeType:    req.Type,
			NoticeLevel:   req.Level,
			TargetScope:   req.TargetScope,
			TargetUserIds: req.TargetUserIds,
			PublishAt:     req.PublishAt,
			ExpireAt:      req.ExpireAt,
		},
	}); err != nil {
		return nil, err
	}
	return &v1.UpdateRes{}, nil
}

// Delete 删除公告
func (c *ControllerV1) Delete(ctx context.Context, req *v1.DeleteReq) (res *v1.DeleteRes, err error) {
	if err = c.svc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteRes{}, nil
}

// Publish 发布公告
func (c *ControllerV1) Publish(ctx context.Context, req *v1.PublishReq) (res *v1.PublishRes, err error) {
	if err = c.svc.Publish(ctx, req.Id, req.PublishAt); err != nil {
		return nil, err
	}
	return &v1.PublishRes{}, nil
}

// Revoke 撤回公告
func (c *ControllerV1) Revoke(ctx context.Context, req *v1.RevokeReq) (res *v1.RevokeRes, err error) {
	if err = c.svc.Revoke(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.RevokeRes{}, nil
}

// MyList 我的通知列表
func (c *ControllerV1) MyList(ctx context.Context, req *v1.MyListReq) (res *v1.MyListRes, err error) {
	out, err := c.svc.MyList(ctx, noticesvc.MyListInput{Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		return nil, err
	}
	items := make([]*v1.MyNoticeItem, 0, len(out.Items))
	for _, row := range out.Items {
		items = append(items, &v1.MyNoticeItem{
			Id:        row.Id,
			Title:     row.Title,
			Content:   row.Content,
			Type:      row.NoticeType,
			Level:     row.NoticeLevel,
			PublishAt: formatTime(row.PublishAt),
			CreatedAt: formatTime(row.CreatedAt),
			IsRead:    row.IsRead,
		})
	}
	return &v1.MyListRes{Items: items, Total: out.Total}, nil
}

// UnreadCount 未读公告数
func (c *ControllerV1) UnreadCount(ctx context.Context, req *v1.UnreadCountReq) (res *v1.UnreadCountRes, err error) {
	count, err := c.svc.UnreadCount(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.UnreadCountRes{Count: count, PollSeconds: noticesvc.PollSeconds(ctx)}, nil
}

// Read 标记已读
func (c *ControllerV1) Read(ctx context.Context, req *v1.ReadReq) (res *v1.ReadRes, err error) {
	if err = c.svc.Read(ctx, noticesvc.ReadInput{Id: req.Id, All: req.All}); err != nil {
		return nil, err
	}
	return &v1.ReadRes{}, nil
}

func toNoticeItem(row *entity.SysNotice) *v1.NoticeItem {
	return &v1.NoticeItem{
		Id:          row.Id,
		Title:       row.Title,
		Content:     row.Content,
		Type:        row.NoticeType,
		Level:       row.NoticeLevel,
		Status:      row.Status,
		TargetScope: row.TargetScope,
		PublisherId: row.PublisherId,
		PublishAt:   formatTime(row.PublishAt),
		ExpireAt:    formatTime(row.ExpireAt),
		CreatedAt:   formatTime(row.CreatedAt),
		UpdatedAt:   formatTime(row.UpdatedAt),
	}
}

func formatTime(t *gtime.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("Y-m-d H:i:s")
}
