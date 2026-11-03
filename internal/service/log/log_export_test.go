package log

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

var dbReady bool

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	code := m.Run()
	os.Exit(code)
}

func insertUser(t *testing.T, username string, pid uint64, role string) uint64 {
	t.Helper()
	id, err := dao.SysUser.Ctx(testutil.Ctx()).Data(do.SysUser{
		Pid: pid, Username: username, Password: "x", RealName: username, Role: role, Status: 1,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入用户 %s 失败: %v", username, err)
	}
	return gconv.Uint64(id)
}

func logActorCtx(id uint64, role string) context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: id, Role: role, Username: "u" + gconv.String(id)})
}

func parseCSVRows(t *testing.T, buf *bytes.Buffer) [][]string {
	t.Helper()
	rows, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatalf("解析 CSV 失败: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("CSV 内容为空")
	}
	return rows[1:]
}

// TestLogExportScopeAndFilter 登录/操作日志导出复用列表数据范围与筛选条件
func TestLogExportScopeAndFilter(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	rootId := insertUser(t, "log_root", 0, consts.RoleSuperAdmin)
	adminId := insertUser(t, "log_admin", rootId, consts.RoleAdmin)
	childId := insertUser(t, "log_child", adminId, consts.RoleUser)
	outsiderId := insertUser(t, "log_outsider", rootId, consts.RoleUser)
	_ = rootId
	_ = outsiderId

	// 写入登录日志: admin 2 条(1 成功 1 失败), child 1 条, outsider 1 条
	now := gtime.Now()
	loginLogs := []do.SysLoginLog{
		{UserId: adminId, Username: "log_admin", Status: 1, Message: "登录成功", Ip: "127.0.0.1", CreatedAt: now},
		{UserId: adminId, Username: "log_admin", Status: 0, Message: "密码错误", Ip: "127.0.0.1", CreatedAt: now},
		{UserId: childId, Username: "log_child", Status: 1, Message: "登录成功", Ip: "127.0.0.1", CreatedAt: now},
		{UserId: outsiderId, Username: "log_outsider", Status: 1, Message: "登录成功", Ip: "127.0.0.1", CreatedAt: now},
	}
	for _, item := range loginLogs {
		if _, err := dao.SysLoginLog.Ctx(testutil.Ctx()).Data(item).Insert(); err != nil {
			t.Fatalf("插入登录日志失败: %v", err)
		}
	}
	opLogs := []do.SysOperationLog{
		{UserId: adminId, Username: "log_admin", Module: "用户管理", Action: "新增用户", Method: "POST", Path: "/user/create", Status: 1, Message: "ok", Ip: "127.0.0.1", CreatedAt: now},
		{UserId: childId, Username: "log_child", Module: "用户管理", Action: "修改用户", Method: "PUT", Path: "/user/update", Status: 1, Message: "ok", Ip: "127.0.0.1", CreatedAt: now},
		{UserId: outsiderId, Username: "log_outsider", Module: "用户管理", Action: "删除用户", Method: "DELETE", Path: "/user/delete", Status: 1, Message: "ok", Ip: "127.0.0.1", CreatedAt: now},
	}
	for _, item := range opLogs {
		if _, err := dao.SysOperationLog.Ctx(testutil.Ctx()).Data(item).Insert(); err != nil {
			t.Fatalf("插入操作日志失败: %v", err)
		}
	}

	s := New()
	adminCtx := logActorCtx(adminId, consts.RoleAdmin)

	// 管理员登录日志列表: 自己 + 子孙 = 3 条
	list, err := s.LoginList(adminCtx, &LoginListInput{Status: -1, Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("管理员查询登录日志失败: %v", err)
	}
	if list.Total != 3 {
		t.Fatalf("管理员登录日志应为 3 条, 实际 %d", list.Total)
	}
	var buf bytes.Buffer
	count, err := s.LoginExport(adminCtx, &LoginListInput{Status: -1}, "csv", &buf)
	if err != nil {
		t.Fatalf("导出登录日志失败: %v", err)
	}
	if count != list.Total {
		t.Fatalf("登录日志导出条数 %d 与列表 %d 不一致", count, list.Total)
	}
	for _, row := range parseCSVRows(t, &buf) {
		if row[1] == "log_outsider" {
			t.Fatalf("管理员导出登录日志越权, 含范围外用户: %v", row)
		}
	}

	// 条件筛选: username=log_child
	buf.Reset()
	count, err = s.LoginExport(adminCtx, &LoginListInput{Username: "log_child", Status: -1}, "csv", &buf)
	if err != nil {
		t.Fatalf("条件导出登录日志失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("条件导出登录日志应为 1 条, 实际 %d", count)
	}

	// 操作日志: 列表 2 条(admin + child), 导出一致且不含 outsider
	opList, err := s.OperationList(adminCtx, &OperationListInput{Status: -1, Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("管理员查询操作日志失败: %v", err)
	}
	if opList.Total != 2 {
		t.Fatalf("管理员操作日志应为 2 条, 实际 %d", opList.Total)
	}
	buf.Reset()
	count, err = s.OperationExport(adminCtx, &OperationListInput{Status: -1}, "csv", &buf)
	if err != nil {
		t.Fatalf("导出操作日志失败: %v", err)
	}
	if count != opList.Total {
		t.Fatalf("操作日志导出条数 %d 与列表 %d 不一致", count, opList.Total)
	}
	for _, row := range parseCSVRows(t, &buf) {
		if row[1] == "log_outsider" {
			t.Fatalf("管理员导出操作日志越权, 含范围外用户: %v", row)
		}
	}

	// 普通用户仅能导出自己的日志
	userCtx := logActorCtx(outsiderId, consts.RoleUser)
	buf.Reset()
	count, err = s.LoginExport(userCtx, &LoginListInput{Status: -1}, "csv", &buf)
	if err != nil {
		t.Fatalf("普通用户导出登录日志失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("普通用户登录日志应为 1 条, 实际 %d", count)
	}
}
