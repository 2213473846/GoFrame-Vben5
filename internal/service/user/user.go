package user

import (
	"context"
	"database/sql"
	"errors"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/identity"

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
func (s *Service) List(ctx context.Context, in ListInput) (users []*entity.SysUser, parents map[uint64]*entity.SysUser, total int, err error) {
	actor := identity.Get(ctx)
	if actor == nil {
		return nil, nil, 0, gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
	}
	cols := dao.SysUser.Columns()
	m := dao.SysUser.Ctx(ctx)
	if !actor.IsSuperAdmin() {
		if actor.Role != consts.RoleAdmin {
			return nil, nil, 0, gerror.NewCode(consts.CodeForbidden, "无权限查看用户列表")
		}
		ids, err := s.descendantIds(ctx, actor.Id)
		if err != nil {
			return nil, nil, 0, err
		}
		if len(ids) == 0 {
			return []*entity.SysUser{}, map[uint64]*entity.SysUser{}, 0, nil
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
	if total, err = m.Count(); err != nil {
		return nil, nil, 0, gerror.Wrap(err, "统计用户数量失败")
	}
	if err = m.Page(in.Page, in.PageSize).OrderDesc(cols.Id).Scan(&users); err != nil {
		return nil, nil, 0, gerror.Wrap(err, "查询用户列表失败")
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
			return nil, nil, 0, gerror.Wrap(err, "查询上级用户失败")
		}
		for _, p := range parentList {
			parents[p.Id] = p
		}
	}
	return users, parents, total, nil
}

// CreateInput 创建输入
type CreateInput struct {
	Username string
	Password string
	RealName string
	Role     string
	Remark   string
}

// Create 创建管理员或用户
// 超管: pid=0; 管理员(admin): pid=自己(创建下级管理员/用户); 普通用户: 禁止
func (s *Service) Create(ctx context.Context, in CreateInput) (id uint64, err error) {
	actor := identity.Get(ctx)
	if actor == nil {
		return 0, gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
	}
	if actor.Role != consts.RoleAdmin && !actor.IsSuperAdmin() {
		return 0, gerror.NewCode(consts.CodeForbidden, "无权限创建用户")
	}
	if in.Role != consts.RoleAdmin && in.Role != consts.RoleUser {
		return 0, gerror.NewCode(consts.CodeBadRequest, "角色必须是 admin 或 user")
	}
	// 用户名唯一性(未删除范围内)
	count, err := dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Username, in.Username).Count()
	if err != nil {
		return 0, gerror.Wrap(err, "校验用户名失败")
	}
	if count > 0 {
		return 0, gerror.NewCode(consts.CodeBadRequest, "用户名已存在")
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
	res, err := dao.SysUser.Ctx(ctx).Data(do.SysUser{
		Pid:      pid,
		Username: in.Username,
		Password: string(hash),
		RealName: realName,
		Role:     in.Role,
		Status:   1,
		Remark:   in.Remark,
	}).InsertAndGetId()
	if err != nil {
		return 0, gerror.Wrap(err, "创建用户失败")
	}
	return gconv.Uint64(res), nil
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
}

// Update 更新用户
func (s *Service) Update(ctx context.Context, in UpdateInput) (err error) {
	actor := identity.Get(ctx)
	if actor == nil {
		return gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
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
	cols := dao.SysUser.Columns()
	data := do.SysUser{}
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
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return gerror.Wrap(err, "密码加密失败")
		}
		data.Password = string(hash)
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
	if gconv.String(data) == "{}" {
		return nil
	}
	if _, err = dao.SysUser.Ctx(ctx).Where(cols.Id, in.Id).Data(data).Update(); err != nil {
		return gerror.Wrap(err, "更新用户失败")
	}
	return nil
}

// Delete 删除用户(软删除), 删除管理员时级联删除其子孙
func (s *Service) Delete(ctx context.Context, id uint64) (err error) {
	actor := identity.Get(ctx)
	if actor == nil {
		return gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
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
	actor := identity.Get(ctx)
	if actor == nil {
		return nil, nil, 0, gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
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
	actor := identity.Get(ctx)
	if actor == nil {
		return gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
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
	actor := identity.Get(ctx)
	if actor == nil {
		return gerror.NewCode(consts.CodeUnauthorized, "未登录或登录已过期")
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
