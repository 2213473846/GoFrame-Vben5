package dict

import (
	"context"
	"os"
	"testing"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/service/identity"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
)

var dbReady bool

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	code := m.Run()
	os.Exit(code)
}

func superCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 1, Role: consts.RoleSuperAdmin, Username: "root"})
}

func adminCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 2, Role: consts.RoleAdmin, Username: "admin"})
}

func userCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: 3, Role: consts.RoleUser, Username: "user"})
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

// TestDictCRUDAndCache 字典 CRUD、批量取值、停用过滤与缓存失效
func TestDictCRUDAndCache(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ClearCache()
	ctx := superCtx()
	s := New()

	// 新增类型/字典项
	typeId, err := s.TypeCreate(ctx, TypeCreateInput{Type: "test_color", Name: "测试颜色", Status: 1, Remark: "ut"})
	if err != nil {
		t.Fatalf("新增字典类型失败: %v", err)
	}
	redId, err := s.DataCreate(ctx, DataCreateInput{DictType: "test_color", Label: "红色", Value: "red", Sort: 1, Status: 1, TagColor: "danger"})
	if err != nil {
		t.Fatalf("新增字典项失败: %v", err)
	}
	if _, err = s.DataCreate(ctx, DataCreateInput{DictType: "test_color", Label: "蓝色", Value: "blue", Sort: 2, Status: 1, TagColor: "info"}); err != nil {
		t.Fatalf("新增字典项失败: %v", err)
	}
	// 同类型下 value 重复应失败
	_, err = s.DataCreate(ctx, DataCreateInput{DictType: "test_color", Label: "红色2", Value: "red", Status: 1})
	expectCode(t, "重复字典值", err, 400)
	// 重复类型编码应失败
	_, err = s.TypeCreate(ctx, TypeCreateInput{Type: "test_color", Name: "重复类型", Status: 1})
	expectCode(t, "重复类型编码", err, 400)

	// 批量取值
	data, err := s.Options(ctx, []string{"test_color", TypeUserStatus})
	if err != nil {
		t.Fatalf("批量取值失败: %v", err)
	}
	if len(data["test_color"]) != 2 {
		t.Fatalf("期望 2 个字典项, 实际 %d", len(data["test_color"]))
	}
	if data["test_color"][0].Value != "red" || data["test_color"][1].Value != "blue" {
		t.Fatalf("排序错误: %+v", data["test_color"])
	}

	// 修改后缓存应立即失效(同一 type 不重复请求由前端缓存, 服务端缓存必须感知变更)
	if err = s.DataUpdate(ctx, DataUpdateInput{
		Id: redId, DictType: "test_color", Label: "红色(新)", Value: "red", Sort: 1, Status: 1, TagColor: "danger",
	}); err != nil {
		t.Fatalf("修改字典项失败: %v", err)
	}
	data, err = s.Options(ctx, []string{"test_color"})
	if err != nil {
		t.Fatalf("批量取值失败: %v", err)
	}
	if data["test_color"][0].Label != "红色(新)" {
		t.Fatalf("缓存未失效, 仍返回旧值: %s", data["test_color"][0].Label)
	}

	// 停用字典项后不返回
	if err = s.DataStatus(ctx, redId, 0); err != nil {
		t.Fatalf("停用字典项失败: %v", err)
	}
	data, err = s.Options(ctx, []string{"test_color"})
	if err != nil {
		t.Fatalf("批量取值失败: %v", err)
	}
	if len(data["test_color"]) != 1 || data["test_color"][0].Value != "blue" {
		t.Fatalf("停用字典项后仍返回: %+v", data["test_color"])
	}

	// 停用类型后整个类型不返回
	if err = s.TypeStatus(ctx, typeId, 0); err != nil {
		t.Fatalf("停用字典类型失败: %v", err)
	}
	data, err = s.Options(ctx, []string{"test_color"})
	if err != nil {
		t.Fatalf("批量取值失败: %v", err)
	}
	if len(data["test_color"]) != 0 {
		t.Fatalf("停用类型后仍返回: %+v", data["test_color"])
	}

	// 恢复后删除类型, 级联删除字典项
	if err = s.TypeStatus(ctx, typeId, 1); err != nil {
		t.Fatalf("启用字典类型失败: %v", err)
	}
	if err = s.TypeDelete(ctx, typeId); err != nil {
		t.Fatalf("删除字典类型失败: %v", err)
	}
	data, err = s.Options(ctx, []string{"test_color"})
	if err != nil {
		t.Fatalf("批量取值失败: %v", err)
	}
	if len(data["test_color"]) != 0 {
		t.Fatalf("删除类型后仍返回: %+v", data["test_color"])
	}
}

// TestDictPermission 三级权限: 超管可写, 管理员只读, 普通用户仅可读下拉字典
func TestDictPermission(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ctx := superCtx()
	s := New()
	if _, err := s.TypeCreate(ctx, TypeCreateInput{Type: "test_perm", Name: "权限测试", Status: 1}); err != nil {
		t.Fatalf("准备字典类型失败: %v", err)
	}

	// 管理员: 可读管理列表, 不可写
	if _, err := s.TypeList(adminCtx(), TypeListInput{Status: -1}); err != nil {
		t.Fatalf("管理员读字典类型列表应成功: %v", err)
	}
	_, err := s.TypeCreate(adminCtx(), TypeCreateInput{Type: "test_admin_write", Name: "管理员写入", Status: 1})
	expectCode(t, "管理员写字典", err, 403)
	err = s.DataDelete(adminCtx(), 1)
	expectCode(t, "管理员删字典项", err, 403)

	// 普通用户: 不可访问管理接口, 但可读取下拉字典
	_, err = s.TypeList(userCtx(), TypeListInput{Status: -1})
	expectCode(t, "普通用户读管理列表", err, 403)
	if _, err = s.Options(userCtx(), []string{TypeUserStatus}); err != nil {
		t.Fatalf("普通用户读取字典下拉应成功: %v", err)
	}

	// 未登录不可读取
	if _, err = s.Options(testutil.Ctx(), []string{TypeUserStatus}); err != nil {
		expectCode(t, "未登录读取字典", err, 401)
	}
}

// TestDictSeedIdempotent 内置种子数据幂等写入
func TestDictSeedIdempotent(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	ctx := superCtx()
	s := New()
	if err := s.Seed(ctx); err != nil {
		t.Fatalf("首次 seed 失败: %v", err)
	}
	if err := s.Seed(ctx); err != nil {
		t.Fatalf("二次 seed 失败: %v", err)
	}
	cols := dao.SysDictType.Columns()
	count, err := dao.SysDictType.Ctx(ctx).Where(cols.Type, TypeUserStatus).Count()
	if err != nil {
		t.Fatalf("统计种子类型失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("用户状态类型应只有 1 条, 实际 %d", count)
	}
	dataCols := dao.SysDictData.Columns()
	dataCount, err := dao.SysDictData.Ctx(ctx).Where(dataCols.DictType, TypeUserStatus).Count()
	if err != nil {
		t.Fatalf("统计种子字典项失败: %v", err)
	}
	if dataCount != 2 {
		t.Fatalf("用户状态字典项应为 2 条, 实际 %d", dataCount)
	}
	// 种子写入后可正常按 type 取值
	options, err := s.Options(ctx, []string{TypeUserStatus, TypeLogType})
	if err != nil {
		t.Fatalf("读取种子字典失败: %v", err)
	}
	if len(options[TypeUserStatus]) != 2 || len(options[TypeLogType]) != 2 {
		t.Fatalf("种子字典项数量异常: %+v", options)
	}
}
