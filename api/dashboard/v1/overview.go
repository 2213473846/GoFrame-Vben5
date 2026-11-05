package v1

import "github.com/gogf/gf/v2/frame/g"

// OverviewReq 工作台概览(仅超管/管理员可见)
type OverviewReq struct {
	g.Meta `path:"/dashboard/overview" method:"get" tags:"Dashboard" summary:"工作台概览"`
}

// TrendPoint 单日登录趋势
type TrendPoint struct {
	Date    string `json:"date"`
	Success int    `json:"success"`
	Failed  int    `json:"failed"`
}

// RecentItem 最近操作动态
type RecentItem struct {
	Id        uint64 `json:"id"`
	Username  string `json:"username"`
	Module    string `json:"module"`
	Action    string `json:"action"`
	Status    int    `json:"status"`
	Ip        string `json:"ip"`
	CreatedAt string `json:"createdAt"`
}

// NoticeBrief 工作台最近公告
type NoticeBrief struct {
	Id        uint64 `json:"id"`
	Title     string `json:"title"`
	Level     string `json:"level"`
	PublishAt string `json:"publishAt"`
	IsRead    bool   `json:"isRead"`
}

// OverviewRes 工作台概览数据
type OverviewRes struct {
	Scope             string        `json:"scope"`
	UserTotal         int           `json:"userTotal"`
	OnlineCount       int           `json:"onlineCount"`
	LoginSuccessToday int           `json:"loginSuccessToday"`
	LoginFailedToday  int           `json:"loginFailedToday"`
	TrendDays         int           `json:"trendDays"`
	Trend             []TrendPoint  `json:"trend"`
	Recent            []RecentItem  `json:"recent"`
	UnreadNoticeCount int           `json:"unreadNoticeCount"`
	RecentNotices     []NoticeBrief `json:"recentNotices"`
}
