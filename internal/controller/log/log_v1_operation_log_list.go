package log

import (
	"context"

	v1 "demo/api/log/v1"
	logsvc "demo/internal/service/log"

	"github.com/gogf/gf/v2/util/gconv"
)

// OperationLogList 操作日志列表
func (c *ControllerV1) OperationLogList(ctx context.Context, req *v1.OperationLogListReq) (res *v1.OperationLogListRes, err error) {
	out, err := logsvc.New().OperationList(ctx, &logsvc.OperationListInput{
		Page:     req.Page,
		PageSize: req.PageSize,
		Username: req.Username,
		Status:   req.Status,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.OperationLogListRes{
		Total: out.Total,
		Items: make([]*v1.OperationLogItem, 0, len(out.Items)),
	}
	for _, item := range out.Items {
		res.Items = append(res.Items, &v1.OperationLogItem{
			Id:        gconv.Int64(item.Id),
			Username:  item.Username,
			Module:    item.Module,
			Action:    item.Action,
			Method:    item.Method,
			Path:      item.Path,
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
