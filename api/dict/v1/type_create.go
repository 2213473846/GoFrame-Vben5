package v1

import "github.com/gogf/gf/v2/frame/g"

// TypeCreateReq 新增字典类型(仅超级管理员)
type TypeCreateReq struct {
	g.Meta `path:"/dict/type/create" method:"post" tags:"Dict" summary:"新增字典类型"`
	Type   string `json:"type" v:"required|length:2,64#请输入字典类型编码|字典类型编码长度为2-64位"`
	Name   string `json:"name" v:"required|length:1,64#请输入字典类型名称|字典类型名称长度为1-64位"`
	Status int    `json:"status" d:"1" v:"in:0,1#状态值非法"`
	Remark string `json:"remark"`
}

// TypeCreateRes 新增结果
type TypeCreateRes struct {
	Id uint64 `json:"id"`
}
