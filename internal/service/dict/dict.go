// Package dict 数据字典服务: 字典类型/字典项管理, 按类型批量取值(内存缓存), 内置种子数据
package dict

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/authz"
	"demo/internal/service/identity"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
)

// 内置字典类型编码
const (
	TypeUserStatus  = "sys_user_status" // 用户状态
	TypeLogType     = "sys_log_type"    // 日志类型(成功/失败)
	TypeNoticeType  = "notice_type"     // 公告类型(通知/公告)
	TypeNoticeLevel = "notice_level"    // 公告级别(普通/重要/紧急)
)

// 字典项最大批量查询数量, 防止单次请求过大
const maxBatchTypes = 50

var typeCodePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{1,63}$`)

// Service 字典服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// ------------------------------------------------------------
// 类型定义
// ------------------------------------------------------------

// TypeListInput 字典类型列表查询条件
type TypeListInput struct {
	Keyword string // 编码/名称模糊匹配
	Status  int    // -1=全部, 0=停用, 1=启用
}

// TypeCreateInput 新增字典类型
type TypeCreateInput struct {
	Type   string
	Name   string
	Status int
	Remark string
}

// TypeUpdateInput 修改字典类型(编码不可变更)
type TypeUpdateInput struct {
	Id     uint64
	Name   string
	Status int
	Remark string
}

// DataListInput 字典项列表查询条件
type DataListInput struct {
	DictType string
	Keyword  string
	Status   int // -1=全部
	Page     int
	PageSize int
}

// DataCreateInput 新增字典项
type DataCreateInput struct {
	DictType string
	Label    string
	Value    string
	Sort     int
	Status   int
	TagColor string
	Remark   string
}

// DataUpdateInput 修改字典项
type DataUpdateInput struct {
	Id       uint64
	DictType string
	Label    string
	Value    string
	Sort     int
	Status   int
	TagColor string
	Remark   string
}

// ------------------------------------------------------------
// 缓存: 同一 type 只查库一次, 写操作后整体失效
// ------------------------------------------------------------

type cacheEntry struct {
	items    []*entity.SysDictData
	expireAt time.Time
}

var (
	optionsCache sync.Map // map[string]cacheEntry
)

// ClearCache 清空字典缓存(测试/运维使用)
func ClearCache() {
	optionsCache.Range(func(key, _ any) bool {
		optionsCache.Delete(key)
		return true
	})
}

func cacheConfig(ctx context.Context) (enabled bool, ttl time.Duration) {
	enabled = g.Cfg().MustGet(ctx, "dict.cache.enabled", true).Bool()
	seconds := g.Cfg().MustGet(ctx, "dict.cache.ttlSeconds", 300).Int()
	if seconds <= 0 {
		seconds = 300
	}
	return enabled, time.Duration(seconds) * time.Second
}

// invalidateCache 字典数据变更后清空全部缓存, 避免各 type 读到旧值
func invalidateCache() {
	ClearCache()
}

// ------------------------------------------------------------
// 权限
// ------------------------------------------------------------

// requireRead 管理端读权限: 超管/管理员
func requireRead(ctx context.Context) (*identity.Identity, error) {
	return authz.Manager(ctx, "无权限查看数据字典")
}

// requireWrite 写权限: 仅超级管理员(与菜单/角色等系统配置类管理一致)
func requireWrite(ctx context.Context) (*identity.Identity, error) {
	return authz.SuperAdmin(ctx, "仅超级管理员可维护数据字典")
}

// ------------------------------------------------------------
// 字典类型 CRUD
// ------------------------------------------------------------

// TypeList 字典类型列表
func (s *Service) TypeList(ctx context.Context, in TypeListInput) (list []*entity.SysDictType, err error) {
	if _, err = requireRead(ctx); err != nil {
		return nil, err
	}
	cols := dao.SysDictType.Columns()
	m := dao.SysDictType.Ctx(ctx)
	if kw := strings.TrimSpace(in.Keyword); kw != "" {
		p := "%" + kw + "%"
		m = m.Where(fmt.Sprintf("(`%s` LIKE ? OR `%s` LIKE ?)", cols.Type, cols.Name), p, p)
	}
	if in.Status == 0 || in.Status == 1 {
		m = m.Where(cols.Status, in.Status)
	}
	if err = m.OrderAsc(cols.Id).Scan(&list); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询字典类型失败")
	}
	return list, nil
}

// TypeCreate 新增字典类型
func (s *Service) TypeCreate(ctx context.Context, in TypeCreateInput) (id uint64, err error) {
	actor, err := requireWrite(ctx)
	if err != nil {
		return 0, err
	}
	in.Type = strings.TrimSpace(in.Type)
	in.Name = strings.TrimSpace(in.Name)
	if !typeCodePattern.MatchString(in.Type) {
		return 0, gerror.NewCode(consts.CodeBadRequest, "字典类型编码需以字母开头, 仅允许字母/数字/下划线, 长度2-64")
	}
	if in.Name == "" {
		return 0, gerror.NewCode(consts.CodeBadRequest, "字典类型名称不能为空")
	}
	if in.Status != 0 && in.Status != 1 {
		return 0, gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	cols := dao.SysDictType.Columns()
	count, err := dao.SysDictType.Ctx(ctx).Where(cols.Type, in.Type).Count()
	if err != nil {
		return 0, gerror.Wrap(err, "校验字典类型失败")
	}
	if count > 0 {
		return 0, gerror.NewCode(consts.CodeBadRequest, "字典类型编码已存在")
	}
	lastId, err := dao.SysDictType.Ctx(ctx).Data(do.SysDictType{
		Type:      in.Type,
		Name:      in.Name,
		Status:    in.Status,
		Remark:    in.Remark,
		CreatedBy: actor.Id,
	}).InsertAndGetId()
	if err != nil {
		return 0, gerror.Wrap(err, "新增字典类型失败")
	}
	invalidateCache()
	return gconv.Uint64(lastId), nil
}

// TypeUpdate 修改字典类型
func (s *Service) TypeUpdate(ctx context.Context, in TypeUpdateInput) (err error) {
	if _, err = requireWrite(ctx); err != nil {
		return err
	}
	if _, err = s.getTypeById(ctx, in.Id); err != nil {
		return err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return gerror.NewCode(consts.CodeBadRequest, "字典类型名称不能为空")
	}
	if in.Status != 0 && in.Status != 1 {
		return gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	if _, err = dao.SysDictType.Ctx(ctx).Where(dao.SysDictType.Columns().Id, in.Id).Data(do.SysDictType{
		Name:   in.Name,
		Status: in.Status,
		Remark: in.Remark,
	}).Update(); err != nil {
		return gerror.Wrap(err, "修改字典类型失败")
	}
	invalidateCache()
	return nil
}

// TypeDelete 删除字典类型(软删除), 同时软删除其下字典项
func (s *Service) TypeDelete(ctx context.Context, id uint64) (err error) {
	if _, err = requireWrite(ctx); err != nil {
		return err
	}
	target, err := s.getTypeById(ctx, id)
	if err != nil {
		return err
	}
	if _, err = dao.SysDictType.Ctx(ctx).Where(dao.SysDictType.Columns().Id, target.Id).Delete(); err != nil {
		return gerror.Wrap(err, "删除字典类型失败")
	}
	// 级联软删除字典项, 避免孤儿数据
	if _, err = dao.SysDictData.Ctx(ctx).Where(dao.SysDictData.Columns().DictType, target.Type).Delete(); err != nil {
		return gerror.Wrap(err, "删除字典项失败")
	}
	invalidateCache()
	return nil
}

// TypeStatus 启用/停用字典类型
func (s *Service) TypeStatus(ctx context.Context, id uint64, status int) (err error) {
	if _, err = requireWrite(ctx); err != nil {
		return err
	}
	if status != 0 && status != 1 {
		return gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	if _, err = s.getTypeById(ctx, id); err != nil {
		return err
	}
	if _, err = dao.SysDictType.Ctx(ctx).Where(dao.SysDictType.Columns().Id, id).Data(do.SysDictType{
		Status: status,
	}).Update(); err != nil {
		return gerror.Wrap(err, "修改字典类型状态失败")
	}
	invalidateCache()
	return nil
}

// ------------------------------------------------------------
// 字典项 CRUD
// ------------------------------------------------------------

// DataList 字典项列表(管理端, 含停用项)
func (s *Service) DataList(ctx context.Context, in DataListInput) (list []*entity.SysDictData, total int, err error) {
	if _, err = requireRead(ctx); err != nil {
		return nil, 0, err
	}
	in.DictType = strings.TrimSpace(in.DictType)
	if in.DictType == "" {
		return nil, 0, gerror.NewCode(consts.CodeBadRequest, "字典类型不能为空")
	}
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.PageSize <= 0 || in.PageSize > 200 {
		in.PageSize = 20
	}
	cols := dao.SysDictData.Columns()
	m := dao.SysDictData.Ctx(ctx).Where(cols.DictType, in.DictType)
	if kw := strings.TrimSpace(in.Keyword); kw != "" {
		p := "%" + kw + "%"
		m = m.Where(fmt.Sprintf("(`%s` LIKE ? OR `%s` LIKE ?)", cols.Label, cols.Value), p, p)
	}
	if in.Status == 0 || in.Status == 1 {
		m = m.Where(cols.Status, in.Status)
	}
	if total, err = m.Count(); err != nil {
		return nil, 0, gerror.Wrap(err, "查询字典项失败")
	}
	if err = m.OrderAsc(cols.Sort).OrderAsc(cols.Id).Page(in.Page, in.PageSize).Scan(&list); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return nil, 0, gerror.Wrap(err, "查询字典项失败")
	}
	return list, total, nil
}

// DataCreate 新增字典项
func (s *Service) DataCreate(ctx context.Context, in DataCreateInput) (id uint64, err error) {
	if _, err = requireWrite(ctx); err != nil {
		return 0, err
	}
	in.DictType = strings.TrimSpace(in.DictType)
	in.Label = strings.TrimSpace(in.Label)
	in.Value = strings.TrimSpace(in.Value)
	if in.DictType == "" {
		return 0, gerror.NewCode(consts.CodeBadRequest, "字典类型不能为空")
	}
	if in.Label == "" || in.Value == "" {
		return 0, gerror.NewCode(consts.CodeBadRequest, "显示名称与字典值不能为空")
	}
	if in.Status != 0 && in.Status != 1 {
		return 0, gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	if _, err = s.getTypeByCode(ctx, in.DictType); err != nil {
		return 0, err
	}
	if err = s.ensureValueUnique(ctx, in.DictType, in.Value, 0); err != nil {
		return 0, err
	}
	lastId, err := dao.SysDictData.Ctx(ctx).Data(do.SysDictData{
		DictType: in.DictType,
		Label:    in.Label,
		Value:    in.Value,
		Sort:     in.Sort,
		Status:   in.Status,
		TagColor: in.TagColor,
		Remark:   in.Remark,
	}).InsertAndGetId()
	if err != nil {
		return 0, gerror.Wrap(err, "新增字典项失败")
	}
	invalidateCache()
	return gconv.Uint64(lastId), nil
}

// DataUpdate 修改字典项
func (s *Service) DataUpdate(ctx context.Context, in DataUpdateInput) (err error) {
	if _, err = requireWrite(ctx); err != nil {
		return err
	}
	target, err := s.getDataById(ctx, in.Id)
	if err != nil {
		return err
	}
	in.Label = strings.TrimSpace(in.Label)
	in.Value = strings.TrimSpace(in.Value)
	if in.Label == "" || in.Value == "" {
		return gerror.NewCode(consts.CodeBadRequest, "显示名称与字典值不能为空")
	}
	if in.Status != 0 && in.Status != 1 {
		return gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	dictType := target.DictType
	if in.DictType != "" {
		dictType = strings.TrimSpace(in.DictType)
	}
	if dictType != target.DictType {
		if _, err = s.getTypeByCode(ctx, dictType); err != nil {
			return err
		}
	}
	if err = s.ensureValueUnique(ctx, dictType, in.Value, target.Id); err != nil {
		return err
	}
	if _, err = dao.SysDictData.Ctx(ctx).Where(dao.SysDictData.Columns().Id, target.Id).Data(do.SysDictData{
		DictType: dictType,
		Label:    in.Label,
		Value:    in.Value,
		Sort:     in.Sort,
		Status:   in.Status,
		TagColor: in.TagColor,
		Remark:   in.Remark,
	}).Update(); err != nil {
		return gerror.Wrap(err, "修改字典项失败")
	}
	invalidateCache()
	return nil
}

// DataDelete 删除字典项(软删除)
func (s *Service) DataDelete(ctx context.Context, id uint64) (err error) {
	if _, err = requireWrite(ctx); err != nil {
		return err
	}
	if _, err = s.getDataById(ctx, id); err != nil {
		return err
	}
	if _, err = dao.SysDictData.Ctx(ctx).Where(dao.SysDictData.Columns().Id, id).Delete(); err != nil {
		return gerror.Wrap(err, "删除字典项失败")
	}
	invalidateCache()
	return nil
}

// DataStatus 启用/停用字典项
func (s *Service) DataStatus(ctx context.Context, id uint64, status int) (err error) {
	if _, err = requireWrite(ctx); err != nil {
		return err
	}
	if status != 0 && status != 1 {
		return gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	if _, err = s.getDataById(ctx, id); err != nil {
		return err
	}
	if _, err = dao.SysDictData.Ctx(ctx).Where(dao.SysDictData.Columns().Id, id).Data(do.SysDictData{
		Status: status,
	}).Update(); err != nil {
		return gerror.Wrap(err, "修改字典项状态失败")
	}
	invalidateCache()
	return nil
}

// ------------------------------------------------------------
// 按 type 批量取值(供前端下拉/标签使用, 任意登录用户可读)
// ------------------------------------------------------------

// Options 批量获取多个字典类型的启用项, 返回 map[type][]item
// 停用的类型/字典项不会返回; 同一 type 在缓存有效期内只查库一次
func (s *Service) Options(ctx context.Context, types []string) (result map[string][]*entity.SysDictData, err error) {
	if _, err = authz.Login(ctx); err != nil {
		return nil, err
	}
	result = make(map[string][]*entity.SysDictData, len(types))
	// 去重 + 过滤空值, 同时限制批量大小
	seen := make(map[string]struct{}, len(types))
	wanted := make([]string, 0, len(types))
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		wanted = append(wanted, t)
		if len(wanted) > maxBatchTypes {
			return nil, gerror.NewCode(consts.CodeBadRequest, fmt.Sprintf("单次最多查询 %d 个字典类型", maxBatchTypes))
		}
	}
	if len(wanted) == 0 {
		return result, nil
	}
	enabled, ttl := cacheConfig(ctx)
	now := time.Now()
	miss := make([]string, 0, len(wanted))
	for _, t := range wanted {
		if enabled {
			if v, ok := optionsCache.Load(t); ok {
				entry := v.(cacheEntry)
				if entry.expireAt.After(now) {
					result[t] = entry.items
					continue
				}
				optionsCache.Delete(t)
			}
		}
		miss = append(miss, t)
	}
	if len(miss) == 0 {
		return result, nil
	}
	loaded, err := s.loadEnabled(ctx, miss)
	if err != nil {
		return nil, err
	}
	for _, t := range miss {
		items := loaded[t]
		if items == nil {
			items = []*entity.SysDictData{}
		}
		result[t] = items
		if enabled {
			optionsCache.Store(t, cacheEntry{items: items, expireAt: now.Add(ttl)})
		}
	}
	return result, nil
}

// loadEnabled 从库里一次加载多个 type 的启用项(类型必须启用)
func (s *Service) loadEnabled(ctx context.Context, types []string) (map[string][]*entity.SysDictData, error) {
	result := make(map[string][]*entity.SysDictData, len(types))
	valid := make([]string, 0, len(types))
	// 一次查询获取有效的类型
	var typeRows []*entity.SysDictType
	cols := dao.SysDictType.Columns()
	err := dao.SysDictType.Ctx(ctx).
		WhereIn(cols.Type, types).
		Where(cols.Status, 1).
		Fields(cols.Type).
		Scan(&typeRows)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询字典类型失败")
	}
	for _, t := range typeRows {
		valid = append(valid, t.Type)
	}
	if len(valid) == 0 {
		return result, nil
	}
	var rows []*entity.SysDictData
	dataCols := dao.SysDictData.Columns()
	err = dao.SysDictData.Ctx(ctx).
		WhereIn(dataCols.DictType, valid).
		Where(dataCols.Status, 1).
		OrderAsc(dataCols.Sort).
		OrderAsc(dataCols.Id).
		Scan(&rows)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询字典项失败")
	}
	for _, row := range rows {
		result[row.DictType] = append(result[row.DictType], row)
	}
	return result, nil
}

// ------------------------------------------------------------
// 内部工具
// ------------------------------------------------------------

func (s *Service) getTypeById(ctx context.Context, id uint64) (*entity.SysDictType, error) {
	var target entity.SysDictType
	err := dao.SysDictType.Ctx(ctx).Where(dao.SysDictType.Columns().Id, id).Scan(&target)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询字典类型失败")
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "字典类型不存在")
	}
	return &target, nil
}

func (s *Service) getTypeByCode(ctx context.Context, code string) (*entity.SysDictType, error) {
	var target entity.SysDictType
	err := dao.SysDictType.Ctx(ctx).Where(dao.SysDictType.Columns().Type, code).Scan(&target)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询字典类型失败")
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeBadRequest, "字典类型不存在")
	}
	return &target, nil
}

func (s *Service) getDataById(ctx context.Context, id uint64) (*entity.SysDictData, error) {
	var target entity.SysDictData
	err := dao.SysDictData.Ctx(ctx).Where(dao.SysDictData.Columns().Id, id).Scan(&target)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询字典项失败")
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "字典项不存在")
	}
	return &target, nil
}

// ensureValueUnique 同一 type 下 value 唯一
func (s *Service) ensureValueUnique(ctx context.Context, dictType, value string, selfId uint64) error {
	cols := dao.SysDictData.Columns()
	m := dao.SysDictData.Ctx(ctx).Where(cols.DictType, dictType).Where(cols.Value, value)
	if selfId > 0 {
		m = m.WhereNot(cols.Id, selfId)
	}
	count, err := m.Count()
	if err != nil {
		return gerror.Wrap(err, "校验字典值失败")
	}
	if count > 0 {
		return gerror.NewCode(consts.CodeBadRequest, "同一字典类型下字典值已存在")
	}
	return nil
}

// ------------------------------------------------------------
// 种子数据(幂等)
// ------------------------------------------------------------

// Seed 写入内置字典类型与字典项, 已存在的跳过
func (s *Service) Seed(ctx context.Context) (err error) {
	typeCols := dao.SysDictType.Columns()
	types := []do.SysDictType{
		{Type: TypeUserStatus, Name: "用户状态", Status: 1, Remark: "系统内置"},
		{Type: TypeLogType, Name: "日志类型", Status: 1, Remark: "系统内置"},
		{Type: TypeNoticeType, Name: "公告类型", Status: 1, Remark: "系统内置"},
		{Type: TypeNoticeLevel, Name: "公告级别", Status: 1, Remark: "系统内置"},
	}
	for _, t := range types {
		count, err := dao.SysDictType.Ctx(ctx).Where(typeCols.Type, t.Type).Count()
		if err != nil {
			return gerror.Wrap(err, "初始化字典类型失败")
		}
		if count > 0 {
			continue
		}
		if _, err = dao.SysDictType.Ctx(ctx).Data(t).Insert(); err != nil {
			return gerror.Wrap(err, "初始化字典类型失败")
		}
	}
	dataCols := dao.SysDictData.Columns()
	items := []do.SysDictData{
		{DictType: TypeUserStatus, Label: "启用", Value: "1", Sort: 1, Status: 1, TagColor: "success", Remark: "系统内置"},
		{DictType: TypeUserStatus, Label: "禁用", Value: "0", Sort: 2, Status: 1, TagColor: "danger", Remark: "系统内置"},
		{DictType: TypeLogType, Label: "成功", Value: "1", Sort: 1, Status: 1, TagColor: "success", Remark: "系统内置"},
		{DictType: TypeLogType, Label: "失败", Value: "0", Sort: 2, Status: 1, TagColor: "danger", Remark: "系统内置"},
		{DictType: TypeNoticeType, Label: "通知", Value: "notice", Sort: 1, Status: 1, TagColor: "info", Remark: "系统内置"},
		{DictType: TypeNoticeType, Label: "公告", Value: "announcement", Sort: 2, Status: 1, TagColor: "success", Remark: "系统内置"},
		{DictType: TypeNoticeLevel, Label: "普通", Value: "normal", Sort: 1, Status: 1, TagColor: "info", Remark: "系统内置"},
		{DictType: TypeNoticeLevel, Label: "重要", Value: "important", Sort: 2, Status: 1, TagColor: "warning", Remark: "系统内置"},
		{DictType: TypeNoticeLevel, Label: "紧急", Value: "urgent", Sort: 3, Status: 1, TagColor: "danger", Remark: "系统内置"},
	}
	for _, item := range items {
		count, err := dao.SysDictData.Ctx(ctx).
			Where(dataCols.DictType, item.DictType).
			Where(dataCols.Value, item.Value).
			Count()
		if err != nil {
			return gerror.Wrap(err, "初始化字典项失败")
		}
		if count > 0 {
			continue
		}
		if _, err = dao.SysDictData.Ctx(ctx).Data(item).Insert(); err != nil {
			return gerror.Wrap(err, "初始化字典项失败")
		}
	}
	invalidateCache()
	return nil
}
