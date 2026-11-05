package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"demo/internal/consts"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
)

// dbReady / redisReady 集成测试依赖是否可用
var (
	dbReady    bool
	redisReady bool
)

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	redisReady = testutil.PrepareRedisDB(13)
	os.Exit(m.Run())
}

func superCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 1, Role: consts.RoleSuperAdmin, Username: "monitor_root"})
}

func adminCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 2, Role: consts.RoleAdmin, Username: "monitor_admin"})
}

func userCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 3, Role: consts.RoleUser, Username: "monitor_user"})
}

// anonCtx 未登录上下文
func anonCtx() context.Context {
	return testutil.Ctx()
}

func assertCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("期望错误码 %d, 实际无错误", want)
	}
	if code := gerror.Code(err); code == nil || code.Code() != want {
		t.Fatalf("期望错误码 %d, 实际 %v (%v)", want, code, err)
	}
}

// TestServerSuperAdminOK 超管可获取完整监控结构, MySQL/Redis 均 ok
func TestServerSuperAdminOK(t *testing.T) {
	if !dbReady || !redisReady {
		t.Skip("MySQL 或 Redis 不可用, 跳过服务监控集成测试")
	}
	stat, err := New().Server(superCtx())
	if err != nil {
		t.Fatalf("超管获取服务监控失败: %v", err)
	}
	if stat.RefreshSeconds <= 0 {
		t.Fatalf("refreshSeconds 应大于 0, 实际 %d", stat.RefreshSeconds)
	}
	if stat.Cpu.Cores <= 0 {
		t.Fatalf("cpu.cores 应大于 0, 实际 %d", stat.Cpu.Cores)
	}
	if stat.Memory.Total == 0 {
		t.Fatalf("memory.total 不应为 0")
	}
	if len(stat.Disks) == 0 {
		t.Fatalf("disks 不应为空")
	}
	for _, d := range stat.Disks {
		if d.Mountpoint == "" {
			t.Fatalf("磁盘 mountpoint 不应为空")
		}
	}
	if stat.Runtime.Goroutines <= 0 {
		t.Fatalf("runtime.goroutines 应大于 0")
	}
	if stat.Runtime.HeapSysBytes == 0 {
		t.Fatalf("runtime.heapSysBytes 不应为 0")
	}
	if stat.UptimeSeconds < 0 {
		t.Fatalf("uptimeSeconds 不应为负")
	}
	if stat.Deps.Mysql.Status != DepOK {
		t.Fatalf("mysql 健康检查应 ok, 实际 %s", stat.Deps.Mysql.Status)
	}
	if stat.Deps.Redis.Status != DepOK {
		t.Fatalf("redis 健康检查应 ok, 实际 %s", stat.Deps.Redis.Status)
	}
	if stat.Deps.Mysql.LatencyMs < 0 || stat.Deps.Redis.LatencyMs < 0 {
		t.Fatalf("依赖延迟不应为负")
	}
}

// TestServerPermission 仅超管可访问: 管理员/普通用户 403, 未登录 401
func TestServerPermission(t *testing.T) {
	_, err := New().Server(adminCtx())
	assertCode(t, err, int(consts.CodeForbidden.Code()))

	_, err = New().Server(userCtx())
	assertCode(t, err, int(consts.CodeForbidden.Code()))

	_, err = New().Server(anonCtx())
	assertCode(t, err, int(consts.CodeUnauthorized.Code()))
}

// TestServerDependencyFail 依赖失败时返回 status=fail, 不 panic 且不泄露错误详情
func TestServerDependencyFail(t *testing.T) {
	oldMysql, oldRedis := mysqlPing, redisPing
	mysqlPing = func(context.Context) error {
		return errors.New("mysql dsn secret")
	}
	redisPing = func(context.Context) error {
		return errors.New("redis addr secret")
	}
	defer func() {
		mysqlPing, redisPing = oldMysql, oldRedis
	}()

	stat, err := New().Server(superCtx())
	if err != nil {
		t.Fatalf("依赖失败不应导致接口报错: %v", err)
	}
	if stat.Deps.Mysql.Status != DepFail || stat.Deps.Redis.Status != DepFail {
		t.Fatalf("依赖应返回 fail, 实际 mysql=%s redis=%s", stat.Deps.Mysql.Status, stat.Deps.Redis.Status)
	}
	raw, err := json.Marshal(stat)
	if err != nil {
		t.Fatalf("序列化监控结果失败: %v", err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("响应不应包含错误详情/连接串: %s", raw)
	}
}
