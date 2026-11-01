package v1

import "github.com/gogf/gf/v2/frame/g"

// ListReq 用户/管理员列表(数据权限: 超管全部, 管理员仅子孙)
type ListReq struct {
	g.Meta   `path:"/user/list" method:"get" tags:"User" summary:"用户列表"`
	Page     int    `json:"page" d:"1" v:"min:1#页码最小为1"`
	PageSize int    `json:"pageSize" d:"10" v:"max:100#每页最多100条"`
	Username string `json:"username"` // 用户名模糊查询
	RealName string `json:"realName"` // 昵称模糊查询
	Role     string `json:"role"`     // 角色过滤: admin/user
	Status   *int   `json:"status"`   // 状态过滤: 1/0
}

// ListItem 列表项
type ListItem struct {
	Id             uint64 `json:"id"`
	Pid            uint64 `json:"pid"`
	Username       string `json:"username"`
	RealName       string `json:"realName"`
	Avatar         string `json:"avatar"`
	Role           string `json:"role"`
	Status         int    `json:"status"`
	Remark         string `json:"remark"`
	CreatedAt      string `json:"createdAt"`
	ParentUsername string `json:"parentUsername"` // 上级用户名, 顶级为空
	ParentRealName string `json:"parentRealName"` // 上级昵称
}

// ListRes 分页结果
type ListRes struct {
	Items []ListItem `json:"items"`
	Total int        `json:"total"`
}
