package v1

import "github.com/gogf/gf/v2/frame/g"

// DictDataItem 字典项(管理端/批量取值共用)
type DictDataItem struct {
	Id       uint64 `json:"id"`
	DictType string `json:"dictType"`
	Label    string `json:"label"`
	Value    string `json:"value"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
	TagColor string `json:"tagColor"`
	Remark   string `json:"remark"`
}

// DataListReq 字典项列表(管理端, 含停用项)
type DataListReq struct {
	g.Meta   `path:"/dict/data/list" method:"get" tags:"Dict" summary:"字典项列表"`
	DictType string `json:"dictType" v:"required#请选择字典类型"`
	Keyword  string `json:"keyword"`
	Status   int    `json:"status" d:"-1"`
	Page     int    `json:"page" d:"1"`
	PageSize int    `json:"pageSize" d:"20"`
}

// DataListRes 字典项列表
type DataListRes struct {
	Items []DictDataItem `json:"items"`
	Total int            `json:"total"`
}
