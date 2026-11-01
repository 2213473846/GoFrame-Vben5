package v1

import "github.com/gogf/gf/v2/frame/g"

// RestoreReq 恢复回收站用户(需先恢复其上级, 用户名未被占用)
type RestoreReq struct {
	g.Meta `path:"/user/restore" method:"put" tags:"User" summary:"恢复用户"`
	Id     uint64 `json:"id" v:"required#请指定用户ID"`
}

// RestoreRes 恢复结果
type RestoreRes struct{}
