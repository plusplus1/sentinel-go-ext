## Why

当前 Sentinel Dashboard 在用户体验、权限设计和运维效率方面存在多个痛点：

1. **Session 不持久**：服务器重启后所有用户被踢出，运维维护成本高
2. **审计日志缺失**：规则变更、发布操作无迹可查，出问题无法溯源
3. **权限粒度粗**：只有 super_admin/line_admin/member 三级，缺少只读、编辑不发布等细粒度权限
4. **搜索效率低**：资源列表无搜索筛选，资源多了找不到
5. **版本对比不直观**：发布前无法清晰看到"哪些字段变了"

已解决的问题（不需要覆盖）：
- ✅ 发布状态对比（左右 diff 对比）
- ✅ 前后端字段一致性（resource_id + resource 双返回）
- ✅ 死代码清理
- ✅ 事务保护
- ✅ 资源名称验证

## What Changes

### P0 — 必须做（阻塞日常运维）

1. **Session 持久化**：MySQL 存储 + 抽象扩展层（未来可平替 Redis）
2. **审计日志接线**：将 LogAudit 调用接入规则 CRUD、发布、回滚等关键操作
3. **发布预览增强**：字段级 diff 高亮（哪些字段变了，变了多少）
4. **create-user 命令行工具**：运维创建超管/普通账号（✅ 已实现）

### P1 — 应该做（提升用户体验）

5. **资源搜索筛选**：按名称搜索、按模块筛选、按发布状态筛选
6. **版本历史变更摘要**：版本列表显示变更概览，回滚前预览
7. **细粒度权限（owner/admin/editor/viewer）**：完整的四级权限模型

### P2 — 可以做（锦上添花）

8. **规则模板**：预设模板（标准 API、高并发、慢调用熔断等）
9. **发布审批（可选配置）**：预留接口，默认不开启，未来扩展

### 不做

- ❌ **批量操作**：当前资源量小，手动操作够用
- ❌ **定时发布**：单环境无需求
- ❌ **多环境管理**：无业务场景
- ❌ **数据库外键约束**：在代码层保证联动一致性，避免迁移麻烦
- ❌ **超级管理员访问资源中心**：超管定位是管理业务线/组织，不访问资源

## Capabilities

### New Capabilities

- `session-persistence`: Session 持久化（MySQL + 抽象层）
- `config-center-abstraction`: 配置中心抽象层（etcd/Nacos/Consul 可插拔）
- `audit-logging`: 审计日志（记录规则变更、发布、权限操作）
- `publish-diff`: 发布预览字段级 diff 高亮
- `resource-search-filter`: 资源搜索和筛选
- `fine-grained-permissions`: 四级权限模型
- `rule-templates`: 规则模板
- `publish-approval`: 发布审批（可选，默认关闭）

### Modified Capabilities

- `publish-status-comparison`: 在现有基础上增加 diff 高亮
- `version-management`: 增加变更摘要字段
- `permission-system`: 从三级扩展到四级

## Non-goals

- 不引入 Redis 等新中间件（Session 用 MySQL，抽象层支持未来平替）
- 不修改 Sentinel 客户端 SDK
- 不改变 etcd 数据结构
- 不在数据库层面加外键约束（代码层保证一致性）
- 不支持批量操作
- 不支持定时发布
- 不支持多环境管理

## 实施计划

| 阶段 | 内容 | 预估周期 |
|------|------|---------|
| P0 | Session 持久化 + 审计日志接线 + 发布 diff 增强 | 1 周 |
| P1 | 资源搜索筛选 + 版本变更摘要 + 四级权限 | 2 周 |
| P2 | 规则模板 + 发布审批接口预留 | 1 周 |
