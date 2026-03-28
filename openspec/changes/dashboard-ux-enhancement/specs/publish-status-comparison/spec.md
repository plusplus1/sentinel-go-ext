## ADDED Requirements

### Requirement: 资源列表显示发布状态

系统 SHALL 在资源列表中显示每个资源的发布状态，包括是否有未发布的变更。

#### Scenario: 资源有未发布的规则变更
- **WHEN** 资源的流控规则或熔断规则与已发布版本不一致
- **THEN** 系统显示"有变更未发布"状态标签
- **AND** 显示变更数量（如"2项变更"）

#### Scenario: 资源配置与发布版本一致
- **WHEN** 资源的规则配置与最新发布版本完全一致
- **THEN** 系统显示当前版本号（如"v5 已发布"）
- **AND** 显示发布时间

#### Scenario: 资源从未发布
- **WHEN** 资源没有任何发布记录
- **THEN** 系统显示"未发布"状态

### Requirement: 发布预览显示差异对比

系统 SHALL 在发布预览弹窗中高亮显示当前配置与已发布版本的差异。

#### Scenario: 流控规则阈值变更
- **WHEN** 用户点击发布按钮
- **AND** 流控规则阈值从100变更为200
- **THEN** 发布预览显示"阈值: 100 → 200"
- **AND** 变更部分使用高亮样式

#### Scenario: 新增熔断规则
- **WHEN** 用户点击发布按钮
- **AND** 该资源新增了熔断规则
- **THEN** 发布预览显示"(新增) 熔断规则"
- **AND** 显示新规则的摘要信息

#### Scenario: 删除规则
- **WHEN** 用户点击发布按钮
- **AND** 已发布版本有规则但当前配置已删除
- **THEN** 发布预览显示"(删除) 规则"
- **AND** 显示被删除规则的摘要

### Requirement: 发布状态API增强

后端 API SHALL 在资源列表接口中返回发布状态信息。

#### Scenario: 请求资源列表带发布状态
- **WHEN** 前端请求 GET /api/resources?include_publish_status=true
- **THEN** 响应包含每个资源的 has_unpublished_changes 字段
- **AND** 响应包含 unpublished_count 字段
- **AND** 响应包含当前规则和已发布规则的对比数据
