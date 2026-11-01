package v1

import "github.com/gogf/gf/v2/frame/g"

// LoginLogListReq 登录日志列表
type LoginLogListReq struct {
	g.Meta `path:"/loginlog/list" method:"get" tags:"LoginLog" summary:"登录日志列表"`
	Page     int    `p:"page" d:"1"`
	PageSize int    `p:"pageSize" d:"10"`
	Username string `p:"username"` // 用户名模糊查询
	Status   int    `p:"status" d:"-1"` // -1=全部 1=成功 0=失败
}

// LoginLogItem 登录日志条目
type LoginLogItem struct {
	Id        int64  `json:"id"`
	Username  string `json:"username"`
	Ip        string `json:"ip"`
	Location  string `json:"location"`
	Browser   string `json:"browser"`
	Os        string `json:"os"`
	Status    int    `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

// LoginLogListRes 登录日志列表结果
type LoginLogListRes struct {
	Items []*LoginLogItem `json:"items"`
	Total int             `json:"total"`
}
