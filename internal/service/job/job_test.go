package job

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
)

var dbReady bool

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	os.Exit(m.Run())
}

func superCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 1, Role: consts.RoleSuperAdmin, Username: "job_root"})
}

func adminCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 2, Role: consts.RoleAdmin, Username: "job_admin"})
}

func userCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 3, Role: consts.RoleUser, Username: "job_user"})
}

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

// registerTestTarget 注入计数型测试任务, 测试结束后移除
func registerTestTarget(t *testing.T, target string, fn JobFunc) {
	t.Helper()
	registryMu.Lock()
	registry[target] = fn
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, target)
		registryMu.Unlock()
	})
}

func validInput(name, target, cron string, status int) CreateInput {
	return CreateInput{
		Name:              name,
		JobGroup:          "default",
		InvokeTarget:      target,
		CronExpression:    cron,
		Status:            status,
		ConcurrencyPolicy: ConcurrencyForbid,
		TimeoutSeconds:    60,
	}
}

// TestCreateValidation 非法 cron 与白名单外任务标识必须被拒绝
func TestCreateValidation(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过定时任务集成测试")
	}
	s := New()
	_, err := s.Create(superCtx(), validInput("bad-cron", TargetClearLog, "* * *", StatusEnabled))
	assertCode(t, err, int(consts.CodeBadRequest.Code()))
	if !strings.Contains(err.Error(), "cron") {
		t.Fatalf("非法 cron 错误信息应包含具体原因, 实际: %v", err)
	}

	// 白名单外标识(绝不能按字符串反射/命令执行)
	_, err = s.Create(superCtx(), validInput("bad-target", "exec_shell", "0 0 3 * * *", StatusEnabled))
	assertCode(t, err, int(consts.CodeBadRequest.Code()))
	if !strings.Contains(err.Error(), "白名单") {
		t.Fatalf("白名单外标识错误信息应明确, 实际: %v", err)
	}

	// 合法任务可创建
	id, err := s.Create(superCtx(), validInput("valid-job", TargetClearLog, "0 0 3 * * *", StatusDisabled))
	if err != nil {
		t.Fatalf("合法任务创建失败: %v", err)
	}
	if id == 0 {
		t.Fatalf("创建成功应返回任务 ID")
	}
}

// TestNonSuperAdminForbidden 非超管的写操作一律 403, 管理员只读
func TestNonSuperAdminForbidden(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过定时任务集成测试")
	}
	s := New()
	id, err := s.Create(superCtx(), validInput("perm-job", TargetClearLog, "0 0 3 * * *", StatusDisabled))
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	defer unregister(id)

	admin := adminCtx()
	user := userCtx()

	_, err = s.Create(admin, validInput("perm-admin-create", TargetClearLog, "0 0 3 * * *", StatusDisabled))
	assertCode(t, err, int(consts.CodeForbidden.Code()))
	_, err = s.Create(user, validInput("perm-user-create", TargetClearLog, "0 0 3 * * *", StatusDisabled))
	assertCode(t, err, int(consts.CodeForbidden.Code()))

	err = s.Update(admin, UpdateInput{Id: id, CreateInput: validInput("perm-job", TargetClearLog, "0 0 3 * * *", StatusDisabled)})
	assertCode(t, err, int(consts.CodeForbidden.Code()))

	err = s.ChangeStatus(user, id, StatusEnabled)
	assertCode(t, err, int(consts.CodeForbidden.Code()))

	err = s.Run(admin, id)
	assertCode(t, err, int(consts.CodeForbidden.Code()))

	err = s.Delete(user, id)
	assertCode(t, err, int(consts.CodeForbidden.Code()))

	// 管理员可读列表/日志
	if _, err = s.List(admin, ListInput{Page: 1, PageSize: 10, Status: -1}); err != nil {
		t.Fatalf("管理员应可查看任务列表: %v", err)
	}
	if _, err = s.Logs(admin, LogListInput{Page: 1, PageSize: 10, Status: -1}); err != nil {
		t.Fatalf("管理员应可查看任务日志: %v", err)
	}
	// 普通用户/未登录不可读
	_, err = s.List(user, ListInput{Page: 1, PageSize: 10, Status: -1})
	assertCode(t, err, int(consts.CodeForbidden.Code()))
	_, err = s.List(anonCtx(), ListInput{Page: 1, PageSize: 10, Status: -1})
	assertCode(t, err, int(consts.CodeUnauthorized.Code()))
}

// TestManualRunWritesLog 手动触发异步写入 sys_job_log
func TestManualRunWritesLog(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过定时任务集成测试")
	}
	s := New()
	id, err := s.Create(superCtx(), validInput("manual-clearlog", TargetClearLog, "0 0 3 * * *", StatusEnabled))
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	t.Cleanup(func() { unregister(id) })

	if err = s.Run(superCtx(), id); err != nil {
		t.Fatalf("手动执行任务失败: %v", err)
	}

	cols := dao.SysJobLog.Columns()
	var logRow entity.SysJobLog
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		logRow = entity.SysJobLog{}
		if err = dao.SysJobLog.Ctx(testutil.Ctx()).
			Where(cols.JobId, id).
			Where(cols.TriggerType, TriggerManual).
			Scan(&logRow); err == nil && logRow.Id > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if logRow.Id == 0 {
		t.Fatalf("手动触发后未写入 sys_job_log")
	}
	if logRow.Status != RunSuccess {
		t.Fatalf("手动执行应成功, 实际 status=%d msg=%s", logRow.Status, logRow.Message)
	}
	if !strings.Contains(logRow.Message, "登录日志") || !strings.Contains(logRow.Message, "操作日志") {
		t.Fatalf("执行日志应包含清理条数, 实际: %s", logRow.Message)
	}

	// 最近执行结果应回写
	var jobRow entity.SysJob
	if err = dao.SysJob.Ctx(testutil.Ctx()).Where(dao.SysJob.Columns().Id, id).Scan(&jobRow); err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if jobRow.LastRunAt == nil {
		t.Fatalf("last_run_at 应被回写")
	}
	if jobRow.LastResult != "成功" {
		t.Fatalf("last_result 应为成功, 实际 %s", jobRow.LastResult)
	}
}

// TestDisabledJobNotScheduled 停用任务不会被调度, 手动触发返回 400
func TestDisabledJobNotScheduled(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过定时任务集成测试")
	}
	var counter atomic.Int64
	registerTestTarget(t, "test_disabled", func(ctx context.Context) (string, error) {
		counter.Add(1)
		return "ok", nil
	})

	s := New()
	id, err := s.Create(superCtx(), validInput("disabled-tick", "test_disabled", "* * * * * *", StatusDisabled))
	if err != nil {
		t.Fatalf("创建停用任务失败: %v", err)
	}
	t.Cleanup(func() { unregister(id) })

	// 停用任务不注册调度, 等待两个周期也不应执行
	time.Sleep(2200 * time.Millisecond)
	if counter.Load() != 0 {
		t.Fatalf("停用任务不应被调度, 实际执行 %d 次", counter.Load())
	}

	err = s.Run(superCtx(), id)
	assertCode(t, err, int(consts.CodeBadRequest.Code()))
	if !strings.Contains(err.Error(), "已停用") {
		t.Fatalf("停用任务手动触发应提示已停用, 实际: %v", err)
	}
}

// TestReloadNoDuplicate 重复 reload 不应造成同一任务被多次执行
func TestReloadNoDuplicate(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过定时任务集成测试")
	}
	var counter atomic.Int64
	registerTestTarget(t, "test_tick", func(ctx context.Context) (string, error) {
		counter.Add(1)
		return "ok", nil
	})

	s := New()
	// 允许并发 + 每秒执行的 6 段表达式
	in := validInput("tick-job", "test_tick", "* * * * * *", StatusEnabled)
	in.ConcurrencyPolicy = ConcurrencyAllow
	in.TimeoutSeconds = 10
	id, err := s.Create(superCtx(), in)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	t.Cleanup(func() { unregister(id) })

	// 单次注册: 2.2 秒内应执行约 2 次(而不是 8 次)
	time.Sleep(2200 * time.Millisecond)
	first := counter.Load()
	if first == 0 {
		t.Fatalf("任务应至少执行一次, 实际 %d", first)
	}
	if first > 4 {
		t.Fatalf("单次注册执行次数异常, 实际 %d 次(疑似重复注册)", first)
	}

	// 重复 reload 3 次后, 执行次数应仍然是"每秒一次"
	for i := 0; i < 3; i++ {
		if err = s.reload(superCtx(), id); err != nil {
			t.Fatalf("reload 失败: %v", err)
		}
	}
	time.Sleep(2200 * time.Millisecond)
	delta := counter.Load() - first
	if delta > 4 {
		t.Fatalf("重复 reload 后执行次数异常, delta=%d, 疑似注册了多个 entry", delta)
	}
}

// TestClearLogRetention clearlog 按保留天数删除旧日志、保留新日志
func TestClearLogRetention(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过定时任务集成测试")
	}
	ctx := testutil.Ctx()
	oldTime := gtime.New(time.Now().AddDate(0, 0, -100))
	newTime := gtime.Now()

	loginCols := dao.SysLoginLog.Columns()
	opCols := dao.SysOperationLog.Columns()

	if _, err := dao.SysLoginLog.Ctx(ctx).Data(do.SysLoginLog{
		Username: "old_user", Status: 1, Message: "old", CreatedAt: oldTime,
	}).Insert(); err != nil {
		t.Fatalf("造登录日志失败: %v", err)
	}
	if _, err := dao.SysLoginLog.Ctx(ctx).Data(do.SysLoginLog{
		Username: "new_user", Status: 1, Message: "new", CreatedAt: newTime,
	}).Insert(); err != nil {
		t.Fatalf("造登录日志失败: %v", err)
	}
	if _, err := dao.SysOperationLog.Ctx(ctx).Data(do.SysOperationLog{
		Username: "old_user", Module: "test", Action: "test", Status: 1, CreatedAt: oldTime,
	}).Insert(); err != nil {
		t.Fatalf("造操作日志失败: %v", err)
	}
	if _, err := dao.SysOperationLog.Ctx(ctx).Data(do.SysOperationLog{
		Username: "new_user", Module: "test", Action: "test", Status: 1, CreatedAt: newTime,
	}).Insert(); err != nil {
		t.Fatalf("造操作日志失败: %v", err)
	}

	message, err := clearLog(ctx)
	if err != nil {
		t.Fatalf("清理日志失败: %v", err)
	}
	if !strings.Contains(message, "登录日志 1 条") || !strings.Contains(message, "操作日志 1 条") {
		t.Fatalf("清理结果应包含具体条数, 实际: %s", message)
	}

	oldLogin, err := dao.SysLoginLog.Ctx(ctx).Where(loginCols.Username, "old_user").Count()
	if err != nil {
		t.Fatalf("查询登录日志失败: %v", err)
	}
	newLogin, err := dao.SysLoginLog.Ctx(ctx).Where(loginCols.Username, "new_user").Count()
	if err != nil {
		t.Fatalf("查询登录日志失败: %v", err)
	}
	if oldLogin != 0 || newLogin != 1 {
		t.Fatalf("登录日志保留策略不符: old=%d new=%d", oldLogin, newLogin)
	}
	oldOp, err := dao.SysOperationLog.Ctx(ctx).Where(opCols.Username, "old_user").Count()
	if err != nil {
		t.Fatalf("查询操作日志失败: %v", err)
	}
	newOp, err := dao.SysOperationLog.Ctx(ctx).Where(opCols.Username, "new_user").Count()
	if err != nil {
		t.Fatalf("查询操作日志失败: %v", err)
	}
	if oldOp != 0 || newOp != 1 {
		t.Fatalf("操作日志保留策略不符: old=%d new=%d", oldOp, newOp)
	}
}
