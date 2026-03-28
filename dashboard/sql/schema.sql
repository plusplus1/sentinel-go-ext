-- ============================================
-- Sentinel Dashboard MySQL Schema
-- 全量建表脚本，可安全重复执行（CREATE IF NOT EXISTS）
--
-- 14 张表：
--   业务数据: business_line_app_groups, business_line_resources,
--            business_line_resource_flowrules, business_line_resource_circuitbreakerrules
--   发布记录: publish_records, publish_versions
--   组织架构: business_lines, business_line_apps, business_line_admins, business_line_members
--   用户权限: users, user_permissions, user_audit_logs, user_tokens
-- ============================================

CREATE DATABASE IF NOT EXISTS sentinel_dashboard
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE sentinel_dashboard;

-- ============================================
-- 1. 组织架构
-- ============================================

CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL UNIQUE COMMENT '用户唯一ID',
    email VARCHAR(256) NOT NULL UNIQUE COMMENT '公司邮箱（登录账号）',
    name VARCHAR(128) NOT NULL COMMENT '显示名称',
    avatar_url VARCHAR(512) COMMENT '头像URL',
    password_hash VARCHAR(256) COMMENT '密码哈希（bcrypt，飞书SSO登录则为空）',
    role VARCHAR(32) NOT NULL DEFAULT 'member' COMMENT '全局角色: super_admin/line_admin/member',
    feishu_user_id VARCHAR(64) COMMENT '飞书用户ID',
    status VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT '状态: active/disabled',
    last_login_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

CREATE TABLE IF NOT EXISTS business_lines (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL UNIQUE COMMENT '业务线名称',
    description VARCHAR(512) COMMENT '描述',
    status VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT '状态(active/deleted)',
    owner_id VARCHAR(64) COMMENT '负责人ID（兼容旧数据，新数据用 business_line_admins）',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务线表';

CREATE TABLE IF NOT EXISTS business_line_apps (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    business_line_id BIGINT UNSIGNED NOT NULL,
    app_key VARCHAR(64) NOT NULL COMMENT '应用标识（英文、数字、下划线）',
    description VARCHAR(512) COMMENT '应用描述',
    settings TEXT COMMENT '应用配置(JSON格式，含etcd地址等，如 {"url":"etcd://host:2379"})',
    status VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT '状态(active/deleted)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_line_app (business_line_id, app_key),
    KEY idx_business_line_id (business_line_id),
    KEY idx_app_key (app_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务线-App关联表';

CREATE TABLE IF NOT EXISTS business_line_admins (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    business_line_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL COMMENT '管理员用户ID(users.id)',
    added_by BIGINT UNSIGNED COMMENT '添加人ID(users.id)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_line_user (business_line_id, user_id),
    KEY idx_business_line_id (business_line_id),
    KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务线管理员关联表（多对多）';

CREATE TABLE IF NOT EXISTS business_line_members (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    business_line_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL COMMENT '成员用户ID(users.id)',
    added_by BIGINT UNSIGNED COMMENT '添加人ID(users.id)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_line_user (business_line_id, user_id),
    KEY idx_business_line_id (business_line_id),
    KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务线成员关联表（多对多）';

-- ============================================
-- 2. 业务数据
-- ============================================

CREATE TABLE IF NOT EXISTS business_line_app_groups (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID(business_line_apps.id)',
    name VARCHAR(128) NOT NULL COMMENT '模块名称',
    description VARCHAR(512) DEFAULT '' COMMENT '模块描述',
    is_default TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否默认模块',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_name (app_id, name),
    KEY idx_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='业务模块表';

CREATE TABLE IF NOT EXISTS business_line_resources (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID(business_line_apps.id)',
    name VARCHAR(256) NOT NULL COMMENT '资源名称',
    description VARCHAR(512) DEFAULT '' COMMENT '资源描述',
    group_id BIGINT UNSIGNED COMMENT '所属模块ID(business_line_app_groups.id)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_name (app_id, name),
    KEY idx_group_id (group_id),
    KEY idx_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源表';

CREATE TABLE IF NOT EXISTS business_line_resource_flowrules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID(business_line_apps.id)',
    resource_id BIGINT UNSIGNED NOT NULL COMMENT '资源ID(business_line_resources.id)',
    threshold DOUBLE NOT NULL DEFAULT 0 COMMENT '阈值',
    metric_type INT NOT NULL DEFAULT 0 COMMENT '指标类型(0=并发数,1=QPS)',
    control_behavior INT NOT NULL DEFAULT 0 COMMENT '控制行为(0=拒绝,1=WarmUp,2=排队)',
    warm_up_period_sec INT NOT NULL DEFAULT 0 COMMENT '预热时长(秒)',
    max_queueing_time_ms INT NOT NULL DEFAULT 0 COMMENT '最大排队时间(ms)',
    cluster_mode TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否集群模式',
    cluster_config JSON COMMENT '集群配置',
    token_calculate_strategy INT NOT NULL DEFAULT 0 COMMENT 'Token计算策略',
    relation_strategy INT NOT NULL DEFAULT 0 COMMENT '关联策略',
    ref_resource VARCHAR(256) DEFAULT '' COMMENT '关联资源',
    warm_up_cold_factor INT NOT NULL DEFAULT 3 COMMENT '预热冷启动因子',
    stat_interval_ms INT NOT NULL DEFAULT 1000 COMMENT '统计窗口(ms)',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_resource (app_id, resource_id),
    KEY idx_app_id (app_id),
    KEY idx_resource_id (resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流控规则表（每资源最多一条）';

CREATE TABLE IF NOT EXISTS business_line_resource_circuitbreakerrules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID(business_line_apps.id)',
    resource_id BIGINT UNSIGNED NOT NULL COMMENT '资源ID(business_line_resources.id)',
    strategy INT NOT NULL DEFAULT 0 COMMENT '熔断策略(0=慢调用比例,1=错误比例,2=错误计数)',
    threshold DOUBLE NOT NULL DEFAULT 0 COMMENT '阈值',
    retry_timeout_ms BIGINT NOT NULL DEFAULT 0 COMMENT '重试超时(ms)',
    min_request_amount INT NOT NULL DEFAULT 5 COMMENT '最小请求数',
    stat_interval_ms INT NOT NULL DEFAULT 1000 COMMENT '统计窗口(ms)',
    stat_sliding_window_bucket_count INT NOT NULL DEFAULT 1 COMMENT '滑动窗口桶数',
    max_allowed_rt_ms BIGINT NOT NULL DEFAULT 200 COMMENT '最大允许RT(ms)',
    probe_num INT NOT NULL DEFAULT 0 COMMENT '探测数量',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_resource (app_id, resource_id),
    KEY idx_app_id (app_id),
    KEY idx_resource_id (resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='熔断规则表（每资源最多一条）';

-- ============================================
-- 3. 发布与版本
-- ============================================

CREATE TABLE IF NOT EXISTS publish_records (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID(business_line_apps.id)',
    rule_type VARCHAR(32) NOT NULL COMMENT '规则类型(flow/circuitbreaker/all)',
    resource VARCHAR(256) DEFAULT '' COMMENT '资源名称(为空表示全量发布)',
    rule_count INT NOT NULL DEFAULT 0 COMMENT '发布规则数量',
    status VARCHAR(16) NOT NULL DEFAULT 'success' COMMENT '发布状态(success/failed)',
    error_msg TEXT COMMENT '错误信息',
    operator VARCHAR(64) DEFAULT '' COMMENT '操作人',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_app_id (app_id),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布记录表';

CREATE TABLE IF NOT EXISTS publish_versions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID(business_line_apps.id)',
    version_number INT NOT NULL DEFAULT 1 COMMENT '版本号(app维度自增)',
    description VARCHAR(512) DEFAULT '' COMMENT '发布描述',
    operator VARCHAR(64) DEFAULT '' COMMENT '操作人',
    rule_count INT NOT NULL DEFAULT 0 COMMENT '规则数量',
    snapshot JSON NOT NULL COMMENT '完整规则快照(JSON: {flow_rules:[], circuit_breaker_rules:[]})',
    status VARCHAR(16) NOT NULL DEFAULT 'success' COMMENT '状态(success/failed)',
    error_msg TEXT COMMENT '错误信息',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_version (app_id, version_number),
    KEY idx_app_id (app_id),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='发布版本表（快照存储，回滚=新版本）';

-- ============================================
-- 4. 用户权限与安全
-- ============================================

CREATE TABLE IF NOT EXISTS user_permissions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    resource_type VARCHAR(32) NOT NULL COMMENT '资源类型: business_line/app/module/group',
    resource_id VARCHAR(128) NOT NULL COMMENT '资源ID',
    role VARCHAR(32) NOT NULL COMMENT '角色: admin/member/viewer/owner',
    granted_by VARCHAR(64) COMMENT '授权人ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_resource (user_id, resource_type, resource_id),
    KEY idx_user_id (user_id),
    KEY idx_resource (resource_type, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户权限表';

CREATE TABLE IF NOT EXISTS user_audit_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL COMMENT '操作类型: rule_create/rule_update/rule_delete/rule_toggle/publish/rollback/resource_delete',
    resource_type VARCHAR(32) NOT NULL COMMENT '资源类型: flow_rule/cb_rule/rules/version/resource',
    resource_id VARCHAR(128) NOT NULL COMMENT '资源ID',
    detail VARCHAR(512) COMMENT '操作详情',
    ip_address VARCHAR(64) COMMENT 'IP地址',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_user_id (user_id),
    KEY idx_action (action),
    KEY idx_created_at (created_at),
    KEY idx_resource (resource_type, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户审计日志表';

CREATE TABLE IF NOT EXISTS user_tokens (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    token VARCHAR(256) NOT NULL COMMENT '会话Token（64位hex）',
    expires_at TIMESTAMP NULL COMMENT '过期时间',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_token (token),
    KEY idx_user_id (user_id),
    KEY idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户会话Token表（SessionStore持久化）';
