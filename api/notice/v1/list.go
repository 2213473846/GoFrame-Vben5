package v1

import "github.com/gogf/gf/v2/frame/g"

// ListReq 公告列表(超管/管理员)
type ListReq struct {
	g.Meta    `path:"/notice/list" method:"get" tags:"Notice" summary:"公告列表"`
	Keyword   string `p:"keyword"`       // 标题模糊查询
	Type      string `p:"type"`          // 字典 notice_type
	Level     string `p:"level"`         // 字典 notice_level
	Status    int    `p:"status" d:"-1"` // -1=全部 0=草稿 1=已发布 2=已撤回
	StartTime string `p:"startTime"`     // publish_at 起
	EndTime   string `p:"endTime"`       // publish_at 止
	Page      int    `p:"page" d:"1"`
	PageSize  int    `p:"pageSize" d:"10"`
}

// NoticeItem 管理端公告条目
type NoticeItem struct {
	Id          uint64 `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	Level       string `json:"level"`
	Status      int    `json:"status"`
	TargetScope int    `json:"targetScope"`
	TargetCount int    `json:"targetCount"` // 定向用户数, 全员时为 0
	PublisherId uint64 `json:"publisherId"`
	PublishAt   string `json:"publishAt"`
	ExpireAt    string `json:"expireAt"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// ListRes 公告列表结果
type ListRes struct {
	Items []*NoticeItem `json:"items"`
	Total int           `json:"total"`
}

// DetailReq 公告详情
type DetailReq struct {
	g.Meta `path:"/notice/detail" method:"get" tags:"Notice" summary:"公告详情"`
	Id     uint64 `p:"id" v:"required#请指定公告"`
}

// DetailRes 公告详情结果
type DetailRes struct {
	Item          *NoticeItem `json:"item"`
	TargetUserIds []uint64    `json:"targetUserIds"` // 定向用户ID, 供编辑回填
}
