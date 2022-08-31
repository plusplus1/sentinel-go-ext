## ADDED Requirements

### Requirement: Session 数据库存储

系统 SHALL 将 Session 数据持久化存储到数据库。

#### Scenario: 创建Session
- **WHEN** 用户登录成功
- **THEN** 系统在 user_tokens 表创建 Session 记录
- **AND** 记录包含 user_id、token、expires_at

#### Scenario: Session过期自动清理
- **WHEN** Session 超过过期时间
- **THEN** 系统自动标记为过期
- **AND** 定时任务每小时清理过期 Session

#### Scenario: 服务重启后Session保持
- **WHEN** 后端服务重启
- **THEN** 已登录用户的 Session 仍然有效
- **AND** 用户无需重新登录

### Requirement: Session 安全性

系统 SHALL 保障 Session 的安全性。

#### Scenario: Token随机性
- **WHEN** 系统生成 Session Token
- **THEN** Token 使用加密安全随机数生成
- **AND** Token 长度至少64字符

#### Scenario: 单设备登录
- **WHEN** 用户在新设备登录
- **THEN** 系统使旧设备的 Session 失效
- **AND** 旧设备收到"已在其他设备登录"提示

#### Scenario: Session有效期
- **WHEN** 用户登录
- **THEN** Session 默认有效期24小时
- **AND** 每次活跃操作延长有效期

### Requirement: Session 验证

系统 SHALL 在每次请求时验证 Session。

#### Scenario: 有效Session验证
- **WHEN** 请求携带有效 Session Token
- **THEN** 系统验证通过
- **AND** 返回用户信息

#### Scenario: 无效Session拒绝
- **WHEN** 请求携带无效或过期的 Session Token
- **THEN** 系统返回401错误
- **AND** 提示"会话已过期，请重新登录"

#### Scenario: Token查询优化
- **WHEN** 系统验证 Session
- **THEN** 使用索引快速查询
- **AND** 查询时间小于10ms

### Requirement: 登出功能

系统 SHALL 支持用户主动登出。

#### Scenario: 主动登出
- **WHEN** 用户点击"退出登录"
- **THEN** 系统删除当前 Session
- **AND** 清除客户端 Cookie

#### Scenario: 全设备登出
- **WHEN** 用户选择"退出所有设备"
- **THEN** 系统删除该用户的所有 Session
- **AND** 所有设备需要重新登录

### Requirement: Session API

后端 SHALL 提供 Session 管理 API。

#### Scenario: 验证Session接口
- **WHEN** 前端请求 GET /api/auth/me
- **AND** 携带有效 Session Token
- **THEN** 响应包含用户信息

#### Scenario: 登出接口
- **WHEN** 前端请求 POST /api/auth/logout
- **THEN** 系统删除当前 Session

#### Scenario: 全设备登出接口
- **WHEN** 前端请求 POST /api/auth/logout-all
- **THEN** 系统删除用户所有 Session
