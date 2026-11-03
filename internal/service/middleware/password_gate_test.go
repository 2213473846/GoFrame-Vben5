package middleware

import (
	"os"
	"testing"

	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

// gateDBReady 本机 MySQL 可用时为 true, 否则跳过强制改密拦截测试
var gateDBReady bool

func TestMain(m *testing.M) {
	gateDBReady = testutil.Prepare()
	os.Exit(m.Run())
}

// TestForcedPasswordChangeGate 首登强制改密拦截: 未改密前仅放行改密与登出, 改密后放行其他接口
func TestForcedPasswordChangeGate(t *testing.T) {
	if !gateDBReady {
		t.Skip("本机 MySQL 不可用, 跳过强制改密拦截测试")
	}
	ctx := testutil.Ctx()
	id, err := dao.SysUser.Ctx(ctx).Data(do.SysUser{
		Username: "gate-user", Password: "x", RealName: "gate", Role: "user",
		Status: 1, MustChangePassword: 1,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入用户失败: %v", err)
	}
	uid := gconv.Uint64(id)

	if !mustChangePassword(ctx, uid) {
		t.Fatalf("must_change_password=1 的用户应命中拦截")
	}
	// 未改密前: 改密与登出放行, 其余接口拦截
	if !passwordChangeAllowed["/auth/change-password"] || !passwordChangeAllowed["/auth/logout"] {
		t.Fatalf("改密与登出应始终放行")
	}
	if passwordChangeAllowed["/user/list"] || passwordChangeAllowed["/role/list"] {
		t.Fatalf("未改密前业务接口不应放行")
	}

	// 改密成功清除标记后, 业务接口放行
	if _, err = dao.SysUser.Ctx(ctx).
		Where(dao.SysUser.Columns().Id, uid).
		Data(g.Map{dao.SysUser.Columns().MustChangePassword: 0}).
		Update(); err != nil {
		t.Fatalf("清除强制改密标记失败: %v", err)
	}
	if mustChangePassword(ctx, uid) {
		t.Fatalf("改密后不应再命中拦截")
	}
}
