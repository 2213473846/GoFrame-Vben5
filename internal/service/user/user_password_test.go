package user

import (
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/service/identity"
	"demo/internal/testutil"
)

// mustChangeOf 读取用户 must_change_password
func mustChangeOf(t *testing.T, id uint64) int {
	t.Helper()
	v, err := dao.SysUser.Ctx(testutil.Ctx()).
		Where(dao.SysUser.Columns().Id, id).
		Fields(dao.SysUser.Columns().MustChangePassword).
		Value()
	if err != nil {
		t.Fatalf("查询 must_change_password 失败: %v", err)
	}
	return v.Int()
}

// TestPasswordPolicyOnCreateAndUpdate 密码策略下沉到 service 层: 创建/重置弱密码被拒, 合规通过且标记首登改密
func TestPasswordPolicyOnCreateAndUpdate(t *testing.T) {
	if !dbReady {
		t.Skip("本机 MySQL 不可用, 跳过集成测试")
	}
	super := insertUser(t, "pwd-super", 0, consts.RoleSuperAdmin)
	ctx := identity.With(testutil.Ctx(), identity.Identity{Id: super, Role: consts.RoleSuperAdmin})
	s := New()

	// 创建: 弱密码(纯数字过短)被拒绝
	if _, err := s.Create(ctx, CreateInput{Username: "pwd-weak", Password: "12345", Role: consts.RoleUser}); err == nil {
		t.Fatalf("创建用户时弱密码应被拒绝")
	}
	// 创建: 合规密码通过, 且新用户标记首登强制改密
	uid, err := s.Create(ctx, CreateInput{Username: "pwd-ok", Password: "abc123", Role: consts.RoleUser})
	if err != nil {
		t.Fatalf("创建用户时合规密码应通过: %v", err)
	}
	if got := mustChangeOf(t, uid); got != 1 {
		t.Fatalf("新建用户应标记首登强制改密, got %d", got)
	}

	// 更新密码: 弱密码被拒绝且不改变原标记
	if err = s.Update(ctx, UpdateInput{Id: uid, Password: "abcdef"}); err == nil {
		t.Fatalf("重置密码时弱密码应被拒绝")
	}
	if got := mustChangeOf(t, uid); got != 1 {
		t.Fatalf("弱密码被拒后标记不应变化, got %d", got)
	}
	// 更新密码: 合规密码通过并重新标记强制改密
	if err = s.Update(ctx, UpdateInput{Id: uid, Password: "Abc12345"}); err != nil {
		t.Fatalf("重置密码时合规密码应通过: %v", err)
	}
	if got := mustChangeOf(t, uid); got != 1 {
		t.Fatalf("管理员重置密码后应重新标记强制改密, got %d", got)
	}
}
