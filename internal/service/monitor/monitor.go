// Package monitor 服务监控: 采集 CPU/内存/磁盘/Go 运行时/依赖中间件健康状态
//
// 跨平台说明:
//   - gopsutil 已做平台适配, 分区在 Windows 为盘符(如 C:\), Linux 为挂载点(如 /)。
//   - 降级策略: 单个采集项失败只把该字段置零并标记 available=false, 绝不让整个接口 500;
//     磁盘按分区逐个采集, 某个分区失败(如权限不足/已拔出)直接跳过该分区。
//   - 安全边界: 仅超级管理员可访问; 依赖健康检查只返回 ok/fail 与延迟毫秒,
//     不返回错误详情、DSN、路径等敏感信息, 失败原因仅记服务端日志(且不打印连接串)。
package monitor

import (
	"context"
	"runtime"
	"time"

	"demo/internal/service/authz"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// processStartAt 进程启动时间(包初始化时间), 用于计算"进程已运行时长"
// 相比 host.Uptime(主机开机时长), 进程运行时长更贴合服务监控的语义。
var processStartAt = time.Now()

// 依赖健康检查函数(包级变量, 便于测试替换以构造 fail 场景)
var (
	mysqlPing = func(ctx context.Context) error {
		return g.DB().PingMaster()
	}
	redisPing = func(ctx context.Context) error {
		_, err := g.Redis().Do(ctx, "PING")
		return err
	}
)

// 依赖状态文案
const (
	DepOK   = "ok"
	DepFail = "fail"
)

// CPUStat CPU 采集结果
type CPUStat struct {
	Cores        int     `json:"cores"`
	UsagePercent float64 `json:"usagePercent"`
	Available    bool    `json:"available"`
}

// MemoryStat 内存采集结果(字节)
type MemoryStat struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	UsagePercent float64 `json:"usagePercent"`
	Available    bool    `json:"available"`
}

// DiskStat 磁盘分区采集结果(字节)
type DiskStat struct {
	Mountpoint   string  `json:"mountpoint"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	UsagePercent float64 `json:"usagePercent"`
}

// RuntimeStat Go 运行时状态
type RuntimeStat struct {
	Goroutines     int    `json:"goroutines"`
	HeapAllocBytes uint64 `json:"heapAllocBytes"`
	HeapSysBytes   uint64 `json:"heapSysBytes"`
	NumGC          uint32 `json:"numGC"`
}

// DepStat 单个依赖的健康状态(只暴露 ok/fail 与延迟, 不含任何错误详情)
type DepStat struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
}

// DepsStat 依赖健康状态集合
type DepsStat struct {
	Mysql DepStat `json:"mysql"`
	Redis DepStat `json:"redis"`
}

// ServerStat 服务监控汇总结果
type ServerStat struct {
	Cpu            CPUStat     `json:"cpu"`
	Memory         MemoryStat  `json:"memory"`
	Disks          []*DiskStat `json:"disks"`
	DisksAvailable bool        `json:"disksAvailable"`
	Runtime        RuntimeStat `json:"runtime"`
	UptimeSeconds  int64       `json:"uptimeSeconds"`
	Deps           DepsStat    `json:"deps"`
	RefreshSeconds int         `json:"refreshSeconds"`
}

// Service 服务监控
type Service struct{}

// New creates and returns a new Service instance.
func New() *Service {
	return &Service{}
}

// defaultRefreshSeconds 前端自动刷新间隔默认值(秒)
const defaultRefreshSeconds = 30

// Server 采集服务监控数据, 仅超级管理员
func (s *Service) Server(ctx context.Context) (*ServerStat, error) {
	if _, err := authz.SuperAdmin(ctx, "无权查看服务监控"); err != nil {
		return nil, err
	}
	stat := &ServerStat{
		Cpu:            collectCPU(ctx),
		Memory:         collectMemory(ctx),
		Disks:          make([]*DiskStat, 0),
		DisksAvailable: true,
		Runtime:        collectRuntime(),
		UptimeSeconds:  int64(time.Since(processStartAt).Seconds()),
		Deps: DepsStat{
			Mysql: checkDep(ctx, "MySQL", mysqlPing),
			Redis: checkDep(ctx, "Redis", redisPing),
		},
		RefreshSeconds: refreshSeconds(ctx),
	}
	stat.Disks, stat.DisksAvailable = collectDisks(ctx)
	return stat, nil
}

// refreshSeconds 读取前端自动刷新间隔(仅来自 config.yaml, 随接口一起下发避免前端写死)
func refreshSeconds(ctx context.Context) int {
	seconds := g.Cfg().MustGet(ctx, "monitor.refreshSeconds", defaultRefreshSeconds).Int()
	if seconds <= 0 {
		return defaultRefreshSeconds
	}
	if seconds > 3600 {
		return 3600
	}
	return seconds
}

// collectCPU 采集 CPU 核数与使用率(短采样 200ms, 避免请求长时间阻塞)
func collectCPU(ctx context.Context) CPUStat {
	stat := CPUStat{}
	sampleCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if cores, err := cpu.CountsWithContext(sampleCtx, true); err == nil {
		stat.Cores = cores
	}
	percents, err := cpu.PercentWithContext(sampleCtx, 200*time.Millisecond, false)
	if err != nil {
		g.Log().Warning(ctx, "服务监控: CPU 使用率采集失败, 已降级为零值")
		return stat
	}
	if len(percents) > 0 {
		stat.UsagePercent = percents[0]
	}
	if stat.Cores <= 0 {
		stat.Cores = runtime.NumCPU()
	}
	stat.Available = len(percents) > 0
	return stat
}

// collectMemory 采集物理内存使用情况
func collectMemory(ctx context.Context) MemoryStat {
	stat := MemoryStat{}
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil || vm == nil {
		g.Log().Warning(ctx, "服务监控: 内存采集失败, 已降级为零值")
		return stat
	}
	stat.Total = vm.Total
	stat.Used = vm.Used
	stat.UsagePercent = vm.UsedPercent
	stat.Available = true
	return stat
}

// collectDisks 采集磁盘分区; 单个分区失败只跳过该分区, 不影响整体
func collectDisks(ctx context.Context) ([]*DiskStat, bool) {
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		g.Log().Warning(ctx, "服务监控: 磁盘分区枚举失败, 已降级为空列表")
		return make([]*DiskStat, 0), false
	}
	result := make([]*DiskStat, 0, len(partitions))
	for _, item := range partitions {
		if item.Mountpoint == "" {
			continue
		}
		usage, err := disk.UsageWithContext(ctx, item.Mountpoint)
		if err != nil || usage == nil {
			// 单个分区不可用(权限不足/设备已移除等)直接跳过
			continue
		}
		result = append(result, &DiskStat{
			Mountpoint:   item.Mountpoint,
			Total:        usage.Total,
			Used:         usage.Used,
			UsagePercent: usage.UsedPercent,
		})
	}
	return result, true
}

// collectRuntime 采集 Go 运行时指标
func collectRuntime() RuntimeStat {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return RuntimeStat{
		Goroutines:     runtime.NumGoroutine(),
		HeapAllocBytes: m.HeapAlloc,
		HeapSysBytes:   m.HeapSys,
		NumGC:          m.NumGC,
	}
}

// checkDep 对依赖做一次健康探测并计时; 失败只记服务端日志(不打印错误详情/连接串)
func checkDep(ctx context.Context, name string, ping func(context.Context) error) DepStat {
	start := time.Now()
	err := ping(ctx)
	latency := time.Since(start).Milliseconds()
	if latency < 0 {
		latency = 0
	}
	if err != nil {
		g.Log().Warningf(ctx, "服务监控: %s 健康检查失败", name)
		return DepStat{Status: DepFail, LatencyMs: latency}
	}
	return DepStat{Status: DepOK, LatencyMs: latency}
}
