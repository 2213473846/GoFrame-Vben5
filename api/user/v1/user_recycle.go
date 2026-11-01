package v1

import "github.com/gogf/gf/v2/frame/g"

// RecycleReq 回收站列表(已软删用户)
// 数据权限: 超管全部, 管理员仅其子孙树中的已删记录
type RecycleReq struct {
	g.Meta   `path:"/user/recycle" method:"get" tags:"User" summary:"回收站列表"`
	Page     int    `json:"page" d:"1" v:"min:1#页码最小为1"`
	PageSize int    `json:"pageSize" d:"10" v:"max:100#每页最多100条"`
	Username string `json:"username"` // 用户名模糊查询
}

// RecycleItem 回收站条目
type RecycleItem struct {
	Id        uint64 `json:"id"`
	Pid       uint64 `json:"pid"`
	Username  string `json:"username"`
	RealName  string `json:"realName"`
	Role      string `json:"role"`
	DeletedAt string `json:"deletedAt"`
	Operator  string `json:"operator"` // 删除操作人用户名
}

// RecycleRes 分页结果
type RecycleRes struct {
	Items []RecycleItem `json:"items"`
	Total int           `json:"total"`
}
