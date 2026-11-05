package notice_test

import (
	"context"
	"os"
	"testing"
	"time"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/service/dashboard"
	dictsvc "demo/internal/service/dict"
	"demo/internal/service/identity"
	noticesvc "demo/internal/service/notice"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gtime"
)

var dbReady bool

var (
	superId    uint64
	adminId    uint64
	userAId    uint64
	userBId    uint64
	outsiderId uint64 // 不在 admin 的子孙范围内
)

func TestMain(m *testing.M) {
	dbReady = testutil.Prepare()
	if dbReady {
		seedUsers()
		// 公告类型/级别校验依赖字典种子
		if err := dictsvc.New().Seed(testutil.Ctx()); err != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}

func seedUsers() {
	ctx := testutil.Ctx()
	cols := dao.SysUser.Columns()
	create := func(username, role string, pid uint64) uint64 {
		var exist struct {
			Id uint64 `orm:"id"`
		}
		_ = dao.SysUser.Ctx(ctx).Where(cols.Username, username).Scan(&exist)
		if exist.Id > 0 {
			return exist.Id
		}
		id, err := dao.SysUser.Ctx(ctx).Data(do.SysUser{
			Pid: pid, Username: username, Password: "x", RealName: username, Role: role, Status: 1,
		}).InsertAndGetId()
		if err != nil {
			panic(err)
		}
		return uint64(id)
	}
	superId = create("notice_root", consts.RoleSuperAdmin, 0)
	adminId = create("notice_admin", consts.RoleAdmin, superId)
	userAId = create("notice_a", consts.RoleUser, adminId)
	userBId = create("notice_b", consts.RoleUser, adminId)
	outsiderId = create("notice_outsider", consts.RoleUser, 0)
}

func superCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: superId, Role: consts.RoleSuperAdmin, Username: "notice_root"})
}

func adminCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: adminId, Role: consts.RoleAdmin, Username: "notice_admin"})
}

func userACtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: userAId, Role: consts.RoleUser, Username: "notice_a"})
}

func userBCtx() context.Context {
	return identity.With(testutil.Ctx(), identity.Identity{Id: userBId, Role: consts.RoleUser, Username: "notice_b"})
}

func TestMyVisibilityConsistency(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	now := gtime.Now()
	allId := insertNotice(t, noticeSpec{title: "全员公告", status: noticesvc.StatusPublished, scope: noticesvc.ScopeAll, publisher: superId, publishAt: now.Add(-time.Hour)})
	aId := insertNotice(t, noticeSpec{title: "定向A", status: noticesvc.StatusPublished, scope: noticesvc.ScopeTargeted, publisher: adminId, publishAt: now.Add(-time.Hour)}, userAId)
	bId := insertNotice(t, noticeSpec{title: "定向B", status: noticesvc.StatusPublished, scope: noticesvc.ScopeTargeted, publisher: adminId, publishAt: now.Add(-time.Hour)}, userBId)
	insertNotice(t, noticeSpec{title: "草稿", status: noticesvc.StatusDraft, scope: noticesvc.ScopeAll, publisher: adminId})
	insertNotice(t, noticeSpec{title: "已撤回", status: noticesvc.StatusRevoked, scope: noticesvc.ScopeAll, publisher: adminId, publishAt: now.Add(-time.Hour)})
	insertNotice(t, noticeSpec{title: "已过期", status: noticesvc.StatusPublished, scope: noticesvc.ScopeAll, publisher: adminId, publishAt: now.Add(-2 * time.Hour), expireAt: now.Add(-time.Hour)})

	out, err := noticesvc.New().MyList(userACtx(), noticesvc.MyListInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("A 查询我的公告失败: %v", err)
	}
	if out.Total != 2 || len(out.Items) != 2 {
		t.Fatalf("A 可见公告应为 2(全员+定向A), 实际 total=%d items=%d", out.Total, len(out.Items))
	}
	unread, err := noticesvc.New().UnreadCount(userACtx())
	if err != nil {
		t.Fatalf("A 查询未读数失败: %v", err)
	}
	if unread != out.Total {
		t.Fatalf("A 未读数(%d)与列表 total(%d)不一致", unread, out.Total)
	}
	ids := map[uint64]bool{}
	for _, item := range out.Items {
		ids[item.Id] = true
	}
	if !ids[allId] || !ids[aId] || ids[bId] {
		t.Fatalf("A 的可见集合错误: %v", ids)
	}
	// B 看不到 A 的定向公告
	outB, err := noticesvc.New().MyList(userBCtx(), noticesvc.MyListInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("B 查询我的公告失败: %v", err)
	}
	for _, item := range outB.Items {
		if item.Id == aId {
			t.Fatal("B 不应看到定向给 A 的公告")
		}
	}
	if outB.Total != 2 {
		t.Fatalf("B 可见公告应为 2(全员+定向B), 实际 %d", outB.Total)
	}
}

func TestReadIdempotent(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	now := gtime.Now()
	allId := insertNotice(t, noticeSpec{title: "全员公告", status: noticesvc.StatusPublished, scope: noticesvc.ScopeAll, publisher: superId, publishAt: now.Add(-time.Hour)})
	aId := insertNotice(t, noticeSpec{title: "定向A", status: noticesvc.StatusPublished, scope: noticesvc.ScopeTargeted, publisher: adminId, publishAt: now.Add(-time.Hour)}, userAId)

	s := noticesvc.New()
	if err := s.Read(userACtx(), noticesvc.ReadInput{Id: aId}); err != nil {
		t.Fatalf("标记单条已读失败: %v", err)
	}
	if err := s.Read(userACtx(), noticesvc.ReadInput{Id: aId}); err != nil {
		t.Fatalf("重复标记已读应幂等, 实际: %v", err)
	}
	unread, err := s.UnreadCount(userACtx())
	if err != nil {
		t.Fatalf("查询未读数失败: %v", err)
	}
	if unread != 1 {
		t.Fatalf("已读 1 条后未读数应为 1, 实际 %d", unread)
	}
	// 重复调用计数不变
	if count := countReads(t, aId, userAId); count != 1 {
		t.Fatalf("重复标记已读不应产生重复记录, 实际 %d", count)
	}
	if err = s.Read(userACtx(), noticesvc.ReadInput{All: true}); err != nil {
		t.Fatalf("全部已读失败: %v", err)
	}
	if err = s.Read(userACtx(), noticesvc.ReadInput{All: true}); err != nil {
		t.Fatalf("重复全部已读应幂等, 实际: %v", err)
	}
	unread, err = s.UnreadCount(userACtx())
	if err != nil {
		t.Fatalf("全部已读后查询未读数失败: %v", err)
	}
	if unread != 0 {
		t.Fatalf("全部已读后未读数应为 0, 实际 %d", unread)
	}
	out, err := s.MyList(userACtx(), noticesvc.MyListInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("查询我的公告失败: %v", err)
	}
	for _, item := range out.Items {
		if !item.IsRead {
			t.Fatalf("全部已读后仍存在未读项: %+v", item)
		}
	}
	if allId == 0 {
		t.Fatal("unreachable")
	}
}

func TestReadInvisibleRejected(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	now := gtime.Now()
	bId := insertNotice(t, noticeSpec{title: "定向B", status: noticesvc.StatusPublished, scope: noticesvc.ScopeTargeted, publisher: adminId, publishAt: now.Add(-time.Hour)}, userBId)
	err := noticesvc.New().Read(userACtx(), noticesvc.ReadInput{Id: bId})
	expectCode(t, "标记不可见公告已读", err, 400)
}

func TestManagerScope(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	s := noticesvc.New()
	base := func(scope int, targets []uint64) noticesvc.CreateInput {
		return noticesvc.CreateInput{
			Title: "范围校验", Content: "内容", NoticeType: "notice", NoticeLevel: "normal",
			TargetScope: scope, TargetUserIds: targets,
		}
	}
	// 管理员不能发全员公告
	if _, err := s.Create(adminCtx(), base(noticesvc.ScopeAll, nil)); err == nil {
		t.Fatal("管理员发布全员公告应被拒绝")
	} else {
		expectCode(t, "管理员发全员公告", err, 403)
	}
	// 管理员定向范围外用户: 整体拒绝且不落库
	if _, err := s.Create(adminCtx(), base(noticesvc.ScopeTargeted, []uint64{userAId, outsiderId})); err == nil {
		t.Fatal("管理员定向范围外用户应被拒绝")
	} else {
		expectCode(t, "管理员定向范围外用户", err, 403)
	}
	if total := countNotices(t); total != 0 {
		t.Fatalf("越权请求不应写入公告, 实际 %d 条", total)
	}
	if total := countNoticeUsers(t); total != 0 {
		t.Fatalf("越权请求不应写入定向关系, 实际 %d 条", total)
	}
	// 管理员可定向子孙
	id, err := s.Create(adminCtx(), base(noticesvc.ScopeTargeted, []uint64{userAId, userBId}))
	if err != nil {
		t.Fatalf("管理员定向子孙失败: %v", err)
	}
	if total := countNoticeUsersByNotice(t, id); total != 2 {
		t.Fatalf("定向关系应为 2 条, 实际 %d", total)
	}
	// 超管可发全员
	if _, err = s.Create(superCtx(), base(noticesvc.ScopeAll, nil)); err != nil {
		t.Fatalf("超管发布全员公告失败: %v", err)
	}
}

func TestAdminOwnership(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	s := noticesvc.New()
	// 超管创建的公告
	superNotice, err := s.Create(superCtx(), noticesvc.CreateInput{
		Title: "超管公告", Content: "内容", NoticeType: "notice", NoticeLevel: "normal", TargetScope: noticesvc.ScopeAll,
	})
	if err != nil {
		t.Fatalf("超管创建公告失败: %v", err)
	}
	if _, err = s.Detail(adminCtx(), superNotice); err == nil {
		t.Fatal("管理员不应看到他人公告详情")
	} else {
		expectCode(t, "管理员查看他人公告", err, 403)
	}
	if err = s.Update(adminCtx(), noticesvc.UpdateInput{Id: superNotice, CreateInput: noticesvc.CreateInput{
		Title: "篡改", Content: "内容", NoticeType: "notice", NoticeLevel: "normal", TargetScope: noticesvc.ScopeTargeted, TargetUserIds: []uint64{userAId},
	}}); err == nil {
		t.Fatal("管理员不应修改他人公告")
	} else {
		expectCode(t, "管理员修改他人公告", err, 403)
	}
	if err = s.Delete(adminCtx(), superNotice); err == nil {
		t.Fatal("管理员不应删除他人公告")
	} else {
		expectCode(t, "管理员删除他人公告", err, 403)
	}
	if err = s.Publish(adminCtx(), superNotice, ""); err == nil {
		t.Fatal("管理员不应发布他人公告")
	} else {
		expectCode(t, "管理员发布他人公告", err, 403)
	}
	// 自己的公告可查看
	own, err := s.Create(adminCtx(), noticesvc.CreateInput{
		Title: "管理员自己的公告", Content: "内容", NoticeType: "notice", NoticeLevel: "normal",
		TargetScope: noticesvc.ScopeTargeted, TargetUserIds: []uint64{userAId},
	})
	if err != nil {
		t.Fatalf("管理员创建公告失败: %v", err)
	}
	if _, err = s.Detail(adminCtx(), own); err != nil {
		t.Fatalf("管理员查看自己公告失败: %v", err)
	}
	// 管理员列表只含自己创建的公告
	out, err := s.List(adminCtx(), noticesvc.ListInput{Status: -1, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("管理员查询公告列表失败: %v", err)
	}
	for _, item := range out.Items {
		if item.PublisherId != adminId {
			t.Fatalf("管理员列表包含他人公告: %+v", item)
		}
	}
}

func TestPublishAndRevoke(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	s := noticesvc.New()
	id, err := s.Create(adminCtx(), noticesvc.CreateInput{
		Title: "待发布", Content: "内容", NoticeType: "notice", NoticeLevel: "important",
		TargetScope: noticesvc.ScopeTargeted, TargetUserIds: []uint64{userAId},
	})
	if err != nil {
		t.Fatalf("创建公告失败: %v", err)
	}
	// 草稿对用户不可见
	if unread, _ := s.UnreadCount(userACtx()); unread != 0 {
		t.Fatalf("草稿不应计入未读, 实际 %d", unread)
	}
	if err = s.Publish(adminCtx(), id, ""); err != nil {
		t.Fatalf("发布公告失败: %v", err)
	}
	unread, err := s.UnreadCount(userACtx())
	if err != nil {
		t.Fatalf("查询未读数失败: %v", err)
	}
	if unread != 1 {
		t.Fatalf("发布后 A 未读数应为 1, 实际 %d", unread)
	}
	// 撤回后列表立刻消失, 未读数减 1
	if err = s.Revoke(adminCtx(), id); err != nil {
		t.Fatalf("撤回公告失败: %v", err)
	}
	out, err := s.MyList(userACtx(), noticesvc.MyListInput{Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("撤回后查询我的公告失败: %v", err)
	}
	if out.Total != 0 {
		t.Fatalf("撤回后 A 列表应为空, 实际 %d", out.Total)
	}
	unread, err = s.UnreadCount(userACtx())
	if err != nil {
		t.Fatalf("撤回后查询未读数失败: %v", err)
	}
	if unread != 0 {
		t.Fatalf("撤回后未读数应为 0, 实际 %d", unread)
	}
	// 已发布不能再次发布, 撤回后可重新发布
	if err = s.Revoke(adminCtx(), id); err == nil {
		t.Fatal("已撤回公告不应再次撤回")
	} else {
		expectCode(t, "重复撤回", err, 400)
	}
	if err = s.Publish(adminCtx(), id, ""); err != nil {
		t.Fatalf("撤回后重新发布失败: %v", err)
	}
}

func TestDeleteCascade(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	s := noticesvc.New()
	id, err := s.Create(adminCtx(), noticesvc.CreateInput{
		Title: "待删除", Content: "内容", NoticeType: "notice", NoticeLevel: "normal",
		TargetScope: noticesvc.ScopeTargeted, TargetUserIds: []uint64{userAId},
	})
	if err != nil {
		t.Fatalf("创建公告失败: %v", err)
	}
	if err = s.Publish(adminCtx(), id, ""); err != nil {
		t.Fatalf("发布公告失败: %v", err)
	}
	if err = s.Read(userACtx(), noticesvc.ReadInput{Id: id}); err != nil {
		t.Fatalf("标记已读失败: %v", err)
	}
	if countNoticeUsersByNotice(t, id) == 0 || countReads(t, id, userAId) == 0 {
		t.Fatal("前置数据缺失: 定向/已读记录应存在")
	}
	if err = s.Delete(adminCtx(), id); err != nil {
		t.Fatalf("删除公告失败: %v", err)
	}
	if countNoticeUsersByNotice(t, id) != 0 {
		t.Fatal("删除后定向用户记录未清理")
	}
	if countReads(t, id, userAId) != 0 {
		t.Fatal("删除后已读记录未清理")
	}
	// 软删后管理端详情不可见
	if _, err = s.Detail(adminCtx(), id); err == nil {
		t.Fatal("删除后仍可查询公告详情")
	} else {
		expectCode(t, "删除后查询详情", err, 404)
	}
}

func TestDashboardConsistency(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	now := gtime.Now()
	for i := 1; i <= 4; i++ {
		insertNotice(t, noticeSpec{
			title: "工作台公告", status: noticesvc.StatusPublished, scope: noticesvc.ScopeAll,
			publisher: superId, publishAt: now.Add(-time.Duration(i) * time.Hour),
		})
	}
	insertNotice(t, noticeSpec{
		title: "定向管理员", status: noticesvc.StatusPublished, scope: noticesvc.ScopeTargeted,
		publisher: superId, publishAt: now.Add(-30 * time.Minute),
	}, adminId)

	overview, err := dashboard.New().Overview(adminCtx())
	if err != nil {
		t.Fatalf("工作台概览失败: %v", err)
	}
	unread, err := noticesvc.New().UnreadCount(adminCtx())
	if err != nil {
		t.Fatalf("查询未读数失败: %v", err)
	}
	if overview.UnreadNoticeCount != unread {
		t.Fatalf("工作台未读数(%d)与 /notice/unread-count(%d)不一致", overview.UnreadNoticeCount, unread)
	}
	if len(overview.RecentNotices) == 0 || len(overview.RecentNotices) > 3 {
		t.Fatalf("最近公告条数异常: %d", len(overview.RecentNotices))
	}
	for _, item := range overview.RecentNotices {
		if item.Id == 0 || item.Title == "" || item.PublishAt == "" {
			t.Fatalf("最近公告字段缺失: %+v", item)
		}
	}
}

func TestPermission(t *testing.T) {
	if !dbReady {
		t.Skip("MySQL 不可用, 跳过集成测试")
	}
	resetNotices(t)
	s := noticesvc.New()
	// 未登录 401
	if _, err := s.MyList(testutil.Ctx(), noticesvc.MyListInput{Page: 1, PageSize: 10}); err == nil {
		t.Fatal("未登录查询我的公告应返回 401")
	} else {
		expectCode(t, "未登录查询我的公告", err, 401)
	}
	if _, err := s.UnreadCount(testutil.Ctx()); err == nil {
		t.Fatal("未登录查询未读数应返回 401")
	} else {
		expectCode(t, "未登录查询未读数", err, 401)
	}
	// 普通用户调管理端 403
	if _, err := s.List(userACtx(), noticesvc.ListInput{Status: -1, Page: 1, PageSize: 10}); err == nil {
		t.Fatal("普通用户查询管理列表应返回 403")
	} else {
		expectCode(t, "普通用户查询管理列表", err, 403)
	}
	if _, err := s.Create(userACtx(), noticesvc.CreateInput{
		Title: "越权", Content: "内容", NoticeType: "notice", NoticeLevel: "normal", TargetScope: noticesvc.ScopeAll,
	}); err == nil {
		t.Fatal("普通用户创建公告应返回 403")
	} else {
		expectCode(t, "普通用户创建公告", err, 403)
	}
}

// ------------------------------------------------------------
// 测试工具
// ------------------------------------------------------------

type noticeSpec struct {
	title      string
	status     int
	scope      int
	publisher  uint64
	publishAt  *gtime.Time
	expireAt   *gtime.Time
	noticeType string
	level      string
}

func insertNotice(t *testing.T, spec noticeSpec, targets ...uint64) uint64 {
	t.Helper()
	ctx := testutil.Ctx()
	if spec.noticeType == "" {
		spec.noticeType = "notice"
	}
	if spec.level == "" {
		spec.level = "normal"
	}
	id, err := dao.SysNotice.Ctx(ctx).Data(do.SysNotice{
		Title: spec.title, Content: "内容", NoticeType: spec.noticeType, NoticeLevel: spec.level,
		Status: spec.status, TargetScope: spec.scope, PublisherId: spec.publisher,
		PublishAt: spec.publishAt, ExpireAt: spec.expireAt,
	}).InsertAndGetId()
	if err != nil {
		t.Fatalf("插入公告失败: %v", err)
	}
	if spec.scope == noticesvc.ScopeTargeted {
		for _, userId := range targets {
			if _, err = dao.SysNoticeUser.Ctx(ctx).Data(do.SysNoticeUser{
				NoticeId: uint64(id), UserId: userId,
			}).Insert(); err != nil {
				t.Fatalf("插入定向用户失败: %v", err)
			}
		}
		if len(targets) == 0 {
			t.Fatal("定向公告缺少目标用户")
		}
	}
	return uint64(id)
}

func resetNotices(t *testing.T) {
	t.Helper()
	ctx := testutil.Ctx()
	if _, err := dao.SysNotice.Ctx(ctx).Unscoped().Where("1=1").Delete(); err != nil {
		t.Fatalf("清理公告失败: %v", err)
	}
	if _, err := dao.SysNoticeUser.Ctx(ctx).Where("1=1").Delete(); err != nil {
		t.Fatalf("清理定向用户失败: %v", err)
	}
	if _, err := dao.SysNoticeRead.Ctx(ctx).Where("1=1").Delete(); err != nil {
		t.Fatalf("清理已读记录失败: %v", err)
	}
}

func countNotices(t *testing.T) int {
	t.Helper()
	count, err := dao.SysNotice.Ctx(testutil.Ctx()).Count()
	if err != nil {
		t.Fatalf("统计公告失败: %v", err)
	}
	return count
}

func countNoticeUsers(t *testing.T) int {
	t.Helper()
	count, err := dao.SysNoticeUser.Ctx(testutil.Ctx()).Count()
	if err != nil {
		t.Fatalf("统计定向关系失败: %v", err)
	}
	return count
}

func countNoticeUsersByNotice(t *testing.T, noticeId uint64) int {
	t.Helper()
	count, err := dao.SysNoticeUser.Ctx(testutil.Ctx()).Where(dao.SysNoticeUser.Columns().NoticeId, noticeId).Count()
	if err != nil {
		t.Fatalf("统计定向关系失败: %v", err)
	}
	return count
}

func countReads(t *testing.T, noticeId, userId uint64) int {
	t.Helper()
	count, err := dao.SysNoticeRead.Ctx(testutil.Ctx()).
		Where(dao.SysNoticeRead.Columns().NoticeId, noticeId).
		Where(dao.SysNoticeRead.Columns().UserId, userId).
		Count()
	if err != nil {
		t.Fatalf("统计已读记录失败: %v", err)
	}
	return count
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
