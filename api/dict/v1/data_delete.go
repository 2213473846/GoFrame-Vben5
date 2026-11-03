package v1

import "github.com/gogf/gf/v2/frame/g"

// DataDeleteReq 删除字典项(软删除)
type DataDeleteReq struct {
	g.Meta `path:"/dict/data/delete" method:"delete" tags:"Dict" summary:"删除字典项"`
	Id     uint64 `json:"id" v:"required|min:1#请选择字典项"`
}

// DataDeleteRes 删除结果
type DataDeleteRes struct{}
