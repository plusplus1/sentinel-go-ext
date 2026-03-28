# 架构设计文档：资源分组与资源中心视图（v2 - MySQL 架构）

## 1. 系统架构概览

### 1.1 整体架构图
```
┌─────────────────────────────────────────────────────────────────────────┐
│                              客户端层                                    │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                  React + TypeScript + Ant Design                │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │   │
│  │  │ 资源中心  │  │ 模块管理  │  │ 规则编辑  │  │ 发布管理  │       │   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                      │                                  │
│                                      ▼                                  │
│                              HTTP/HTTPS 请求                            │
└─────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              应用层 (API Layer)                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                       Go + Gin Web 框架                          │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │   │
│  │  │ 模块 API    │  │ 资源 API    │  │ 规则 API    │             │   │
│  │  │ /api/groups │  │/api/resource│  │/api/rule/*  │             │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘             │   │
│  │  ┌─────────────┐                                              │   │
│  │  │ 发布 API    │                                              │   │
│  │  │/api/publish │                                              │   │
│  │  └─────────────┘                                              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    ▼                                   ▼
┌──────────────────────────────┐    ┌──────────────────────────────────┐
│      MySQL 存储层（主）       │    │      etcd 存储层（发布目标）      │
│                              │    │                                  │
│  - 模块配置                  │    │  - Sentinel 客户端消费规则       │
│  - 资源元数据                │───▶│  - 运行时规则推送                │
│  - 流控规则                  │    │  - 客户端心跳注册                │
│  - 熔断规则                  │    │                                  │
│  - 发布记录                  │    │                                  │
│                              │    │                                  │
│  所有 CRUD 操作 → MySQL      │    │  仅发布操作 → etcd              │
└──────────────────────────────┘    └──────────────────────────────────┘
```

### 1.2 核心架构原则

1. **MySQL 为主存储**：所有配置（模块、资源、规则）存储在 MySQL
2. **etcd 为发布目标**：仅在配置发布时同步到 etcd，供 Sentinel 客户端消费
3. **API 接口不变**：保持与原 etcd 架构完全兼容的 API 接口
4. **发布即生效**：配置修改后需发布才能推送到客户端

---

## 2. 数据库设计

### 2.1 库表结构总览
```sql
sentinel_dashboard/
├── apps                  -- 应用表
├── groups                -- 业务模块表
├── resources             -- 资源表
├── flow_rules            -- 流控规则表
├── circuit_breaker_rules -- 熔断规则表
├── system_rules          -- 系统规则表（扩展）
├── hotspot_rules         -- 热点参数规则表（扩展）
└── publish_records       -- 发布记录表
```

### 2.2 核心表设计

#### apps（应用表）
```sql
CREATE TABLE apps (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_key VARCHAR(64) NOT NULL UNIQUE COMMENT '应用唯一标识(hash)',
    name VARCHAR(128) NOT NULL COMMENT '应用名称',
    type VARCHAR(32) NOT NULL DEFAULT 'etcd' COMMENT '数据源类型',
    description VARCHAR(512) DEFAULT '',
    settings JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

#### groups（业务模块表）
```sql
CREATE TABLE `groups` (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_key VARCHAR(64) NOT NULL COMMENT '应用标识',
    env VARCHAR(32) NOT NULL DEFAULT 'prod' COMMENT '环境',
    name VARCHAR(128) NOT NULL COMMENT '模块名称',
    description VARCHAR(512) DEFAULT '',
    is_default TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否默认模块',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_env_name (app_key, env, name),
    KEY idx_app_env (app_key, env)
);
```

#### resources（资源表）
```sql
CREATE TABLE resources (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_key VARCHAR(64) NOT NULL,
    env VARCHAR(32) NOT NULL DEFAULT 'prod',
    name VARCHAR(256) NOT NULL COMMENT '资源名称',
    group_id BIGINT UNSIGNED COMMENT '所属模块ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_env_name (app_key, env, name),
    KEY idx_group_id (group_id)
);
```

#### flow_rules（流控规则表）
```sql
CREATE TABLE flow_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    rule_id VARCHAR(128) NOT NULL UNIQUE COMMENT '规则唯一标识',
    app_key VARCHAR(64) NOT NULL,
    env VARCHAR(32) NOT NULL DEFAULT 'prod',
    resource VARCHAR(256) NOT NULL,
    threshold DOUBLE NOT NULL DEFAULT 0,
    metric_type INT NOT NULL DEFAULT 0 COMMENT '0=并发数,1=QPS',
    control_behavior INT NOT NULL DEFAULT 0 COMMENT '0=拒绝,1=WarmUp,2=排队',
    warm_up_period_sec INT NOT NULL DEFAULT 0,
    max_queueing_time_ms INT NOT NULL DEFAULT 0,
    cluster_mode TINYINT(1) NOT NULL DEFAULT 0,
    cluster_config JSON,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_app_env_resource (app_key, env, resource)
);
```

#### circuit_breaker_rules（熔断规则表）
```sql
CREATE TABLE circuit_breaker_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    rule_id VARCHAR(128) NOT NULL UNIQUE,
    app_key VARCHAR(64) NOT NULL,
    env VARCHAR(32) NOT NULL DEFAULT 'prod',
    resource VARCHAR(256) NOT NULL,
    strategy INT NOT NULL DEFAULT 0 COMMENT '0=慢调用比例,1=错误比例,2=错误计数',
    threshold DOUBLE NOT NULL DEFAULT 0,
    retry_timeout_ms BIGINT NOT NULL DEFAULT 0,
    min_request_amount INT NOT NULL DEFAULT 5,
    stat_interval_ms INT NOT NULL DEFAULT 1000,
    max_allowed_rt_ms BIGINT NOT NULL DEFAULT 200,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_app_env_resource (app_key, env, resource)
);
```

---

## 3. 数据流向

### 3.1 配置编辑流程
```
用户操作 → 前端 → API → MySQL DAO → MySQL 存储
                                    ↓
                              返回最新数据
                                    ↓
                            前端更新展示
```

### 3.2 配置发布流程
```
用户点击发布 → POST /api/publish
                    ↓
            PublishService
                    ↓
        从 MySQL 读取规则（flow_rules + cb_rules）
                    ↓
        按资源分组，写入 etcd
                    ↓
        记录发布记录到 publish_records
                    ↓
        Sentinel 客户端从 etcd 消费规则
```

### 3.3 客户端消费流程
```
Sentinel 客户端 → 连接 etcd
                → 注册心跳
                → 监听规则变更
                → 拉取规则到本地
                → 应用流控/熔断逻辑
```

---

## 4. 技术栈

### 后端
- **语言**: Go 1.20
- **Web 框架**: Gin
- **MySQL 驱动**: go-sql-driver/mysql v1.8.1
- **etcd 客户端**: go.etcd.io/etcd/client/v3
- **JSON**: encoding/json (标准库)

### 前端
- **框架**: React 18 + TypeScript
- **构建工具**: Vite
- **UI 库**: Ant Design
- **路由**: React Router v6
- **HTTP 客户端**: Axios

### 存储
- **主存储**: MySQL 5.7
- **发布目标**: etcd v3
- **字符集**: utf8mb4

---

## 5. 目录结构

```
sentinel-go-ext/
├── dashboard/
│   ├── api/app/          # API handlers
│   │   ├── group.go      # 模块管理 API（MySQL）
│   │   ├── resource.go   # 资源管理 API（MySQL）
│   │   ├── rule_flow.go  # 流控规则 API
│   │   └── rule_circuitbreaker.go  # 熔断规则 API
│   ├── dao/
│   │   ├── group_dao.go  # etcd DAO（保留）
│   │   ├── resource_dao.go  # etcd DAO（保留）
│   │   └── mysql.go      # MySQL DAO（新增）
│   ├── model/
│   │   ├── group.go      # 模块模型
│   │   └── resource.go   # 资源模型
│   ├── service/
│   │   ├── group_service.go  # 模块服务
│   │   ├── resource_service.go  # 资源服务
│   │   └── publish_service.go   # 发布服务（新增）
│   ├── sql/
│   │   └── schema.sql    # 数据库 schema
│   └── install.go        # 路由注册
├── frontend/
│   └── src/
│       ├── pages/
│       │   ├── Groups.tsx     # 模块管理页面
│       │   ├── Resources.tsx  # 资源中心页面
│       │   ├── FlowRules.tsx  # 流控规则页面
│       │   └── CircuitBreakerRules.tsx  # 熔断规则页面
│       ├── context/
│       │   └── AppContext.tsx # 应用上下文
│       └── App.tsx            # 主应用 + 路由
└── source/etcd/          # etcd 连接（GetClient 暴露）
```

---

## 6. 与旧架构的对比

| 维度 | 旧架构（纯 etcd） | 新架构（MySQL + etcd） |
|------|-------------------|------------------------|
| 配置存储 | etcd | MySQL |
| 规则消费 | etcd（客户端直连） | etcd（客户端直连） |
| 数据一致性 | etcd 强一致 | MySQL 事务 + 发布同步 |
| 查询性能 | etcd 范围查询 | MySQL 索引查询 |
| 配置版本 | 无版本管理 | publish_records 记录 |
| 扩展性 | 受 etcd 限制 | MySQL 水平扩展 |
| API 接口 | 原始接口 | 完全兼容 |

---

## 7. 安全考虑

1. **MySQL 安全**：生产环境需配置密码、SSL 连接、IP 白名单
2. **API 认证**：保留现有 API 认证机制
3. **etcd 安全**：保留现有 etcd TLS 配置
4. **SQL 注入防护**：使用参数化查询（prepared statements）

---

## 8. 部署要求

### MySQL 部署
- MySQL 5.7+ 或 MySQL 8.0+
- 最小配置：2GB RAM，10GB 磁盘
- 字符集：utf8mb4
- 时区：Asia/Shanghai

### Dashboard 部署
- Go 1.20+
- 连接 MySQL 和 etcd
- 前端静态资源内置

---

## 更新日志
- 2026-03-14 08:56: 初版（基于 etcd 架构）
- 2026-03-14 12:25: v2 更新（MySQL 架构升级，保持 API 兼容）

---

## 9. 发布版本管理架构

### 9.1 设计目标
- 每次发布自动创建规则快照
- 支持版本历史查看
- 支持版本回滚
- 支持版本对比

### 9.2 数据模型
```
publish_versions
├── id (PK)
├── app_key
├── env  
├── version_number (自增: v1, v2, v3...)
├── description (发布描述)
├── operator (操作人)
├── rule_count (规则数量)
├── snapshot (JSON - 完整规则快照)
├── status (success/failed)
└── created_at

snapshot JSON 结构:
{
  "flow_rules": [
    {"rule_id": "...", "resource": "...", "threshold": 100, ...},
    ...
  ],
  "cb_rules": [
    {"rule_id": "...", "resource": "...", "strategy": 0, ...},
    ...
  ]
}
```

### 9.3 发布流程
```
用户点击发布
    ↓
从 MySQL 读取所有启用的规则
    ↓
序列化为 JSON 快照
    ↓
写入 publish_versions（新版本）
    ↓
同步规则到 etcd
    ↓
更新 publish_records（关联版本号）
    ↓
返回版本号给用户
```

### 9.4 回滚流程
```
用户选择回滚到版本 N
    ↓
从 publish_versions 读取版本 N 的快照
    ↓
清空 MySQL 中当前规则
    ↓
从快照恢复规则到 MySQL
    ↓
重新发布到 etcd
    ↓
创建新版本（标记为回滚操作）
```

### 9.5 API 设计
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/versions | 版本列表 |
| GET | /api/versions/:id | 版本详情 + 快照内容 |
| POST | /api/versions/:id/rollback | 回滚到指定版本 |
| POST | /api/publish | 发布（增强：自动创建版本）|

### 9.6 前端页面
- 版本历史时间线
- 版本详情弹窗（规则列表）
- 回滚确认弹窗
- 发布弹窗（增加版本描述）

