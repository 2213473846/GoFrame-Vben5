// Package notice 通知公告: 管理端维护、用户端可见性查询、已读状态与工作台联动
//
// 可见性统一收口在 visibleQuery: /notice/my、/notice/unread-count、/notice/read(all)、
// RecentNotices 与工作台统计全部复用同一个查询, 禁止任何路径各自拼装 where 条件
// (历史上导出功能曾因多处拼装条件导致可见范围不一致)。
//
// XSS 处理: content 入库前统一剥离 HTML 标签, 只允许纯文本; 前端必须使用文本插值渲染,
// 禁止 v-html, 否则剥离后残留的文本仍可能被解析为 HTML。
//
// 范围权限: 管理员只能发布定向公告(不能发全员), 目标用户必须在其子孙范围内;
// 且管理员只能查询/修改/删除/发布/撤回自己创建的公告, 越权返回 403。
package notice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/authz"
	dictsvc "demo/internal/service/dict"
	"demo/internal/service/identity"
	logsvc "demo/internal/service/log"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

// 公告状态
const (
	StatusDraft     = 0 // 草稿
	StatusPublished = 1 // 已发布
	StatusRevoked   = 2 // 已撤回
)

// 可见范围
const (
	ScopeAll      = 1 // 全员可见
	ScopeTargeted = 2 // 仅指定用户可见
)

// 校验上限与默认值
const (
	MaxTitleLength        = 128
	MaxContentLength      = 20000
	DefaultPollSeconds    = 60
	recentNoticeLimit     = 3
	readAllBatchSize      = 500
	defaultNoticePageSize = 20
)

// htmlTagPattern 匹配任意 HTML 标签, 用于把富文本降级为纯文本
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// Service 通知公告服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// ------------------------------------------------------------
// 入参/出参
// ------------------------------------------------------------

// ListInput 管理端列表查询条件
type ListInput struct {
	Keyword    string // 标题模糊匹配
	NoticeType string // 类型(字典 notice_type)
	Level      string // 级别(字典 notice_level)
	Status     int    // -1=全部 0=草稿 1=已发布 2=已撤回
	StartTime  string // publish_at 起(含)
	EndTime    string // publish_at 止(含)
	Page       int
	PageSize   int
}

// ListOutput 管理端列表结果
type ListOutput struct {
	Items []*entity.SysNotice
	Total int
}

// CreateInput 新增/修改公告输入
type CreateInput struct {
	Title         string
	Content       string
	NoticeType    string
	NoticeLevel   string
	TargetScope   int
	TargetUserIds []uint64
	PublishAt     string // 可选, 空表示发布时取当前时间
	ExpireAt      string // 可选, 空表示永不过期
}

// UpdateInput 修改公告输入
type UpdateInput struct {
	Id uint64
	CreateInput
}

// MyListInput 用户端公告列表查询条件
type MyListInput struct {
	Page     int
	PageSize int
}

// MyItem 用户端公告条目(含当前用户已读状态)
type MyItem struct {
	Id          uint64
	Title       string
	Content     string
	NoticeType  string
	NoticeLevel string
	PublishAt   *gtime.Time
	CreatedAt   *gtime.Time
	IsRead      bool
}

// MyListOutput 用户端公告列表结果
type MyListOutput struct {
	Items []*MyItem
	Total int
}

// ReadInput 标记已读输入: Id 指定单条, All=true 表示全部已读
type ReadInput struct {
	Id  uint64
	All bool
}

// ------------------------------------------------------------
// 统一可见性查询
// ------------------------------------------------------------

// visibleQuery 构造"当前用户可见公告"查询, 所有用户端查询与工作台统计必须复用它。
// 条件: status=1 + publish_at<=now + (expire_at 为空或 >=now)
//   - (target_scope=1 全员可见 或 target_scope=2 且 sys_notice_user 中存在该用户)
//
// 软删字段由 ORM 对 sys_notice 表自动过滤, 不允许在此处手写 deleted_at 条件。
func visibleQuery(ctx context.Context, userId uint64) *gdb.Model {
	cols := dao.SysNotice.Columns()
	noticeTable := dao.SysNotice.Table()
	userCols := dao.SysNoticeUser.Columns()
	now := gtime.Now()
	return dao.SysNotice.Ctx(ctx).
		Where(cols.Status, StatusPublished).
		WhereLTE(cols.PublishAt, now).
		Where(fmt.Sprintf("(`%s` IS NULL OR `%s` >= ?)", cols.ExpireAt, cols.ExpireAt), now).
		Where(fmt.Sprintf(
			"(`%s` = ? OR EXISTS (SELECT 1 FROM `%s` nu WHERE nu.`%s` = `%s`.`%s` AND nu.`%s` = ?))",
			cols.TargetScope, dao.SysNoticeUser.Table(), userCols.NoticeId, noticeTable, cols.Id, userCols.UserId,
		), ScopeAll, userId)
}

// ------------------------------------------------------------
// 权限
// ------------------------------------------------------------

// requireManager 管理端读/写权限: 超管/管理员
func requireManager(ctx context.Context) (*identity.Identity, error) {
	return authz.Manager(ctx, "无权管理通知公告")
}

// checkOwner 管理员只能操作自己创建的公告, 超管不限
func checkOwner(actor *identity.Identity, target *entity.SysNotice) error {
	if actor.IsSuperAdmin() {
		return nil
	}
	if target.PublisherId != actor.Id {
		return gerror.NewCode(consts.CodeForbidden, "无权操作他人创建的公告")
	}
	return nil
}

// ------------------------------------------------------------
// 管理端查询
// ------------------------------------------------------------

// List 管理端公告列表: 超管全量, 管理员仅自己创建的公告
func (s *Service) List(ctx context.Context, in ListInput) (*ListOutput, error) {
	actor, err := requireManager(ctx)
	if err != nil {
		return nil, err
	}
	cols := dao.SysNotice.Columns()
	m := dao.SysNotice.Ctx(ctx)
	if !actor.IsSuperAdmin() {
		m = m.Where(cols.PublisherId, actor.Id)
	}
	if kw := strings.TrimSpace(in.Keyword); kw != "" {
		m = m.WhereLike(cols.Title, "%"+kw+"%")
	}
	if v := strings.TrimSpace(in.NoticeType); v != "" {
		m = m.Where(cols.NoticeType, v)
	}
	if v := strings.TrimSpace(in.Level); v != "" {
		m = m.Where(cols.NoticeLevel, v)
	}
	if in.Status == StatusDraft || in.Status == StatusPublished || in.Status == StatusRevoked {
		m = m.Where(cols.Status, in.Status)
	}
	if start, err := parseOptionalTime(in.StartTime); err != nil {
		return nil, err
	} else if start != nil {
		m = m.WhereGTE(cols.PublishAt, start)
	}
	if end, err := parseOptionalTime(in.EndTime); err != nil {
		return nil, err
	} else if end != nil {
		m = m.WhereLTE(cols.PublishAt, end)
	}
	total, err := m.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "查询公告列表失败")
	}
	page, pageSize := normalizePage(in.Page, in.PageSize)
	var items []*entity.SysNotice
	if err = m.Page(page, pageSize).OrderDesc(cols.Id).Scan(&items); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询公告列表失败")
	}
	return &ListOutput{Items: items, Total: total}, nil
}

// TargetCounts 批量统计公告的定向用户数(列表展示用, 一次聚合查询避免 N+1)
func (s *Service) TargetCounts(ctx context.Context, noticeIds []uint64) (map[uint64]int, error) {
	out := make(map[uint64]int, len(noticeIds))
	if len(noticeIds) == 0 {
		return out, nil
	}
	var rows []struct {
		NoticeId uint64 `orm:"notice_id"`
		Cnt      int    `orm:"cnt"`
	}
	uc := dao.SysNoticeUser.Columns()
	err := dao.SysNoticeUser.Ctx(ctx).
		Fields(uc.NoticeId+", COUNT(*) AS cnt").
		WhereIn(uc.NoticeId, noticeIds).
		Group(uc.NoticeId).
		Scan(&rows)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "统计公告定向用户失败")
	}
	for _, r := range rows {
		out[r.NoticeId] = r.Cnt
	}
	return out, nil
}

// TargetUserIds 查询公告的定向用户ID(详情编辑回填用)
func (s *Service) TargetUserIds(ctx context.Context, noticeId uint64) ([]uint64, error) {
	uc := dao.SysNoticeUser.Columns()
	values, err := dao.SysNoticeUser.Ctx(ctx).
		Where(uc.NoticeId, noticeId).
		Array(uc.UserId)
	if err != nil {
		return nil, gerror.Wrap(err, "查询公告定向用户失败")
	}
	return gconv.Uint64s(values), nil
}

// Detail 公告详情(管理员仅限自己创建的公告)
func (s *Service) Detail(ctx context.Context, id uint64) (*entity.SysNotice, error) {
	actor, err := requireManager(ctx)
	if err != nil {
		return nil, err
	}
	target, err := s.getById(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = checkOwner(actor, target); err != nil {
		return nil, err
	}
	return target, nil
}

// ------------------------------------------------------------
// 管理端写入
// ------------------------------------------------------------

// Create 新增公告(默认草稿), 定向用户与公告在同一事务内写入
func (s *Service) Create(ctx context.Context, in CreateInput) (id uint64, err error) {
	actor, err := requireManager(ctx)
	if err != nil {
		return 0, err
	}
	prepared, err := s.prepare(ctx, actor, in)
	if err != nil {
		return 0, err
	}
	cols := dao.SysNotice.Columns()
	userCols := dao.SysNoticeUser.Columns()
	if err = dao.SysNotice.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		data := g.Map{
			cols.Title:       prepared.title,
			cols.Content:     prepared.content,
			cols.NoticeType:  prepared.noticeType,
			cols.NoticeLevel: prepared.noticeLevel,
			cols.Status:      StatusDraft,
			cols.TargetScope: prepared.targetScope,
			cols.PublisherId: actor.Id,
			cols.PublishAt:   timeValue(prepared.publishAt),
			cols.ExpireAt:    timeValue(prepared.expireAt),
		}
		lastId, insertErr := tx.Model(dao.SysNotice.Table()).Data(data).InsertAndGetId()
		if insertErr != nil {
			return insertErr
		}
		id = uint64(lastId)
		if prepared.targetScope != ScopeTargeted {
			return nil
		}
		records := make([]g.Map, 0, len(prepared.targetUserIds))
		for _, userId := range prepared.targetUserIds {
			records = append(records, g.Map{userCols.NoticeId: id, userCols.UserId: userId})
		}
		_, insertErr = tx.Model(dao.SysNoticeUser.Table()).Data(records).InsertIgnore()
		return insertErr
	}); err != nil {
		return 0, gerror.Wrap(err, "新增公告失败")
	}
	return id, nil
}

// Update 修改公告内容/范围/时间; 已发布公告允许纠错编辑, 通过操作日志追溯;
// 管理员不能把定向公告改成全员公告(由范围校验统一拦截)。
func (s *Service) Update(ctx context.Context, in UpdateInput) (err error) {
	actor, err := requireManager(ctx)
	if err != nil {
		return err
	}
	target, err := s.getById(ctx, in.Id)
	if err != nil {
		return err
	}
	if err = checkOwner(actor, target); err != nil {
		return err
	}
	prepared, err := s.prepare(ctx, actor, in.CreateInput)
	if err != nil {
		return err
	}
	// 已发布公告编辑内容时, 未传 publishAt 则保留原发布时间, 避免编辑后公告"消失"
	if target.Status == StatusPublished && prepared.publishAt == nil && target.PublishAt != nil {
		prepared.publishAt = target.PublishAt
	}
	if prepared.publishAt != nil && prepared.expireAt != nil && !prepared.expireAt.After(prepared.publishAt) {
		return gerror.NewCode(consts.CodeBadRequest, "失效时间必须晚于发布时间")
	}
	cols := dao.SysNotice.Columns()
	userCols := dao.SysNoticeUser.Columns()
	if err = dao.SysNotice.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(dao.SysNotice.Table()).Where(cols.Id, target.Id).Data(g.Map{
			cols.Title:       prepared.title,
			cols.Content:     prepared.content,
			cols.NoticeType:  prepared.noticeType,
			cols.NoticeLevel: prepared.noticeLevel,
			cols.TargetScope: prepared.targetScope,
			cols.PublishAt:   timeValue(prepared.publishAt),
			cols.ExpireAt:    timeValue(prepared.expireAt),
		}).Update(); err != nil {
			return err
		}
		// 定向关系整体覆盖: 先删旧关系再写新关系, 避免残留越权可见
		if _, err := tx.Model(dao.SysNoticeUser.Table()).Where(userCols.NoticeId, target.Id).Delete(); err != nil {
			return err
		}
		if prepared.targetScope != ScopeTargeted {
			return nil
		}
		records := make([]g.Map, 0, len(prepared.targetUserIds))
		for _, userId := range prepared.targetUserIds {
			records = append(records, g.Map{userCols.NoticeId: target.Id, userCols.UserId: userId})
		}
		_, err := tx.Model(dao.SysNoticeUser.Table()).Data(records).InsertIgnore()
		return err
	}); err != nil {
		return gerror.Wrap(err, "修改公告失败")
	}
	return nil
}

// Delete 删除公告: 事务内软删公告并物理清理定向用户/已读记录
func (s *Service) Delete(ctx context.Context, id uint64) (err error) {
	actor, err := requireManager(ctx)
	if err != nil {
		return err
	}
	target, err := s.getById(ctx, id)
	if err != nil {
		return err
	}
	if err = checkOwner(actor, target); err != nil {
		return err
	}
	cols := dao.SysNotice.Columns()
	userCols := dao.SysNoticeUser.Columns()
	readCols := dao.SysNoticeRead.Columns()
	if err = dao.SysNotice.Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		if _, err := tx.Model(dao.SysNotice.Table()).Where(cols.Id, target.Id).
			Data(g.Map{cols.DeletedAt: gtime.Now()}).Update(); err != nil {
			return err
		}
		if _, err := tx.Model(dao.SysNoticeUser.Table()).Where(userCols.NoticeId, target.Id).Delete(); err != nil {
			return err
		}
		_, err := tx.Model(dao.SysNoticeRead.Table()).Where(readCols.NoticeId, target.Id).Delete()
		return err
	}); err != nil {
		return gerror.Wrap(err, "删除公告失败")
	}
	return nil
}

// Publish 发布公告: 仅草稿/已撤回可发布; publish_at 为空取当前时间
func (s *Service) Publish(ctx context.Context, id uint64, publishAt string) (err error) {
	actor, err := requireManager(ctx)
	if err != nil {
		return err
	}
	target, err := s.getById(ctx, id)
	if err != nil {
		return err
	}
	if err = checkOwner(actor, target); err != nil {
		return err
	}
	if target.Status != StatusDraft && target.Status != StatusRevoked {
		return gerror.NewCode(consts.CodeBadRequest, "只能发布草稿或已撤回的公告")
	}
	publishTime, err := parseOptionalTime(publishAt)
	if err != nil {
		return err
	}
	if publishTime == nil {
		// 截断到秒: MySQL DATETIME 会对小数秒四舍五入, 直接写入当前时间可能"提前"1秒导致刚发布不可见
		publishTime = gtime.New(gtime.Now().Format("Y-m-d H:i:s"))
	}
	if target.ExpireAt != nil && !target.ExpireAt.After(publishTime) {
		return gerror.NewCode(consts.CodeBadRequest, "失效时间必须晚于发布时间")
	}
	cols := dao.SysNotice.Columns()
	if _, err = dao.SysNotice.Ctx(ctx).Where(cols.Id, target.Id).Data(g.Map{
		cols.Status:    StatusPublished,
		cols.PublishAt: publishTime,
	}).Update(); err != nil {
		return gerror.Wrap(err, "发布公告失败")
	}
	return nil
}

// Revoke 撤回公告: status 改为已撤回, 记录保留但用户端立即不可见
func (s *Service) Revoke(ctx context.Context, id uint64) (err error) {
	actor, err := requireManager(ctx)
	if err != nil {
		return err
	}
	target, err := s.getById(ctx, id)
	if err != nil {
		return err
	}
	if err = checkOwner(actor, target); err != nil {
		return err
	}
	if target.Status != StatusPublished {
		return gerror.NewCode(consts.CodeBadRequest, "仅已发布的公告可撤回")
	}
	cols := dao.SysNotice.Columns()
	if _, err = dao.SysNotice.Ctx(ctx).Where(cols.Id, target.Id).Data(g.Map{
		cols.Status: StatusRevoked,
	}).Update(); err != nil {
		return gerror.Wrap(err, "撤回公告失败")
	}
	return nil
}

// ------------------------------------------------------------
// 用户端
// ------------------------------------------------------------

// MyList 当前用户可见公告分页列表, 返回每条已读状态
func (s *Service) MyList(ctx context.Context, in MyListInput) (*MyListOutput, error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, err
	}
	cols := dao.SysNotice.Columns()
	m := visibleQuery(ctx, actor.Id)
	total, err := m.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "查询我的公告失败")
	}
	page, pageSize := normalizePage(in.Page, in.PageSize)
	var rows []*entity.SysNotice
	if err = m.Page(page, pageSize).OrderDesc(cols.PublishAt).OrderDesc(cols.Id).Scan(&rows); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询我的公告失败")
	}
	items, err := s.toMyItems(ctx, actor.Id, rows)
	if err != nil {
		return nil, err
	}
	return &MyListOutput{Items: items, Total: total}, nil
}

// UnreadCount 当前用户可见且未读的公告数(复用 visibleQuery)
func (s *Service) UnreadCount(ctx context.Context) (int, error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return 0, err
	}
	cols := dao.SysNotice.Columns()
	readCols := dao.SysNoticeRead.Columns()
	notRead := fmt.Sprintf(
		"NOT EXISTS (SELECT 1 FROM `%s` nr WHERE nr.`%s` = `%s`.`%s` AND nr.`%s` = ?)",
		dao.SysNoticeRead.Table(), readCols.NoticeId, dao.SysNotice.Table(), cols.Id, readCols.UserId,
	)
	count, err := visibleQuery(ctx, actor.Id).Where(notRead, actor.Id).Count()
	if err != nil {
		return 0, gerror.Wrap(err, "统计未读公告失败")
	}
	return count, nil
}

// RecentNotices 当前用户最近可见公告(工作台展示), 复用 visibleQuery
func (s *Service) RecentNotices(ctx context.Context, limit int) ([]*MyItem, error) {
	actor, err := authz.Login(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 10 {
		limit = recentNoticeLimit
	}
	cols := dao.SysNotice.Columns()
	var rows []*entity.SysNotice
	if err = visibleQuery(ctx, actor.Id).
		OrderDesc(cols.PublishAt).OrderDesc(cols.Id).
		Limit(limit).Scan(&rows); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询最近公告失败")
	}
	return s.toMyItems(ctx, actor.Id, rows)
}

// Read 标记已读: 单条必须落在可见范围内, 全部已读只对当前可见公告写入
// 幂等实现: INSERT IGNORE, 依赖唯一键 uk_notice_user, 重复调用不报错
func (s *Service) Read(ctx context.Context, in ReadInput) error {
	actor, err := authz.Login(ctx)
	if err != nil {
		return err
	}
	if in.All {
		return s.readAllVisible(ctx, actor.Id)
	}
	if in.Id == 0 {
		return gerror.NewCode(consts.CodeBadRequest, "请指定公告或选择全部已读")
	}
	cols := dao.SysNotice.Columns()
	visible, err := visibleQuery(ctx, actor.Id).Where(cols.Id, in.Id).Count()
	if err != nil {
		return gerror.Wrap(err, "校验公告可见性失败")
	}
	if visible == 0 {
		return gerror.NewCode(consts.CodeBadRequest, "公告不存在或不在可见范围内")
	}
	if _, err = dao.SysNoticeRead.Ctx(ctx).Data(do.SysNoticeRead{
		NoticeId: in.Id, UserId: actor.Id, ReadAt: gtime.Now(),
	}).InsertIgnore(); err != nil {
		return gerror.Wrap(err, "标记已读失败")
	}
	return nil
}

// readAllVisible 全部已读: 先取当前可见公告ID, 再批量 INSERT IGNORE
func (s *Service) readAllVisible(ctx context.Context, userId uint64) error {
	cols := dao.SysNotice.Columns()
	var rows []struct {
		Id uint64 `orm:"id"`
	}
	if err := visibleQuery(ctx, userId).Fields(cols.Id).Scan(&rows); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return gerror.Wrap(err, "查询可见公告失败")
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Id)
	}
	for start := 0; start < len(ids); start += readAllBatchSize {
		end := start + readAllBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		records := make([]do.SysNoticeRead, 0, end-start)
		now := gtime.Now()
		for _, id := range ids[start:end] {
			records = append(records, do.SysNoticeRead{NoticeId: id, UserId: userId, ReadAt: now})
		}
		if _, err := dao.SysNoticeRead.Ctx(ctx).Data(records).InsertIgnore(); err != nil {
			return gerror.Wrap(err, "全部已读失败")
		}
	}
	return nil
}

// PollSeconds 用户端未读数轮询间隔(秒), 读取 config notice.pollSeconds, 默认 60
func PollSeconds(ctx context.Context) int {
	n := g.Cfg().MustGet(ctx, "notice.pollSeconds", DefaultPollSeconds).Int()
	if n <= 0 {
		n = DefaultPollSeconds
	}
	return n
}

// ------------------------------------------------------------
// 内部工具
// ------------------------------------------------------------

// preparedData 校验/清洗后的公告数据
type preparedData struct {
	title         string
	content       string
	noticeType    string
	noticeLevel   string
	targetScope   int
	targetUserIds []uint64
	publishAt     *gtime.Time
	expireAt      *gtime.Time
}

// prepare 统一校验与清洗(内容剥离 HTML, 字典取值, 时间范围, 可见范围权限)
func (s *Service) prepare(ctx context.Context, actor *identity.Identity, in CreateInput) (*preparedData, error) {
	title := strings.TrimSpace(in.Title)
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > MaxTitleLength {
		return nil, gerror.NewCode(consts.CodeBadRequest, fmt.Sprintf("标题长度需为 1-%d 个字符", MaxTitleLength))
	}
	// 统一剥离 HTML 标签, 只存纯文本
	content := sanitizeContent(in.Content)
	if content == "" {
		return nil, gerror.NewCode(consts.CodeBadRequest, "公告内容不能为空")
	}
	if utf8.RuneCountInString(content) > MaxContentLength {
		return nil, gerror.NewCode(consts.CodeBadRequest, fmt.Sprintf("公告内容不能超过 %d 个字符", MaxContentLength))
	}
	noticeType := strings.TrimSpace(in.NoticeType)
	noticeLevel := strings.TrimSpace(in.NoticeLevel)
	if err := validateDictValues(ctx, noticeType, noticeLevel); err != nil {
		return nil, err
	}
	publishAt, err := parseOptionalTime(in.PublishAt)
	if err != nil {
		return nil, err
	}
	expireAt, err := parseOptionalTime(in.ExpireAt)
	if err != nil {
		return nil, err
	}
	if publishAt != nil && expireAt != nil && !expireAt.After(publishAt) {
		return nil, gerror.NewCode(consts.CodeBadRequest, "失效时间必须晚于发布时间")
	}
	targetUserIds, err := checkScope(ctx, actor, in.TargetScope, in.TargetUserIds)
	if err != nil {
		return nil, err
	}
	return &preparedData{
		title:         title,
		content:       content,
		noticeType:    noticeType,
		noticeLevel:   noticeLevel,
		targetScope:   in.TargetScope,
		targetUserIds: targetUserIds,
		publishAt:     publishAt,
		expireAt:      expireAt,
	}, nil
}

// checkScope 范围权限校验: 整个校验在写库前完成, 任一个目标用户越权即整体拒绝
func checkScope(ctx context.Context, actor *identity.Identity, scope int, rawIds []uint64) ([]uint64, error) {
	if scope != ScopeAll && scope != ScopeTargeted {
		return nil, gerror.NewCode(consts.CodeBadRequest, "可见范围取值非法")
	}
	if scope == ScopeAll {
		if !actor.IsSuperAdmin() {
			return nil, gerror.NewCode(consts.CodeForbidden, "管理员不能发布全员公告")
		}
		return nil, nil
	}
	ids := uniqueUserIds(rawIds)
	if len(ids) == 0 {
		return nil, gerror.NewCode(consts.CodeBadRequest, "定向公告必须指定接收用户")
	}
	if actor.IsSuperAdmin() {
		return ids, nil
	}
	descendants, err := logsvc.DescendantUserIds(ctx, actor.Id)
	if err != nil {
		return nil, err
	}
	allowed := make(map[uint64]struct{}, len(descendants))
	for _, id := range descendants {
		allowed[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := allowed[id]; !ok {
			return nil, gerror.NewCode(consts.CodeForbidden, "目标用户超出可管理范围")
		}
	}
	return ids, nil
}

// validateDictValues 类型/级别必须是字典 notice_type / notice_level 的启用值
func validateDictValues(ctx context.Context, noticeType, noticeLevel string) error {
	if noticeType == "" {
		return gerror.NewCode(consts.CodeBadRequest, "请选择公告类型")
	}
	if noticeLevel == "" {
		return gerror.NewCode(consts.CodeBadRequest, "请选择公告级别")
	}
	options, err := dictsvc.New().Options(ctx, []string{dictsvc.TypeNoticeType, dictsvc.TypeNoticeLevel})
	if err != nil {
		return err
	}
	if !hasDictValue(options[dictsvc.TypeNoticeType], noticeType) {
		return gerror.NewCode(consts.CodeBadRequest, "公告类型取值非法")
	}
	if !hasDictValue(options[dictsvc.TypeNoticeLevel], noticeLevel) {
		return gerror.NewCode(consts.CodeBadRequest, "公告级别取值非法")
	}
	return nil
}

// hasDictValue 判断字典启用项中是否存在该值
func hasDictValue(items []*entity.SysDictData, value string) bool {
	for _, item := range items {
		if item.Value == value {
			return true
		}
	}
	return false
}

// sanitizeContent 剥离 HTML 标签, 避免把可执行标记原样入库
func sanitizeContent(raw string) string {
	return strings.TrimSpace(htmlTagPattern.ReplaceAllString(raw, ""))
}

// parseOptionalTime 解析可空时间字符串, 空串返回 nil
func parseOptionalTime(value string) (*gtime.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	t, err := gtime.StrToTime(value)
	if err != nil {
		return nil, gerror.NewCode(consts.CodeBadRequest, "时间格式不正确, 请使用 YYYY-MM-DD HH:mm:ss")
	}
	return t, nil
}

// timeValue 时间写入 Data: nil 显式写 NULL
func timeValue(t *gtime.Time) interface{} {
	if t == nil {
		return gdb.Raw("NULL")
	}
	return t
}

// uniqueUserIds 去重并过滤 0
func uniqueUserIds(raw []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(raw))
	ids := make([]uint64, 0, len(raw))
	for _, id := range raw {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// normalizePage 分页参数归一化
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = defaultNoticePageSize
	}
	return page, pageSize
}

// getById 查询公告(软删数据由 ORM 自动过滤)
func (s *Service) getById(ctx context.Context, id uint64) (*entity.SysNotice, error) {
	var target entity.SysNotice
	err := dao.SysNotice.Ctx(ctx).Where(dao.SysNotice.Columns().Id, id).Scan(&target)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询公告失败")
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "公告不存在")
	}
	return &target, nil
}

// toMyItems 批量补充已读状态(一次查询, 避免逐条查库)
func (s *Service) toMyItems(ctx context.Context, userId uint64, rows []*entity.SysNotice) ([]*MyItem, error) {
	items := make([]*MyItem, 0, len(rows))
	if len(rows) == 0 {
		return items, nil
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Id)
	}
	readCols := dao.SysNoticeRead.Columns()
	var reads []*entity.SysNoticeRead
	if err := dao.SysNoticeRead.Ctx(ctx).
		Where(readCols.UserId, userId).
		WhereIn(readCols.NoticeId, ids).
		Scan(&reads); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询已读记录失败")
	}
	readSet := make(map[uint64]bool, len(reads))
	for _, read := range reads {
		readSet[read.NoticeId] = true
	}
	for _, row := range rows {
		items = append(items, &MyItem{
			Id:          row.Id,
			Title:       row.Title,
			Content:     row.Content,
			NoticeType:  row.NoticeType,
			NoticeLevel: row.NoticeLevel,
			PublishAt:   row.PublishAt,
			CreatedAt:   row.CreatedAt,
			IsRead:      readSet[row.Id],
		})
	}
	return items, nil
}
