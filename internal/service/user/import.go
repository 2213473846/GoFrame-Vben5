package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	"demo/internal/model/entity"
	"demo/internal/service/authz"
	"demo/internal/service/identity"
	rolesvc "demo/internal/service/role"
	secsvc "demo/internal/service/security"
	sysconfigsvc "demo/internal/service/sysconfig"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

// 导入模板列(超管可见"最大设备数", 管理员模板不含该列)
const (
	importColUsername   = "用户名"
	importColRealName   = "姓名"
	importColPassword   = "密码"
	importColRole       = "角色"
	importColStatus     = "状态"
	importColMaxDevices = "最大设备数"
	importColRoleCodes  = "功能角色"
)

const (
	importDefaultMaxFileBytes = 5 * 1024 * 1024
	importDefaultMaxRows      = 5000
	importDefaultBatchSize    = 200
)

// ImportRowError 单行导入失败明细
type ImportRowError struct {
	Row      int    `json:"row"`      // Excel 行号(从 1 开始, 表头为第 1 行)
	Username string `json:"username"` // 该行用户名(可能为空)
	Reason   string `json:"reason"`   // 失败原因
}

// ImportResult 导入结果(支持部分成功)
type ImportResult struct {
	FileName string           `json:"fileName"`
	Atomic   bool             `json:"atomic"` // 是否全量回滚模式
	Total    int              `json:"total"`  // 解析到的数据行数
	Success  int              `json:"success"`
	Failed   int              `json:"failed"`
	Errors   []ImportRowError `json:"errors"`
}

// ImportTemplate 返回导入模板内容(表头 + 说明行 + 示例行)
// 仅超管/管理员可下载; "最大设备数"列仅超管模板包含
func (s *Service) ImportTemplate(ctx context.Context) (rows [][]string, err error) {
	actor, err := authz.Manager(ctx, "无权限导入用户")
	if err != nil {
		return nil, err
	}
	headers := []string{importColUsername, importColRealName, importColPassword, importColRole, importColStatus}
	if actor.IsSuperAdmin() {
		headers = append(headers, importColMaxDevices)
	}
	headers = append(headers, importColRoleCodes)

	instruction := []string{"说明: 用户名/密码/角色/状态为必填; 角色填 admin 或 user(也可填 管理员/用户); 状态填 1/0 或 启用/禁用; 功能角色填角色编码, 多个用英文逗号分隔; 下载后请删除本说明行与示例行"}
	if actor.IsSuperAdmin() {
		instruction = append(instruction, "最大设备数仅超级管理员可填, 留空表示使用系统默认")
	}
	instruction = append(instruction, "")

	example := []string{"示例: demo01", "演示用户", "Abc123456", "user", "启用"}
	if actor.IsSuperAdmin() {
		example = append(example, "1")
	}
	example = append(example, "")
	return [][]string{headers, instruction, example}, nil
}

// importLimits 读取导入限制(sys_config 优先, config.yaml 兜底)
func importLimits(ctx context.Context) (maxFileBytes int64, maxRows, batchSize int) {
	cfg := sysconfigsvc.New()
	maxFileBytes = int64(cfg.GetInt(ctx, sysconfigsvc.KeyImportMaxFileBytes,
		g.Cfg().MustGet(ctx, "import.maxFileBytes", importDefaultMaxFileBytes).Int()))
	maxRows = cfg.GetInt(ctx, sysconfigsvc.KeyImportMaxRows,
		g.Cfg().MustGet(ctx, "import.maxRows", importDefaultMaxRows).Int())
	batchSize = g.Cfg().MustGet(ctx, "import.batchSize", importDefaultBatchSize).Int()
	if maxFileBytes <= 0 {
		maxFileBytes = importDefaultMaxFileBytes
	}
	if maxRows <= 0 {
		maxRows = importDefaultMaxRows
	}
	if batchSize <= 0 || batchSize > 1000 {
		batchSize = importDefaultBatchSize
	}
	return
}

type importRow struct {
	row      int
	username string
	realName string
	password string
	role     string
	status   int
	maxDev   *int
	roleIds  []uint64
}

// Import 从 Excel 导入用户(流式读取, 分批处理, 逐行校验)
// 权限: 超管可导入 admin/user; 管理员只能在自己名下创建 admin/user, 且只能分配自己可分配的功能角色;
// 管理员携带"最大设备数"列或填值会被逐行拒绝(不静默降级)。
// atomic=true 时任一失败全量回滚(默认 false, 即部分成功)。
func (s *Service) Import(ctx context.Context, reader io.Reader, size int64, fileName string, atomic bool) (result *ImportResult, err error) {
	actor, err := authz.Manager(ctx, "无权限导入用户")
	if err != nil {
		return nil, err
	}
	maxFileBytes, maxRows, batchSize := importLimits(ctx)
	if size > maxFileBytes {
		return nil, gerror.NewCode(consts.CodeBadRequest,
			fmt.Sprintf("文件大小超过限制(最大 %s)", humanBytes(maxFileBytes)))
	}

	book, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, gerror.NewCode(consts.CodeBadRequest, "无法解析 Excel 文件, 请上传 .xlsx 模板")
	}
	defer func() { _ = book.Close() }()
	sheet := book.GetSheetName(0)
	if sheet == "" {
		return nil, gerror.NewCode(consts.CodeBadRequest, "Excel 文件没有可用的工作表")
	}
	rowIter, err := book.Rows(sheet)
	if err != nil {
		return nil, gerror.NewCode(consts.CodeBadRequest, "读取 Excel 内容失败")
	}
	defer func() { _ = rowIter.Close() }()

	result = &ImportResult{FileName: fileName, Atomic: atomic, Errors: []ImportRowError{}}
	// 角色可分配清单(管理员仅返回其可分配角色)
	roleMap, err := assignableRoleMap(ctx)
	if err != nil {
		return nil, err
	}

	var (
		headerMap map[string]int
		dataSeen  int
		rowNo     int
		batch     []importRow
		seenNames = make(map[string]int, 256)
	)

	flush := func(txCtx context.Context) error {
		if len(batch) == 0 {
			return nil
		}
		if err := s.processImportBatch(txCtx, batch, result, atomic); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	run := func(txCtx context.Context) error {
		for rowIter.Next() {
			rowNo++
			cols, colErr := rowIter.Columns()
			if colErr != nil {
				return gerror.NewCode(consts.CodeBadRequest, "读取 Excel 行失败")
			}
			// 表头: 第 1 行
			if headerMap == nil {
				headerMap = buildHeaderMap(cols)
				for _, required := range []string{importColUsername, importColPassword, importColRole, importColStatus} {
					if _, ok := headerMap[required]; !ok {
						return gerror.NewCode(consts.CodeBadRequest, "模板表头不正确, 缺少列: "+required)
					}
				}
				continue
			}
			// 跳过说明行/示例行/空行
			if isSkippableRow(cols) {
				continue
			}
			dataSeen++
			if dataSeen > maxRows {
				return gerror.NewCode(consts.CodeBadRequest, fmt.Sprintf("数据行数超过限制(最大 %d 行)", maxRows))
			}
			parsed, parseErr := parseImportRow(ctx, rowNo, cols, headerMap, actor, roleMap)
			if parseErr != nil {
				result.Errors = append(result.Errors, ImportRowError{Row: rowNo, Username: parsed.username, Reason: parseErr.Error()})
				result.Failed++
				if atomic {
					result.Total = dataSeen
					return errAtomicRollback
				}
				continue
			}
			// 文件内用户名重复: 首次出现可用, 后续行报错
			if firstRow, ok := seenNames[parsed.username]; ok {
				result.Errors = append(result.Errors, ImportRowError{
					Row: rowNo, Username: parsed.username,
					Reason: fmt.Sprintf("用户名在文件内重复(与第 %d 行相同)", firstRow),
				})
				result.Failed++
				if atomic {
					result.Total = dataSeen
					return errAtomicRollback
				}
				continue
			}
			seenNames[parsed.username] = rowNo
			batch = append(batch, parsed)
			if len(batch) >= batchSize {
				if err := flush(txCtx); err != nil {
					return err
				}
			}
		}
		if rowIter.Error() != nil {
			return gerror.NewCode(consts.CodeBadRequest, "读取 Excel 内容失败")
		}
		return flush(txCtx)
	}

	if atomic {
		err = g.DB().Transaction(ctx, func(txCtx context.Context, tx gdb.TX) error {
			if runErr := run(txCtx); runErr != nil {
				return runErr
			}
			return nil
		})
		if err != nil && !errors.Is(err, errAtomicRollback) {
			return nil, err
		}
	} else {
		if err = run(ctx); err != nil {
			return nil, err
		}
	}
	result.Total = dataSeen
	if atomic && result.Failed > 0 {
		// 全量回滚: 成功数归零, 失败数等于失败明细行数
		result.Success = 0
	}
	return result, nil
}

// errAtomicRollback 全量回滚哨兵错误(触发事务 rollback)
var errAtomicRollback = errors.New("atomic import rollback")

// processImportBatch 批量校验库内重名并逐行写入
func (s *Service) processImportBatch(ctx context.Context, batch []importRow,
	result *ImportResult, atomic bool) error {
	if len(batch) == 0 {
		return nil
	}
	cols := dao.SysUser.Columns()
	names := make([]string, 0, len(batch))
	for _, item := range batch {
		names = append(names, item.username)
	}
	var existing []*entity.SysUser
	if err := dao.SysUser.Ctx(ctx).WhereIn(cols.Username, names).Fields(cols.Username).Scan(&existing); err != nil {
		return gerror.Wrap(err, "校验用户名失败")
	}
	exists := make(map[string]bool, len(existing))
	for _, e := range existing {
		exists[e.Username] = true
	}
	actor := identity.Get(ctx)
	pid := uint64(0)
	if !actor.IsSuperAdmin() {
		pid = actor.Id
	}
	for _, item := range batch {
		if exists[item.username] {
			result.Errors = append(result.Errors, ImportRowError{
				Row: item.row, Username: item.username, Reason: "用户名已存在",
			})
			result.Failed++
			if atomic {
				return errAtomicRollback
			}
			continue
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(item.password), bcrypt.DefaultCost)
		if hashErr != nil {
			return gerror.Wrap(hashErr, "密码加密失败")
		}
		realName := item.realName
		if realName == "" {
			realName = item.username
		}
		data := do.SysUser{
			Pid:                pid,
			Username:           item.username,
			Password:           string(hash),
			RealName:           realName,
			Role:               item.role,
			Status:             item.status,
			Remark:             "Excel导入(" + result.FileName + ")",
			MustChangePassword: 1,
		}
		if item.maxDev != nil && *item.maxDev > 0 {
			data.MaxDevices = *item.maxDev
		}
		newId, insErr := dao.SysUser.Ctx(ctx).Data(data).InsertAndGetId()
		if insErr != nil {
			result.Errors = append(result.Errors, ImportRowError{
				Row: item.row, Username: item.username, Reason: "写入用户失败: " + insErr.Error(),
			})
			result.Failed++
			if atomic {
				return errAtomicRollback
			}
			continue
		}
		userId := gconv.Uint64(newId)
		if len(item.roleIds) > 0 {
			if roleErr := (rolesvc.New()).SetUserRoles(ctx, userId, item.roleIds); roleErr != nil {
				result.Errors = append(result.Errors, ImportRowError{
					Row: item.row, Username: item.username, Reason: "分配功能角色失败: " + roleErr.Error(),
				})
				result.Failed++
				if atomic {
					return errAtomicRollback
				}
				// 部分成功模式下该用户已写入, 角色分配失败需删除以免留下半成品
				if _, delErr := dao.SysUser.Ctx(ctx).Where(cols.Id, userId).Delete(); delErr != nil {
					return gerror.Wrap(delErr, "回滚导入用户失败")
				}
				continue
			}
		}
		exists[item.username] = true
		result.Success++
	}
	return nil
}

// assignableRoleMap 返回当前操作人可分配的功能角色: 编码/名称 -> 角色ID(仅启用角色)
func assignableRoleMap(ctx context.Context) (map[string]uint64, error) {
	roles, err := (rolesvc.New()).List(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]uint64, len(roles)*2)
	for _, r := range roles {
		if r.Status != 1 {
			continue
		}
		m[strings.ToLower(strings.TrimSpace(r.Code))] = r.Id
		m[strings.TrimSpace(r.Name)] = r.Id
	}
	return m, nil
}

func buildHeaderMap(row []string) map[string]int {
	m := make(map[string]int, len(row))
	for i, cell := range row {
		key := strings.TrimSpace(cell)
		if key == "" {
			continue
		}
		m[key] = i
	}
	return m
}

// isSkippableRow 说明行/示例行/空行
func isSkippableRow(row []string) bool {
	nonEmpty := 0
	first := ""
	for i, cell := range row {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		nonEmpty++
		if i == 0 {
			first = cell
		}
	}
	if nonEmpty == 0 {
		return true
	}
	return strings.HasPrefix(first, "说明") || strings.HasPrefix(first, "示例")
}

func parseImportRow(ctx context.Context, rowNo int, cols []string, header map[string]int, actor *identity.Identity,
	roleMap map[string]uint64) (out importRow, err error) {
	out.row = rowNo
	out.status = 1
	get := func(name string) string {
		if idx, ok := header[name]; ok && idx < len(cols) {
			return strings.TrimSpace(cols[idx])
		}
		return ""
	}
	out.username = get(importColUsername)
	out.realName = get(importColRealName)
	out.password = get(importColPassword)
	if out.username == "" {
		return out, gerror.New("用户名不能为空")
	}
	if len([]rune(out.username)) > 64 {
		return out, gerror.New("用户名长度不能超过 64 个字符")
	}
	if out.password == "" {
		return out, gerror.New("密码不能为空")
	}
	if err := (secsvc.New()).ValidatePassword(ctx, out.password); err != nil {
		return out, gerror.New("密码不符合安全策略: " + err.Error())
	}
	switch strings.ToLower(get(importColRole)) {
	case "admin", "管理员":
		out.role = consts.RoleAdmin
	case "user", "用户", "普通用户":
		out.role = consts.RoleUser
	case "":
		return out, gerror.New("角色不能为空(admin 或 user)")
	default:
		return out, gerror.New("角色非法: 仅支持 admin 或 user")
	}
	switch strings.ToLower(get(importColStatus)) {
	case "", "1", "启用", "正常":
		out.status = 1
	case "0", "禁用", "停用":
		out.status = 0
	default:
		return out, gerror.New("状态非法: 仅支持 1/0 或 启用/禁用")
	}
	// 最大设备数: 仅超管可填; 管理员模板不含该列, 若手工添加并填值必须拒绝(不静默忽略)
	if raw := get(importColMaxDevices); raw != "" {
		if !actor.IsSuperAdmin() {
			return out, gerror.New("仅超级管理员可设置最大在线设备数")
		}
		n, convErr := strconv.Atoi(raw)
		if convErr != nil || n < 0 {
			return out, gerror.New("最大设备数必须为非负整数")
		}
		out.maxDev = &n
	}
	// 功能角色: 编码或名称, 逗号分隔; 不存在或超出可分配范围一律拒绝
	rawRoles := get(importColRoleCodes)
	if rawRoles != "" {
		for _, part := range strings.Split(rawRoles, ",") {
			name := strings.TrimSpace(part)
			if name == "" {
				continue
			}
			id, ok := roleMap[strings.ToLower(name)]
			if !ok {
				id, ok = roleMap[name]
			}
			if !ok {
				return out, gerror.New("功能角色不存在或无权限分配: " + name)
			}
			out.roleIds = append(out.roleIds, id)
		}
	}
	return out, nil
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
