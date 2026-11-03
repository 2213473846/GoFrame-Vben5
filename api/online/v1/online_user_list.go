package v1

import "github.com/gogf/gf/v2/frame/g"

// OnlineUserListReq 在线用户列表
type OnlineUserListReq struct {
	g.Meta   `path:"/online/list" method:"get" tags:"OnlineUser" summary:"在线用户列表"`
	Page     int    `p:"page" d:"1"`
	PageSize int    `p:"pageSize" d:"10"`
	Username string `p:"username"` // 用户名模糊查询
}

// OnlineUserItem 在线用户条目(一台设备/一个浏览器的登录会话)
type OnlineUserItem struct {
	UserId        uint64 `json:"userId"`
	SessionId     string `json:"sessionId"`
	Username      string `json:"username"`
	RealName      string `json:"realName"`
	Role          string `json:"role"`
	Ip            string `json:"ip"`
	Location      string `json:"location"`
	Browser       string `json:"browser"`
	Os            string `json:"os"`
	LoginTime     string `json:"loginTime"`
	Status        int    `json:"status"` // 1=在线 0=已下线
	OfflineReason string `json:"offlineReason"`
	OfflineAt     string `json:"offlineAt"`
}

// OnlineUserListRes 在线用户列表结果
type OnlineUserListRes struct {
	Items []*OnlineUserItem `json:"items"`
	Total int               `json:"total"`
}
