package user

import (
	"context"
	"fmt"

	"demo/api/user/v1"
	"demo/internal/consts"
	logsvc "demo/internal/service/log"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/xuri/excelize/v2"
)

// ImportTemplate 下载用户导入模板(xlsx: 表头 + 说明行 + 示例行)
func (c *ControllerV1) ImportTemplate(ctx context.Context, req *v1.ImportTemplateReq) (res *v1.ImportTemplateRes, err error) {
	r := g.RequestFromCtx(ctx)
	rows, err := c.userSvc.ImportTemplate(ctx)
	if err != nil {
		return nil, err
	}
	book := excelize.NewFile()
	defer func() { _ = book.Close() }()
	sheet := book.GetSheetName(0)
	for i, row := range rows {
		for j, cell := range row {
			axis, _ := excelize.CoordinatesToCellName(j+1, i+1)
			if err = book.SetCellStr(sheet, axis, cell); err != nil {
				return nil, gerror.Wrap(err, "生成导入模板失败")
			}
		}
	}
	// 表头加粗并设置列宽
	if err = book.SetRowHeight(sheet, 1, 24); err != nil {
		return nil, gerror.Wrap(err, "生成导入模板失败")
	}
	style, styleErr := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if styleErr == nil {
		lastAxis, _ := excelize.CoordinatesToCellName(len(rows[0]), 1)
		_ = book.SetCellStyle(sheet, "A1", lastAxis, style)
	}
	lastCol, _ := excelize.ColumnNumberToName(len(rows[0]))
	_ = book.SetColWidth(sheet, "A", lastCol, 18)

	filename := fmt.Sprintf("用户导入模板_%s.xlsx", gtime.Now().Format("YmdHis"))
	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	if err = book.Write(r.Response.RawWriter()); err != nil {
		return nil, gerror.Wrap(err, "输出导入模板失败")
	}
	return &v1.ImportTemplateRes{}, nil
}

// Import 上传 Excel 导入用户(支持部分成功, atomic=true 时全量回滚)
func (c *ControllerV1) Import(ctx context.Context, req *v1.ImportReq) (res *v1.ImportRes, err error) {
	r := g.RequestFromCtx(ctx)
	upload := r.GetUploadFile("file")
	if upload == nil {
		return nil, gerror.NewCode(consts.CodeBadRequest, "请上传 Excel 文件(file 字段)")
	}
	file, openErr := upload.Open()
	if openErr != nil {
		return nil, gerror.Wrap(openErr, "读取上传文件失败")
	}
	defer func() { _ = file.Close() }()

	result, err := c.userSvc.Import(ctx, file, upload.Size, upload.Filename, req.Atomic)
	if err != nil {
		logsvc.RecordOperationFromRequest(r, "用户管理", "导入用户", false,
			fmt.Sprintf("文件: %s; 导入失败: %v", upload.Filename, err))
		return nil, err
	}
	msg := fmt.Sprintf("文件: %s; 总行数: %d; 成功: %d; 失败: %d", result.FileName, result.Total, result.Success, result.Failed)
	if result.Atomic {
		msg += "; 全量回滚模式"
	}
	logsvc.RecordOperationFromRequest(r, "用户管理", "导入用户", result.Failed == 0, msg)

	out := &v1.ImportRes{
		FileName: result.FileName,
		Atomic:   result.Atomic,
		Total:    result.Total,
		Success:  result.Success,
		Failed:   result.Failed,
		Errors:   make([]v1.ImportRowError, 0, len(result.Errors)),
	}
	for _, item := range result.Errors {
		out.Errors = append(out.Errors, v1.ImportRowError{Row: item.Row, Username: item.Username, Reason: item.Reason})
	}
	return out, nil
}
