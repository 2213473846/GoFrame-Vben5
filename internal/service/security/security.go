// Package security 登录安全: 失败计数/锁定、图形验证码、密码策略。
package security

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"demo/internal/consts"

	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
)

// Service 登录安全服务(无状态, 配置全部来自 config.yaml 的 security.loginGuard / security.password)
type Service struct{}

// New 创建登录安全服务
func New() *Service { return &Service{} }

// redis 返回 Redis 实例; 使用 g.Redis() 以便从 config.yaml 的 redis.default 动态创建实例。
// 注意: gredis.Instance() 只返回显式 SetConfig 注册的实例, 服务运行时为 nil。
func redis() *gredis.Redis {
	return g.Redis()
}

// ---------- 配置 ----------

func (s *Service) enabled(ctx context.Context) bool {
	return g.Cfg().MustGet(ctx, "security.loginGuard.enabled", true).Bool()
}

func (s *Service) failWindow(ctx context.Context) int {
	v := g.Cfg().MustGet(ctx, "security.loginGuard.failWindowSeconds", 600).Int()
	if v <= 0 {
		v = 600
	}
	return v
}

func (s *Service) lockThreshold(ctx context.Context) int {
	v := g.Cfg().MustGet(ctx, "security.loginGuard.lockThreshold", 5).Int()
	if v <= 0 {
		v = 5
	}
	return v
}

func (s *Service) lockSeconds(ctx context.Context) int {
	v := g.Cfg().MustGet(ctx, "security.loginGuard.lockSeconds", 900).Int()
	if v <= 0 {
		v = 900
	}
	return v
}

func (s *Service) captchaThreshold(ctx context.Context) int {
	v := g.Cfg().MustGet(ctx, "security.loginGuard.captchaThreshold", 2).Int()
	if v <= 0 {
		v = 2
	}
	return v
}

func (s *Service) captchaTTL(ctx context.Context) int {
	v := g.Cfg().MustGet(ctx, "security.loginGuard.captchaTTLSeconds", 120).Int()
	if v <= 0 {
		v = 120
	}
	return v
}

func (s *Service) lockedMessage(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "security.loginGuard.lockedMessage", "登录失败次数过多, 请稍后重试").String()
}

func (s *Service) captchaRequiredMessage(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "security.loginGuard.captchaRequiredMessage", "登录失败次数较多, 请输入图形验证码后重试").String()
}

func (s *Service) captchaInvalidMessage(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "security.loginGuard.captchaInvalidMessage", "验证码错误, 请重新输入").String()
}

// ChangeRequiredMessage 强制改密拦截提示
func (s *Service) ChangeRequiredMessage(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "security.password.changeRequiredMessage", "请先修改初始密码后再使用系统").String()
}

// ---------- 密码策略 ----------

func (s *Service) passwordRules(ctx context.Context) (minLen, maxLen int, letter, digit, special bool) {
	minLen = g.Cfg().MustGet(ctx, "security.password.minLength", 6).Int()
	maxLen = g.Cfg().MustGet(ctx, "security.password.maxLength", 32).Int()
	if minLen <= 0 {
		minLen = 6
	}
	if maxLen <= 0 || maxLen < minLen {
		maxLen = 32
	}
	letter = g.Cfg().MustGet(ctx, "security.password.requireLetter", false).Bool()
	digit = g.Cfg().MustGet(ctx, "security.password.requireDigit", true).Bool()
	special = g.Cfg().MustGet(ctx, "security.password.requireSpecial", false).Bool()
	return
}

// ValidatePassword 统一密码策略校验(创建用户/重置密码/修改密码/登录时识别存量弱密码均走这里)
func (s *Service) ValidatePassword(ctx context.Context, pwd string) error {
	minLen, maxLen, letter, digit, special := s.passwordRules(ctx)
	msg := g.Cfg().MustGet(ctx, "security.password.policyMessage", "").String()
	if msg == "" {
		msg = fmt.Sprintf("密码长度需为%d-%d位", minLen, maxLen)
		if letter {
			msg += ", 至少包含1个字母"
		}
		if digit {
			msg += ", 至少包含1个数字"
		}
		if special {
			msg += ", 至少包含1个特殊字符"
		}
	}
	n := len([]rune(pwd))
	if n < minLen || n > maxLen {
		return gerror.NewCode(consts.CodeBadRequest, msg)
	}
	hasLetter, hasDigit, hasSpecial := false, false, false
	for _, r := range pwd {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case strings.ContainsRune("~!@#$%^&*()_+-=[]{}|;:',.<>/?", r):
			hasSpecial = true
		}
	}
	if (letter && !hasLetter) || (digit && !hasDigit) || (special && !hasSpecial) {
		return gerror.NewCode(consts.CodeBadRequest, msg)
	}
	return nil
}

// ---------- 失败计数与锁定 ----------

func userKey(username string) string {
	return consts.RedisKeyLoginFailUser + gstr.ToLower(strings.TrimSpace(username))
}

func ipKey(ip string) string {
	return consts.RedisKeyLoginFailIp + ip
}

func lockUserKey(username string) string {
	return consts.RedisKeyLoginLockUser + gstr.ToLower(strings.TrimSpace(username))
}

func lockIpKey(ip string) string {
	return consts.RedisKeyLoginLockIp + ip
}

// incrWindow 固定窗口计数: 首次写入时设置 TTL, 后续失败不会延长窗口(避免无限延长)
const incrWindowScript = `
local c = redis.call('INCR', KEYS[1])
if c == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return c
`

func (s *Service) incrWindow(ctx context.Context, key string) int {
	r := redis()
	if r == nil {
		return 0
	}
	v, err := r.Do(ctx, "EVAL", incrWindowScript, 1, key, s.failWindow(ctx))
	if err != nil {
		g.Log().Warningf(ctx, "登录失败计数失败 key=%s: %v", key, err)
		return 0
	}
	return v.Int()
}

func (s *Service) currentCount(ctx context.Context, username, ip string) int {
	r := redis()
	if r == nil {
		return 0
	}
	uc, ic := 0, 0
	if username != "" {
		if v, err := r.Do(ctx, "GET", userKey(username)); err == nil {
			uc = v.Int()
		}
	}
	if ip != "" {
		if v, err := r.Do(ctx, "GET", ipKey(ip)); err == nil {
			ic = v.Int()
		}
	}
	return max(uc, ic)
}

// LoginLocked 是否处于锁定期(按用户名/IP 任一命中即锁定); 返回统一文案, 不暴露账号是否存在
func (s *Service) LoginLocked(ctx context.Context, username, ip string) (bool, string) {
	if !s.enabled(ctx) || redis() == nil {
		return false, ""
	}
	r := redis()
	hit := false
	for _, key := range []string{lockUserKey(username), lockIpKey(ip)} {
		if v, err := r.Do(ctx, "EXISTS", key); err == nil && v.Int() > 0 {
			hit = true
		}
	}
	if hit {
		return true, s.lockedMessage(ctx)
	}
	return false, ""
}

// CaptchaRequired 失败次数达到较低阈值后, 下一次登录必须先通过验证码
func (s *Service) CaptchaRequired(ctx context.Context, username, ip string) bool {
	if !s.enabled(ctx) || redis() == nil {
		return false
	}
	return s.currentCount(ctx, username, ip) >= s.captchaThreshold(ctx)
}

// RecordFailure 记录一次登录失败(用户名与 IP 两个维度), 达到锁定阈值返回 true 并写入锁定标记。
// 锁定期内调用方应先经 LoginLocked 拦截, 不会走到这里, 因此锁定不会因继续失败而叠加/延长。
func (s *Service) RecordFailure(ctx context.Context, username, ip string) bool {
	if !s.enabled(ctx) || redis() == nil {
		return false
	}
	uc, ic := 0, 0
	if username != "" {
		uc = s.incrWindow(ctx, userKey(username))
	}
	if ip != "" {
		ic = s.incrWindow(ctx, ipKey(ip))
	}
	threshold := s.lockThreshold(ctx)
	if max(uc, ic) < threshold {
		return false
	}
	r := redis()
	lockTTL := s.lockSeconds(ctx)
	if username != "" {
		_, _ = r.Do(ctx, "SET", lockUserKey(username), 1, "EX", lockTTL)
	}
	if ip != "" {
		_, _ = r.Do(ctx, "SET", lockIpKey(ip), 1, "EX", lockTTL)
	}
	return true
}

// Reset 登录成功后清空失败计数与锁定标记
func (s *Service) Reset(ctx context.Context, username, ip string) {
	r := redis()
	if r == nil {
		return
	}
	keys := []string{}
	if username != "" {
		keys = append(keys, userKey(username), lockUserKey(username))
	}
	if ip != "" {
		keys = append(keys, ipKey(ip), lockIpKey(ip))
	}
	if len(keys) > 0 {
		args := make([]any, 0, len(keys))
		for _, k := range keys {
			args = append(args, k)
		}
		_, _ = r.Do(ctx, "DEL", args...)
	}
}

// CaptchaRequiredError 需要验证码时的错误
func (s *Service) CaptchaRequiredError(ctx context.Context) error {
	return gerror.NewCode(consts.CodeCaptchaRequired, s.captchaRequiredMessage(ctx))
}

// CaptchaInvalidError 验证码错误时的错误
func (s *Service) CaptchaInvalidError(ctx context.Context) error {
	return gerror.NewCode(consts.CodeCaptchaInvalid, s.captchaInvalidMessage(ctx))
}

// LockedError 锁定时的错误
func (s *Service) LockedError(ctx context.Context) error {
	return gerror.NewCode(consts.CodeLoginLocked, s.lockedMessage(ctx))
}
