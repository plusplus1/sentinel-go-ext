# 变更日志

## 2026-03-15（管理员角色权限隔离）

### 后端
- **auth.go 文件拆分**：
  - `auth.go` - 公共部分（数据库连接、认证服务、飞书SSO、登录/登出/获取当前用户）
  - `auth_admin.go` - 超级管理员 API
  - `auth_line.go` - 业务线管理员 API

- **超级管理员 API**（`/api/admin/`）：
  - 用户管理：`GET /api/admin/users`、`POST /api/admin/users`、`GET /api/admin/users/search`
  - 业务线管理：`GET/POST/PUT/DELETE /api/admin/lines`
  - 管理员绑定：`POST /api/admin/lines/:id/owner`、`DELETE /api/admin/lines/:id/owner`
  - 应用管理：`GET/POST/PUT/DELETE /api/apps`
  - 权限管理：`GET/POST /api/permissions`、`DELETE /api/permissions/:id`

- **业务线管理员 API**（`/line-admin/`）：
  - `GET /line-admin/lines` - 查看自己管理的业务线
  - `PUT /line-admin/lines/:id` - 修改业务线描述
  - `GET /line-admin/lines/:id/apps` - 查看业务线内的应用
  - `POST /line-admin/lines/:id/apps` - 创建应用
  - `PUT /line-admin/lines/:id/apps/:app_id` - 修改应用
  - `DELETE /line-admin/lines/:id/apps/:app_id` - 删除应用

- **接口清理**：
  - 删除 `PUT /api/admin/users/:user_id/role`（前端未使用）

### 前端
- **Admin.tsx 重构**：
  - 根据用户角色渲染不同的面板组件
  - `SuperAdminPanel` - 超级管理员界面
  - `LineAdminPanel` - 业务线管理员界面（卡片式布局）
  - 移除页面标题+用户状态显示

- **SuperAdminPanel**：
  - 业务线列表（active/deleted tabs）
  - 创建业务线按钮与 tab 平齐
  - 修改描述、绑定/移除管理员、激活/下线

- **LineAdminPanel**：
  - 每个业务线作为卡片
  - App 横向排列，卡片式展示
  - 支持创建、修改、移除应用

### Bug 修复
- **2026-03-15（前端 API 路径）**：
  - 问题：`Admin.tsx` 中 LineAdminPanel 的 API 路径缺少 `/api/` 前缀
  - 修复：所有 `/line-admin/` 改为 `/api/line-admin/`（共 6 处）

## 2026-03-15（数据库架构升级）
- **数据库结构调整**：
  - `business_line_apps` 表：添加 `app_id` 外键字段，删除 `app_key`、`app_name`、`etcd_url`、`description` 字段
  - `groups` 表：添加 `app_id` 外键字段，删除 `app_key` 和 `env` 字段
  - 新增 `group_owners` 表：建立 group 与 user 的多对多关系
- **API 路径统一基于 ID**：所有资源的更新和删除操作均使用 ID，而非 key 或 name
- **超级管理员页面优化**：
  - 新增业务线管理 API：绑定/移除管理员、搜索用户
  - 动态更新逻辑：只更新传入的字段
- **经验教训**：
  - 所有资源的更新和删除操作必须基于 ID
  - 更新操作应使用动态更新逻辑
  - 所有管理接口必须添加登录验证和权限检查

## 2026-03-15（数据库架构调整：合并 apps 表到 business_line_apps）

### 数据库变更
- **删除 `apps` 表**，字段合并到 `business_line_apps`
- **`business_line_apps` 表结构调整**：
  - `id` - 主键，同时代表 app_id
  - `business_line_id` - 业务线 ID
  - `app_key` - 应用标识（英文、数字、下划线，3-50字符，唯一）
  - `description` - 应用描述（中文）
  - `settings` - etcd 配置（JSON）
  - `status` - 应用状态（active/deleted）
  - `created_at` / `updated_at` - 时间戳
  - 唯一约束：`(business_line_id, app_key)`

### 后端变更
- **auth_admin.go**：
  - `ListApps` - 直接从 `business_line_apps` 查询，不再 JOIN apps 表
  - `ListBusinessLineApps` - 返回 app_key, description, status
  - `CreateApp` - 直接在 business_line_apps 创建记录，验证 app_key 格式
  - `UpdateApp` - 更新 business_line_apps 记录，验证 app_key 格式
  - `DeleteApp` - 软删除 business_line_apps 记录
  - `AssociateAppWithBusinessLine` - 重写为创建应用
  - `DisassociateAppFromBusinessLine` - 直接删除记录
- **auth_line.go**：
  - `CreateBusinessLineApp` - 直接在 business_line_apps 创建记录
  - `UpdateBusinessLineApp` - 直接更新 business_line_apps 记录
  - `DeleteBusinessLineApp` - 直接删除记录

### 前端变更
- **Admin.tsx**：
  - `AppInfo` 接口更新：`app_key`, `description`, `status`, `created_at`
  - 表单字段：`app_key`（应用标识）、`description`（应用描述）
  - API 调用适配新的字段名

### 经验教训
- **表结构优化**：当两个表存在强关联且频繁一起查询时，考虑合并为一个表
- **app_key 作为唯一标识**：使用有意义的标识符，而非随机哈希值
- **字段命名**：`app_key` 表示应用标识，`description` 表示中文描述，语义更清晰
