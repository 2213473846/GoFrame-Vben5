package online

import (
	"context"

	v1 "demo/api/online/v1"
	onlinesvc "demo/internal/service/online"
)

// OnlineUserList 在线用户列表
func (c *ControllerV1) OnlineUserList(ctx context.Context, req *v1.OnlineUserListReq) (res *v1.OnlineUserListRes, err error) {
	out, err := onlinesvc.List(ctx, &onlinesvc.ListInput{
		Page:     req.Page,
		PageSize: req.PageSize,
		Username: req.Username,
	})
	if err != nil {
		return nil, err
	}
	res = &v1.OnlineUserListRes{
		Total: out.Total,
		Items: make([]*v1.OnlineUserItem, 0, len(out.Items)),
	}
	for _, item := range out.Items {
		res.Items = append(res.Items, &v1.OnlineUserItem{
			UserId:        item.UserId,
			SessionId:     item.SessionId,
			Username:      item.Username,
			RealName:      item.RealName,
			Role:          item.Role,
			Ip:            item.Ip,
			Location:      item.Location,
			Browser:       item.Browser,
			Os:            item.Os,
			LoginTime:     item.LoginTime,
			Status:        item.Status,
			OfflineReason: item.OfflineReason,
			OfflineAt:     item.OfflineAt,
		})
	}
	return
}
