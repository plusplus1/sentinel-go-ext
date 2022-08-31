# 用户权限体系 — 产品设计提案

⚠️ 当前状态：部分实现（2026-03-15）

---

## 📅 日期：2026-03-14

### 当前实现状态

| 组件 | 状态 | 说明 |
|------|------|------|
| 数据库表 | ✅ 已实现 | users, business_lines, business_line_apps, user_permissions, user_audit_logs, user_tokens |
| 模型层 | ✅ 已实现 | model/user.go 全部模型 |
| 服务层 | ✅ 已实现 | AuthService 完整功能 |
| 中间件 | ✅ 已实现 | AuthMiddleware, RequireRoleMiddleware, RequirePermissionMiddleware |
| API 处理器 | ⚠️ 部分实现 | auth.go 包含登录、用户管理、业务线管理、授权，但未应用中间件 |
| 前端登录页 | ✅ 已实现 | Login.tsx |
| 前端管理中心 | ⚠️ 部分实现 | Admin.tsx 存在但功能可能不完整 |
| 路由 | ⚠️ 部分实现 | 已注册但未应用认证中间件 |
| 飞书 SSO | ❌ 未实现 | |
| 会话持久化 | ❌ 未实现 | 当前仅内存存储 |
| 用户个人中心 | ❌ 未实现 | |
| 密码管理 | ❌ 未实现 | |
| 审计日志查看 | ❌ 未实现 | |
| 测试 | ❌ 未实现 | |

---

## 一、用户角色与权限体系

### 1.1 角色定义

| 角色 | 标识 | 权限范围 | 可见内容 |
|------|------|----------|----------|
| 超级管理员 | `super_admin` | 全局 | 所有业务线分类、成员管理、系统设置（不需要关联业务线/应用） |
| 业务线管理员 | `line_admin` | 业务线维度（可关联多个业务线） | 本业务线下的 App、模块、资源、成员（不关联应用，由超管设定） |
| 普通成员 | `member` | App/模块维度 | 被授权的 App、模块、资源（只能看到自己所在应用的范围） |

### 1.2 权限矩阵

| 功能 | 超级管理员 | 业务线管理员 | 普通成员 |
|------|-----------|-------------|---------|
| 创建业务线 | ✅（仅超管） | ❌ | ❌ |
| 查看所有业务线 | ✅ | 仅自己的（关联的业务线） | ❌ |
| 修改业务线（描述/状态） | ✅ | ✅（本业务线） | ❌ |
| 删除业务线（软删除） | ✅（仅超管） | ❌ | ❌ |
| 创建 App | ❌ | ✅（本业务线） | ❌ |
| 编辑 App 配置 | ❌ | ✅（本业务线） | ❌ |
| 删除 App（软删除） | ✅（仅超管） | ❌ | ❌ |
| 绑定用户到 App | ❌ | ✅（本业务线） | ❌ |
| 查看 App 列表 | ✅（全部） | ✅（本业务线下全部） | ✅（仅自己所在应用） |
| 权限管理（列表/撤销） | ✅（全部） | ✅（仅本业务线权限） | ❌（无批量操作） |
| 创建模块 | ❌ | ✅ | ✅（被授权 App） |
| 编辑模块 | ❌ | ✅ | ✅（模块成员） |
| 管理模块成员 | ❌ | ✅ | ✅（模块拥有者） |
| 查看/编辑资源 | ❌ | ✅ | ✅（模块成员） |
| 发布规则 | ❌ | ✅ | ✅（被授权） |
| 查看操作日志 | 全局 | 本业务线 | 自己 |

### 1.3 权限边界

**超级管理员**：
- 可以创建业务线分类（如"用户中心业务线"、"支付业务线"）
- 可以把成员设置为超管或其他角色
- **不能**操作业务线 App 下的具体配置（不展示业务线内部内容）
- 目的是保持管理的纯粹性，不干涉业务操作

**业务线管理员**：
- 可以在本业务线下创建二级 App
- 可以编辑 App 对应的内容（ETCD 链接配置等）
- 可以添加业务线普通成员
- 可以提权普通成员为业务线管理员
- **不能**跨业务线操作

**普通成员**：
- 可以被业务线管理员加入到某个 App
- 可以创建模块，或被模块拥有者拉到某个 Group
- 可以查看和编辑对应 Group 下的资源信息
- **不能**创建 App 或管理业务线

---

## 二、数据模型

### 2.1 业务线（Business Line）
```sql
CREATE TABLE business_lines (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL UNIQUE COMMENT '业务线名称',
    description VARCHAR(512) COMMENT '描述',
    status VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT '状态(active/deleted)',
    owner_id VARCHAR(64) COMMENT '负责人ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```
- 软删除：status='deleted'
- 唯一性：业务线名称唯一（UNIQUE KEY）
- 修改限制：不能修改名称，仅可修改描述/状态

### 2.2 用户表
```sql
CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL UNIQUE COMMENT '用户唯一ID',
    email VARCHAR(256) NOT NULL UNIQUE COMMENT '公司邮箱（登录账号）',
    name VARCHAR(128) NOT NULL COMMENT '显示名称',
    avatar_url VARCHAR(512) COMMENT '头像URL',
    password_hash VARCHAR(256) COMMENT '密码哈希（飞SSO登录则为空）',
    role VARCHAR(32) NOT NULL DEFAULT 'member' COMMENT '全局角色: super_admin/line_admin/member',
    feishu_user_id VARCHAR(64) COMMENT '飞书用户ID',
    status VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT '状态: active/disabled',
    last_login_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### 2.3 业务线-App 关联
```sql
CREATE TABLE business_line_apps (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    business_line_id BIGINT UNSIGNED NOT NULL,
    app_key VARCHAR(64) NOT NULL COMMENT '应用标识',
    app_name VARCHAR(128) NOT NULL COMMENT '应用名称',
    etcd_url VARCHAR(256) COMMENT 'ETCD链接配置',
    description VARCHAR(512),
    status VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT '状态(active/deleted)',
    created_by VARCHAR(64) COMMENT '创建者ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_line_app (business_line_id, app_key),
    UNIQUE KEY uk_line_app_name (business_line_id, app_name),
    KEY idx_business_line_id (business_line_id),
    KEY idx_app_key (app_key)
);
```
- 软删除：status='deleted'
- 唯一性：业务线内应用名称唯一（uk_line_app_name）
- 业务线内应用名称唯一约束

### 2.4 成员权限
```sql
CREATE TABLE user_permissions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    resource_type VARCHAR(32) NOT NULL COMMENT '资源类型: business_line/app/module/group',
    resource_id VARCHAR(128) NOT NULL COMMENT '资源ID',
    role VARCHAR(32) NOT NULL COMMENT '角色: admin/member/viewer/owner',
    granted_by VARCHAR(64) COMMENT '授权人ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_resource (user_id, resource_type, resource_id)
);
```

---

## 三、登录方式

### 3.1 账号密码登录
- 账号：公司邮箱
- 密码：用户自行设置（首次通过邮箱验证设置）
- 支持忘记密码（邮箱验证重置）

### 3.2 飞书 SSO 登录
- 点击「飞书登录」按钮
- OAuth 2.0 授权流程
- 首次登录自动创建用户记录
- 后续登录自动关联飞书用户 ID

### 3.3 登录流程
```
用户访问 → 检查登录状态
  ├─ 已登录 → 进入系统
  └─ 未登录 → 登录页面
       ├─ 账号密码 → 验证 → 进入系统
       └─ 飞书 SSO → OAuth → 进入系统
```

---

## 四、前端页面设计

### 4.1 侧边栏权限控制
```
超级管理员看到：
├── 管理中心（业务线管理、成员管理、角色设置）
└── 个人中心

业务线管理员看到：
├── 本业务线 App 列表
├── 模块管理（对有权限的 App）
├── 资源中心（对有权限的 App）
└── 个人中心

普通成员看到：
├── 已授权的 App
├── 模块管理（对有权限的 App）
├── 资源中心（对有权限的 App）
└── 个人中心
```

### 4.2 新增页面
- `/web/admin/lines` — 业务线管理（超管）
- `/web/admin/members` — 成员管理（超管）
- `/web/admin/apps` — App 分配（超管/业务线管理员）
- `/web/login` — 登录页面

---

## 五、API 设计

### 认证
| 方法 | 路径 | 说明 | 状态 |
|------|------|------|------|
| POST | /api/auth/login | 账号密码登录 | ✅ 已实现 |
| POST | /api/auth/feishu | 飞书 SSO 登录 | ❌ 未实现 |
| POST | /api/auth/logout | 退出登录 | ✅ 已实现 |
| GET | /api/auth/me | 获取当前用户信息 | ✅ 已实现 |

### 业务线管理
| 方法 | 路径 | 说明 | 状态 |
|------|------|------|------|
| GET | /api/admin/lines | 业务线列表 | ✅ 已实现 |
| POST | /api/admin/lines | 创建业务线（仅超管） | ✅ 已实现 |
| PUT | /api/admin/lines/:id | 更新业务线（仅描述/状态，不能改名称） | ✅ 已实现 |
| DELETE | /api/admin/lines/:id | 软删除业务线（status='deleted'，仅超管） | ✅ 已实现 |

### 应用管理（business_line_apps）
| 方法 | 路径 | 说明 | 状态 |
|------|------|------|------|
| GET | /api/apps?line_id=xxx | 获取应用列表（二级结构：业务线/应用） | ✅ 已实现 |
| POST | /api/apps | 创建应用（业务线管理员可创建本业务线应用） | ✅ 已实现 |
| PUT | /api/apps/:app_key | 更新应用（业务线管理员可修改本业务线应用） | ✅ 已实现 |
| DELETE | /api/apps/:app_key | 删除应用（软删除，仅超管） | ✅ 已实现 |

### 成员管理
| 方法 | 路径 | 说明 | 状态 |
|------|------|------|------|
| GET | /api/admin/members | 成员列表 | ✅ 已实现（ListUsers） |
| POST | /api/admin/members | 添加成员 | ✅ 已实现（CreateUser） |
| PUT | /api/admin/members/:id/role | 修改角色 | ✅ 已实现 |
| DELETE | /api/admin/members/:id | 删除成员 | ❌ 未实现 |

### 权限管理
| 方法 | 路径 | 说明 | 状态 |
|------|------|------|------|
| GET | /api/permissions | 获取权限列表（支持按用户/资源过滤） | ✅ 已实现 |
| POST | /api/permissions | 授权 | ✅ 已实现 |
| DELETE | /api/permissions/:id | 撤销权限（无批量操作） | ✅ 已实现 |

---

## 六、安全边界

1. **API 鉴权**：所有 API 需要登录态（Session/Cookie 或 Token）✅ 已实现（AuthMiddleware 已应用）✅ 已测试通过
2. **角色校验**：每个 API 校验用户角色是否允许操作 ✅ 已实现（RequireRoleMiddleware 已应用）✅ 已测试通过
3. **资源隔离**：业务线管理员只能操作本业务线资源 ✅ 已实现（权限控制逻辑已实现）✅ 已测试通过
4. **操作审计**：所有操作记录到 user_audit_logs ⚠️ 部分实现
5. **密码安全**：bcrypt 哈希存储，不存明文 ✅ 已实现
6. **SSO 安全**：飞书 OAuth 2.0，验证 state 防 CSRF ❌ 未实现
7. **Token 安全**：API Token 带权限范围和过期时间 ❌ 未实现

### 测试状态（2026-03-15）
- **Stage 2.1（业务线 CRUD）**：✅ 全部通过（创建、更新、软删除、唯一性约束）
- **Stage 2.3（应用管理 API）**：✅ 全部通过（CRUD、二级结构、权限控制）
- **Stage 2.4（权限管理 API）**：✅ 全部通过（列表、撤销、无批量操作）
- **认证中间件**：✅ 已应用，所有路由受保护
- **角色校验**：✅ 已应用，权限控制逻辑已实现
- **验收状态**：✅ 所有 Stage 2.1, 2.3, 2.4 要求已满足，验收通过
