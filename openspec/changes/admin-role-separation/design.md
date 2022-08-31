# 管理员角色权限隔离设计文档

## 背景

原始需求：超级管理员和业务线管理员的管理中心页面和 API 应该完全隔离，各自有独立的功能和界面。

## 设计决策

### 1. API 路径隔离

**决策**：使用不同的 URL 前缀隔离两类管理员的 API。

| 角色 | 路径前缀 | 说明 |
|------|----------|------|
| 超级管理员 | `/api/admin/` | 用户管理、业务线完整 CRUD、绑定管理员 |
| 业务线管理员 | `/api/line-admin/` | 查看自己的业务线、修改描述、管理应用 |

**理由**：
- 路由级隔离比函数内权限检查更清晰
- 前端可以根据角色直接调用不同的 API，减少条件判断
- 安全性更高：两类 API 物理隔离，不会误调用

### 2. 超级管理员功能

| API | 说明 |
|-----|------|
| `GET /api/admin/users` | 用户列表 |
| `POST /api/admin/users` | 创建用户 |
| `GET /api/admin/users/search?keyword=` | 搜索用户 |
| `GET /api/admin/lines` | 业务线列表（含管理员信息） |
| `POST /api/admin/lines` | 创建业务线 |
| `PUT /api/admin/lines/:id` | 更新业务线（描述、状态） |
| `DELETE /api/admin/lines/:id` | 删除业务线（软删除） |
| `POST /api/admin/lines/:id/owner` | 绑定业务线管理员 |
| `DELETE /api/admin/lines/:id/owner` | 移除业务线管理员 |
| `GET /api/permissions` | 权限列表 |
| `POST /api/permissions` | 授权 |
| `DELETE /api/permissions/:id` | 撤销权限 |
| `GET /api/apps` | 应用列表（二级结构） |
| `POST /api/apps` | 创建应用 |
| `PUT /api/apps/:app_id` | 更新应用 |
| `DELETE /api/apps/:app_id` | 删除应用 |

### 3. 业务线管理员功能

| API | 说明 |
|-----|------|
| `GET /line-admin/lines` | 查看自己管理的业务线 |
| `PUT /line-admin/lines/:id` | 修改业务线描述（不能改名称/状态） |
| `GET /line-admin/lines/:id/apps` | 查看业务线内的应用 |
| `POST /line-admin/lines/:id/apps` | 在业务线内创建应用 |
| `PUT /line-admin/lines/:id/apps/:app_id` | 修改业务线内的应用 |
| `DELETE /line-admin/lines/:id/apps/:app_id` | 删除业务线内的应用 |

### 4. 前端组件分离

**决策**：根据用户角色渲染不同的面板组件。

```
Admin.tsx
├── SuperAdminPanel  # 超级管理员面板
│   ├── 业务线列表（active/deleted tabs）
│   ├── 创建业务线
│   ├── 修改描述
│   ├── 绑定/移除管理员
│   └── 激活/下线
└── LineAdminPanel    # 业务线管理员面板（卡片式布局）
    ├── LineCard[]      # 每个业务线一个卡片
    │   ├── 业务线标题（名称 + 状态 + 描述）
    │   ├── 修改描述按钮
    │   ├── 创建app按钮
    │   └── App 卡片列表
    │       ├── App名称
    │       ├── 修改按钮
    │       └── 移除按钮
    └── 修改描述 Modal
    └── 创建应用 Modal
    └── 修改应用 Modal
```

#### Line Admin UI 设计

**布局**：每个业务线作为一个卡片，横向展示 app 列表。

```
业务线名称 [生效] 业务线描述                    [修改描述] [创建app]
┌──────────┐  ┌──────────┐  ┌──────────┐
│  app1    │  │  app2    │  │  app3    │
│  [修改]  │  │  [修改]  │  │  [修改]  │
│  [移除]  │  │  [移除]  │  │  [移除]  │
└──────────┘  └──────────┘  └──────────┘

另一个业务线 [生效] 业务线描述                [修改描述] [创建app]
┌──────────┐  ┌──────────┐
│  app1    │  │  app2    │
│  [修改]  │  │  [修改]  │
│  [移除]  │  │  [移除]  │
└──────────┘  └──────────┘
```

**特点**：
- 业务线作为卡片标题，显示名称、状态、描述
- 每个 app 以小卡片形式横向排列（宽度 180px）
- App 卡片包含应用名称、状态、修改/移除操作按钮
- 右上角有"修改描述"和"创建app"按钮
- 支持空状态展示（暂无应用）

## 权限检查

### 后端权限检查

1. **超级管理员专属接口**：检查 `user.Role == "super_admin"`，否则返回 403
2. **业务线管理员专属接口**：
   - 检查 `user.Role == "line_admin"`，否则返回 403
   - 检查 `business_line.owner_id == user.UserID`，否则返回 403
3. **共享接口**：允许两种角色访问，内部根据角色返回不同数据

### Middleware 配置

```go
// 超级管理员路由组
superAdminGroup := protectedGroup.Group("/admin", base.RequireRoleMiddleware("super_admin"))

// 业务线管理员路由组
lineAdminGroup := protectedGroup.Group("/line-admin", base.RequireRoleMiddleware("line_admin"))
```

## 经验教训

### 1. 文件拆分原则

**问题**：`auth.go` 文件超过 1800 行，难以维护。

**解决方案**：按职责拆分为三个文件：
- `auth.go` - 公共部分（数据库连接、认证服务、飞书SSO、登录/登出）
- `auth_admin.go` - 超级管理员 API（用户管理、业务线管理、应用管理、权限管理）
- `auth_line.go` - 业务线管理员 API（自己的业务线、管理应用）

**教训**：
- 单个文件超过 500 行就应考虑拆分
- 按角色/职责拆分比按功能拆分更清晰
- 文件名应清晰表达内容

### 2. 动态更新查询

**问题**：`UpdateBusinessLine` 使用静态 SQL，修改描述时会清空状态。

**解决方案**：使用动态 SET 子句，只更新传入的字段。

```go
setClauses := []string{}
args := []interface{}{}

if req.Description != "" {
    setClauses = append(setClauses, "description = ?")
    args = append(args, req.Description)
}
if req.Status != "" {
    setClauses = append(setClauses, "status = ?")
    args = append(args, req.Status)
}
```

**教训**：
- 更新操作必须使用动态查询
- 永远不要用空值覆盖未传入的字段
- 测试时要覆盖"部分更新"场景

### 3. 前端角色分离

**问题**：超级管理员和业务线管理员的功能完全不同，不应共用同一套 UI。

**解决方案**：根据角色渲染不同的面板组件。

**教训**：
- 权限差异大的角色应该有独立的 UI 组件
- 不要试图用一个组件处理所有角色的逻辑
- 前端应通过 API 获取用户角色，后端根据角色返回不同数据

### 4. 接口清理

**问题**：`PUT /api/admin/users/:user_id/role` 接口前端未使用，但仍然保留。

**解决方案**：删除未使用的接口，保持 API 清洁。

**教训**：
- 定期检查 API 使用情况，删除未使用的接口
- 前端和后端应同步更新，避免接口残留
- 删除前确认前端确实未使用

### 5. API 路径设计

**问题**：超级管理员和业务线管理员的 API 混在一起，权限检查复杂。

**解决方案**：使用不同的 URL 前缀隔离。

**教训**：
- 物理隔离比逻辑隔离更安全
- 路径应清晰表达权限级别
- 前端调用时不需要额外的权限判断

## 表单字段设计

### 新建应用表单
| 字段 | 标识 | 类型 | 必填 | 验证 | 说明 |
|------|------|------|------|------|------|
| 应用标识 | `app_key` | Input | 是 | `^[a-zA-Z0-9_]{3,50}$` | 英文、数字、下划线，3-50字符 |
| 应用描述 | `description` | TextArea | 是 | - | 中文描述 |
| etcd配置 | `settings` | TextArea | 否 | JSON 格式 | 要发布到 etcd 的配置 |

### 修改应用表单
- 同新建应用表单，但所有字段都显示当前值
