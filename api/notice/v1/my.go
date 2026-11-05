package v1

import "github.com/gogf/gf/v2/frame/g"

// MyListReq 我的通知列表(任意已登录用户)
type MyListReq struct {
	g.Meta   `path:"/notice/my" method:"get" tags:"Notice" summary:"我的通知列表"`
	Page     int `p:"page" d:"1"`
	PageSize int `p:"pageSize" d:"10"`
}

// MyNoticeItem 用户端公告条目
type MyNoticeItem struct {
	Id        uint64 `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	Level     string `json:"level"`
	PublishAt string `json:"publishAt"`
	CreatedAt string `json:"createdAt"`
	IsRead    bool   `json:"isRead"`
}

// MyListRes 我的通知列表结果
type MyListRes struct {
	Items []*MyNoticeItem `json:"items"`
	Total int             `json:"total"`
}

// UnreadCountReq 未读公告数
type UnreadCountReq struct {
	g.Meta `path:"/notice/unread-count" method:"get" tags:"Notice" summary:"未读公告数"`
}

// UnreadCountRes 未读公告数结果
type UnreadCountRes struct {
	Count       int `json:"count"`
	PollSeconds int `json:"pollSeconds"`
}

// ReadReq 标记已读: id 指定单条, all=true 全部已读
type ReadReq struct {
	g.Meta `path:"/notice/read" method:"post" tags:"Notice" summary:"标记公告已读"`
	Id     uint64 `json:"id"`
	All    bool   `json:"all"`
}

// ReadRes 标记结果
type ReadRes struct{}
