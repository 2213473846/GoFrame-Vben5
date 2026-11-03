package v1

import "github.com/gogf/gf/v2/frame/g"

// TypeListReq 字典类型列表(超管/管理员可读)
type TypeListReq struct {
	g.Meta  `path:"/dict/type/list" method:"get" tags:"Dict" summary:"字典类型列表"`
	Keyword string `json:"keyword"`
	Status  int    `json:"status" d:"-1"`
}

// TypeItem 字典类型
type TypeItem struct {
	Id        uint64 `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
	CreatedAt string `json:"createdAt"`
}

// TypeListRes 字典类型列表
type TypeListRes struct {
	Items []TypeItem `json:"items"`
}
