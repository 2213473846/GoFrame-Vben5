package dashboard

import (
	"context"
	"os"
	"strings"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/service/identity"
	onlinesvc "demo/internal/service/online"
	sysconfigsvc "demo/internal/service/sysconfig"
	usersvc "demo/internal/service/user"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
)

var dbReady bool

var (
	superId uint64
	adminId uint64
	childId uint64
	grandId uint64
	otherId uint64
)

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	if dbReady {
		seedUsers()
	}
	code := m.Run()
	os.Exit(code)
}

func seedUsers() {
	ctx := testutil.Ctx()
	cols := dao.SysUser.Columns()
	create := func(username, role string, pid uint64) uint64 {
		var exist struct {
			Id uint64 `orm:"id"`
		}
		_ = dao.SysUser.Ctx(ctx).Where(cols.Username, username).Scan(&exist)
		if exist.Id > 0 {
			return exist.Id
		}
		id, err := dao.SysUser.Ctx(ctx).Data(do.SysUser{
			Pid: pid, Username: username, Password: "x", RealName: username, Role: role, Status: 1,
		}).InsertAndGetId()
		if err != nil {
			panic(err)
		}
		return uint64(id)
	}
	superId = create("dash_root", consts.RoleSuperAdmin, 0)
	adminId = create("dash_admin", consts.RoleAdmin, 0)
	childId = create("dash_child", consts.RoleUser, adminId)
	grandId = create("dash_grand", consts.RoleUser, childId)
	otherId = create("dash_other", consts.RoleUser, 0)
}

func superCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: superId, Role: consts.RoleSuperAdmin, Username: "dash_root"})
}

func adminCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: adminId, Role: consts.RoleAdmin, Username: "dash_admin"})
}

func userCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: childId, Role: consts.RoleUser, Username: "dash_child"})
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

func insertLoginLog(t *testing.T, userId uint64, username string, status int, at *gtime.Time) {
	t.Helper()
	if _, err := dao.SysLoginLog.Ctx(testutil.Ctx()).Data(do.SysLoginLog{
		UserId: userId, Username: username, Ip: "127.0.0.1", Status: status, Message: "ut",
		CreatedAt: at,
	}).Insert(); err != nil {
		t.Fatalf("插入登录日志失败: %v", err)
	}
}

func insertOpLog(t *testing.T, userId uint64, username, action string, status int) {
	t.Helper()
	if _, err := dao.SysOperationLog.Ctx(testutil.Ctx()).Data(do.SysOperationLog{
		UserId: userId, Username: username, Module: "ut", Action: action,
		Method: "GET", Path: "/ut", Status: status, CreatedAt: gtime.Now(),
	}).Insert(); err != nil {
		t.Fatalf("插入操作日志失败: %v", err)
	}
}

// TestOverviewPermission 普通用户禁止查看管理统计
func TestOverviewPermission(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	_, err := New().Overview(userCtx())
	expectCode(t, "普通用户查看工作台", err, 403)
	_, err = New().Overview(testutil.Ctx())
	expectCode(t, "未登录查看工作台", err, 401)
}

// TestOverviewScope 管理员统计只覆盖可见范围(自身+子孙), 不含范围外用户
func TestOverviewScope(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ctx := superCtx()
	// 清理本包构造的日志, 保证数字可断言
	_, _ = dao.SysLoginLog.Ctx(ctx).Where("1=1").Delete()
	_, _ = dao.SysOperationLog.Ctx(ctx).Where("1=1").Delete()

	today := gtime.New(gtime.Now().Format("Y-m-d") + " 00:00:00")
	yesterday := today.AddDate(0, 0, -1)

	// 管理员可见范围: 自己 + 子孙
	insertLoginLog(t, adminId, "dash_admin", 1, today)
	insertLoginLog(t, childId, "dash_child", 1, today)
	insertLoginLog(t, childId, "dash_child", 0, today)
	insertLoginLog(t, grandId, "dash_grand", 1, today)
	// 范围外用户
	insertLoginLog(t, otherId, "dash_other", 1, today)
	insertLoginLog(t, otherId, "dash_other", 0, today)
	// 昨天(趋势)
	insertLoginLog(t, childId, "dash_child", 1, yesterday)

	insertOpLog(t, childId, "dash_child", "子用户操作", 1)
	insertOpLog(t, grandId, "dash_grand", "孙用户操作", 1)
	insertOpLog(t, otherId, "dash_other", "范围外操作", 1)

	adminOverview, err := New().Overview(adminCtx())
	if err != nil {
		t.Fatalf("管理员查看工作台失败: %v", err)
	}
	// 今日: 成功 = 自身1 + 子1 + 孙1 = 3; 失败 = 子1
	if adminOverview.LoginSuccessToday != 3 || adminOverview.LoginFailedToday != 1 {
		t.Fatalf("管理员今日登录统计异常: success=%d failed=%d",
			adminOverview.LoginSuccessToday, adminOverview.LoginFailedToday)
	}
	// 范围外用户绝不能被计入
	if adminOverview.LoginSuccessToday > 3 || adminOverview.LoginFailedToday > 1 {
		t.Fatal("管理员统计包含了范围外用户")
	}

	// 最近动态只包含可见范围
	for _, item := range adminOverview.Recent {
		if strings.Contains(item.Username, "dash_other") {
			t.Fatalf("最近动态包含范围外用户: %+v", item)
		}
	}

	// 用户总数卡片必须与用户列表接口的 total 一致(管理员仅子孙, 不含自己)
	_, _, _, listTotal, err := usersvc.New().List(adminCtx(), usersvc.ListInput{Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("查询用户列表失败: %v", err)
	}
	if adminOverview.UserTotal != listTotal {
		t.Fatalf("用户总数卡片(%d)与列表 total(%d)不一致", adminOverview.UserTotal, listTotal)
	}
	if adminOverview.UserTotal != 2 {
		t.Fatalf("管理员可见用户应为 2(子+孙), 实际 %d", adminOverview.UserTotal)
	}

	// 在线数卡片与在线列表 total 一致
	online, err := onlinesvc.List(adminCtx(), &onlinesvc.ListInput{Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("在线用户列表失败: %v", err)
	}
	if adminOverview.OnlineCount != online.Total {
		t.Fatalf("在线数卡片(%d)与在线列表(%d)不一致", adminOverview.OnlineCount, online.Total)
	}

	// 超管: 全量
	superOverview, err := New().Overview(ctx)
	if err != nil {
		t.Fatalf("超管查看工作台失败: %v", err)
	}
	if superOverview.LoginSuccessToday != 4 || superOverview.LoginFailedToday != 2 {
		t.Fatalf("超管今日登录统计异常: success=%d failed=%d",
			superOverview.LoginSuccessToday, superOverview.LoginFailedToday)
	}
	if superOverview.UserTotal <= adminOverview.UserTotal {
		t.Fatalf("超管用户总数(%d)应大于管理员(%d)", superOverview.UserTotal, adminOverview.UserTotal)
	}

	// 趋势: 默认天数来自 sys_config, 且昨天数据落在倒数第二天
	if len(superOverview.Trend) != superOverview.TrendDays {
		t.Fatalf("趋势点数(%d)与天数(%d)不一致", len(superOverview.Trend), superOverview.TrendDays)
	}
	last := superOverview.Trend[len(superOverview.Trend)-1]
	if last.Success != 4 || last.Failed != 2 {
		t.Fatalf("趋势最后一天应与今日统计一致: %+v", last)
	}
	if superOverview.TrendDays > 1 {
		prev := superOverview.Trend[len(superOverview.Trend)-2]
		if prev.Success != 1 {
			t.Fatalf("昨天应有 1 条成功登录, 实际 %+v", prev)
		}
	}

	// sys_config 调整趋势天数后立即生效
	if err = sysconfigsvc.New().Seed(ctx); err != nil {
		t.Fatalf("初始化参数失败: %v", err)
	}
	if err = sysconfigsvc.New().Update(ctx, sysconfigsvc.KeyDashboardTrendDays, "3"); err != nil {
		t.Fatalf("修改趋势天数失败: %v", err)
	}
	changed, err := New().Overview(ctx)
	if err != nil {
		t.Fatalf("修改配置后查看工作台失败: %v", err)
	}
	if changed.TrendDays != 3 || len(changed.Trend) != 3 {
		t.Fatalf("趋势天数应即时变为 3, 实际 days=%d points=%d", changed.TrendDays, len(changed.Trend))
	}
	// 复原, 避免影响其他用例
	_, _ = sysconfigsvc.New().Reset(ctx, sysconfigsvc.KeyDashboardTrendDays)
}
