package v1

import "github.com/gogf/gf/v2/frame/g"

// CreateReq 新增公告(超管/管理员)
type CreateReq struct {
	g.Meta        `path:"/notice/create" method:"post" tags:"Notice" summary:"新增公告"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Type          string   `json:"type"`
	Level         string   `json:"level"`
	TargetScope   int      `json:"targetScope" d:"1"` // 1=全员 2=指定用户
	TargetUserIds []uint64 `json:"targetUserIds"`
	PublishAt     string   `json:"publishAt"`
	ExpireAt      string   `json:"expireAt"`
}

// CreateRes 新增结果
type CreateRes struct {
	Id uint64 `json:"id"`
}

// UpdateReq 修改公告(超管/管理员, 管理员仅限自己创建)
type UpdateReq struct {
	g.Meta        `path:"/notice/update" method:"put" tags:"Notice" summary:"修改公告"`
	Id            uint64   `json:"id" v:"required#请指定公告"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Type          string   `json:"type"`
	Level         string   `json:"level"`
	TargetScope   int      `json:"targetScope" d:"1"`
	TargetUserIds []uint64 `json:"targetUserIds"`
	PublishAt     string   `json:"publishAt"`
	ExpireAt      string   `json:"expireAt"`
}

// UpdateRes 修改结果
type UpdateRes struct{}

// DeleteReq 删除公告
type DeleteReq struct {
	g.Meta `path:"/notice/delete" method:"delete" tags:"Notice" summary:"删除公告"`
	Id     uint64 `json:"id" v:"required#请指定公告"`
}

// DeleteRes 删除结果
type DeleteRes struct{}

// PublishReq 发布公告
type PublishReq struct {
	g.Meta    `path:"/notice/publish" method:"put" tags:"Notice" summary:"发布公告"`
	Id        uint64 `json:"id" v:"required#请指定公告"`
	PublishAt string `json:"publishAt"` // 为空取当前时间
}

// PublishRes 发布结果
type PublishRes struct{}

// RevokeReq 撤回公告
type RevokeReq struct {
	g.Meta `path:"/notice/revoke" method:"put" tags:"Notice" summary:"撤回公告"`
	Id     uint64 `json:"id" v:"required#请指定公告"`
}

// RevokeRes 撤回结果
type RevokeRes struct{}
