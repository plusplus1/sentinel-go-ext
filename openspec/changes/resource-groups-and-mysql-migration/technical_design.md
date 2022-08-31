# 技术设计文档：资源分组与资源中心视图（v2 - MySQL 架构）

## 1. 技术概述

### 1.1 技术选型
- **后端**: Go 1.20 + Gin + MySQL + etcd
- **前端**: React 18 + TypeScript + Ant Design + Vite
- **存储**: MySQL 5.7（主存储）+ etcd v3（发布目标）
- **MySQL 驱动**: go-sql-driver/mysql v1.8.1

### 1.2 核心设计决策

#### 决策1：MySQL 作为主存储
**原因**：
- 支持复杂查询（JOIN、聚合、分页）
- 事务支持，数据一致性更好
- 熟悉的 SQL 生态，易于维护和迁移
- 支持版本管理和审计

#### 决策2：etcd 仅作为发布目标
**原因**：
- Sentinel 客户端已适配 etcd 消费
- etcd watch 机制适合实时推送
- 保持客户端零改动
- 分离存储和运行时

#### 决策3：API 接口完全兼容
**原因**：
- 前端无需大幅改动
- 降低迁移成本
- 保持向后兼容

---

## 2. 数据库设计

### 2.1 设计原则
1. **命名规范**：表名小写下划线，字段名小写下划线
2. **主键策略**：自增 BIGINT 作为主键，业务键使用唯一索引
3. **软删除**：暂不实现，使用硬删除（后续可扩展 deleted_at 字段）
4. **时间戳**：created_at + updated_at 自动维护
5. **字符集**：utf8mb4，支持 emoji 和多语言

### 2.2 完整 Schema
见 `dashboard/sql/schema.sql`

### 2.3 索引策略
```sql
-- 唯一索引：防止重复数据
UNIQUE KEY uk_app_env_name (app_key, env, name)
UNIQUE KEY uk_rule_id (rule_id)

-- 普通索引：加速查询
KEY idx_app_env (app_key, env)
KEY idx_group_id (group_id)
KEY idx_app_env_resource (app_key, env, resource)
```

### 2.4 数据类型映射

| Go 类型 | MySQL 类型 | 说明 |
|---------|-----------|------|
| string | VARCHAR(n) | 变长字符串 |
| int/int64 | BIGINT | 整数 |
| float64 | DOUBLE | 浮点数 |
| bool | TINYINT(1) | 布尔值 |
| time.Time | TIMESTAMP | 时间戳 |
| *string | VARCHAR(n) NULL | 可空字符串 |
| map/slice | JSON | JSON 对象 |

---

## 3. 后端设计

### 3.1 DAO 层设计

#### MySQLGroupDAO
```go
type MySQLGroupDAO struct {
    db *sql.DB
}

// 方法
CreateGroup(group *model.Group) error
GetGroup(appKey, env, groupID string) (*model.Group, error)
GetGroupByName(appKey, env, name string) (*model.Group, error)
UpdateGroup(groupID string, group *model.Group) error
DeleteGroup(groupID string) error
ListGroups(appKey, env string) ([]*model.Group, error)
GetOrCreateDefaultGroup(appKey, env string) (*model.Group, error)
AddResourceToGroup(appKey, env, groupID, resourceName string) error
RemoveResourceFromGroup(appKey, env, groupID, resourceName string) error
```

#### MySQLResourceDAO
```go
type MySQLResourceDAO struct {
    db *sql.DB
}

GetOrCreateResource(appKey, env, resourceName string) (*model.Resource, error)
GetResource(appKey, env, resourceName string) (*model.Resource, error)
ListResources(appKey, env string) ([]*model.Resource, error)
UpdateResourceGroup(appKey, env, resourceName, groupID string) error
DeleteResource(appKey, env, resourceName string) error
```

#### MySQLFlowRuleDAO
```go
type MySQLFlowRuleDAO struct {
    db *sql.DB
}

CreateOrUpdateRule(rule *FlowRuleRecord) error
DeleteRule(ruleID string) error
GetRule(ruleID string) (*FlowRuleRecord, error)
ListRules(appKey, env string, resource ...string) ([]*FlowRuleRecord, error)
ToggleRule(ruleID string, enabled bool) error
```

#### MySQLCBRuleDAO
```go
type MySQLCBRuleDAO struct {
    db *sql.DB
}

CreateOrUpdateRule(rule *CBRuleRecord) error
DeleteRule(ruleID string) error
ListRules(appKey, env string, resource ...string) ([]*CBRuleRecord, error)
ToggleRule(ruleID string, enabled bool) error
```

### 3.2 模型定义

#### DB 记录模型
```go
type FlowRuleRecord struct {
    RuleID                 string         `json:"rule_id"`
    AppKey                 string         `json:"app_key"`
    Env                    string         `json:"env"`
    Resource               string         `json:"resource"`
    Threshold              float64        `json:"threshold"`
    MetricType             int            `json:"metric_type"`
    ControlBehavior        int            `json:"control_behavior"`
    WarmUpPeriodSec        int            `json:"warm_up_period_sec"`
    MaxQueueingTimeMs      int            `json:"max_queueing_time_ms"`
    ClusterMode            bool           `json:"cluster_mode"`
    ClusterConfig          sql.NullString `json:"cluster_config"`
    Enabled                bool           `json:"enabled"`
    CreatedAt              time.Time      `json:"created_at"`
    UpdatedAt              time.Time      `json:"updated_at"`
}

type CBRuleRecord struct {
    RuleID           string    `json:"rule_id"`
    AppKey           string    `json:"app_key"`
    Env              string    `json:"env"`
    Resource         string    `json:"resource"`
    Strategy         int       `json:"strategy"`
    Threshold        float64   `json:"threshold"`
    RetryTimeoutMs   int64     `json:"retry_timeout_ms"`
    MinRequestAmount int       `json:"min_request_amount"`
    Enabled          bool      `json:"enabled"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

### 3.3 发布服务设计

```go
type PublishService struct {
    db         *MySQLFlowRuleDAO
    cbDB       *MySQLCBRuleDAO
    recordDB   *MySQLPublishRecordDAO
    etcdClient *clientv3.Client
}

// 发布单资源流控规则
PublishFlowRules(appKey, env, resource string) error

// 发布单资源熔断规则
PublishCBRules(appKey, env, resource string) error

// 全量发布
PublishAll(appKey, env string) error
```

**发布逻辑**：
1. 从 MySQL 读取启用的规则
2. 按资源分组
3. 序列化为 JSON
4. 写入 etcd（key: `/sentinel-go/{app}/{env}/rules/{type}/{resource}`）
5. 记录发布历史

---

## 4. API 设计

### 4.1 设计原则
1. **RESTful 风格**：资源路径 + HTTP 方法
2. **统一响应格式**：`{code, message, data}`
3. **查询参数**：app + env 标识环境
4. **向后兼容**：保持与原 etcd API 完全一致

### 4.2 响应格式
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

| code | 含义 |
|------|------|
| 0 | 成功 |
| 100 | 参数错误 |
| 999 | 服务内部错误 |

### 4.3 完整 API 列表
见 tasks.md 的 API 清单章节

---

## 5. 前端设计

### 5.1 技术栈
- React 18 + TypeScript
- Vite 构建
- Ant Design 组件库
- React Router v6
- Axios HTTP 客户端

### 5.2 页面结构
```
src/
├── pages/
│   ├── Groups.tsx        # 模块管理（原分组管理）
│   ├── Resources.tsx     # 资源中心
│   ├── FlowRules.tsx     # 流控规则（保留）
│   └── CircuitBreakerRules.tsx  # 熔断规则（保留）
├── context/
│   └── AppContext.tsx    # 全局应用状态
└── App.tsx               # 主应用 + 路由配置
```

### 5.3 关键组件

#### Groups.tsx（模块管理）
- 模块列表表格（名称、描述、成员数、操作）
- 创建/编辑模块弹窗
- 成员管理抽屉（添加/移除资源）
- 删除确认（成员移至默认模块）

#### Resources.tsx（资源中心）
- 资源列表表格（名称、所属模块、流控规则数、熔断规则数、操作）
- 规则详情弹窗（分组展示流控/熔断规则）
- 规则编辑弹窗（流控表单 + 熔断表单）
- 规则创建（空状态添加按钮）
- 规则切换开关
- 变更模块弹窗
- 添加/删除资源

### 5.4 API 调用模式
```tsx
import axios from 'axios';
import { useApp } from '../context/AppContext';

const { selectedApp } = useApp();

// 列表查询
const response = await axios.get('/api/groups', {
  params: { app: selectedApp?.id, env: 'prod' }
});
const { code, data, message } = response.data;

// 创建
await axios.post('/api/groups', {
  app_id: selectedApp?.id,
  env: 'prod',
  name: '模块名称',
  description: '描述'
});

// 更新
await axios.put(`/api/groups/${groupId}`, {
  app_id: selectedApp?.id,
  env: 'prod',
  name: '新名称'
});

// 删除
await axios.delete(`/api/groups/${groupId}`, {
  params: { app: selectedApp?.id, env: 'prod' },
  data: { action: 'move_to_default' }
});
```

---

## 6. 数据迁移

### 6.1 etcd → MySQL 迁移脚本
```bash
# 导出 etcd 数据
etcdctl get --prefix "/sentinel/go/" --print-value-only > etcd_dump.json

# 转换并导入 MySQL（需要编写转换脚本）
python3 migrate_etcd_to_mysql.py etcd_dump.json
```

### 6.2 迁移步骤
1. 备份 etcd 数据
2. 运行 schema.sql 创建表结构
3. 运行迁移脚本导入数据
4. 验证数据完整性
5. 切换 dashboard 到新版本
6. 测试所有 API
7. 发布全量规则到 etcd

---

## 7. 性能优化

### 7.1 MySQL 优化
- 连接池配置（MaxOpenConns=25, MaxIdleConns=5）
- 索引覆盖常用查询
- 批量操作使用事务
- 大分页使用游标分页

### 7.2 缓存策略
- 规则列表可缓存 5 分钟
- 模块列表可缓存 10 分钟
- 发布后清除缓存

---

## 8. 监控与日志

### 8.1 关键指标
- MySQL 连接数
- API 响应时间
- 规则发布成功率
- etcd 同步延迟

### 8.2 日志记录
- 所有 API 请求（GIN 自动记录）
- 规则发布操作（publish_records 表）
- 数据库错误日志
- etcd 连接状态

---

## 更新日志
- 2026-03-14 08:56: 初版（基于 etcd 架构）
- 2026-03-14 12:25: v2 更新（MySQL 架构升级）

---

## 9. 发布版本管理技术设计

### 9.1 数据库 Schema
```sql
CREATE TABLE publish_versions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_key VARCHAR(64) NOT NULL,
    env VARCHAR(32) NOT NULL DEFAULT 'prod',
    version_number INT NOT NULL DEFAULT 1,
    description VARCHAR(512) DEFAULT '',
    operator VARCHAR(64) DEFAULT '',
    rule_count INT NOT NULL DEFAULT 0,
    snapshot JSON NOT NULL COMMENT '完整规则快照',
    status VARCHAR(16) NOT NULL DEFAULT 'success',
    error_msg TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_app_env_version (app_key, env, version_number),
    KEY idx_app_env (app_key, env)
);
```

### 9.2 快照 JSON 格式
```json
{
  "flow_rules": [
    {
      "rule_id": "flow-001",
      "resource": "API:GET:/users",
      "threshold": 100,
      "metric_type": 1,
      "control_behavior": 0,
      "enabled": true
    }
  ],
  "circuit_breaker_rules": [
    {
      "rule_id": "cb-001",
      "resource": "API:GET:/users",
      "strategy": 0,
      "threshold": 0.5,
      "retry_timeout_ms": 10000,
      "enabled": true
    }
  ]
}
```

### 9.3 DAO 设计
```go
type MySQLPublishVersionDAO struct {
    db *sql.DB
}

// 创建版本快照
CreateVersion(version *PublishVersion) error

// 获取版本列表
ListVersions(appKey, env string, limit int) ([]*PublishVersion, error)

// 获取版本详情
GetVersion(versionID int64) (*PublishVersion, error)

// 获取最新版本号
GetLatestVersionNumber(appKey, env string) (int, error)
```

### 9.4 回滚逻辑
```go
func (s *PublishService) RollbackToVersion(appKey, env string, versionID int64) error {
    // 1. 读取目标版本快照
    version := versionDAO.GetVersion(versionID)
    snapshot := json.Unmarshal(version.Snapshot)
    
    // 2. 清空当前规则
    flowRuleDAO.DeleteAll(appKey, env)
    cbRuleDAO.DeleteAll(appKey, env)
    
    // 3. 从快照恢复
    for _, rule := range snapshot.FlowRules {
        flowRuleDAO.CreateOrUpdateRule(rule)
    }
    for _, rule := range snapshot.CBRules {
        cbRuleDAO.CreateOrUpdateRule(rule)
    }
    
    // 4. 发布到 etcd
    s.PublishAll(appKey, env)
    
    // 5. 创建回滚版本记录
    versionDAO.CreateVersion(...)
}
```

### 9.5 API 实现
- `GET /api/versions` → MySQL 查询，返回版本列表
- `GET /api/versions/:id` → MySQL 查询，返回版本详情 + 快照
- `POST /api/versions/:id/rollback` → 执行回滚逻辑
- `POST /api/publish` → 增强：创建版本快照后再发布到 etcd

