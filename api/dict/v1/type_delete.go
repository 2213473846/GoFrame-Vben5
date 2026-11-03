package v1

import "github.com/gogf/gf/v2/frame/g"

// TypeDeleteReq 删除字典类型(软删除, 级联删除字典项)
type TypeDeleteReq struct {
	g.Meta `path:"/dict/type/delete" method:"delete" tags:"Dict" summary:"删除字典类型"`
	Id     uint64 `json:"id" v:"required|min:1#请选择字典类型"`
}

// TypeDeleteRes 删除结果
type TypeDeleteRes struct{}
