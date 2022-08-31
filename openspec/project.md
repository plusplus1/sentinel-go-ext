# 流云卫士 — Sentinel Dashboard 项目规范

## 项目概述

**流云卫士**是 Sentinel-Go 的扩展项目，提供现代化的 Web 控制台用于流量控制和熔断规则管理。采用 MySQL 作为配置存储，etcd 作为运行时规则推送目标。

### 核心能力
- **服务模块管理**：按业务模块组织资源，批量管理规则
- **资源中心**：统一查看所有资源及其规则状态
- **规则管理**：流控规则、熔断规则的 CRUD 和发布（每资源一条流控规则 + 一条熔断规则，UNIQUE 约束）
- **用户权限**：超级管理员、业务线管理员、普通成员三级权限（✅ 已实现）
- **发布机制**：按资源维度发布到 etcd，支持发布前预览和 diff 对比（✅ 已实现）
- **版本管理**：规则版本快照与回滚，回滚等于一次新发布（✅ 已实现）

---

## 技术栈

### 后端
- **语言**: Go 1.20
- **Web 框架**: Gin v1.9.1
- **数据库**: MySQL 5.7（go-sql-driver/mysql v1.8.1）
- **运行时**: etcd v3（客户端 v3.5.11）
- **JSON**: Sonic v1.10.2
- **日志**: Uber Zap v1.26.0

### 前端
- **框架**: React 18 + TypeScript
- **构建**: Vite 8.0.0
- **UI**: Ant Design 5.x
- **路由**: React Router v6
- **HTTP**: Axios

### 存储架构
- **MySQL 5.7**：所有配置数据（模块、资源、规则、用户、权限）
- **etcd v3**：仅发布后的运行时规则（Sentinel 客户端消费）

---

## 项目结构

```
sentinel-go-ext/
├── dashboard/                  # 后端（全部代码在此）
│   ├── api/
│   │   ├── app/               # API handlers
│   │   │   ├── auth.go        # 认证 API
│   │   │   ├── auth_admin.go  # 超级管理员 API
│   │   │   ├── auth_line.go   # 业务线管理员 API
│   │   │   ├── group.go       # 模块管理 API（MySQL）
│   │   │   ├── resource.go    # 资源中心 API（MySQL）
│   │   │   ├── rule_flow.go   # 流控规则 API（MySQL）
│   │   │   └── rule_circuitbreaker.go
│   │   └── base/              # 基础工具、鉴权中间件
│   ├── cmd/                   # 命令行入口
│   ├── config/                # 配置管理
│   ├── dao/                   # 数据访问层
│   │   └── mysql.go           # MySQL DAO（Group, Resource, Rule, Publish）
│   ├── model/                 # 数据模型
│   │   ├── group.go           # 模块模型
│   │   ├── resource.go        # 资源模型
│   │   └── user.go            # 用户模型
│   ├── provider/              # 配置中心抽象层
│   │   ├── rule_provider.go   # RulePublisher / RulePathBuilder 接口
│   │   ├── etcd_publisher.go  # EtcdRulePublisher + EtcdClientManager
│   │   └── etcd_path_builder.go # EtcdPathBuilder
│   ├── service/               # 业务服务
│   │   ├── auth_service.go    # 认证服务（含 LogAudit）
│   │   ├── feishu_service.go  # 飞书SSO服务
│   │   ├── session_store.go   # SessionStore 接口
│   │   └── session_store_mysql.go # MySQL Session 实现
│   ├── source/                # Sentinel SDK imports
│   │   └── reg/               # 注册中心
│   ├── sql/                   # 数据库脚本
│   │   └── schema.sql         # 全量建表脚本（14 张表，可重复执行）
│   └── install.go             # 路由注册
├── frontend/                  # 前端 React 应用
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Resources.tsx   # 资源中心（规则、发布、版本、diff）
│   │   │   ├── Groups.tsx      # 服务模块管理
│   │   │   ├── Admin.tsx       # 管理中心（超管 + 审计日志）
│   │   │   └── Login.tsx       # 登录页（支持飞书SSO）
│   │   ├── context/
│   │   │   └── AppContext.tsx   # 应用状态
│   │   └── App.tsx             # 主应用 + 路由 + 布局
│   ├── public/
│   │   └── logo.png            # 品牌 Logo
│   └── dist/                   # 构建输出
├── openspec/                   # 设计文档
│   └── changes/
│       ├── admin-role-separation/        # 管理员角色分离
│       ├── resource-groups-and-mysql-migration/
│       └── user-permissions-system/      # 用户权限系统
└── conf/
    └── dashboard-settings.yaml # 配置文件
```

---

## 用户角色与权限

| 角色 | 标识 | 可见菜单 | 权限范围 | 状态 |
|------|------|---------|---------|------|
| 超级管理员 | `super_admin` | 管理中心 | 全局（用户、业务线、应用、权限） | ✅ 已实现 |
| 业务线管理员 | `line_admin` | 管理中心 | 自己管理的业务线和应用 | ✅ 已实现 |
| 普通成员 | `member` | 资源中心 + 服务模块 | 被授权的模块和资源 | ✅ 已实现 |

**说明**：
- 超级管理员 API 路径前缀：`/api/admin/`（用户管理、业务线 CRUD、管理员管理、权限管理）
- 业务线管理员 API 路径前缀：`/api/line-admin/`（业务线、应用、成员管理）
- 超管支持给业务线添加多个管理员（多对多）
- 线管支持给业务线添加普通成员，成员可访问资源中心和服务模块

### 测试账号
| 邮箱 | 密码 | 角色 |
|------|------|------|
| admin@test.com | admin123 | super_admin |
| line_admin@test.com | admin123 | line_admin |
| member1@test.com | member123 | member |

---

## API 清单

> 以 `dashboard/install.go` 为准（single source of truth）

### 公开 API（无需认证）
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/login | 密码登录 |
| POST | /api/auth/logout | 退出 |
| GET | /api/auth/feishu | 飞书 SSO 跳转 |
| GET | /api/auth/feishu/callback | 飞书 SSO 回调 |

### 认证后 API（需登录）
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/auth/me | 当前用户信息 |
| GET | /api/users/search | 搜索用户（所有已认证用户可用） |

### 规则 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/app/rule/flow/list | 流控规则列表 |
| POST | /api/app/rule/flow/update | 创建/更新流控规则 |
| POST | /api/app/rule/flow/del | 删除流控规则 |
| GET | /api/app/rule/circuitbreaker/list | 熔断规则列表 |
| POST | /api/app/rule/circuitbreaker/update | 创建/更新熔断规则 |
| POST | /api/app/rule/circuitbreaker/del | 删除熔断规则 |
| PUT | /api/rule/:type/:id/toggle | 规则启用/禁用开关（type=flow/circuitbreaker） |

### 模块管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/groups | 模块列表 |
| POST | /api/groups | 创建模块 |
| GET | /api/groups/:id | 模块详情 |
| PUT | /api/groups/:id | 编辑模块 |
| DELETE | /api/groups/:id | 删除模块 |
| GET | /api/groups/:id/members | 模块资源列表 |
| POST | /api/groups/:id/members | 添加资源到模块 |
| DELETE | /api/groups/:id/members/:resource | 从模块移除资源 |

### 资源中心 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/resources?app_id=xxx | 资源列表 |
| GET | /api/resource/:id/rules | 资源规则详情 |
| GET | /api/resource/:id/diff?app=xxx | 字段级 diff（当前 vs 已发布） |
| GET | /api/resource/:id | 资源元数据 |
| PUT | /api/resource/:id | 更新资源（变更模块等） |
| DELETE | /api/resource/:id | 删除资源 |

### 发布与版本 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/publish | 发布规则到 etcd（按资源维度） |
| GET | /api/publish/records | 发布记录列表 |
| GET | /api/versions | 版本列表 |
| GET | /api/versions/:id | 版本详情（含快照） |
| POST | /api/versions/:id/rollback | 回滚到指定版本 |

#### etcd 发布路径
```
/sentinel/{业务线名}/{app_key}/{group名}/{资源名}/flow            → JSON 数组
/sentinel/{业务线名}/{app_key}/{group名}/{资源名}/circuitbreaker   → JSON 数组
```

#### etcd 连接配置
- 优先从 `business_line_apps.settings` JSON 的 `url` 字段解析
- 格式：`etcd://host1:2379,host2:2379`
- 如果 settings 为空，默认 `http://127.0.0.1:2379`

### 超级管理员 API（/api/admin/*，需 super_admin 角色）
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/admin/users | 用户列表 |
| GET | /api/admin/lines | 业务线列表 |
| POST | /api/admin/lines | 创建业务线 |
| PUT | /api/admin/lines/:id | 更新业务线 |
| DELETE | /api/admin/lines/:id | 删除业务线 |
| POST | /api/admin/lines/:id/admins | 添加业务线管理员 |
| DELETE | /api/admin/lines/:id/admins/:user_id | 移除业务线管理员 |
| GET | /api/admin/audit-logs | 审计日志查询（支持 action 筛选 + 分页） |

### 业务线管理员 API（/api/line-admin/*，需 line_admin 角色）
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/line-admin/lines | 我的业务线列表 |
| PUT | /api/line-admin/lines/:id | 更新业务线描述 |
| GET | /api/line-admin/lines/:id/apps | 应用列表 |
| POST | /api/line-admin/lines/:id/apps | 创建应用 |
| PUT | /api/line-admin/lines/:id/apps/:app_id | 更新应用 |
| DELETE | /api/line-admin/lines/:id/apps/:app_id | 删除应用 |
| GET | /api/line-admin/lines/:id/members | 成员列表 |
| POST | /api/line-admin/lines/:id/members | 添加成员 |
| DELETE | /api/line-admin/lines/:id/members/:user_id | 移除成员 |

### 权限与应用 API（通用，需认证）
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/permissions | 授予权限 |
| GET | /api/permissions | 权限列表 |
| DELETE | /api/permissions/:id | 撤销权限 |
| GET | /api/apps | 应用列表 |
| POST | /api/apps | 创建应用 |
| PUT | /api/apps/:app_id | 更新应用 |
| DELETE | /api/apps/:app_id | 删除应用 |

---

## 数据库

### 表结构（14 张表，详见 `dashboard/sql/schema.sql`）

**组织架构：**
| 表名 | 说明 |
|------|------|
| `users` | 用户（role: super_admin/line_admin/member） |
| `business_lines` | 业务线 |
| `business_line_apps` | 业务线-应用关联（含 etcd settings） |
| `business_line_admins` | 业务线-管理员关联（多对多） |
| `business_line_members` | 业务线-成员关联（多对多） |

**业务数据：**
| 表名 | 说明 |
|------|------|
| `business_line_app_groups` | 业务模块 |
| `business_line_resources` | 资源 |
| `business_line_resource_flowrules` | 流控规则（每资源最多一条） |
| `business_line_resource_circuitbreakerrules` | 熔断规则（每资源最多一条） |

**发布与安全：**
| 表名 | 说明 |
|------|------|
| `publish_records` | 发布记录 |
| `publish_versions` | 版本快照（含 JSON snapshot） |
| `user_permissions` | 用户权限 |
| `user_audit_logs` | 审计日志 |
| `user_tokens` | 会话 Token（SessionStore 持久化） |

**关键约束**：
- `UNIQUE KEY uk_app_resource (app_id, resource_id)` — 每资源最多一条流控规则 + 一条熔断规则
- **无 DB 外键** — 引用完整性由代码保证（删资源时先删关联规则）
- DAO 使用 `INSERT ... ON DUPLICATE KEY UPDATE` 保证幂等
- Session token 使用 `uk_token` 唯一索引 + `idx_expires_at` 过期清理索引

### 数据访问层（DAO）

| 类型 | 文件 | 说明 |
|------|------|------|
| MySQL | `dao/mysql.go` | **唯一的数据存储**（所有 CRUD） |
| etcd | `provider/etcd_publisher.go` | 规则发布到运行时（通过 RulePublisher 接口） |

**说明**：
- 所有配置 CRUD 均通过 MySQL DAO 完成
- etcd 仅作为 Sentinel 客户端的规则消费目标（发布用），通过 `provider.EtcdRulePublisher` 抽象
- Session 持久化通过 `service/session_store_mysql.go` → `user_tokens` 表

### MySQL 连接
- Host: 127.0.0.1:3306
- User: root
- Password: (空)
- Database: sentinel_dashboard

### etcd 连接
- Host: 127.0.0.1:2379

---

## 变更日志

详见 `openspec/changes/` 目录下的各提案文档：

| 提案 | 说明 |
|------|------|
| [dashboard-ux-enhancement](changes/dashboard-ux-enhancement/) | P0: Session 持久化、配置中心抽象、审计日志、发布 diff |
| [admin-role-separation](changes/admin-role-separation/) | 管理员角色权限隔离（超级管理员/业务线管理员） |
| [resource-groups-and-mysql-migration](changes/resource-groups-and-mysql-migration/) | 资源分组和 MySQL 迁移 |
| [user-permissions-system](changes/user-permissions-system/) | 用户权限体系 |

---

## 启动命令

```bash
# 1. 启动 MySQL & etcd
brew services start mysql@5.7   # macOS
etcd --data-dir /tmp/etcd-data &

# 2. 构建（推荐用 make）
make build   # 前端 + 后端一键构建

# 3. 启动
./bin/sentinel_dashboard -c conf/dashboard-settings.yaml -p 6111

# 访问: http://localhost:6111/web/
```

---

## 已完成 P0 需求

| 需求 | 状态 | 说明 |
|------|------|------|
| Session 持久化 | ✅ | MySQL `user_tokens` 表，SessionStore 接口 + 飞书统一 |
| 配置中心抽象 | ✅ | RulePublisher 接口 + EtcdRulePublisher + EtcdClientManager |
| RollbackVersion 路径修复 | ✅ | 从旧路径 `/sentinel-go/{id}/rules/` 改为新路径 `/sentinel/{line}/{app}/{group}/{resource}/{type}` |
| 审计日志 | ✅ | 8 个 handler 接入 LogAudit + 查询 API + 前端 Tab |
| 发布预览 Diff | ✅ | 字段级 diff API + 黄色高亮 + 变更摘要 |
| 事务保护 | ✅ | PublishRules + RollbackVersion 使用 MySQL 事务 |
| bcrypt 密码 | ✅ | create-user CLI 使用 bcrypt |

## 待做需求（P1+）

| 优先级 | 需求 |
|--------|------|
| P1 | 资源搜索筛选（纯前端） |
| P1 | 版本历史 change_summary |
| P1 | 四级 RBAC（owner/admin/editor/viewer） |
| P2 | 规则模板 |
| P2 | 发布审批（可选，默认关闭） |

