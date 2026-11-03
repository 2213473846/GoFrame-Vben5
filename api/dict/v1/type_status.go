package v1

import "github.com/gogf/gf/v2/frame/g"

// TypeStatusReq 启用/停用字典类型
type TypeStatusReq struct {
	g.Meta `path:"/dict/type/status" method:"put" tags:"Dict" summary:"启用/停用字典类型"`
	Id     uint64 `json:"id" v:"required|min:1#请选择字典类型"`
	Status int    `json:"status" v:"in:0,1#状态值非法"`
}

// TypeStatusRes 操作结果
type TypeStatusRes struct{}
