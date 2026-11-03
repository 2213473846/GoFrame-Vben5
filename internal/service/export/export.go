// Package export 列表导出: 使用 excelize 流式写出 XLSX, 同时支持 CSV
// 数据分批从数据库读取, 避免大数据量一次性加载到内存
package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

// 导出格式
const (
	FormatXLSX = "xlsx"
	FormatCSV  = "csv"
)

// BatchFunc 分批取数: offset 起始行, limit 批次大小; 返回行数据(字符串)
type BatchFunc func(offset, limit int) ([][]string, error)

// Options 导出选项
type Options struct {
	Sheet   string   // sheet 名称
	Headers []string // 表头
}

// NormalizeFormat 规范化导出格式, 默认 xlsx
func NormalizeFormat(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", FormatXLSX:
		return FormatXLSX, nil
	case FormatCSV:
		return FormatCSV, nil
	default:
		return "", gerror.Newf("不支持的导出格式: %s", raw)
	}
}

// ContentType 返回响应的 Content-Type
func ContentType(format string) string {
	if format == FormatCSV {
		return "text/csv; charset=utf-8"
	}
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// BatchSize 从配置读取导出批次大小(默认 2000, 限制 100-10000)
func BatchSize(ctx context.Context) int {
	size := g.Cfg().MustGet(ctx, "export.batchSize", 2000).Int()
	if size < 100 {
		size = 100
	}
	if size > 10000 {
		size = 10000
	}
	return size
}

// Write 按格式流式写出数据, 返回导出的数据行数(不含表头)
func Write(w io.Writer, format string, opts Options, batchSize int, next BatchFunc) (int, error) {
	if batchSize <= 0 {
		batchSize = 2000
	}
	if format == FormatCSV {
		return writeCSV(w, opts, batchSize, next)
	}
	return writeXLSX(w, opts, batchSize, next)
}

func writeCSV(w io.Writer, opts Options, batchSize int, next BatchFunc) (int, error) {
	// UTF-8 BOM, 保证 Excel 直接打开 CSV 时中文不乱码
	if _, err := fmt.Fprint(w, "\uFEFF"); err != nil {
		return 0, gerror.Wrap(err, "写入 CSV 编码标记失败")
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(opts.Headers); err != nil {
		return 0, gerror.Wrap(err, "写入 CSV 表头失败")
	}
	total := 0
	for {
		rows, err := next(total, batchSize)
		if err != nil {
			return total, err
		}
		for _, row := range rows {
			if err = cw.Write(row); err != nil {
				return total, gerror.Wrap(err, "写入 CSV 数据失败")
			}
		}
		total += len(rows)
		cw.Flush()
		if err = cw.Error(); err != nil {
			return total, gerror.Wrap(err, "写入 CSV 数据失败")
		}
		if len(rows) < batchSize {
			break
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return total, gerror.Wrap(err, "写入 CSV 数据失败")
	}
	return total, nil
}

func writeXLSX(w io.Writer, opts Options, batchSize int, next BatchFunc) (int, error) {
	sheet := strings.TrimSpace(opts.Sheet)
	if sheet == "" {
		sheet = "Sheet1"
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if sheet != "Sheet1" {
		if _, err := f.NewSheet(sheet); err != nil {
			return 0, gerror.Wrap(err, "创建工作表失败")
		}
		_ = f.DeleteSheet("Sheet1")
	}
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return 0, gerror.Wrap(err, "初始化流式写入失败")
	}
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#E8EEF7"}},
	})
	if err != nil {
		return 0, gerror.Wrap(err, "创建表头样式失败")
	}
	headerRow := make([]interface{}, 0, len(opts.Headers))
	for _, h := range opts.Headers {
		headerRow = append(headerRow, excelize.Cell{StyleID: headerStyle, Value: h})
	}
	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err = sw.SetRow(cell, headerRow); err != nil {
		return 0, gerror.Wrap(err, "写入表头失败")
	}
	rowNum := 2
	total := 0
	for {
		rows, err := next(total, batchSize)
		if err != nil {
			return total, err
		}
		for _, row := range rows {
			values := make([]interface{}, len(row))
			for i, v := range row {
				values[i] = v
			}
			cellName, _ := excelize.CoordinatesToCellName(1, rowNum)
			if err = sw.SetRow(cellName, values); err != nil {
				return total, gerror.Wrap(err, "写入表格数据失败")
			}
			rowNum++
			total++
		}
		if len(rows) < batchSize {
			break
		}
	}
	if len(opts.Headers) > 0 {
		_ = sw.SetColWidth(1, len(opts.Headers), 16)
	}
	if err = sw.Flush(); err != nil {
		return total, gerror.Wrap(err, "刷新表格失败")
	}
	// 冻结首行表头, 提升可读性
	if len(opts.Headers) > 0 {
		_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	}
	if _, err = f.WriteTo(w); err != nil {
		return total, gerror.Wrap(err, "输出 Excel 文件失败")
	}
	return total, nil
}

// FileName 生成下载文件名: base_yyyymmddHHMMSS.ext
func FileName(base, format, timestamp string) string {
	ext := format
	if ext == "" {
		ext = FormatXLSX
	}
	return fmt.Sprintf("%s_%s.%s", base, timestamp, ext)
}
