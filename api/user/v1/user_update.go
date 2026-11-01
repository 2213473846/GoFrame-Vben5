package v1

import "github.com/gogf/gf/v2/frame/g"

// UpdateReq 更新用户(字段为空则不修改)
type UpdateReq struct {
	g.Meta   `path:"/user/update" method:"put" tags:"User" summary:"更新管理员/用户"`
	Id       uint64 `json:"id" v:"required#请指定用户ID"`
	Username string `json:"username" v:"length:3,32#用户名长度为3-32位"` // 非空则修改用户名(需唯一)
	RealName string `json:"realName"`
	Avatar   string `json:"avatar"`
	Password string `json:"password"` // 非空则重置密码, 长度6-32
	Status   *int   `json:"status"`   // 1=启用 0=禁用
	Role     string `json:"role"`     // 仅超管可改: admin/user
	Remark   string `json:"remark"`
}

// UpdateRes 更新结果
type UpdateRes struct{}
