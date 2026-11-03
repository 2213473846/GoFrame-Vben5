package v1

import "github.com/gogf/gf/v2/frame/g"

// ChangePasswordReq 修改当前登录用户密码(首次登录强制改密也走这里)
type ChangePasswordReq struct {
	g.Meta      `path:"/auth/change-password" method:"post" tags:"Auth" summary:"修改密码"`
	OldPassword string `json:"oldPassword" v:"required#请输入原密码"`
	NewPassword string `json:"newPassword" v:"required#请输入新密码"`
}

// ChangePasswordRes 修改结果
type ChangePasswordRes struct{}
