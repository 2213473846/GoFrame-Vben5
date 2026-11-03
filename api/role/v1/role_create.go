package v1

import "github.com/gogf/gf/v2/frame/g"

// CreateReq 新增角色(仅超级管理员)
type CreateReq struct {
	g.Meta `path:"/role/create" method:"post" tags:"Role" summary:"新增角色"`
	Code   string `json:"code" v:"required|length:2,64#请输入角色编码|角色编码长度为2-64位"`
	Name   string `json:"name" v:"required|length:1,64#请输入角色名称|角色名称长度为1-64位"`
	Remark string `json:"remark"`
	Status int    `json:"status" d:"1" v:"in:0,1#状态值非法"`
}

// CreateRes 新增结果
type CreateRes struct {
	Id uint64 `json:"id"`
}
