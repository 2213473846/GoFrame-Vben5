// Package job 定时任务管理: CRUD、启停、手动执行、执行日志与基于 gcron 的调度
//
// 安全边界(重要):
//   - invoke_target 只是白名单标识, 只能取 registry 中已注册的内置任务;
//     绝不按字符串反射调用方法或执行 shell 命令, 避免演变成远程命令执行入口。
//   - 调度使用独立的 gcron 实例(不使用包级默认实例), 避免与其他模块互相影响。
//   - 注册前按唯一 entry 名 job:<id> 先 Remove 再 Add, 并由互斥锁保护 jobId→Entry 映射,
//     保证重复 reload 不会重复注册造成"同一任务每秒跑多次"。
package job

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/authz"
	logsvc "demo/internal/service/log"
	sysconfigsvc "demo/internal/service/sysconfig"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"
)

// 任务状态
const (
	StatusDisabled = 0 // 停用
	StatusEnabled  = 1 // 启用
)

// 触发方式
const (
	TriggerCron   = 1 // 定时触发
	TriggerManual = 2 // 手动触发
)

// 并发策略
const (
	ConcurrencyForbid = 0 // 禁止并发(上次未结束时跳过)
	ConcurrencyAllow  = 1 // 允许并发
)

// 内置任务标识(白名单 key)
const TargetClearLog = "clearlog"

// 执行状态
const (
	RunSuccess = 1
	RunFailed  = 0
)

// JobFunc 内置任务实现: 返回执行摘要(成功时写入执行日志), 失败返回错误
type JobFunc func(ctx context.Context) (message string, err error)

// registry 白名单任务注册表: invoke_target => 任务实现
var (
	registryMu sync.RWMutex
	registry   = map[string]JobFunc{
		TargetClearLog: clearLog,
	}
)

// lookupJobFunc 从白名单注册表取任务实现
func lookupJobFunc(target string) (JobFunc, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	fn, ok := registry[target]
	return fn, ok
}

// scheduler 独立调度器实例(不使用 gcron 包级默认实例)
var scheduler = gcron.New()

// 调度注册表: jobId => gcron Entry, 由 mu 保护
var (
	mu      sync.Mutex
	entries = map[uint64]*gcron.Entry{}
)

// entryName 唯一 entry 名, 用于注册与移除
func entryName(jobId uint64) string {
	return fmt.Sprintf("job:%d", jobId)
}

// unregister 移除某任务的调度注册(幂等)
func unregister(jobId uint64) {
	mu.Lock()
	defer mu.Unlock()
	scheduler.Remove(entryName(jobId))
	delete(entries, jobId)
}

// Service 定时任务服务
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// ------------------------------------------------------------
// 查询与写入
// ------------------------------------------------------------

// ListInput 任务列表查询条件
type ListInput struct {
	Page     int
	PageSize int
	Name     string
	Status   int // -1=全部 1=启用 0=停用
}

// ListOutput 任务列表结果
type ListOutput struct {
	Items []*entity.SysJob
	Total int
}

// List 任务列表(超管/管理员可读)
func (s *Service) List(ctx context.Context, in ListInput) (*ListOutput, error) {
	if _, err := authz.Manager(ctx, "无权查看定时任务"); err != nil {
		return nil, err
	}
	cols := dao.SysJob.Columns()
	m := dao.SysJob.Ctx(ctx)
	if name := strings.TrimSpace(in.Name); name != "" {
		m = m.WhereLike(cols.Name, "%"+name+"%")
	}
	if in.Status == StatusDisabled || in.Status == StatusEnabled {
		m = m.Where(cols.Status, in.Status)
	}
	total, err := m.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "查询定时任务失败")
	}
	var items []*entity.SysJob
	if err = m.Page(in.Page, in.PageSize).OrderDesc(cols.Id).Scan(&items); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询定时任务失败")
	}
	return &ListOutput{Items: items, Total: total}, nil
}

// LogListInput 执行日志查询条件
type LogListInput struct {
	Page     int
	PageSize int
	JobId    uint64
	Status   int // -1=全部 1=成功 0=失败
}

// LogListOutput 执行日志结果
type LogListOutput struct {
	Items []*entity.SysJobLog
	Total int
}

// Logs 执行日志列表(超管/管理员可读)
func (s *Service) Logs(ctx context.Context, in LogListInput) (*LogListOutput, error) {
	if _, err := authz.Manager(ctx, "无权查看定时任务日志"); err != nil {
		return nil, err
	}
	cols := dao.SysJobLog.Columns()
	m := dao.SysJobLog.Ctx(ctx)
	if in.JobId > 0 {
		m = m.Where(cols.JobId, in.JobId)
	}
	if in.Status == RunSuccess || in.Status == RunFailed {
		m = m.Where(cols.Status, in.Status)
	}
	total, err := m.Count()
	if err != nil {
		return nil, gerror.Wrap(err, "查询任务日志失败")
	}
	var items []*entity.SysJobLog
	if err = m.Page(in.Page, in.PageSize).OrderDesc(cols.Id).Scan(&items); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return nil, gerror.Wrap(err, "查询任务日志失败")
	}
	return &LogListOutput{Items: items, Total: total}, nil
}

// CreateInput 新增/修改任务输入
type CreateInput struct {
	Name              string
	JobGroup          string
	InvokeTarget      string
	CronExpression    string
	Status            int
	ConcurrencyPolicy int
	TimeoutSeconds    int
	Remark            string
}

// UpdateInput 修改任务输入
type UpdateInput struct {
	Id uint64
	CreateInput
}

// Create 新增任务, 仅超管
func (s *Service) Create(ctx context.Context, in CreateInput) (uint64, error) {
	if _, err := authz.SuperAdmin(ctx, "仅超级管理员可新增定时任务"); err != nil {
		return 0, err
	}
	normalized, err := s.prepare(ctx, in, 0)
	if err != nil {
		return 0, err
	}
	lastId, err := dao.SysJob.Ctx(ctx).Data(normalized).InsertAndGetId()
	if err != nil {
		return 0, gerror.Wrap(err, "新增定时任务失败")
	}
	if err = s.reload(ctx, uint64(lastId)); err != nil {
		return 0, err
	}
	return uint64(lastId), nil
}

// Update 修改任务, 仅超管
func (s *Service) Update(ctx context.Context, in UpdateInput) error {
	if _, err := authz.SuperAdmin(ctx, "仅超级管理员可修改定时任务"); err != nil {
		return err
	}
	if _, err := s.getById(ctx, in.Id); err != nil {
		return err
	}
	normalized, err := s.prepare(ctx, in.CreateInput, in.Id)
	if err != nil {
		return err
	}
	if _, err = dao.SysJob.Ctx(ctx).Where(dao.SysJob.Columns().Id, in.Id).Data(normalized).Update(); err != nil {
		return gerror.Wrap(err, "修改定时任务失败")
	}
	return s.reload(ctx, in.Id)
}

// Delete 删除任务(软删除), 仅超管
func (s *Service) Delete(ctx context.Context, id uint64) error {
	if _, err := authz.SuperAdmin(ctx, "仅超级管理员可删除定时任务"); err != nil {
		return err
	}
	if _, err := s.getById(ctx, id); err != nil {
		return err
	}
	if _, err := dao.SysJob.Ctx(ctx).Where(dao.SysJob.Columns().Id, id).Delete(); err != nil {
		return gerror.Wrap(err, "删除定时任务失败")
	}
	unregister(id)
	return nil
}

// ChangeStatus 启停任务, 仅超管
func (s *Service) ChangeStatus(ctx context.Context, id uint64, status int) error {
	if _, err := authz.SuperAdmin(ctx, "仅超级管理员可启停定时任务"); err != nil {
		return err
	}
	if status != StatusEnabled && status != StatusDisabled {
		return gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	if _, err := s.getById(ctx, id); err != nil {
		return err
	}
	if _, err := dao.SysJob.Ctx(ctx).Where(dao.SysJob.Columns().Id, id).Data(do.SysJob{
		Status: status,
	}).Update(); err != nil {
		return gerror.Wrap(err, "启停定时任务失败")
	}
	return s.reload(ctx, id)
}

// Run 手动执行任务(异步受理, 立即返回), 仅超管
// 停用任务拒绝手动执行: 与"停用即不得调度"语义保持一致, 避免绕过停用。
func (s *Service) Run(ctx context.Context, id uint64) error {
	actor, err := authz.SuperAdmin(ctx, "仅超级管理员可手动执行定时任务")
	if err != nil {
		return err
	}
	job, err := s.getById(ctx, id)
	if err != nil {
		return err
	}
	if job.Status != StatusEnabled {
		return gerror.NewCode(consts.CodeBadRequest, "任务已停用")
	}
	if _, ok := lookupJobFunc(job.InvokeTarget); !ok {
		return gerror.NewCode(consts.CodeBadRequest, "任务标识不在白名单中")
	}
	jobCopy := *job
	go func() {
		// 使用独立 context, 避免 HTTP 请求结束后 context 被取消影响异步执行
		s.execute(gctx.New(), &jobCopy, TriggerManual, actor.Username)
	}()
	return nil
}

// ------------------------------------------------------------
// 校验与转换
// ------------------------------------------------------------

// prepare 校验输入并转换为 DO 对象
func (s *Service) prepare(ctx context.Context, in CreateInput, selfId uint64) (do.SysJob, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return do.SysJob{}, gerror.NewCode(consts.CodeBadRequest, "任务名称不能为空")
	}
	if len(name) > 64 {
		return do.SysJob{}, gerror.NewCode(consts.CodeBadRequest, "任务名称不能超过 64 个字符")
	}
	group := strings.TrimSpace(in.JobGroup)
	if group == "" {
		group = "default"
	}
	if len(group) > 64 {
		return do.SysJob{}, gerror.NewCode(consts.CodeBadRequest, "任务分组不能超过 64 个字符")
	}
	target := strings.TrimSpace(in.InvokeTarget)
	if target == "" {
		return do.SysJob{}, gerror.NewCode(consts.CodeBadRequest, "任务标识不能为空")
	}
	if _, ok := lookupJobFunc(target); !ok {
		return do.SysJob{}, gerror.NewCode(consts.CodeBadRequest, "任务标识不在白名单中")
	}
	if _, err := normalizeCronExpression(in.CronExpression); err != nil {
		return do.SysJob{}, err
	}
	if in.Status != StatusEnabled && in.Status != StatusDisabled {
		return do.SysJob{}, gerror.NewCode(consts.CodeBadRequest, "状态值非法")
	}
	if in.ConcurrencyPolicy != ConcurrencyAllow && in.ConcurrencyPolicy != ConcurrencyForbid {
		return do.SysJob{}, gerror.NewCode(consts.CodeBadRequest, "并发策略非法")
	}
	if in.TimeoutSeconds < 0 || in.TimeoutSeconds > 86400 {
		return do.SysJob{}, gerror.NewCode(consts.CodeBadRequest, "超时秒数需在 0-86400 之间, 0 表示不限制")
	}
	if err := s.checkNameUnique(ctx, name, selfId); err != nil {
		return do.SysJob{}, err
	}
	return do.SysJob{
		Name:              name,
		JobGroup:          group,
		InvokeTarget:      target,
		CronExpression:    strings.TrimSpace(in.CronExpression),
		Status:            in.Status,
		ConcurrencyPolicy: in.ConcurrencyPolicy,
		TimeoutSeconds:    in.TimeoutSeconds,
		Remark:            in.Remark,
	}, nil
}

// checkNameUnique 任务名称唯一(表上为普通索引, 由服务层保证)
func (s *Service) checkNameUnique(ctx context.Context, name string, selfId uint64) error {
	cols := dao.SysJob.Columns()
	m := dao.SysJob.Ctx(ctx).Where(cols.Name, name)
	if selfId > 0 {
		m = m.WhereNot(cols.Id, selfId)
	}
	count, err := m.Count()
	if err != nil {
		return gerror.Wrap(err, "校验任务名称失败")
	}
	if count > 0 {
		return gerror.NewCode(consts.CodeBadRequest, "任务名称已存在")
	}
	return nil
}

// normalizeCronExpression 校验 5/6 段 cron 表达式并返回 gcron 可用的表达式
// gcron 标准写法需要 6 段(秒 分 时 日 月 周); 5 段标准 crontab 用 "# " 前缀表示省略秒。
func normalizeCronExpression(expression string) (string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return "", gerror.NewCode(consts.CodeBadRequest, "cron 表达式不能为空")
	}
	fields := strings.Fields(expression)
	if len(fields) != 5 && len(fields) != 6 {
		return "", gerror.NewCode(consts.CodeBadRequest,
			"cron 表达式必须为 5 段(分 时 日 月 周)或 6 段(秒 分 时 日 月 周)")
	}
	pattern := strings.Join(fields, " ")
	if len(fields) == 5 {
		pattern = "# " + pattern
	}
	// 借用 gcron 的解析能力做最终校验(校验实例用完即移除, 不参与实际调度)
	validator := gcron.New()
	if _, err := validator.AddEntry(context.Background(), pattern, func(ctx context.Context) {}, -1, false, "validate"); err != nil {
		return "", gerror.NewCode(consts.CodeBadRequest, "cron 表达式非法: "+err.Error())
	}
	validator.Remove("validate")
	return pattern, nil
}

// getById 查询任务
func (s *Service) getById(ctx context.Context, id uint64) (*entity.SysJob, error) {
	var target entity.SysJob
	if err := dao.SysJob.Ctx(ctx).Where(dao.SysJob.Columns().Id, id).Scan(&target); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, gerror.Wrap(err, "查询定时任务失败")
		}
	}
	if target.Id == 0 {
		return nil, gerror.NewCode(consts.CodeNotFound, "定时任务不存在")
	}
	return &target, nil
}

// ------------------------------------------------------------
// 调度注册
// ------------------------------------------------------------

// Start 服务启动时对所有启用任务注册调度
func (s *Service) Start(ctx context.Context) error {
	cols := dao.SysJob.Columns()
	var jobs []*entity.SysJob
	if err := dao.SysJob.Ctx(ctx).Where(cols.Status, StatusEnabled).Scan(&jobs); err != nil {
		return gerror.Wrap(err, "加载定时任务失败")
	}
	for _, item := range jobs {
		if err := s.reload(ctx, item.Id); err != nil {
			// 单个任务注册失败不影响服务启动, 记录日志后继续
			g.Log().Warningf(ctx, "启动加载定时任务失败 id=%d: %v", item.Id, err)
		}
	}
	return nil
}

// reload 按任务最新配置重建调度: 先移除同名 entry, 再注册, 防止重复注册导致任务被多次执行
func (s *Service) reload(ctx context.Context, jobId uint64) error {
	// 先移除旧注册(幂等)
	unregister(jobId)

	var target entity.SysJob
	if err := dao.SysJob.Ctx(ctx).Where(dao.SysJob.Columns().Id, jobId).Scan(&target); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return gerror.Wrap(err, "查询定时任务失败")
	}
	if target.Id == 0 || target.Status != StatusEnabled {
		return nil
	}
	if _, ok := lookupJobFunc(target.InvokeTarget); !ok {
		return gerror.NewCode(consts.CodeBadRequest, "任务标识不在白名单中")
	}
	pattern, err := normalizeCronExpression(target.CronExpression)
	if err != nil {
		return err
	}
	jobCopy := target
	runner := func(jobCtx context.Context) {
		s.execute(jobCtx, &jobCopy, TriggerCron, "")
	}
	name := entryName(jobId)
	// 使用独立 context 注册, 避免绑定到某次 HTTP 请求
	registerCtx := gctx.New()

	mu.Lock()
	defer mu.Unlock()
	var entry *gcron.Entry
	if target.ConcurrencyPolicy == ConcurrencyAllow {
		entry, err = scheduler.Add(registerCtx, pattern, runner, name)
	} else {
		entry, err = scheduler.AddSingleton(registerCtx, pattern, runner, name)
	}
	if err != nil {
		return gerror.Wrap(err, "注册定时任务失败")
	}
	entries[jobId] = entry
	return nil
}

// ------------------------------------------------------------
// 执行与日志
// ------------------------------------------------------------

// execute 执行任务并写入执行日志、回写最近执行结果
func (s *Service) execute(ctx context.Context, job *entity.SysJob, triggerType int, operator string) {
	startedAt := gtime.Now()
	startTime := time.Now()

	message, err := runWithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second, job.InvokeTarget)
	finishedAt := gtime.Now()
	durationMs := time.Since(startTime).Milliseconds()

	status := RunSuccess
	summary := strings.TrimSpace(message)
	if err != nil {
		status = RunFailed
		if summary == "" {
			summary = err.Error()
		} else {
			summary = summary + "; 错误: " + err.Error()
		}
	}
	// 手动触发时补充操作人, 便于审计(可选信息, 不影响任务本身)
	if triggerType == TriggerManual && operator != "" {
		summary = strings.TrimSpace(summary + " (操作人: " + operator + ")")
	}
	if summary == "" {
		summary = "执行成功"
	}

	lastResult := "成功"
	if status == RunFailed {
		lastResult = "失败"
	}
	// 先回写任务最近执行结果, 再写执行日志: 保证"能查到执行日志"时任务状态已是最新,
	// 避免异步执行场景下前端/测试读到旧值。
	// next_run_at: gcron 未暴露下次执行时间计算接口, 保持为空, 不造数据
	if _, err := dao.SysJob.Ctx(gctx.New()).Where(dao.SysJob.Columns().Id, job.Id).Data(do.SysJob{
		LastRunAt:  startedAt,
		LastResult: lastResult,
	}).Update(); err != nil {
		g.Log().Warningf(ctx, "定时任务最近执行结果回写失败 jobId=%d: %v", job.Id, err)
	}

	// 日志写入使用独立 context, 保证异步/请求结束后仍能落库
	_, logErr := dao.SysJobLog.Ctx(gctx.New()).Data(do.SysJobLog{
		JobId:       job.Id,
		JobName:     job.Name,
		TriggerType: triggerType,
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		DurationMs:  durationMs,
		Status:      status,
		Message:     summary,
	}).Insert()
	if logErr != nil {
		g.Log().Warningf(ctx, "定时任务执行日志写入失败 jobId=%d: %v", job.Id, logErr)
	}
}

// runWithTimeout 在独立 goroutine 中执行任务, 并按超时时间等待结果
func runWithTimeout(ctx context.Context, timeout time.Duration, target string) (string, error) {
	fn, ok := lookupJobFunc(target)
	if !ok {
		return "", gerror.NewCode(consts.CodeBadRequest, "任务标识不在白名单中")
	}
	runCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	type result struct {
		message string
		err     error
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: gerror.Newf("任务执行 panic: %v", r)}
			}
		}()
		message, err := fn(runCtx)
		done <- result{message: message, err: err}
	}()
	if timeout <= 0 {
		r := <-done
		return r.message, r.err
	}
	select {
	case r := <-done:
		return r.message, r.err
	case <-runCtx.Done():
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return "", gerror.New("执行超时")
		}
		return "", gerror.New("任务已取消")
	}
}

// ------------------------------------------------------------
// 内置任务
// ------------------------------------------------------------

// clearLog 清理过期登录日志与操作日志, 保留天数读 sys_config 的 log.retainDays
func clearLog(ctx context.Context) (string, error) {
	retainDays := sysconfigsvc.New().GetInt(ctx, sysconfigsvc.KeyLogRetainDays, 90)
	result, err := logsvc.New().ClearExpired(ctx, retainDays)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("登录日志 %d 条 / 操作日志 %d 条", result.LoginLogCount, result.OperationLogCount), nil
}

// ------------------------------------------------------------
// 种子数据
// ------------------------------------------------------------

// Seed 内置 clearlog 任务幂等插入(按 invoke_target 判存)
// 默认停用: 定时清理会物理删除数据, 默认不让它悄悄执行, 由超管确认后手动启用。
func (s *Service) Seed(ctx context.Context) error {
	cols := dao.SysJob.Columns()
	count, err := dao.SysJob.Ctx(ctx).Where(cols.InvokeTarget, TargetClearLog).Count()
	if err != nil {
		return gerror.Wrap(err, "初始化内置定时任务失败")
	}
	if count > 0 {
		return nil
	}
	if _, err = dao.SysJob.Ctx(ctx).Data(do.SysJob{
		Name:              "清理过期日志",
		JobGroup:          "default",
		InvokeTarget:      TargetClearLog,
		CronExpression:    "0 0 3 * * *",
		Status:            StatusDisabled,
		ConcurrencyPolicy: ConcurrencyForbid,
		TimeoutSeconds:    300,
		Remark:            "系统内置: 按 sys_config 的 log.retainDays 清理过期登录/操作日志, 默认停用",
	}).Insert(); err != nil {
		return gerror.Wrap(err, "初始化内置定时任务失败")
	}
	return nil
}
