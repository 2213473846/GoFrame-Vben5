package user

import (
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
)

func intPtr(v int) *int { return &v }

// maxDevicesOf 读取用户当前 max_devices, NULL 返回 nil
func maxDevicesOf(t *testing.T, id uint64) any {
	t.Helper()
	v, err := dao.SysUser.Ctx(testutil.Ctx()).
		Where(dao.SysUser.Columns().Id, id).
		Fields(dao.SysUser.Columns().MaxDevices).
		Value()
	if err != nil {
		t.Fatalf("查询 max_devices 失败: %v", err)
	}
	if v.IsNil() {
		return nil
	}
	return v.Int()
}

// TestAdminCannotUpdateMaxDevices 管理员不能借更新接口顺带修改 max_devices(越权点)
func TestAdminCannotUpdateMaxDevices(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "dev-super", 0, consts.RoleSuperAdmin)
	admin := insertUser(t, "dev-admin", super, consts.RoleAdmin)
	target := insertUser(t, "dev-target", admin, consts.RoleUser)

	adminCtx := identity.With(testutil.Ctx(), identity.Identity{Id: admin, Role: consts.RoleAdmin})

	// 顺带携带 max_devices 与普通字段一起提交, 必须整体拒绝且不落库
	if err := s.Update(adminCtx, UpdateInput{
		Id: target, RealName: "越权改名", MaxDevices: intPtr(3),
	}); err == nil {
		t.Fatalf("管理员修改 max_devices 应被拒绝")
	} else if gerror.Code(err).Code() != int(consts.CodeForbidden.Code()) {
		t.Fatalf("期望 403, got %v", err)
	}
	if got := maxDevicesOf(t, target); got != nil {
		t.Fatalf("管理员越权修改后 max_devices 不应被写入, got %v", got)
	}
	// 同一请求中的普通字段也不应生效(校验在写库前)
	var row struct{ RealName string }
	if err := dao.SysUser.Ctx(testutil.Ctx()).
		Where(dao.SysUser.Columns().Id, target).
		Fields(dao.SysUser.Columns().RealName).
		Scan(&row); err != nil {
		t.Fatalf("查询昵称失败: %v", err)
	}
	if row.RealName == "越权改名" {
		t.Fatalf("越权请求中的普通字段也不应生效")
	}

	// 管理员修改普通字段正常成功
	if err := s.Update(adminCtx, UpdateInput{Id: target, RealName: "管理员改名"}); err != nil {
		t.Fatalf("管理员修改普通字段应成功: %v", err)
	}
	if got := maxDevicesOf(t, target); got != nil {
		t.Fatalf("普通更新不应改变 max_devices, got %v", got)
	}
}

// TestAdminCannotCreateWithMaxDevices 管理员创建用户时携带 max_devices 应被拒绝
func TestAdminCannotCreateWithMaxDevices(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "dev2-super", 0, consts.RoleSuperAdmin)
	admin := insertUser(t, "dev2-admin", super, consts.RoleAdmin)
	adminCtx := identity.With(testutil.Ctx(), identity.Identity{Id: admin, Role: consts.RoleAdmin})

	_, err := s.Create(adminCtx, CreateInput{
		Username: "dev2-u1", Password: "123456", Role: consts.RoleUser, MaxDevices: intPtr(2),
	})
	if err == nil {
		t.Fatalf("管理员创建用户携带 max_devices 应被拒绝")
	}
	if gerror.Code(err).Code() != int(consts.CodeForbidden.Code()) {
		t.Fatalf("期望 403, got %v", err)
	}
}

// TestSuperAdminMaxDevices 超管可设置/清空 max_devices, 且落库正确
func TestSuperAdminMaxDevices(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	s := New()

	super := insertUser(t, "dev3-super", 0, consts.RoleSuperAdmin)
	target := insertUser(t, "dev3-target", super, consts.RoleUser)
	superCtx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})

	// 设置上限 3
	if err := s.Update(superCtx, UpdateInput{Id: target, MaxDevices: intPtr(3)}); err != nil {
		t.Fatalf("超管设置 max_devices 失败: %v", err)
	}
	if got := maxDevicesOf(t, target); got != 3 {
		t.Fatalf("max_devices 应为 3, got %v", got)
	}

	// 设为 0: 清空, 恢复使用全局默认(NULL)
	if err := s.Update(superCtx, UpdateInput{Id: target, MaxDevices: intPtr(0)}); err != nil {
		t.Fatalf("超管清空 max_devices 失败: %v", err)
	}
	if got := maxDevicesOf(t, target); got != nil {
		t.Fatalf("max_devices 应被清空为 NULL, got %v", got)
	}

	// 超管创建用户时可直接设置
	created, err := s.Create(superCtx, CreateInput{
		Username: "dev3-new", Password: "123456", Role: consts.RoleUser, MaxDevices: intPtr(2),
	})
	if err != nil {
		t.Fatalf("超管创建用户并设置 max_devices 失败: %v", err)
	}
	if got := maxDevicesOf(t, created); got != 2 {
		t.Fatalf("新建用户 max_devices 应为 2, got %v", got)
	}
}
