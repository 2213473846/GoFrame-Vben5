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
    `max_devices` int(11) NULL DEFAULT NULL COMMENT '最大同时在线设备数, NULL表示使用全局默认',
    `must_change_password` tinyint(4) NOT NULL DEFAULT 0 COMMENT '是否需强制修改密码: 1=是 0=否',
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
-- 功能角色表(RBAC)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_role` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '角色ID',
    `code`       varchar(64)  NOT NULL COMMENT '角色编码(全局唯一, 如 ops)',
    `name`       varchar(64)  NOT NULL COMMENT '角色名称(显示用)',
    `remark`     varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
    `status`     tinyint(4)   NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用',
    `created_by` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `created_at` datetime DEFAULT NULL COMMENT '创建时间',
    `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code_deleted` (`code`, `deleted_at`),
    KEY `idx_name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '功能角色表';

-- ----------------------------
-- 用户-角色关联表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_user_role` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `user_id`    bigint(20) unsigned NOT NULL COMMENT '用户ID',
    `role_id`    bigint(20) unsigned NOT NULL COMMENT '角色ID',
    `created_at` datetime DEFAULT NULL COMMENT '分配时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_role` (`user_id`, `role_id`),
    KEY `idx_role` (`role_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '用户角色关联表';

-- ----------------------------
-- 角色-菜单授权表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_role_menu` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `role_id`    bigint(20) unsigned NOT NULL COMMENT '角色ID',
    `menu_id`    bigint(20) unsigned NOT NULL COMMENT '菜单ID',
    `created_at` datetime DEFAULT NULL COMMENT '授权时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_role_menu` (`role_id`, `menu_id`),
    KEY `idx_menu` (`menu_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '角色菜单授权表';

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

-- ----------------------------
-- 数据字典类型表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_dict_type` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '字典类型ID',
    `type`       varchar(64)  NOT NULL COMMENT '字典类型编码(全局唯一)',
    `name`       varchar(64)  NOT NULL COMMENT '字典类型名称',
    `status`     tinyint(4)   NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用',
    `remark`     varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
    `created_by` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `created_at` datetime DEFAULT NULL COMMENT '创建时间',
    `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_type_deleted` (`type`, `deleted_at`),
    KEY `idx_name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '数据字典类型表';

-- ----------------------------
-- 数据字典项表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_dict_data` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '字典项ID',
    `dict_type`  varchar(64)  NOT NULL COMMENT '所属字典类型编码',
    `label`      varchar(64)  NOT NULL COMMENT '显示名称',
    `value`      varchar(64)  NOT NULL COMMENT '字典值',
    `sort`       int(11)      NOT NULL DEFAULT 0 COMMENT '排序(升序)',
    `status`     tinyint(4)   NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用',
    `tag_color`  varchar(32)  NOT NULL DEFAULT '' COMMENT '前端标签颜色: success/warning/danger/info',
    `remark`     varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
    `created_at` datetime DEFAULT NULL COMMENT '创建时间',
    `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间(软删除)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_dict_type_value_deleted` (`dict_type`, `value`, `deleted_at`),
    KEY `idx_dict_type` (`dict_type`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '数据字典项表';

-- ----------------------------
-- 系统参数配置表
-- 仅存放运行时可变的业务/展示参数(仪表盘趋势天数、列表条数、导入导出上限等);
-- 登录失败阈值、锁定窗口、验证码阈值、密码复杂度、会话上限等安全参数禁止放入本表,
-- 必须继续由 config.yaml 维护, 避免"可被管理员修改的值"反过来约束管理员自己的登录安全。
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_config` (
    `id`           bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '参数ID',
    `config_key`   varchar(128) NOT NULL COMMENT '参数键(唯一)',
    `config_value` text         NOT NULL COMMENT '参数值',
    `name`         varchar(64)  NOT NULL DEFAULT '' COMMENT '参数名称',
    `remark`       varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
    `status`       tinyint(4)   NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用',
    `builtin`      tinyint(4)   NOT NULL DEFAULT 0 COMMENT '是否内置: 1=内置(不可删除, key不可修改)',
    `created_by`   bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '创建人ID',
    `created_at`   datetime DEFAULT NULL COMMENT '创建时间',
    `updated_at`   datetime DEFAULT NULL COMMENT '更新时间',
    `deleted_at`   datetime DEFAULT NULL COMMENT '删除时间(软删除)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_config_key_deleted` (`config_key`, `deleted_at`),
    KEY `idx_name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '系统参数配置表';

-- ----------------------------
-- 定时任务表
-- invoke_target 为白名单标识(不是类名/命令), 仅允许注册表中的内置任务;
-- 禁止按字符串反射执行, 避免演变成远程命令执行入口。
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_job` (
    `id`                 bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '任务ID',
    `name`               varchar(64)  NOT NULL COMMENT '任务名称',
    `job_group`          varchar(64)  NOT NULL DEFAULT 'default' COMMENT '任务分组',
    `invoke_target`      varchar(128) NOT NULL COMMENT '白名单任务标识',
    `cron_expression`    varchar(64)  NOT NULL COMMENT 'cron 表达式(5 段或 6 段)',
    `status`             tinyint(4)   NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=停用',
    `concurrency_policy` tinyint(4)   NOT NULL DEFAULT 0 COMMENT '并发策略: 1=允许并发 0=禁止并发(上次未完成则跳过)',
    `timeout_seconds`    int(11)      NOT NULL DEFAULT 60 COMMENT '执行超时秒数, 0=不限制',
    `remark`             varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
    `last_run_at`        datetime DEFAULT NULL COMMENT '最近执行时间',
    `last_result`        varchar(128) NOT NULL DEFAULT '' COMMENT '最近执行结果文案',
    `next_run_at`        datetime DEFAULT NULL COMMENT '下次执行时间(gcron 未暴露计算接口时留空)',
    `created_at`         datetime DEFAULT NULL COMMENT '创建时间',
    `updated_at`         datetime DEFAULT NULL COMMENT '更新时间',
    `deleted_at`         datetime DEFAULT NULL COMMENT '删除时间(软删除)',
    PRIMARY KEY (`id`),
    KEY `idx_name` (`name`),
    KEY `idx_status` (`status`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '定时任务表';

-- ----------------------------
-- 定时任务执行日志表
-- message 同时承载执行结果摘要(成功时记录处理条数)与失败原因。
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_job_log` (
    `id`            bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `job_id`        bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '任务ID',
    `job_name`      varchar(64)  NOT NULL DEFAULT '' COMMENT '任务名称(冗余, 任务删除后仍可追溯)',
    `trigger_type`  tinyint(4)   NOT NULL DEFAULT 1 COMMENT '触发方式: 1=定时 2=手动',
    `started_at`    datetime DEFAULT NULL COMMENT '开始时间',
    `finished_at`   datetime DEFAULT NULL COMMENT '结束时间',
    `duration_ms`   bigint(20)   NOT NULL DEFAULT 0 COMMENT '执行耗时(毫秒)',
    `status`        tinyint(4)   NOT NULL DEFAULT 1 COMMENT '执行状态: 1=成功 0=失败',
    `message`      text         NULL COMMENT '执行结果摘要/失败原因',
    `created_at`    datetime DEFAULT NULL COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_job_id` (`job_id`),
    KEY `idx_started_at` (`started_at`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '定时任务执行日志表';

-- ----------------------------
-- 通知公告表
-- 内容只存纯文本(入库前剥离 HTML 标签), 前端必须使用文本插值渲染, 禁止 v-html。
-- target_scope: 1=全员可见 2=仅 sys_notice_user 中的指定用户可见。
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_notice` (
    `id`           bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '公告ID',
    `title`        varchar(128) NOT NULL COMMENT '标题',
    `content`      text         NOT NULL COMMENT '正文(纯文本, 不含HTML)',
    `notice_type`  varchar(32)  NOT NULL DEFAULT '' COMMENT '类型(字典 notice_type 的值)',
    `notice_level` varchar(32)  NOT NULL DEFAULT '' COMMENT '级别(字典 notice_level 的值)',
    `status`       tinyint(4)   NOT NULL DEFAULT 0 COMMENT '状态: 0=草稿 1=已发布 2=已撤回',
    `target_scope` tinyint(4)   NOT NULL DEFAULT 1 COMMENT '可见范围: 1=全员 2=指定用户',
    `publisher_id` bigint(20) unsigned NOT NULL DEFAULT 0 COMMENT '发布人ID',
    `publish_at`   datetime DEFAULT NULL COMMENT '发布时间',
    `expire_at`    datetime DEFAULT NULL COMMENT '失效时间, 空表示永不过期',
    `created_at`   datetime DEFAULT NULL COMMENT '创建时间',
    `updated_at`   datetime DEFAULT NULL COMMENT '更新时间',
    `deleted_at`   datetime DEFAULT NULL COMMENT '删除时间(软删除)',
    PRIMARY KEY (`id`),
    KEY `idx_status` (`status`, `deleted_at`),
    KEY `idx_publish_at` (`publish_at`),
    KEY `idx_publisher` (`publisher_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '通知公告表';

-- ----------------------------
-- 公告定向用户表(仅 target_scope=2 时写入)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_notice_user` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `notice_id`  bigint(20) unsigned NOT NULL COMMENT '公告ID',
    `user_id`    bigint(20) unsigned NOT NULL COMMENT '接收用户ID',
    `created_at` datetime DEFAULT NULL COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_notice_user` (`notice_id`, `user_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '公告定向用户表';

-- ----------------------------
-- 公告阅读记录表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `sys_notice_read` (
    `id`         bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
    `notice_id`  bigint(20) unsigned NOT NULL COMMENT '公告ID',
    `user_id`    bigint(20) unsigned NOT NULL COMMENT '用户ID',
    `read_at`    datetime DEFAULT NULL COMMENT '阅读时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_notice_user` (`notice_id`, `user_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '公告阅读记录表';
