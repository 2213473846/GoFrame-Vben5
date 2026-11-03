// Package online 在线用户服务: 基于 Redis 维护在线会话
// 会话按 sessionId 登记(vben:online:<sessionId>), 一台设备/浏览器一次登录对应一条记录;
// 登出、被顶替、超出设备数上限、管理员强退时标记为已下线并记录原因,
// 认证中间件每次请求续期, 记录随 TTL 自动过期
package online

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"demo/internal/consts"
	"demo/internal/service/authz"
	"demo/internal/service/identity"
	logsvc "demo/internal/service/log"
	"demo/internal/service/paging"
	tokensvc "demo/internal/service/token"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

// 在线状态
const (
	StatusOnline  = 1 // 在线
	StatusOffline = 0 // 已下线
)

// OnlineUser 在线用户会话信息(同时作为 Redis 存储结构)
type OnlineUser struct {
	UserId        uint64 `json:"userId"`
	SessionId     string `json:"sessionId"`
	Username      string `json:"username"`
	RealName      string `json:"realName"`
	Role          string `json:"role"`
	Ip            string `json:"ip"`
	Location      string `json:"location"`
	Browser       string `json:"browser"`
	Os            string `json:"os"`
	LoginTime     string `json:"loginTime"`
	Status        int    `json:"status"` // 1=在线 0=已下线
	OfflineReason string `json:"offlineReason,omitempty"`
	OfflineAt     string `json:"offlineAt,omitempty"`
}

// ListInput 在线用户查询入参
type ListInput struct {
	Page     int
	PageSize int
	Username string
}

// ListOutput 在线用户查询结果
type ListOutput struct {
	Items []*OnlineUser `json:"items"`
	Total int           `json:"total"`
}

func onlineKey(sessionId string) string {
	return consts.RedisKeyOnline + sessionId
}

// Record 登录成功后登记在线会话(同步写入保证状态实时, 归属地异步补充)。
// 必须在新会话进入有效集合前调用, 避免被并发登录淘汰后异步写入产生僵尸在线记录。
func Record(ctx context.Context, sess *OnlineUser) {
	if sess == nil || sess.SessionId == "" || sess.UserId == 0 {
		return
	}
	sess.Status = StatusOnline
	b, err := json.Marshal(sess)
	if err != nil {
		g.Log().Errorf(ctx, "序列化在线会话失败: %v", err)
		return
	}
	if _, err = g.Redis().Do(ctx, "SET", onlineKey(sess.SessionId), b, "EX", consts.AccessTTL); err != nil {
		g.Log().Errorf(ctx, "写入在线会话失败: %v", err)
		return
	}
	ip := sess.Ip
	if ip == "" {
		return
	}
	// 异步解析归属地, 仅在该会话仍在线时补充, 不覆盖已下线记录
	go func(sessionId string) {
		defer func() {
			if e := recover(); e != nil {
				g.Log().Errorf(gctx.New(), "记录在线会话 panic: %v", e)
			}
		}()
		bgCtx := gctx.New()
		location := logsvc.ResolveLocation(bgCtx, ip)
		if location == "" {
			return
		}
		v, err := g.Redis().Do(bgCtx, "GET", onlineKey(sessionId))
		if err != nil || v.IsNil() || v.IsEmpty() {
			return
		}
		var cur OnlineUser
		if json.Unmarshal([]byte(v.String()), &cur) != nil || cur.Status != StatusOnline {
			return
		}
		cur.Location = location
		b, err := json.Marshal(&cur)
		if err != nil {
			return
		}
		_, _ = g.Redis().Do(bgCtx, "SET", onlineKey(sessionId), b, "EX", consts.AccessTTL)
	}(sess.SessionId)
}

// Touch 续期在线会话(认证中间件/刷新令牌时调用), 会话不存在时静默跳过
func Touch(ctx context.Context, sessionId string) {
	if sessionId == "" {
		return
	}
	_, _ = g.Redis().Do(ctx, "EXPIRE", onlineKey(sessionId), consts.AccessTTL)
}

// Drop 登出时移除在线会话
func Drop(ctx context.Context, sessionId string) {
	if sessionId == "" {
		return
	}
	_, _ = g.Redis().Do(ctx, "DEL", onlineKey(sessionId))
}

// MarkOffline 标记会话已下线并记录原因, 保留一段时间供管理员查看真实状态
func MarkOffline(ctx context.Context, sessionId, reason string) {
	if sessionId == "" {
		return
	}
	v, err := g.Redis().Do(ctx, "GET", onlineKey(sessionId))
	if err != nil || v.IsNil() || v.IsEmpty() {
		return
	}
	var sess OnlineUser
	if json.Unmarshal([]byte(v.String()), &sess) != nil {
		return
	}
	sess.Status = StatusOffline
	sess.OfflineReason = reason
	sess.OfflineAt = gtime.Now().Format("Y-m-d H:i:s")
	b, err := json.Marshal(&sess)
	if err != nil {
		return
	}
	_, _ = g.Redis().Do(ctx, "SET", onlineKey(sessionId), b, "EX", consts.OnlineOfflineRetention)
}

// List 在线用户列表(数据范围: 超管全部, 管理员自己+子孙, 普通用户仅自己)
func List(ctx context.Context, in *ListInput) (*ListOutput, error) {
	all, ids, err := logsvc.VisibleUserIds(ctx)
	if err != nil {
		return nil, err
	}
	allowed := map[uint64]bool{}
	for _, id := range ids {
		allowed[id] = true
	}
	keys, err := scanKeys(ctx, consts.RedisKeyOnline+"*")
	if err != nil {
		return nil, gerror.Wrap(err, "查询在线用户失败")
	}
	redis := g.Redis()
	items := make([]*OnlineUser, 0, len(keys))
	for _, key := range keys {
		v, err := redis.Do(ctx, "GET", key)
		if err != nil || v.IsNil() || v.IsEmpty() {
			continue
		}
		var sess OnlineUser
		if err = json.Unmarshal([]byte(v.String()), &sess); err != nil || sess.UserId == 0 {
			continue
		}
		if !all && !allowed[sess.UserId] {
			continue
		}
		if in.Username != "" && !strings.Contains(sess.Username, in.Username) {
			continue
		}
		// 兼容升级前按 accessToken 登记的旧记录(无 status/reason)
		if sess.Status == 0 && sess.OfflineReason == "" {
			sess.Status = StatusOnline
		}
		items = append(items, &sess)
	}
	// 在线优先, 再按登录时间倒序
	sort.Slice(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return items[i].Status > items[j].Status
		}
		return items[i].LoginTime > items[j].LoginTime
	})
	total := len(items)
	start, end := paging.Calc(total, in.Page, in.PageSize)
	return &ListOutput{Items: items[start:end], Total: total}, nil
}

// Kick 强退用户: 会话登记与 token 一并清理
// 超管可强退任何人, 管理员仅可强退自己添加的用户, 不能强退自己
func Kick(ctx context.Context, userId uint64) error {
	actor, err := authz.Login(ctx)
	if err != nil {
		return err
	}
	if userId == 0 {
		return gerror.NewCode(consts.CodeBadRequest, "请指定要强退的用户")
	}
	if userId == actor.Id {
		return gerror.NewCode(consts.CodeBadRequest, "不能强退当前登录账号")
	}
	if !actor.IsSuperAdmin() {
		if actor.Role != consts.RoleAdmin {
			return gerror.NewCode(consts.CodeForbidden, "无权强退用户")
		}
		descendants, err := logsvc.DescendantUserIds(ctx, actor.Id)
		if err != nil {
			return err
		}
		found := false
		for _, id := range descendants {
			if id == userId {
				found = true
				break
			}
		}
		if !found {
			return gerror.NewCode(consts.CodeForbidden, "只能强退自己添加的用户")
		}
	}
	tsvc := tokensvc.New()
	reason := tokensvc.AdminKickMessage()
	enabled := tsvc.LimitEnabled(ctx)
	// 会话维度清理: 删除 refreshToken 与注册信息, 保留 accessToken 记录用于返回被踢原因
	sessions, err := tsvc.KickUserSessions(ctx, userId, reason)
	if err != nil {
		return err
	}
	for _, sess := range sessions {
		MarkOffline(ctx, sess.SessionId, reason)
	}
	// token 维度兜底: 功能未启用时无注册表, 需删除全部 token; 启用时只清理无会话ID的旧 token
	return deleteUserTokens(ctx, userId, !enabled)
}

// deleteUserTokens 扫描并删除用户的 token 记录。
// all=false 时只清理没有 sessionId 的旧格式 token(避免误删仍有会话的 accessToken, 保留被踢提示)。
func deleteUserTokens(ctx context.Context, userId uint64, all bool) error {
	redis := g.Redis()
	keys, err := scanKeys(ctx, consts.RedisKeyAccessToken+"*")
	if err != nil {
		return gerror.Wrap(err, "强退用户失败")
	}
	for _, key := range keys {
		v, err := redis.Do(ctx, "GET", key)
		if err != nil || v.IsNil() || v.IsEmpty() {
			continue
		}
		var id identity.Identity
		if err = json.Unmarshal([]byte(v.String()), &id); err != nil || id.Id != userId {
			continue
		}
		if !all && id.SessionId != "" {
			continue
		}
		_, _ = redis.Do(ctx, "DEL", key)
	}
	refreshKeys, err := scanKeys(ctx, consts.RedisKeyRefreshToken+"*")
	if err != nil {
		return gerror.Wrap(err, "强退用户失败")
	}
	for _, key := range refreshKeys {
		v, err := redis.Do(ctx, "GET", key)
		if err != nil || v.IsNil() || v.IsEmpty() {
			continue
		}
		// 新格式为 {"userId":..,"sessionId":..}, 旧格式为纯 userId
		if gconv.Uint64(v.String()) == userId {
			_, _ = redis.Do(ctx, "DEL", key)
			continue
		}
		var p struct {
			UserId    uint64 `json:"userId"`
			SessionId string `json:"sessionId"`
		}
		if json.Unmarshal([]byte(v.String()), &p) != nil || p.UserId != userId {
			continue
		}
		if !all && p.SessionId != "" {
			continue
		}
		_, _ = redis.Do(ctx, "DEL", key)
	}
	return nil
}

// scanKeys SCAN 遍历匹配 key(避免 KEYS 阻塞 Redis)
// SCAN 响应为嵌套数组 [cursor, [key...]], 需取第二层再转字符串
func scanKeys(ctx context.Context, pattern string) ([]string, error) {
	keys := make([]string, 0)
	cursor := "0"
	for {
		v, err := g.Redis().Do(ctx, "SCAN", cursor, "MATCH", pattern, "COUNT", 200)
		if err != nil {
			return nil, err
		}
		arr := v.Interfaces()
		if len(arr) < 2 {
			break
		}
		cursor = gconv.String(arr[0])
		keys = append(keys, gconv.Strings(arr[1])...)
		if cursor == "" || cursor == "0" {
			break
		}
	}
	return keys, nil
}
