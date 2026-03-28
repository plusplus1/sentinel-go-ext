# Sentinel Dashboard — 流量哨兵控制台

## 系统定位

Sentinel Dashboard 是一个**流量哨兵控制台**，用于管理和监控微服务的流量控制（限流）和熔断规则。采用 MySQL 作为配置存储，etcd 作为运行时规则推送目标。

### 核心能力
- **服务模块管理**：按业务模块组织资源，批量管理规则
- **资源中心**：统一查看所有资源及其规则状态
- **规则管理**：流控规则、熔断规则的 CRUD 和发布
- **用户权限**：超级管理员、业务线管理员、普通成员三级权限
- **发布机制**：配置修改后发布到 etcd，Sentinel 客户端实时消费
- **版本管理**：规则版本快照与回滚

---

## 架构

```
┌─────────────────────────────────────────────────┐
│              React + Ant Design 前端             │
│         资源中心 | 模块管理 | 规则编辑            │
└──────────────────────┬──────────────────────────┘
                       │ HTTP
┌──────────────────────▼──────────────────────────┐
│            Go + Gin Web 框架 (Dashboard)         │
│  模块API | 资源API | 规则API | 发布API           │
└──────┬───────────────────────────────┬──────────┘
       │ 读写                          │ 发布时
       ▼                               ▼
┌──────────────┐              ┌──────────────┐
│   MySQL 9.2  │              │   etcd v3    │
│  (主存储)     │              │  (发布目标)   │
└──────────────┘              └──────────────┘
```

### 数据流
1. **编辑配置** → 写入 MySQL
2. **发布配置** → 从 MySQL 读取 → 写入 etcd
3. **客户端消费** → 从 etcd 拉取规则 → 应用限流/熔断

---

## 技术栈

### 后端
- **语言**: Go 1.20
- **Web 框架**: Gin v1.9.1
- **MySQL 驱动**: go-sql-driver/mysql v1.8.1
- **etcd 客户端**: go.etcd.io/etcd/client/v3 v3.5.11
- **JSON**: Sonic v1.10.2
- **日志**: Uber Zap v1.26.0

### 前端
- **框架**: React 18 + TypeScript
- **构建工具**: Vite 8.0.0
- **UI 库**: Ant Design 6.x
- **路由**: React Router v6
- **HTTP 客户端**: Axios

### 存储
- **主存储**: MySQL 9.2（127.0.0.1:3306）
- **发布目标**: etcd v3（127.0.0.1:2379）

---

## 项目结构

```
sentinel-go-ext/
├── dashboard/                  # 后端（全部代码在此）
│   ├── api/
│   │   ├── app/               # API handlers
│   │   │   ├── auth.go        # 认证 API（登录/退出/SSO）
│   │   │   ├── auth_admin.go  # 管理中心 API（超管）
│   │   │   ├── auth_line.go   # 业务线管理 API（线管）
│   │   │   ├── group.go       # 模块管理 API
│   │   │   ├── resource.go    # 资源中心 API
│   │   │   ├── rule_flow.go   # 流控规则 API
│   │   │   └── rule_circuitbreaker.go
│   │   └── base/              # 鉴权中间件、应用注册
│   ├── cmd/                   # 命令行入口
│   ├── config/                # YAML 配置加载
│   ├── dao/                   # MySQL DAO
│   ├── model/                 # 数据模型
│   ├── service/               # 业务服务（认证、发布）
│   ├── source/                # etcd 连接
│   ├── sql/                   # 数据库脚本
│   ├── util/                  # 工具函数
│   └── install.go             # 路由注册
├── frontend/                  # 前端 React 应用
│   ├── src/
│   │   ├── pages/             # 页面组件
│   │   ├── context/           # React Context
│   │   └── App.tsx            # 主应用 + 路由
│   └── dist/                  # 构建输出
├── conf/
│   └── dashboard-settings.yaml
└── openspec/                  # 设计文档
```

---

## 用户角色与权限

| 角色 | 标识 | 可见菜单 | 权限范围 |
|------|------|---------|---------|
| 超级管理员 | `super_admin` | 管理中心 | 全局（用户、业务线、应用、权限） |
| 业务线管理员 | `line_admin` | 管理中心 | 自己管理的业务线和应用 |
| 普通成员 | `member` | 资源中心 + 服务模块 | 所属业务线的资源和模块 |

### 测试账号

| 邮箱 | 密码 | 角色 |
|------|------|------|
| admin@test.com | admin123 | super_admin |
| line_admin@test.com | admin123 | line_admin |
| member1@test.com | member123 | member |

---

## API 清单

### 认证 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/login | 登录（email + password） |
| POST | /api/auth/logout | 退出 |
| GET | /api/auth/feishu | 飞书 SSO 授权 |
| GET | /api/auth/feishu/callback | 飞书 SSO 回调 |
| GET | /api/auth/me | 当前用户信息 |

### 模块管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/groups | 模块列表 |
| POST | /api/groups | 创建模块 |
| GET | /api/groups/:id | 模块详情 |
| PUT | /api/groups/:id | 编辑模块 |
| DELETE | /api/groups/:id | 删除模块 |
| GET | /api/groups/:id/members | 成员（资源）列表 |
| POST | /api/groups/:id/members | 添加资源到模块 |
| DELETE | /api/groups/:id/members/:resource | 从模块移除资源 |

### 资源中心 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/resources | 资源列表 |
| GET | /api/resource/:id/rules | 资源规则详情 |
| GET | /api/resource/:id | 资源元数据 |
| PUT | /api/resource/:id | 变更模块 |
| DELETE | /api/resource/:id | 删除资源 |

### 规则管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/app/rule/flow/list | 流控规则列表 |
| POST | /api/app/rule/flow/update | 创建/更新流控规则 |
| POST | /api/app/rule/flow/del | 删除流控规则 |
| GET | /api/app/rule/circuitbreaker/list | 熔断规则列表 |
| POST | /api/app/rule/circuitbreaker/update | 创建/更新熔断规则 |
| POST | /api/app/rule/circuitbreaker/del | 删除熔断规则 |
| PUT | /api/resource/:id/flow/:rule_id/toggle | 流控规则开关 |
| PUT | /api/resource/:id/circuitbreaker/:rule_id/toggle | 熔断规则开关 |

### 发布 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/publish | 发布规则到 etcd |
| GET | /api/publish/records | 发布记录 |

### 版本管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/versions | 版本列表 |
| GET | /api/versions/:id | 版本详情 |
| POST | /api/versions/:id/rollback | 回滚到指定版本 |

### 应用管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/apps | 应用列表（按业务线分组） |
| POST | /api/apps | 创建应用 |
| PUT | /api/apps/:app_id | 更新应用 |
| DELETE | /api/apps/:app_id | 删除应用（软删除） |

### 权限管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/permissions | 授权用户权限 |
| GET | /api/permissions | 权限列表 |
| DELETE | /api/permissions/:id | 撤销权限 |

### 超级管理员 API（/api/admin/*）
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/admin/users | 用户列表 |
| GET | /api/admin/users/search | 搜索用户 |
| GET | /api/admin/lines | 业务线列表 |
| POST | /api/admin/lines | 创建业务线 |
| PUT | /api/admin/lines/:id | 更新业务线 |
| DELETE | /api/admin/lines/:id | 删除业务线 |
| POST | /api/admin/lines/:id/admins | 添加业务线管理员 |
| DELETE | /api/admin/lines/:id/admins/:user_id | 移除业务线管理员 |

### 业务线管理员 API（/api/line-admin/*）
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/line-admin/lines | 我的业务线列表 |
| PUT | /api/line-admin/lines/:id | 更新业务线描述 |
| GET | /api/line-admin/lines/:id/apps | 业务线下的应用列表 |
| POST | /api/line-admin/lines/:id/apps | 在业务线下创建应用 |
| PUT | /api/line-admin/lines/:id/apps/:app_id | 更新应用 |
| DELETE | /api/line-admin/lines/:id/apps/:app_id | 删除应用 |
| GET | /api/line-admin/lines/:id/members | 业务线成员列表 |
| POST | /api/line-admin/lines/:id/members | 添加业务线成员 |
| DELETE | /api/line-admin/lines/:id/members/:user_id | 移除业务线成员 |

---

## 数据库

### 表结构（13 张表）
| 表名 | 说明 |
|------|------|
| `apps` | etcd 数据源注册 |
| `groups` | 业务模块 |
| `resources` | 资源 |
| `flow_rules` | 流控规则 |
| `circuit_breaker_rules` | 熔断规则 |
| `publish_records` | 发布记录 |
| `publish_versions` | 版本快照 |
| `users` | 用户 |
| `business_lines` | 业务线 |
| `business_line_apps` | 业务线-应用关联 |
| `business_line_admins` | 业务线-管理员关联（多对多） |
| `business_line_members` | 业务线-成员关联（多对多） |
| `user_permissions` | 用户权限 |
| `user_audit_logs` | 审计日志 |
| `user_tokens` | 会话 Token |

### 初始化
```bash
mysql -u root < dashboard/sql/schema.sql
```

---

## 构建与运行

### 前置依赖
- Go 1.20+
- MySQL 9.2（127.0.0.1:3306, root/空密码）
- etcd v3（127.0.0.1:2379）
- Node.js 18+（前端构建）

### 后端
```bash
# 构建
go build -o bin/sentinel_dashboard dashboard/cmd/main.go

# 运行
./bin/sentinel_dashboard -c conf/dashboard-settings.yaml -p 6111
```

### 前端
```bash
cd frontend && npm install && npm run build
```

### 访问地址
- **Dashboard**: http://localhost:6111/web/
- **登录**: admin@test.com / admin123

---

## License

Internal use only.
