package token_test

import (
	"os"
	"sync"
	"testing"
	"time"

	"demo/internal/consts"
	"demo/internal/service/identity"
	"demo/internal/service/token"
	"demo/internal/testutil"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

// redisReady 本机 Redis 可用时为 true, 否则跳过会话测试
var redisReady bool

func TestMain(m *testing.M) {
	redisReady = testutil.PrepareRedis()
	os.Exit(m.Run())
}

// flush 每个用例前清空测试库(DB 15), 用例相互独立
func flush(t *testing.T) {
	t.Helper()
	if _, err := g.Redis().Do(testutil.Ctx(), "FLUSHDB"); err != nil {
		t.Fatalf("清空 Redis 失败: %v", err)
	}
}

// zcard 用户有效会话集合大小
func zcard(t *testing.T, userId uint64) int {
	t.Helper()
	v, err := g.Redis().Do(testutil.Ctx(), "ZCARD", consts.RedisKeySessionSet+gconvUid(userId))
	if err != nil {
		t.Fatalf("ZCARD 失败: %v", err)
	}
	return v.Int()
}

func gconvUid(userId uint64) string {
	return gconv.String(userId)
}

// login 模拟一次登录: 签发 token 并登记会话, 返回本次被淘汰的会话
func login(t *testing.T, svc *token.Service, userId uint64, sessionId string, limit int, at time.Time) (token.SessionTokens, []token.SessionMeta) {
	t.Helper()
	id := identity.Identity{Id: userId, Username: "u", Role: consts.RoleUser, SessionId: sessionId}
	tks, err := svc.CreateSession(testutil.Ctx(), id)
	if err != nil {
		t.Fatalf("CreateSession 失败: %v", err)
	}
	evicted, err := svc.RegisterSession(testutil.Ctx(), token.RegisterInput{
		UserId:    userId,
		SessionId: sessionId,
		Access:    tks.Access,
		Refresh:   tks.Refresh,
		Limit:     limit,
		LoginAt:   at,
		Reason:    "账号已在其他设备登录, 请重新登录",
		Meta: token.SessionMeta{
			SessionId: sessionId,
			Access:    tks.Access,
			Refresh:   tks.Refresh,
			Ip:        "127.0.0.1",
			Browser:   "test",
			LoginTime: gtime.New(at).Format("Y-m-d H:i:s"),
		},
	})
	if err != nil {
		t.Fatalf("RegisterSession 失败: %v", err)
	}
	return tks, evicted
}

// TestLimitOneEvictsPrevious 上限为 1: 第二个设备登录后旧会话立即失效
func TestLimitOneEvictsPrevious(t *testing.T) {
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过会话测试")
	}
	flush(t)
	svc := token.New()
	ctx := testutil.Ctx()
	uid := uint64(9001)
	t0 := time.Now()
	s1, ev1 := login(t, svc, uid, "s1", 1, t0)
	t1 := t0.Add(time.Second)
	_, ev2 := login(t, svc, uid, "s2", 1, t1)

	if len(ev1) != 0 {
		t.Fatalf("首个会话不应淘汰任何会话, got %+v", ev1)
	}
	if len(ev2) != 1 || ev2[0].SessionId != "s1" {
		t.Fatalf("上限 1 时新登录应淘汰 s1, got %+v", ev2)
	}
	if n := zcard(t, uid); n != 1 {
		t.Fatalf("有效会话数应为 1, got %d", n)
	}
	alive, _, err := svc.SessionAlive(ctx, uid, "s1")
	if err != nil || alive {
		t.Fatalf("s1 应已失效, alive=%v err=%v", alive, err)
	}
	alive, _, err = svc.SessionAlive(ctx, uid, "s2")
	if err != nil || !alive {
		t.Fatalf("s2 应在线, alive=%v err=%v", alive, err)
	}
	// 旧 accessToken 下一次请求应被识别为"被顶替", 带回具体文案
	_, err = svc.Verify(ctx, s1.Access)
	if err == nil {
		t.Fatalf("旧 accessToken 不应再通过校验")
	}
	if gerror.Code(err).Code() != int(consts.CodeSessionKicked.Code()) {
		t.Fatalf("旧 accessToken 应返回被踢业务码, got %v", err)
	}
	if gerror.Code(err).Message() == "" {
		t.Fatalf("被踢错误应携带下线原因")
	}
}

// TestLimitThreeEvictsEarliest 上限为 3: 第 4 个设备登录淘汰最早建立的会话
func TestLimitThreeEvictsEarliest(t *testing.T) {
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过会话测试")
	}
	flush(t)
	svc := token.New()
	uid := uint64(9002)
	t0 := time.Now()
	for i, sid := range []string{"d1", "d2", "d3"} {
		_, ev := login(t, svc, uid, sid, 3, t0.Add(time.Duration(i)*time.Second))
		if len(ev) != 0 {
			t.Fatalf("上限内登录不应淘汰会话, got %+v", ev)
		}
	}
	_, ev := login(t, svc, uid, "d4", 3, t0.Add(3*time.Second))
	if len(ev) != 1 || ev[0].SessionId != "d1" {
		t.Fatalf("第 4 个设备应淘汰最早登录的 d1, got %+v", ev)
	}
	if n := zcard(t, uid); n != 3 {
		t.Fatalf("有效会话数应保持 3, got %d", n)
	}
	// d2/d3/d4 均在线
	for _, sid := range []string{"d2", "d3", "d4"} {
		if alive, _, _ := svc.SessionAlive(testutil.Ctx(), uid, sid); !alive {
			t.Fatalf("%s 应在线", sid)
		}
	}
}

// TestMultiTabNotEvicted 同一浏览器多标签共享 refreshToken 正常刷新, 不应互相顶掉
func TestMultiTabNotEvicted(t *testing.T) {
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过会话测试")
	}
	flush(t)
	svc := token.New()
	ctx := testutil.Ctx()
	uid := uint64(9003)
	tks, _ := login(t, svc, uid, "tab", 1, time.Now())

	// 两个标签各自刷新, 各拿到一个新的 accessToken
	for i := 0; i < 2; i++ {
		id := identity.Identity{Id: uid, Username: "u", Role: consts.RoleUser, SessionId: "tab"}
		access, err := svc.NewAccess(ctx, id)
		if err != nil {
			t.Fatalf("NewAccess 失败: %v", err)
		}
		if alive, _, err := svc.SessionAlive(ctx, uid, "tab"); err != nil || !alive {
			t.Fatalf("刷新后会话应仍在线, alive=%v err=%v", alive, err)
		}
		if _, err = svc.Verify(ctx, access); err != nil {
			t.Fatalf("第 %d 个标签的新 accessToken 应有效: %v", i+1, err)
		}
		if _, err = svc.Verify(ctx, tks.Access); err != nil {
			t.Fatalf("同一会话的旧 accessToken 仍应有效(多标签场景): %v", err)
		}
		if n := zcard(t, uid); n != 1 {
			t.Fatalf("多标签刷新不应新增会话, ZCARD=%d", n)
		}
	}
}

// TestRefreshCannotBypass 旧 refreshToken 被淘汰后无法续命, 也不能反顶新会话
func TestRefreshCannotBypass(t *testing.T) {
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过会话测试")
	}
	flush(t)
	svc := token.New()
	ctx := testutil.Ctx()
	uid := uint64(9004)
	s1, _ := login(t, svc, uid, "old", 1, time.Now())
	_, _ = login(t, svc, uid, "new", 1, time.Now().Add(time.Second))

	if _, err := svc.ReadRefresh(ctx, s1.Refresh); err == nil {
		t.Fatalf("被淘汰会话的 refreshToken 应已作废")
	} else if gerror.Code(err).Code() != int(consts.CodeUnauthorized.Code()) {
		t.Fatalf("旧 refreshToken 应返回 401, got %v", err)
	}
	// 用已作废的 refresh 再登录路径不成立: 新会话仍是当前唯一有效会话
	if alive, _, _ := svc.SessionAlive(ctx, uid, "new"); !alive {
		t.Fatalf("新会话应保持在线")
	}
}

// TestLogoutCleanup 退出后登记被精确清理, 不会残留导致下次登录被误踢
func TestLogoutCleanup(t *testing.T) {
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过会话测试")
	}
	flush(t)
	svc := token.New()
	ctx := testutil.Ctx()
	uid := uint64(9005)
	s1, _ := login(t, svc, uid, "s1", 1, time.Now())

	if sid := svc.RemoveSession(ctx, s1.Access, s1.Refresh); sid != "s1" {
		t.Fatalf("登出应返回会话ID, got %q", sid)
	}
	if n := zcard(t, uid); n != 0 {
		t.Fatalf("登出后会话登记应清空, ZCARD=%d", n)
	}
	if alive, _, _ := svc.SessionAlive(ctx, uid, "s1"); alive {
		t.Fatalf("登出后会话不应在线")
	}
	// 再次登录不应淘汰任何会话(无残留)
	if _, ev := login(t, svc, uid, "s2", 1, time.Now()); len(ev) != 0 {
		t.Fatalf("无残留时重新登录不应淘汰会话, got %+v", ev)
	}
}

// TestConcurrentLoginSingleWinner 并发登录同一账号, 同一时刻只有一个会话胜出
func TestConcurrentLoginSingleWinner(t *testing.T) {
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过会话测试")
	}
	flush(t)
	svc := token.New()
	uid := uint64(9006)
	at := time.Now()
	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			sid := "c" + string(rune('a'+i))
			id := identity.Identity{Id: uid, Username: "u", Role: consts.RoleUser, SessionId: sid}
			tks, err := svc.CreateSession(testutil.Ctx(), id)
			if err != nil {
				t.Errorf("CreateSession 失败: %v", err)
				return
			}
			if _, err = svc.RegisterSession(testutil.Ctx(), token.RegisterInput{
				UserId: uid, SessionId: sid, Access: tks.Access, Refresh: tks.Refresh,
				Limit: 1, LoginAt: at,
				Meta: token.SessionMeta{SessionId: sid, LoginTime: gtime.New(at).Format("Y-m-d H:i:s")},
			}); err != nil {
				t.Errorf("RegisterSession 失败: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if c := zcard(t, uid); c != 1 {
		t.Fatalf("并发登录后应只剩 1 个有效会话, ZCARD=%d", c)
	}
	aliveCount := 0
	for i := 0; i < n; i++ {
		sid := "c" + string(rune('a'+i))
		if alive, _, _ := svc.SessionAlive(testutil.Ctx(), uid, sid); alive {
			aliveCount++
		}
	}
	if aliveCount != 1 {
		t.Fatalf("并发登录应只有一个胜者, alive=%d", aliveCount)
	}
}

// TestRebalanceTrimsSessions 上限调小后立即清理多余会话(保留最近登录的)
func TestRebalanceTrimsSessions(t *testing.T) {
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过会话测试")
	}
	flush(t)
	svc := token.New()
	ctx := testutil.Ctx()
	uid := uint64(9007)
	t0 := time.Now()
	for i, sid := range []string{"r1", "r2", "r3"} {
		_, _ = login(t, svc, uid, sid, 3, t0.Add(time.Duration(i)*time.Second))
	}
	evicted, err := svc.Rebalance(ctx, uid, 1, "超出设备数上限")
	if err != nil {
		t.Fatalf("Rebalance 失败: %v", err)
	}
	if len(evicted) != 2 || evicted[0].SessionId != "r1" || evicted[1].SessionId != "r2" {
		t.Fatalf("调小上限应淘汰最早的 r1/r2, got %+v", evicted)
	}
	if n := zcard(t, uid); n != 1 {
		t.Fatalf("调小上限后应只保留 1 个会话, ZCARD=%d", n)
	}
	if alive, _, _ := svc.SessionAlive(ctx, uid, "r3"); !alive {
		t.Fatalf("最近的 r3 应保留")
	}
	// 被淘汰会话的下线原因可读
	if _, reason, _ := svc.SessionAlive(ctx, uid, "r1"); reason == "" {
		t.Fatalf("被淘汰会话应记录下线原因")
	}
}

// TestStaleRegistrationCleanup 超过 refresh 生命周期的陈旧登记会被清理, 不再占用名额
func TestStaleRegistrationCleanup(t *testing.T) {
	if !redisReady {
		t.Skip("本机 Redis 不可用, 跳过会话测试")
	}
	flush(t)
	svc := token.New()
	uid := uint64(9008)
	staleAt := time.Now().Add(-time.Duration(consts.RefreshTTL+10) * time.Second)
	_, _ = login(t, svc, uid, "stale", 1, staleAt)
	if _, ev := login(t, svc, uid, "fresh", 1, time.Now()); len(ev) != 0 {
		t.Fatalf("陈旧登记应先被清理, 不应淘汰并报错, got %+v", ev)
	}
	if n := zcard(t, uid); n != 1 {
		t.Fatalf("陈旧会话应被清理, ZCARD=%d", n)
	}
	if alive, _, _ := svc.SessionAlive(testutil.Ctx(), uid, "fresh"); !alive {
		t.Fatalf("新会话应在线")
	}
}
