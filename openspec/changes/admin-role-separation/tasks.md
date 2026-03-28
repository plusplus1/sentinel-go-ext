# 管理员角色权限隔离 — 任务清单

## 已完成

### 后端
- [x] 拆分 `auth.go` 为三个文件（auth.go、auth_admin.go、auth_line.go）
- [x] 创建 `auth_admin.go`（超级管理员 API）
- [x] 创建 `auth_line.go`（业务线管理员 API）
- [x] 添加 `ListMyBusinessLines` 函数
- [x] 添加 `UpdateMyBusinessLine` 函数
- [x] 添加 `CreateBusinessLineApp` 函数
- [x] 添加 `UpdateBusinessLineApp` 函数
- [x] 添加 `DeleteBusinessLineApp` 函数
- [x] 修改 `install.go` 添加 `lineAdminGroup` 路由组
- [x] 删除 `UpdateUserRole` 函数（前端未使用）
- [x] 删除 `PUT /api/admin/users/:user_id/role` 路由

### 前端
- [x] 重构 `Admin.tsx`，添加角色判断
- [x] 创建 `SuperAdminPanel` 组件
- [x] 创建 `LineAdminPanel` 组件
- [x] 实现超级管理员界面（业务线 CRUD、绑定管理员）
- [x] 实现业务线管理员界面（查看业务线、管理应用）

### 文档
- [x] 创建 `openspec/changes/admin-role-separation/` 目录
- [x] 创建 `design.md` 设计文档
- [x] 创建 `tasks.md` 任务清单

## 待完成

### 后端
- [ ] Dashboard 启动问题排查（端口监听问题）
- [ ] 端到端测试所有 API

### 前端
- [ ] LineAdminPanel 添加业务线内应用管理 Modal
- [ ] 添加创建应用功能
- [ ] 添加修改应用功能
- [ ] 添加删除应用功能

### 文档
- [ ] 更新 `openspec/project.md` API 清单
- [ ] 更新 `openspec/project.md` 角色权限表格
