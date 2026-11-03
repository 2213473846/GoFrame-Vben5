// Package log 日志服务: 登录日志/操作日志的记录与查询
// 记录为异步写入, 不阻塞主请求; 解析失败不影响业务
package log

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	dictsvc "demo/internal/service/dict"
	exportsvc "demo/internal/service/export"
	"demo/internal/service/identity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

// Service 日志服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// ============================================================
// UserAgent 解析
// ============================================================

// parseUserAgent 解析 UA 得到 (浏览器, 操作系统), 不引第三方依赖
func parseUserAgent(ua string) (browser string, os string) {
	browser, os = "未知", "未知"
	if ua == "" {
		return
	}
	switch {
	case strings.Contains(ua, "Edg/"), strings.Contains(ua, "Edge/"):
		browser = "Edge"
	case strings.Contains(ua, "OPR/"), strings.Contains(ua, "Opera"):
		browser = "Opera"
	case strings.Contains(ua, "Chrome/"):
		// Chrome 需放在 Safari 前判断(Chrome UA 同时含 Safari 标识)
		browser = "Chrome"
	case strings.Contains(ua, "Firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "Safari/"):
		browser = "Safari"
	case strings.Contains(ua, "MSIE "), strings.Contains(ua, "Trident/"):
		browser = "IE"
	case strings.Contains(ua, "MicroMessenger"):
		browser = "WeChat"
	case strings.Contains(ua, "WindowsPowerShell"), strings.Contains(ua, "pwsh/"):
		browser = "PowerShell"
	case strings.Contains(ua, "curl/"):
		browser = "curl"
	case strings.Contains(ua, "PostmanRuntime"):
		browser = "Postman"
	}
	if v := browserVersion(ua, browser); v != "" {
		browser = fmt.Sprintf("%s %s", browser, v)
	}
	switch {
	case strings.Contains(ua, "Windows NT 10.0"), strings.Contains(ua, "Windows NT 11.0"):
		os = "Windows 10+"
	case strings.Contains(ua, "Windows NT 6.3"):
		os = "Windows 8.1"
	case strings.Contains(ua, "Windows NT 6.1"):
		os = "Windows 7"
	case strings.Contains(ua, "Windows"):
		os = "Windows"
	case strings.Contains(ua, "Mac OS X"):
		os = "macOS"
	case strings.Contains(ua, "Android"):
		os = "Android"
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"), strings.Contains(ua, "iOS"):
		os = "iOS"
	case strings.Contains(ua, "Linux"):
		os = "Linux"
	}
	return
}

// browserVersion 提取浏览器主版本号
func browserVersion(ua string, browser string) string {
	markers := map[string]string{
		"Edge": "Edg/", "Chrome": "Chrome/", "Firefox": "Firefox/",
		"Opera": "OPR/", "Safari": "Version/", "IE": "MSIE ",
	}
	marker, ok := markers[browser]
	if !ok {
		return ""
	}
	idx := strings.Index(ua, marker)
	if idx < 0 {
		return ""
	}
	rest := ua[idx+len(marker):]
	end := strings.IndexAny(rest, " );")
	if end < 0 {
		end = len(rest)
	}
	ver := rest[:end]
	// 只保留主版本
	if dot := strings.Index(ver, "."); dot > 0 {
		ver = ver[:dot]
	}
	return ver
}

func parseBrowser(ua string) string {
	b, _ := parseUserAgent(ua)
	return b
}

func parseOs(ua string) string {
	_, o := parseUserAgent(ua)
	return o
}

// ============================================================
// IP 归属地解析
// ============================================================

var (
	locationCache   = sync.Map{} // ip => 归属地字符串
	locationTimeout = 2 * time.Second
)

// isPrivateIP 判断是否内网 IP
func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast()
}

// resolveLocation 解析 IP 归属地: 内网直接返回, 公网走在线接口(带缓存, 失败返回"未知")
func resolveLocation(ctx context.Context, ip string) string {
	if ip == "" {
		return "未知"
	}
	if isPrivateIP(ip) {
		return "内网"
	}
	if cached, ok := locationCache.Load(ip); ok {
		return gconv.String(cached)
	}
	location := "未知"
	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN&fields=status,country,regionName,city", ip)
	if resp, err := g.Client().Timeout(locationTimeout).Get(ctx, url); err == nil {
		defer resp.Close()
		var data struct {
			Status     string `json:"status"`
			Country    string `json:"country"`
			RegionName string `json:"regionName"`
			City       string `json:"city"`
		}
		if err = json.Unmarshal(resp.ReadAll(), &data); err == nil && data.Status == "success" {
			parts := make([]string, 0, 3)
			for _, part := range []string{data.Country, data.RegionName, data.City} {
				if part != "" {
					parts = append(parts, part)
				}
			}
			if len(parts) > 0 {
				location = strings.Join(parts, " ")
			}
		}
	}
	locationCache.Store(ip, location)
	return location
}

// ============================================================
// 记录埋点(异步)
// ============================================================

// RecordLoginFromRequest 从 HTTP 请求记录登录日志(异步)
func RecordLoginFromRequest(r *ghttp.Request, userId uint64, username string, success bool, message string) {
	status := 1
	if !success {
		status = 0
	}
	ip := r.GetClientIp()
	ua := r.Header.Get("User-Agent")
	doRecord(&do.SysLoginLog{
		UserId:   userId,
		Username: username,
		Ip:       ip,
		Browser:  parseBrowser(ua),
		Os:       parseOs(ua),
		Status:   status,
		Message:  message,
	})
}

// RecordOperationFromRequest 从 HTTP 请求记录操作日志(异步)
// 未认证请求(identity 为空, 如 401)不记录
func RecordOperationFromRequest(r *ghttp.Request, module string, action string, success bool, message string) {
	id := identity.Get(r.Context())
	if id == nil {
		return
	}
	status := 1
	if !success {
		status = 0
	}
	ip := r.GetClientIp()
	ua := r.Header.Get("User-Agent")
	doRecord(&do.SysOperationLog{
		UserId:   id.Id,
		Username: id.Username,
		Module:   module,
		Action:   action,
		Method:   r.Method,
		Path:     r.URL.Path,
		Ip:       ip,
		Browser:  parseBrowser(ua),
		Os:       parseOs(ua),
		Status:   status,
		Message:  message,
	})
}

// doRecord 异步写库, 使用独立 context 避免随请求结束取消; 失败仅打印日志
func doRecord(row interface{}) {
	go func() {
		bgCtx := gctx.New()
		defer func() {
			if r := recover(); r != nil {
				g.Log().Errorf(bgCtx, "日志记录 panic: %v", r)
			}
		}()
		var err error
		switch data := row.(type) {
		case *do.SysLoginLog:
			data.Location = resolveLocation(bgCtx, gconv.String(data.Ip))
			_, err = dao.SysLoginLog.Ctx(bgCtx).Data(data).Insert()
		case *do.SysOperationLog:
			data.Location = resolveLocation(bgCtx, gconv.String(data.Ip))
			_, err = dao.SysOperationLog.Ctx(bgCtx).Data(data).Insert()
		}
		if err != nil {
			g.Log().Errorf(bgCtx, "日志记录失败: %v", err)
		}
	}()
}

// ============================================================
// 查询(数据范围: 超管全部, 管理员自己+子孙, 普通用户仅自己)
// ============================================================

// LoginListInput 登录日志查询入参
type LoginListInput struct {
	Page     int
	PageSize int
	Username string
	Status   int // -1=全部 1=成功 0=失败
}

// LoginListOutput 登录日志查询结果
type LoginListOutput struct {
	Items []*entity.SysLoginLog `json:"items"`
	Total int                   `json:"total"`
}

// LoginList 登录日志分页列表
func (s *Service) LoginList(ctx context.Context, in *LoginListInput) (*LoginListOutput, error) {
	m, empty, err := s.loginQuery(ctx, in)
	cols := dao.SysLoginLog.Columns()
	if err != nil {
		return nil, err
	}
	if empty {
		return &LoginListOutput{Items: []*entity.SysLoginLog{}}, nil
	}
	total, err := m.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "查询登录日志失败")
	}
	var items []*entity.SysLoginLog
	if err = m.Page(in.Page, in.PageSize).OrderDesc(cols.Id).Scan(&items); err != nil {
		return nil, gerror.Wrap(err, "查询登录日志失败")
	}
	return &LoginListOutput{Items: items, Total: total}, nil
}

// loginQuery 构建登录日志查询(数据范围 + 筛选), 供列表与导出复用
func (s *Service) loginQuery(ctx context.Context, in *LoginListInput) (*gdb.Model, bool, error) {
	all, ids, err := visibleUserIds(ctx)
	if err != nil {
		return nil, false, err
	}
	if !all && len(ids) == 0 {
		return nil, true, nil
	}
	cols := dao.SysLoginLog.Columns()
	m := dao.SysLoginLog.Ctx(ctx)
	if !all {
		m = m.WhereIn(cols.UserId, ids)
	}
	if in.Username != "" {
		m = m.WhereLike(cols.Username, "%"+in.Username+"%")
	}
	if in.Status >= 0 {
		m = m.Where(cols.Status, in.Status)
	}
	return m, false, nil
}

// OperationListInput 操作日志查询入参
type OperationListInput struct {
	Page     int
	PageSize int
	Username string
	Status   int // -1=全部 1=成功 0=失败
}

// OperationListOutput 操作日志查询结果
type OperationListOutput struct {
	Items []*entity.SysOperationLog `json:"items"`
	Total int                       `json:"total"`
}

// OperationList 操作日志分页列表
func (s *Service) OperationList(ctx context.Context, in *OperationListInput) (*OperationListOutput, error) {
	m, empty, err := s.operationQuery(ctx, in)
	cols := dao.SysOperationLog.Columns()
	if err != nil {
		return nil, err
	}
	if empty {
		return &OperationListOutput{Items: []*entity.SysOperationLog{}}, nil
	}
	total, err := m.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "查询操作日志失败")
	}
	var items []*entity.SysOperationLog
	if err = m.Page(in.Page, in.PageSize).OrderDesc(cols.Id).Scan(&items); err != nil {
		return nil, gerror.Wrap(err, "查询操作日志失败")
	}
	return &OperationListOutput{Items: items, Total: total}, nil
}

// operationQuery 构建操作日志查询(数据范围 + 筛选), 供列表与导出复用
func (s *Service) operationQuery(ctx context.Context, in *OperationListInput) (*gdb.Model, bool, error) {
	all, ids, err := visibleUserIds(ctx)
	if err != nil {
		return nil, false, err
	}
	if !all && len(ids) == 0 {
		return nil, true, nil
	}
	cols := dao.SysOperationLog.Columns()
	m := dao.SysOperationLog.Ctx(ctx)
	if !all {
		m = m.WhereIn(cols.UserId, ids)
	}
	if in.Username != "" {
		m = m.WhereLike(cols.Username, "%"+in.Username+"%")
	}
	if in.Status >= 0 {
		m = m.Where(cols.Status, in.Status)
	}
	return m, false, nil
}

// LoginExport 导出登录日志, 复用 loginQuery 的筛选与数据范围
func (s *Service) LoginExport(ctx context.Context, in *LoginListInput, format string, w io.Writer) (count int, err error) {
	format, err = exportsvc.NormalizeFormat(format)
	if err != nil {
		return 0, gerror.NewCode(consts.CodeBadRequest, err.Error())
	}
	m, empty, err := s.loginQuery(ctx, in)
	if err != nil {
		return 0, err
	}
	cols := dao.SysLoginLog.Columns()
	statusLabels := statusLabelMap(ctx)
	headers := []string{"ID", "用户名", "状态", "信息", "IP", "归属地", "浏览器", "操作系统", "登录时间"}
	next := func(offset, limit int) ([][]string, error) {
		if empty {
			return nil, nil
		}
		var items []*entity.SysLoginLog
		if scanErr := m.Limit(offset, limit).OrderDesc(cols.Id).Scan(&items); scanErr != nil {
			if errors.Is(scanErr, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, gerror.Wrap(scanErr, "查询导出数据失败")
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				gconv.String(item.Id),
				item.Username,
				statusLabels[item.Status],
				item.Message,
				item.Ip,
				item.Location,
				item.Browser,
				item.Os,
				formatExportTime(item.CreatedAt),
			})
		}
		return rows, nil
	}
	return exportsvc.Write(w, format, exportsvc.Options{Sheet: "登录日志", Headers: headers}, exportsvc.BatchSize(ctx), next)
}

// OperationExport 导出操作日志, 复用 operationQuery 的筛选与数据范围
func (s *Service) OperationExport(ctx context.Context, in *OperationListInput, format string, w io.Writer) (count int, err error) {
	format, err = exportsvc.NormalizeFormat(format)
	if err != nil {
		return 0, gerror.NewCode(consts.CodeBadRequest, err.Error())
	}
	m, empty, err := s.operationQuery(ctx, in)
	if err != nil {
		return 0, err
	}
	cols := dao.SysOperationLog.Columns()
	statusLabels := statusLabelMap(ctx)
	headers := []string{"ID", "用户名", "模块", "操作", "方法", "路径", "状态", "信息", "IP", "归属地", "浏览器", "操作系统", "操作时间"}
	next := func(offset, limit int) ([][]string, error) {
		if empty {
			return nil, nil
		}
		var items []*entity.SysOperationLog
		if scanErr := m.Limit(offset, limit).OrderDesc(cols.Id).Scan(&items); scanErr != nil {
			if errors.Is(scanErr, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, gerror.Wrap(scanErr, "查询导出数据失败")
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				gconv.String(item.Id),
				item.Username,
				item.Module,
				item.Action,
				item.Method,
				item.Path,
				statusLabels[item.Status],
				item.Message,
				item.Ip,
				item.Location,
				item.Browser,
				item.Os,
				formatExportTime(item.CreatedAt),
			})
		}
		return rows, nil
	}
	return exportsvc.Write(w, format, exportsvc.Options{Sheet: "操作日志", Headers: headers}, exportsvc.BatchSize(ctx), next)
}

// statusLabelMap 日志状态文案: 优先字典, 取不到回退内置默认值
func statusLabelMap(ctx context.Context) map[int]string {
	labels := map[int]string{1: "成功", 0: "失败"}
	if options, err := dictsvc.New().Options(ctx, []string{dictsvc.TypeLogType}); err == nil {
		for _, item := range options[dictsvc.TypeLogType] {
			labels[gconv.Int(item.Value)] = item.Label
		}
	}
	return labels
}

func formatExportTime(t *gtime.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("Y-m-d H:i:s")
}

// visibleUserIds 当前用户可见的日志数据范围
// all=true 表示超管可见全部(不过滤); 否则 ids 为可见用户ID集合
// 管理员=自己+子孙, 普通用户=仅自己
func visibleUserIds(ctx context.Context) (all bool, ids []uint64, err error) {
	id := identity.Get(ctx)
	switch id.Role {
	case consts.RoleSuperAdmin:
		return true, nil, nil
	case consts.RoleAdmin:
		ids, err = descendantUserIds(ctx, id.Id)
		if err != nil {
			return false, nil, err
		}
		return false, append(ids, id.Id), nil
	default:
		return false, []uint64{id.Id}, nil
	}
}

// descendantUserIds 层级查询子孙用户ID(不含自身, 软删用户已被 ORM 自动过滤)
func descendantUserIds(ctx context.Context, rootId uint64) ([]uint64, error) {
	cols := dao.SysUser.Columns()
	level := []uint64{rootId}
	result := make([]uint64, 0)
	for i := 0; i < 50 && len(level) > 0; i++ {
		var records []*entity.SysUser
		if err := dao.SysUser.Ctx(ctx).WhereIn(cols.Pid, level).Fields(cols.Id).Scan(&records); err != nil {
			return nil, gerror.Wrap(err, "查询下级用户失败")
		}
		if len(records) == 0 {
			break
		}
		level = make([]uint64, 0, len(records))
		for _, r := range records {
			level = append(level, r.Id)
			result = append(result, r.Id)
		}
	}
	return result, nil
}

// ============================================================
// 导出封装(供在线用户等其他模块复用)
// ============================================================

// VisibleUserIds 当前用户可见数据范围(同 visibleUserIds)
func VisibleUserIds(ctx context.Context) (all bool, ids []uint64, err error) {
	return visibleUserIds(ctx)
}

// DescendantUserIds 层级查询子孙用户ID(不含自身)
func DescendantUserIds(ctx context.Context, rootId uint64) ([]uint64, error) {
	return descendantUserIds(ctx, rootId)
}

// ParseUserAgent 解析 UA 得到 (浏览器, 操作系统)
func ParseUserAgent(ua string) (browser string, os string) {
	return parseUserAgent(ua)
}

// ResolveLocation 解析 IP 归属地(内网/缓存/在线接口)
func ResolveLocation(ctx context.Context, ip string) string {
	return resolveLocation(ctx, ip)
}
