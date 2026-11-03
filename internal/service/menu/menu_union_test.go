package menu

import (
	"os"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/util/gconv"
)

// dbReady 本机 MySQL 可用时为 true, 否则跳过集成测试
var dbReady bool

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	code := m.Run()
	os.Exit(code)
}

// ---------------- 纯逻辑测试(不依赖数据库) ----------------

// TestMergeMenuIds 菜单并集去重
func TestMergeMenuIds(t *testing.T) {
	cases := []struct {
		name     string
		direct   []uint64
		fromRole []uint64
		want     []uint64
	}{
		{"both empty", nil, nil, []uint64{}},
		{"direct only", []uint64{1, 2}, nil, []uint64{1, 2}},
		{"role only", nil, []uint64{3}, []uint64{3}},
		{"overlap dedup", []uint64{1, 2}, []uint64{2, 3}, []uint64{1, 2, 3}},
		{"full overlap", []uint64{5}, []uint64{5}, []uint64{5}},
		{"keeps direct order first", []uint64{9, 1}, []uint64{4, 7}, []uint64{9, 1, 4, 7}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mergeMenuIds(c.direct, c.fromRole)
			if len(got) != len(c.want) {
				t.Fatalf("mergeMenuIds(%v, %v) = %v, want %v", c.direct, c.fromRole, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("mergeMenuIds(%v, %v) = %v, want %v", c.direct, c.fromRole, got, c.want)
				}
			}
		})
	}
}

// TestWithAncestors 祖先补齐: 角色只授了子菜单时, 路由树需自动补齐父目录
func TestWithAncestors(t *testing.T) {
	s := New()
	all := []*entity.SysMenu{
		{Id: 1, Pid: 0, Title: "root"},
		{Id: 2, Pid: 1, Title: "child"},
		{Id: 3, Pid: 2, Title: "leaf"},
		{Id: 4, Pid: 0, Title: "other"},
	}
	picked := []*entity.SysMenu{all[2]} // 仅选中 leaf(id=3)
	got := s.withAncestors(all, picked)
	gotIds := make(map[uint64]bool, len(got))
	for _, m := range got {
		gotIds[m.Id] = true
	}
	// 3 的祖先链 2 -> 1 都应补齐, 无关的 4 不应出现
	for _, want := range []uint64{1, 2, 3} {
		if !gotIds[want] {
			t.Fatalf("withAncestors 缺少祖先节点 %d, got %v", want, gotIds)
		}
	}
	if gotIds[4] {
		t.Fatalf("withAncestors 不应包含无关节点 4")
	}
}

// ---------------- 集成测试(依赖本机 MySQL) ----------------

// insertTestMenu 插入测试菜单并返回ID
func insertTestMenu(t *testing.T, name string, pid uint64) uint64 {
	t.Helper()
	id, err := dao.SysMenu.Ctx(testutil.Ctx()).Data(do.SysMenu{
		Pid: pid, Name: name, Path: "/" + name, Type: TypeMenu,
		Title: name, Status: 1, Component: "/test/" + name,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入菜单 %s 失败: %v", name, err)
	}
	return gconv.Uint64(id)
}

// insertTestUser 插入测试用户并返回ID
func insertTestUser(t *testing.T, username string, pid uint64, role string) uint64 {
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

// insertTestRole 插入测试角色并返回ID
func insertTestRole(t *testing.T, code string, status int) uint64 {
	t.Helper()
	id, err := dao.SysRole.Ctx(testutil.Ctx()).Data(do.SysRole{
		Code: code, Name: code, Status: status,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入角色 %s 失败: %v", code, err)
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

// grantMenuToRole 给角色授权菜单
func grantMenuToRole(t *testing.T, roleId, menuId uint64) {
	t.Helper()
	if _, err := dao.SysRoleMenu.Ctx(testutil.Ctx()).Data(do.SysRoleMenu{
		RoleId: roleId, MenuId: menuId,
	}).Insert(); err != nil {
		t.Fatalf("角色授权菜单失败: %v", err)
	}
}

// assignRole 给用户分配角色
func assignRole(t *testing.T, userId, roleId uint64) {
	t.Helper()
	if _, err := dao.SysUserRole.Ctx(testutil.Ctx()).Data(do.SysUserRole{
		UserId: userId, RoleId: roleId,
	}).Insert(); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}
}

// assertIdsEqual 断言ID集合相等(忽略顺序)
func assertIdsEqual(t *testing.T, name string, got []uint64, want ...uint64) {
	t.Helper()
	gotSet := make(map[uint64]bool, len(got))
	for _, id := range got {
		gotSet[id] = true
	}
	if len(gotSet) != len(got) {
		t.Fatalf("%s 存在重复ID: %v", name, got)
	}
	if len(gotSet) != len(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	for _, w := range want {
		if !gotSet[w] {
			t.Fatalf("%s = %v, 缺少 %d", name, got, w)
		}
	}
}

// TestGrantedMenuIdsUnion 用户有效菜单 = 直授权 ∪ 启用角色的菜单
func TestGrantedMenuIdsUnion(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	ctx := testutil.Ctx()
	s := New()

	m1 := insertTestMenu(t, "union-m1", 0)
	m2 := insertTestMenu(t, "union-m2", 0)
	m3 := insertTestMenu(t, "union-m3", 0)

	// 用户A: 直授 m1; 角色R1(启用)授 m2; 角色R2(停用)授 m3
	userA := insertTestUser(t, "union-user-a", 0, consts.RoleUser)
	r1 := insertTestRole(t, "union-r1", 1)
	r2 := insertTestRole(t, "union-r2", 0)
	grantMenuToUser(t, userA, m1)
	grantMenuToRole(t, r1, m2)
	grantMenuToRole(t, r2, m3)
	assignRole(t, userA, r1)
	assignRole(t, userA, r2)

	ids, err := s.grantedMenuIds(ctx, userA)
	if err != nil {
		t.Fatalf("grantedMenuIds 失败: %v", err)
	}
	// m3 因角色停用不生效
	assertIdsEqual(t, "userA 有效菜单", ids, m1, m2)

	// 用户B: 无任何角色与直授, 行为与历史一致(空)
	userB := insertTestUser(t, "union-user-b", 0, consts.RoleUser)
	ids, err = s.grantedMenuIds(ctx, userB)
	if err != nil {
		t.Fatalf("grantedMenuIds 失败: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("无角色无直授的用户应返回空, got %v", ids)
	}

	// 用户C: 仅角色(启用)授 m2, 无直授
	userC := insertTestUser(t, "union-user-c", 0, consts.RoleUser)
	assignRole(t, userC, r1)
	ids, err = s.grantedMenuIds(ctx, userC)
	if err != nil {
		t.Fatalf("grantedMenuIds 失败: %v", err)
	}
	assertIdsEqual(t, "userC 有效菜单", ids, m2)

	// 用户D: 直授与角色菜单重叠, 去重
	userD := insertTestUser(t, "union-user-d", 0, consts.RoleUser)
	grantMenuToUser(t, userD, m2)
	assignRole(t, userD, r1)
	ids, err = s.grantedMenuIds(ctx, userD)
	if err != nil {
		t.Fatalf("grantedMenuIds 失败: %v", err)
	}
	assertIdsEqual(t, "userD 有效菜单", ids, m2)
}

// TestGrantableMenusAdminScope 管理员可授权范围 = 超管下发给自己的菜单(含角色带来的)
func TestGrantableMenusAdminScope(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	m1 := insertTestMenu(t, "scope-m1", 0)
	m2 := insertTestMenu(t, "scope-m2", 0)
	admin := insertTestUser(t, "scope-admin", 0, consts.RoleAdmin)
	// 超管只直授 m1 给管理员; m2 未下发
	grantMenuToUser(t, admin, m1)

	ctx := identity.With(testutil.Ctx(), identity.Identity{Id: admin, Role: consts.RoleAdmin})
	menus, err := s.grantableMenus(ctx, identity.Get(ctx))
	if err != nil {
		t.Fatalf("grantableMenus 失败: %v", err)
	}
	assertIdsEqual(t, "管理员可授权菜单", menuIdsOf(menus), m1)

	// 给管理员挂一个含 m2 的角色后, m2 也进入可授权范围
	r := insertTestRole(t, "scope-r", 1)
	grantMenuToRole(t, r, m2)
	assignRole(t, admin, r)
	menus, err = s.grantableMenus(ctx, identity.Get(ctx))
	if err != nil {
		t.Fatalf("grantableMenus 失败: %v", err)
	}
	assertIdsEqual(t, "挂角色后可授权菜单", menuIdsOf(menus), m1, m2)
}

// TestAllRoutesRoleMenusWithAncestors 角色只授子菜单时, All 自动补齐祖先目录
func TestAllRoutesRoleMenusWithAncestors(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	catalog := insertTestMenu(t, "all-catalog", 0)
	leaf := insertTestMenu(t, "all-leaf", catalog)
	insertTestMenu(t, "all-unrelated", 0)

	user := insertTestUser(t, "all-user", 0, consts.RoleUser)
	r := insertTestRole(t, "all-r", 1)
	// 角色仅授子菜单, 未授父目录
	grantMenuToRole(t, r, leaf)
	assignRole(t, user, r)

	ctx := identity.With(testutil.Ctx(), identity.Identity{Id: user, Role: consts.RoleUser})
	routes, err := s.All(ctx)
	if err != nil {
		t.Fatalf("All 失败: %v", err)
	}
	// 期望: catalog -> leaf 的两层树, 且不含 unrelated
	if len(routes) != 1 {
		t.Fatalf("期望根路由仅 1 个(父目录补齐), got %d: %+v", len(routes), routes)
	}
	if routes[0].Name != "all-catalog" {
		t.Fatalf("期望根路由为补齐的父目录, got %s", routes[0].Name)
	}
	if len(routes[0].Children) != 1 || routes[0].Children[0].Name != "all-leaf" {
		t.Fatalf("期望子路由为 all-leaf, got %+v", routes[0].Children)
	}
	for _, r := range routes {
		if r.Name == "all-unrelated" {
			t.Fatalf("不应包含未授权的 all-unrelated")
		}
	}
}

// menuIdsOf 提取菜单ID列表
func menuIdsOf(menus []*entity.SysMenu) []uint64 {
	ids := make([]uint64, 0, len(menus))
	for _, m := range menus {
		ids = append(ids, m.Id)
	}
	return ids
}
