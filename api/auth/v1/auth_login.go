package v1

import "github.com/gogf/gf/v2/frame/g"

// LoginReq 登录
type LoginReq struct {
	g.Meta   `path:"/auth/login" method:"post" tags:"Auth" summary:"登录"`
	Username string `json:"username" v:"required#请输入用户名"`
	Password string `json:"password" v:"required#请输入密码"`
}

// LoginRes 登录结果(前端仅使用 accessToken, 其余字段冗余返回)
type LoginRes struct {
	AccessToken string   `json:"accessToken"`
	UserId      string   `json:"userId"`
	Username    string   `json:"username"`
	RealName    string   `json:"realName"`
	Avatar      string   `json:"avatar"`
	Roles       []string `json:"roles"`
}
