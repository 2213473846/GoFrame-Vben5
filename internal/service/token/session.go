package token

import (
	"context"
	"encoding/json"
	"time"

	"demo/internal/consts"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

// SessionMeta 会话摘要, 登记在 Redis 中, 供在线列表与被踢提示使用
type SessionMeta struct {
	SessionId string `json:"sessionId"`
	Access    string `json:"access"`
	Refresh   string `json:"refresh"`
	Ip        string `json:"ip,omitempty"`
	Browser   string `json:"browser,omitempty"`
	Os        string `json:"os,omitempty"`
	Location  string `json:"location,omitempty"`
	LoginTime string `json:"loginTime"`
}

// RegisterInput 登录时登记会话的输入
type RegisterInput struct {
	UserId    uint64
	SessionId string
	Access    string
	Refresh   string
	Limit     int // 生效的设备数上限, <=0 时按 1 处理
	Meta      SessionMeta
	LoginAt   time.Time // 零值取当前时间
	// Reason 本次被淘汰会话的下线原因(写入被踢标记并作废 refreshToken)
	Reason string
}

// 原子登记会话: 清理超过 refresh 生命周期的陈旧登记 -> 按登录时间淘汰超出上限的最早会话 -> 写入新会话。
// Redis 单线程执行 Lua, 并发登录时串行判定, 只有最后写入的会话保留。
const registerSessionScript = `
local zkey, mkey, lkey = KEYS[1], KEYS[2], KEYS[3]
local sessionId = ARGV[1]
local score = ARGV[2]
local limit = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])
local meta = ARGV[5]
local now = tonumber(ARGV[6])
local cutoff = now - ttl * 1000
local expired = redis.call('ZRANGEBYSCORE', zkey, '-inf', '(' .. tostring(cutoff))
if #expired > 0 then
    redis.call('ZREM', zkey, unpack(expired))
    redis.call('HDEL', mkey, unpack(expired))
end
local removed = {}
if limit > 0 then
    local cnt = redis.call('ZCARD', zkey)
    local excess = cnt + 1 - limit
    if excess > 0 then
        local olds = redis.call('ZRANGE', zkey, 0, excess - 1)
        for _, sid in ipairs(olds) do
            local m = redis.call('HGET', mkey, sid)
            if m then table.insert(removed, m) end
        end
        if #olds > 0 then
            redis.call('ZREM', zkey, unpack(olds))
            redis.call('HDEL', mkey, unpack(olds))
        end
    end
end
redis.call('ZADD', zkey, score, sessionId)
redis.call('HSET', mkey, sessionId, meta)
redis.call('EXPIRE', zkey, ttl)
redis.call('EXPIRE', mkey, ttl)
redis.call('SET', lkey, ARGV[3], 'EX', ttl)
if #removed == 0 then return '[]' end
return '[' .. table.concat(removed, ',') .. ']'
`

// 上限调小后立即清理超出部分, 保留最近登录的会话
const rebalanceSessionScript = `
local zkey, mkey = KEYS[1], KEYS[2]
local limit = tonumber(ARGV[1])
local removed = {}
local cnt = redis.call('ZCARD', zkey)
local excess = cnt - limit
if limit > 0 and excess > 0 then
    local olds = redis.call('ZRANGE', zkey, 0, excess - 1)
    for _, sid in ipairs(olds) do
        local m = redis.call('HGET', mkey, sid)
        if m then table.insert(removed, m) end
    end
    redis.call('ZREM', zkey, unpack(olds))
    redis.call('HDEL', mkey, unpack(olds))
end
if #removed == 0 then return '[]' end
return '[' .. table.concat(removed, ',') .. ']'
`

// 管理员强制下线: 清空该用户全部会话登记
const kickSessionScript = `
local zkey, mkey = KEYS[1], KEYS[2]
local removed = {}
local ids = redis.call('ZRANGE', zkey, 0, -1)
for _, sid in ipairs(ids) do
    local m = redis.call('HGET', mkey, sid)
    if m then table.insert(removed, m) end
end
if #ids > 0 then
    redis.call('DEL', zkey)
    redis.call('DEL', mkey)
end
if #removed == 0 then return '[]' end
return '[' .. table.concat(removed, ',') .. ']'
`

// RegisterSession 原子登记登录会话, 返回本次被淘汰的会话。
// 淘汰顺序: 先清理超过 refresh 生命周期的陈旧登记, 再按登录时间从早到晚淘汰。
func (s *Service) RegisterSession(ctx context.Context, in RegisterInput) (evicted []SessionMeta, err error) {
	if in.Limit <= 0 {
		in.Limit = 1
	}
	if in.LoginAt.IsZero() {
		in.LoginAt = time.Now()
	}
	if in.Meta.SessionId == "" {
		in.Meta.SessionId = in.SessionId
	}
	if in.Meta.LoginTime == "" {
		in.Meta.LoginTime = gtime.New(in.LoginAt).Format("Y-m-d H:i:s")
	}
	meta, err := json.Marshal(in.Meta)
	if err != nil {
		return nil, gerror.Wrap(err, "序列化会话摘要失败")
	}
	uid := gconv.String(in.UserId)
	v, err := g.Redis().Do(ctx, "EVAL", registerSessionScript, 3,
		consts.RedisKeySessionSet+uid,
		consts.RedisKeySessionMeta+uid,
		consts.RedisKeySessionLimit+uid,
		in.SessionId,
		in.LoginAt.UnixMilli(),
		in.Limit,
		consts.RefreshTTL,
		string(meta),
		time.Now().UnixMilli(),
	)
	if err != nil {
		return nil, gerror.Wrap(err, "登记登录会话失败")
	}
	evicted = parseSessionList(v.String())
	if in.Reason != "" {
		s.markEvicted(ctx, evicted, in.Reason)
	}
	return evicted, nil
}

// SessionAlive 判断会话是否仍在有效集合中; 不在集合中时返回被踢原因(为空表示普通失效)
func (s *Service) SessionAlive(ctx context.Context, userId uint64, sessionId string) (alive bool, reason string, err error) {
	if sessionId == "" {
		return false, "", nil
	}
	uid := gconv.String(userId)
	v, err := g.Redis().Do(ctx, "ZSCORE", consts.RedisKeySessionSet+uid, sessionId)
	if err != nil {
		return false, "", gerror.Wrap(err, "查询登录会话失败")
	}
	if !v.IsNil() && !v.IsEmpty() {
		return true, "", nil
	}
	r, err := g.Redis().Do(ctx, "GET", consts.RedisKeySessionKick+sessionId)
	if err != nil {
		return false, "", nil
	}
	if r.IsNil() {
		return false, "", nil
	}
	return false, r.String(), nil
}

// UpdateSessionAccess 刷新后把会话摘要中的当前 accessToken 更新为新值(不影响会话生命)
func (s *Service) UpdateSessionAccess(ctx context.Context, userId uint64, sessionId, access string) {
	if sessionId == "" {
		return
	}
	uid := gconv.String(userId)
	v, err := g.Redis().Do(ctx, "HGET", consts.RedisKeySessionMeta+uid, sessionId)
	if err != nil || v.IsNil() {
		return
	}
	var m SessionMeta
	if json.Unmarshal([]byte(v.String()), &m) != nil {
		return
	}
	m.Access = access
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	_, _ = g.Redis().Do(ctx, "HSET", consts.RedisKeySessionMeta+uid, sessionId, string(b))
}

// RevokeRefresh 作废某个 refreshToken(会话被淘汰/强制下线时调用)
func (s *Service) RevokeRefresh(ctx context.Context, refresh string) {
	if refresh == "" {
		return
	}
	_, _ = g.Redis().Do(ctx, "DEL", consts.RedisKeyRefreshToken+refresh)
}

// SetLimit 记录用户当前生效的设备数上限(登录与超管修改时写入)
func (s *Service) SetLimit(ctx context.Context, userId uint64, limit int) error {
	_, err := g.Redis().Do(ctx, "SET", consts.RedisKeySessionLimit+gconv.String(userId), limit, "EX", consts.RefreshTTL)
	if err != nil {
		return gerror.Wrap(err, "写入设备数上限失败")
	}
	return nil
}

// Rebalance 上限调小后立即淘汰多余会话(保留最近登录的), 返回被淘汰会话
func (s *Service) Rebalance(ctx context.Context, userId uint64, limit int, reason string) ([]SessionMeta, error) {
	uid := gconv.String(userId)
	v, err := g.Redis().Do(ctx, "EVAL", rebalanceSessionScript, 2,
		consts.RedisKeySessionSet+uid, consts.RedisKeySessionMeta+uid, limit)
	if err != nil {
		return nil, gerror.Wrap(err, "清理超出上限的会话失败")
	}
	evicted := parseSessionList(v.String())
	s.markEvicted(ctx, evicted, reason)
	return evicted, nil
}

// KickUserSessions 强制下线某用户的全部会话登记, 返回被清理的会话
func (s *Service) KickUserSessions(ctx context.Context, userId uint64, reason string) ([]SessionMeta, error) {
	uid := gconv.String(userId)
	v, err := g.Redis().Do(ctx, "EVAL", kickSessionScript, 2,
		consts.RedisKeySessionSet+uid, consts.RedisKeySessionMeta+uid)
	if err != nil {
		return nil, gerror.Wrap(err, "清理用户会话失败")
	}
	evicted := parseSessionList(v.String())
	s.markEvicted(ctx, evicted, reason)
	return evicted, nil
}

// RevokeSessionTokens 彻底删除被清理会话的 access/refresh token(用于强制改密等需要立即失效的场景)
func (s *Service) RevokeSessionTokens(ctx context.Context, sessions []SessionMeta) {
	for _, m := range sessions {
		if m.Access != "" {
			_, _ = g.Redis().Do(ctx, "DEL", consts.RedisKeyAccessToken+m.Access)
		}
		if m.Refresh != "" {
			_, _ = g.Redis().Do(ctx, "DEL", consts.RedisKeyRefreshToken+m.Refresh)
		}
	}
}

// markEvicted 记录被踢原因并作废 refreshToken。
// accessToken 记录保留到自然过期, 这样旧端下一次请求能读到具体下线原因;
// 会话已从集合移除, 该 accessToken 一律无法通过校验, 也不会被刷新续命。
func (s *Service) markEvicted(ctx context.Context, evicted []SessionMeta, reason string) {
	for _, m := range evicted {
		s.RevokeRefresh(ctx, m.Refresh)
		if m.SessionId != "" {
			_, _ = g.Redis().Do(ctx, "SET", consts.RedisKeySessionKick+m.SessionId, reason, "EX", consts.SessionKickReasonTTL)
		}
	}
}

func parseSessionList(raw string) []SessionMeta {
	if raw == "" || raw == "null" {
		return nil
	}
	var list []SessionMeta
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}
	return list
}

// LimitEnabled 是否启用登录设备数限制
func (s *Service) LimitEnabled(ctx context.Context) bool {
	return g.Cfg().MustGet(ctx, "security.sessionLimit.enabled", true).Bool()
}

// EffectiveLimit 用户维度上限为空(0)时取全局默认值
func (s *Service) EffectiveLimit(ctx context.Context, userMax int) int {
	if userMax > 0 {
		return userMax
	}
	limit := g.Cfg().MustGet(ctx, "security.sessionLimit.defaultMaxDevices", 1).Int()
	if limit <= 0 {
		limit = 1
	}
	return limit
}

// KickedMessage 被新设备顶替时的提示文案
func (s *Service) KickedMessage(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "security.sessionLimit.kickedMessage", "账号已在其他设备登录, 请重新登录").String()
}

// LimitExceededMessage 超出设备数上限时的提示文案
func (s *Service) LimitExceededMessage(ctx context.Context) string {
	return g.Cfg().MustGet(ctx, "security.sessionLimit.limitExceededMessage", "已超出允许的登录设备数, 请重新登录").String()
}

// KickedReasonForNewLogin 新登录淘汰旧会话时的下线原因: 单设备场景提示"其他设备登录", 多设备上限提示"超出设备数"
func (s *Service) KickedReasonForNewLogin(ctx context.Context, limit int) string {
	if limit <= 1 {
		return s.KickedMessage(ctx)
	}
	return s.LimitExceededMessage(ctx)
}

// AdminKickMessage 管理员强制下线提示文案
func AdminKickMessage() string {
	return "账号已被管理员强制下线, 请重新登录"
}
