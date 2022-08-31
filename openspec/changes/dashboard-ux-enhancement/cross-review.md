# Dashboard UX Enhancement — 交叉 Review 报告

> 基于 needs.md（安全/数据完整性审查）× dashboard-ux-enhancement（UX 功能增强提案）交叉比对

---

## 一、已修复项（Proposal 不需要覆盖）

以下问题已在最近一轮开发中修复，Proposal 可以从 tasks.md 中移除或标记为已完成：

| 编号 | 问题 | 实际修复情况 |
|------|------|-------------|
| P0 发布状态对比 | 2.1 ~ 2.5 | ✅ 已实现：资源列表显示 last_publish_at / running_version；发布预览含左右 diff 对比 |
| N-010 死代码 | — | ✅ 已删除 AppList.tsx / FlowRules.tsx / CircuitBreakerRules.tsx / App.css / publish_service.go / source/etcd/ |
| N-014 死亡 UI | — | ✅ 已移除通知铃铛按钮 |
| N-013 资源名称验证 | — | ✅ 已添加正则验证 |

---

## 二、Proposal 遗漏的 P0 安全问题

这些是 needs.md 中发现但 Proposal 完全未覆盖的关键问题：

### ❌ N-001: CreateUser 密码明文存储
- **Proposal 状态**: 未提及
- **实际状态**: **仍未修复**
- **影响**: 管理员创建的用户无法登录
- **根因**: `auth_admin.go:158` 将明文密码存入 `password_hash` 列，而 Login 校验的是 bcrypt hash

### ❌ N-003: 外键约束缺失
- **Proposal 状态**: 未提及
- **实际状态**: **仍未修复** — 14 张表仅 2 个外键
- **影响**: 删除业务线/用户会产生大量孤立记录
- **方案**: 至少为 business_line_apps / business_line_admins / business_line_members 添加 CASCADE 外键

### ❌ N-006: 安全加固（Cookie/CSRF/端点）
- **Proposal 状态**: 未提及
- **实际状态**: **仍未修复**
  - Session cookie `Secure: false`
  - 无 CSRF 保护
  - `/fs/` 调试端点暴露
  - 硬编码 Feishu 测试凭据

### ⚠️ N-002/N-005: 事务保护
- **Proposal 状态**: 未提及
- **实际状态**: ✅ **已修复**（RollbackVersion + PublishRules 已加事务）

---

## 三、Proposal 遗漏的 P1/P2 问题

| 编号 | 问题 | Proposal 状态 | 建议补充 |
|------|------|-------------|---------|
| N-007 | 规则计数始终为 0（N+1 查询） | 未提及 | P1 任务：在 SQL 中 JOIN 统计 |
| N-008 | 静默错误处理（5 处空 catch） | 未提及 | ✅ 已修复 |
| N-009 | 超级管理员无法访问资源中心 | 未提及 | P0 修改侧边栏菜单逻辑 |
| N-011 | TypeScript `any` 类型 | 未提及 | ✅ 已修复（CurrentUser 接口） |
| N-015 | 管理中心在线状态误导 | 未提及 | P1 移除或改为账号状态 |
| N-016 | 数据库迁移机制缺失 | 未提及 | P2 引入 golang-migrate |

---

## 四、Proposal 过度工程化的问题

以下 P2/P3 功能对当前团队规模而言过于超前，建议降级或砍掉：

### ⚠️ 10. 发布审批流程
- **Proposal**: 完整的 publish_approvals 表 + 审批通知机制
- **现实**: 当前只有 1~3 个测试账号，无真实用户量
- **建议**: 降级为 P3，等有实际团队协作需求再做

### ⚠️ 8. 批量操作（batch_tasks + 异步轮询）
- **Proposal**: 新建 2 张表 + 异步任务系统 + 前端轮询进度
- **现实**: 当前每个 app 最多几十个资源，手动操作完全够用
- **建议**: 降级为 P2，先做简单的批量发布（循环调用现有 API）即可

### ⚠️ 3. 定时发布 + 9. 多环境管理
- **Proposal**: P3 级别的定时任务 + 开发/测试/生产环境隔离
- **现实**: 单环境单集群，短期内无需求
- **建议**: 从 tasks.md 中移除，作为独立提案留给未来

---

## 五、Proposal 设计问题

### 问题 1: Session 持久化方案过重
Proposal 使用 MySQL 完整存储 Session（INSERT ... ON DUPLICATE KEY UPDATE），但当前方案是内存 map。更轻量的方案：
- **推荐**: 只在内存 map 中加一层 —— `ValidateSession` 先查内存，miss 时再查 `user_tokens` 表
- **优势**: 保留内存速度，重启后自动从 DB 恢复
- **劣势**: 需要在 Login 时也写 DB（当前 Feishu 已写，密码登录没写）

### 问题 2: 权限模型过于复杂
Proposal 提出 owner/admin/editor/viewer 四级权限，但当前只有 super_admin/line_admin/member 三级。增加四级权限需要：
- 修改 `user_permissions` 表
- 创建权限继承视图
- 修改所有 API 的权限检查
- 前端权限管理界面

**建议**: 先保留三级权限，在现有基础上增加「只读成员」role 即可，不必引入完整的 RBAC 模型。

### 问题 3: 规则模板对当前需求过早
当前只有 2~3 个资源，手动配置完全够用。规则模板需要：
- 新建 rule_templates 表
- 实现模板 CRUD API
- 前端模板选择器组件

**建议**: 降级为 P3，等资源数达到 100+ 再考虑。

### 问题 4: 发布状态对比设计重复
Proposal 设计了全新的 `has_unpublished_changes` + `unpublished_count` API 响应，但我们已经实现了：
- `last_publish_at` / `running_version` / `latest_version` 字段
- 发布预览弹窗左右 diff 对比

Proposal 的设计应该基于现有实现迭代，而不是重新设计。

---

## 六、建议的合并需求清单

### P0 — 必须做（合并 needs.md + Proposal）

| 编号 | 内容 | 来源 | 状态 |
|------|------|------|------|
| N-001 | 修复 CreateUser 密码明文存储 | needs.md | ❌ 未修复 |
| N-003 | 添加外键约束（至少 3 张表） | needs.md | ❌ 未修复 |
| N-004 | Session 持久化（轻量方案） | Proposal #1 | ❌ 未修复 |
| N-006 | 安全加固（Cookie Secure + CSRF + 移除 /fs/） | needs.md | ❌ 未修复 |
| N-009 | 超级管理员允许访问资源中心 | needs.md | ❌ 未修复 |
| #2 | 发布状态对比增强（变更摘要） | Proposal #2 | ⚠️ 基础已实现，差摘要 |

### P1 — 应该做

| 编号 | 内容 | 来源 | 状态 |
|------|------|------|------|
| N-007 | 规则计数优化（消除 N+1 查询） | needs.md | ❌ 未修复 |
| N-015 | 移除在线/离线状态误导 | needs.md | ❌ 未修复 |
| N-016 | 引入数据库迁移机制 | needs.md | ❌ 未修复 |
| #4 | 资源搜索筛选 | Proposal | ❌ 未实现 |
| #6 | 审计日志完善 | Proposal | ❌ 未实现 |
| #7 | 版本历史增强（变更摘要 + 回滚预览） | Proposal | ❌ 未实现 |

### P2 — 可以做

| 编号 | 内容 | 来源 |
|------|------|------|
| #5 | 规则模板（预设 + 自定义） | Proposal |
| #8 | 批量操作（简化版：循环调用现有 API） | Proposal |
| #9 | 细粒度权限（先加只读成员） | Proposal |

### P3 — 暂不做

| 编号 | 内容 | 原因 |
|------|------|------|
| #10 | 发布审批 | 无实际团队协作需求 |
| #3 | 定时发布 | 单环境，无需求 |
| #11 | 多环境管理 | 需要先有业务场景 |

---

## 七、优先执行建议

**第一周（阻塞性问题）:**
1. 修复 CreateUser 密码存储（N-001）— 1 小时
2. Session 持久化轻量方案（N-004）— 4 小时
3. 安全加固（N-006）— 4 小时
4. 超级管理员权限修复（N-009）— 30 分钟

**第二周（体验提升）:**
1. 资源搜索筛选（#4）— 8 小时
2. 规则计数优化（N-007）— 4 小时
3. 版本历史变更摘要（#7）— 4 小时
4. 审计日志（#6）— 8 小时

**第三周（锦上添花）:**
1. 规则模板（#5）— 12 小时
2. 批量操作简化版（#8）— 8 小时
3. 只读成员权限 — 4 小时
