package v1

import "github.com/gogf/gf/v2/frame/g"

// RefreshReq 刷新 accessToken, refreshToken 从 Cookie(jwt) 读取
type RefreshReq struct {
	g.Meta `path:"/auth/refresh" method:"post" tags:"Auth" summary:"刷新accessToken"`
}

// RefreshRes 该接口直接输出原始 token 字符串(与前端约定), 不走统一响应包装
type RefreshRes struct{}
