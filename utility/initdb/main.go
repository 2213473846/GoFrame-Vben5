// 一次性初始化脚本：创建数据库与数据表。
// 用法: go run ./utility/initdb
package main

import (
	"fmt"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gstr"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	ctx := gctx.GetInitCtx()

	// 创建数据库
	local, err := gdb.New(gdb.ConfigNode{
		Link: "mysql:root:123456@tcp(127.0.0.1:3306)/mysql",
	})
	if err != nil {
		panic(err)
	}
	defer local.Close(ctx)
	if _, err = local.Exec(ctx, "CREATE DATABASE IF NOT EXISTS `vben_admin` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		panic(err)
	}
	// 建库建表: 逐条执行 create.sql 中的语句
	db, err := gdb.New(gdb.ConfigNode{
		Link: "mysql:root:123456@tcp(127.0.0.1:3306)/mysql",
	})
	if err != nil {
		panic(err)
	}
	defer db.Close(ctx)
	for _, sql := range gstr.Split(gfile.GetContents("manifest/sql/create.sql"), ";") {
		sql = gstr.Trim(sql)
		if sql == "" || gstr.HasPrefix(sql, "--") {
			continue
		}
		if _, err = db.Exec(ctx, sql); err != nil {
			panic(fmt.Errorf("%s: %w", gstr.StrLimit(sql, 80, "..."), err))
		}
	}

	// 种子数据(幂等): admin/agent01/user01, 密码均为 123456
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	seeds := []string{
		fmt.Sprintf("INSERT IGNORE INTO `sys_user` (`id`,`pid`,`username`,`password`,`real_name`,`role`,`status`,`remark`) VALUES (1,0,'admin','%s','超级管理员','super_admin',1,'系统内置'),(2,1,'agent01','%s','管理员一号','admin',1,'演示管理员'),(3,2,'user01','%s','用户一号','user',1,'演示用户')", hash, hash, hash),
	}
	for _, sql := range seeds {
		if _, err = db.Exec(ctx, sql); err != nil {
			panic(err)
		}
	}

	count, _ := db.Model("sys_user").Ctx(ctx).Count()
	fmt.Println("init ok, sys_user rows:", count)
}
