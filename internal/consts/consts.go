package consts

import (
	"github.com/gogf/gf/v2/errors/gcode"
)

// 角色
const (
	RoleSuperAdmin = "super_admin" // 超级管理员
	RoleAdmin      = "admin"       // 管理员(代理)
	RoleUser       = "user"        // 普通用户
)

// Cookie 名称
const (
	CookieRefreshToken = "jwt"
)

// Redis Key 前缀
const (
	RedisKeyAccessToken  = "vben:token:"   // + accessToken
	RedisKeyRefreshToken = "vben:refresh:" // + refreshToken
	RedisKeyOnline       = "vben:online:"  // + accessToken, 在线用户会话信息
)

// Token 有效期
const (
	AccessTTL  = 2 * 60 * 60        // accessToken: 2小时
	RefreshTTL = 7 * 24 * 60 * 60   // refreshToken: 7天
)

// 错误码(与 HTTP 状态码一致, 由响应中间件映射)
var (
	CodeUnauthorized = gcode.New(401, "Unauthorized", nil)
	CodeForbidden    = gcode.New(403, "Forbidden", nil)
	CodeNotFound     = gcode.New(404, "NotFound", nil)
	CodeBadRequest   = gcode.New(400, "BadRequest", nil)
)

// AuthWhitelist 无需登录即可访问的接口
var AuthWhitelist = map[string]bool{
	"/auth/login":   true,
	"/auth/refresh": true,
	"/auth/logout":  true,
}

// RoleCodes 各角色对应的权限码(/auth/codes)
var RoleCodes = map[string][]string{
	RoleSuperAdmin: {
		"sys:agent:list", "sys:agent:create", "sys:agent:update", "sys:agent:delete",
		"sys:user:list", "sys:user:create", "sys:user:update", "sys:user:delete",
		"AC_100100", "AC_100110", "AC_100120", "AC_100010",
	},
	RoleAdmin: {
		"sys:agent:list", "sys:agent:create", "sys:agent:update", "sys:agent:delete",
		"sys:user:list", "sys:user:create", "sys:user:update", "sys:user:delete",
	},
	RoleUser: {},
}
