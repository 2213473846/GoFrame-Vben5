package sysconfig

import (
	"context"
	"os"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
)

var dbReady bool

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	code := m.Run()
	os.Exit(code)
}

func superCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 1, Role: consts.RoleSuperAdmin, Username: "root"})
}

func adminCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 2, Role: consts.RoleAdmin, Username: "admin"})
}

func userCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 3, Role: consts.RoleUser, Username: "user"})
}

func expectCode(t *testing.T, name string, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 期望错误码 %d, 实际无错误", name, want)
	}
	if code := gerror.Code(err); code == nil || code.Code() != want {
		t.Fatalf("%s: 期望错误码 %d, 实际 %v(%v)", name, want, code, err)
	}
}

// TestConfigCRUDAndCache 参数读取/修改/重置/批量与缓存失效
func TestConfigCRUDAndCache(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ClearCache()
	ctx := superCtx()
	s := New()
	if err := s.Seed(ctx); err != nil {
		t.Fatalf("seed 失败: %v", err)
	}

	// 默认值读取(类型化)
	if got := s.GetInt(ctx, KeyDashboardTrendDays, 1); got != 7 {
		t.Fatalf("默认趋势天数应为 7, 实际 %d", got)
	}
	if got := s.GetBool(ctx, "not.exists", false); got != false {
		t.Fatalf("不存在参数应回退默认值")
	}
	if got := s.GetDuration(ctx, KeyDashboardTrendDays, 0); got <= 0 {
		t.Fatalf("时长读取失败: %v", got)
	}

	// 修改后下一次读取立即生效(缓存必须失效)
	if err := s.Update(ctx, KeyDashboardTrendDays, "30"); err != nil {
		t.Fatalf("修改参数失败: %v", err)
	}
	if got := s.GetInt(ctx, KeyDashboardTrendDays, 7); got != 30 {
		t.Fatalf("修改后应读取到 30, 实际 %d", got)
	}

	// 越界校验
	expectCode(t, "趋势天数越界", s.Update(ctx, KeyDashboardTrendDays, "999"), 400)
	expectCode(t, "非数字", s.Update(ctx, KeyDashboardTrendDays, "abc"), 400)

	// 重置为默认
	value, err := s.Reset(ctx, KeyDashboardTrendDays)
	if err != nil {
		t.Fatalf("重置失败: %v", err)
	}
	if value != "7" {
		t.Fatalf("重置后应为 7, 实际 %s", value)
	}
	if got := s.GetInt(ctx, KeyDashboardTrendDays, 0); got != 7 {
		t.Fatalf("重置后读取应为 7, 实际 %d", got)
	}

	// 批量读取
	values, err := s.Batch(ctx, []string{KeyDashboardTrendDays, KeyDashboardRecentLimit, KeyExportMaxRows})
	if err != nil {
		t.Fatalf("批量读取失败: %v", err)
	}
	if len(values) != 3 || values[KeyDashboardTrendDays] != "7" || values[KeyExportMaxRows] != "500000" {
		t.Fatalf("批量读取结果异常: %+v", values)
	}

	// 列表(已登录可读)
	list, err := s.List(ctx, ListInput{Status: -1})
	if err != nil {
		t.Fatalf("列表读取失败: %v", err)
	}
	if len(list) < len(Defaults()) {
		t.Fatalf("列表数量应不少于内置参数数量: %d < %d", len(list), len(Defaults()))
	}
	for _, item := range list {
		if item.ConfigKey == KeyImportMaxRows && item.Builtin != 1 {
			t.Fatalf("内置参数 builtin 标记应为 1: %+v", item)
		}
	}
}

// TestConfigPermission 写操作仅超管, 读对已登录用户开放
func TestConfigPermission(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ctx := superCtx()
	s := New()
	if err := s.Seed(ctx); err != nil {
		t.Fatalf("seed 失败: %v", err)
	}

	// 管理员/普通用户可读
	if _, err := s.List(adminCtx(), ListInput{Status: -1}); err != nil {
		t.Fatalf("管理员读参数列表应成功: %v", err)
	}
	if _, err := s.List(userCtx(), ListInput{Status: -1}); err != nil {
		t.Fatalf("普通用户读参数列表应成功: %v", err)
	}
	if _, err := s.Batch(userCtx(), []string{KeyDashboardTrendDays}); err != nil {
		t.Fatalf("普通用户批量读取应成功: %v", err)
	}

	// 非超管写: 403
	expectCode(t, "管理员改参数", s.Update(adminCtx(), KeyDashboardTrendDays, "10"), 403)
	expectCode(t, "普通用户改参数", s.Update(userCtx(), KeyDashboardTrendDays, "10"), 403)
	_, err := s.Reset(adminCtx(), KeyDashboardTrendDays)
	expectCode(t, "管理员重置参数", err, 403)

	// 未登录: 401
	_, err = s.List(testutil.Ctx(), ListInput{Status: -1})
	expectCode(t, "未登录读参数", err, 401)
	expectCode(t, "未登录写参数", s.Update(testutil.Ctx(), KeyDashboardTrendDays, "10"), 401)
}

// TestConfigBuiltinProtection 内置参数保护: 非内置参数不支持重置; 种子幂等
func TestConfigBuiltinProtection(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ctx := superCtx()
	s := New()
	if err := s.Seed(ctx); err != nil {
		t.Fatalf("seed 失败: %v", err)
	}
	if err := s.Seed(ctx); err != nil {
		t.Fatalf("二次 seed 失败: %v", err)
	}
	cols := dao.SysConfig.Columns()
	for _, item := range Defaults() {
		count, err := dao.SysConfig.Ctx(ctx).Where(cols.ConfigKey, item.Key).Count()
		if err != nil {
			t.Fatalf("统计内置参数失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("内置参数 %s 应只有 1 条, 实际 %d", item.Key, count)
		}
	}
	// 非内置参数不支持重置默认值
	if _, err := dao.SysConfig.Ctx(ctx).Data(map[string]interface{}{
		"config_key":   "custom.demo",
		"config_value": "x",
		"name":         "自定义参数",
		"status":       1,
		"builtin":      0,
	}).Insert(); err != nil {
		t.Fatalf("插入自定义参数失败: %v", err)
	}
	_, err := s.Reset(ctx, "custom.demo")
	expectCode(t, "非内置参数重置", err, 400)

	// 不存在参数修改: 404
	expectCode(t, "不存在参数修改", s.Update(ctx, "not.exists.key", "1"), 404)
}
