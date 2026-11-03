package user

import (
	"context"
	"fmt"

	"demo/api/user/v1"
	exportsvc "demo/internal/service/export"
	logsvc "demo/internal/service/log"
	usersvc "demo/internal/service/user"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

// Export 导出用户列表(xlsx/csv), 查询条件与数据权限复用列表逻辑
func (c *ControllerV1) Export(ctx context.Context, req *v1.ExportReq) (res *v1.ExportRes, err error) {
	r := g.RequestFromCtx(ctx)
	format, err := exportsvc.NormalizeFormat(req.Format)
	if err != nil {
		return nil, err
	}
	in := usersvc.ListInput{
		Username: req.Username,
		RealName: req.RealName,
		Role:     req.Role,
		Status:   req.Status,
	}
	// 下载响应: 直接写入原始 ResponseWriter, 由 Response 中间件识别 Content-Disposition 后跳过 JSON 包装
	filename := exportsvc.FileName("users", format, gtime.Now().Format("YmdHis"))
	r.Response.Header().Set("Content-Type", exportsvc.ContentType(format))
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	count, err := c.userSvc.Export(ctx, in, format, r.Response.RawWriter())
	if err != nil {
		return nil, err
	}
	statusText := "全部"
	if req.Status != nil {
		statusText = gconv.String(*req.Status)
	}
	logsvc.RecordOperationFromRequest(r, "用户管理", "导出用户列表", true, fmt.Sprintf(
		"筛选条件: username=%q realName=%q role=%q status=%s; 共导出 %d 条", req.Username, req.RealName, req.Role, statusText, count,
	))
	return &v1.ExportRes{}, nil
}
