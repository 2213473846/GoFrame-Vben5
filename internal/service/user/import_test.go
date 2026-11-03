package user

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/identity"
	rolesvc "demo/internal/service/role"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/xuri/excelize/v2"
)

func superCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 1, Role: consts.RoleSuperAdmin, Username: "root"})
}

func adminCtx(id uint64) context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: id, Role: consts.RoleAdmin, Username: "agent"})
}

func userCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 99, Role: consts.RoleUser, Username: "user"})
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

// buildBook 生成 xlsx 字节流
func buildBook(t *testing.T, rows [][]string) []byte {
	t.Helper()
	book := excelize.NewFile()
	defer func() { _ = book.Close() }()
	sheet := book.GetSheetName(0)
	for i, row := range rows {
		for j, cell := range row {
			axis, _ := excelize.CoordinatesToCellName(j+1, i+1)
			if err := book.SetCellStr(sheet, axis, cell); err != nil {
				t.Fatalf("生成测试文件失败: %v", err)
			}
		}
	}
	buf, err := book.WriteToBuffer()
	if err != nil {
		t.Fatalf("生成测试文件失败: %v", err)
	}
	return buf.Bytes()
}

// prepareRoles 准备两个功能角色: ops(无菜单, 可被管理员分配) 与 sys_x(关联管理员无权菜单, 不可分配)
func prepareRoles(t *testing.T, ctx context.Context) (opsCode, closedCode string) {
	t.Helper()
	roleCols := dao.SysRole.Columns()
	for _, item := range []struct {
		code, name string
		menuId     uint64
	}{
		{"ops", "运营角色", 0},
		{"sys_x", "系统角色", 9999},
	} {
		count, err := dao.SysRole.Ctx(ctx).Where(roleCols.Code, item.code).Count()
		if err != nil {
			t.Fatalf("查询角色失败: %v", err)
		}
		if count > 0 {
			continue
		}
		id, err := dao.SysRole.Ctx(ctx).Data(do.SysRole{
			Code: item.code, Name: item.name, Status: 1, Remark: "ut",
		}).InsertAndGetId()
		if err != nil {
			t.Fatalf("创建角色失败: %v", err)
		}
		if item.menuId > 0 {
			if _, err = dao.SysRoleMenu.Ctx(ctx).Data(do.SysRoleMenu{
				RoleId: uint64(id), MenuId: item.menuId,
			}).Insert(); err != nil {
				t.Fatalf("创建角色菜单失败: %v", err)
			}
		}
	}
	return "ops", "sys_x"
}

// TestImportPartialSuccess 正常导入 + 各类校验失败 + 行号与原因准确
func TestImportPartialSuccess(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ctx := superCtx()
	opsCode, _ := prepareRoles(t, ctx)
	s := New()

	base, err := s.ImportTemplate(ctx)
	if err != nil {
		t.Fatalf("获取模板失败: %v", err)
	}
	rows := append([][]string{}, base...)
	rows = append(rows,
		[]string{"imp_ok", "张三", "Abc123456", "user", "启用", "2", opsCode},
		[]string{"imp_second", "李四", "Abc123456", "user", "禁用", "", ""},
		[]string{"imp_ok", "重复用户名", "Abc123456", "user", "启用", "", ""},
		[]string{"imp_badrole", "王五", "Abc123456", "guest", "启用", "", ""},
		[]string{"imp_weak", "赵六", "123", "user", "启用", "", ""},
		[]string{"imp_badstatus", "钱七", "Abc123456", "user", "9", "", ""},
	)
	book := buildBook(t, rows)
	result, err := s.Import(ctx, bytes.NewReader(book), int64(len(book)), "users.xlsx", false)
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if result.Total != 6 || result.Success != 2 || result.Failed != 4 {
		t.Fatalf("统计异常: total=%d success=%d failed=%d", result.Total, result.Success, result.Failed)
	}
	// 数据类型: 第 4 行成功, 第 5 行成功, 失败行 6/7/8/9
	wantRows := []int{6, 7, 8, 9}
	for i, item := range result.Errors {
		if i < len(wantRows) && item.Row != wantRows[i] {
			t.Fatalf("第 %d 个失败行行号应为 %d, 实际 %d(%s)", i, wantRows[i], item.Row, item.Reason)
		}
	}
	reasonMap := map[int]string{}
	for _, item := range result.Errors {
		reasonMap[item.Row] = item.Reason
	}
	if !strings.Contains(reasonMap[6], "文件内重复") {
		t.Fatalf("第 6 行应提示文件内重复, 实际: %s", reasonMap[6])
	}
	if !strings.Contains(reasonMap[7], "角色非法") {
		t.Fatalf("第 7 行应提示角色非法, 实际: %s", reasonMap[7])
	}
	if !strings.Contains(reasonMap[8], "密码不符合安全策略") {
		t.Fatalf("第 8 行应提示密码策略, 实际: %s", reasonMap[8])
	}
	if !strings.Contains(reasonMap[9], "状态非法") {
		t.Fatalf("第 9 行应提示状态非法, 实际: %s", reasonMap[9])
	}

	cols := dao.SysUser.Columns()
	var created entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(cols.Username, "imp_ok").Scan(&created); err != nil {
		t.Fatalf("查询导入用户失败: %v", err)
	}
	if created.Id == 0 {
		t.Fatal("imp_ok 未写入")
	}
	if created.MaxDevices != 2 {
		t.Fatalf("超管设置的最大设备数应为 2, 实际 %d", created.MaxDevices)
	}
	if created.MustChangePassword != 1 {
		t.Fatal("导入用户应强制首次改密")
	}
	roles, err := (rolesvc.New()).UserRoles(ctx, []uint64{created.Id})
	if err != nil {
		t.Fatalf("查询用户角色失败: %v", err)
	}
	if len(roles[created.Id]) != 1 || roles[created.Id][0].Code != opsCode {
		t.Fatalf("功能角色未正确分配: %+v", roles[created.Id])
	}
	// 状态列生效
	var disabled entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(cols.Username, "imp_second").Scan(&disabled); err != nil {
		t.Fatalf("查询导入用户失败: %v", err)
	}
	if disabled.Status != 0 {
		t.Fatalf("imp_second 状态应为禁用(0), 实际 %d", disabled.Status)
	}

	// 再次导入同名用户 -> 提示用户名已存在
	dupRows := append([][]string{}, base...)
	dupRows = append(dupRows, []string{"imp_ok", "张三", "Abc123456", "user", "启用", "", ""})
	dupBook := buildBook(t, dupRows)
	dupResult, err := s.Import(ctx, bytes.NewReader(dupBook), int64(len(dupBook)), "dup.xlsx", false)
	if err != nil {
		t.Fatalf("重复导入失败: %v", err)
	}
	if dupResult.Failed != 1 || !strings.Contains(dupResult.Errors[0].Reason, "用户名已存在") {
		t.Fatalf("库内重名应被拒绝: %+v", dupResult)
	}
	if dupResult.Errors[0].Row != 4 {
		t.Fatalf("库内重名行号应为 4, 实际 %d", dupResult.Errors[0].Row)
	}
}

// TestImportAdminScope 管理员: 只能创建名下用户, 越权角色/设备数列必须被拒
func TestImportAdminScope(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ctx := superCtx()
	opsCode, closedCode := prepareRoles(t, ctx)
	// 管理员账号(仅用于 pid 关联与可分配菜单范围)
	adminId, err := dao.SysUser.Ctx(ctx).Data(do.SysUser{
		Username: "imp_admin_x", Password: "x", RealName: "导入管理员", Role: consts.RoleAdmin, Status: 1,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("创建管理员失败: %v", err)
	}
	actor := adminCtx(uint64(adminId))
	s := New()

	base, err := s.ImportTemplate(actor)
	if err != nil {
		t.Fatalf("管理员获取模板失败: %v", err)
	}
	if len(base[0]) != 6 {
		t.Fatalf("管理员模板不应包含最大设备数列: %v", base[0])
	}
	rows := append([][]string{}, base...)
	rows = append(rows,
		[]string{"imp_admin_ok", "管理员下级", "Abc123456", "user", "启用", opsCode},
		[]string{"imp_admin_closed", "越权角色", "Abc123456", "user", "启用", closedCode},
	)
	book := buildBook(t, rows)
	result, err := s.Import(actor, bytes.NewReader(book), int64(len(book)), "admin.xlsx", false)
	if err != nil {
		t.Fatalf("管理员导入失败: %v", err)
	}
	if result.Success != 1 || result.Failed != 1 {
		t.Fatalf("管理员导入统计异常: %+v", result)
	}
	if !strings.Contains(result.Errors[0].Reason, "功能角色不存在或无权限分配") {
		t.Fatalf("越权角色应被拒绝, 实际: %s", result.Errors[0].Reason)
	}
	cols := dao.SysUser.Columns()
	var created entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(cols.Username, "imp_admin_ok").Scan(&created); err != nil {
		t.Fatalf("查询导入用户失败: %v", err)
	}
	if created.Pid != uint64(adminId) {
		t.Fatalf("管理员导入的用户 pid 应为自己(%d), 实际 %d", adminId, created.Pid)
	}

	// 管理员带"最大设备数"列并填值 -> 拒绝
	customHeaders := append([]string{}, base[0]...)
	customHeaders = append(customHeaders, "最大设备数")
	customBook := buildBook(t, [][]string{
		customHeaders,
		{"说明: 测试越权列", "", "", "", "", "", ""},
		{"imp_admin_dev", "越权设备", "Abc123456", "user", "启用", "", "3"},
	})
	customResult, err := s.Import(actor, bytes.NewReader(customBook), int64(len(customBook)), "admin_dev.xlsx", false)
	if err != nil {
		t.Fatalf("管理员导入失败: %v", err)
	}
	if customResult.Failed != 1 || !strings.Contains(customResult.Errors[0].Reason, "仅超级管理员可设置最大在线设备数") {
		t.Fatalf("管理员携带设备数列应被拒绝: %+v", customResult)
	}
	var notCreated entity.SysUser
	_ = dao.SysUser.Ctx(ctx).Where(cols.Username, "imp_admin_dev").Scan(&notCreated)
	if notCreated.Id != 0 {
		t.Fatal("越权行不应写入数据库(不能静默降级)")
	}
}

// TestImportAtomicRollback 全量回滚模式: 任一行失败则不写入任何用户
func TestImportAtomicRollback(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ctx := superCtx()
	opsCode, _ := prepareRoles(t, ctx)
	s := New()
	base, err := s.ImportTemplate(ctx)
	if err != nil {
		t.Fatalf("获取模板失败: %v", err)
	}
	rows := append([][]string{}, base...)
	rows = append(rows,
		[]string{"imp_atomic_ok", "原子一", "Abc123456", "user", "启用", "", opsCode},
		[]string{"imp_atomic_bad", "原子二", "123", "user", "启用", "", ""},
	)
	book := buildBook(t, rows)
	result, err := s.Import(ctx, bytes.NewReader(book), int64(len(book)), "atomic.xlsx", true)
	if err != nil {
		t.Fatalf("原子导入失败: %v", err)
	}
	if result.Success != 0 {
		t.Fatalf("全量回滚模式成功数应为 0, 实际 %d", result.Success)
	}
	if !result.Atomic {
		t.Fatal("结果应标记为原子模式")
	}
	cols := dao.SysUser.Columns()
	count, err := dao.SysUser.Ctx(ctx).Where(cols.Username, "imp_atomic_ok").Count()
	if err != nil {
		t.Fatalf("统计失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("全量回滚后不应写入任何用户, 实际存在 %d 条", count)
	}
}

// TestImportPermission 普通用户禁止导入
func TestImportPermission(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	s := New()
	_, err := s.ImportTemplate(userCtx())
	expectCode(t, "普通用户下载模板", err, 403)
	book := buildBook(t, [][]string{{"用户名", "姓名", "密码", "角色", "状态"}})
	_, err = s.Import(userCtx(), bytes.NewReader(book), int64(len(book)), "x.xlsx", false)
	expectCode(t, "普通用户导入", err, 403)
	_, err = s.Import(testutil.Ctx(), bytes.NewReader(book), int64(len(book)), "x.xlsx", false)
	expectCode(t, "未登录导入", err, 401)
}
