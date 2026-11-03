package auth

import (
	"os"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	secsvc "demo/internal/service/security"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"golang.org/x/crypto/bcrypt"
)

// dbReady/redisReady 依赖的本机设施可用时为 true, 否则跳过集成测试
var (
	dbReady    bool
	redisReady bool
)

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	redisReady = testutil.PrepareRedisDB(13)
	os.Exit(m.Run())
}

func skipIfNotReady(t *testing.T) {
	t.Helper()
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过登录集成测试")
	}
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过登录集成测试")
	}
}

func flushRedis(t *testing.T) {
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

func exists(t *testing.T, key string) bool {
	t.Helper()
	v, err := g.Redis().Do(testutil.Ctx(), "EXISTS", key)
	if err != nil {
		t.Fatalf("EXISTS %s 失败: %v", key, err)
	}
	return v.Int() > 0
}

func sessionCount(t *testing.T, userId uint64) int {
	t.Helper()
	v, err := g.Redis().Do(testutil.Ctx(), "ZCARD", consts.RedisKeySessionSet+gconv.String(userId))
	if err != nil {
		t.Fatalf("ZCARD 失败: %v", err)
	}
	return v.Int()
}

// insertUser 插入带 bcrypt 密码的测试用户
func insertUser(t *testing.T, username, password string, mustChange int) uint64 {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("生成密码失败: %v", err)
	}
	id, err := dao.SysUser.Ctx(testutil.Ctx()).Data(do.SysUser{
		Username:           username,
		Password:           string(hash),
		RealName:           username,
		Role:               consts.RoleUser,
		Status:             1,
		MustChangePassword: mustChange,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入用户 %s 失败: %v", username, err)
	}
	return gconv.Uint64(id)
}

func mustChangeFlag(t *testing.T, userId uint64) int {
	t.Helper()
	v, err := dao.SysUser.Ctx(testutil.Ctx()).
		Where(dao.SysUser.Columns().Id, userId).
		Fields(dao.SysUser.Columns().MustChangePassword).
		Value()
	if err != nil {
		t.Fatalf("查询 must_change_password 失败: %v", err)
	}
	return v.Int()
}

// doLogin 以固定 IP 登录, 便于按 IP 维度观察失败计数
func doLogin(svc *Service, username, password, captchaId, captchaCode string) (*LoginResult, error) {
	return svc.Login(testutil.Ctx(), LoginInput{
		Username:    username,
		Password:    password,
		CaptchaId:   captchaId,
		CaptchaCode: captchaCode,
		Meta:        LoginMeta{Ip: "127.0.0.10"},
	})
}

// captchaWithAnswer 生成验证码并直接从 Redis 取答案(测试专用)
func captchaWithAnswer(t *testing.T) (id, answer string) {
	t.Helper()
	c, err := secsvc.New().NewCaptcha(testutil.Ctx())
	if err != nil {
		t.Fatalf("生成验证码失败: %v", err)
	}
	return c.Id, redisValue(t, consts.RedisKeyCaptcha+c.Id)
}

func errCode(err error) int {
	if err == nil {
		return 0
	}
	return gerror.Code(err).Code()
}

// failCount 失败计数: 用户名与 IP 两个维度取较大值(与 service 口径一致)
func failCount(t *testing.T, username, ip string) int {
	t.Helper()
	uc := 0
	if v, err := g.Redis().Do(testutil.Ctx(), "GET", consts.RedisKeyLoginFailUser+username); err == nil {
		uc = v.Int()
	}
	ic := 0
	if v, err := g.Redis().Do(testutil.Ctx(), "GET", consts.RedisKeyLoginFailIp+ip); err == nil {
		ic = v.Int()
	}
	if uc > ic {
		return uc
	}
	return ic
}

// lockThreshold 读取配置的锁定阈值(未加载配置文件时与服务默认值一致)
func lockThreshold() int {
	v := g.Cfg().MustGet(testutil.Ctx(), "security.loginGuard.lockThreshold", 5).Int()
	if v <= 0 {
		v = 5
	}
	return v
}

// TestLoginFailureCounterAndLock 连续失败 -> 触发验证码 -> 继续失败 -> 锁定 -> 锁定期内正确密码也被拒
func TestLoginFailureCounterAndLock(t *testing.T) {
	skipIfNotReady(t)
	flushRedis(t)
	svc := New()
	sec := secsvc.New()
	ctx := testutil.Ctx()

	insertUser(t, "lock-alice", "Passw0rd", 0)

	// 第 1、2 次失败(未到验证码阈值)
	if _, err := doLogin(svc, "lock-alice", "wrong", "", ""); errCode(err) != int(consts.CodeForbidden.Code()) {
		t.Fatalf("第 1 次失败期望 403, got %v", err)
	}
	if _, err := doLogin(svc, "lock-alice", "wrong", "", ""); errCode(err) != int(consts.CodeForbidden.Code()) {
		t.Fatalf("第 2 次失败期望 403, got %v", err)
	}
	if !sec.CaptchaRequired(ctx, "lock-alice", "127.0.0.10") {
		t.Fatalf("失败 2 次后应要求验证码")
	}
	// 第 3 次: 不带验证码 -> 4028, 不增加失败计数
	countBefore := failCount(t, "lock-alice", "127.0.0.10")
	if _, err := doLogin(svc, "lock-alice", "wrong", "", ""); errCode(err) != int(consts.CodeCaptchaRequired.Code()) {
		t.Fatalf("需要验证码时期望 4028, got %v", err)
	}
	if got := failCount(t, "lock-alice", "127.0.0.10"); got != countBefore {
		t.Fatalf("缺少验证码不应增加失败计数, before=%d after=%d", countBefore, got)
	}

	// 第 3、4 次: 带正确验证码但密码错误 -> 计数 3、4
	for i := 3; i <= 4; i++ {
		id, answer := captchaWithAnswer(t)
		if _, err := doLogin(svc, "lock-alice", "wrong", id, answer); errCode(err) != int(consts.CodeForbidden.Code()) {
			t.Fatalf("第 %d 次失败期望 403, got %v", i, err)
		}
	}
	// 第 5 次: 带正确验证码仍密码错误 -> 达到阈值, 直接返回锁定错误
	id, answer := captchaWithAnswer(t)
	if _, err := doLogin(svc, "lock-alice", "wrong", id, answer); errCode(err) != int(consts.CodeLoginLocked.Code()) {
		t.Fatalf("达到阈值期望 4029, got %v", err)
	}
	locked, msg := sec.LoginLocked(ctx, "lock-alice", "127.0.0.10")
	if !locked || msg == "" {
		t.Fatalf("达到阈值后应锁定, locked=%v msg=%q", locked, msg)
	}
	if !exists(t, consts.RedisKeyLoginLockUser+"lock-alice") || !exists(t, consts.RedisKeyLoginLockIp+"127.0.0.10") {
		t.Fatalf("用户名与 IP 维度都应写入锁定标记")
	}

	// 锁定期内: 即使密码正确也被拒, 且不叠加计数(计数保持阈值)
	if _, err := doLogin(svc, "lock-alice", "Passw0rd", "", ""); errCode(err) != int(consts.CodeLoginLocked.Code()) {
		t.Fatalf("锁定期内正确密码也期望 4029, got %v", err)
	}
	if got := failCount(t, "lock-alice", "127.0.0.10"); got != lockThreshold() {
		t.Fatalf("锁定期内失败计数不应叠加, got %d", got)
	}
}

// TestCaptchaFailureCountsTowardLock 验证码校验失败也计入失败锁定
func TestCaptchaFailureCountsTowardLock(t *testing.T) {
	skipIfNotReady(t)
	flushRedis(t)
	svc := New()
	sec := secsvc.New()
	ctx := testutil.Ctx()

	insertUser(t, "cap-bob", "Passw0rd", 0)
	// 直接把计数推到阈值前一次
	for i := 0; i < lockThreshold()-1; i++ {
		sec.RecordFailure(ctx, "cap-bob", "127.0.0.10")
	}
	// 验证码错误 -> 计数到达阈值 -> 锁定
	id, _ := captchaWithAnswer(t)
	if _, err := doLogin(svc, "cap-bob", "Passw0rd", id, "0"); errCode(err) != int(consts.CodeLoginLocked.Code()) {
		t.Fatalf("验证码校验失败应计入锁定, got %v", err)
	}
	if got := failCount(t, "cap-bob", "127.0.0.10"); got != lockThreshold() {
		t.Fatalf("验证码失败应计入计数, got %d", got)
	}
	if !exists(t, consts.RedisKeyLoginLockUser+"cap-bob") {
		t.Fatalf("验证码失败后应写入锁定标记")
	}
	if exists(t, consts.RedisKeyCaptcha+id) {
		t.Fatalf("验证码校验后应被删除(一次性)")
	}
}

// TestLoginSuccessResetsCounter 验证通过并登录成功后, 失败计数清零
func TestLoginSuccessResetsCounter(t *testing.T) {
	skipIfNotReady(t)
	flushRedis(t)
	svc := New()
	sec := secsvc.New()
	ctx := testutil.Ctx()

	insertUser(t, "reset-carol", "Passw0rd", 0)
	if _, err := doLogin(svc, "reset-carol", "wrong", "", ""); errCode(err) != int(consts.CodeForbidden.Code()) {
		t.Fatalf("第 1 次失败期望 403, got %v", err)
	}
	if _, err := doLogin(svc, "reset-carol", "wrong", "", ""); errCode(err) != int(consts.CodeForbidden.Code()) {
		t.Fatalf("第 2 次失败期望 403, got %v", err)
	}
	// 带正确验证码与正确密码登录成功
	id, answer := captchaWithAnswer(t)
	out, err := doLogin(svc, "reset-carol", "Passw0rd", id, answer)
	if err != nil {
		t.Fatalf("验证码正确且密码正确时应登录成功: %v", err)
	}
	if out.MustChangePassword {
		t.Fatalf("密码合规且未标记用户不应要求改密")
	}
	if got := failCount(t, "reset-carol", "127.0.0.10"); got != 0 {
		t.Fatalf("登录成功后失败计数应清零, got %d", got)
	}
	if locked, _ := sec.LoginLocked(ctx, "reset-carol", "127.0.0.10"); locked {
		t.Fatalf("登录成功后不应处于锁定状态")
	}
	if !exists(t, consts.RedisKeyAccessToken+out.AccessToken) {
		t.Fatalf("登录成功应签发 accessToken")
	}
}

// TestFirstLoginMustChangePassword 首登强制改密标记: 登录成功但要求改密; 弱密码存量账号也被覆盖
func TestFirstLoginMustChangePassword(t *testing.T) {
	skipIfNotReady(t)
	flushRedis(t)

	// 新建用户(must_change_password=1)首登即要求改密
	svc := New()
	created := insertUser(t, "new-dave", "Passw0rd", 1)
	out, err := doLogin(svc, "new-dave", "Passw0rd", "", "")
	if err != nil {
		t.Fatalf("首登应允许登录成功: %v", err)
	}
	if !out.MustChangePassword {
		t.Fatalf("新建用户首登应要求改密")
	}
	if flag := mustChangeFlag(t, created); flag != 1 {
		t.Fatalf("改密前标记应为 1, got %d", flag)
	}

	// 存量弱密码(不满足当前策略)在登录成功时被自动标记
	weak := insertUser(t, "weak-erin", "abcdef", 0)
	weakOut, err := doLogin(svc, "weak-erin", "abcdef", "", "")
	if err != nil {
		t.Fatalf("存量弱密码账号仍应允许登录(再强制改密): %v", err)
	}
	if !weakOut.MustChangePassword {
		t.Fatalf("存量弱密码账号登录后应要求改密")
	}
	if flag := mustChangeFlag(t, weak); flag != 1 {
		t.Fatalf("存量弱密码应被持久标记 must_change_password=1, got %d", flag)
	}
}

// TestChangePasswordResetsFlagAndInvalidatesSessions 改密成功: 清标记 + 全部旧会话失效重登 + 会话登记不残留
func TestChangePasswordResetsFlagAndInvalidatesSessions(t *testing.T) {
	skipIfNotReady(t)
	flushRedis(t)
	svc := New()

	user := insertUser(t, "chg-frank", "Passw0rd", 1)
	out, err := doLogin(svc, "chg-frank", "Passw0rd", "", "")
	if err != nil {
		t.Fatalf("首登应成功: %v", err)
	}
	if sessionCount(t, user) != 1 {
		t.Fatalf("登录后应有 1 个会话登记, got %d", sessionCount(t, user))
	}

	// 原密码错误 / 新旧相同 / 新密码不合规 均拒绝
	if err := svc.ChangePassword(testutil.Ctx(), user, "bad-old", "Abc12345"); errCode(err) != int(consts.CodeForbidden.Code()) {
		t.Fatalf("原密码错误期望 403, got %v", err)
	}
	if err := svc.ChangePassword(testutil.Ctx(), user, "Passw0rd", "Passw0rd"); errCode(err) != int(consts.CodeBadRequest.Code()) {
		t.Fatalf("新旧密码相同期望 400, got %v", err)
	}
	if err := svc.ChangePassword(testutil.Ctx(), user, "Passw0rd", "abcdef"); errCode(err) != int(consts.CodeBadRequest.Code()) {
		t.Fatalf("新密码不合规期望 400, got %v", err)
	}

	// 正常改密
	if err := svc.ChangePassword(testutil.Ctx(), user, "Passw0rd", "Abc12345"); err != nil {
		t.Fatalf("合规改密应成功: %v", err)
	}
	if flag := mustChangeFlag(t, user); flag != 0 {
		t.Fatalf("改密后标记应清零, got %d", flag)
	}
	// 旧 access/refresh 立即失效, 会话登记清空(不影响设备数上限)
	if exists(t, consts.RedisKeyAccessToken+out.AccessToken) {
		t.Fatalf("改密后旧 accessToken 应失效")
	}
	if exists(t, consts.RedisKeyRefreshToken+out.RefreshToken) {
		t.Fatalf("改密后旧 refreshToken 应失效")
	}
	if got := sessionCount(t, user); got != 0 {
		t.Fatalf("改密后会话登记应清空, got %d", got)
	}

	// 旧密码不可登录, 新密码可登录且不再要求改密(改密后放行)
	if _, err := doLogin(svc, "chg-frank", "Passw0rd", "", ""); errCode(err) != int(consts.CodeForbidden.Code()) {
		t.Fatalf("旧密码应无法登录, got %v", err)
	}
	again, err := doLogin(svc, "chg-frank", "Abc12345", "", "")
	if err != nil {
		t.Fatalf("新密码应能正常登录: %v", err)
	}
	if again.MustChangePassword {
		t.Fatalf("改密后不应再要求改密")
	}
	if got := sessionCount(t, user); got != 1 {
		t.Fatalf("重新登录后应只有 1 个会话登记(不残留旧会话), got %d", got)
	}
	if exists(t, consts.RedisKeyAccessToken+again.AccessToken) != true {
		t.Fatalf("重新登录应签发新的 accessToken")
	}
}
