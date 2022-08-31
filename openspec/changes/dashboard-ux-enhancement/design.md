## Context

### 背景

Sentinel Dashboard 是一个流量控制规则管理平台。当前系统架构：

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Browser (React)  →  Go Backend (Gin)  →  MySQL (配置存储)  →  etcd (运行时规则) │
│ Session: 内存 map（重启丢失）  │  权限: users.role + business_line_admins       │
└─────────────────────────────────────────────────────────────────────────┘
```

### 已实现的基础能力

- ✅ 发布状态对比（last_publish_at / running_version / latest_version）
- ✅ 发布预览左右 diff 对比
- ✅ 前后端字段一致性（resource_id + resource 双返回）
- ✅ 事务保护（RollbackVersion / PublishRules）
- ✅ create-user 命令行工具

### 约束条件

- 后端 Go 1.20 + Gin，无 Redis
- 前端 React 18 + Ant Design 5.x
- MySQL 5.7
- 不在数据库层面加外键（代码层保证联动一致性）
- 超级管理员不访问资源中心（定位是管理组织）

---

## Decisions

### 1. Session 持久化：MySQL + Store 抽象层

**决策**：用 MySQL 存储 Session，但通过 Store 接口抽象，未来可平替 Redis。

**理由**：
- 当前无 Redis 依赖，MySQL 性能对管理后台够用
- 抽象接口使未来迁移零改动业务代码
- Login 时写 DB，ValidateSession 时查 DB（miss 时）

**接口设计**：
```go
// dashboard/service/session_store.go
type SessionStore interface {
    CreateSession(userID string, ttl time.Duration) (token string, err error)
    ValidateSession(token string) (*SessionInfo, error)
    DeleteSession(token string) error
    DeleteUserSessions(userID string) error
    CleanExpired() error
}

type SessionInfo struct {
    UserID    string
    ExpiresAt time.Time
}
```

**MySQL 实现**：
```go
// dashboard/service/session_store_mysql.go
type MySQLSessionStore struct {
    db *sql.DB
}

func (s *MySQLSessionStore) CreateSession(userID string, ttl time.Duration) (string, error) {
    token := generateToken()
    expiresAt := time.Now().Add(ttl)
    _, err := s.db.Exec(`
        INSERT INTO user_tokens (user_id, token, expires_at)
        VALUES (?, ?, ?)
        ON DUPLICATE KEY UPDATE token = VALUES(token), expires_at = VALUES(expires_at)
    `, userID, token, expiresAt)
    return token, err
}

func (s *MySQLSessionStore) ValidateSession(token string) (*SessionInfo, error) {
    var info SessionInfo
    err := s.db.QueryRow(`
        SELECT user_id, expires_at FROM user_tokens
        WHERE token = ? AND expires_at > NOW()
    `, token).Scan(&info.UserID, &info.ExpiresAt)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("会话不存在或已过期")
    }
    return &info, err
}

func (s *MySQLSessionStore) CleanExpired() error {
    _, err := s.db.Exec("DELETE FROM user_tokens WHERE expires_at < NOW()")
    return err
}
```

**AuthService 改造**：
```go
type AuthService struct {
    db       *sql.DB
    sessions SessionStore   // 替换原来的 sessionStore map
    mu       sync.RWMutex
}

func NewAuthService(db *sql.DB) *AuthService {
    return &AuthService{
        db:       db,
        sessions: &MySQLSessionStore{db: db},
    }
}

func (s *AuthService) Login(email, password string) (*model.User, string, error) {
    // ... 验证密码
    token, err := s.sessions.CreateSession(user.UserID, 24*time.Hour)
    return &user, token, err
}

func (s *AuthService) ValidateSession(token string) (*model.User, error) {
    info, err := s.sessions.ValidateSession(token)
    if err != nil {
        return nil, err
    }
    return s.GetUserByID(info.UserID)
}
```

**定时清理**：在 `dashboard/cmd/main.go` 的 `before()` 中启动 goroutine：
```go
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        authService.CleanExpiredSessions()
    }
}()
```

**未来平替 Redis**：实现 `RedisSessionStore`，替换 `NewAuthService(db)` 为 `NewAuthService(redisClient)`，零业务代码改动。

**数据库变更**：`user_tokens` 表添加索引：
```sql
ALTER TABLE user_tokens ADD INDEX idx_expires_at (expires_at);
```

---

### 2. 配置中心抽象层（DynamicRuleProvider / Publisher）

**决策**：定义 `RuleProvider` 和 `RulePublisher` 接口，当前实现 etcd 版本，未来可扩展 Nacos/Consul。

**理由**：
- 当前使用 etcd 作为运行时规则存储，客户端轮询 etcd 获取规则
- 未来可能需要支持 Nacos、Consul 等其他配置中心
- 接口抽象使扩展零改动业务代码

**接口设计**：
```go
// dashboard/provider/rule_provider.go
package provider

// RuleProvider 从配置中心读取规则
type RuleProvider interface {
    GetFlowRules(appKey, group, resource string) ([]FlowRule, error)
    GetCBRules(appKey, group, resource string) ([]CBRule, error)
    GetSystemRules(appKey, group, resource string) ([]SystemRule, error)
}

// RulePublisher 将规则推送到配置中心
type RulePublisher interface {
    PublishFlowRules(appKey, group, resource string, rules []FlowRule) error
    PublishCBRules(appKey, group, resource string, rules []CBRule) error
    PublishSystemRules(appKey, group, resource string, rules []SystemRule) error
    DeleteRule(appKey, group, resource, ruleType string) error
}

// RulePathBuilder 构建配置中心的 key 路径
type RulePathBuilder interface {
    BuildPath(line, app, group, resource, ruleType string) string
    ParsePath(path string) (line, app, group, resource, ruleType string, err error)
}
```

**etcd 实现**：
```go
// dashboard/provider/etcd_provider.go
package provider

import (
    clientv3 "go.etcd.io/etcd/client/v3"
    "context"
    "encoding/json"
    "fmt"
)

type EtcdRuleProvider struct {
    client *clientv3.Client
}

type EtcdRulePublisher struct {
    client *clientv3.Client
}

type EtcdPathBuilder struct {
    prefix string // 默认 "/sentinel"
}

// BuildPath: /sentinel/{业务线}/{app_key}/{group}/{resource}/{rule_type}
func (b *EtcdPathBuilder) BuildPath(line, app, group, resource, ruleType string) string {
    return fmt.Sprintf("%s/%s/%s/%s/%s/%s", b.prefix, line, app, group, resource, ruleType)
}

func (p *EtcdRulePublisher) PublishFlowRules(appKey, group, resource string, rules []FlowRule) error {
    data, err := json.Marshal(rules)
    if err != nil {
        return err
    }
    path := p.pathBuilder.BuildPath(p.lineName, appKey, group, resource, "flow")
    _, err = p.client.Put(context.Background(), path, string(data))
    return err
}
```

**未来 Nacos 实现**（预留）：
```go
// dashboard/provider/nacos_provider.go
package provider

import nacos "github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"

type NacosRuleProvider struct {
    client nacos.IConfigClient
    group  string // nacos group
}

func (p *NacosRuleProvider) GetFlowRules(appKey, group, resource string) ([]FlowRule, error) {
    dataId := fmt.Sprintf("%s-%s-flow", appKey, resource)
    content, err := p.client.GetConfig(vo.ConfigParam{
        DataId: dataId,
        Group:  p.group,
    })
    // ... 反序列化
}
```

**配置选择**（dashboard-settings.yaml）：
```yaml
rule_store:
  provider: etcd          # etcd | nacos | consul
  etcd:
    endpoints:
      - 127.0.0.1:2379
  # nacos:
  #   server_addr: 127.0.0.1:8848
  #   namespace: public
  #   group: SENTINEL_GROUP
```

**业务代码改动**：
```go
// 原来的代码（直接操作 etcd）
client.Put(ctx, key, string(data))

// 改为（通过接口）
publisher.PublishFlowRules(appKey, group, resource, rules)
```

**影响范围**：
- `dashboard/api/app/resource.go:PublishRules` — 改用 Publisher 接口
- `dashboard/api/app/resource.go:RollbackVersion` — 改用 Publisher 接口
- 新增 `dashboard/provider/` 目录，包含接口定义和实现

---

### 3. 审计日志

**决策**：在所有关键操作处调用 LogAudit，记录操作人 + 操作类型 + 变更前后值。

**审计操作类型**：
```
rule_create      规则创建
rule_update      规则更新
rule_delete      规则删除
rule_toggle      规则启用/禁用
publish          发布规则到 etcd
rollback         回滚版本
resource_create  创建资源
resource_delete  删除资源
permission_grant 授权
permission_revoke 撤销授权
```

**LogAudit 调用位置**（代码中需要插入）：

| 操作 | 文件 | 函数 |
|------|------|------|
| 规则创建/更新 | `rule_flow.go` | SaveOrUpdateFlowRule |
| 规则创建/更新 | `rule_circuitbreaker.go` | SaveOrUpdateCircuitbreakerRule |
| 规则删除 | `rule_flow.go` | DeleteFlowRule |
| 规则删除 | `rule_circuitbreaker.go` | DeleteCircuitbreakerRule |
| 规则启停 | `resource.go` | ToggleRule |
| 发布 | `resource.go` | PublishRules |
| 回滚 | `resource.go` | RollbackVersion |
| 资源创建 | `resource.go` | (添加资源 API) |
| 资源删除 | `resource.go` | DeleteResource |
| 权限授权 | `auth_admin.go` | GrantPermission |
| 权限撤销 | `auth_admin.go` | RevokePermission |

**调用示例**：
```go
// 在 rule_flow.go 的 SaveOrUpdateFlowRule 成功后
authService.LogAudit(
    user.UserID,               // 操作人
    "rule_update",             // 操作类型
    "flow_rule",               // 资源类型
    fmt.Sprintf("%d", rule.ID), // 资源ID
    fmt.Sprintf("threshold: %v → %v", oldThreshold, newThreshold), // 变更摘要
    c.ClientIP(),              // IP
)
```

**审计日志查询 API**：`GET /api/admin/audit-logs`
- 支持按 user_id、action、时间范围筛选
- 分页返回（page, page_size）
- 仅 super_admin 可访问

**前端**：Admin.tsx 中添加「审计日志」Tab，Table 展示：
- 操作人 | 操作类型 | 资源类型 | 资源ID | 变更摘要 | IP | 时间

---

### 4. 发布预览 Diff 增强

**决策**：在现有左右对比基础上，增加字段级 diff 高亮。

**UI 设计**（修改 Resources.tsx 发布预览 Modal）：

```
┌─ 发布确认 - abcd ──────────────────────────────────┐
│                                                     │
│  🟢 待发布（当前配置）   │  ⚪ 已发布（运行中）      │
│                          │                           │
│  ⚡ 流控规则              │  ⚡ 流控规则              │
│  ┌──────────────────┐    │  ┌──────────────────┐    │
│  │ 指标: QPS         │    │  │ 指标: QPS         │    │
│  │ 阈值: 100 [↑50%]  │ ←  │  │ 阈值: 200         │    │
│  │ 行为: 直接拒绝     │    │  │ 行为: 直接拒绝     │    │
│  │ 策略: Direct       │    │  │ 策略: WarmUp      │ ←  │
│  └──────────────────┘    │  └──────────────────┘    │
│                          │                           │
│  变更摘要: 2 项变更       │                           │
│  • 阈值: 100 → 200       │                           │
│  • 策略: Direct → WarmUp │                           │
└─────────────────────────────────────────────────────┘
```

**实现**：
1. 后端：新增 `GET /api/resource/:id/diff` 接口，返回字段级 diff
2. 前端：发布预览 Modal 底部显示变更摘要列表
3. 有变更的字段用黄色背景高亮

---

### 5. 资源搜索筛选

**UI 设计**（修改 Resources.tsx 顶部）：

```
┌─ 资源中心 ──────────────────────────────────────────┐
│                                                     │
│  [搜索资源名称...]  [模块 ▼]  [发布状态 ▼]  [刷新]  │
│                                                     │
│  ┌──────────────┬────────┬────────┬──────────────┐  │
│  │ 资源名称      │ 所属模块 │ 流控   │ 熔断        │  │
│  ├──────────────┼────────┼────────┼──────────────┤  │
│  │ API:GET:/user │ theme  │ 已配置 │ 已配置       │  │
│  │ API:POST:/app │ theme  │ 未配置 │ 已配置       │  │
│  └──────────────┴────────┴────────┴──────────────┘  │
│  共 12 条，筛选后 2 条                               │
└─────────────────────────────────────────────────────┘
```

**筛选条件**：
- 搜索框：按资源名称模糊匹配（前端本地筛选，不调 API）
- 模块下拉：按 group_id 筛选（前端本地筛选）
- 发布状态：全部 / 已发布 / 有变更 / 未发布（前端本地筛选）

**实现**：纯前端筛选，不增加后端 API。
```tsx
const [searchText, setSearchText] = useState('');
const [filterGroup, setFilterGroup] = useState<string>('');
const [filterPublishStatus, setFilterPublishStatus] = useState<string>('');

const filteredResources = resources.filter(r => {
  if (searchText && !r.name.toLowerCase().includes(searchText.toLowerCase())) return false;
  if (filterGroup && r.group_id !== filterGroup) return false;
  if (filterPublishStatus === 'published' && !r.running_version) return false;
  if (filterPublishStatus === 'changed' && r.latest_version <= (r.running_version || 0)) return false;
  if (filterPublishStatus === 'unpublished' && r.last_publish_at) return false;
  return true;
});
```

---

### 6. 版本历史变更摘要

**决策**：在 `publish_versions` 表添加 `change_summary` 字段，发布时自动生成。

**数据库变更**：
```sql
ALTER TABLE publish_versions ADD COLUMN change_summary TEXT;
```

**后端实现**（修改 PublishRules 和 RollbackVersion）：
```go
// 发布时生成变更摘要
func generateChangeSummary(oldRules, newRules []RuleSnapshot) string {
    var summary []string
    // 对比 oldRules 和 newRules
    // 生成 "阈值: 100 → 200" 格式的摘要
    return strings.Join(summary, "\n")
}
```

**前端**：版本历史 Modal 显示 change_summary 列

---

### 7. 细粒度权限（四级 RBAC）

**权限矩阵**：

| 操作 | owner | admin | editor | viewer | 无权限 |
|------|-------|-------|--------|--------|--------|
| 查看资源 | ✅ | ✅ | ✅ | ✅ | ❌ |
| 创建规则 | ✅ | ✅ | ✅ | ❌ | ❌ |
| 编辑规则 | ✅ | ✅ | ✅ | ❌ | ❌ |
| 删除规则 | ✅ | ✅ | ❌ | ❌ | ❌ |
| 发布规则 | ✅ | ✅ | ❌ | ❌ | ❌ |
| 回滚版本 | ✅ | ✅ | ❌ | ❌ | ❌ |
| 管理成员 | ✅ | ❌ | ❌ | ❌ | ❌ |
| 删除资源 | ✅ | ❌ | ❌ | ❌ | ❌ |

**数据库变更**：
```sql
ALTER TABLE user_permissions MODIFY COLUMN role VARCHAR(32) NOT NULL DEFAULT 'viewer';
```

**权限映射**（当前 role → 新 role）：
- `super_admin` → 不变，永远全权限
- `line_admin` → 映射为 `admin`
- `member` → 映射为 `editor`

**后端中间件**（新增 RequireActionMiddleware）：
```go
func RequireActionMiddleware(action string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := c.MustGet("user").(*model.User)
        resourceID := c.Param("id")
        resourceType := c.Query("type")

        // super_admin 永远有权限
        if user.Role == "super_admin" {
            c.Next()
            return
        }

        // 查 user_permissions 获取 role
        role := getUserRole(user.UserID, resourceType, resourceID)
        if !hasPermission(role, action) {
            c.JSON(200, appResp{Code: 403, Msg: "权限不足"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**路由改造**：
```go
// 发布需要 admin 权限
protectedGroup.POST("/publish", app.PublishRules)
// 同时添加权限检查中间件
protectedGroup.POST("/publish", base.RequireActionMiddleware("publish"), app.PublishRules)
```

**前端**：根据用户 role 显示/隐藏操作按钮
```tsx
const canPublish = userRole === 'owner' || userRole === 'admin' || user.role === 'super_admin';
const canEdit = canPublish || userRole === 'editor';

{canPublish && <Button onClick={handlePublish}>发布</Button>}
{canEdit && <Button onClick={handleEdit}>编辑</Button>}
```

---

### 8. 规则模板（P2）

**数据库设计**：
```sql
CREATE TABLE rule_templates (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512),
    type VARCHAR(32) NOT NULL COMMENT 'flow/circuitbreaker',
    is_system TINYINT(1) NOT NULL DEFAULT 0 COMMENT '1=系统预设',
    config JSON NOT NULL,
    created_by VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**预设模板**：
| 名称 | 类型 | 适用场景 |
|------|------|---------|
| 标准 API 限流 | flow | 普通 API 接口，QPS=100 |
| 高并发保护 | flow | 高并发场景，并发数=500+排队 |
| 慢调用熔断 | circuitbreaker | 依赖外部服务，慢调用>50% |
| 错误率熔断 | circuitbreaker | 关键接口，错误率>30% |

**UI**：规则编辑表单顶部添加「从模板创建」下拉选择器

---

### 8. 发布审批（可选配置，默认关闭）

**配置项**（dashboard-settings.yaml）：
```yaml
publish:
  approval_required: false    # 默认关闭
  approver_roles:             # 审批人角色
    - super_admin
    - line_admin
```

**数据库设计**：
```sql
CREATE TABLE publish_approvals (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id VARCHAR(64) NOT NULL,
    resource_id VARCHAR(64),
    rule_type VARCHAR(32) NOT NULL,
    rule_snapshot JSON NOT NULL COMMENT '待发布的规则快照',
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/approved/rejected',
    requester_id VARCHAR(64) NOT NULL,
    reviewer_id VARCHAR(64),
    review_comment VARCHAR(512),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP NULL
);
```

**流程**：
1. editor 提交发布 → 创建 approval 记录，status=pending
2. admin/approver 收到通知（预留通知接口）
3. admin 审批通过 → 执行实际发布（调用现有 PublishRules）
4. admin 审批拒绝 → status=rejected，附带原因

**通知接口（预留）**：
```go
type Notifier interface {
    NotifyApprovalRequired(approvalID int64, approverEmail string) error
    NotifyApprovalResult(approvalID int64, status string) error
}
// 默认实现：NoopNotifier（什么都不做）
// 未来实现：FeishuNotifier, EmailNotifier
```

---

## Risks / Trade-offs

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| Session 改造影响现有登录流程 | 高 | 保留 Login/ValidateSession 接口签名不变，只改内部实现 |
| 权限改造影响现有用户 | 中 | 现有 role 映射：super_admin 不变，line_admin → admin，member → editor |
| 审计日志写入性能 | 低 | 异步写入（goroutine），不阻塞主流程 |
| 前端筛选量大时性能 | 低 | 资源量 <1000 时客户端筛选够用 |

## Open Questions

1. **Session 过期策略**：固定 24h 还是支持"记住我"延长？
2. **权限继承**：owner 是否自动继承到业务线下的所有应用？
3. **审计日志保留**：保留多久？需要定期清理吗？
