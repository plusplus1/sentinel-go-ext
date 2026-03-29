# AGENTS.md — Sentinel Dashboard Go

## ⚠️ 知识沉淀规则（必须遵守）

**所有项目知识必须按 openspec 规范沉淀到 `openspec/` 目录下。** 包括但不限于：架构设计、API 规范、数据模型、变更提案、需求分析、技术决策。AGENTS.md 仅放编码约定和构建命令，不放业务知识。

## Project Overview

Sentinel Dashboard is a traffic control console for managing rate-limiting and circuit-breaking rules. It consists of a Go backend (Gin web framework) and a React/TypeScript frontend (Vite + Ant Design). Data is stored in MySQL 5.7 and published to etcd v3 for Sentinel clients to consume.

## Build & Run Commands

### 构建项目（推荐使用 Make）

```bash
# 构建完整项目（前端 + 后端）
make build

# 仅构建前端
make frontend

# 仅构建后端
make backend

# 清理构建产物
make clean

# 格式化修改的 Go 文件
make format
```

### Frontend (React 18 + TypeScript + Vite)

```bash
# Build frontend
cd frontend && npm run build

# Dev server with HMR
cd frontend && npm run dev

# Lint
cd frontend && npm run lint
```

### Linting & Database

```bash
# Go lint
golangci-lint run

# Frontend lint
cd frontend && npm run lint

# Database setup
mysql -u root < dashboard/sql/schema.sql
brew services start mysql@5.7  # macOS
etcd --data-dir /tmp/etcd-data &
```

## Testing

**No tests currently exist.** When adding tests:

```bash
go test ./...                                    # All tests
go test ./dashboard/dao/...                      # Specific package
go test -run TestFunctionName ./dashboard/service/  # Single test
go test -v ./...                                 # Verbose
go test -cover ./...                             # Coverage
```

- Place test files alongside source: `foo_test.go` next to `foo.go`
- Use `testing` package; table-driven tests preferred
- Use `github.com/stretchr/testify` for assertions (already a dependency)

## Code Style Guidelines

### Go Conventions

**Imports** — Three groups, separated by blank lines:
```go
import (
    "fmt"           // 1. Standard library
    "net/http"

    "github.com/gin-gonic/gin"  // 2. Third-party

    "github.com/plusplus1/sentinel-go-ext/dashboard/dao"  // 3. Internal
    "github.com/plusplus1/sentinel-go-ext/dashboard/model"
)
```

**Naming**:
- Exported types/functions: `PascalCase` (e.g., `ListGroups`, `MySQLGroupDAO`)
- Unexported: `camelCase` (e.g., `getMySQLDB`, `etcdClientMap`)
- Package names: lowercase single word (e.g., `dao`, `model`, `service`)
- Acronyms stay uppercase: `MySQL`, `HTTP`, `API`, `ID`, `URL`
- File names: `snake_case.go` (e.g., `group_dao.go`, `rule_flow.go`)

**Struct Tags**:
```go
type Group struct {
    ID    string `json:"id"`
    Name  string `json:"name" yaml:"name"`
    Count int    `json:"member_count,omitempty"`
}
```

**Error Handling**:
- Wrap errors with context: `fmt.Errorf("failed to create group: %v", err)`
- Return errors to callers; don't log-and-continue in library code
- API handlers return `appResp{Code: 999, Msg: err.Error()}`
- Code 100 = validation/client error; Code 999 = server error

**API Response Format**:
```go
type appResp struct {
    Code int         `json:"code"`
    Msg  string      `json:"message,omitempty"`
    Data interface{} `json:"data,omitempty"`
}
// Usage: c.JSON(http.StatusOK, appResp{Data: groups})
```

**Database Access**:
- Use parameterized queries: `db.Query("SELECT ... WHERE id = ?", id)`
- DAO structs wrap `*sql.DB`: `type MySQLGroupDAO struct { db *sql.DB }`
- Factory pattern: `func NewMySQLGroupDAO(db *sql.DB) *MySQLGroupDAO`
- Defer `rows.Close()` after every query

### Frontend Conventions (React/TypeScript)

**Linting**: ESLint with `typescript-eslint`, `react-hooks`, and `react-refresh` plugins.

**General**: TypeScript strict mode, components in `.tsx`, Ant Design for UI, React Router v6, Axios for HTTP.

**表单输入 Trim 规范**:
- 所有表单提交的字符串类型字段，必须在提交前调用 `.trim()` 去除首尾空格
- 例外情况（不需要trim）：
  - 密码字段（如 `password`）
  - JSON 配置字段（如 `settings`，可能有意保留格式空格）
  - 用户明确要求保留空格的特殊字段
- 实现方式：在表单提交处理函数中，对 `values` 对象的字符串字段调用 `.trim()`
- 示例：
  ```typescript
  const values = await form.validateFields();
  const resp = await axios.post('/api/xxx', {
    name: values.name.trim(),
    description: (values.description || '').trim(),
  });
  ```

## Project Structure

```
dashboard/
├── api/app/         # HTTP handlers (Gin)
├── api/base/        # Auth middleware, app registry
├── cmd/             # Entry point (main.go + create_user.go CLI)
├── config/          # YAML settings loader
├── dao/             # MySQL data access
├── model/           # Domain structs
├── provider/        # Config center abstraction (RulePublisher interface + EtcdPublisher)
├── service/         # Business logic (auth, session store)
├── source/          # Sentinel SDK imports
├── sql/             # schema.sql（14 张表全量建表，可重复执行）
└── install.go       # Route registration
frontend/
├── src/pages/       # React page components
├── src/App.tsx      # Router + layout
└── package.json     # Dependencies + scripts
```

## Config & Dependencies

- `conf/dashboard-settings.yaml` — MySQL credentials, Feishu SSO settings
- **Key deps**: gin, mysql v1.8.1, etcd v3.5.11, zap v1.26.0, sonic v1.10.2 (backend); React 18, Ant Design 5.x, Vite 8.x (frontend)
- Config uses YAML with struct tags for both `json` and `yaml` unmarshaling
- Warning: `frontend/frontend` symlink breaks recursive searches — use `find` or targeted paths

## Study Directory

项目包含 `study/` 目录，存放参考和调研的开源实现方案：

| 路径 | 内容 |
|------|------|
| `study/Sentinel/` | 阿里巴巴官方 Sentinel（Java SDK + Dashboard） |
| `study/Sentinel/sentinel-dashboard/` | 官方 Dashboard 实现（Spring Boot + AngularJS） |
| `study/Sentinel/sentinel-dashboard/Sentinel_Dashboard_Feature.md` | 官方功能介绍文档 |

**使用场景**：当需要参考官方实现的设计模式、API 接口、数据模型时，可直接查阅 `study/` 目录下的源码。

**注意事项**：`study/` 目录仅供参考，不要修改其中的代码。本项目是独立的 Go 实现，不需要与官方 Java 版本保持一致。

## ⚠️ CRITICAL: 前后端一致性规则

**修改后端 API 响应时，必须同步检查并修改前端代码。**

### 字段命名约定
| 后端（Go） | 前端（TypeScript） | 说明 |
|------------|-------------------|------|
| `resource_id` (int64) | `resource_id` (number) | 数据库外键 |
| `resource` (string) | `resource` (string) | 资源名称（通过 JOIN 查询获取） |
| `id` (int64) | `id` (string) | 主键，前端需转换为 string |

### 检查清单（修改规则 API 时必做）
1. **后端响应格式**：检查 `rule_flow.go` 和 `rule_circuitbreaker.go` 的响应字段
2. **前端接口定义**：检查 `Resources.tsx` 中的规则类型定义和渲染逻辑
3. **字段映射**：确保前端 state 与后端 JSON 字段名一致（规则相关全在 `Resources.tsx`）
4. **缺失字段**：前端期望的字段（如 `lowMemUsageThreshold`）后端必须返回（可给默认值）

### 常见不匹配场景
- 后端返回 `resource_id`（数字），前端期望 `resource`（名称）→ 需后端同时返回两个字段
- 后端字段为 snake_case，前端期望 camelCase → 需在响应中转换或前端 transform 函数处理
- 后端缺少前端需要的可选字段 → 需后端返回默认值（如 0.8, 1000 等）

### 验证方法
```bash
# 1. 检查后端响应字段
grep -A 20 "result = append" dashboard/api/app/rule_flow.go

# 2. 检查前端接口定义
grep -A 15 "interface FlowRuleAPI" frontend/src/pages/Resources.tsx

# 3. 对比字段是否匹配
```

## ⚠️ CRITICAL: 发布规则到 etcd

### etcd Key 路径结构
```
/sentinel/{业务线名}/{app_key}/{group名}/{资源名}/flow
/sentinel/{业务线名}/{app_key}/{group名}/{资源名}/circuitbreaker
```

### 发布流程
1. 发布按钮在资源列表的每一行（按资源维度发布）
2. 点击发布 → 弹出预览确认弹窗（左右对比：待发布 vs 已发布，变更字段黄色高亮）
3. 确认后调用 `POST /api/publish`，body: `{app_key, rule_type: "all", resource: 资源ID}`
4. 发布通过 `provider.EtcdRulePublisher` 接口，从 `business_line_apps.settings` 读取 etcd 地址，无则默认 `127.0.0.1:2379`
5. 发布后自动创建版本快照（publish_versions）和发布记录（publish_records）
6. 所有关键操作（规则 CRUD、发布、回滚、删除资源）自动记录审计日志到 `user_audit_logs`

### 数据库约束
- 每个资源最多一条流控规则 + 一条熔断规则
- `UNIQUE KEY uk_app_resource (app_id, resource_id)` 确保唯一性
- DAO 使用 `INSERT ... ON DUPLICATE KEY UPDATE` 保证幂等

### 版本管理
- 版本历史入口在资源列表的每一行（按资源查看）
- 回滚操作等同于一次新的发布
- 版本快照存储在 `publish_versions.snapshot`（JSON 格式）

### 注意事项
- `ListVersions` 和 `ListPublishRecords` 使用 `appId`（数字ID）查询，不是 `appKey`
- 前端 `Resources.tsx` 是资源中心的主页面，包含所有规则、发布、版本功能
- 前端不再有全局发布按钮，发布操作在每个资源行的操作列中

## Session 管理

- Session 持久化到 MySQL `user_tokens` 表，服务重启后 Session 仍有效
- `SessionStore` 接口定义在 `dashboard/service/session_store.go`，MySQL 实现在 `session_store_mysql.go`
- `AuthService` 通过 `SessionStore` 接口操作 session（不再使用内存 map）
- 飞书 SSO 登录和密码登录统一使用同一个 `SessionStore`
- `main.go` 启动 1 小时间隔的 session 清理 goroutine
- 用户创建通过 CLI 命令 `create-user`（`dashboard/cmd/create_user.go`），不通过 API

## 配置中心抽象

- `dashboard/provider/` 包定义了 `RulePublisher` 和 `RulePathBuilder` 接口
- 当前实现：`EtcdRulePublisher`（etcd v3），未来可扩展 Nacos
- `EtcdClientManager` 管理 per-app 的 etcd 客户端连接池
- `PublishRules` 和 `RollbackVersion` 均通过 provider 接口发布，不直接操作 etcd 客户端
- 每个 app 的 etcd 设置从 `business_line_apps.settings` JSON 字段读取

## 审计日志

- 所有关键写操作（8 个 handler）调用 `AuthService.LogAudit()` 记录到 `user_audit_logs` 表
- 审计的操作：rule_create, rule_update, rule_delete, rule_toggle, publish, rollback, resource_delete
- 查询接口：`GET /api/admin/audit-logs`（超级管理员，支持 action 筛选 + 分页）
- 前端：Admin.tsx 超管面板下方有 `AuditLogPanel` 组件
