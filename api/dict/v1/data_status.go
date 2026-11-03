package v1

import "github.com/gogf/gf/v2/frame/g"

// DataStatusReq 启用/停用字典项
type DataStatusReq struct {
	g.Meta `path:"/dict/data/status" method:"put" tags:"Dict" summary:"启用/停用字典项"`
	Id     uint64 `json:"id" v:"required|min:1#请选择字典项"`
	Status int    `json:"status" v:"in:0,1#状态值非法"`
}

// DataStatusRes 操作结果
type DataStatusRes struct{}
