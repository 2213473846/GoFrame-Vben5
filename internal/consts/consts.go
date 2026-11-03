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
	RedisKeyOnline       = "vben:online:"  // + sessionId, 在线用户会话信息
)

// Redis Key 前缀(登录设备数限制)
const (
	RedisKeySessionSet   = "vben:sess:user:"  // ZSET: 用户 -> 有效会话集合, member=sessionId, score=登录时间(ms)
	RedisKeySessionMeta  = "vben:sess:meta:"  // HASH: sessionId -> 会话摘要(access/refresh/设备/登录时间)
	RedisKeySessionLimit = "vben:sess:limit:" // + userId, 当前生效的设备数上限
	RedisKeySessionKick  = "vben:sess:kick:"  // + sessionId, 会话被踢原因(短期保留, 供被拒请求提示)
)

// Redis Key 前缀(登录安全: 失败计数/锁定/验证码)
const (
	RedisKeyLoginFailUser = "vben:sec:fail:user:" // + username(小写), 失败计数
	RedisKeyLoginFailIp   = "vben:sec:fail:ip:"   // + IP, 失败计数
	RedisKeyLoginLockUser = "vben:sec:lock:user:" // + username(小写), 锁定标记
	RedisKeyLoginLockIp   = "vben:sec:lock:ip:"   // + IP, 锁定标记
	RedisKeyCaptcha       = "vben:sec:captcha:"   // + captchaId, 图形验证码答案(一次性)
)

// Token 有效期
const (
	AccessTTL  = 2 * 60 * 60      // accessToken: 2小时
	RefreshTTL = 7 * 24 * 60 * 60 // refreshToken: 7天
)

// 会话相关保留时长(秒)
const (
	SessionKickReasonTTL   = 600  // 被踢原因保留 10 分钟, 覆盖旧端下一次请求
	OnlineOfflineRetention = 1800 // 已下线会话记录保留 30 分钟, 便于管理员查看下线原因
)

// 错误码(与 HTTP 状态码一致, 由响应中间件映射)
var (
	CodeUnauthorized = gcode.New(401, "Unauthorized", nil)
	CodeForbidden    = gcode.New(403, "Forbidden", nil)
	CodeNotFound     = gcode.New(404, "NotFound", nil)
	CodeBadRequest   = gcode.New(400, "BadRequest", nil)
	// CodeSessionKicked 会话已被顶替/超出设备数上限/被强制下线, HTTP 状态映射为 401
	CodeSessionKicked = gcode.New(4001, "SessionKicked", nil)

	// 登录安全相关业务码
	CodeCaptchaInvalid         = gcode.New(4027, "CaptchaInvalid", nil)         // 验证码错误, HTTP 400
	CodeCaptchaRequired        = gcode.New(4028, "CaptchaRequired", nil)        // 需要先输入验证码, HTTP 400
	CodeLoginLocked            = gcode.New(4029, "LoginLocked", nil)            // 登录失败次数过多已锁定, HTTP 429
	CodePasswordChangeRequired = gcode.New(4030, "PasswordChangeRequired", nil) // 需先修改初始密码, HTTP 403
)

// AuthWhitelist 无需登录即可访问的接口
var AuthWhitelist = map[string]bool{
	"/auth/login":   true,
	"/auth/refresh": true,
	"/auth/logout":  true,
	"/auth/captcha": true,
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
