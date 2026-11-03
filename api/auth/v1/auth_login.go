package v1

import "github.com/gogf/gf/v2/frame/g"

// LoginReq 登录(验证码字段可选, 仅在失败次数达到阈值后由后端要求)
type LoginReq struct {
	g.Meta      `path:"/auth/login" method:"post" tags:"Auth" summary:"登录"`
	Username    string `json:"username" v:"required#请输入用户名"`
	Password    string `json:"password" v:"required#请输入密码"`
	CaptchaId   string `json:"captchaId"`   // 图形验证码ID(GET /auth/captcha 获取)
	CaptchaCode string `json:"captchaCode"` // 图形验证码答案
}

// LoginRes 登录结果(前端仅使用 accessToken, 其余字段冗余返回)
type LoginRes struct {
	AccessToken        string   `json:"accessToken"`
	UserId             string   `json:"userId"`
	Username           string   `json:"username"`
	RealName           string   `json:"realName"`
	Avatar             string   `json:"avatar"`
	Roles              []string `json:"roles"`
	MustChangePassword bool     `json:"mustChangePassword"` // true 时前端须先引导修改密码
}
