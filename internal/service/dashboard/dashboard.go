// Package dashboard 工作台(首页)统计: 卡片 + 登录趋势 + 最近动态
//
// 数据权限: 统计全部复用现有数据范围规则
//   - 用户总数: 与用户列表同一套规则(超管全量 / 管理员仅子孙 / 普通用户无权查看工作台)
//   - 登录/操作日志: 与日志列表同一套规则(logsvc.VisibleUserIds: 超管全量 / 管理员自身+子孙)
//   - 在线用户: 与在线用户列表同一套规则
package dashboard

import (
	"context"
	"fmt"
	"time"

	"demo/internal/dao"
	"demo/internal/model/entity"
	"demo/internal/service/authz"
	"demo/internal/service/identity"
	logsvc "demo/internal/service/log"
	noticesvc "demo/internal/service/notice"
	onlinesvc "demo/internal/service/online"
	sysconfigsvc "demo/internal/service/sysconfig"
	usersvc "demo/internal/service/user"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

const (
	defaultTrendDays   = 7
	defaultRecentLimit = 8
	maxTrendDays       = 90
	maxRecentLimit     = 50
)

// TrendPoint 单日登录趋势
type TrendPoint struct {
	Date    string `json:"date"`
	Success int    `json:"success"`
	Failed  int    `json:"failed"`
}

// RecentItem 最近操作动态
type RecentItem struct {
	Id        uint64 `json:"id"`
	Username  string `json:"username"`
	Module    string `json:"module"`
	Action    string `json:"action"`
	Status    int    `json:"status"`
	Ip        string `json:"ip"`
	CreatedAt string `json:"createdAt"`
}

// NoticeBrief 工作台最近公告
type NoticeBrief struct {
	Id        uint64 `json:"id"`
	Title     string `json:"title"`
	Level     string `json:"level"`
	PublishAt string `json:"publishAt"`
	IsRead    bool   `json:"isRead"`
}

// Overview 工作台概览
type Overview struct {
	Scope             string        `json:"scope"` // all=超管全量, descendants=管理员可见范围
	UserTotal         int           `json:"userTotal"`
	OnlineCount       int           `json:"onlineCount"`
	LoginSuccessToday int           `json:"loginSuccessToday"`
	LoginFailedToday  int           `json:"loginFailedToday"`
	TrendDays         int           `json:"trendDays"`
	Trend             []TrendPoint  `json:"trend"`
	Recent            []RecentItem  `json:"recent"`
	UnreadNoticeCount int           `json:"unreadNoticeCount"`
	RecentNotices     []NoticeBrief `json:"recentNotices"`
}

// Service 工作台服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// requireManager 工作台管理统计仅超管/管理员可见; 普通用户不展示管理卡片
func requireManager(ctx context.Context) (*identity.Identity, error) {
	return authz.Manager(ctx, "无权查看工作台统计数据")
}

// trendDays 趋势天数: sys_config 优先, config.yaml 兜底, 限制 1-90
func trendDays(ctx context.Context) int {
	n := sysconfigsvc.New().GetInt(ctx, sysconfigsvc.KeyDashboardTrendDays,
		g.Cfg().MustGet(ctx, "dashboard.trendDays", defaultTrendDays).Int())
	if n < 1 {
		n = defaultTrendDays
	}
	if n > maxTrendDays {
		n = maxTrendDays
	}
	return n
}

// recentLimit 最近动态条数: sys_config 优先, config.yaml 兜底, 限制 1-50
func recentLimit(ctx context.Context) int {
	n := sysconfigsvc.New().GetInt(ctx, sysconfigsvc.KeyDashboardRecentLimit,
		g.Cfg().MustGet(ctx, "dashboard.recentLimit", defaultRecentLimit).Int())
	if n < 1 {
		n = defaultRecentLimit
	}
	if n > maxRecentLimit {
		n = maxRecentLimit
	}
	return n
}

// Overview 汇总概览: 用户/在线卡片 + 今日登录成功失败 + 近 N 天趋势 + 最近动态
// 全部使用聚合查询, 不在循环内查库
func (s *Service) Overview(ctx context.Context) (out *Overview, err error) {
	actor, err := requireManager(ctx)
	if err != nil {
		return nil, err
	}
	// 用户总数: 与用户列表完全一致的数据范围(管理员仅子孙, 不含自己)
	_, _, _, userTotal, err := usersvc.New().List(ctx, usersvc.ListInput{Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	// 在线用户: 复用在线用户列表的数据范围
	online, err := onlinesvc.List(ctx, &onlinesvc.ListInput{Page: 1, PageSize: 1})
	if err != nil {
		return nil, gerror.Wrap(err, "统计在线用户失败")
	}

	now := gtime.Now()
	todayStart := gtime.New(now.Format("Y-m-d") + " 00:00:00")
	days := trendDays(ctx)
	startDay := todayStart.AddDate(0, 0, -(days - 1))

	success, failed, err := todayLoginCounts(ctx, todayStart)
	if err != nil {
		return nil, err
	}
	trend, err := loginTrend(ctx, startDay, days)
	if err != nil {
		return nil, err
	}
	recent, err := recentOperations(ctx, recentLimit(ctx))
	if err != nil {
		return nil, err
	}
	// 通知公告: 未读数与最近 3 条, 内部复用 notice.visibleQuery 保证可见范围一致
	unreadNoticeCount, err := noticesvc.New().UnreadCount(ctx)
	if err != nil {
		return nil, err
	}
	recentNotices, err := noticesvc.New().RecentNotices(ctx, 3)
	if err != nil {
		return nil, err
	}

	scope := "descendants"
	if actor.IsSuperAdmin() {
		scope = "all"
	}
	return &Overview{
		Scope:             scope,
		UserTotal:         userTotal,
		OnlineCount:       online.Total,
		LoginSuccessToday: success,
		LoginFailedToday:  failed,
		TrendDays:         days,
		Trend:             trend,
		Recent:            recent,
		UnreadNoticeCount: unreadNoticeCount,
		RecentNotices:     toNoticeBriefs(recentNotices),
	}, nil
}

// toNoticeBriefs 转换最近公告为工作台展示结构
func toNoticeBriefs(items []*noticesvc.MyItem) []NoticeBrief {
	briefs := make([]NoticeBrief, 0, len(items))
	for _, item := range items {
		brief := NoticeBrief{Id: item.Id, Title: item.Title, Level: item.NoticeLevel, IsRead: item.IsRead}
		if item.PublishAt != nil {
			brief.PublishAt = item.PublishAt.Format("Y-m-d H:i:s")
		}
		briefs = append(briefs, brief)
	}
	return briefs
}

// todayLoginCounts 今日登录成功/失败数(一次分组聚合)
func todayLoginCounts(ctx context.Context, start *gtime.Time) (success, failed int, err error) {
	all, ids, err := logsvc.VisibleUserIds(ctx)
	if err != nil {
		return 0, 0, err
	}
	if !all && len(ids) == 0 {
		return 0, 0, nil
	}
	cols := dao.SysLoginLog.Columns()
	where := "`" + cols.CreatedAt + "` >= ?"
	args := []interface{}{start}
	if !all {
		where += " AND `" + cols.UserId + "` IN (?)"
		args = append(args, ids)
	}
	sqlStr := fmt.Sprintf("SELECT `%s` AS st, COUNT(*) AS cnt FROM `%s` WHERE %s GROUP BY `%s`",
		cols.Status, dao.SysLoginLog.Table(), where, cols.Status)
	rows, err := g.DB().GetAll(ctx, sqlStr, args...)
	if err != nil {
		return 0, 0, gerror.Wrap(err, "统计今日登录失败")
	}
	for _, row := range rows {
		cnt := row["cnt"].Int()
		if row["st"].Int() == 1 {
			success = cnt
		} else {
			failed += cnt
		}
	}
	return success, failed, nil
}

// loginTrend 近 N 天登录趋势(一次分组聚合, 缺失日期补零)
func loginTrend(ctx context.Context, start *gtime.Time, days int) ([]TrendPoint, error) {
	all, ids, err := logsvc.VisibleUserIds(ctx)
	if err != nil {
		return nil, err
	}
	points := make([]TrendPoint, 0, days)
	index := make(map[string]int, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("Y-m-d")
		index[date] = len(points)
		points = append(points, TrendPoint{Date: date})
	}
	if !all && len(ids) == 0 {
		return points, nil
	}
	cols := dao.SysLoginLog.Columns()
	where := "`" + cols.CreatedAt + "` >= ?"
	args := []interface{}{zoneOffsetMinutes(), start}
	if !all {
		where += " AND `" + cols.UserId + "` IN (?)"
		args = append(args, ids)
	}
	// 数据库连接以 UTC 存储 datetime, 这里按本地时区偏移换算成"本地自然日"再分组,
	// 保证趋势日期与前端展示的本地日期一致
	sqlStr := fmt.Sprintf(
		"SELECT DATE_FORMAT(DATE_ADD(`%s`, INTERVAL ? MINUTE), '%%Y-%%m-%%d') AS day, `%s` AS st, COUNT(*) AS cnt "+
			"FROM `%s` WHERE %s GROUP BY day, st",
		cols.CreatedAt, cols.Status, dao.SysLoginLog.Table(), where)
	rows, err := g.DB().GetAll(ctx, sqlStr, args...)
	if err != nil {
		return nil, gerror.Wrap(err, "统计登录趋势失败")
	}
	for _, row := range rows {
		idx, ok := index[row["day"].String()]
		if !ok {
			continue
		}
		cnt := row["cnt"].Int()
		if row["st"].Int() == 1 {
			points[idx].Success = cnt
		} else {
			points[idx].Failed = cnt
		}
	}
	// 日期按 start 递增生成, 与分组结果按 day 对齐
	return points, nil
}

// zoneOffsetMinutes 本地时区相对 UTC 的偏移(分钟), 用于把 DB 中 UTC 的 datetime 换算成本地自然日
func zoneOffsetMinutes() int {
	_, offset := time.Now().Zone()
	return offset / 60
}

// recentOperations 最近操作动态(单次查询, 复用日志模块可见范围)
func recentOperations(ctx context.Context, limit int) ([]RecentItem, error) {
	all, ids, err := logsvc.VisibleUserIds(ctx)
	if err != nil {
		return nil, err
	}
	if !all && len(ids) == 0 {
		return []RecentItem{}, nil
	}
	cols := dao.SysOperationLog.Columns()
	m := dao.SysOperationLog.Ctx(ctx)
	if !all {
		m = m.WhereIn(cols.UserId, ids)
	}
	var list []*entity.SysOperationLog
	if err = m.OrderDesc(cols.Id).Limit(limit).Scan(&list); err != nil {
		return nil, gerror.Wrap(err, "查询最近动态失败")
	}
	items := make([]RecentItem, 0, len(list))
	for _, row := range list {
		item := RecentItem{
			Id: row.Id, Username: row.Username, Module: row.Module,
			Action: row.Action, Status: row.Status, Ip: row.Ip,
		}
		if row.CreatedAt != nil {
			item.CreatedAt = row.CreatedAt.Format("Y-m-d H:i:s")
		}
		items = append(items, item)
	}
	return items, nil
}
