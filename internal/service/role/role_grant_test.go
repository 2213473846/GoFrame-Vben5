package role

import (
	"os"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/gconv"
)

// dbReady 本机 MySQL 可用时为 true, 否则跳过集成测试
var dbReady bool

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	code := m.Run()
	os.Exit(code)
}

// insertUser 插入测试用户并返回ID
func insertUser(t *testing.T, username string, pid uint64, role string) uint64 {
	t.Helper()
	id, err := dao.SysUser.Ctx(testutil.Ctx()).Data(do.SysUser{
		Pid: pid, Username: username, Password: "x", RealName: username,
		Role: role, Status: 1,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入用户 %s 失败: %v", username, err)
	}
	return gconv.Uint64(id)
}

// insertMenu 插入测试菜单并返回ID
func insertMenu(t *testing.T, name string, pid uint64) uint64 {
	t.Helper()
	id, err := dao.SysMenu.Ctx(testutil.Ctx()).Data(do.SysMenu{
		Pid: pid, Name: name, Path: "/" + name, Type: "menu", Title: name, Status: 1, Component: "/t/" + name,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入菜单 %s 失败: %v", name, err)
	}
	return gconv.Uint64(id)
}

// grantMenuToUser 给用户直授菜单
func grantMenuToUser(t *testing.T, userId, menuId uint64) {
	t.Helper()
	if _, err := dao.SysUserMenu.Ctx(testutil.Ctx()).Data(do.SysUserMenu{
		UserId: userId, MenuId: menuId,
	}).Insert(); err != nil {
		t.Fatalf("直授菜单失败: %v", err)
	}
}

// roleMenuIdsOf 查询角色的当前菜单ID集合
func roleMenuIdsOf(t *testing.T, roleId uint64) map[uint64]bool {
	t.Helper()
	var records []*entity.SysRoleMenu
	if err := dao.SysRoleMenu.Ctx(testutil.Ctx()).
		Where(dao.SysRoleMenu.Columns().RoleId, roleId).Scan(&records); err != nil {
		t.Fatalf("查询角色菜单失败: %v", err)
	}
	got := make(map[uint64]bool, len(records))
	for _, r := range records {
		got[r.MenuId] = true
	}
	return got
}

// expectCode 断言错误码
func expectCode(t *testing.T, name string, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 期望报错(code=%d), 实际成功", name, want)
	}
	if gerror.Code(err).Code() != want {
		t.Fatalf("%s: 期望错误码 %d, got %v", name, want, err)
	}
}

// TestGrantRoleScope 角色菜单授权的三级权限规则
func TestGrantRoleScope(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "rg-super", 0, consts.RoleSuperAdmin)
	admin := insertUser(t, "rg-admin", super, consts.RoleAdmin)
	user := insertUser(t, "rg-user", super, consts.RoleUser)

	m1 := insertMenu(t, "rg-m1", 0)
	m2 := insertMenu(t, "rg-m2", 0)
	// 超管只给管理员直授 m1
	grantMenuToUser(t, admin, m1)

	superCtx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})
	adminCtx := identity.With(testutil.Ctx(), identity.Identity{Id: admin, Role: consts.RoleAdmin})
	userCtx := identity.With(testutil.Ctx(), identity.Identity{Id: user, Role: consts.RoleUser})

	// 普通用户不可授权
	expectCode(t, "普通用户Grant", s.Grant(userCtx, 1, []uint64{m1}), int(consts.CodeForbidden.Code()))

	// 超管创建角色并授权 [m1, m2]
	roleId, err := s.Create(superCtx, CreateInput{Code: "rg-a", Name: "角色A", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	if err = s.Grant(superCtx, roleId, []uint64{m1, m2}); err != nil {
		t.Fatalf("超管给角色授权失败: %v", err)
	}
	if got := roleMenuIdsOf(t, roleId); len(got) != 2 || !got[m1] || !got[m2] {
		t.Fatalf("角色应授权 m1/m2, got %v", got)
	}

	// 管理员可操作含范围外菜单的角色: 范围内可改, 范围外保留(与用户授权规则一致)
	if err = s.Grant(adminCtx, roleId, []uint64{m1}); err != nil {
		t.Fatalf("管理员操作含范围外菜单的角色应成功: %v", err)
	}
	if got := roleMenuIdsOf(t, roleId); len(got) != 2 || !got[m1] || !got[m2] {
		t.Fatalf("管理员保存后应保留超管直授的 m2, got %v", got)
	}

	// 管理员创建只含 m1 的角色, 可正常转发授权
	roleB, err := s.Create(superCtx, CreateInput{Code: "rg-b", Name: "角色B", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	if err = s.Grant(superCtx, roleB, []uint64{m1}); err != nil {
		t.Fatalf("超管给角色授权失败: %v", err)
	}
	if err = s.Grant(adminCtx, roleB, []uint64{m1}); err != nil {
		t.Fatalf("管理员转发范围内菜单应成功: %v", err)
	}

	// 管理员授权超出自己范围的菜单 -> 403
	expectCode(t, "管理员授权范围外菜单", s.Grant(adminCtx, roleB, []uint64{m2}), int(consts.CodeForbidden.Code()))
}

// TestGrantRolePreserveSuperGranted 管理员保存角色授权时, 保留超管直授、自己无权管理的菜单
func TestGrantRolePreserveSuperGranted(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "rp-super", 0, consts.RoleSuperAdmin)
	admin := insertUser(t, "rp-admin", super, consts.RoleAdmin)

	m1 := insertMenu(t, "rp-m1", 0)
	m2 := insertMenu(t, "rp-m2", 0)
	// 超管仅给管理员直授 m1; m2 对管理员不可授权
	grantMenuToUser(t, admin, m1)

	superCtx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})
	adminCtx := identity.With(testutil.Ctx(), identity.Identity{Id: admin, Role: consts.RoleAdmin})

	// 超管给角色授权 [m1, m2]
	roleId, err := s.Create(superCtx, CreateInput{Code: "rp-a", Name: "角色A", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	if err = s.Grant(superCtx, roleId, []uint64{m1, m2}); err != nil {
		t.Fatalf("超管给角色授权失败: %v", err)
	}
	// 管理员仅下发自己可授权的 m1, 超管直授的 m2 应被保留
	if err = s.Grant(adminCtx, roleId, []uint64{m1}); err != nil {
		t.Fatalf("管理员保存角色授权失败: %v", err)
	}
	got := roleMenuIdsOf(t, roleId)
	if len(got) != 2 || !got[m1] || !got[m2] {
		t.Fatalf("管理员保存后应保留超管直授的 m2, got %v", got)
	}

	// 管理员提交范围外菜单(m2)时整体拒绝
	expectCode(t, "管理员提交范围外菜单", s.Grant(adminCtx, roleId, []uint64{m1, m2}), int(consts.CodeForbidden.Code()))
}

// TestSetUserRolesPreserveAdmin 管理员更新用户角色时, 强制保留超管分配的不可分配角色
func TestSetUserRolesPreserveAdmin(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "sp-super", 0, consts.RoleSuperAdmin)
	admin := insertUser(t, "sp-admin", super, consts.RoleAdmin)
	target := insertUser(t, "sp-target", admin, consts.RoleUser)

	m1 := insertMenu(t, "sp-m1", 0)
	m2 := insertMenu(t, "sp-m2", 0)
	// 超管仅给管理员直授 m1
	grantMenuToUser(t, admin, m1)

	superCtx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})
	adminCtx := identity.With(testutil.Ctx(), identity.Identity{Id: admin, Role: consts.RoleAdmin})

	// roleA: 菜单 m1 在管理员可分配范围内; roleB: 菜单 m2 超范围
	roleA, err := s.Create(superCtx, CreateInput{Code: "sp-a", Name: "角色A", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	roleB, err := s.Create(superCtx, CreateInput{Code: "sp-b", Name: "角色B", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	if err = s.Grant(superCtx, roleA, []uint64{m1}); err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	if err = s.Grant(superCtx, roleB, []uint64{m2}); err != nil {
		t.Fatalf("授权失败: %v", err)
	}

	// 超管给用户分配 roleB(管理员不可分配)
	if _, err = dao.SysUserRole.Ctx(testutil.Ctx()).Data(do.SysUserRole{
		UserId: target, RoleId: roleB,
	}).Insert(); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}

	// 管理员提交 [roleA](不包含 roleB): roleB 被强制保留
	if err = s.SetUserRoles(adminCtx, target, []uint64{roleA}); err != nil {
		t.Fatalf("管理员更新用户角色失败: %v", err)
	}
	var records []*entity.SysUserRole
	if err = dao.SysUserRole.Ctx(testutil.Ctx()).
		Where(dao.SysUserRole.Columns().UserId, target).Scan(&records); err != nil {
		t.Fatalf("查询用户角色失败: %v", err)
	}
	got := make(map[uint64]bool, len(records))
	for _, r := range records {
		got[r.RoleId] = true
	}
	if !got[roleA] || !got[roleB] || len(got) != 2 {
		t.Fatalf("管理员保存后应保留超管分配的 roleB, got %v", got)
	}

	// 管理员提交 [roleA, roleB](roleB 用户已有): 不算越权新增, roleB 保留
	if err = s.SetUserRoles(adminCtx, target, []uint64{roleA, roleB}); err != nil {
		t.Fatalf("提交用户已有的范围外角色不应报错: %v", err)
	}

	// 管理员提交用户没有的范围外角色 roleC: 越权拒绝
	roleC, err := s.Create(superCtx, CreateInput{Code: "sp-c", Name: "角色C", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	if err = s.Grant(superCtx, roleC, []uint64{m2}); err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	expectCode(t, "管理员新增范围外角色", s.SetUserRoles(adminCtx, target, []uint64{roleA, roleC}), int(consts.CodeForbidden.Code()))

	// 管理员清空: roleB 仍保留
	if err = s.SetUserRoles(adminCtx, target, []uint64{}); err != nil {
		t.Fatalf("管理员清空用户角色失败: %v", err)
	}
	records = nil
	if err = dao.SysUserRole.Ctx(testutil.Ctx()).
		Where(dao.SysUserRole.Columns().UserId, target).Scan(&records); err != nil {
		t.Fatalf("查询用户角色失败: %v", err)
	}
	if len(records) != 1 || records[0].RoleId != roleB {
		t.Fatalf("管理员清空后 roleB 仍应保留, got %+v", records)
	}

	// 超管可以真正移除 roleB
	if err = s.SetUserRoles(superCtx, target, []uint64{}); err != nil {
		t.Fatalf("超管清空用户角色失败: %v", err)
	}
	count, _ := dao.SysUserRole.Ctx(testutil.Ctx()).
		Where(dao.SysUserRole.Columns().UserId, target).Count()
	if count != 0 {
		t.Fatalf("超管清空后不应有角色关联, got %d", count)
	}
}

// TestListRoleVisibility 角色列表可见性: 超管全部; 管理员仅可分配的; 普通用户 403
func TestListRoleVisibility(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "lv-super", 0, consts.RoleSuperAdmin)
	admin := insertUser(t, "lv-admin", super, consts.RoleAdmin)
	user := insertUser(t, "lv-user", super, consts.RoleUser)

	m1 := insertMenu(t, "lv-m1", 0)
	m2 := insertMenu(t, "lv-m2", 0)
	grantMenuToUser(t, admin, m1)

	superCtx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})
	adminCtx := identity.With(testutil.Ctx(), identity.Identity{Id: admin, Role: consts.RoleAdmin})
	userCtx := identity.With(testutil.Ctx(), identity.Identity{Id: user, Role: consts.RoleUser})

	roleA, err := s.Create(superCtx, CreateInput{Code: "lv-a", Name: "角色A", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	roleB, err := s.Create(superCtx, CreateInput{Code: "lv-b", Name: "角色B", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	roleC, err := s.Create(superCtx, CreateInput{Code: "lv-c", Name: "角色C", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	// roleA: m1(管理员范围内); roleB: m2(范围外); roleC: 停用
	if err = s.Grant(superCtx, roleA, []uint64{m1}); err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	if err = s.Grant(superCtx, roleB, []uint64{m2}); err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	if _, err = dao.SysRole.Ctx(testutil.Ctx()).
		Where(dao.SysRole.Columns().Id, roleC).
		Data(do.SysRole{Status: 0}).Update(); err != nil {
		t.Fatalf("停用角色失败: %v", err)
	}

	// 超管: 本用例创建的 3 个角色全部可见
	roles, err := s.List(superCtx)
	if err != nil {
		t.Fatalf("超管查询角色列表失败: %v", err)
	}
	visible := make(map[string]bool, len(roles))
	for _, r := range roles {
		visible[r.Code] = true
	}
	for _, code := range []string{"lv-a", "lv-b", "lv-c"} {
		if !visible[code] {
			t.Fatalf("超管应可见角色 %s, 实际可见 %v", code, visible)
		}
	}

	// 管理员: 本用例中仅 roleA(lv-a, 菜单 m1 在范围内)可见;
	// roleB(lv-b, m2 超范围)与停用的 roleC(lv-c)不可见
	roles, err = s.List(adminCtx)
	if err != nil {
		t.Fatalf("管理员查询角色列表失败: %v", err)
	}
	visible = make(map[string]bool, len(roles))
	for _, r := range roles {
		visible[r.Code] = true
	}
	if !visible["lv-a"] {
		t.Fatalf("管理员应可见可分配角色 lv-a, 实际可见 %v", visible)
	}
	if visible["lv-b"] || visible["lv-c"] {
		t.Fatalf("管理员不应可见范围外/停用角色, 实际可见 %v", visible)
	}

	// 普通用户: 403
	_, err = s.List(userCtx)
	expectCode(t, "普通用户查询角色列表", err, int(consts.CodeForbidden.Code()))
}

// TestDeleteRoleCascade 删除角色时级联清理菜单授权与用户关联
func TestDeleteRoleCascade(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "dc-super", 0, consts.RoleSuperAdmin)
	target := insertUser(t, "dc-target", super, consts.RoleUser)
	m1 := insertMenu(t, "dc-m1", 0)

	superCtx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})
	roleId, err := s.Create(superCtx, CreateInput{Code: "dc-a", Name: "角色A", Status: 1})
	if err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}
	if err = s.Grant(superCtx, roleId, []uint64{m1}); err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	if _, err = dao.SysUserRole.Ctx(testutil.Ctx()).Data(do.SysUserRole{
		UserId: target, RoleId: roleId,
	}).Insert(); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}

	if err = s.Delete(superCtx, roleId); err != nil {
		t.Fatalf("删除角色失败: %v", err)
	}
	for name, count := range map[string]int64{
		"sys_role(软删除)":      0,
		"sys_role_menu(已清理)": 0,
		"sys_user_role(已清理)": 0,
	} {
		var n int
		switch name {
		case "sys_role(软删除)":
			n, _ = dao.SysRole.Ctx(testutil.Ctx()).Where(dao.SysRole.Columns().Id, roleId).Count()
		case "sys_role_menu(已清理)":
			n, _ = dao.SysRoleMenu.Ctx(testutil.Ctx()).Where(dao.SysRoleMenu.Columns().RoleId, roleId).Count()
		case "sys_user_role(已清理)":
			n, _ = dao.SysUserRole.Ctx(testutil.Ctx()).Where(dao.SysUserRole.Columns().RoleId, roleId).Count()
		}
		if n != int(count) {
			t.Fatalf("%s 期望 %d 条, got %d", name, count, n)
		}
	}
}
