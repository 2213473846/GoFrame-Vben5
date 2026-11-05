package boot

import (
	"context"

	"demo/internal/consts"
	"demo/internal/dao"
	"demo/internal/model/do"
	dictsvc "demo/internal/service/dict"
	jobsvc "demo/internal/service/job"
	menusvc "demo/internal/service/menu"
	sysconfigsvc "demo/internal/service/sysconfig"

	"github.com/gogf/gf/v2/errors/gerror"
	"golang.org/x/crypto/bcrypt"
)

// Run 服务启动前的初始化(幂等): 确保种子账号与内置菜单存在
// 种子账号: admin/123456(超管), agent01/123456(管理员), user01/123456(用户)
func Run(ctx context.Context) (err error) {
	if err = seedUsers(ctx); err != nil {
		return err
	}
	if err = seedMenus(ctx); err != nil {
		return err
	}
	// 日志管理菜单独立幂等 seed(存量数据库也能增量生效)
	if err = menusvc.New().SeedLogs(ctx); err != nil {
		return err
	}
	// 角色管理菜单独立幂等 seed
	if err = menusvc.New().SeedRoleMenu(ctx); err != nil {
		return err
	}
	// 字典管理菜单与内置字典数据独立幂等 seed
	if err = menusvc.New().SeedDict(ctx); err != nil {
		return err
	}
	if err = dictsvc.New().Seed(ctx); err != nil {
		return err
	}
	// 系统参数配置: 菜单与内置参数独立幂等 seed
	if err = menusvc.New().SeedConfig(ctx); err != nil {
		return err
	}
	if err = sysconfigsvc.New().Seed(ctx); err != nil {
		return err
	}
	// 工作台菜单独立幂等 seed(只下发给管理员)
	if err = menusvc.New().SeedDashboard(ctx); err != nil {
		return err
	}
	// 系统监控菜单独立幂等 seed
	if err = menusvc.New().SeedMonitor(ctx); err != nil {
		return err
	}
	// 系统监控子菜单: 服务监控 / 定时任务(幂等 + 对存量管理员授权)
	if err = menusvc.New().SeedMonitorServer(ctx); err != nil {
		return err
	}
	if err = menusvc.New().SeedMonitorJob(ctx); err != nil {
		return err
	}
	// 通知公告菜单(系统管理下)与顶级「我的通知」菜单独立幂等 seed
	if err = menusvc.New().SeedNotice(ctx); err != nil {
		return err
	}
	if err = menusvc.New().SeedMyNotice(ctx); err != nil {
		return err
	}
	// 内置定时任务(clearlog)幂等插入(默认停用)并加载启用任务到调度器
	if err = jobsvc.New().Seed(ctx); err != nil {
		return err
	}
	return jobsvc.New().Start(ctx)
}

// seedMenus 初始化内置菜单; 首次初始化时给存量非超管用户下发默认菜单
func seedMenus(ctx context.Context) (err error) {
	menuCount, err := dao.SysMenu.Ctx(ctx).Count()
	if err != nil {
		return gerror.Wrap(err, "初始化菜单失败: 数据库不可用, 请先执行 go run ./utility/migratemenu")
	}
	if menuCount > 0 {
		return nil
	}
	if err = menusvc.New().Seed(ctx); err != nil {
		return err
	}
	// 仅首次初始化菜单时下发默认授权, 之后由超管在界面上管理
	return menusvc.New().SeedGrants(ctx)
}

// seedUsers 初始化种子账号
func seedUsers(ctx context.Context) (err error) {
	count, err := dao.SysUser.Ctx(ctx).Count()
	if err != nil {
		return gerror.Wrap(err, "初始化失败: 数据库不可用, 请先执行 go run ./utility/initdb")
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return gerror.Wrap(err, "密码加密失败")
	}
	adminId, err := dao.SysUser.Ctx(ctx).Data(do.SysUser{
		Pid:      0,
		Username: "admin",
		Password: string(hash),
		RealName: "超级管理员",
		Role:     consts.RoleSuperAdmin,
		Status:   1,
		Remark:   "系统内置",
	}).InsertAndGetId()
	if err != nil {
		return gerror.Wrap(err, "初始化种子账号失败")
	}
	agentId, err := dao.SysUser.Ctx(ctx).Data(do.SysUser{
		Pid:      adminId,
		Username: "agent01",
		Password: string(hash),
		RealName: "代理一号",
		Role:     consts.RoleAdmin,
		Status:   1,
		Remark:   "演示代理",
	}).InsertAndGetId()
	if err != nil {
		return gerror.Wrap(err, "初始化种子账号失败")
	}
	if _, err = dao.SysUser.Ctx(ctx).Data(do.SysUser{
		Pid:      agentId,
		Username: "user01",
		Password: string(hash),
		RealName: "用户一号",
		Role:     consts.RoleUser,
		Status:   1,
		Remark:   "演示用户",
	}).InsertAndGetId(); err != nil {
		return gerror.Wrap(err, "初始化种子账号失败")
	}
	return nil
}
