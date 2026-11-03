package user

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/util/gconv"
	"github.com/xuri/excelize/v2"
)

func actorCtx(id uint64, pid uint64, role string) context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: id, Pid: pid, Role: role, Username: "u" + gconv.String(id)})
}

// TestUserExportScopeAndFilter 导出复用列表查询与数据权限: 内容一致、条件生效、管理员仅子孙
func TestUserExportScopeAndFilter(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	rootId := insertUser(t, "exp_root", 0, consts.RoleSuperAdmin)
	adminId := insertUser(t, "exp_admin", rootId, consts.RoleAdmin)
	childId := insertUser(t, "exp_child", adminId, consts.RoleUser)
	grandChildId := insertUser(t, "exp_grandchild", childId, consts.RoleUser)
	disabledId := insertUser(t, "exp_disabled", adminId, consts.RoleUser)
	outsiderId := insertUser(t, "exp_outsider", rootId, consts.RoleUser)
	// 停用一个子孙用户, 用于验证条件筛选
	if _, err := dao.SysUser.Ctx(testutil.Ctx()).Data(do.SysUser{Status: 0}).Where(dao.SysUser.Columns().Id, disabledId).Update(); err != nil {
		t.Fatalf("更新用户状态失败: %v", err)
	}
	_ = grandChildId

	s := New()
	adminCtx := actorCtx(adminId, rootId, consts.RoleAdmin)

	// 列表(管理员)结果
	list, _, _, total, err := s.List(adminCtx, ListInput{Username: "exp_", Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("管理员查询列表失败: %v", err)
	}
	if total != 3 {
		t.Fatalf("管理员列表应为 3 条(子/孙/停用子), 实际 %d", total)
	}

	// 导出(管理员)内容应与列表一致, 且不含范围外用户
	var buf bytes.Buffer
	count, err := s.Export(adminCtx, ListInput{Username: "exp_"}, "csv", &buf)
	if err != nil {
		t.Fatalf("管理员导出失败: %v", err)
	}
	if count != total {
		t.Fatalf("导出条数 %d 与列表条数 %d 不一致", count, total)
	}
	rows := parseCSV(t, &buf)
	exported := make(map[string]bool, len(rows))
	for _, row := range rows {
		exported[row[1]] = true
	}
	for _, u := range list {
		if !exported[u.Username] {
			t.Fatalf("列表用户 %s 未出现在导出结果中", u.Username)
		}
	}
	if exported["exp_admin"] || exported["exp_outsider"] || exported["exp_self"] {
		t.Fatalf("管理员导出越权, 结果包含范围外用户: %v", exported)
	}
	if !exported["exp_child"] || !exported["exp_grandchild"] || !exported["exp_disabled"] {
		t.Fatalf("管理员导出缺少子孙用户: %v", exported)
	}

	// 条件筛选生效: status=1 时排除停用用户
	status := 1
	buf.Reset()
	count, err = s.Export(adminCtx, ListInput{Username: "exp_", Status: &status}, "csv", &buf)
	if err != nil {
		t.Fatalf("带条件导出失败: %v", err)
	}
	if count != 2 {
		t.Fatalf("status=1 导出应为 2 条, 实际 %d", count)
	}
	rows = parseCSV(t, &buf)
	for _, row := range rows {
		if row[4] != "启用" {
			t.Fatalf("状态筛选未生效, 出现非启用用户: %v", row)
		}
	}

	// 超管可导出全部(仍受条件约束)
	buf.Reset()
	rootCtx := actorCtx(rootId, 0, consts.RoleSuperAdmin)
	count, err = s.Export(rootCtx, ListInput{Username: "exp_"}, "xlsx", &buf)
	if err != nil {
		t.Fatalf("超管导出 xlsx 失败: %v", err)
	}
	if count != 6 {
		t.Fatalf("超管导出应为 6 条, 实际 %d", count)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("PK")) {
		t.Fatal("xlsx 导出内容不是合法的 zip/excel 文件")
	}
	// 用 excelize 重新打开, 确认文件结构可用且行数正确(表头 + 6 条数据)
	book, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("xlsx 文件无法打开: %v", err)
	}
	defer func() { _ = book.Close() }()
	sheetRows, err := book.GetRows(book.GetSheetName(0))
	if err != nil {
		t.Fatalf("xlsx 读取失败: %v", err)
	}
	if len(sheetRows) != 7 {
		t.Fatalf("xlsx 行数应为 7(含表头), 实际 %d", len(sheetRows))
	}

	// 普通用户不可导出
	userCtx := actorCtx(outsiderId, rootId, consts.RoleUser)
	if _, err = s.Export(userCtx, ListInput{}, "csv", &bytes.Buffer{}); err == nil {
		t.Fatal("普通用户导出应被拒绝")
	}
}

func parseCSV(t *testing.T, buf *bytes.Buffer) [][]string {
	t.Helper()
	reader := csv.NewReader(strings.NewReader(buf.String()))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("解析 CSV 失败: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("CSV 内容为空")
	}
	return rows[1:]
}
