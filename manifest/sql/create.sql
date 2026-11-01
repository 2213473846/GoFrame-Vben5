-- ==============================================
-- vben_admin 数据库初始化脚本
-- 执行: mysql -uroot -p123456 < manifest/sql/create.sql
-- ==============================================

CREATE DATABASE IF NOT EXISTS `vben_admin` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;

USE `vben_admin`;

CREATE TABLE IF NOT EXISTS `sys_user` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '用户ID',
    `pid`        bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '父级管理员ID, 0表示顶级',
    `username`   varchar(64)  NOT NULL COMMENT '登录用户名',
    `password`   varchar(100) NOT NULL COMMENT '密码(bcrypt)',
    `real_name`  varchar(64)  NOT NULL DEFAULT '' COMMENT '昵称',
    `avatar`     varchar(255) NOT NULL DEFAULT '' COMMENT '头像地址',
    `role`       varchar(32)  NOT NULL DEFAULT 'user' COMMENT '角色: super_admin/admin/user',
    `status`     tinyint(4)   NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=禁用',
    `remark`     varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
    `created_at` datetime DEFAULT NULL COMMENT '创建时间',
    `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)',
    `deleted_by` bigint(20) unsigned NULL DEFAULT NULL COMMENT '删除操作人ID',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_username_deleted` (`username`, `deleted_at`),
    KEY `idx_pid` (`pid`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '系统用户表(含层级关系)';

CREATE TABLE IF NOT EXISTS `sys_menu` (
    `id`          bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '菜单ID',
    `pid`         bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '父级ID, 0为顶级',
    `name`        varchar(64)  NOT NULL COMMENT '路由名称(vue-router name, 全局唯一)',
    `path`        varchar(128) NOT NULL COMMENT '路由路径, 顶级以/开头, 子级相对路径',
    `component`   varchar(255) NOT NULL DEFAULT '' COMMENT '组件路径, 目录填BasicLayout',
    `redirect`    varchar(255) NOT NULL DEFAULT '' COMMENT '重定向地址',
    `type`        varchar(16)  NOT NULL DEFAULT 'menu' COMMENT '类型: catalog=目录 menu=菜单',
    `title`       varchar(64)  NOT NULL COMMENT '菜单标题',
    `icon`        varchar(64)  NOT NULL DEFAULT '' COMMENT '图标',
    `sort`        int(11)      NOT NULL DEFAULT 0 COMMENT '排序(升序)',
    `status`      tinyint(4)   NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用',
    `hide_in_menu` tinyint(4)  NOT NULL DEFAULT 0 COMMENT '是否隐藏菜单',
    `keep_alive`  tinyint(4)   NOT NULL DEFAULT 0 COMMENT '是否缓存页面',
    `affix_tab`   tinyint(4)   NOT NULL DEFAULT 0 COMMENT '是否固定标签页',
    `is_public`   tinyint(4)   NOT NULL DEFAULT 0 COMMENT '是否公共菜单(所有登录用户可见, 无需授权)',
    `remark`      varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
    `created_by`  bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `created_at`  datetime DEFAULT NULL COMMENT '创建时间',
    `updated_at`  datetime DEFAULT NULL COMMENT '更新时间',
    `deleted_at`  datetime DEFAULT NULL COMMENT '删除时间(软删除)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name_deleted` (`name`, `deleted_at`),
    KEY `idx_pid` (`pid`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '系统菜单/路由表';

CREATE TABLE IF NOT EXISTS `sys_user_menu` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `user_id`    bigint(20) unsigned NOT NULL COMMENT '用户ID',
    `menu_id`    bigint(20) unsigned NOT NULL COMMENT '菜单ID',
    `created_at` datetime DEFAULT NULL COMMENT '授权时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_menu` (`user_id`, `menu_id`),
    KEY `idx_menu` (`menu_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '用户菜单授权表';

-- ----------------------------
-- 登录日志表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_login_log` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `user_id`    bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '用户ID, 登录失败且用户不存在时为0',
    `username`   varchar(64) NOT NULL DEFAULT '' COMMENT '登录用户名',
    `ip`         varchar(64) NOT NULL DEFAULT '' COMMENT '登录地址(IP)',
    `location`   varchar(128) NOT NULL DEFAULT '' COMMENT '登录地点',
    `browser`    varchar(64) NOT NULL DEFAULT '' COMMENT '浏览器',
    `os`         varchar(64) NOT NULL DEFAULT '' COMMENT '操作系统',
    `status`     tinyint(4) NOT NULL DEFAULT 1 COMMENT '登录状态: 1=成功 0=失败',
    `message`    varchar(255) NOT NULL DEFAULT '' COMMENT '登录信息',
    `created_at` datetime DEFAULT NULL COMMENT '登录时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_created_at` (`created_at`),
    KEY `idx_username` (`username`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '登录日志表';

-- ----------------------------
-- 操作日志表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_operation_log` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `user_id`    bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '用户ID',
    `username`   varchar(64) NOT NULL DEFAULT '' COMMENT '用户名',
    `module`     varchar(64) NOT NULL DEFAULT '' COMMENT '操作模块',
    `action`     varchar(64) NOT NULL DEFAULT '' COMMENT '操作类型',
    `method`     varchar(16) NOT NULL DEFAULT '' COMMENT '请求方法',
    `path`       varchar(255) NOT NULL DEFAULT '' COMMENT '请求地址',
    `ip`         varchar(64) NOT NULL DEFAULT '' COMMENT '操作地址(IP)',
    `location`   varchar(128) NOT NULL DEFAULT '' COMMENT '操作地点',
    `browser`    varchar(64) NOT NULL DEFAULT '' COMMENT '浏览器',
    `os`         varchar(64) NOT NULL DEFAULT '' COMMENT '操作系统',
    `status`     tinyint(4) NOT NULL DEFAULT 1 COMMENT '操作状态: 1=成功 0=失败',
    `message`    varchar(255) NOT NULL DEFAULT '' COMMENT '操作信息',
    `created_at` datetime DEFAULT NULL COMMENT '操作时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_created_at` (`created_at`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '操作日志表';
