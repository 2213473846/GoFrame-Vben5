package v1

import "github.com/gogf/gf/v2/frame/g"

// DataBatchReq 按 type 批量取值(任意登录用户可用, 仅返回启用项)
type DataBatchReq struct {
	g.Meta `path:"/dict/data/batch" method:"get" tags:"Dict" summary:"按类型批量获取字典项"`
	Types  string `json:"types" v:"required#请指定字典类型"`
}

// DataBatchRes 批量取值结果: key 为字典类型编码
type DataBatchRes struct {
	Data map[string][]DictDataItem `json:"data"`
}
