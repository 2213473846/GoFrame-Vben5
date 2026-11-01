package v1

import "github.com/gogf/gf/v2/frame/g"

// InfoReq 当前登录用户信息
type InfoReq struct {
	g.Meta `path:"/user/info" method:"get" tags:"User" summary:"获取当前用户信息"`
}

// InfoRes 与前端 UserInfo 类型对齐
type InfoRes struct {
	UserId   string   `json:"userId"`
	Username string   `json:"username"`
	RealName string   `json:"realName"`
	Avatar   string   `json:"avatar"`
	Desc     string   `json:"desc"`
	HomePath string   `json:"homePath"`
	Roles    []string `json:"roles"`
	Token    string   `json:"token"`
}
