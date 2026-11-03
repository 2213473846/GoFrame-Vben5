// Package role 功能角色(RBAC)服务
// sys_role(角色) / sys_user_role(用户-角色) / sys_role_menu(角色-菜单授权)
// 与 sys_user.role 数据权限分级(super_admin/admin/user)解耦:
// role 字段决定"能看到哪些用户/日志", 功能角色决定"能用哪些菜单"。
package role

import (
	"context"
	"database/sql"
	"errors"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/authz"
	"demo/internal/service/identity"
	menusvc "demo/internal/service/menu"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

// Service 功能角色服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// CreateInput 新增角色输入
type CreateInput struct {
	Code   string
	Name   string
	Remark string
	Status int
}

// UpdateInput 修改角色输入
type UpdateInput struct {
	Id uint64
	CreateInput
}

// List 角色列表
// 超管: 全部角色; 管理员: 仅自己"可分配"的角色(菜单范围不超管自己的可授权菜单, 只读);
// 普通用户: 无权限
func (s *Service) List(ctx context.Context) (roles []*entity.SysRole, err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, err
	}
	cols := dao.SysRole.Columns()
	if err = dao.SysRole.Ctx(ctx).OrderAsc(cols.Id).Scan(&roles); err != nil {
		return nil, gerror.Wrap(err, "查询角色失败")
	}
	if !actor.IsSuperAdmin() {
		if actor.Role != consts.RoleAdmin {
			return nil, gerror.NewCode(consts.CodeForbidden, "无权限查看角色")
		}
		return s.filterAssignable(ctx, actor, roles)
	}
	return roles, nil
}

// Create 新增角色, 仅超级管理员
func (s *Service) Create(ctx context.Context, in CreateInput) (id uint64, err error) {
	actor, err := authz.SuperAdmin(ctx, "仅超级管理员可管理角色")
	if err != nil {
		return 0, err
	}
	if err = s.validateInput(ctx, in, 0); err != nil {
		return 0, err
	}
	lastId, err := dao.SysRole.Ctx(ctx).Data(do.SysRole{
		Code:      in.Code,
		Name:      in.Name,
		Remark:    in.Remark,
		Status:    in.Status,
		CreatedBy: actor.Id,
	}).InsertAndGetId()
	if err != nil {
		return 0, gerror.Wrap(err, "新增角色失败")
	}
	return gconv.Uint64(lastId), nil
}

// Update 修改角色, 仅超级管理员
func (s *Service) Update(ctx context.Context, in UpdateInput) (err error) {
	if err = s.requireSuperAdmin(ctx); err != nil {
		return err
	}
	target, err := s.getById(ctx, in.Id)
	if err != nil {
		return err
	}
	if err = s.validateInput(ctx, in.CreateInput, target.Id); err != nil {
		return err
	}
	data := do.SysRole{
		Code:   in.Code,
		Name:   in.Name,
		Remark: in.Remark,
		Status: in.Status,
	}
	if _, err = dao.SysRole.Ctx(ctx).Where(dao.SysRole.Columns().Id, target.Id).Data(data).Update(); err != nil {
		return gerror.Wrap(err, "修改角色失败")
	}
	return nil
}

// Delete 删除角色(软删除), 仅超级管理员; 事务内级联清理角色菜单授权与用户关联
func (s *Service) Delete(ctx context.Context, id uint64) (err error) {
	if err = s.requireSuperAdmin(ctx); err != nil {
		return err
	}
	target, err := s.getById(ctx, id)
	if err != nil {
		return err
	}
	if err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.SysRole.Ctx(ctx).Where(dao.SysRole.Columns().Id, target.Id).Delete(); err != nil {
			return err
		}
		if _, err := dao.SysRoleMenu.Ctx(ctx).Where(dao.SysRoleMenu.Columns().RoleId, target.Id).Delete(); err != nil {
			return err
		}
		if _, err := dao.SysUserRole.Ctx(ctx).Where(dao.SysUserRole.Columns().RoleId, target.Id).Delete(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return gerror.Wrap(err, "删除角色失败")
	}
	return nil
}

// GrantInfo 查询角色的可授权菜单树与已授权菜单
// 超管: 可授权全部启用菜单; 管理员: 仅超管下发给自己的菜单
func (s *Service) GrantInfo(ctx context.Context, roleId uint64) (target *entity.SysRole, nodes []*menusvc.MenuNode, checked []uint64, err error) {
	if _, err = authz.Manager(ctx, "无权限分配菜单"); err != nil {
		return nil, nil, nil, err
	}
	target, err = s.getById(ctx, roleId)
	if err != nil {
		return nil, nil, nil, err
	}
	grantable, err := menusvc.New().GrantableMenus(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	checked, err = s.roleMenuIds(ctx, target.Id)
	if err != nil {
		return nil, nil, nil, err
	}
	return target, menusvc.New().BuildMenuTree(grantable, 0), checked, nil
}

// Grant 保存角色菜单授权(全量覆盖), 并自动补齐祖先菜单
// 超管: 任意启用菜单; 管理员: 与用户授权规则一致——仅可下发自己可授权的菜单,
// 且保留超管直授、自己无权管理的菜单(角色含范围外菜单时不整体拒绝, 只保护不可改部分)
func (s *Service) Grant(ctx context.Context, roleId uint64, menuIds []uint64) (err error) {
	actor, err := authz.Manager(ctx, "无权限分配菜单")
	if err != nil {
		return err
	}
	target, err := s.getById(ctx, roleId)
	if err != nil {
		return err
	}
	menuSvc := menusvc.New()
	grantable, err := menuSvc.GrantableMenus(ctx)
	if err != nil {
		return err
	}
	grantableSet := make(map[uint64]bool, len(grantable))
	menuMap := make(map[uint64]*entity.SysMenu, len(grantable))
	for _, m := range grantable {
		grantableSet[m.Id] = true
		menuMap[m.Id] = m
	}
	// 去重 + 补齐祖先 + 校验可授权范围
	finalSet := make(map[uint64]bool)
	for _, id := range menuIds {
		if id == 0 || finalSet[id] {
			continue
		}
		if !grantableSet[id] {
			return gerror.NewCode(consts.CodeForbidden, "无权下发该菜单")
		}
		for cur := id; cur > 0; {
			finalSet[cur] = true
			parent, ok := menuMap[cur]
			if !ok {
				return gerror.NewCode(consts.CodeBadRequest, "菜单数据异常, 无法下发")
			}
			if parent.Pid == 0 || finalSet[parent.Pid] {
				break
			}
			if !grantableSet[parent.Pid] {
				return gerror.NewCode(consts.CodeForbidden, "无权下发该菜单")
			}
			cur = parent.Pid
		}
	}
	ids := make([]uint64, 0, len(finalSet))
	for id := range finalSet {
		ids = append(ids, id)
	}
	// 管理员保存授权时, 保留超管直接下发但当前管理员无权管理的菜单
	if !actor.IsSuperAdmin() {
		existing, err := s.roleMenuIds(ctx, target.Id)
		if err != nil {
			return err
		}
		for _, id := range existing {
			if !grantableSet[id] {
				ids = append(ids, id)
			}
		}
	}
	if err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		cols := dao.SysRoleMenu.Columns()
		if _, err := dao.SysRoleMenu.Ctx(ctx).Where(cols.RoleId, target.Id).Delete(); err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		records := make([]do.SysRoleMenu, 0, len(ids))
		for _, id := range ids {
			records = append(records, do.SysRoleMenu{RoleId: target.Id, MenuId: id})
		}
		if _, err := dao.SysRoleMenu.Ctx(ctx).Data(records).Insert(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return gerror.Wrap(err, "保存角色菜单授权失败")
	}
	return nil
}

// SetUserRoles 全量覆盖用户的角色关联(事务)
// 超管: 任意存在且启用的角色;
// 管理员: 只能增删自己"可分配"的角色, 目标用户已有的、超管分配的不可分配角色会被强制保留
// (与菜单授权的"保留超管直授"规则一致, 避免越权移除或静默丢失权限)
func (s *Service) SetUserRoles(ctx context.Context, userId uint64, roleIds []uint64) (err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return err
	}
	// 去重
	reqSet := make(map[uint64]bool, len(roleIds))
	req := make([]uint64, 0, len(roleIds))
	for _, id := range roleIds {
		if id == 0 || reqSet[id] {
			continue
		}
		reqSet[id] = true
		req = append(req, id)
	}
	roleCols := dao.SysRole.Columns()
	var final []uint64
	if actor.IsSuperAdmin() {
		if len(req) > 0 {
			var roles []*entity.SysRole
			if err = dao.SysRole.Ctx(ctx).WhereIn(roleCols.Id, req).Where(roleCols.Status, 1).Scan(&roles); err != nil {
				return gerror.Wrap(err, "查询角色失败")
			}
			if len(roles) != len(req) {
				return gerror.NewCode(consts.CodeBadRequest, "存在无效或已停用的角色")
			}
		}
		final = req
	} else {
		if actor.Role != consts.RoleAdmin {
			return gerror.NewCode(consts.CodeForbidden, "无权限分配角色")
		}
		// 目标用户当前已有角色(含停用角色的关联)
		existing, err := s.userRoleIdsOf(ctx, userId)
		if err != nil {
			return err
		}
		existingSet := make(map[uint64]bool, len(existing))
		for _, id := range existing {
			existingSet[id] = true
		}
		// 管理员可分配的角色
		var allRoles []*entity.SysRole
		if err = dao.SysRole.Ctx(ctx).Scan(&allRoles); err != nil {
			return gerror.Wrap(err, "查询角色失败")
		}
		assignable, err := s.filterAssignable(ctx, actor, allRoles)
		if err != nil {
			return err
		}
		assignableSet := make(map[uint64]bool, len(assignable))
		for _, r := range assignable {
			assignableSet[r.Id] = true
		}
		final = make([]uint64, 0, len(req)+len(existing))
		for _, id := range req {
			if assignableSet[id] {
				final = append(final, id)
			} else if !existingSet[id] {
				return gerror.NewCode(consts.CodeForbidden, "存在无权分配的角色")
			}
		}
		// 强制保留超管分配的、管理员不可分配的角色
		for _, id := range existing {
			if !assignableSet[id] {
				final = append(final, id)
			}
		}
	}
	return g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		cols := dao.SysUserRole.Columns()
		if _, err := dao.SysUserRole.Ctx(ctx).Where(cols.UserId, userId).Delete(); err != nil {
			return err
		}
		if len(final) == 0 {
			return nil
		}
		records := make([]do.SysUserRole, 0, len(final))
		for _, id := range final {
			records = append(records, do.SysUserRole{UserId: userId, RoleId: id})
		}
		if _, err := dao.SysUserRole.Ctx(ctx).Data(records).Insert(); err != nil {
			return err
		}
		return nil
	})
}

// userRoleIdsOf 查询用户关联的角色ID(原始关联, 含停用角色)
func (s *Service) userRoleIdsOf(ctx context.Context, userId uint64) (ids []uint64, err error) {
	var records []*entity.SysUserRole
	cols := dao.SysUserRole.Columns()
	if err = dao.SysUserRole.Ctx(ctx).Where(cols.UserId, userId).Fields(cols.RoleId).Scan(&records); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, gerror.Wrap(err, "查询用户角色失败")
	}
	ids = make([]uint64, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.RoleId)
	}
	return ids, nil
}

// UserRoles 批量查询用户的功能角色(userId -> 角色列表)
func (s *Service) UserRoles(ctx context.Context, userIds []uint64) (result map[uint64][]*entity.SysRole, err error) {
	result = make(map[uint64][]*entity.SysRole, len(userIds))
	if len(userIds) == 0 {
		return result, nil
	}
	userRoleCols := dao.SysUserRole.Columns()
	roleCols := dao.SysRole.Columns()
	var records []*entity.SysUserRole
	if err = dao.SysUserRole.Ctx(ctx).WhereIn(userRoleCols.UserId, userIds).Scan(&records); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, nil
		}
		return nil, gerror.Wrap(err, "查询用户角色失败")
	}
	roleIds := make([]uint64, 0, len(records))
	for _, r := range records {
		roleIds = append(roleIds, r.RoleId)
	}
	var roles []*entity.SysRole
	if len(roleIds) > 0 {
		if err = dao.SysRole.Ctx(ctx).WhereIn(roleCols.Id, roleIds).Scan(&roles); err != nil {
			return nil, gerror.Wrap(err, "查询角色失败")
		}
	}
	roleMap := make(map[uint64]*entity.SysRole, len(roles))
	for _, r := range roles {
		roleMap[r.Id] = r
	}
	for _, r := range records {
		if role, ok := roleMap[r.RoleId]; ok {
			result[r.UserId] = append(result[r.UserId], role)
		}
	}
	return result, nil
}

// requireSuperAdmin 校验超管身份
func (s *Service) requireSuperAdmin(ctx context.Context) error {
	_, err := authz.SuperAdmin(ctx, "仅超级管理员可管理角色")
	return err
}

// validateInput 校验角色输入
func (s *Service) validateInput(ctx context.Context, in CreateInput, selfId uint64) error {
	if in.Name == "" {
		return gerror.NewCode(consts.CodeBadRequest, "角色名称不能为空")
	}
	if in.Status != 0 && in.Status != 1 {
		return gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	// 角色编码唯一(未删除范围内)
	cols := dao.SysRole.Columns()
	m := dao.SysRole.Ctx(ctx).Where(cols.Code, in.Code)
	if selfId > 0 {
		m = m.WhereNot(cols.Id, selfId)
	}
	count, err := m.Count()
	if err != nil {
		return gerror.Wrap(err, "校验角色编码失败")
	}
	if count > 0 {
		return gerror.NewCode(consts.CodeBadRequest, "角色编码已存在")
	}
	return nil
}

// getById 查询角色
func (s *Service) getById(ctx context.Context, id uint64) (*entity.SysRole, error) {
	var target entity.SysRole
	if err := dao.SysRole.Ctx(ctx).Where(dao.SysRole.Columns().Id, id).Scan(&target); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询角色失败")
		}
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "角色不存在")
	}
	return &target, nil
}

// roleMenuIds 查询角色的已授权菜单ID
func (s *Service) roleMenuIds(ctx context.Context, roleId uint64) (ids []uint64, err error) {
	var records []*entity.SysRoleMenu
	cols := dao.SysRoleMenu.Columns()
	if err = dao.SysRoleMenu.Ctx(ctx).Where(cols.RoleId, roleId).Fields(cols.MenuId).Scan(&records); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, gerror.Wrap(err, "查询角色菜单授权失败")
	}
	ids = make([]uint64, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.MenuId)
	}
	return ids, nil
}

// filterAssignable 从候选角色中筛选出管理员可分配的角色:
// 角色启用且其全部菜单都在管理员的可授权范围内(避免越权扩大下级权限)
func (s *Service) filterAssignable(ctx context.Context, actor *identity.Identity, candidates []*entity.SysRole) ([]*entity.SysRole, error) {
	grantable, err := menusvc.New().GrantableMenus(ctx)
	if err != nil {
		return nil, err
	}
	grantableSet := make(map[uint64]bool, len(grantable))
	for _, m := range grantable {
		grantableSet[m.Id] = true
	}
	result := make([]*entity.SysRole, 0, len(candidates))
	for _, r := range candidates {
		if r.Status != 1 {
			continue
		}
		menuIds, err := s.roleMenuIds(ctx, r.Id)
		if err != nil {
			return nil, err
		}
		within := true
		for _, id := range menuIds {
			if !grantableSet[id] {
				within = false
				break
			}
		}
		if within {
			result = append(result, r)
		}
	}
	return result, nil
}
