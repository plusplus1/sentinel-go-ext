## ADDED Requirements

### Requirement: 四级权限角色

系统 SHALL 支持 owner/admin/editor/viewer 四级权限角色。

#### Scenario: owner权限
- **WHEN** 用户对资源拥有 owner 权限
- **THEN** 用户可以执行所有操作
- **AND** 包括查看、编辑、删除、发布、管理成员

#### Scenario: admin权限
- **WHEN** 用户对资源拥有 admin 权限
- **THEN** 用户可以查看、编辑、删除、发布
- **AND** 不能管理成员

#### Scenario: editor权限
- **WHEN** 用户对资源拥有 editor 权限
- **THEN** 用户可以查看、编辑规则
- **AND** 不能删除、发布、管理成员

#### Scenario: viewer权限
- **WHEN** 用户对资源拥有 viewer 权限
- **THEN** 用户只能查看
- **AND** 不能执行任何修改操作

### Requirement: 权限继承

系统 SHALL 支持权限从上级资源继承到下级资源。

#### Scenario: 业务线管理员继承应用权限
- **WHEN** 用户是业务线的管理员
- **THEN** 用户自动拥有该业务线下所有应用的 admin 权限

#### Scenario: 应用管理员继承资源权限
- **WHEN** 用户是应用的管理员
- **THEN** 用户自动拥有该应用下所有资源的 admin 权限

#### Scenario: 显式权限覆盖继承权限
- **WHEN** 用户对某资源有显式授权
- **THEN** 显式权限优先于继承权限

### Requirement: 权限检查

系统 SHALL 在每个操作前检查用户权限。

#### Scenario: 无权限操作被拒绝
- **WHEN** 用户尝试执行无权限的操作
- **THEN** 系统返回403错误
- **AND** 显示"权限不足：需要 X 权限"

#### Scenario: 超级管理员绕过权限检查
- **WHEN** 超级管理员执行任何操作
- **THEN** 系统允许操作，不检查具体权限

### Requirement: 权限管理界面

系统 SHALL 提供权限管理界面。

#### Scenario: 查看资源权限
- **WHEN** 资源 owner 点击"权限管理"
- **THEN** 系统显示当前资源的权限列表
- **AND** 显示每个用户的权限级别

#### Scenario: 授予权限
- **WHEN** 资源 owner 搜索并选择用户
- **AND** 选择权限级别后点击"授权"
- **THEN** 系统授予该用户相应权限

#### Scenario: 修改权限
- **WHEN** 资源 owner 修改某用户的权限级别
- **THEN** 系统更新用户权限

#### Scenario: 撤销权限
- **WHEN** 资源 owner 点击"撤销权限"
- **THEN** 系统移除该用户对此资源的权限

### Requirement: 权限API

后端 SHALL 提供权限管理API。

#### Scenario: 查询用户权限
- **WHEN** 前端请求 GET /api/permissions?resource_id=xxx
- **THEN** 响应包含该资源的权限列表

#### Scenario: 授予权限
- **WHEN** 前端请求 POST /api/permissions
- **AND** 请求体包含 user_id, resource_type, resource_id, role
- **THEN** 系统授予相应权限

#### Scenario: 检查权限
- **WHEN** 前端请求 GET /api/permissions/check?resource_id=xxx&action=publish
- **THEN** 响应包含用户是否有执行该操作的权限
