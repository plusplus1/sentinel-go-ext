⚠️ 当前状态：部分实现（2026-03-15）

### 已实现组件
- ✅ 数据库表（6张表）
- ✅ 模型层（model/user.go）
- ✅ 服务层（AuthService 完整功能）
- ✅ 中间件（AuthMiddleware, RequireRoleMiddleware, RequirePermissionMiddleware）
- ✅ API 处理器（auth.go - 登录、用户管理、业务线管理、授权）
- ✅ 路由注册（install.go）
- ✅ 前端登录页（Login.tsx）
- ✅ 前端管理中心（Admin.tsx）

### 未实现组件
- ❌ 路由中间件应用（已实现但未应用到路由）
- ❌ 业务线 CRUD 完整（缺更新、删除）
- ❌ 应用管理 API
- ❌ 权限列表/撤销 API
- ❌ 审计日志查看 API
- ❌ 用户个人中心 API
- ❌ 密码管理
- ❌ 飞书 SSO
- ❌ 会话持久化
- ❌ 测试

---

# 用户权限体系 — 技术架构设计

## 📅 日期：2026-03-14

---

## 一、系统架构

### 1.1 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      前端层 (React)                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │ 登录页面  │ │ 管理中心  │ │ 业务工作台 │ │ 个人中心      │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
│         │            │            │              │           │
│         └────────────┴────────────┴──────────────┘           │
│                        │ HTTP + Auth Token                   │
└────────────────────────┼────────────────────────────────────┘
                         │
┌────────────────────────┼────────────────────────────────────┐
│                   API 网关层 (Gin)                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐     │
│  │ 认证中间件   │  │ 鉴权中间件   │  │ 审计日志中间件  │     │
│  └─────────────┘  └─────────────┘  └─────────────────┘     │
│         │                │                    │             │
└─────────┼────────────────┼────────────────────┼─────────────┘
          │                │                    │
┌─────────┼────────────────┼────────────────────┼─────────────┐
│         ▼                ▼                    ▼             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐     │
│  │ Auth Service│  │ Perm Service│  │ Audit Service   │     │
│  └─────────────┘  └─────────────┘  └─────────────────┘     │
│         │                │                    │             │
│  ┌──────┴────────────────┴────────────────────┴──────────┐ │
│  │                    MySQL DAO 层                       │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                           │
│  MySQL: users, business_lines, user_permissions,           │
│         user_tokens, user_audit_logs                      │
│                                                           │
│  etcd:  仅发布规则（Sentinel 客户端消费）                   │
└───────────────────────────────────────────────────────────┘
```

### 1.2 认证流程

```
用户登录 → Auth Service → 验证凭证 → 生成 Session Token
                                              │
                                              ▼
                                    写入 Cookie/Token
                                              │
                                              ▼
                                    后续请求携带 Token
                                              │
                                              ▼
                                    认证中间件验证 Token
                                              │
                                              ▼
                                    鉴权中间件检查角色
                                              │
                                              ▼
                                    API Handler 执行
```

### 1.3 鉴权流程

```
请求进入 → 认证中间件（提取用户信息）
              │
              ▼
         鉴权中间件（检查权限）
              │
        ┌─────┴─────┐
        ▼           ▼
    超级管理员    普通用户
        │           │
        ▼           ▼
    全局操作    检查 resource_type + resource_id
                    │
              ┌─────┴─────┐
              ▼           ▼
          有权限        无权限
              │           │
              ▼           ▼
          继续执行    返回 403
```

---

## 二、技术方案

### 2.1 认证方案

**账号密码登录**：
```
POST /api/auth/login
Body: { "email": "user@company.com", "password": "xxx" }
流程: email 查 users 表 → bcrypt 验证密码 → 生成 token → 写入 session
```

**飞书 SSO 登录**：
```
1. 前端重定向到飞书 OAuth URL
2. 飞书授权后回调到 /api/auth/feishu/callback
3. 后端用 code 换取 access_token
4. 用 access_token 获取用户信息
5. 查找或创建 users 记录（关联 feishu_user_id）
6. 生成 token → 写入 session
```

**Session 方案**：
- 使用 Cookie + Session ID（服务端存储）
- 或 JWT Token（无状态，带过期时间）
- 推荐：Cookie + Session（更安全，支持服务端撤销）

### 2.2 鉴权方案

**中间件设计**：
```go
// 认证中间件 - 提取用户信息
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        sessionToken := c.Cookie("session_token")
        user := authService.ValidateSession(sessionToken)
        if user == nil {
            c.JSON(401, gin.H{"error": "未登录"})
            c.Abort()
            return
        }
        c.Set("user", user)
        c.Next()
    }
}

// 鉴权中间件 - 检查角色
func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := c.MustGet("user").(*model.User)
        for _, role := range roles {
            if user.Role == role {
                c.Next()
                return
            }
        }
        c.JSON(403, gin.H{"error": "权限不足"})
        c.Abort()
    }
}

// 资源鉴权中间件 - 检查业务线权限
func RequireResourceAccess(resourceType string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := c.MustGet("user").(*model.User)
        resourceId := c.Param("id")
        if !permService.HasAccess(user.UserID, resourceType, resourceId) {
            c.JSON(403, gin.H{"error": "无权访问该资源"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 2.3 数据库方案

**用户创建**：
- 首次密码登录：通过邀请邮件或管理员创建
- 首次飞书登录：自动创建记录

**权限存储**：
- 全局角色存在 users.role 字段
- 资源级权限存在 user_permissions 表
- 查询时合并两者

**审计日志**：
- 所有写操作自动记录到 user_audit_logs
- 记录：用户ID、操作类型、资源、详情、IP、时间

### 2.4 前端方案

**路由守卫**：
```tsx
// 根据用户角色渲染不同菜单
const menuItems = useMemo(() => {
  if (user.role === 'super_admin') {
    return [adminMenu, personalCenter];
  }
  if (user.role === 'line_admin') {
    return [businessLineMenu, modulesMenu, resourcesMenu, personalCenter];
  }
  return [authorizedAppsMenu, modulesMenu, resourcesMenu, personalCenter];
}, [user.role]);

// 路由守卫
<Route path="/admin/*" element={
  <RequireRole role="super_admin">
    <AdminPages />
  </RequireRole>
} />
```

**登录页面**：
```
┌─────────────────────────────────┐
│         流云哨 登录              │
│                                 │
│  ┌─────────────────────────┐   │
│  │ 邮箱                    │   │
│  ├─────────────────────────┤   │
│  │ 密码                    │   │
│  └─────────────────────────┘   │
│  [ 登录 ]                      │
│                                 │
│  ─── 或 ───                    │
│                                 │
│  [ 飞书登录 ]                   │
└─────────────────────────────────┘
```

---

## 三、API 设计详细

### 3.1 认证 API

```yaml
POST /api/auth/login:
  body: { email, password }
  response: { code, data: { user, token } }
  
POST /api/auth/feishu:
  body: { code }  # OAuth code
  response: { code, data: { user, token } }

POST /api/auth/logout:
  response: { code }

GET /api/auth/me:
  response: { code, data: { user, permissions } }
```

### 3.2 业务线 API（超管）

```yaml
GET /api/admin/lines:
  response: { code, data: [business_lines] }
  
POST /api/admin/lines:
  body: { name, description }
  response: { code, data: { line } }

PUT /api/admin/lines/:id:
  body: { name, description }
  response: { code }

DELETE /api/admin/lines/:id:
  response: { code }
```

### 3.3 App 管理 API（业务线管理员）

```yaml
GET /api/admin/apps?line_id=xxx:
  response: { code, data: [apps] }
  
POST /api/admin/apps:
  body: { line_id, app_name, etcd_url, description }
  response: { code, data: { app } }

PUT /api/admin/apps/:app_key:
  body: { app_name, etcd_url, description }
  response: { code }
```

### 3.4 权限 API

```yaml
GET /api/permissions?resource_type=xxx&resource_id=xxx:
  response: { code, data: [permissions] }
  
POST /api/permissions:
  body: { user_id, resource_type, resource_id, role }
  response: { code }

DELETE /api/permissions/:id:
  response: { code }
```

---

## 四、安全设计

### 4.1 密码安全
- bcrypt 哈希（cost=12）
- 最少 8 位，包含字母+数字
- 登录失败 5 次锁定 15 分钟

### 4.2 Session 安全
- Session ID 32 字节随机
- 24 小时过期
- 支持多设备同时登录（最多 5 个）
- 可查看并撤销其他设备

### 4.3 飞书 SSO 安全
- OAuth state 参数防 CSRF
- code 单次使用，5 分钟过期
- 验证 redirect_uri

### 4.4 API 安全
- 所有 API 需要认证（除登录）
- 角色校验中间件
- 资源级权限校验
- 操作审计日志
