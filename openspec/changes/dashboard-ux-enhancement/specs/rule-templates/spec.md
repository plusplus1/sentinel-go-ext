## ADDED Requirements

### Requirement: 系统预设模板

系统 SHALL 提供预设的规则模板供用户选择。

#### Scenario: 查看可用模板列表
- **WHEN** 用户创建新规则时
- **THEN** 系统显示可用模板列表
- **AND** 每个模板显示名称、描述和适用场景

#### Scenario: 选择模板创建规则
- **WHEN** 用户选择"标准API限流"模板
- **THEN** 系统自动填充模板配置
- **AND** 用户可以修改模板配置后再保存

#### Scenario: 系统模板不可删除
- **WHEN** 用户尝试删除系统预设模板
- **THEN** 系统拒绝操作并提示"系统模板不可删除"

### Requirement: 用户自定义模板

系统 SHALL 支持用户创建自定义规则模板。

#### Scenario: 从现有规则创建模板
- **WHEN** 用户编辑规则时点击"保存为模板"
- **THEN** 系统弹出模板命名对话框
- **AND** 保存后模板出现在用户模板列表中

#### Scenario: 创建新模板
- **WHEN** 用户在模板管理页面点击"新建模板"
- **THEN** 系统显示模板编辑表单
- **AND** 用户填写模板名称、描述和配置

#### Scenario: 编辑自定义模板
- **WHEN** 用户修改自己的自定义模板
- **THEN** 系统保存修改后的配置
- **AND** 不影响已使用该模板创建的规则

#### Scenario: 删除自定义模板
- **WHEN** 用户删除自定义模板
- **THEN** 模板从列表中移除
- **AND** 不影响已使用该模板创建的规则

### Requirement: 模板分类

系统 SHALL 按类型分类显示模板。

#### Scenario: 流控规则模板
- **WHEN** 用户创建流控规则
- **THEN** 系统只显示流控类型的模板

#### Scenario: 熔断规则模板
- **WHEN** 用户创建熔断规则
- **THEN** 系统只显示熔断类型的模板

### Requirement: 模板API

后端 SHALL 提供模板管理API。

#### Scenario: 获取模板列表
- **WHEN** 前端请求 GET /api/templates?type=flow
- **THEN** 响应包含所有流控类型的模板
- **AND** 包含系统模板和当前用户创建的模板

#### Scenario: 创建模板
- **WHEN** 前端请求 POST /api/templates
- **AND** 请求体包含模板配置
- **THEN** 系统创建新模板并返回模板ID

#### Scenario: 更新模板
- **WHEN** 前端请求 PUT /api/templates/{id}
- **THEN** 系统更新模板配置

#### Scenario: 删除模板
- **WHEN** 前端请求 DELETE /api/templates/{id}
- **AND** 模板不是系统模板
- **THEN** 系统删除模板
