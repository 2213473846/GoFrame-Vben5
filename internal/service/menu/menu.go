package menu

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/authz"
	"demo/internal/service/identity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

// 菜单类型
const (
	TypeCatalog = "catalog" // 目录
	TypeMenu    = "menu"    // 菜单
)

// ComponentLayout 目录节点使用布局组件
const ComponentLayout = "BasicLayout"

// Service 菜单/路由服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// RouteMeta 路由元信息
type RouteMeta struct {
	Title      string
	Icon       string
	Order      int
	HideInMenu bool
	KeepAlive  bool
	AffixTab   bool
}

// RouteNode 下发给前端的路由节点
type RouteNode struct {
	Name      string
	Path      string
	Component string
	Redirect  string
	Meta      RouteMeta
	Children  []*RouteNode
}

// MenuNode 菜单管理树节点
type MenuNode struct {
	Menu     *entity.SysMenu
	Children []*MenuNode
}

// CreateInput 新增菜单输入
type CreateInput struct {
	Pid        uint64
	Name       string
	Path       string
	Component  string
	Redirect   string
	Type       string
	Title      string
	Icon       string
	Sort       int
	Status     int
	HideInMenu bool
	KeepAlive  bool
	AffixTab   bool
	IsPublic   bool
	Remark     string
}

// UpdateInput 修改菜单输入
type UpdateInput struct {
	Id uint64
	CreateInput
}

// All 获取当前登录用户可见的路由树(后端下发)
// 超管: 全部启用菜单; 其他角色: 已授权菜单 + 公共菜单, 并自动补齐祖先节点
func (s *Service) All(ctx context.Context) (routes []*RouteNode, err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, err
	}
	cols := dao.SysMenu.Columns()
	var menus []*entity.SysMenu
	if err = dao.SysMenu.Ctx(ctx).Where(cols.Status, 1).OrderAsc(cols.Sort).OrderAsc(cols.Id).Scan(&menus); err != nil {
		return nil, gerror.Wrap(err, "查询菜单失败")
	}
	if !actor.IsSuperAdmin() {
		grantedIds, err := s.grantedMenuIds(ctx, actor.Id)
		if err != nil {
			return nil, err
		}
		allowed := make(map[uint64]bool, len(grantedIds))
		for _, id := range grantedIds {
			allowed[id] = true
		}
		var picked []*entity.SysMenu
		for _, m := range menus {
			if m.IsPublic == 1 || allowed[m.Id] {
				picked = append(picked, m)
			}
		}
		// 补齐祖先节点, 保证路由树完整
		menus = s.withAncestors(menus, picked)
	}
	return s.buildRoutes(menus, 0), nil
}

// FirstPath 返回当前用户可见的第一个菜单完整路径, 作为登录后的默认首页
// 没有任何可见菜单时回退到个人中心
func (s *Service) FirstPath(ctx context.Context) (path string, err error) {
	routes, err := s.All(ctx)
	if err != nil {
		return "", err
	}
	if p := firstVisiblePath(routes, ""); p != "" {
		return p, nil
	}
	return "/profile", nil
}

// firstVisiblePath 深度优先查找第一个可见叶子节点的完整路径
func firstVisiblePath(nodes []*RouteNode, parentPath string) string {
	for _, node := range nodes {
		if node == nil || node.Meta.HideInMenu {
			continue
		}
		fullPath := joinRoutePath(parentPath, node.Path)
		if len(node.Children) > 0 {
			// 优先使用目录下第一个可见叶子
			if p := firstVisiblePath(node.Children, fullPath); p != "" {
				return p
			}
			// 目录下无可见叶子时, 有重定向则使用重定向
			if node.Redirect != "" {
				return node.Redirect
			}
			continue
		}
		// 目录类型节点(组件为布局)不产生页面路径
		if fullPath == "" || fullPath == "/" || node.Component == ComponentLayout {
			continue
		}
		return fullPath
	}
	return ""
}

// joinRoutePath 拼接父子路由路径
func joinRoutePath(parentPath, path string) string {
	if path == "" {
		return parentPath
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	if parentPath == "" {
		return "/" + path
	}
	return strings.TrimSuffix(parentPath, "/") + "/" + path
}

// List 菜单管理列表(树形)
// 超级管理员: 全部菜单; 管理员: 仅查看自己可下发的菜单(只读, 不可增删改)
func (s *Service) List(ctx context.Context) (nodes []*MenuNode, err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, err
	}
	if !actor.IsSuperAdmin() {
		if actor.Role != consts.RoleAdmin {
			return nil, gerror.NewCode(consts.CodeForbidden, "无权限查看菜单")
		}
		var grantable []*entity.SysMenu
		if grantable, err = s.grantableMenus(ctx, actor); err != nil {
			return nil, err
		}
		return s.buildMenuTree(grantable, 0), nil
	}
	cols := dao.SysMenu.Columns()
	var menus []*entity.SysMenu
	if err = dao.SysMenu.Ctx(ctx).OrderAsc(cols.Sort).OrderAsc(cols.Id).Scan(&menus); err != nil {
		return nil, gerror.Wrap(err, "查询菜单失败")
	}
	return s.buildMenuTree(menus, 0), nil
}

// Create 新增菜单, 仅超级管理员
func (s *Service) Create(ctx context.Context, in CreateInput) (id uint64, err error) {
	actor, err := authz.SuperAdmin(ctx, "仅超级管理员可新增路由")
	if err != nil {
		return 0, err
	}
	if err = s.validateInput(ctx, in, 0); err != nil {
		return 0, err
	}
	data := s.toDo(in)
	if data.Component == "" && in.Type == TypeCatalog {
		data.Component = ComponentLayout
	}
	data.CreatedBy = actor.Id
	lastId, err := dao.SysMenu.Ctx(ctx).Data(data).InsertAndGetId()
	if err != nil {
		return 0, gerror.Wrap(err, "新增菜单失败")
	}
	return gconv.Uint64(lastId), nil
}

// Update 修改菜单, 仅超级管理员
func (s *Service) Update(ctx context.Context, in UpdateInput) (err error) {
	if err = s.requireSuperAdmin(ctx); err != nil {
		return err
	}
	target, err := s.getById(ctx, in.Id)
	if err != nil {
		return err
	}
	if err = s.validateInput(ctx, in.CreateInput, in.Id); err != nil {
		return err
	}
	// 上级不能是自己或自己的后代(防止成环)
	if in.Pid > 0 {
		if in.Pid == in.Id {
			return gerror.NewCode(consts.CodeBadRequest, "上级菜单不能是自己")
		}
		descendants, err := s.descendantMenuIds(ctx, in.Id)
		if err != nil {
			return err
		}
		for _, v := range descendants {
			if v == in.Pid {
				return gerror.NewCode(consts.CodeBadRequest, "上级菜单不能是自己的子菜单")
			}
		}
	}
	data := s.toDo(in.CreateInput)
	if data.Component == "" && in.Type == TypeCatalog {
		data.Component = ComponentLayout
	}
	if _, err = dao.SysMenu.Ctx(ctx).Where(dao.SysMenu.Columns().Id, target.Id).Data(data).Update(); err != nil {
		return gerror.Wrap(err, "修改菜单失败")
	}
	return nil
}

// Delete 删除菜单(软删除), 仅超级管理员; 同时清理授权关系
func (s *Service) Delete(ctx context.Context, id uint64) (err error) {
	if err = s.requireSuperAdmin(ctx); err != nil {
		return err
	}
	target, err := s.getById(ctx, id)
	if err != nil {
		return err
	}
	cols := dao.SysMenu.Columns()
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Pid, target.Id).Count()
	if err != nil {
		return gerror.Wrap(err, "检查子菜单失败")
	}
	if count > 0 {
		return gerror.NewCode(consts.CodeBadRequest, "请先删除该菜单下的子菜单")
	}
	if err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := dao.SysMenu.Ctx(ctx).Where(cols.Id, target.Id).Delete(); err != nil {
			return err
		}
		if _, err := dao.SysUserMenu.Ctx(ctx).Where(dao.SysUserMenu.Columns().MenuId, target.Id).Delete(); err != nil {
			return err
		}
		// 同步清理角色对该菜单的授权, 避免孤儿数据
		if _, err := dao.SysRoleMenu.Ctx(ctx).Where(dao.SysRoleMenu.Columns().MenuId, target.Id).Delete(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return gerror.Wrap(err, "删除菜单失败")
	}
	return nil
}

// GrantInfo 查询目标用户的可授权菜单树与已授权菜单
// 超管: 目标为管理员/普通用户, 可授权全部启用菜单
// 管理员: 目标仅限自己的下级普通用户, 仅能下发超管下发给自己的菜单
func (s *Service) GrantInfo(ctx context.Context, userId uint64) (target *entity.SysUser, nodes []*MenuNode, checked []uint64, err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	target, err = s.getUser(ctx, userId)
	if err != nil {
		return nil, nil, nil, err
	}
	if err = s.checkGrantTarget(ctx, actor, target); err != nil {
		return nil, nil, nil, err
	}
	grantable, err := s.grantableMenus(ctx, actor)
	if err != nil {
		return nil, nil, nil, err
	}
	checked, err = s.grantedMenuIds(ctx, target.Id)
	if err != nil {
		return nil, nil, nil, err
	}
	return target, s.buildMenuTree(grantable, 0), checked, nil
}

// Grant 保存用户菜单授权(全量覆盖), 并自动补齐祖先菜单
func (s *Service) Grant(ctx context.Context, userId uint64, menuIds []uint64) (err error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return err
	}
	target, err := s.getUser(ctx, userId)
	if err != nil {
		return err
	}
	if err = s.checkGrantTarget(ctx, actor, target); err != nil {
		return err
	}
	grantable, err := s.grantableMenus(ctx, actor)
	if err != nil {
		return err
	}
	grantableSet := make(map[uint64]bool, len(grantable))
	for _, m := range grantable {
		grantableSet[m.Id] = true
	}
	// 去重 + 补齐祖先 + 校验可授权范围
	finalSet := make(map[uint64]bool)
	menuMap := make(map[uint64]*entity.SysMenu, len(grantable))
	for _, m := range grantable {
		menuMap[m.Id] = m
	}
	for _, id := range menuIds {
		if id == 0 || finalSet[id] {
			continue
		}
		if !grantableSet[id] {
			return gerror.NewCode(consts.CodeForbidden, "无权下发该菜单")
		}
		// 逐级补齐祖先
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
		existing, err := s.grantedMenuIds(ctx, target.Id)
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
		cols := dao.SysUserMenu.Columns()
		if _, err := dao.SysUserMenu.Ctx(ctx).Where(cols.UserId, target.Id).Delete(); err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		records := make([]do.SysUserMenu, 0, len(ids))
		for _, id := range ids {
			records = append(records, do.SysUserMenu{UserId: target.Id, MenuId: id})
		}
		if _, err := dao.SysUserMenu.Ctx(ctx).Data(records).Insert(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return gerror.Wrap(err, "保存菜单授权失败")
	}
	return nil
}

// requireSuperAdmin 校验超管身份
func (s *Service) requireSuperAdmin(ctx context.Context) error {
	_, err := authz.SuperAdmin(ctx, "仅超级管理员可管理路由")
	return err
}

// validateInput 校验菜单输入
func (s *Service) validateInput(ctx context.Context, in CreateInput, selfId uint64) error {
	cols := dao.SysMenu.Columns()
	// 路由名称唯一
	m := dao.SysMenu.Ctx(ctx).Where(cols.Name, in.Name)
	if selfId > 0 {
		m = m.WhereNot(cols.Id, selfId)
	}
	count, err := m.Count()
	if err != nil {
		return gerror.Wrap(err, "校验路由名称失败")
	}
	if count > 0 {
		return gerror.NewCode(consts.CodeBadRequest, "路由名称已存在")
	}
	// 上级菜单必须存在
	if in.Pid > 0 {
		if _, err = s.getById(ctx, in.Pid); err != nil {
			return gerror.NewCode(consts.CodeBadRequest, "上级菜单不存在")
		}
	}
	// 顶级路径必须以 / 开头, 子级使用相对路径
	if in.Pid == 0 {
		if len(in.Path) == 0 || in.Path[0] != '/' {
			return gerror.NewCode(consts.CodeBadRequest, "顶级路由路径必须以 / 开头")
		}
	} else if len(in.Path) > 0 && in.Path[0] == '/' {
		return gerror.NewCode(consts.CodeBadRequest, "子级路由路径不能以 / 开头")
	}
	// 菜单节点必须有组件
	if in.Type == TypeMenu && in.Component == "" {
		return gerror.NewCode(consts.CodeBadRequest, "菜单类型必须填写组件路径")
	}
	if in.Status != 0 && in.Status != 1 {
		return gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	return nil
}

// toDo 转换成 DO 对象
func (s *Service) toDo(in CreateInput) do.SysMenu {
	return do.SysMenu{
		Pid:        in.Pid,
		Name:       in.Name,
		Path:       in.Path,
		Component:  in.Component,
		Redirect:   in.Redirect,
		Type:       in.Type,
		Title:      in.Title,
		Icon:       in.Icon,
		Sort:       in.Sort,
		Status:     in.Status,
		HideInMenu: in.HideInMenu,
		KeepAlive:  in.KeepAlive,
		AffixTab:   in.AffixTab,
		IsPublic:   in.IsPublic,
		Remark:     in.Remark,
	}
}

// getById 查询菜单
func (s *Service) getById(ctx context.Context, id uint64) (*entity.SysMenu, error) {
	var target entity.SysMenu
	if err := dao.SysMenu.Ctx(ctx).Where(dao.SysMenu.Columns().Id, id).Scan(&target); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询菜单失败")
		}
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "菜单不存在")
	}
	return &target, nil
}

// getUser 查询用户
func (s *Service) getUser(ctx context.Context, id uint64) (*entity.SysUser, error) {
	var target entity.SysUser
	if err := dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Id, id).Scan(&target); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询用户失败")
		}
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "用户不存在")
	}
	return &target, nil
}

// checkGrantTarget 校验授权对象是否在操作人的可操作范围内
func (s *Service) checkGrantTarget(ctx context.Context, actor *identity.Identity, target *entity.SysUser) error {
	if target.Role == consts.RoleSuperAdmin {
		return gerror.NewCode(consts.CodeBadRequest, "超级管理员默认拥有全部菜单, 无需授权")
	}
	if actor.IsSuperAdmin() {
		return nil
	}
	if actor.Role != consts.RoleAdmin {
		return gerror.NewCode(consts.CodeForbidden, "无权限分配菜单")
	}
	// 管理员只能给自己的下级普通用户授权
	if target.Role != consts.RoleUser {
		return gerror.NewCode(consts.CodeForbidden, "管理员只能给普通用户分配菜单")
	}
	isDescendant, err := s.isDescendantUser(ctx, actor.Id, target.Id)
	if err != nil {
		return err
	}
	if !isDescendant {
		return gerror.NewCode(consts.CodeForbidden, "只能给自己的下级用户分配菜单")
	}
	return nil
}

// grantableMenus 当前操作人可授权的菜单(启用状态)
func (s *Service) grantableMenus(ctx context.Context, actor *identity.Identity) ([]*entity.SysMenu, error) {
	cols := dao.SysMenu.Columns()
	var menus []*entity.SysMenu
	if err := dao.SysMenu.Ctx(ctx).Where(cols.Status, 1).OrderAsc(cols.Sort).OrderAsc(cols.Id).Scan(&menus); err != nil {
		return nil, gerror.Wrap(err, "查询菜单失败")
	}
	if actor.IsSuperAdmin() {
		return menus, nil
	}
	// 管理员只能下发超管已下发给自己的菜单
	grantedIds, err := s.grantedMenuIds(ctx, actor.Id)
	if err != nil {
		return nil, err
	}
	allowed := make(map[uint64]bool, len(grantedIds))
	for _, id := range grantedIds {
		allowed[id] = true
	}
	var result []*entity.SysMenu
	for _, m := range menus {
		if allowed[m.Id] {
			result = append(result, m)
		}
	}
	return result, nil
}

// grantedMenuIds 查询用户的有效菜单ID = 直授权(sys_user_menu) ∪ 所有启用角色的菜单(sys_role_menu)
// 角色需处于启用状态; 用户无任何角色时与历史行为一致(仅直授权)
func (s *Service) grantedMenuIds(ctx context.Context, userId uint64) (ids []uint64, err error) {
	direct, err := s.directMenuIds(ctx, userId)
	if err != nil {
		return nil, err
	}
	roleIds, err := s.userRoleIds(ctx, userId)
	if err != nil {
		return nil, err
	}
	if len(roleIds) == 0 {
		return direct, nil
	}
	fromRole, err := s.roleMenuIds(ctx, roleIds)
	if err != nil {
		return nil, err
	}
	return mergeMenuIds(direct, fromRole), nil
}

// directMenuIds 查询用户直授权的菜单ID(sys_user_menu)
func (s *Service) directMenuIds(ctx context.Context, userId uint64) (ids []uint64, err error) {
	var records []*entity.SysUserMenu
	cols := dao.SysUserMenu.Columns()
	if err = dao.SysUserMenu.Ctx(ctx).Where(cols.UserId, userId).Fields(cols.MenuId).Scan(&records); err != nil {
		// 无授权记录时 Scan 返回 ErrNoRows
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, gerror.Wrap(err, "查询菜单授权失败")
	}
	ids = make([]uint64, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.MenuId)
	}
	return ids, nil
}

// userRoleIds 查询用户的启用角色ID列表
func (s *Service) userRoleIds(ctx context.Context, userId uint64) (roleIds []uint64, err error) {
	var records []*entity.SysUserRole
	userRoleCols := dao.SysUserRole.Columns()
	roleCols := dao.SysRole.Columns()
	// 仅统计启用且未删除的角色(停用角色的菜单不再生效)
	if err = dao.SysUserRole.Ctx(ctx).
		As("ur").
		LeftJoin(dao.SysRole.Table()+" r", "r."+roleCols.Id+" = ur."+userRoleCols.RoleId).
		Where("ur."+userRoleCols.UserId, userId).
		Where("r."+roleCols.Status, 1).
		Fields("ur." + userRoleCols.RoleId).
		Scan(&records); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, gerror.Wrap(err, "查询用户角色失败")
	}
	roleIds = make([]uint64, 0, len(records))
	for _, r := range records {
		roleIds = append(roleIds, r.RoleId)
	}
	return roleIds, nil
}

// roleMenuIds 查询一组角色的已授权菜单ID(去重)
func (s *Service) roleMenuIds(ctx context.Context, roleIds []uint64) (ids []uint64, err error) {
	if len(roleIds) == 0 {
		return nil, nil
	}
	var records []*entity.SysRoleMenu
	cols := dao.SysRoleMenu.Columns()
	if err = dao.SysRoleMenu.Ctx(ctx).WhereIn(cols.RoleId, roleIds).Fields(cols.MenuId).Scan(&records); err != nil {
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

// mergeMenuIds 合并两组菜单ID并去重(保序: 先 direct 后 fromRole)
func mergeMenuIds(direct []uint64, fromRole []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(direct)+len(fromRole))
	merged := make([]uint64, 0, len(direct)+len(fromRole))
	for _, id := range direct {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, id)
	}
	for _, id := range fromRole {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, id)
	}
	return merged
}

// GrantableMenus 当前操作人可授权的菜单(启用状态), 供角色授权复用
// 超管: 全部启用菜单; 管理员: 超管已下发给自己的菜单(含角色带来的)
func (s *Service) GrantableMenus(ctx context.Context) ([]*entity.SysMenu, error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, err
	}
	return s.grantableMenus(ctx, actor)
}

// GrantedMenuIds 查询用户的有效菜单ID(直授权 ∪ 角色菜单), 供其他服务复用
func (s *Service) GrantedMenuIds(ctx context.Context, userId uint64) ([]uint64, error) {
	return s.grantedMenuIds(ctx, userId)
}

// withAncestors 从全量菜单中补齐 picked 节点的所有祖先
func (s *Service) withAncestors(all []*entity.SysMenu, picked []*entity.SysMenu) []*entity.SysMenu {
	byId := make(map[uint64]*entity.SysMenu, len(all))
	for _, m := range all {
		byId[m.Id] = m
	}
	result := make(map[uint64]*entity.SysMenu, len(picked))
	for _, m := range picked {
		result[m.Id] = m
		parentId := m.Pid
		// 层级防御上限, 防止脏数据成环
		for i := 0; i < 50 && parentId > 0; i++ {
			parent, ok := byId[parentId]
			if !ok {
				break
			}
			if _, exists := result[parent.Id]; exists {
				break
			}
			result[parent.Id] = parent
			parentId = parent.Pid
		}
	}
	// 按排序重新输出
	kept := make([]*entity.SysMenu, 0, len(result))
	for _, m := range all {
		if _, ok := result[m.Id]; ok {
			kept = append(kept, m)
		}
	}
	return kept
}

// buildRoutes 构建下发给前端的路由树
func (s *Service) buildRoutes(menus []*entity.SysMenu, pid uint64) []*RouteNode {
	children := make(map[uint64][]*entity.SysMenu)
	for _, m := range menus {
		children[m.Pid] = append(children[m.Pid], m)
	}
	var walk func(parentId uint64) []*RouteNode
	walk = func(parentId uint64) []*RouteNode {
		list := children[parentId]
		if len(list) == 0 {
			return nil
		}
		nodes := make([]*RouteNode, 0, len(list))
		for _, m := range list {
			node := &RouteNode{
				Name:      m.Name,
				Path:      m.Path,
				Component: m.Component,
				Redirect:  m.Redirect,
				Meta: RouteMeta{
					Title:      m.Title,
					Icon:       m.Icon,
					Order:      m.Sort,
					HideInMenu: m.HideInMenu == 1,
					KeepAlive:  m.KeepAlive == 1,
					AffixTab:   m.AffixTab == 1,
				},
			}
			node.Children = walk(m.Id)
			nodes = append(nodes, node)
		}
		return nodes
	}
	return walk(pid)
}

// BuildMenuTree 构建管理端菜单树(导出供角色服务复用)
func (s *Service) BuildMenuTree(menus []*entity.SysMenu, pid uint64) []*MenuNode {
	return s.buildMenuTree(menus, pid)
}

// buildMenuTree 构建管理端菜单树
func (s *Service) buildMenuTree(menus []*entity.SysMenu, pid uint64) []*MenuNode {
	children := make(map[uint64][]*entity.SysMenu)
	for _, m := range menus {
		children[m.Pid] = append(children[m.Pid], m)
	}
	var walk func(parentId uint64) []*MenuNode
	walk = func(parentId uint64) []*MenuNode {
		list := children[parentId]
		if len(list) == 0 {
			return nil
		}
		nodes := make([]*MenuNode, 0, len(list))
		for _, m := range list {
			nodes = append(nodes, &MenuNode{Menu: m, Children: walk(m.Id)})
		}
		return nodes
	}
	return walk(pid)
}

// descendantMenuIds 获取菜单的所有子孙ID
func (s *Service) descendantMenuIds(ctx context.Context, rootId uint64) (result []uint64, err error) {
	cols := dao.SysMenu.Columns()
	level := []uint64{rootId}
	for i := 0; i < 50 && len(level) > 0; i++ {
		var records []*entity.SysMenu
		if err = dao.SysMenu.Ctx(ctx).WhereIn(cols.Pid, level).Fields(cols.Id).Scan(&records); err != nil {
			return nil, gerror.Wrap(err, "查询子菜单失败")
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

// isDescendantUser 判断 target 是否 root 的子孙用户
func (s *Service) isDescendantUser(ctx context.Context, rootId, targetId uint64) (bool, error) {
	var (
		cols = dao.SysUser.Columns()
		cur  = targetId
	)
	for i := 0; i < 50 && cur > 0; i++ {
		var user entity.SysUser
		if err := dao.SysUser.Ctx(ctx).Where(cols.Id, cur).Fields(cols.Id, cols.Pid).Scan(&user); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, gerror.Wrap(err, "查询用户层级失败")
		}
		if user.Id == 0 {
			return false, nil
		}
		if user.Pid == rootId {
			return true, nil
		}
		cur = user.Pid
	}
	return false, nil
}

// Seed 初始化内置菜单与默认授权(幂等)
func (s *Service) Seed(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       0,
		Name:      "System",
		Path:      "/system",
		Component: ComponentLayout,
		Type:      TypeCatalog,
		Title:     "系统管理",
		Icon:      "lucide:settings",
		Sort:      900,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化菜单失败")
	}
	sysId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "System").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化菜单失败")
	}
	sysMenuId := gconv.Uint64(sysId)
	items := []do.SysMenu{
		{
			Pid: sysMenuId, Name: "SystemUser", Path: "user",
			Component: "/system/user/index", Type: TypeMenu, Title: "用户管理",
			Icon: "lucide:users", Sort: 1, Status: 1, Remark: "系统内置",
		},
		{
			Pid: sysMenuId, Name: "SystemMenu", Path: "menu",
			Component: "/system/menu/index", Type: TypeMenu, Title: "路由管理",
			Icon: "lucide:route", Sort: 2, Status: 1, Remark: "系统内置",
		},
		{
			Pid: 0, Name: "Profile", Path: "/profile", Component: "/_core/profile/index",
			Type: TypeMenu, Title: "个人中心", Icon: "lucide:user",
			Sort: 9999, Status: 1, HideInMenu: true, IsPublic: true, Remark: "所有登录用户可见",
		},
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(items).Insert(); err != nil {
		return gerror.Wrap(err, "初始化菜单失败")
	}
	return nil
}

// SeedGrants 给存量非超管用户下发默认菜单(仅首次初始化时调用)
func (s *Service) SeedGrants(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	sysId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "System").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "查询菜单失败")
	}
	userMenuId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemUser").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "查询菜单失败")
	}
	menuMenuId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemMenu").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "查询菜单失败")
	}
	var users []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).WhereNot(dao.SysUser.Columns().Role, consts.RoleSuperAdmin).Scan(&users); err != nil {
		return gerror.Wrap(err, "查询用户失败")
	}
	var records []do.SysUserMenu
	for _, u := range users {
		// 普通用户默认不下发管理类菜单, 仅保留公共菜单; 由上级按需授权
		if u.Role != consts.RoleAdmin {
			continue
		}
		records = append(records,
			do.SysUserMenu{UserId: u.Id, MenuId: gconv.Uint64(sysId)},
			do.SysUserMenu{UserId: u.Id, MenuId: gconv.Uint64(userMenuId)},
			do.SysUserMenu{UserId: u.Id, MenuId: gconv.Uint64(menuMenuId)},
		)
	}
	if len(records) == 0 {
		return nil
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).Insert(); err != nil {
		return gerror.Wrap(err, "初始化菜单授权失败")
	}
	return nil
}

// SeedLogs 初始化日志管理菜单(幂等, 服务启动时调用)
// 顶级目录: 日志管理; 二级: 登录日志/操作日志; 默认下发给所有存量管理员
func (s *Service) SeedLogs(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	// 幂等: 已存在则跳过
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SysLog").Count()
	if err != nil {
		return gerror.Wrap(err, "初始化日志菜单失败")
	}
	if count > 0 {
		return nil
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       0,
		Name:      "SysLog",
		Path:      "/log",
		Component: ComponentLayout,
		Type:      TypeCatalog,
		Title:     "日志管理",
		Icon:      "lucide:file-clock",
		Sort:      901,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化日志菜单失败")
	}
	sysLogId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SysLog").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化日志菜单失败")
	}
	logMenuId := gconv.Uint64(sysLogId)
	if _, err = dao.SysMenu.Ctx(ctx).Data([]do.SysMenu{
		{
			Pid: logMenuId, Name: "LoginLog", Path: "login",
			Component: "/system/log/login/index", Type: TypeMenu, Title: "登录日志",
			Icon: "lucide:key-round", Sort: 1, Status: 1, Remark: "系统内置",
		},
		{
			Pid: logMenuId, Name: "OperationLog", Path: "operation",
			Component: "/system/log/operation/index", Type: TypeMenu, Title: "操作日志",
			Icon: "lucide:activity", Sort: 2, Status: 1, Remark: "系统内置",
		},
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化日志菜单失败")
	}
	loginLogId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "LoginLog").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化日志菜单失败")
	}
	operationLogId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "OperationLog").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化日志菜单失败")
	}
	// 给所有存量管理员下发日志菜单(InsertIgnore 防止重复授权)
	var admins []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Role, consts.RoleAdmin).Scan(&admins); err != nil {
		return gerror.Wrap(err, "初始化日志菜单授权失败")
	}
	if len(admins) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(admins)*3)
	for _, a := range admins {
		records = append(records,
			do.SysUserMenu{UserId: a.Id, MenuId: logMenuId},
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(loginLogId)},
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(operationLogId)},
		)
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "初始化日志菜单授权失败")
	}
	return nil
}

// SeedRoleMenu 初始化角色管理菜单(幂等, 服务启动时调用)
// 二级菜单: 系统管理/角色管理; 默认下发给所有存量管理员
func (s *Service) SeedRoleMenu(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	// 幂等: 已存在则跳过
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemRole").Count()
	if err != nil {
		return gerror.Wrap(err, "初始化角色管理菜单失败")
	}
	if count > 0 {
		return nil
	}
	sysId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "System").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化角色管理菜单失败")
	}
	if sysId.IsNil() || sysId.Int64() == 0 {
		// 系统管理目录不存在(可能被删除), 跳过本次初始化
		return nil
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       gconv.Uint64(sysId),
		Name:      "SystemRole",
		Path:      "role",
		Component: "/system/role/index",
		Type:      TypeMenu,
		Title:     "角色管理",
		Icon:      "lucide:shield-check",
		Sort:      3,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化角色管理菜单失败")
	}
	roleMenuId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemRole").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化角色管理菜单失败")
	}
	// 给所有存量管理员下发角色管理菜单(InsertIgnore 防止重复授权)
	var admins []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Role, consts.RoleAdmin).Scan(&admins); err != nil {
		return gerror.Wrap(err, "初始化角色管理菜单授权失败")
	}
	if len(admins) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(admins)*2)
	for _, a := range admins {
		records = append(records,
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(sysId)},
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(roleMenuId)},
		)
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "初始化角色管理菜单授权失败")
	}
	return nil
}

// SeedMonitor 初始化系统监控菜单(幂等, 服务启动时调用)
// 顶级目录: 系统监控; 二级: 在线用户; 默认下发给所有存量管理员
func (s *Service) SeedMonitor(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	// 幂等: 已存在则跳过
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SysMonitor").Count()
	if err != nil {
		return gerror.Wrap(err, "初始化系统监控菜单失败")
	}
	if count > 0 {
		return nil
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       0,
		Name:      "SysMonitor",
		Path:      "/monitor",
		Component: ComponentLayout,
		Type:      TypeCatalog,
		Title:     "系统监控",
		Icon:      "lucide:monitor-dot",
		Sort:      902,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化系统监控菜单失败")
	}
	sysMonitorId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SysMonitor").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化系统监控菜单失败")
	}
	monitorMenuId := gconv.Uint64(sysMonitorId)
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       monitorMenuId,
		Name:      "OnlineUser",
		Path:      "online",
		Component: "/monitor/online/index",
		Type:      TypeMenu,
		Title:     "在线用户",
		Icon:      "lucide:users",
		Sort:      1,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化系统监控菜单失败")
	}
	onlineUserId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "OnlineUser").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化系统监控菜单失败")
	}
	// 给所有存量管理员下发系统监控菜单(InsertIgnore 防止重复授权)
	var admins []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Role, consts.RoleAdmin).Scan(&admins); err != nil {
		return gerror.Wrap(err, "初始化系统监控菜单授权失败")
	}
	if len(admins) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(admins)*2)
	for _, a := range admins {
		records = append(records,
			do.SysUserMenu{UserId: a.Id, MenuId: monitorMenuId},
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(onlineUserId)},
		)
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "初始化系统监控菜单授权失败")
	}
	return nil
}

// SeedDict 初始化数据字典管理菜单(幂等, 服务启动时调用)
// 二级菜单: 系统管理/字典管理; 默认下发给所有存量管理员(只读, 写操作仍由服务层限制为超管)
func (s *Service) SeedDict(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	// 幂等: 已存在则跳过
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemDict").Count()
	if err != nil {
		return gerror.Wrap(err, "初始化字典管理菜单失败")
	}
	if count > 0 {
		return nil
	}
	sysId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "System").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化字典管理菜单失败")
	}
	if sysId.IsNil() || sysId.Int64() == 0 {
		// 系统管理目录不存在(可能被删除), 跳过本次初始化
		return nil
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       gconv.Uint64(sysId),
		Name:      "SystemDict",
		Path:      "dict",
		Component: "/system/dict/index",
		Type:      TypeMenu,
		Title:     "字典管理",
		Icon:      "lucide:book-open",
		Sort:      4,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化字典管理菜单失败")
	}
	dictMenuId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemDict").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化字典管理菜单失败")
	}
	// 给所有存量管理员下发字典管理菜单(InsertIgnore 防止重复授权)
	var admins []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Role, consts.RoleAdmin).Scan(&admins); err != nil {
		return gerror.Wrap(err, "初始化字典管理菜单授权失败")
	}
	if len(admins) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(admins)*2)
	for _, a := range admins {
		records = append(records,
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(sysId)},
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(dictMenuId)},
		)
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "初始化字典管理菜单授权失败")
	}
	return nil
}

// SeedConfig 初始化系统参数配置菜单(幂等, 服务启动时调用)
// 二级菜单: 系统管理/参数配置; 默认下发给所有存量管理员(只读, 写操作仅超管)
func (s *Service) SeedConfig(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemConfig").Count()
	if err != nil {
		return gerror.Wrap(err, "初始化参数配置菜单失败")
	}
	if count > 0 {
		return nil
	}
	sysId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "System").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化参数配置菜单失败")
	}
	if sysId.IsNil() || sysId.Int64() == 0 {
		// 系统管理目录不存在(可能被删除), 跳过本次初始化
		return nil
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       gconv.Uint64(sysId),
		Name:      "SystemConfig",
		Path:      "config",
		Component: "/system/config/index",
		Type:      TypeMenu,
		Title:     "参数配置",
		Icon:      "lucide:settings-2",
		Sort:      5,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化参数配置菜单失败")
	}
	configMenuId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemConfig").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化参数配置菜单失败")
	}
	var admins []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Role, consts.RoleAdmin).Scan(&admins); err != nil {
		return gerror.Wrap(err, "初始化参数配置菜单授权失败")
	}
	if len(admins) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(admins)*2)
	for _, a := range admins {
		records = append(records,
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(sysId)},
			do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(configMenuId)},
		)
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "初始化参数配置菜单授权失败")
	}
	return nil
}

// SeedDashboard 初始化工作台菜单(幂等, 服务启动时调用)
// 仅在菜单首次创建时下发给存量管理员; 菜单已存在时不补授权, 由超管在界面上管理
func (s *Service) SeedDashboard(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "Dashboard").Count()
	if err != nil {
		return gerror.Wrap(err, "初始化工作台菜单失败")
	}
	if count > 0 {
		return nil
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       0,
		Name:      "Dashboard",
		Path:      "/dashboard",
		Component: "/dashboard/index",
		Type:      TypeMenu,
		Title:     "工作台",
		Icon:      "lucide:layout-dashboard",
		Sort:      1,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化工作台菜单失败")
	}
	dashboardId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "Dashboard").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化工作台菜单失败")
	}
	if dashboardId.IsNil() || dashboardId.Int64() == 0 {
		return nil
	}
	// 首次创建时下发给存量管理员(InsertIgnore 防止重复授权)
	var admins []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Role, consts.RoleAdmin).Scan(&admins); err != nil {
		return gerror.Wrap(err, "初始化工作台菜单授权失败")
	}
	if len(admins) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(admins))
	for _, a := range admins {
		records = append(records, do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(dashboardId)})
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "初始化工作台菜单授权失败")
	}
	return nil
}

// SeedMonitorServer 初始化「服务监控」子菜单(幂等, 服务启动时调用)
// 注意: 接口仅超级管理员可访问; 菜单仍按既有约定下发给存量管理员,
// 管理员点击后由前端提示"无权查看"(后端返回 403), 而不是把菜单隐藏掉。
func (s *Service) SeedMonitorServer(ctx context.Context) (err error) {
	return s.seedMonitorChild(ctx, "MonitorServer", do.SysMenu{
		Path:      "server",
		Component: "/monitor/server/index",
		Title:     "服务监控",
		Icon:      "lucide:server",
		Sort:      2,
	}, "初始化服务监控菜单失败")
}

// SeedMonitorJob 初始化「定时任务」子菜单(幂等, 服务启动时调用)
// 管理员可查看列表与日志(只读), 写操作仅超级管理员。
func (s *Service) SeedMonitorJob(ctx context.Context) (err error) {
	return s.seedMonitorChild(ctx, "MonitorJob", do.SysMenu{
		Path:      "job",
		Component: "/monitor/job/index",
		Title:     "定时任务",
		Icon:      "lucide:timer",
		Sort:      3,
	}, "初始化定时任务菜单失败")
}

// SeedNotice 初始化「通知公告」菜单(幂等, 服务启动时调用)
// 二级菜单: 系统管理/通知公告; 仅在菜单首次创建时下发给存量管理员
func (s *Service) SeedNotice(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemNotice").Count()
	if err != nil {
		return gerror.Wrap(err, "初始化通知公告菜单失败")
	}
	// 仅在菜单首次创建时授权; 菜单已存在时不补授权, 由超管在界面上管理
	if count > 0 {
		return nil
	}
	sysId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "System").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化通知公告菜单失败")
	}
	if sysId.IsNil() || sysId.Int64() == 0 {
		// 系统管理目录不存在(可能被删除), 跳过本次初始化
		return nil
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       gconv.Uint64(sysId),
		Name:      "SystemNotice",
		Path:      "notice",
		Component: "/system/notice/index",
		Type:      TypeMenu,
		Title:     "通知公告",
		Icon:      "lucide:megaphone",
		Sort:      6,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化通知公告菜单失败")
	}
	menuId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SystemNotice").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化通知公告菜单失败")
	}
	if menuId.IsNil() || menuId.Int64() == 0 {
		return nil
	}
	// 首次创建时下发给存量管理员(InsertIgnore 防止重复授权)
	var admins []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Role, consts.RoleAdmin).Scan(&admins); err != nil {
		return gerror.Wrap(err, "初始化通知公告菜单授权失败")
	}
	if len(admins) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(admins)*2)
	for _, a := range admins {
		records = append(records, do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(menuId)})
		if !sysId.IsNil() && sysId.Int64() > 0 {
			records = append(records, do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(sysId)})
		}
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "初始化通知公告菜单授权失败")
	}
	return nil
}

// SeedMyNotice 初始化顶级「我的通知」菜单(幂等, 服务启动时调用)
// 仅在菜单首次创建时对全部存量角色(admin、user)授权, 超管天然可见全部菜单
func (s *Service) SeedMyNotice(ctx context.Context) (err error) {
	cols := dao.SysMenu.Columns()
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "MyNotice").Count()
	if err != nil {
		return gerror.Wrap(err, "初始化我的通知菜单失败")
	}
	// 仅在菜单首次创建时授权; 菜单已存在时不补授权, 由超管在界面上管理
	if count > 0 {
		return nil
	}
	if _, err = dao.SysMenu.Ctx(ctx).Data(do.SysMenu{
		Pid:       0,
		Name:      "MyNotice",
		Path:      "/notice",
		Component: "/notice/index",
		Type:      TypeMenu,
		Title:     "我的通知",
		Icon:      "lucide:bell",
		Sort:      2,
		Status:    1,
		Remark:    "系统内置",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化我的通知菜单失败")
	}
	menuId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "MyNotice").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, "初始化我的通知菜单失败")
	}
	if menuId.IsNil() || menuId.Int64() == 0 {
		return nil
	}
	// 首次创建时对存量 admin/user 授权(InsertIgnore 幂等)
	var users []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).
		WhereIn(dao.SysUser.Columns().Role, []string{consts.RoleAdmin, consts.RoleUser}).
		Scan(&users); err != nil {
		return gerror.Wrap(err, "初始化我的通知菜单授权失败")
	}
	if len(users) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(users))
	for _, u := range users {
		records = append(records, do.SysUserMenu{UserId: u.Id, MenuId: gconv.Uint64(menuId)})
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "初始化我的通知菜单授权失败")
	}
	return nil
}

// seedMonitorChild 在系统监控父菜单下幂等插入子菜单, 仅在子菜单首次创建时对存量管理员授权
func (s *Service) seedMonitorChild(ctx context.Context, name string, menu do.SysMenu, errMsg string) error {
	cols := dao.SysMenu.Columns()
	parentId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, "SysMonitor").Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, errMsg)
	}
	if parentId.IsNil() || parentId.Int64() == 0 {
		// 父菜单尚未初始化(理论上 SeedMonitor 已保证), 这里安全跳过
		return nil
	}
	count, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, name).Count()
	if err != nil {
		return gerror.Wrap(err, errMsg)
	}
	// 仅在子菜单首次创建时授权; 菜单已存在时不补授权, 由超管在界面上管理
	if count > 0 {
		return nil
	}
	menu.Pid = gconv.Uint64(parentId)
	menu.Name = name
	menu.Type = TypeMenu
	menu.Status = 1
	menu.Remark = "系统内置"
	if _, err = dao.SysMenu.Ctx(ctx).Data(menu).Insert(); err != nil {
		return gerror.Wrap(err, errMsg)
	}
	menuId, err := dao.SysMenu.Ctx(ctx).Where(cols.Name, name).Value(cols.Id)
	if err != nil {
		return gerror.Wrap(err, errMsg)
	}
	if menuId.IsNil() || menuId.Int64() == 0 {
		return nil
	}
	// 首次创建时对存量管理员授权(InsertIgnore 幂等)
	var admins []*entity.SysUser
	if err = dao.SysUser.Ctx(ctx).Where(dao.SysUser.Columns().Role, consts.RoleAdmin).Scan(&admins); err != nil {
		return gerror.Wrap(err, errMsg+"(授权)")
	}
	if len(admins) == 0 {
		return nil
	}
	records := make([]do.SysUserMenu, 0, len(admins))
	for _, a := range admins {
		records = append(records, do.SysUserMenu{UserId: a.Id, MenuId: gconv.Uint64(menuId)})
	}
	if _, err = dao.SysUserMenu.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
		return gerror.Wrap(err, errMsg+"(授权)")
	}
	return nil
}
