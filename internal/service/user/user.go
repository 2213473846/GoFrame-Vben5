package user

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/authz"
	dictsvc "demo/internal/service/dict"
	exportsvc "demo/internal/service/export"
	"demo/internal/service/identity"
	onlinesvc "demo/internal/service/online"
	rolesvc "demo/internal/service/role"
	secsvc "demo/internal/service/security"
	tokensvc "demo/internal/service/token"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"golang.org/x/crypto/bcrypt"
)

// Service 用户/管理员管理服务(含数据权限控制)
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// ListInput 列表查询条件
type ListInput struct {
	Page     int
	PageSize int
	Username string
	RealName string
	Role     string
	Status   *int
}

// List 用户列表
// 数据权限: 超管可见全部; 管理员(admin)仅可见其子孙(不含自己); 其他角色无权访问
// 返回 parents: pid -> 上级用户信息(用于列表展示)
// 返回 userRoles: userId -> 功能角色列表(用于列表展示与编辑回显)
func (s *Service) List(ctx context.Context, in ListInput) (users []*entity.SysUser, parents map[uint64]*entity.SysUser, userRoles map[uint64][]*entity.SysRole, total int, err error) {
	m, empty, err := s.buildListQuery(ctx, in)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	if empty {
		return []*entity.SysUser{}, map[uint64]*entity.SysUser{}, map[uint64][]*entity.SysRole{}, 0, nil
	}
	cols := dao.SysUser.Columns()
	if total, err = m.Count(); err != nil {
		return nil, nil, nil, 0, gerror.Wrap(err, "统计用户数量失败")
	}
	if err = m.Page(in.Page, in.PageSize).OrderDesc(cols.Id).Scan(&users); err != nil {
		return nil, nil, nil, 0, gerror.Wrap(err, "查询用户列表失败")
	}
	// 批量查询上级用户信息
	parents = make(map[uint64]*entity.SysUser, len(users))
	pidSet := make(map[uint64]struct{}, len(users))
	for _, u := range users {
		if u.Pid > 0 {
			pidSet[u.Pid] = struct{}{}
		}
	}
	if len(pidSet) > 0 {
		pids := make([]uint64, 0, len(pidSet))
		for pid := range pidSet {
			pids = append(pids, pid)
		}
		var parentList []*entity.SysUser
		if err = dao.SysUser.Ctx(ctx).WhereIn(cols.Id, pids).Scan(&parentList); err != nil {
			return nil, nil, nil, 0, gerror.Wrap(err, "查询上级用户失败")
		}
		for _, p := range parentList {
			parents[p.Id] = p
		}
	}
	// 批量查询功能角色
	userIds := make([]uint64, 0, len(users))
	for _, u := range users {
		userIds = append(userIds, u.Id)
	}
	if userRoles, err = (rolesvc.New()).UserRoles(ctx, userIds); err != nil {
		return nil, nil, nil, 0, err
	}
	return users, parents, userRoles, total, nil
}

// buildListQuery 构建用户列表查询(数据权限 + 筛选条件), 供列表与导出复用
// empty=true 表示当前操作人数据权限范围内没有任何用户, 无需查库
func (s *Service) buildListQuery(ctx context.Context, in ListInput) (m *gdb.Model, empty bool, err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, false, err
	}
	cols := dao.SysUser.Columns()
	m = dao.SysUser.Ctx(ctx)
	if !actor.IsSuperAdmin() {
		if actor.Role != consts.RoleAdmin {
			return nil, false, gerror.NewCode(consts.CodeForbidden, "无权限查看用户列表")
		}
		ids, err := s.descendantIds(ctx, actor.Id)
		if err != nil {
			return nil, false, err
		}
		if len(ids) == 0 {
			return nil, true, nil
		}
		m = m.WhereIn(cols.Id, ids)
	}
	if in.Username != "" {
		m = m.WhereLike(cols.Username, "%"+in.Username+"%")
	}
	if in.RealName != "" {
		m = m.WhereLike(cols.RealName, "%"+in.RealName+"%")
	}
	if in.Role != "" {
		m = m.Where(cols.Role, in.Role)
	}
	if in.Status != nil {
		m = m.Where(cols.Status, *in.Status)
	}
	return m, false, nil
}

// Export 导出用户列表(CSV/XLSX)
// 复用 buildListQuery 的筛选条件与数据权限: 管理员导出范围与列表完全一致(仅子孙)
func (s *Service) Export(ctx context.Context, in ListInput, format string, w io.Writer) (count int, err error) {
	format, err = exportsvc.NormalizeFormat(format)
	if err != nil {
		return 0, gerror.NewCode(consts.CodeBadRequest, err.Error())
	}
	m, empty, err := s.buildListQuery(ctx, in)
	if err != nil {
		return 0, err
	}
	cols := dao.SysUser.Columns()
	// 状态展示优先取字典, 取不到时回退内置文案
	statusLabels := map[int]string{1: "启用", 0: "禁用"}
	if options, err := dictsvc.New().Options(ctx, []string{dictsvc.TypeUserStatus}); err == nil {
		for _, item := range options[dictsvc.TypeUserStatus] {
			statusLabels[gconv.Int(item.Value)] = item.Label
		}
	}
	headers := []string{"ID", "用户名", "姓名", "角色", "状态", "上级用户", "功能角色", "创建时间", "备注"}
	next := func(offset, limit int) ([][]string, error) {
		if empty {
			return nil, nil
		}
		var users []*entity.SysUser
		if scanErr := m.Limit(offset, limit).OrderDesc(cols.Id).Scan(&users); scanErr != nil {
			if errors.Is(scanErr, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, gerror.Wrap(scanErr, "查询导出数据失败")
		}
		if len(users) == 0 {
			return nil, nil
		}
		// 批量查询上级用户名
		parentNames := make(map[uint64]string, len(users))
		pidSet := make(map[uint64]struct{}, len(users))
		userIds := make([]uint64, 0, len(users))
		for _, u := range users {
			userIds = append(userIds, u.Id)
			if u.Pid > 0 {
				pidSet[u.Pid] = struct{}{}
			}
		}
		if len(pidSet) > 0 {
			pids := make([]uint64, 0, len(pidSet))
			for pid := range pidSet {
				pids = append(pids, pid)
			}
			var parents []*entity.SysUser
			if scanErr := dao.SysUser.Ctx(ctx).WhereIn(cols.Id, pids).Fields(cols.Id, cols.Username).Scan(&parents); scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
				return nil, gerror.Wrap(scanErr, "查询上级用户失败")
			}
			for _, p := range parents {
				parentNames[p.Id] = p.Username
			}
		}
		userRoles, roleErr := (rolesvc.New()).UserRoles(ctx, userIds)
		if roleErr != nil {
			return nil, roleErr
		}
		rows := make([][]string, 0, len(users))
		for _, u := range users {
			roleNames := make([]string, 0, len(userRoles[u.Id]))
			for _, r := range userRoles[u.Id] {
				roleNames = append(roleNames, r.Name)
			}
			createdAt := ""
			if u.CreatedAt != nil {
				createdAt = u.CreatedAt.String()
			}
			rows = append(rows, []string{
				gconv.String(u.Id),
				u.Username,
				u.RealName,
				u.Role,
				statusLabels[u.Status],
				parentNames[u.Pid],
				strings.Join(roleNames, ","),
				createdAt,
				u.Remark,
			})
		}
		return rows, nil
	}
	return exportsvc.Write(w, format, exportsvc.Options{Sheet: "用户列表", Headers: headers}, exportsvc.BatchSize(ctx), next)
}

// CreateInput 创建输入
type CreateInput struct {
	Username   string
	Password   string
	RealName   string
	Role       string
	Remark     string
	RoleIds    []uint64 // 功能角色(RBAC), 可为空
	MaxDevices *int     // 最大同时在线设备数, 仅超管可设置; nil/0 表示使用全局默认
}

// Create 创建管理员或用户
// 超管: pid=0; 管理员(admin): pid=自己(创建下级管理员/用户); 普通用户: 禁止
func (s *Service) Create(ctx context.Context, in CreateInput) (id uint64, err error) {
	actor, err := authz.Manager(ctx, "无权限创建用户")
	if err != nil {
		return 0, err
	}
	if in.Role != consts.RoleAdmin && in.Role != consts.RoleUser {
		return 0, gerror.NewCode(consts.CodeBadRequest, "角色必须是 admin 或 user")
	}
	// 设备数上限仅超管可设置, 管理员创建用户时不能携带
	if in.MaxDevices != nil {
		if !actor.IsSuperAdmin() {
			return 0, gerror.NewCode(consts.CodeForbidden, "仅超级管理员可设置最大在线设备数")
		}
		if *in.MaxDevices < 0 {
			return 0, gerror.NewCode(consts.CodeBadRequest, "最大在线设备数不能为负数")
		}
	}
	// 用户名唯一性(未删除范围内)
	count, err := dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Username, in.Username).Count()
	if err != nil {
		return 0, gerror.Wrap(err, "校验用户名失败")
	}
	if count > 0 {
		return 0, gerror.NewCode(consts.CodeBadRequest, "用户名已存在")
	}
	// 密码策略统一在 service 层校验, 不依赖各 API 的 v 标签
	if err = secsvc.New().ValidatePassword(ctx, in.Password); err != nil {
		return 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, gerror.Wrap(err, "密码加密失败")
	}
	realName := in.RealName
	if realName == "" {
		realName = in.Username
	}
	pid := uint64(0)
	if !actor.IsSuperAdmin() {
		pid = actor.Id
	}
	data := do.SysUser{
		Pid:                pid,
		Username:           in.Username,
		Password:           string(hash),
		RealName:           realName,
		Role:               in.Role,
		Status:             1,
		Remark:             in.Remark,
		MustChangePassword: 1, // 新建用户首次登录强制改密
	}
	if in.MaxDevices != nil && *in.MaxDevices > 0 {
		data.MaxDevices = *in.MaxDevices
	}
	res, err := dao.SysUser.Ctx(ctx).Data(data).InsertAndGetId()
	if err != nil {
		return 0, gerror.Wrap(err, "创建用户失败")
	}
	userId := gconv.Uint64(res)
	// 分配功能角色(RBAC): 超管任意启用角色; 管理员限自己可分配的角色
	if len(in.RoleIds) > 0 {
		if err = (rolesvc.New()).SetUserRoles(ctx, userId, in.RoleIds); err != nil {
			return 0, err
		}
	}
	return userId, nil
}

// UpdateInput 更新输入(字段为空则不修改)
type UpdateInput struct {
	Id       uint64
	Username string
	RealName string
	Avatar   string
	Password string
	Status   *int
	Role     string
	Remark   string
	RoleIds  *[]uint64 // 功能角色(RBAC), 非 nil 时全量覆盖; nil 表示不修改
	// MaxDevices 最大同时在线设备数: nil=不修改, 0=清空(使用全局默认), >0=设置。仅超管可携带
	MaxDevices *int
}

// Update 更新用户
func (s *Service) Update(ctx context.Context, in UpdateInput) (err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return err
	}
	target, err := s.getById(ctx, in.Id)
	if err != nil {
		return err
	}
	if err = s.checkScope(ctx, actor, target); err != nil {
		return err
	}
	// 超级管理员账号保护
	if target.Role == consts.RoleSuperAdmin && target.Id != actor.Id {
		return gerror.NewCode(consts.CodeForbidden, "无权修改超级管理员")
	}
	// 设备数上限仅超管可改: 在任何写库动作前拦截, 防止管理员借更新接口顺带越权修改
	if in.MaxDevices != nil {
		if !actor.IsSuperAdmin() {
			return gerror.NewCode(consts.CodeForbidden, "仅超级管理员可修改最大在线设备数")
		}
		if *in.MaxDevices < 0 {
			return gerror.NewCode(consts.CodeBadRequest, "最大在线设备数不能为负数")
		}
	}
	cols := dao.SysUser.Columns()
	data := do.SysUser{}
	if in.MaxDevices != nil && *in.MaxDevices > 0 {
		data.MaxDevices = *in.MaxDevices
	}
	if in.Username != "" && in.Username != target.Username {
		// 用户名唯一性校验(排除自身)
		count, err := dao.SysUser.Ctx(ctx).Where(cols.Username, in.Username).WhereNot(cols.Id, in.Id).Count()
		if err != nil {
			return gerror.Wrap(err, "校验用户名失败")
		}
		if count > 0 {
			return gerror.NewCode(consts.CodeBadRequest, "用户名已存在")
		}
		data.Username = in.Username
	}
	if in.RealName != "" {
		data.RealName = in.RealName
	}
	if in.Avatar != "" {
		data.Avatar = in.Avatar
	}
	if in.Remark != "" {
		data.Remark = in.Remark
	}
	if in.Password != "" {
		if err = secsvc.New().ValidatePassword(ctx, in.Password); err != nil {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return gerror.Wrap(err, "密码加密失败")
		}
		data.Password = string(hash)
		// 管理员重置密码后, 该用户下次登录需重新改密
		data.MustChangePassword = 1
	}
	if in.Status != nil {
		if target.Role == consts.RoleSuperAdmin && *in.Status != 1 {
			return gerror.NewCode(consts.CodeBadRequest, "不能禁用超级管理员")
		}
		data.Status = *in.Status
	}
	if in.Role != "" && in.Role != target.Role {
		if !actor.IsSuperAdmin() {
			return gerror.NewCode(consts.CodeForbidden, "仅超级管理员可修改角色")
		}
		if target.Role == consts.RoleSuperAdmin {
			return gerror.NewCode(consts.CodeBadRequest, "不能修改超级管理员角色")
		}
		if in.Role != consts.RoleAdmin && in.Role != consts.RoleUser {
			return gerror.NewCode(consts.CodeBadRequest, "角色必须是 admin 或 user")
		}
		data.Role = in.Role
	}
	if gconv.String(data) == "{}" && in.RoleIds == nil && in.MaxDevices == nil {
		return nil
	}
	if gconv.String(data) != "{}" {
		if _, err = dao.SysUser.Ctx(ctx).Where(cols.Id, in.Id).Data(data).Update(); err != nil {
			return gerror.Wrap(err, "更新用户失败")
		}
	}
	// 清空设备数上限(恢复使用全局默认)
	if in.MaxDevices != nil && *in.MaxDevices == 0 {
		if _, err = dao.SysUser.Ctx(ctx).Where(cols.Id, in.Id).Data(g.Map{cols.MaxDevices: gdb.Raw("NULL")}).Update(); err != nil {
			return gerror.Wrap(err, "更新最大在线设备数失败")
		}
	}
	// 设备数上限变更后立即生效: 调小时清理多余会话, 避免只是"不再新增"
	if in.MaxDevices != nil {
		s.applyDeviceLimit(ctx, target.Id, *in.MaxDevices)
	}
	// 功能角色(RBAC): 非 nil 时全量覆盖
	if in.RoleIds != nil {
		// 仅超管/管理员可分配角色; 目标不能是超级管理员(默认拥有全部菜单)
		if !actor.IsSuperAdmin() && actor.Role != consts.RoleAdmin {
			return gerror.NewCode(consts.CodeForbidden, "无权限分配角色")
		}
		if target.Role == consts.RoleSuperAdmin {
			return gerror.NewCode(consts.CodeBadRequest, "超级管理员默认拥有全部菜单, 无需分配角色")
		}
		if err = (rolesvc.New()).SetUserRoles(ctx, target.Id, *in.RoleIds); err != nil {
			return err
		}
	}
	return nil
}

// applyDeviceLimit 使设备数上限立即生效: 写入生效值并清理超出上限的会话(保留最近登录的)
func (s *Service) applyDeviceLimit(ctx context.Context, userId uint64, userMax int) {
	tsvc := tokensvc.New()
	if !tsvc.LimitEnabled(ctx) {
		return
	}
	limit := tsvc.EffectiveLimit(ctx, userMax)
	if err := tsvc.SetLimit(ctx, userId, limit); err != nil {
		g.Log().Warningf(ctx, "写入设备数上限失败: %v", err)
	}
	reason := tsvc.LimitExceededMessage(ctx)
	evicted, err := tsvc.Rebalance(ctx, userId, limit, reason)
	if err != nil {
		g.Log().Warningf(ctx, "清理超出上限的会话失败: %v", err)
		return
	}
	for _, e := range evicted {
		onlinesvc.MarkOffline(ctx, e.SessionId, reason)
	}
}

// Delete 删除用户(软删除), 删除管理员时级联删除其子孙
func (s *Service) Delete(ctx context.Context, id uint64) (err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return err
	}
	if id == actor.Id {
		return gerror.NewCode(consts.CodeBadRequest, "不能删除自己的账号")
	}
	target, err := s.getById(ctx, id)
	if err != nil {
		return err
	}
	if target.Role == consts.RoleSuperAdmin {
		return gerror.NewCode(consts.CodeForbidden, "超级管理员不允许删除")
	}
	if err = s.checkScope(ctx, actor, target); err != nil {
		return err
	}
	cols := dao.SysUser.Columns()
	// 级联收集子孙后统一软删除
	ids, err := s.descendantIds(ctx, target.Id)
	if err != nil {
		return err
	}
	ids = append([]uint64{target.Id}, ids...)
	// 显式 Update 实现软删, 同时记录删除操作人(Delete() 无法附带 Data)
	if _, err = dao.SysUser.Ctx(ctx).Unscoped().WhereIn(cols.Id, ids).Data(do.SysUser{
		DeletedAt: gtime.Now(),
		DeletedBy: actor.Id,
	}).Update(); err != nil {
		return gerror.Wrap(err, "删除用户失败")
	}
	return nil
}

// getById 按ID查询用户
func (s *Service) getById(ctx context.Context, id uint64) (*entity.SysUser, error) {
	var target entity.SysUser
	if err := dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Id, id).Scan(&target); err != nil {
		// 记录不存在时 Scan 返回 ErrNoRows, 按零值走下方 404 分支
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询用户失败")
		}
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "用户不存在")
	}
	return &target, nil
}

// getDeletedById 按ID查询回收站中的用户(已软删)
func (s *Service) getDeletedById(ctx context.Context, id uint64) (*entity.SysUser, error) {
	var target entity.SysUser
	cols := dao.SysUser.Columns()
	if err := dao.SysUser.Ctx(ctx).Unscoped().Where(cols.Id, id).WhereNotNull(cols.DeletedAt).Scan(&target); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询回收站失败")
		}
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "回收站中不存在该用户")
	}
	return &target, nil
}

// Recycle 回收站列表(已软删用户)
// 数据权限: 超管全部; 管理员(admin)仅其子孙树(含已删中间节点)中的已删记录
// 返回 operators: deletedBy -> 操作人用户名
func (s *Service) Recycle(ctx context.Context, in ListInput) (items []*entity.SysUser, operators map[uint64]string, total int, err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	cols := dao.SysUser.Columns()
	m := dao.SysUser.Ctx(ctx).Unscoped().WhereNotNull(cols.DeletedAt)
	if !actor.IsSuperAdmin() {
		if actor.Role != consts.RoleAdmin {
			return nil, nil, 0, gerror.NewCode(consts.CodeForbidden, "无权限查看回收站")
		}
		ids, err := s.descendantIdsAll(ctx, actor.Id)
		if err != nil {
			return nil, nil, 0, err
		}
		if len(ids) == 0 {
			return []*entity.SysUser{}, map[uint64]string{}, 0, nil
		}
		m = m.WhereIn(cols.Id, ids)
	}
	if in.Username != "" {
		m = m.WhereLike(cols.Username, "%"+in.Username+"%")
	}
	if total, err = m.Count(); err != nil {
		return nil, nil, 0, gerror.Wrap(err, "统计回收站数量失败")
	}
	if err = m.Page(in.Page, in.PageSize).OrderDesc(cols.DeletedAt).Scan(&items); err != nil {
		return nil, nil, 0, gerror.Wrap(err, "查询回收站失败")
	}
	// 批量查询删除操作人
	operators = make(map[uint64]string)
	delBySet := make(map[uint64]struct{})
	for _, u := range items {
		if u.DeletedBy > 0 {
			delBySet[u.DeletedBy] = struct{}{}
		}
	}
	if len(delBySet) > 0 {
		delBys := make([]uint64, 0, len(delBySet))
		for id := range delBySet {
			delBys = append(delBys, id)
		}
		var operatorsList []*entity.SysUser
		if err = dao.SysUser.Ctx(ctx).Unscoped().WhereIn(cols.Id, delBys).Fields(cols.Id, cols.Username).Scan(&operatorsList); err != nil {
			return nil, nil, 0, gerror.Wrap(err, "查询操作人失败")
		}
		for _, o := range operatorsList {
			operators[o.Id] = o.Username
		}
	}
	return items, operators, total, nil
}

// Restore 恢复回收站用户
func (s *Service) Restore(ctx context.Context, id uint64) (err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return err
	}
	target, err := s.getDeletedById(ctx, id)
	if err != nil {
		return err
	}
	if err = s.checkScopeDeleted(ctx, actor, target); err != nil {
		return err
	}
	cols := dao.SysUser.Columns()
	// 上级必须处于活跃状态, 否则恢复后脱离数据权限树
	if target.Pid > 0 {
		var parent entity.SysUser
		if err = dao.SysUser.Ctx(ctx).Unscoped().Where(cols.Id, target.Pid).Scan(&parent); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return gerror.Wrap(err, "查询上级用户失败")
		}
		if parent.Id == 0 || parent.DeletedAt != nil {
			return gerror.NewCode(consts.CodeBadRequest, "其上级账号仍在回收站, 请先恢复上级")
		}
	}
	// 用户名不能与活跃用户冲突(软删同名可共存)
	count, err := dao.SysUser.Ctx(ctx).Where(cols.Username, target.Username).Count()
	if err != nil {
		return gerror.Wrap(err, "校验用户名失败")
	}
	if count > 0 {
		return gerror.NewCode(consts.CodeBadRequest, "用户名已被其他账号使用, 无法恢复")
	}
	if _, err = dao.SysUser.Ctx(ctx).Unscoped().Where(cols.Id, id).Data(g.Map{cols.DeletedAt: gdb.Raw("NULL")}).Update(); err != nil {
		return gerror.Wrap(err, "恢复用户失败")
	}
	return nil
}

// Purge 彻底删除(物理删除, 不可恢复, 级联清除其已删子孙)
func (s *Service) Purge(ctx context.Context, id uint64) (err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return err
	}
	target, err := s.getDeletedById(ctx, id)
	if err != nil {
		return err
	}
	if target.Role == consts.RoleSuperAdmin {
		return gerror.NewCode(consts.CodeForbidden, "超级管理员不允许彻底删除")
	}
	if err = s.checkScopeDeleted(ctx, actor, target); err != nil {
		return err
	}
	cols := dao.SysUser.Columns()
	// 存在活跃下级时不允许物理删除, 避免孤儿节点
	activeCount, err := dao.SysUser.Ctx(ctx).Where(cols.Pid, target.Id).Count()
	if err != nil {
		return gerror.Wrap(err, "检查下级用户失败")
	}
	if activeCount > 0 {
		return gerror.NewCode(consts.CodeBadRequest, "该账号存在未删除的下级用户, 无法彻底删除")
	}
	ids, err := s.deletedDescendantIds(ctx, target.Id)
	if err != nil {
		return err
	}
	ids = append([]uint64{target.Id}, ids...)
	if _, err = dao.SysUser.Ctx(ctx).Unscoped().WhereIn(cols.Id, ids).Delete(); err != nil {
		return gerror.Wrap(err, "彻底删除用户失败")
	}
	// 同步清理这些用户的菜单授权记录, 避免产生孤儿数据
	if _, err = dao.SysUserMenu.Ctx(ctx).WhereIn(dao.SysUserMenu.Columns().UserId, ids).Delete(); err != nil {
		return gerror.Wrap(err, "清理菜单授权失败")
	}
	// 同步清理这些用户的功能角色关联
	if _, err = dao.SysUserRole.Ctx(ctx).WhereIn(dao.SysUserRole.Columns().UserId, ids).Delete(); err != nil {
		return gerror.Wrap(err, "清理用户角色关联失败")
	}
	return nil
}

// checkScope 数据权限校验: 超管全部; 本人可操作自己; 管理员(admin)可操作子孙
func (s *Service) checkScope(ctx context.Context, actor *identity.Identity, target *entity.SysUser) (err error) {
	if actor.IsSuperAdmin() {
		return nil
	}
	if target.Id == actor.Id {
		return nil
	}
	if actor.Role != consts.RoleAdmin {
		return gerror.NewCode(consts.CodeForbidden, "无权限操作该用户")
	}
	ids, err := s.descendantIds(ctx, actor.Id)
	if err != nil {
		return err
	}
	for _, v := range ids {
		if v == target.Id {
			return nil
		}
	}
	return gerror.NewCode(consts.CodeForbidden, "无权限操作该用户")
}

// checkScopeDeleted 回收站数据权限校验: 超管全部; 管理员(admin)可操作其子孙树中的已删记录
func (s *Service) checkScopeDeleted(ctx context.Context, actor *identity.Identity, target *entity.SysUser) (err error) {
	if actor.IsSuperAdmin() {
		return nil
	}
	if actor.Role != consts.RoleAdmin {
		return gerror.NewCode(consts.CodeForbidden, "无权限操作该用户")
	}
	ids, err := s.descendantIdsAll(ctx, actor.Id)
	if err != nil {
		return err
	}
	for _, v := range ids {
		if v == target.Id {
			return nil
		}
	}
	return gerror.NewCode(consts.CodeForbidden, "无权限操作该用户")
}

// descendantIds 获取 root 的所有子孙ID(不含自身)
func (s *Service) descendantIds(ctx context.Context, rootId uint64) (result []uint64, err error) {
	cols := dao.SysUser.Columns()
	level := []uint64{rootId}
	// 层级防御上限, 防止脏数据成环导致死循环
	for i := 0; i < 50 && len(level) > 0; i++ {
		var records []*entity.SysUser
		if err = dao.SysUser.Ctx(ctx).WhereIn(cols.Pid, level).Fields(cols.Id).Scan(&records); err != nil {
			return nil, gerror.Wrap(err, "查询下级用户失败")
		}
		if len(records) == 0 {
			break
		}
		level = make([]uint64, 0, len(records))
		for _, r := range records {
			level = append(level, r.Id)
		}
		result = append(result, level...)
	}
	return result, nil
}

// descendantIdsAll 获取 root 的所有子孙ID(含已删节点及其后代, 不含自身), 用于回收站数据范围
func (s *Service) descendantIdsAll(ctx context.Context, rootId uint64) (result []uint64, err error) {
	cols := dao.SysUser.Columns()
	level := []uint64{rootId}
	// 层级防御上限, 防止脏数据成环导致死循环
	for i := 0; i < 50 && len(level) > 0; i++ {
		var records []*entity.SysUser
		if err = dao.SysUser.Ctx(ctx).Unscoped().WhereIn(cols.Pid, level).Fields(cols.Id).Scan(&records); err != nil {
			return nil, gerror.Wrap(err, "查询下级用户失败")
		}
		if len(records) == 0 {
			break
		}
		level = make([]uint64, 0, len(records))
		for _, r := range records {
			level = append(level, r.Id)
		}
		result = append(result, level...)
	}
	return result, nil
}

// deletedDescendantIds 获取 root 的所有已删子孙ID(仅 deleted_at 非空的节点), 用于级联物理删除
func (s *Service) deletedDescendantIds(ctx context.Context, rootId uint64) (result []uint64, err error) {
	cols := dao.SysUser.Columns()
	level := []uint64{rootId}
	// 层级防御上限, 防止脏数据成环导致死循环
	for i := 0; i < 50 && len(level) > 0; i++ {
		var records []*entity.SysUser
		if err = dao.SysUser.Ctx(ctx).Unscoped().WhereIn(cols.Pid, level).Fields(cols.Id, cols.DeletedAt).Scan(&records); err != nil {
			return nil, gerror.Wrap(err, "查询下级用户失败")
		}
		if len(records) == 0 {
			break
		}
		level = make([]uint64, 0, len(records))
		for _, r := range records {
			if r.DeletedAt != nil {
				result = append(result, r.Id)
			}
			level = append(level, r.Id)
		}
	}
	return result, nil
}
