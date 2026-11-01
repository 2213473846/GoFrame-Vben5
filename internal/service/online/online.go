// Package online 在线用户服务: 基于 Redis 维护在线会话
// 会话随 accessToken 签发时写入(vben:online:<token>), 登出/强退时删除,
// 认证中间件每次请求续期, 停止活动后随 TTL 自动过期
package online

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"demo/internal/consts"
	"demo/internal/service/identity"
	logsvc "demo/internal/service/log"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

// OnlineUser 在线用户会话信息(同时作为 Redis 存储结构)
type OnlineUser struct {
	UserId    uint64 `json:"userId"`
	Username  string `json:"username"`
	RealName  string `json:"realName"`
	Role      string `json:"role"`
	Ip        string `json:"ip"`
	Location  string `json:"location"`
	Browser   string `json:"browser"`
	Os        string `json:"os"`
	LoginTime string `json:"loginTime"`
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

// RecordFromRequest 登录/刷新成功后写入在线会话(异步, 归属地解析不阻塞响应)
func RecordFromRequest(r *ghttp.Request, token string, id identity.Identity) {
	if token == "" || id.Id == 0 {
		return
	}
	ip := r.GetClientIp()
	browser, os := logsvc.ParseUserAgent(r.Header.Get("User-Agent"))
	sess := &OnlineUser{
		UserId:    id.Id,
		Username:  id.Username,
		RealName:  id.RealName,
		Role:      id.Role,
		Ip:        ip,
		Browser:   browser,
		Os:        os,
		LoginTime: gtime.Now().Format("Y-m-d H:i:s"),
	}
	go func() {
		defer func() {
			if e := recover(); e != nil {
				g.Log().Errorf(gctx.New(), "记录在线会话 panic: %v", e)
			}
		}()
		bgCtx := gctx.New()
		sess.Location = logsvc.ResolveLocation(bgCtx, ip)
		b, err := json.Marshal(sess)
		if err != nil {
			g.Log().Errorf(bgCtx, "序列化在线会话失败: %v", err)
			return
		}
		if _, err = g.Redis().Do(bgCtx, "SET", consts.RedisKeyOnline+token, b, "EX", consts.AccessTTL); err != nil {
			g.Log().Errorf(bgCtx, "写入在线会话失败: %v", err)
		}
	}()
}

// Touch 续期在线会话(认证中间件每次请求调用), 未登录或会话不存在时静默跳过
func Touch(ctx context.Context, token string) {
	if token == "" {
		return
	}
	_, _ = g.Redis().Do(ctx, "EXPIRE", consts.RedisKeyOnline+token, consts.AccessTTL)
}

// Drop 登出时移除在线会话
func Drop(ctx context.Context, token string) {
	if token == "" {
		return
	}
	_, _ = g.Redis().Do(ctx, "DEL", consts.RedisKeyOnline+token)
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
		items = append(items, &sess)
	}
	// 按登录时间倒序
	sort.Slice(items, func(i, j int) bool {
		return items[i].LoginTime > items[j].LoginTime
	})
	total := len(items)
	start := (in.Page - 1) * in.PageSize
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + in.PageSize
	if end > total || end <= 0 {
		end = total
	}
	return &ListOutput{Items: items[start:end], Total: total}, nil
}

// Kick 强退用户: 删除其全部 accessToken/refreshToken/在线会话
// 超管可强退任何人, 管理员仅可强退自己添加的用户, 不能强退自己
func Kick(ctx context.Context, userId uint64) error {
	actor := identity.Get(ctx)
	if actor == nil {
		return gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
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
	redis := g.Redis()
	target := gconv.String(userId)
	// accessToken: 值为身份 JSON, 匹配 userId 后连在线会话一起删除
	tokenKeys, err := scanKeys(ctx, consts.RedisKeyAccessToken+"*")
	if err != nil {
		return gerror.Wrap(err, "强退用户失败")
	}
	for _, key := range tokenKeys {
		v, err := redis.Do(ctx, "GET", key)
		if err != nil || v.IsNil() || v.IsEmpty() {
			continue
		}
		var id identity.Identity
		if err = json.Unmarshal([]byte(v.String()), &id); err != nil || id.Id != userId {
			continue
		}
		_, _ = redis.Do(ctx, "DEL", key)
		_, _ = redis.Do(ctx, "DEL", consts.RedisKeyOnline+strings.TrimPrefix(key, consts.RedisKeyAccessToken))
	}
	// refreshToken: 值为用户ID
	refreshKeys, err := scanKeys(ctx, consts.RedisKeyRefreshToken+"*")
	if err != nil {
		return gerror.Wrap(err, "强退用户失败")
	}
	for _, key := range refreshKeys {
		v, err := redis.Do(ctx, "GET", key)
		if err != nil || v.IsNil() || v.IsEmpty() {
			continue
		}
		if gconv.String(v.Val()) != target {
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
