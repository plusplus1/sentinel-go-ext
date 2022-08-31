# Sentinel Dashboard 功能对比分析

> 本项目（Go 实现） vs 官方 Sentinel Dashboard（Java 实现）

## 项目差异概述

| 维度 | 本项目 (Go) | 官方 Dashboard (Java) |
|------|------------|---------------------|
| **语言** | Go 1.20 + Gin | Java 8 + Spring Boot |
| **前端** | React 18 + Ant Design 5.x | AngularJS 1.x |
| **数据存储** | MySQL 持久化 | 仅内存（重启丢失） |
| **多租户** | 业务线 + 应用两级 | 仅应用级别 |
| **权限** | RBAC 三级权限 | 简单用户名/密码 |
| **SSO** | 支持飞书 SSO | 无 |
| **版本管理** | 版本快照 + 回滚 | 无 |

---

## 功能对比矩阵

### ✅ 本项目独有（官方没有）

| 功能 | 说明 |
|------|------|
| **MySQL 持久化** | 规则存储在 MySQL，重启不丢失 |
| **业务线管理** | 多租户层级：业务线 → 应用 → 模块 → 资源 |
| **用户权限系统** | super_admin / line_admin / member 三级权限 |
| **飞书 SSO** | 企业级单点登录 |
| **版本快照与回滚** | 每次发布创建版本快照，支持回滚 |
| **发布记录** | 记录每次发布的操作人、时间、规则数 |
| **规则启停** | 可临时禁用规则而不删除 |
| **资源分组** | 按模块组织资源 |

### ✅ 官方独有（本项目没有）

| 功能 | 说明 | 优先级 |
|------|------|--------|
| **实时监控** | 按资源展示 QPS 曲线（pass/block/success/exception/RT） | 🔴 高 |
| **系统规则** | 系统负载、CPU、RT、线程数、QPS 阈值保护 | 🔴 高 |
| **热点参数限流** | 按参数值进行差异化限流 | 🟡 中 |
| **授权规则** | 资源级别的黑白名单控制 | 🟡 中 |
| **集群流控** | Token Server 模式的集群限流 | 🟢 低 |
| **网关规则** | API Gateway 专用的路由级限流 | 🟢 低 |
| **机器自发现** | 心跳注册机制（`/registry/machine`） | 🟢 低 |
| **簇点链路** | 资源树形结构展示 | 🟢 低 |

### ✅ 双方都有（实现方式不同）

| 功能 | 本项目 | 官方 |
|------|--------|------|
| **流控规则** | MySQL CRUD + etcd 发布 | 内存存储 + HTTP 直推 / 配置中心 |
| **熔断规则** | MySQL CRUD + etcd 发布 | 内存存储 + HTTP 直推 |
| **登录认证** | MySQL 用户表 + bcrypt | JVM 参数指定用户名/密码 |

---

## 官方 Dashboard 的关键设计模式

### 1. 双版本发布策略（V1 vs V2）

官方提供了两种发布模式：

**V1（FlowControllerV1）— 直推客户端**：
- 规则存储在内存中
- 发布时直接 HTTP 推送到 Sentinel 客户端
- 客户端需要暴露 HTTP 端口供 Dashboard 调用

**V2（FlowControllerV2）— 配置中心模式**：
- 通过 `DynamicRuleProvider` 接口从配置中心读取规则
- 通过 `DynamicRulePublisher` 接口将规则推送到配置中心
- 支持 Nacos、Apollo、ZooKeeper 等多种配置中心

**本项目设计**：使用 etcd 作为配置中心，客户端轮询 etcd 获取规则。与 V2 模式类似。

### 2. 规则实体模型

官方的 `RuleEntity` 接口定义了统一的规则实体模式：

```java
public interface RuleEntity extends Serializable {
    Long getId();
    String getApp();
    String getIp();
    Integer getPort();
    String getResource();
    Rule toRule();  // 转换为领域规则对象
}
```

每个规则实体都包含 `app`、`ip`、`port` 字段，用于标识规则归属的机器。

**本项目设计**：使用 `app_id`（数字）+ `resource_id`（外键）标识规则归属，更适合多租户场景。

### 3. 机器自动发现

官方通过 `/registry/machine` 端点接收客户端心跳，自动发现机器：

```java
// MachineRegistryController.java
@GetMapping("/registry/machine")
public Result<?> receiveHeartbeat(String app, String appType, Long hostname, 
                                   Long port, Long pid, String version) {
    // 注册机器到 AppManagement
}
```

**本项目设计**：使用 etcd 作为服务注册中心，客户端通过 etcd 注册。无需内置心跳机制。

---

## 建议借鉴的功能

### P0 — 建议尽快实现

#### 1. 实时监控（实时监控页面）

**官方实现**：`MetricController.java` + `MetricEntity`
- 客户端上报指标到 Dashboard
- Dashboard 存储 5 分钟的滚动窗口数据
- 前端展示 pass/block/success/exception/RT 的折线图

**本项目实现建议**：
- 后端：新增 `/api/metrics` 端点，从 etcd 读取客户端上报的指标
- 前端：新增「监控」页面，使用 Ant Design Charts 展示折线图
- 存储：MySQL 存储指标（按分钟聚合），或使用内存 + 定时清理

#### 2. 系统规则（系统自适应保护）

**官方数据模型**：
```java
public class SystemRuleEntity {
    Double highestSystemLoad;  // 最高系统负载
    Double highestCpuUsage;    // 最高 CPU 使用率 [0.0-1.0]
    Long avgRt;                // 最大平均响应时间 (ms)
    Long maxThread;            // 最大并发线程数
    Double qps;                // 最大 QPS
}
```

**本项目实现建议**：
- 后端：新增 `system_rules` 表 + CRUD API
- 前端：新增「系统规则」Tab 或页面
- etcd 发布：`/sentinel/{line}/{app}/{group}/{resource}/system`

### P1 — 建议中期实现

#### 3. 热点参数限流（ParamFlowRule）

**官方数据模型**：
```java
public class ParamFlowRuleEntity {
    String resource;
    Integer paramIdx;           // 参数索引
    Integer grade;              // QPS
    Double count;               // 阈值
    Long durationInSec;         // 统计窗口
    Integer controlBehavior;    // 拒绝/匀速
    List<ParamFlowItem> paramFlowItemList;  // 按值差异化配置
}
```

**适用场景**：秒杀场景中，对热门商品 ID 进行更严格的限流。

#### 4. 授权规则（AuthorityRule）

**官方数据模型**：
```java
public class AuthorityRuleEntity {
    String resource;
    String limitApp;       // 逗号分隔的调用方列表
    Integer strategy;      // AUTHORITY_WHITE(0) 或 AUTHORITY_BLACK(1)
}
```

**适用场景**：资源级别的黑白名单控制，例如禁止某个服务调用敏感接口。

### P2 — 暂不建议实现

#### 5. 集群流控

**复杂度**：高。需要 Token Server 管理、服务器分配、客户端模式切换等。
**适用场景**：单机限流无法满足需求的集群部署。
**建议**：暂不实现。等有实际集群限流需求时再考虑。

#### 6. 网关规则

**复杂度**：中。仅适用于 API Gateway 场景。
**适用场景**：使用 Spring Cloud Gateway 等网关的项目。
**建议**：暂不实现。当前聚焦于普通微服务场景。

---

## 架构设计借鉴

### 1. DynamicRuleProvider / Publisher 接口

官方的可插拔配置中心设计值得借鉴：

```go
// 推荐接口设计
type RuleProvider interface {
    GetFlowRules(app string) ([]FlowRule, error)
    GetCBRules(app string) ([]CBRule, error)
    GetSystemRules(app string) ([]SystemRule, error)
}

type RulePublisher interface {
    PublishFlowRules(app string, rules []FlowRule) error
    PublishCBRules(app string, rules []CBRule) error
    PublishSystemRules(app string, rules []SystemRule) error
}

// 实现
type EtcdRuleProvider struct { client *clientv3.Client }
type NacosRuleProvider struct { client *nacos.Client }
```

**优势**：未来支持多种配置中心（etcd、Nacos、Consul）时，只需新增实现即可。

### 2. 资源树形结构

官方的 `ResourceTreeNode` 将资源组织为树形结构：

```
├── API:GET:/users
│   ├── API:GET:/users/{id}
│   └── API:GET:/users/search
├── API:POST:/users
└── SERVICE:UserService
    ├── SERVICE:UserService.create
    └── SERVICE:UserService.delete
```

**本项目**：当前是扁平列表，可考虑在资源中心页面增加树形视图。

### 3. 统一的 Result 响应封装

```go
type Result struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func Success(data interface{}) Result { return Result{Code: 0, Data: data} }
func Fail(code int, msg string) Result { return Result{Code: code, Message: msg} }
```

**本项目**：当前使用 `appResp` 结构，与官方的 `Result` 类似。

---

## 总结

### 本项目优势

| 维度 | 优势 |
|------|------|
| **持久化** | MySQL 存储，重启不丢失 |
| **多租户** | 业务线层级，适合企业级使用 |
| **权限** | RBAC 三级权限，安全可控 |
| **版本管理** | 版本快照 + 回滚，操作可追溯 |
| **SSO** | 飞书单点登录，企业级认证 |

### 需要补足的功能

| 功能 | 优先级 | 工作量 | 说明 |
|------|--------|--------|------|
| 授权规则 | 🟡 中 | 1 周 | 黑白名单控制，简单实用 |
| 集群流控 | 🟢 低 | 3-4 周 | 复杂度高，需求不明确 |
| 网关规则 | 🟢 低 | 2 周 | 仅适用于 API Gateway 场景 |
| 实时监控 | ⚪ 极长期 | 1-2 周 | 需要客户端上报指标，当前无需求 |
| 系统规则 | ⚪ 极长期 | 1 周 | 需要客户端采集系统指标，当前无需求 |
| 热点参数限流 | ⚪ 极长期 | 2 周 | 秒杀场景专用，当前无需求 |

### 建议的优先级

1. **近期**：聚焦权限系统（RBAC 细化）、Session 持久化、审计日志等核心功能
2. **中期**：可考虑授权规则（黑白名单，简单实用）
3. **长期**：实时监控、系统规则、热点参数限流（需要客户端 SDK 配合，当前无需求）
