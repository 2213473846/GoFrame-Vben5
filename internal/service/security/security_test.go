package security

import (
	"os"
	"strings"
	"testing"

	"demo/internal/consts"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/frame/g"
)

// redisReady 本机 Redis 可用时为 true, 否则跳过登录安全测试
var redisReady bool

func TestMain(m *testing.M) {
	redisReady = testutil.PrepareRedisDB(14)
	os.Exit(m.Run())
}

func skipIfNoRedis(t *testing.T) {
	t.Helper()
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过登录安全测试")
	}
}

// flush 每个用例前清空测试库(DB 15), 用例相互独立
func flush(t *testing.T) {
	t.Helper()
	if _, err := g.Redis().Do(testutil.Ctx(), "FLUSHDB"); err != nil {
		t.Fatalf("清空 Redis 失败: %v", err)
	}
}

func redisValue(t *testing.T, key string) string {
	t.Helper()
	v, err := g.Redis().Do(testutil.Ctx(), "GET", key)
	if err != nil {
		t.Fatalf("GET %s 失败: %v", key, err)
	}
	return v.String()
}

func keyExists(t *testing.T, key string) bool {
	t.Helper()
	v, err := g.Redis().Do(testutil.Ctx(), "EXISTS", key)
	if err != nil {
		t.Fatalf("EXISTS %s 失败: %v", key, err)
	}
	return v.Int() > 0
}

// TestFailureCounterAndLock 连续失败累加, 达到阈值后按用户名与 IP 双维度写入锁定标记
func TestFailureCounterAndLock(t *testing.T) {
	skipIfNoRedis(t)
	flush(t)
	ctx := testutil.Ctx()
	s := New()

	const username = "lock-user"
	const ip = "10.0.0.1"
	threshold := s.lockThreshold(ctx)
	for i := 1; i < threshold; i++ {
		if s.RecordFailure(ctx, username, ip) {
			t.Fatalf("第 %d 次失败不应触发锁定", i)
		}
	}
	if got := s.currentCount(ctx, username, ip); got != threshold-1 {
		t.Fatalf("失败计数期望 %d, got %d", threshold-1, got)
	}
	if !s.RecordFailure(ctx, username, ip) {
		t.Fatalf("第 %d 次失败应触发锁定", threshold)
	}
	if locked, msg := s.LoginLocked(ctx, username, ip); !locked || msg == "" {
		t.Fatalf("达到阈值后应处于锁定状态, locked=%v msg=%q", locked, msg)
	}
	// 两个维度都应有锁定标记
	if !keyExists(t, consts.RedisKeyLoginLockUser+username) || !keyExists(t, consts.RedisKeyLoginLockIp+ip) {
		t.Fatalf("用户名与 IP 维度都应写入锁定标记")
	}
	// 锁定期内不应继续叠加计数(调用方在 LoginLocked 前置拦截)
	if got := redisValue(t, consts.RedisKeyLoginFailUser+username); got != "" {
		// 计数允许在固定窗口内保留, 但锁定后不应再被调用方调用 RecordFailure
		if n := s.currentCount(ctx, username, ip); n != threshold {
			t.Fatalf("锁定后计数不应变化, got %d", n)
		}
	}
}

// TestResetClearsFailureAndLock 登录成功后清空失败计数与锁定标记
func TestResetClearsFailureAndLock(t *testing.T) {
	skipIfNoRedis(t)
	flush(t)
	ctx := testutil.Ctx()
	s := New()
	for i := 0; i < s.lockThreshold(ctx); i++ {
		s.RecordFailure(ctx, "reset-user", "10.0.0.2")
	}
	if s.currentCount(ctx, "reset-user", "10.0.0.2") == 0 {
		t.Fatalf("重置前应有失败计数")
	}
	s.Reset(ctx, "reset-user", "10.0.0.2")
	if got := s.currentCount(ctx, "reset-user", "10.0.0.2"); got != 0 {
		t.Fatalf("重置后失败计数应为 0, got %d", got)
	}
	if locked, _ := s.LoginLocked(ctx, "reset-user", "10.0.0.2"); locked {
		t.Fatalf("重置后不应处于锁定状态")
	}
	if keyExists(t, consts.RedisKeyLoginLockUser+"reset-user") || keyExists(t, consts.RedisKeyLoginLockIp+"10.0.0.2") {
		t.Fatalf("重置后锁定标记应被删除")
	}
}

// TestCaptchaTriggerAndOneTime 验证码在较低阈值触发, 一次性使用, 校验后立即失效(防重放)
func TestCaptchaTriggerAndOneTime(t *testing.T) {
	skipIfNoRedis(t)
	flush(t)
	ctx := testutil.Ctx()
	s := New()

	const username = "cap-user"
	const ip = "10.0.0.3"
	s.RecordFailure(ctx, username, ip)
	if s.CaptchaRequired(ctx, username, ip) {
		t.Fatalf("仅失败 1 次不应要求验证码")
	}
	s.RecordFailure(ctx, username, ip)
	if !s.CaptchaRequired(ctx, username, ip) {
		t.Fatalf("失败次数达到阈值后应要求验证码")
	}

	c, err := s.NewCaptcha(ctx)
	if err != nil {
		t.Fatalf("生成验证码失败: %v", err)
	}
	if c.Id == "" || !strings.HasPrefix(c.Image, "data:image/svg+xml;base64,") {
		t.Fatalf("验证码返回结构不完整: %+v", c)
	}
	answer := redisValue(t, consts.RedisKeyCaptcha+c.Id)
	if answer == "" {
		t.Fatalf("验证码答案未写入 Redis")
	}
	// 有效期检查
	v, err := g.Redis().Do(ctx, "TTL", consts.RedisKeyCaptcha+c.Id)
	if err != nil {
		t.Fatalf("TTL 查询失败: %v", err)
	}
	if ttl := v.Int(); ttl <= 0 || ttl > c.ExpireSeconds {
		t.Fatalf("验证码 TTL 异常: %d", ttl)
	}
	if !s.VerifyCaptcha(ctx, c.Id, answer) {
		t.Fatalf("正确答案应校验通过")
	}
	// 一次性: 同一 captchaId 再提交必须失败, 且答案已被删除(防重放)
	if s.VerifyCaptcha(ctx, c.Id, answer) {
		t.Fatalf("同一验证码重复使用应失败")
	}
	if keyExists(t, consts.RedisKeyCaptcha+c.Id) {
		t.Fatalf("校验后验证码答案应被删除")
	}

	// 错误答案同样立即删除
	c2, err := s.NewCaptcha(ctx)
	if err != nil {
		t.Fatalf("生成验证码失败: %v", err)
	}
	if s.VerifyCaptcha(ctx, c2.Id, "999") {
		t.Fatalf("错误答案不应通过")
	}
	if keyExists(t, consts.RedisKeyCaptcha+c2.Id) {
		t.Fatalf("校验失败后验证码答案也应被删除")
	}
	// 未知 captchaId 直接失败
	if s.VerifyCaptcha(ctx, "not-exists", "1") {
		t.Fatalf("不存在的验证码不应通过")
	}
}

// TestValidatePassword 密码策略统一校验(长度/数字), 覆盖创建、重置、改密与存量弱密码识别
func TestValidatePassword(t *testing.T) {
	ctx := testutil.Ctx()
	s := New()
	valid := []string{"abc123", "Passw0rd", "12345678"}
	for _, p := range valid {
		if err := s.ValidatePassword(ctx, p); err != nil {
			t.Fatalf("密码 %q 应通过策略: %v", p, err)
		}
	}
	minLen, maxLen, _, _, _ := s.passwordRules(ctx)
	invalid := []string{
		strings.Repeat("1", minLen-1), // 长度不足
		strings.Repeat("1", maxLen+1), // 超长
		strings.Repeat("a", minLen),   // 无数字
	}
	for _, p := range invalid {
		if err := s.ValidatePassword(ctx, p); err == nil {
			t.Fatalf("密码 %q 应被策略拒绝", p)
		}
	}
}
