# 实施任务清单：资源分组与资源中心视图（含 MySQL 架构升级）

## 任务概览
基于提案一（资源分组与资源中心视图）的设计文档，已完成实施。任务分为后端API实现、前端组件实现、MySQL架构升级、集成测试四个阶段。

**当前状态**: 阶段一、二、四已完成，阶段三已完成，阶段五（版本管理）未开始

---

## 阶段一：后端API实现 ✅ 已完成

### 1.1 存储结构实现
- [x] 创建分组元数据存储结构（MySQL `groups` 表）
- [x] 创建资源元数据存储结构（MySQL `resources` 表）
- [x] 创建流控规则存储结构（MySQL `flow_rules` 表）
- [x] 创建熔断规则存储结构（MySQL `circuit_breaker_rules` 表）
- [x] 创建发布记录存储结构（MySQL `publish_records` 表）
- [x] 实现分组DAO层 (dao/mysql.go - MySQLGroupDAO)
- [x] 实现资源DAO层 (dao/mysql.go - MySQLResourceDAO)
- [x] 实现流控规则DAO层 (dao/mysql.go - MySQLFlowRuleDAO)
- [x] 实现熔断规则DAO层 (dao/mysql.go - MySQLCBRuleDAO)

### 1.2 分组管理API
- [x] 实现GET /api/groups - 获取分组列表
- [x] 实现POST /api/groups - 创建分组
- [x] 实现GET /api/groups/:id - 获取分组详情
- [x] 实现PUT /api/groups/:id - 更新分组
- [x] 实现DELETE /api/groups/:id - 删除分组（move_to_default）
- [x] 实现分组成员管理API (添加/移除资源)
- [x] 实现GET /api/groups/:id/members - 获取分组成员

### 1.3 资源聚合查询API
- [x] 实现GET /api/resources - 资源列表（含模块归属）
- [x] 实现GET /api/resource/:id/rules - 聚合查询（流控+熔断规则）
- [x] 实现GET /api/resource/:id - 获取资源元数据
- [x] 实现PUT /api/resource/:id - 更新资源模块
- [x] 实现DELETE /api/resource/:id - 删除资源（支持 query ?id= 或 path /:id）
- [x] 实现规则快速操作API (启用/禁用 toggle)

### 1.4 规则管理API（MySQL底层）
- [x] 实现GET /api/app/rule/flow/list - 流控规则列表（MySQL）
- [x] 实现POST /api/app/rule/flow/update - 创建/更新流控规则（MySQL）
- [x] 实现POST /api/app/rule/flow/del - 删除流控规则（MySQL）
- [x] 实现GET /api/app/rule/circuitbreaker/list - 熔断规则列表（MySQL）
- [x] 实现POST /api/app/rule/circuitbreaker/update - 创建/更新熔断规则（MySQL）
- [x] 实现POST /api/app/rule/circuitbreaker/del - 删除熔断规则（MySQL）

### 1.5 发布API（MySQL → etcd）
- [x] 实现POST /api/publish - 发布规则到 etcd
- [x] 实现发布服务 (service/publish_service.go)

---

## 阶段二：前端组件实现 ✅ 已完成

### 2.1 模块管理页面（原分组管理）
- [x] 创建Groups.tsx页面组件
- [x] 实现模块列表（表格展示）
- [x] 实现创建/编辑/删除模块
- [x] 实现成员管理抽屉（添加/移除资源）
- [x] 命名统一：分组 → 模块

### 2.2 资源中心页面
- [x] 创建Resources.tsx页面组件
- [x] 实现资源列表（含流控/熔断规则数列）
- [x] 实现规则详情弹窗（流控规则 + 熔断规则分组）
- [x] 实现规则编辑弹窗（流控规则表单）
- [x] 实现规则编辑弹窗（熔断规则表单）
- [x] 实现规则创建功能（空状态添加按钮）
- [x] 实现规则切换开关（启用/禁用）
- [x] 实现变更模块功能
- [x] 实现添加资源功能
- [x] 实现删除资源功能

### 2.3 路由与导航
- [x] 添加模块管理路由 /web/modules
- [x] 添加资源中心路由 /web/resources
- [x] 更新侧边栏菜单（移除旧的流控/熔断入口）
- [x] 默认路由改为 /web/resources

---

## 阶段三：MySQL 架构升级 ✅ 已完成

### 3.1 数据库设计
- [x] 设计完整库表结构（15张表）
- [x] 创建SQL初始化脚本 (dashboard/sql/schema.sql)
- [x] MySQL 5.7 安装与配置
- [x] 数据库和表创建

### 3.2 MySQL DAO 层
- [x] 实现MySQLGroupDAO（模块 CRUD）
- [x] 实现MySQLResourceDAO（资源 CRUD）
- [x] 实现MySQLFlowRuleDAO（流控规则 CRUD + Toggle）
- [x] 实现MySQLCBRuleDAO（熔断规则 CRUD + Toggle）
- [x] 实现MySQLPublishRecordDAO（发布记录）

### 3.3 底层切换
- [x] 重写 group.go — 所有 handler 使用 MySQL
- [x] 重写 resource.go — 所有 handler 使用 MySQL
- [x] API 接口完全不变，仅底层变更
- [x] go-sql-driver/mysql v1.8.1（兼容 Go 1.20）
- [x] 修复 model.Group（添加 IsDefault, MemberCount 字段）
- [x] 修复 model.Resource（添加 ID 字段）
- [x] 删除旧 etcd DAO（group_dao.go, resource_dao.go）
- [x] 删除旧服务层（group_service.go, resource_service.go）

### 3.4 发布服务
- [x] 实现PublishService（MySQL → etcd 同步）
- [x] 支持单资源发布
- [x] 支持全量发布
- [x] 发布记录自动记录

---

## 阶段四：集成测试 ✅ 已完成

### 4.1 API 测试
- [x] 分组管理 API 全部通过
- [x] 资源管理 API 全部通过
- [x] 规则管理 API 全部通过
- [x] 资源聚合 API 全部通过
- [x] MySQL 底层切换测试通过

### 4.2 前端测试
- [x] 模块管理页面功能验证
- [x] 资源中心页面功能验证
- [x] 规则编辑功能验证
- [x] 前端构建成功（npm run build）

### 4.3 Bug 修复
- [x] 修复 UpdateResourceGroup 未更新组成员列表
- [x] 修复流控规则 API 参数格式（query vs JSON body）
- [x] 修复删除模块需要 JSON body
- [x] 修复编辑模块需要 app_id/env
- [x] 修复添加资源需要 app/env 查询参数
- [x] 修复规则为空时后端返回 stub 数据
- [x] 修复 mysql driver v1.9.3 不兼容 Go 1.20
- [x] 修复 FlowRuleRecord ClusterConfig NULL 扫描问题
- [x] 修复资源 API 使用 name 而非 id 的问题

---

## 阶段五：版本管理 ⚠️ 未开始

### 5.1 数据库设计
- [x] 创建 publish_versions 表（版本快照存储）
- [ ] 修改 publish_records 表（增加版本关联）

### 5.2 后端实现
- [ ] 实现版本号自增逻辑
- [ ] 发布时自动创建版本快照（JSON 格式存储所有规则）
- [ ] GET /api/versions - 版本列表（路由已注册，handler 未实现）
- [ ] GET /api/versions/:id - 版本详情（路由已注册，handler 未实现）
- [ ] POST /api/versions/:id/rollback - 回滚到指定版本（路由已注册，handler 未实现）
- [ ] POST /api/versions/:id/compare - 版本对比

### 5.3 前端实现
- [ ] 发布弹窗增加版本描述输入
- [ ] 版本历史页面（版本列表 + 时间线）
- [ ] 版本详情弹窗（快照内容展示）
- [ ] 回滚确认弹窗
- [ ] 版本对比视图（显示差异）

### 5.4 验收标准
- [ ] 每次发布自动生成版本快照
- [ ] 版本列表按时间倒序
- [ ] 回滚功能正常（恢复到历史版本的规则）
- [ ] 版本对比显示新增/删除/修改的规则

---

## 库表结构（最终版）

### 已实现表（15 张）
| 表名 | 说明 | 状态 |
|------|------|------|
| `apps` | 应用 | ✅ |
| `groups` | 业务模块 | ✅ |
| `resources` | 资源 | ✅ |
| `flow_rules` | 流控规则 | ✅ |
| `circuit_breaker_rules` | 熔断规则 | ✅ |
| `system_rules` | 系统规则 | ✅ |
| `hotspot_rules` | 热点参数规则 | ✅ |
| `publish_records` | 发布记录 | ✅ |
| `publish_versions` | 版本快照 | ✅ |
| `users` | 用户表 | ⚠️ 设计中 |
| `business_lines` | 业务线表 | ⚠️ 设计中 |
| `business_line_apps` | 业务线-App 关联 | ⚠️ 设计中 |
| `user_permissions` | 用户权限 | ⚠️ 设计中 |
| `user_audit_logs` | 审计日志 | ⚠️ 设计中 |
| `user_tokens` | API Token | ⚠️ 设计中 |

### 索引设计
- 唯一索引：`uk_app_env_name`（防止重复）
- 普通索引：`idx_app_env`（按应用+环境查询）
- 普通索引：`idx_group_id`（按分组查询资源）
- 唯一索引：`uk_rule_id`（规则唯一标识）

---

## API 清单（最终版）

### 模块管理（原分组管理）
| 方法 | 路径 | 说明 | 数据源 |
|------|------|------|--------|
| GET | /api/groups | 模块列表 | MySQL |
| POST | /api/groups | 创建模块 | MySQL |
| GET | /api/groups/:id | 模块详情 | MySQL |
| PUT | /api/groups/:id | 编辑模块 | MySQL |
| DELETE | /api/groups/:id | 删除模块 | MySQL |
| GET | /api/groups/:id/members | 成员列表 | MySQL |
| POST | /api/groups/:id/members | 添加成员 | MySQL |
| DELETE | /api/groups/:id/members/:resource | 移除成员 | MySQL |

### 资源中心
| 方法 | 路径 | 说明 | 数据源 |
|------|------|------|--------|
| GET | /api/resources | 资源列表 | MySQL |
| GET | /api/resource/:id | 资源元数据 | MySQL |
| GET | /api/resource/:id/rules | 资源规则详情 | MySQL |
| PUT | /api/resource/:id | 变更模块 | MySQL |
| DELETE | /api/resource/:id | 删除资源 | MySQL |

### 规则管理
| 方法 | 路径 | 说明 | 数据源 |
|------|------|------|--------|
| GET | /api/app/rule/flow/list | 流控规则列表 | MySQL |
| POST | /api/app/rule/flow/update | 创建/更新流控 | MySQL |
| POST | /api/app/rule/flow/del | 删除流控 | MySQL |
| GET | /api/app/rule/circuitbreaker/list | 熔断规则列表 | MySQL |
| POST | /api/app/rule/circuitbreaker/update | 创建/更新熔断 | MySQL |
| POST | /api/app/rule/circuitbreaker/del | 删除熔断 | MySQL |
| PUT | /api/resource/:id/flow/:rule_id/toggle | 流控开关 | MySQL |
| PUT | /api/resource/:id/circuitbreaker/:rule_id/toggle | 熔断开关 | MySQL |

### 发布管理
| 方法 | 路径 | 说明 | 数据源 |
|------|------|------|--------|
| POST | /api/publish | 发布到 etcd | MySQL→etcd |

### 版本管理（未实现）
| 方法 | 路径 | 说明 | 数据源 | 状态 |
|------|------|------|--------|------|
| GET | /api/versions | 版本列表 | MySQL | ⚠️ 未实现 |
| GET | /api/versions/:id | 版本详情 | MySQL | ⚠️ 未实现 |
| POST | /api/versions/:id/rollback | 回滚 | MySQL | ⚠️ 未实现 |

---

## 验收标准 ✅ 全部通过

### 后端验收
- [x] 所有API端点正常工作，返回正确响应
- [x] 分组CRUD操作正常
- [x] 资源分组归属管理正常
- [x] 资源聚合查询正常
- [x] 规则 CRUD 正常
- [x] 规则开关功能正常
- [x] MySQL 读写正常
- [x] 错误处理完善，返回友好错误信息

### 前端验收
- [x] 模块管理页面功能完整
- [x] 资源中心页面功能完整
- [x] 规则编辑功能正常
- [x] 规则创建功能正常
- [x] 规则开关功能正常
- [x] 添加/删除资源功能正常
- [x] 变更模块功能正常

---

## 更新日志
- 2026-03-14 08:56: 创建任务清单，开始实施
- 2026-03-14 09:00: 完成后端 DAO + Service + API 层
- 2026-03-14 09:20: 完成所有后端 API 测试
- 2026-03-14 09:45: 发现前端是 React（非 Vue.js），开始创建前端组件
- 2026-03-14 10:09: 完成前端 UI 实现（Groups.tsx, Resources.tsx）
- 2026-03-14 10:28: 用户验收反馈：命名统一、规则编辑、去掉旧入口
- 2026-03-14 10:55: 完成改进（模块命名、规则编辑、添加/删除资源）
- 2026-03-14 11:35: 修复多个 bug（API 参数、后端 stub、null 返回值）
- 2026-03-14 12:05: MySQL 测试环境搭建完成
- 2026-03-14 12:20: 底层切换到 MySQL，API 接口不变
- 2026-03-15: 完成资源 API ID 化，删除 etcd DAO，简化架构
- 2026-03-15: 更新 openspec 全量文档，校正信息
