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

// Service 基于 Redis 的 token 管理服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// Create 为用户签发 accessToken 与 refreshToken
func (s *Service) Create(ctx context.Context, in identity.Identity) (access string, refresh string, err error) {
	access, refresh = guid.S(), guid.S()
	b, err := json.Marshal(in)
	if err != nil {
		return "", "", gerror.Wrap(err, "序列化用户身份失败")
	}
	redis := g.Redis()
	if _, err = redis.Do(ctx, "SET", consts.RedisKeyAccessToken+access, b, "EX", consts.AccessTTL); err != nil {
		return "", "", gerror.Wrap(err, "写入 accessToken 失败")
	}
	if _, err = redis.Do(ctx, "SET", consts.RedisKeyRefreshToken+refresh, in.Id, "EX", consts.RefreshTTL); err != nil {
		return "", "", gerror.Wrap(err, "写入 refreshToken 失败")
	}
	return access, refresh, nil
}

// Verify 校验 accessToken, 返回用户身份
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
	return &id, nil
}

// TakeRefresh 消费 refreshToken(一次性), 返回对应用户ID
func (s *Service) TakeRefresh(ctx context.Context, refresh string) (uint64, error) {
	if refresh == "" {
		return 0, gerror.NewCode(consts.CodeForbidden, "无效的 refreshToken")
	}
	v, err := g.Redis().Do(ctx, "GETDEL", consts.RedisKeyRefreshToken+refresh)
	if err != nil {
		return 0, gerror.Wrap(err, "校验 refreshToken 失败")
	}
	if v.IsNil() || v.IsEmpty() {
		return 0, gerror.NewCode(consts.CodeForbidden, "登录已过期, 请重新登录")
	}
	return gconv.Uint64(v.String()), nil
}

// Remove 删除 accessToken 与 refreshToken
func (s *Service) Remove(ctx context.Context, access string, refresh string) {
	if access != "" {
		_, _ = g.Redis().Do(ctx, "DEL", consts.RedisKeyAccessToken+access)
	}
	if refresh != "" {
		_, _ = g.Redis().Do(ctx, "DEL", consts.RedisKeyRefreshToken+refresh)
	}
}

// AccessTTLDuration accessToken 有效期(用于 Cookie 设置)
func AccessTTLDuration() time.Duration {
	return time.Duration(consts.AccessTTL) * time.Second
}
