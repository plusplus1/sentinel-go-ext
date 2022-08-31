## ADDED Requirements

### Requirement: 批量选择资源

系统 SHALL 支持用户在资源列表中选择多个资源。

#### Scenario: 勾选多个资源
- **WHEN** 用户勾选资源列表中的多个资源
- **THEN** 系统高亮显示已选中的资源
- **AND** 顶部显示"已选择 X 个资源"

#### Scenario: 全选当前页
- **WHEN** 用户点击表头的全选复选框
- **THEN** 系统选中当前页所有资源

#### Scenario: 取消选择
- **WHEN** 用户点击"取消选择"按钮
- **THEN** 系统清除所有选择状态

### Requirement: 批量发布

系统 SHALL 支持批量发布多个资源的规则。

#### Scenario: 批量发布确认
- **WHEN** 用户选择多个资源后点击"批量发布"
- **THEN** 系统显示发布预览弹窗
- **AND** 显示所有选中资源的规则摘要

#### Scenario: 批量发布执行
- **WHEN** 用户确认批量发布
- **THEN** 系统创建批量发布任务
- **AND** 显示任务进度

#### Scenario: 批量发布进度显示
- **WHEN** 批量发布任务执行中
- **THEN** 系统显示进度条（如"3/10 已完成"）
- **AND** 实时更新完成数量

#### Scenario: 批量发布部分失败
- **WHEN** 批量发布任务部分资源发布失败
- **THEN** 系统显示失败列表和失败原因
- **AND** 提供"重试失败项"按钮

### Requirement: 批量启用/禁用规则

系统 SHALL 支持批量启用或禁用规则。

#### Scenario: 批量启用规则
- **WHEN** 用户选择多个资源后点击"批量启用"
- **THEN** 系统启用所有选中资源的规则

#### Scenario: 批量禁用规则
- **WHEN** 用户选择多个资源后点击"批量禁用"
- **THEN** 系统禁用所有选中资源的规则

### Requirement: 批量操作权限控制

系统 SHALL 根据用户权限控制批量操作。

#### Scenario: 无权限资源提示
- **WHEN** 用户选择的资源中包含无发布权限的资源
- **THEN** 系统提示"X 个资源无发布权限，已自动排除"
- **AND** 只对有权限的资源执行操作

### Requirement: 批量任务管理API

后端 SHALL 提供批量任务管理API。

#### Scenario: 创建批量任务
- **WHEN** 前端请求 POST /api/batch/publish
- **AND** 请求体包含 resource_ids 数组
- **THEN** 系统创建批量任务并返回任务ID

#### Scenario: 查询任务进度
- **WHEN** 前端请求 GET /api/batch/tasks/{task_id}
- **THEN** 响应包含任务状态、进度、失败列表

#### Scenario: 重试失败项
- **WHEN** 前端请求 POST /api/batch/tasks/{task_id}/retry
- **THEN** 系统重新执行失败的任务项
