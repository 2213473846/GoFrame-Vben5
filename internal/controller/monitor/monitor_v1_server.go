package monitor

import (
	"context"

	v1 "demo/api/monitor/v1"
	monitorsvc "demo/internal/service/monitor"
)

// Server 服务监控
func (c *ControllerV1) Server(ctx context.Context, req *v1.ServerReq) (res *v1.ServerRes, err error) {
	stat, err := monitorsvc.New().Server(ctx)
	if err != nil {
		return nil, err
	}
	res = &v1.ServerRes{
		RefreshSeconds: stat.RefreshSeconds,
		UptimeSeconds:  stat.UptimeSeconds,
		Cpu: v1.ServerCpu{
			Cores:        stat.Cpu.Cores,
			UsagePercent: stat.Cpu.UsagePercent,
			Available:    stat.Cpu.Available,
		},
		Memory: v1.ServerMemory{
			Total:        stat.Memory.Total,
			Used:         stat.Memory.Used,
			UsagePercent: stat.Memory.UsagePercent,
			Available:    stat.Memory.Available,
		},
		Disks:          make([]*v1.ServerDisk, 0, len(stat.Disks)),
		DisksAvailable: stat.DisksAvailable,
		Runtime: v1.ServerRuntime{
			Goroutines:     stat.Runtime.Goroutines,
			HeapAllocBytes: stat.Runtime.HeapAllocBytes,
			HeapSysBytes:   stat.Runtime.HeapSysBytes,
			NumGC:          stat.Runtime.NumGC,
		},
		Deps: v1.ServerDeps{
			Mysql: v1.ServerDep{Status: stat.Deps.Mysql.Status, LatencyMs: stat.Deps.Mysql.LatencyMs},
			Redis: v1.ServerDep{Status: stat.Deps.Redis.Status, LatencyMs: stat.Deps.Redis.LatencyMs},
		},
	}
	for _, item := range stat.Disks {
		res.Disks = append(res.Disks, &v1.ServerDisk{
			Mountpoint:   item.Mountpoint,
			Total:        item.Total,
			Used:         item.Used,
			UsagePercent: item.UsagePercent,
		})
	}
	return res, nil
}
