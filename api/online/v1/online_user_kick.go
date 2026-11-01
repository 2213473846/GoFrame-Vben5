package v1

import "github.com/gogf/gf/v2/frame/g"

// OnlineUserKickReq 强退用户(删除其全部登录会话)
type OnlineUserKickReq struct {
	g.Meta `path:"/online/kick" method:"post" tags:"OnlineUser" summary:"强退用户"`
	UserId uint64 `p:"userId" v:"required#请指定要强退的用户"`
}

// OnlineUserKickRes 强退用户结果
type OnlineUserKickRes struct{}
