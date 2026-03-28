## P0 — 必须做

### 1. Session 持久化（MySQL + 抽象层） ✅

- [x] 1.1 创建 `service/session_store.go`，定义 `SessionStore` 接口
- [x] 1.2 创建 `service/session_store_mysql.go`，实现 MySQL 版本
- [x] 1.3 改造 `AuthService`：替换 `sessionStore map` 为 `SessionStore` 接口
- [x] 1.4 改造 `Login`：调用 `sessions.CreateSession()` 写 DB
- [x] 1.5 改造 `ValidateSession`：调用 `sessions.ValidateSession()` 查 DB
- [x] 1.6 改造 `Logout` + `FeishuCallback`：统一走 `SessionStore`
- [x] 1.7 在 `main.go before()` 中启动 Session 清理 goroutine（1h 间隔）
- [x] 1.8 数据库迁移：`migrate_session.sql` — `ADD INDEX idx_expires_at`

**验收标准**：重启服务后已登录用户无需重新登录

---

### 2. 配置中心抽象层（RulePublisher） ✅

- [x] 2.1 创建 `dashboard/provider/rule_provider.go`，定义 `RulePublisher` / `RulePathBuilder` 接口
- [x] 2.2 创建 `dashboard/provider/etcd_publisher.go`，实现 `EtcdRulePublisher` + `EtcdClientManager`
- [x] 2.3 创建 `dashboard/provider/etcd_path_builder.go`，实现路径构建
- [x] 2.4 改造 `resource.go:PublishRules`，使用 `RulePublisher` 接口
- [x] 2.5 改造 `resource.go:RollbackVersion`，使用 `RulePublisher` 接口（修复旧路径 bug）

**验收标准**：发布/回滚通过接口调用，业务代码不直接依赖 etcd 客户端

**未来扩展**（P2/P3）：
- [ ] 创建 `dashboard/provider/nacos_publisher.go`，实现 Nacos 版本
- [ ] 在配置文件中支持 `provider: nacos`

---

### 4. 审计日志接线 ✅

- [x] 4.1 在 `rule_flow.go:SaveOrUpdateFlowRule` 成功后调用 LogAudit
- [x] 4.2 在 `rule_flow.go:DeleteFlowRule` 成功后调用 LogAudit
- [x] 4.3 在 `rule_circuitbreaker.go:SaveOrUpdateCircuitbreakerRule` 成功后调用 LogAudit
- [x] 4.4 在 `rule_circuitbreaker.go:DeleteCircuitbreakerRule` 成功后调用 LogAudit
- [x] 4.5 在 `resource.go:ToggleRule` 成功后调用 LogAudit
- [x] 4.6 在 `resource.go:PublishRules` 成功后调用 LogAudit
- [x] 4.7 在 `resource.go:RollbackVersion` 成功后调用 LogAudit
- [x] 4.8 在 `resource.go:DeleteResource` 成功后调用 LogAudit
- [x] 4.9 创建 `GET /api/admin/audit-logs` 查询接口（支持 action 筛选 + 分页）
- [x] 4.10 前端 Admin.tsx 添加 `AuditLogPanel` 组件

**验收标准**：所有关键操作有审计日志记录，可查询和查看详情

---

### 5. 发布预览 Diff 增强 ✅

- [x] 5.1 后端：新增 `GET /api/resource/:id/diff?app=xxx` 接口 + `FieldDiff` 结构
- [x] 5.2 后端：实现字段级 diff（`compareFlowRule` / `compareCBRule`）
- [x] 5.3 前端：发布预览 Modal 底部添加变更摘要区域（绿色框）
- [x] 5.4 前端：有变更的字段黄色背景高亮（`changedStyle`）

**验收标准**：发布预览清晰显示哪些字段变更了，从什么值变成什么值

---

## P1 — 应该做

### 6. 资源搜索筛选

- [ ] 4.1 前端 Resources.tsx 顶部添加搜索输入框
- [ ] 4.2 前端添加模块下拉筛选器
- [ ] 4.3 前端添加发布状态筛选器（全部/已发布/有变更/未发布）
- [ ] 4.4 前端实现客户端筛选逻辑
- [ ] 4.5 前端显示筛选结果计数（"共 N 条，筛选后 M 条"）

**验收标准**：支持按名称搜索、按模块筛选、按发布状态筛选，结果准确

---

### 7. 版本历史变更摘要

- [ ] 5.1 数据库迁移：`ALTER TABLE publish_versions ADD COLUMN change_summary TEXT`
- [ ] 5.2 后端：修改 PublishRules，发布时自动生成 change_summary
- [ ] 5.3 后端：修改 RollbackVersion，回滚时自动生成 change_summary
- [ ] 5.4 前端：版本历史 Modal 添加变更摘要列

**验收标准**：版本列表显示变更概览，回滚前可预览

---

### 8. 细粒度权限（四级 RBAC）

- [ ] 6.1 数据库迁移：修改 `user_permissions.role` 支持 owner/admin/editor/viewer
- [ ] 6.2 后端：创建 `service/rbac.go`，实现 `hasPermission(role, action)` 逻辑
- [ ] 6.3 后端：创建 `RequireActionMiddleware(action)` 中间件
- [ ] 6.4 后端：在关键 API 路由上添加权限中间件
- [ ] 6.5 后端：改造 GrantPermission API，支持选择 role
- [ ] 6.6 前端：Admin.tsx 权限管理界面支持选择 role
- [ ] 6.7 前端：Resources.tsx 根据用户 role 显示/隐藏操作按钮
- [ ] 6.8 数据迁移脚本：现有 member → editor，line_admin → admin

**验收标准**：四级权限正确生效，editor 只能编辑不能发布，viewer 只能查看

---

## P2 — 可以做

### 9. 规则模板

- [ ] 7.1 数据库：创建 `rule_templates` 表
- [ ] 7.2 数据库：插入 4 条预设模板
- [ ] 7.3 后端：实现模板 CRUD API
- [ ] 7.4 前端：规则编辑表单添加「从模板创建」下拉

**验收标准**：可使用预设模板创建规则，可保存自定义模板

---

### 10. 发布审批（可选配置）

- [ ] 8.1 数据库：创建 `publish_approvals` 表
- [ ] 8.2 后端：创建 `Notifier` 接口 + `NoopNotifier` 默认实现
- [ ] 8.3 后端：在 PublishRules 中检查 `approval_required` 配置
- [ ] 8.4 后端：创建审批流 API（提交/审批/拒绝）
- [ ] 8.5 后端：预留通知接口（SendApprovalNotification）

**验收标准**：配置开启后，editor 发布需 admin 审批；配置关闭时直接发布

---

## 验收标准

### P0 ✅

- [x] 重启服务后已登录用户无需重新登录（SessionStore → MySQL）
- [x] 所有关键操作有审计日志记录，可查询（8 个 handler + admin API + 前端）
- [x] 发布预览显示字段级 diff（黄色高亮 + 变更摘要）
- [x] 发布/回滚通过 provider 接口，不直接依赖 etcd 客户端
- [x] RollbackVersion 使用正确的 etcd 路径

### P1

- [ ] 支持按名称/模块/发布状态筛选
- [ ] 版本列表显示变更摘要
- [ ] 四级权限正确生效

### P2

- [ ] 可使用预设模板创建规则
- [ ] 审批流可选开启
