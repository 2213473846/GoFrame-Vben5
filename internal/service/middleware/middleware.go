package middleware

import (
	"demo/internal/consts"
	"demo/internal/service/identity"
	logsvc "demo/internal/service/log"
	onlinesvc "demo/internal/service/online"
	"demo/internal/service/token"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/text/gstr"
)

// Service HTTP 中间件服务
type Service struct {
	tokenSvc *token.Service
}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{
		tokenSvc: token.New(),
	}
}

// Response 统一响应包装(vben 格式: {code, data, error, message})
// - 成功: HTTP 200, code=0
// - 业务错误(默认): HTTP 200, code=-1, message=错误信息
// - 未登录/过期: HTTP 401(触发前端刷新token流程)
// - 禁止访问: HTTP 403
// 若 handler 已自行输出(如 /auth/refresh 返回原始token), 则不再包装
func (s *Service) Response(r *ghttp.Request) {
	r.Middleware.Next()
	// 操作日志埋点: 记录管理接口的写操作(登录接口单独记录在登录日志)
	recordOperationLog(r)
	if r.Response.BufferLength() > 0 {
		return
	}
	if err := r.GetError(); err != nil {
		msg := err.Error()
		httpStatus := 200
		bodyCode := -1
		switch gerror.Code(err).Code() {
		case int(consts.CodeUnauthorized.Code()):
			httpStatus, bodyCode = 401, 401
		case int(consts.CodeForbidden.Code()):
			httpStatus, bodyCode = 403, -1
		}
		// 直接设置状态码(WriteStatus 会把状态文本写入响应体, 导致 JSON 前缀污染)
		r.Response.Status = httpStatus
		r.Response.WriteJson(g.Map{
			"code":    bodyCode,
			"data":    nil,
			"error":   msg,
			"message": msg,
		})
		return
	}
	r.Response.WriteJson(g.Map{
		"code":    0,
		"data":    r.GetHandlerResponse(),
		"error":   nil,
		"message": "ok",
	})
}

// Auth 登录态校验: 解析 Bearer token 并将身份注入上下文
func (s *Service) Auth(r *ghttp.Request) {
	// 白名单接口直接放行
	if consts.AuthWhitelist[r.URL.Path] {
		r.Middleware.Next()
		return
	}
	id, err := s.tokenSvc.Verify(r.Context(), GetBearerToken(r))
	if err != nil {
		msg := err.Error()
		r.Response.Status = 401
		r.Response.WriteJson(g.Map{
			"code":    401,
			"data":    nil,
			"error":   msg,
			"message": msg,
		})
		return
	}
	r.SetCtx(identity.With(r.Context(), *id))
	// 在线会话续期: 活跃会话保持在线, 停止活动后随 TTL 过期
	onlinesvc.Touch(r.Context(), GetBearerToken(r))
	r.Middleware.Next()
}

// GetBearerToken 从请求头解析 accessToken
func GetBearerToken(r *ghttp.Request) string {
	auth := r.Header.Get("Authorization")
	if !gstr.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return gstr.Trim(auth[7:])
}

// operationLogRoutes 操作日志埋点路由映射: 路径 => (模块, 操作类型)
var operationLogRoutes = map[string][2]string{
	"/user/create":  {"用户管理", "新增用户"},
	"/user/update":  {"用户管理", "修改用户"},
	"/user/delete":  {"用户管理", "删除用户"},
	"/user/purge":   {"用户管理", "彻底删除用户"},
	"/user/restore": {"用户管理", "恢复用户"},
	"/menu/create":  {"路由管理", "新增路由"},
	"/menu/update":  {"路由管理", "修改路由"},
	"/menu/delete":  {"路由管理", "删除路由"},
	"/menu/grant":   {"路由管理", "菜单授权"},
	"/online/kick":  {"在线用户", "强退用户"},
}

// recordOperationLog 记录管理写操作日志(登录/登出等认证接口除外)
func recordOperationLog(r *ghttp.Request) {
	if r.Method == "GET" {
		return
	}
	route, ok := operationLogRoutes[r.URL.Path]
	if !ok {
		return
	}
	// Next 之后取错误与响应: 有错误记失败, 否则记成功
	success := true
	message := "操作成功"
	if err := r.GetError(); err != nil {
		success = false
		message = err.Error()
	}
	logsvc.RecordOperationFromRequest(r, route[0], route[1], success, message)
}
