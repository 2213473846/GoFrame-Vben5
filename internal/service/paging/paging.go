// Package paging 统一的内存分页工具: 收口 page/size 规范化与切片逻辑。
package paging

// DefaultPage / DefaultSize 列表分页默认值
const (
	DefaultPage = 1
	DefaultSize = 10
	MaxSize     = 100
)

// Normalize 规范化分页参数: page < 1 取 1; size < 1 取默认, size 超上限取上限。
func Normalize(page, size int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if size < 1 {
		size = DefaultSize
	}
	if size > MaxSize {
		size = MaxSize
	}
	return page, size
}

// Calc 根据总数计算切片下标 [start, end), 语义与既有手写分页完全一致:
// start 为负取 0, 超出总数取总数; end 超出总数或非正时取总数。
func Calc(total, page, size int) (int, int) {
	start := (page - 1) * size
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + size
	if end > total || end <= 0 {
		end = total
	}
	return start, end
}

// Slice 对内存切片分页, 返回当前页数据与总数。
// page/size 先经 Normalize 规范化, 再按 Calc 语义切片。
func Slice[T any](items []T, page, size int) ([]T, int) {
	page, size = Normalize(page, size)
	start, end := Calc(len(items), page, size)
	return items[start:end], len(items)
}
