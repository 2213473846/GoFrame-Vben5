package log

import (
	"context"
	"fmt"

	v1 "demo/api/log/v1"
	exportsvc "demo/internal/service/export"
	logsvc "demo/internal/service/log"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// LoginLogExport 导出登录日志
func (c *ControllerV1) LoginLogExport(ctx context.Context, req *v1.LoginLogExportReq) (res *v1.LoginLogExportRes, err error) {
	r := g.RequestFromCtx(ctx)
	format, err := exportsvc.NormalizeFormat(req.Format)
	if err != nil {
		return nil, err
	}
	in := &logsvc.LoginListInput{
		Username: req.Username,
		Status:   req.Status,
	}
	filename := exportsvc.FileName("login_logs", format, gtime.Now().Format("YmdHis"))
	r.Response.Header().Set("Content-Type", exportsvc.ContentType(format))
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	count, err := logsvc.New().LoginExport(ctx, in, format, r.Response.RawWriter())
	if err != nil {
		return nil, err
	}
	logsvc.RecordOperationFromRequest(r, "日志管理", "导出登录日志", true, fmt.Sprintf(
		"筛选条件: username=%q status=%d; 共导出 %d 条", req.Username, req.Status, count,
	))
	return &v1.LoginLogExportRes{}, nil
}

// OperationLogExport 导出操作日志
func (c *ControllerV1) OperationLogExport(ctx context.Context, req *v1.OperationLogExportReq) (res *v1.OperationLogExportRes, err error) {
	r := g.RequestFromCtx(ctx)
	format, err := exportsvc.NormalizeFormat(req.Format)
	if err != nil {
		return nil, err
	}
	in := &logsvc.OperationListInput{
		Username: req.Username,
		Status:   req.Status,
	}
	filename := exportsvc.FileName("operation_logs", format, gtime.Now().Format("YmdHis"))
	r.Response.Header().Set("Content-Type", exportsvc.ContentType(format))
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	count, err := logsvc.New().OperationExport(ctx, in, format, r.Response.RawWriter())
	if err != nil {
		return nil, err
	}
	logsvc.RecordOperationFromRequest(r, "日志管理", "导出操作日志", true, fmt.Sprintf(
		"筛选条件: username=%q status=%d; 共导出 %d 条", req.Username, req.Status, count,
	))
	return &v1.OperationLogExportRes{}, nil
}
