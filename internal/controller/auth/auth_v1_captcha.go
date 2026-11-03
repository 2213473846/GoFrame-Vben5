package auth

import (
	"context"

	"demo/api/auth/v1"
	secsvc "demo/internal/service/security"
)

// Captcha 获取图形验证码
func (c *ControllerV1) Captcha(ctx context.Context, req *v1.CaptchaReq) (res *v1.CaptchaRes, err error) {
	cap, err := secsvc.New().NewCaptcha(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.CaptchaRes{
		CaptchaId:     cap.Id,
		Image:         cap.Image,
		ExpireSeconds: cap.ExpireSeconds,
	}, nil
}
