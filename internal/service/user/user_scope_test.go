package user

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
	// 设备数上限变更会立即清理超出会话, 这里使用独立 Redis DB(12), 与其它测试包隔离
	_ = testutil.PrepareRedisDB(12)
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

// insertRole 插入测试角色并返回ID
func insertRole(t *testing.T, code string) uint64 {
	t.Helper()
	id, err := dao.SysRole.Ctx(testutil.Ctx()).Data(do.SysRole{
		Code: code, Name: code, Status: 1,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入角色 %s 失败: %v", code, err)
	}
	return gconv.Uint64(id)
}

// insertMenu 插入测试菜单并返回ID
func insertMenu(t *testing.T, name string) uint64 {
	t.Helper()
	id, err := dao.SysMenu.Ctx(testutil.Ctx()).Data(do.SysMenu{
		Name: name, Path: "/" + name, Type: "menu", Title: name, Status: 1, Component: "/t/" + name,
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

// TestListDataScope 三级权限数据可见范围:
// 超管可见全部; 管理员仅可见自己的子孙(不含自己); 普通用户无权访问
func TestListDataScope(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	// 层级: super -> admin1 -> user11, user12; super -> admin2 -> user21
	super := insertUser(t, "scope-super", 0, consts.RoleSuperAdmin)
	admin1 := insertUser(t, "scope-admin1", super, consts.RoleAdmin)
	user11 := insertUser(t, "scope-user11", admin1, consts.RoleUser)
	user12 := insertUser(t, "scope-user12", admin1, consts.RoleUser)
	admin2 := insertUser(t, "scope-admin2", super, consts.RoleAdmin)
	user21 := insertUser(t, "scope-user21", admin2, consts.RoleUser)

	listOf := func(actorId uint64, role string) map[uint64]bool {
		ctx := identity.With(testutil.Ctx(), identity.Identity{Id: actorId, Role: role})
		users, _, _, total, err := s.List(ctx, ListInput{Page: 1, PageSize: 100})
		if err != nil {
			t.Fatalf("List 失败: %v", err)
		}
		if total != len(users) {
			t.Fatalf("total(%d) 与返回条数(%d) 不一致", total, len(users))
		}
		got := make(map[uint64]bool, len(users))
		for _, u := range users {
			got[u.Id] = true
		}
		return got
	}

	// 超管: 全部 6 人
	got := listOf(super, consts.RoleSuperAdmin)
	for _, id := range []uint64{super, admin1, user11, user12, admin2, user21} {
		if !got[id] {
			t.Fatalf("超管应可见用户 %d, 实际可见 %v", id, got)
		}
	}

	// 管理员1: 仅自己的子孙(user11, user12), 不含自己/admin2/user21
	got = listOf(admin1, consts.RoleAdmin)
	if len(got) != 2 || !got[user11] || !got[user12] {
		t.Fatalf("admin1 应仅可见 user11/user12, 实际可见 %v", got)
	}

	// 管理员2: 仅 user21
	got = listOf(admin2, consts.RoleAdmin)
	if len(got) != 1 || !got[user21] {
		t.Fatalf("admin2 应仅可见 user21, 实际可见 %v", got)
	}

	// 普通用户: 403
	ctx := identity.With(testutil.Ctx(), identity.Identity{Id: user11, Role: consts.RoleUser})
	_, _, _, _, err := s.List(ctx, ListInput{Page: 1, PageSize: 10})
	if err == nil {
		t.Fatalf("普通用户访问用户列表应被拒绝")
	}
	if gerror.Code(err).Code() != int(consts.CodeForbidden.Code()) {
		t.Fatalf("普通用户访问用户列表应返回 403, got %v", err)
	}
}

// TestListReturnsUserRoles 用户列表返回功能角色信息
func TestListReturnsUserRoles(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "urole-super", 0, consts.RoleSuperAdmin)
	target := insertUser(t, "urole-target", super, consts.RoleUser)
	r1 := insertRole(t, "urole-r1")
	r2 := insertRole(t, "urole-r2")
	for _, rid := range []uint64{r1, r2} {
		if _, err := dao.SysUserRole.Ctx(testutil.Ctx()).Data(do.SysUserRole{
			UserId: target, RoleId: rid,
		}).Insert(); err != nil {
			t.Fatalf("分配角色失败: %v", err)
		}
	}

	ctx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})
	users, _, userRoles, _, err := s.List(ctx, ListInput{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	var targetRoles []uint64
	for _, u := range users {
		if u.Id == target {
			for _, r := range userRoles[u.Id] {
				targetRoles = append(targetRoles, r.Id)
			}
		}
	}
	if len(targetRoles) != 2 {
		t.Fatalf("用户应关联 2 个功能角色, got %v", targetRoles)
	}
}

// TestCreateUserRoleScope 管理员创建用户时, 只能分配"自己可分配"的角色
func TestCreateUserRoleScope(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "crole-super", 0, consts.RoleSuperAdmin)
	admin := insertUser(t, "crole-admin", super, consts.RoleAdmin)

	m1 := insertMenu(t, "crole-m1")
	m2 := insertMenu(t, "crole-m2")
	// 超管只给管理员直授 m1
	grantMenuToUser(t, admin, m1)

	// 角色A: 菜单都在管理员的可授权范围内(m1)
	roleA := insertRole(t, "crole-a")
	if _, err := dao.SysRoleMenu.Ctx(testutil.Ctx()).Data(do.SysRoleMenu{
		RoleId: roleA, MenuId: m1,
	}).Insert(); err != nil {
		t.Fatalf("角色授权失败: %v", err)
	}
	// 角色B: 菜单超出管理员的可授权范围(m2)
	roleB := insertRole(t, "crole-b")
	if _, err := dao.SysRoleMenu.Ctx(testutil.Ctx()).Data(do.SysRoleMenu{
		RoleId: roleB, MenuId: m2,
	}).Insert(); err != nil {
		t.Fatalf("角色授权失败: %v", err)
	}

	// 管理员分配范围外角色 -> 403
	adminCtx := identity.With(testutil.Ctx(), identity.Identity{Id: admin, Role: consts.RoleAdmin})
	_, err := s.Create(adminCtx, CreateInput{
		Username: "crole-u1", Password: "123456", Role: consts.RoleUser, RoleIds: []uint64{roleB},
	})
	if err == nil {
		t.Fatalf("管理员分配超出自己可授权范围的角色应被拒绝")
	}
	if gerror.Code(err).Code() != int(consts.CodeForbidden.Code()) {
		t.Fatalf("期望 403, got %v", err)
	}

	// 管理员分配范围内角色 -> 成功
	id, err := s.Create(adminCtx, CreateInput{
		Username: "crole-u2", Password: "123456", Role: consts.RoleUser, RoleIds: []uint64{roleA},
	})
	if err != nil {
		t.Fatalf("管理员分配范围内角色应成功: %v", err)
	}
	count, err := dao.SysUserRole.Ctx(testutil.Ctx()).
		Where(dao.SysUserRole.Columns().UserId, id).Count()
	if err != nil {
		t.Fatalf("查询用户角色失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("用户应关联 1 个角色, got %d", count)
	}

	// 超管分配任意启用角色 -> 成功
	superCtx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})
	if _, err = s.Create(superCtx, CreateInput{
		Username: "crole-u3", Password: "123456", Role: consts.RoleUser, RoleIds: []uint64{roleA, roleB},
	}); err != nil {
		t.Fatalf("超管分配任意角色应成功: %v", err)
	}
}

// TestUpdateUserRoleOverwrite 更新用户时 roleIds 全量覆盖, 不传则不修改
func TestUpdateUserRoleOverwrite(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "urole2-super", 0, consts.RoleSuperAdmin)
	target := insertUser(t, "urole2-target", super, consts.RoleUser)
	r1 := insertRole(t, "urole2-r1")
	r2 := insertRole(t, "urole2-r2")
	if _, err := dao.SysUserRole.Ctx(testutil.Ctx()).Data(do.SysUserRole{
		UserId: target, RoleId: r1,
	}).Insert(); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}
	superCtx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})

	// 不传 roleIds(nil): 原有角色不变
	if err := s.Update(superCtx, UpdateInput{Id: target, RealName: "新昵称"}); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	count, _ := dao.SysUserRole.Ctx(testutil.Ctx()).
		Where(dao.SysUserRole.Columns().UserId, target).Count()
	if count != 1 {
		t.Fatalf("不传 roleIds 时不应修改角色关联, got %d", count)
	}

	// 传空数组: 清空角色
	empty := []uint64{}
	if err := s.Update(superCtx, UpdateInput{Id: target, RoleIds: &empty}); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	count, _ = dao.SysUserRole.Ctx(testutil.Ctx()).
		Where(dao.SysUserRole.Columns().UserId, target).Count()
	if count != 0 {
		t.Fatalf("空 roleIds 应清空角色关联, got %d", count)
	}

	// 传新数组: 覆盖为 r2
	if err := s.Update(superCtx, UpdateInput{Id: target, RoleIds: &[]uint64{r2}}); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	var records []*entity.SysUserRole
	if err := dao.SysUserRole.Ctx(testutil.Ctx()).
		Where(dao.SysUserRole.Columns().UserId, target).Scan(&records); err != nil {
		t.Fatalf("查询角色关联失败: %v", err)
	}
	if len(records) != 1 || gconv.Uint64(records[0].RoleId) != r2 {
		t.Fatalf("角色关联应覆盖为 r2, got %+v", records)
	}
}
