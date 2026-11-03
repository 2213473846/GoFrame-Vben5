package v1

import "github.com/gogf/gf/v2/frame/g"

// TypeUpdateReq 修改字典类型(编码不可变更)
type TypeUpdateReq struct {
	g.Meta `path:"/dict/type/update" method:"put" tags:"Dict" summary:"修改字典类型"`
	Id     uint64 `json:"id" v:"required|min:1#请选择字典类型"`
	Name   string `json:"name" v:"required|length:1,64#请输入字典类型名称|字典类型名称长度为1-64位"`
	Status int    `json:"status" d:"1" v:"in:0,1#状态值非法"`
	Remark string `json:"remark"`
}

// TypeUpdateRes 修改结果
type TypeUpdateRes struct{}
