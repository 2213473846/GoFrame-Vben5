package log

import (
	"context"

	v1 "demo/api/log/v1"
	logsvc "demo/internal/service/log"

	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/os/gtime"
)

// LoginLogList 登录日志列表
func (c *ControllerV1) LoginLogList(ctx context.Context, req *v1.LoginLogListReq) (res *v1.LoginLogListRes, err error) {
	out, err := logsvc.New().LoginList(ctx, &logsvc.LoginListInput{
		Page:     req.Page,
		PageSize: req.PageSize,
		Username: req.Username,
		Status:   req.Status,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.LoginLogListRes{
		Total: out.Total,
		Items: make([]*v1.LoginLogItem, 0, len(out.Items)),
	}
	for _, item := range out.Items {
		res.Items = append(res.Items, &v1.LoginLogItem{
			Id:        gconv.Int64(item.Id),
			Username:  item.Username,
			Ip:        item.Ip,
			Location:  item.Location,
			Browser:   item.Browser,
			Os:        item.Os,
			Status:    item.Status,
			Message:   item.Message,
			CreatedAt: formatLogTime(item.CreatedAt),
		})
	}
	return
}

// formatLogTime 时间格式化
func formatLogTime(t *gtime.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("Y-m-d H:i:s")
}
