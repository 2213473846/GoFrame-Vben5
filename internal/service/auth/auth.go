package auth

import (
	"context"
	"database/sql"
	"errors"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/identity"
	"demo/internal/service/token"

	"github.com/gogf/gf/v2/errors/gerror"
	"golang.org/x/crypto/bcrypt"
)

// Service 认证服务: 登录/刷新/登出
type Service struct {
	tokenSvc *token.Service
}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{
		tokenSvc: token.New(),
	}
}

// LoginResult 登录/刷新结果
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	Identity     identity.Identity
}

// Login 用户名密码登录
func (s *Service) Login(ctx context.Context, username string, password string) (*LoginResult, error) {
	var user entity.SysUser
	if err := dao.SysUser.Ctx(ctx).Where(do.SysUser{Username: username}).Scan(&user); err != nil {
		// 用户不存在时统一按用户名或密码错误处理, 避免暴露账号是否存在
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询用户失败")
		}
	}
	if user.Id == 0 {
		return nil, gerror.NewCode(consts.CodeForbidden, "用户名或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, gerror.NewCode(consts.CodeForbidden, "用户名或密码错误")
	}
	if user.Status != 1 {
		return nil, gerror.NewCode(consts.CodeForbidden, "账号已被禁用, 请联系管理员")
	}
	id := identity.Identity{
		Id:       user.Id,
		Pid:      user.Pid,
		Username: user.Username,
		Role:     user.Role,
		RealName: user.RealName,
		Avatar:   user.Avatar,
	}
	access, refresh, err := s.tokenSvc.Create(ctx, id)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		AccessToken:  access,
		RefreshToken: refresh,
		Identity:     id,
	}, nil
}

// Refresh 使用 refreshToken 换发新 token
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*LoginResult, error) {
	userId, err := s.tokenSvc.TakeRefresh(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	var user entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(do.SysUser{Id: userId}).Scan(&user); err != nil {
		// 用户已被删除等场景, Scan 返回 ErrNoRows 时按账号不可用处理
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询用户失败")
		}
	}
	if user.Id == 0 || user.Status != 1 {
		return nil, gerror.NewCode(consts.CodeForbidden, "账号不可用, 请重新登录")
	}
	id := identity.Identity{
		Id:       user.Id,
		Pid:      user.Pid,
		Username: user.Username,
		Role:     user.Role,
		RealName: user.RealName,
		Avatar:   user.Avatar,
	}
	access, newRefresh, err := s.tokenSvc.Create(ctx, id)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		AccessToken:  access,
		RefreshToken: newRefresh,
		Identity:     id,
	}, nil
}

// Logout 登出, 作废 token
func (s *Service) Logout(ctx context.Context, accessToken string, refreshToken string) {
	s.tokenSvc.Remove(ctx, accessToken, refreshToken)
}

// Codes 按角色返回权限码
func (s *Service) Codes(role string) []string {
	codes := consts.RoleCodes[role]
	if codes == nil {
		return []string{}
	}
	return codes
}
