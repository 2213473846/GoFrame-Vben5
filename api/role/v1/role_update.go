package v1

import "github.com/gogf/gf/v2/frame/g"

// UpdateReq 修改角色(仅超级管理员)
type UpdateReq struct {
	g.Meta `path:"/role/update" method:"put" tags:"Role" summary:"修改角色"`
	Id     uint64 `json:"id" v:"required#请指定角色ID"`
	Code   string `json:"code" v:"required|length:2,64#请输入角色编码|角色编码长度为2-64位"`
	Name   string `json:"name" v:"required|length:1,64#请输入角色名称|角色名称长度为1-64位"`
	Remark string `json:"remark"`
	Status int    `json:"status" d:"1" v:"in:0,1#状态值非法"`
}

// UpdateRes 修改结果
type UpdateRes struct{}
