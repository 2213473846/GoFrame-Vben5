package v1

import "github.com/gogf/gf/v2/frame/g"

// ServerReq 服务监控(仅超级管理员)
type ServerReq struct {
	g.Meta `path:"/monitor/server" method:"get" tags:"Monitor" summary:"服务监控"`
}

// ServerCpu CPU 指标
type ServerCpu struct {
	Cores        int     `json:"cores"`
	UsagePercent float64 `json:"usagePercent"`
	Available    bool    `json:"available"`
}

// ServerMemory 内存指标(字节)
type ServerMemory struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	UsagePercent float64 `json:"usagePercent"`
	Available    bool    `json:"available"`
}

// ServerDisk 磁盘分区指标(字节)
type ServerDisk struct {
	Mountpoint   string  `json:"mountpoint"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	UsagePercent float64 `json:"usagePercent"`
}

// ServerRuntime Go 运行时指标
type ServerRuntime struct {
	Goroutines     int    `json:"goroutines"`
	HeapAllocBytes uint64 `json:"heapAllocBytes"`
	HeapSysBytes   uint64 `json:"heapSysBytes"`
	NumGC          uint32 `json:"numGC"`
}

// ServerDep 单个依赖健康状态(只含 ok/fail 与延迟, 不含错误详情)
type ServerDep struct {
	Status    string `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
}

// ServerDeps 依赖健康状态
type ServerDeps struct {
	Mysql ServerDep `json:"mysql"`
	Redis ServerDep `json:"redis"`
}

// ServerRes 服务监控结果
type ServerRes struct {
	RefreshSeconds int           `json:"refreshSeconds"` // 前端自动刷新间隔(秒), 来自 config.yaml 的 monitor.refreshSeconds
	UptimeSeconds  int64         `json:"uptimeSeconds"`  // 进程已运行时长(秒)
	Cpu            ServerCpu     `json:"cpu"`
	Memory         ServerMemory  `json:"memory"`
	Disks          []*ServerDisk `json:"disks"`
	DisksAvailable bool          `json:"disksAvailable"`
	Runtime        ServerRuntime `json:"runtime"`
	Deps           ServerDeps    `json:"deps"`
}
