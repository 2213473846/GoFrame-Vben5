package token

import (
	"context"
	"encoding/json"
	"time"

	"demo/internal/consts"
	"demo/internal/service/identity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/guid"
)

// Service 基于 Redis 的 token 与登录会话管理服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// SessionTokens 一次登录签发的一组凭据(同一会话, 刷新只换 accessToken)
type SessionTokens struct {
	SessionId string
	Access    string
	Refresh   string
}

// RefreshInfo refreshToken 解析结果
type RefreshInfo struct {
	UserId    uint64
	SessionId string
}

// refreshPayload refreshToken 存储内容: 用户 + 会话, 用于刷新时定位会话
type refreshPayload struct {
	UserId    uint64 `json:"userId"`
	SessionId string `json:"sessionId"`
}

// CreateSession 登录时签发 accessToken + refreshToken, 并分配会话ID。
// refreshToken 与 accessToken 均绑定同一 sessionId, 刷新时只替换 accessToken。
func (s *Service) CreateSession(ctx context.Context, in identity.Identity) (SessionTokens, error) {
	if in.SessionId == "" {
		in.SessionId = guid.S()
	}
	access, refresh := guid.S(), guid.S()
	b, err := json.Marshal(in)
	if err != nil {
		return SessionTokens{}, gerror.Wrap(err, "序列化用户身份失败")
	}
	rp, err := json.Marshal(refreshPayload{UserId: in.Id, SessionId: in.SessionId})
	if err != nil {
		return SessionTokens{}, gerror.Wrap(err, "序列化会话信息失败")
	}
	redis := g.Redis()
	if _, err = redis.Do(ctx, "SET", consts.RedisKeyAccessToken+access, b, "EX", consts.AccessTTL); err != nil {
		return SessionTokens{}, gerror.Wrap(err, "写入 accessToken 失败")
	}
	if _, err = redis.Do(ctx, "SET", consts.RedisKeyRefreshToken+refresh, rp, "EX", consts.RefreshTTL); err != nil {
		_, _ = redis.Do(ctx, "DEL", consts.RedisKeyAccessToken+access)
		return SessionTokens{}, gerror.Wrap(err, "写入 refreshToken 失败")
	}
	return SessionTokens{SessionId: in.SessionId, Access: access, Refresh: refresh}, nil
}

// NewAccess 刷新时签发新的 accessToken(会话不变, refreshToken 不变)
func (s *Service) NewAccess(ctx context.Context, in identity.Identity) (string, error) {
	access := guid.S()
	b, err := json.Marshal(in)
	if err != nil {
		return "", gerror.Wrap(err, "序列化用户身份失败")
	}
	if _, err = g.Redis().Do(ctx, "SET", consts.RedisKeyAccessToken+access, b, "EX", consts.AccessTTL); err != nil {
		return "", gerror.Wrap(err, "写入 accessToken 失败")
	}
	return access, nil
}

// Verify 校验 accessToken, 返回用户身份。
// 启用登录设备数限制时, 还会校验该 token 所属会话仍在有效会话集合内;
// 被顶替的会话返回 CodeSessionKicked 与具体下线原因。
func (s *Service) Verify(ctx context.Context, access string) (*identity.Identity, error) {
	if access == "" {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
	}
	v, err := g.Redis().Do(ctx, "GET", consts.RedisKeyAccessToken+access)
	if err != nil {
		return nil, gerror.Wrap(err, "校验 token 失败")
	}
	if v.IsNil() || v.IsEmpty() {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "登录已过期, 请重新登录")
	}
	var id identity.Identity
	if err = json.Unmarshal([]byte(v.String()), &id); err != nil {
		return nil, gerror.WrapCode(consts.CodeUnauthorized, err, "解析用户身份失败")
	}
	if id.SessionId != "" && s.LimitEnabled(ctx) {
		alive, reason, err := s.SessionAlive(ctx, id.Id, id.SessionId)
		if err != nil {
			return nil, err
		}
		if !alive {
			if reason != "" {
				return nil, gerror.NewCode(consts.CodeSessionKicked, reason)
			}
			return nil, gerror.NewCode(consts.CodeUnauthorized, "登录已失效, 请重新登录")
		}
	}
	return &id, nil
}

// ReadRefresh 读取 refreshToken 对应的用户与会话(不消费, 同一浏览器的多个标签页可共用)。
func (s *Service) ReadRefresh(ctx context.Context, refresh string) (*RefreshInfo, error) {
	if refresh == "" {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "无效的 refreshToken")
	}
	v, err := g.Redis().Do(ctx, "GET", consts.RedisKeyRefreshToken+refresh)
	if err != nil {
		return nil, gerror.Wrap(err, "校验 refreshToken 失败")
	}
	if v.IsNil() || v.IsEmpty() {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "登录已过期, 请重新登录")
	}
	var p refreshPayload
	if err = json.Unmarshal([]byte(v.String()), &p); err != nil {
		// 兼容旧格式(仅存 userId), 旧会话需重新登录
		return &RefreshInfo{UserId: gconv.Uint64(v.String())}, nil
	}
	if p.UserId == 0 {
		return nil, gerror.NewCode(consts.CodeUnauthorized, "登录已过期, 请重新登录")
	}
	return &RefreshInfo{UserId: p.UserId, SessionId: p.SessionId}, nil
}

// RemoveSession 登出: 删除本会话的 token 与注册信息。
// 只按 sessionId 精确清理, 不会误删同一用户其他设备/之后新建的会话。
func (s *Service) RemoveSession(ctx context.Context, access, refresh string) (sessionId string) {
	redis := g.Redis()
	var userId uint64
	if refresh != "" {
		if v, err := redis.Do(ctx, "GET", consts.RedisKeyRefreshToken+refresh); err == nil && !v.IsNil() {
			var p refreshPayload
			if json.Unmarshal([]byte(v.String()), &p) == nil {
				userId, sessionId = p.UserId, p.SessionId
			}
		}
	}
	if access != "" {
		if v, err := redis.Do(ctx, "GET", consts.RedisKeyAccessToken+access); err == nil && !v.IsNil() {
			var id identity.Identity
			if json.Unmarshal([]byte(v.String()), &id) == nil {
				if sessionId == "" {
					sessionId = id.SessionId
				}
				if userId == 0 {
					userId = id.Id
				}
			}
		}
		_, _ = redis.Do(ctx, "DEL", consts.RedisKeyAccessToken+access)
	}
	if refresh != "" {
		_, _ = redis.Do(ctx, "DEL", consts.RedisKeyRefreshToken+refresh)
	}
	if sessionId != "" && userId > 0 {
		_, _ = redis.Do(ctx, "ZREM", consts.RedisKeySessionSet+gconv.String(userId), sessionId)
		_, _ = redis.Do(ctx, "HDEL", consts.RedisKeySessionMeta+gconv.String(userId), sessionId)
	}
	return sessionId
}

// AccessTTLDuration accessToken 有效期(用于 Cookie 设置)
func AccessTTLDuration() time.Duration {
	return time.Duration(consts.AccessTTL) * time.Second
}
