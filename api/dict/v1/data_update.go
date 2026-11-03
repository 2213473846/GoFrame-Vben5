package v1

import "github.com/gogf/gf/v2/frame/g"

// DataUpdateReq 修改字典项
type DataUpdateReq struct {
	g.Meta   `path:"/dict/data/update" method:"put" tags:"Dict" summary:"修改字典项"`
	Id       uint64 `json:"id" v:"required|min:1#请选择字典项"`
	DictType string `json:"dictType"`
	Label    string `json:"label" v:"required|length:1,64#请输入显示名称|显示名称长度为1-64位"`
	Value    string `json:"value" v:"required|length:1,64#请输入字典值|字典值长度为1-64位"`
	Sort     int    `json:"sort" d:"0"`
	Status   int    `json:"status" d:"1" v:"in:0,1#状态值非法"`
	TagColor string `json:"tagColor" v:"in:,success,warning,danger,info#标签颜色非法"`
	Remark   string `json:"remark"`
}

// DataUpdateRes 修改结果
type DataUpdateRes struct{}
