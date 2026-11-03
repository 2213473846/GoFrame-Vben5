package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/identity"
	"demo/internal/service/online"
	"demo/internal/service/security"
	"demo/internal/service/token"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"golang.org/x/crypto/bcrypt"
)

// Service 认证服务: 登录/刷新/登出/修改密码
type Service struct {
	tokenSvc *token.Service
	secSvc   *security.Service
}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{
		tokenSvc: token.New(),
		secSvc:   security.New(),
	}
}

// LoginMeta 登录设备信息(写入在线会话与会话登记, 用于展示设备指纹)
type LoginMeta struct {
	Ip       string
	Browser  string
	Os       string
	Location string
}

// LoginInput 登录参数; 验证码字段可选, 仅在触发阈值后由后端要求
type LoginInput struct {
	Username    string
	Password    string
	CaptchaId   string
	CaptchaCode string
	Meta        LoginMeta
}

// LoginResult 登录/刷新结果
type LoginResult struct {
	AccessToken        string
	RefreshToken       string
	SessionId          string
	Identity           identity.Identity
	MustChangePassword bool
}

// Login 用户名密码登录。
// 安全流程: 锁定检查 -> 验证码阈值检查/校验 -> 密码校验 -> 失败计数/锁定 -> 成功后清空计数。
// 启用登录设备数限制时, 按用户上限淘汰最早建立的会话, 被淘汰端下一次请求会收到被踢错误码。
func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	username := in.Username
	ip := in.Meta.Ip

	// 1) 锁定期内直接拒绝, 即使密码正确; 不再叠加失败计数, 避免无限延长
	if locked, msg := s.secSvc.LoginLocked(ctx, username, ip); locked {
		return nil, gerror.NewCode(consts.CodeLoginLocked, msg)
	}

	// 2) 失败次数达到较低阈值后要求图形验证码
	if s.secSvc.CaptchaRequired(ctx, username, ip) {
		if in.CaptchaId == "" || in.CaptchaCode == "" {
			return nil, s.secSvc.CaptchaRequiredError(ctx)
		}
		if !s.secSvc.VerifyCaptcha(ctx, in.CaptchaId, in.CaptchaCode) {
			// 验证码校验失败同样计入失败锁定
			if s.secSvc.RecordFailure(ctx, username, ip) {
				return nil, s.secSvc.LockedError(ctx)
			}
			return nil, s.secSvc.CaptchaInvalidError(ctx)
		}
	}

	// 3) 用户与密码校验; 用户不存在/密码错误统一文案, 不暴露账号是否存在
	var user entity.SysUser
	if err := dao.SysUser.Ctx(ctx).Where(do.SysUser{Username: username}).Scan(&user); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询用户失败")
		}
	}
	if user.Id == 0 || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)) != nil {
		if s.secSvc.RecordFailure(ctx, username, ip) {
			return nil, s.secSvc.LockedError(ctx)
		}
		return nil, gerror.NewCode(consts.CodeForbidden, "用户名或密码错误")
	}
	if user.Status != 1 {
		return nil, gerror.NewCode(consts.CodeForbidden, "账号已被禁用, 请联系管理员")
	}

	// 4) 存量弱密码(不满足当前策略)在登录成功时标记强制改密, 覆盖种子账号与历史数据
	mustChange := user.MustChangePassword == 1
	if !mustChange && s.secSvc.ValidatePassword(ctx, in.Password) != nil {
		mustChange = true
		if _, err := dao.SysUser.Ctx(ctx).
			Where(dao.SysUser.Columns().Id, user.Id).
			Data(g.Map{dao.SysUser.Columns().MustChangePassword: 1}).
			Update(); err != nil {
			g.Log().Warningf(ctx, "标记强制改密失败 userId=%d: %v", user.Id, err)
		}
	}

	// 5) 登录成功: 清空失败计数与锁定标记
	s.secSvc.Reset(ctx, username, ip)

	id := identity.Identity{
		Id:       user.Id,
		Pid:      user.Pid,
		Username: user.Username,
		Role:     user.Role,
		RealName: user.RealName,
		Avatar:   user.Avatar,
	}
	tokens, err := s.tokenSvc.CreateSession(ctx, id)
	if err != nil {
		return nil, err
	}
	id.SessionId = tokens.SessionId
	limit := s.tokenSvc.EffectiveLimit(ctx, user.MaxDevices)
	// 先写入在线会话, 再登记有效会话集合: 并发登录淘汰时不会留下僵尸在线记录
	online.Record(ctx, &online.OnlineUser{
		UserId:    id.Id,
		SessionId: tokens.SessionId,
		Username:  id.Username,
		RealName:  id.RealName,
		Role:      id.Role,
		Ip:        in.Meta.Ip,
		Location:  in.Meta.Location,
		Browser:   in.Meta.Browser,
		Os:        in.Meta.Os,
		LoginTime: gtime.Now().Format("Y-m-d H:i:s"),
	})
	if s.tokenSvc.LimitEnabled(ctx) {
		reason := s.tokenSvc.KickedReasonForNewLogin(ctx, limit)
		evicted, err := s.tokenSvc.RegisterSession(ctx, token.RegisterInput{
			UserId:    id.Id,
			SessionId: tokens.SessionId,
			Access:    tokens.Access,
			Refresh:   tokens.Refresh,
			Limit:     limit,
			LoginAt:   time.Now(),
			Reason:    reason,
			Meta: token.SessionMeta{
				SessionId: tokens.SessionId,
				Access:    tokens.Access,
				Refresh:   tokens.Refresh,
				Ip:        in.Meta.Ip,
				Browser:   in.Meta.Browser,
				Os:        in.Meta.Os,
				Location:  in.Meta.Location,
			},
		})
		if err != nil {
			s.tokenSvc.RemoveSession(ctx, tokens.Access, tokens.Refresh)
			online.Drop(ctx, tokens.SessionId)
			return nil, err
		}
		// 被淘汰的会话已作废 refreshToken 并记录下线原因, 同步标记在线记录
		for _, e := range evicted {
			online.MarkOffline(ctx, e.SessionId, reason)
		}
	}
	return &LoginResult{
		AccessToken:        tokens.Access,
		RefreshToken:       tokens.Refresh,
		SessionId:          tokens.SessionId,
		Identity:           id,
		MustChangePassword: mustChange,
	}, nil
}

// Refresh 使用 refreshToken 换发新的 accessToken。
// 刷新不改变会话ID与 refreshToken, 因此同一浏览器多标签页并发刷新不会互相顶掉;
// 已被新登录淘汰的会话 refreshToken 已删除, 无法续命。
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*LoginResult, error) {
	info, err := s.tokenSvc.ReadRefresh(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	var user entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(do.SysUser{Id: info.UserId}).Scan(&user); err != nil {
		// 用户已被删除等场景, Scan 返回 ErrNoRows 时按账号不可用处理
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询用户失败")
		}
	}
	if user.Id == 0 || user.Status != 1 {
		return nil, gerror.NewCode(consts.CodeForbidden, "账号不可用, 请重新登录")
	}
	if s.tokenSvc.LimitEnabled(ctx) {
		alive, reason, err := s.tokenSvc.SessionAlive(ctx, user.Id, info.SessionId)
		if err != nil {
			return nil, err
		}
		if !alive {
			if reason != "" {
				return nil, gerror.NewCode(consts.CodeSessionKicked, reason)
			}
			return nil, gerror.NewCode(consts.CodeUnauthorized, "登录状态已失效, 请重新登录")
		}
	}
	id := identity.Identity{
		Id:        user.Id,
		Pid:       user.Pid,
		Username:  user.Username,
		Role:      user.Role,
		RealName:  user.RealName,
		Avatar:    user.Avatar,
		SessionId: info.SessionId,
	}
	access, err := s.tokenSvc.NewAccess(ctx, id)
	if err != nil {
		return nil, err
	}
	s.tokenSvc.UpdateSessionAccess(ctx, id.Id, id.SessionId, access)
	online.Touch(ctx, id.SessionId)
	// refreshToken 保持原值, 避免多标签并发刷新时一方拿到已消费的旧 refresh
	return &LoginResult{
		AccessToken:  access,
		RefreshToken: refreshToken,
		SessionId:    id.SessionId,
		Identity:     id,
	}, nil
}

// Logout 登出, 作废当前会话的 token 与注册信息, 返回会话ID供在线模块清理
func (s *Service) Logout(ctx context.Context, accessToken string, refreshToken string) string {
	return s.tokenSvc.RemoveSession(ctx, accessToken, refreshToken)
}

// ChangePassword 修改当前用户密码。
// 改密成功后: 清空强制改密标记, 并让该用户全部旧会话立即失效(失效重登), 避免旧 token 继续使用;
// 会话注册由 token 服务统一清理, 不会残留登记导致设备数上限错乱。
func (s *Service) ChangePassword(ctx context.Context, userId uint64, oldPassword, newPassword string) error {
	var user entity.SysUser
	if err := dao.SysUser.Ctx(ctx).Where(do.SysUser{Id: userId}).Scan(&user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return gerror.NewCode(consts.CodeNotFound, "用户不存在")
		}
		return gerror.Wrap(err, "查询用户失败")
	}
	if user.Id == 0 {
		return gerror.NewCode(consts.CodeNotFound, "用户不存在")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)) != nil {
		return gerror.NewCode(consts.CodeForbidden, "原密码错误")
	}
	if oldPassword == newPassword {
		return gerror.NewCode(consts.CodeBadRequest, "新密码不能与原密码相同")
	}
	if err := s.secSvc.ValidatePassword(ctx, newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return gerror.Wrap(err, "生成密码失败")
	}
	if _, err = dao.SysUser.Ctx(ctx).
		Where(dao.SysUser.Columns().Id, userId).
		Data(g.Map{
			dao.SysUser.Columns().Password:           string(hash),
			dao.SysUser.Columns().MustChangePassword: 0,
		}).
		Update(); err != nil {
		return gerror.Wrap(err, "更新密码失败")
	}
	// 改密后旧会话失效重登: 清理会话登记 + 作废 access/refresh, 并标记在线记录下线
	reason := "密码已修改, 请重新登录"
	evicted, err := s.tokenSvc.KickUserSessions(ctx, userId, reason)
	if err != nil {
		g.Log().Warningf(ctx, "改密后清理旧会话失败 userId=%d: %v", userId, err)
	}
	s.tokenSvc.RevokeSessionTokens(ctx, evicted)
	for _, e := range evicted {
		online.MarkOffline(ctx, e.SessionId, reason)
	}
	return nil
}

// Codes 按角色返回权限码
func (s *Service) Codes(role string) []string {
	codes := consts.RoleCodes[role]
	if codes == nil {
		return []string{}
	}
	return codes
}
