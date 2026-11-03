// Package sysconfig 系统参数配置: 运行时可变的业务/展示参数, 带内存缓存与类型化读取
//
// 边界说明(重要): 本表只允许存放"运行时可变的业务/展示参数"(仪表盘趋势天数、
// 列表默认条数、导入导出上限等)。登录失败阈值、锁定窗口、验证码阈值、密码复杂度、
// 会话上限等安全参数必须继续由 manifest/config/config.yaml 的 security 段维护,
// 否则等于让"管理员可以修改的值"去约束管理员自己的登录安全。
package sysconfig

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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
)

// 内置参数键(运行时业务/展示参数)
const (
	KeyDashboardTrendDays   = "dashboard.trendDays"   // 仪表盘趋势天数
	KeyDashboardRecentLimit = "dashboard.recentLimit" // 最近动态条数
	KeyExportMaxRows        = "export.maxRows"        // 单次导出最大行数
	KeyImportMaxRows        = "import.maxRows"        // 单次导入最大数据行数
	KeyImportMaxFileBytes   = "import.maxFileBytes"   // 导入文件大小上限(字节)
)

// maxBatchKeys 单次批量读取的参数个数上限
const maxBatchKeys = 50

// DefaultItem 内置参数定义
type DefaultItem struct {
	Key     string
	Value   string // 默认值(重置为默认时使用)
	Name    string
	Remark  string
	Min     int // 数值型参数的最小值, <=0 表示不校验下限
	Max     int // 数值型参数的最大值, <=0 表示不校验上限
	Numeric bool
}

// builtinDefaults 内置参数清单(种子数据与重置默认的唯一来源)
var builtinDefaults = []DefaultItem{
	{
		Key: KeyDashboardTrendDays, Value: "7", Name: "仪表盘趋势天数",
		Remark: "工作台登录趋势展示的天数, 范围 1-90", Min: 1, Max: 90, Numeric: true,
	},
	{
		Key: KeyDashboardRecentLimit, Value: "8", Name: "仪表盘最近动态条数",
		Remark: "工作台最近操作动态展示条数, 范围 1-50", Min: 1, Max: 50, Numeric: true,
	},
	{
		Key: KeyExportMaxRows, Value: "500000", Name: "单次导出行数上限",
		Remark: "列表导出单次最大行数, 0 表示不限制", Min: 0, Max: 1000000, Numeric: true,
	},
	{
		Key: KeyImportMaxRows, Value: "5000", Name: "单次导入最大行数",
		Remark: "Excel 导入单次最大数据行数", Min: 1, Max: 50000, Numeric: true,
	},
	{
		Key: KeyImportMaxFileBytes, Value: "5242880", Name: "导入文件大小上限",
		Remark: "Excel 导入文件最大字节数, 默认 5MB", Min: 1024, Max: 52428800, Numeric: true,
	},
}

// Service 系统参数配置服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// ------------------------------------------------------------
// 缓存: 整表快照, TTL 到期或任意写操作后失效
// ------------------------------------------------------------

type snapshot struct {
	values   map[string]string
	expireAt time.Time
}

var (
	snapMu sync.RWMutex
	snap   *snapshot
)

// ClearCache 清空参数缓存(测试/运维使用)
func ClearCache() {
	snapMu.Lock()
	snap = nil
	snapMu.Unlock()
}

func cacheConfig(ctx context.Context) (enabled bool, ttl time.Duration) {
	enabled = g.Cfg().MustGet(ctx, "sysConfig.cache.enabled", true).Bool()
	seconds := g.Cfg().MustGet(ctx, "sysConfig.cache.ttlSeconds", 300).Int()
	if seconds <= 0 {
		seconds = 300
	}
	return enabled, time.Duration(seconds) * time.Second
}

// loadValues 返回参数快照(含内置默认值兜底), 缓存有效期内不查库
func loadValues(ctx context.Context) (map[string]string, error) {
	enabled, ttl := cacheConfig(ctx)
	now := time.Now()
	if enabled {
		snapMu.RLock()
		if snap != nil && snap.expireAt.After(now) {
			values := snap.values
			snapMu.RUnlock()
			return values, nil
		}
		snapMu.RUnlock()
	}
	cols := dao.SysConfig.Columns()
	var rows []*entity.SysConfig
	if err := dao.SysConfig.Ctx(ctx).
		Where(cols.Status, 1).
		Fields(cols.ConfigKey, cols.ConfigValue).
		Scan(&rows); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询系统参数失败")
	}
	values := make(map[string]string, len(rows)+len(builtinDefaults))
	for _, item := range builtinDefaults {
		values[item.Key] = item.Value
	}
	for _, row := range rows {
		values[row.ConfigKey] = row.ConfigValue
	}
	if enabled {
		snapMu.Lock()
		snap = &snapshot{values: values, expireAt: now.Add(ttl)}
		snapMu.Unlock()
	}
	return values, nil
}

// ------------------------------------------------------------
// 类型化读取(供其他 service 直接调用, 避免各处硬编码)
// ------------------------------------------------------------

// GetString 读取字符串参数: 库里无值/停用/为空时回退内置默认值, 再回退 def
func (s *Service) GetString(ctx context.Context, key, def string) string {
	values, err := loadValues(ctx)
	if err != nil {
		if builtin, ok := DefaultValue(key); ok {
			return builtin
		}
		return def
	}
	if v, ok := values[key]; ok {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return def
}

// GetInt 读取整型参数, 解析失败时回退 def
func (s *Service) GetInt(ctx context.Context, key string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(s.GetString(ctx, key, "")))
	if err != nil {
		return def
	}
	return v
}

// GetBool 读取布尔参数
func (s *Service) GetBool(ctx context.Context, key string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s.GetString(ctx, key, ""))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

// GetDuration 读取时长参数(值为秒数)
func (s *Service) GetDuration(ctx context.Context, key string, def time.Duration) time.Duration {
	v, err := strconv.Atoi(strings.TrimSpace(s.GetString(ctx, key, "")))
	if err != nil || v < 0 {
		return def
	}
	return time.Duration(v) * time.Second
}

// ------------------------------------------------------------
// 权限
// ------------------------------------------------------------

// requireLogin 读权限: 任意已登录用户
func requireLogin(ctx context.Context) (*identity.Identity, error) {
	return authz.Login(ctx)
}

// requireWrite 写权限: 仅超级管理员
func requireWrite(ctx context.Context) (*identity.Identity, error) {
	return authz.SuperAdmin(ctx, "仅超级管理员可维护系统参数")
}

// ------------------------------------------------------------
// 查询
// ------------------------------------------------------------

// ListInput 列表查询条件
type ListInput struct {
	Keyword string // 键/名称模糊匹配
	Status  int    // -1=全部 1=启用 0=停用
}

// List 系统参数列表(已登录用户可读)
func (s *Service) List(ctx context.Context, in ListInput) (list []*entity.SysConfig, err error) {
	if _, err = requireLogin(ctx); err != nil {
		return nil, err
	}
	cols := dao.SysConfig.Columns()
	m := dao.SysConfig.Ctx(ctx)
	if kw := strings.TrimSpace(in.Keyword); kw != "" {
		p := "%" + kw + "%"
		m = m.Where(fmt.Sprintf("(`%s` LIKE ? OR `%s` LIKE ?)", cols.ConfigKey, cols.Name), p, p)
	}
	if in.Status == 0 || in.Status == 1 {
		m = m.Where(cols.Status, in.Status)
	}
	if err = m.OrderAsc(cols.Id).Scan(&list); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询系统参数失败")
	}
	return list, nil
}

// Get 按 key 取值(取不到时返回内置默认值)
func (s *Service) Get(ctx context.Context, key string) (string, error) {
	if _, err := requireLogin(ctx); err != nil {
		return "", err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", gerror.NewCode(consts.CodeBadRequest, "参数键不能为空")
	}
	return s.GetString(ctx, key, ""), nil
}

// Batch 批量取多个参数(前端一次拿多个常用参数), 未知 key 返回空串
func (s *Service) Batch(ctx context.Context, keys []string) (map[string]string, error) {
	if _, err := requireLogin(ctx); err != nil {
		return nil, err
	}
	result := make(map[string]string, len(keys))
	seen := make(map[string]struct{}, len(keys))
	count := 0
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		count++
		if count > maxBatchKeys {
			return nil, gerror.NewCode(consts.CodeBadRequest, fmt.Sprintf("单次最多查询 %d 个参数", maxBatchKeys))
		}
		result[key] = s.GetString(ctx, key, "")
	}
	return result, nil
}

// ------------------------------------------------------------
// 写入
// ------------------------------------------------------------

// Update 修改参数值(仅超管), 内置数值型参数校验范围
func (s *Service) Update(ctx context.Context, key, value string) (err error) {
	if _, err = requireWrite(ctx); err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return gerror.NewCode(consts.CodeBadRequest, "参数键不能为空")
	}
	value = strings.TrimSpace(value)
	if len(value) > 65535 {
		return gerror.NewCode(consts.CodeBadRequest, "参数值过长")
	}
	target, err := s.getByKey(ctx, key)
	if err != nil {
		return err
	}
	if err = validateValue(target, value); err != nil {
		return err
	}
	if _, err = dao.SysConfig.Ctx(ctx).Where(dao.SysConfig.Columns().Id, target.Id).Data(do.SysConfig{
		ConfigValue: value,
	}).Update(); err != nil {
		return gerror.Wrap(err, "修改系统参数失败")
	}
	ClearCache()
	return nil
}

// Reset 重置内置参数为默认值(仅超管, 仅内置项支持)
func (s *Service) Reset(ctx context.Context, key string) (value string, err error) {
	if _, err = requireWrite(ctx); err != nil {
		return "", err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", gerror.NewCode(consts.CodeBadRequest, "参数键不能为空")
	}
	item, ok := builtinItem(key)
	if !ok {
		return "", gerror.NewCode(consts.CodeBadRequest, "仅内置参数支持重置为默认值")
	}
	target, err := s.getByKey(ctx, key)
	if err != nil {
		return "", err
	}
	if _, err = dao.SysConfig.Ctx(ctx).Where(dao.SysConfig.Columns().Id, target.Id).Data(do.SysConfig{
		ConfigValue: item.Value,
	}).Update(); err != nil {
		return "", gerror.Wrap(err, "重置系统参数失败")
	}
	ClearCache()
	return item.Value, nil
}

// getByKey 按 key 查询参数(含软删过滤)
func (s *Service) getByKey(ctx context.Context, key string) (*entity.SysConfig, error) {
	var target entity.SysConfig
	cols := dao.SysConfig.Columns()
	if err := dao.SysConfig.Ctx(ctx).Where(cols.ConfigKey, key).Scan(&target); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询系统参数失败")
		}
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "系统参数不存在")
	}
	return &target, nil
}

// ------------------------------------------------------------
// 内置参数工具
// ------------------------------------------------------------

// Defaults 返回内置参数清单(种子与重置使用)
func Defaults() []DefaultItem {
	return builtinDefaults
}

// DefaultValue 查询内置参数默认值
func DefaultValue(key string) (string, bool) {
	item, ok := builtinItem(key)
	if !ok {
		return "", false
	}
	return item.Value, true
}

func builtinItem(key string) (DefaultItem, bool) {
	for _, item := range builtinDefaults {
		if item.Key == key {
			return item, true
		}
	}
	return DefaultItem{}, false
}

// validateValue 校验参数值: 内置数值型参数必须为整数且落在允许范围
func validateValue(target *entity.SysConfig, value string) error {
	item, ok := builtinItem(target.ConfigKey)
	if !ok || !item.Numeric {
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return gerror.NewCode(consts.CodeBadRequest, "参数值必须为整数")
	}
	if item.Min > 0 && n < item.Min {
		return gerror.NewCode(consts.CodeBadRequest, fmt.Sprintf("参数值不能小于 %d", item.Min))
	}
	if item.Max > 0 && n > item.Max {
		return gerror.NewCode(consts.CodeBadRequest, fmt.Sprintf("参数值不能大于 %d", item.Max))
	}
	return nil
}

// ------------------------------------------------------------
// 种子数据(幂等)
// ------------------------------------------------------------

// Seed 写入内置参数, 已存在的跳过(不覆盖运行时已修改的值)
func (s *Service) Seed(ctx context.Context) (err error) {
	cols := dao.SysConfig.Columns()
	for _, item := range builtinDefaults {
		count, err := dao.SysConfig.Ctx(ctx).Where(cols.ConfigKey, item.Key).Count()
		if err != nil {
			return gerror.Wrap(err, "初始化系统参数失败")
		}
		if count > 0 {
			continue
		}
		if _, err = dao.SysConfig.Ctx(ctx).Data(do.SysConfig{
			ConfigKey:   item.Key,
			ConfigValue: item.Value,
			Name:        item.Name,
			Remark:      item.Remark,
			Status:      1,
			Builtin:     1,
		}).Insert(); err != nil {
			return gerror.Wrap(err, "初始化系统参数失败")
		}
	}
	ClearCache()
	return nil
}

// IntOrDefault 便捷读取: 带 ctx 的 int 参数(供 service 内部使用)
func IntOrDefault(ctx context.Context, key string, def int) int {
	return New().GetInt(ctx, key, def)
}

// StringOrDefault 便捷读取: 带 ctx 的字符串参数
func StringOrDefault(ctx context.Context, key, def string) string {
	return New().GetString(ctx, key, def)
}
