package v1

import "github.com/gogf/gf/v2/frame/g"

// CaptchaReq 获取图形验证码(公开接口)
type CaptchaReq struct {
	g.Meta `path:"/auth/captcha" method:"get" tags:"Auth" summary:"获取图形验证码"`
}

// CaptchaRes 验证码图片与ID
type CaptchaRes struct {
	CaptchaId     string `json:"captchaId"`
	Image         string `json:"image"` // data:image/svg+xml;base64,...
	ExpireSeconds int    `json:"expireSeconds"`
}
