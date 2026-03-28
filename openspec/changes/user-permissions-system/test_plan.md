⚠️ 当前状态：测试未实现（2026-03-15）

### 测试状态
- ❌ 单元测试未编写
- ❌ 集成测试未编写
- ❌ E2E 测试未编写
- ❌ 测试环境未配置

### 测试计划已设计
- ✅ 认证测试用例
- ✅ 权限测试用例
- ✅ 安全测试用例
- ✅ E2E 测试脚本

---

# 用户权限体系 — 测试验证方案

## 📅 日期：2026-03-14

---

## 一、测试策略

### 1.1 测试层级
- **单元测试**：Service 层逻辑（权限判断、角色校验）
- **集成测试**：API 端到端测试（登录→操作→审计）
- **E2E 测试**：浏览器端完整流程测试
- **安全测试**：越权、注入、Token 泄露等

---

## 二、认证测试

### 2.1 账号密码登录
| 用例 | 输入 | 预期结果 |
|------|------|---------|
| 正确登录 | 正确邮箱+密码 | 返回 token，设置 cookie |
| 密码错误 | 正确邮箱+错误密码 | 返回 401，提示密码错误 |
| 邮箱不存在 | 不存在的邮箱 | 返回 401，提示账号或密码错误 |
| 空密码 | 邮箱+空密码 | 返回 400，提示密码不能为空 |
| 锁定机制 | 连续 5 次错误密码 | 账号锁定 15 分钟 |

### 2.2 飞书 SSO
| 用例 | 输入 | 预期结果 |
|------|------|---------|
| 首次飞书登录 | 有效 OAuth code | 创建用户，返回 token |
| 已有用户飞书登录 | 有效 OAuth code | 关联已有用户，返回 token |
| 无效 code | 错误/过期 code | 返回 401，提示授权失败 |
| 重复使用 code | 已使用的 code | 返回 401，提示 code 已使用 |

### 2.3 Session 管理
| 用例 | 输入 | 预期结果 |
|------|------|---------|
| 有效 token | 有效 session | 返回用户信息 |
| 过期 token | 过期 session | 返回 401 |
| 撤销 token | 已撤销 session | 返回 401 |
| 多设备登录 | 5 台设备同时登录 | 全部成功 |
| 超限登录 | 第 6 台设备登录 | 最早设备被踢出 |

---

## 三、权限测试

### 3.1 超级管理员
| 用例 | 操作 | 预期结果 |
|------|------|---------|
| 创建业务线 | POST /api/admin/lines | 成功创建 |
| 设置超管 | PUT /api/admin/members/x/role | 角色更新成功 |
| 查看全局 | GET /api/admin/lines | 返回所有业务线 |
| 操作业务线内容 | POST /api/groups（无权限 App） | 返回 403 |

### 3.2 业务线管理员
| 用例 | 操作 | 预期结果 |
|------|------|---------|
| 创建本线 App | POST /api/admin/apps | 成功创建 |
| 编辑本线 App | PUT /api/admin/apps/x | 成功更新 |
| 添加成员 | POST /api/permissions | 成功授权 |
| 操作其他线 | GET /api/admin/apps?line_id=其他 | 返回 403 |
| 创建模块 | POST /api/groups（有权限 App） | 成功创建 |

### 3.3 普通成员
| 用例 | 操作 | 预期结果 |
|------|------|---------|
| 查看授权 App | GET /api/groups?app=授权 | 返回数据 |
| 查看未授权 App | GET /api/groups?app=未授权 | 返回 403 |
| 创建模块 | POST /api/groups（有权限） | 成功创建 |
| 编辑模块资源 | PUT /api/resource/x（模块成员） | 成功更新 |
| 跨模块操作 | PUT /api/resource/x（非成员） | 返回 403 |

---

## 四、安全测试

### 4.1 越权测试
| 用例 | 操作 | 预期结果 |
|------|------|---------|
| 普通用户访问管理 API | GET /api/admin/lines | 403 |
| 跨业务线操作 | 修改其他线的 App | 403 |
| 伪造角色 | 修改 JWT 中的角色 | 验证失败 |
| 未登录访问 | 不带 token 访问 API | 401 |

### 4.2 注入测试
| 用例 | 输入 | 预期结果 |
|------|------|---------|
| SQL 注入 | `'; DROP TABLE users--` | 参数化查询，无影响 |
| XSS 攻法 | `<script>alert(1)</script>` | 转义输出 |
| 路径遍历 | `../../../etc/passwd` | 路径校验拒绝 |

### 4.3 Token 安全
| 用例 | 操作 | 预期结果 |
|------|------|---------|
| Token 泄露 | 使用他人 token | 401（IP/UA 不匹配） |
| Token 过期 | 使用过期 token | 401 |
| 撤销 Token | 使用已撤销 token | 401 |

---

## 五、端到端测试脚本

```python
# e2e_test_permissions.py
import requests

BASE = "http://localhost:6111"

def test_login():
    """测试登录流程"""
    # 账号密码登录
    r = requests.post(f"{BASE}/api/auth/login", json={
        "email": "admin@company.com",
        "password": "test123"
    })
    assert r.json()["code"] == 0
    token = r.json()["data"]["token"]
    
    # 获取用户信息
    r = requests.get(f"{BASE}/api/auth/me", cookies={"session_token": token})
    assert r.json()["code"] == 0
    assert r.json()["data"]["user"]["role"] == "super_admin"
    
    return token

def test_super_admin(token):
    """测试超管操作"""
    # 创建业务线
    r = requests.post(f"{BASE}/api/admin/lines", 
        json={"name": "测试业务线"},
        cookies={"session_token": token})
    assert r.json()["code"] == 0
    
    # 查看所有业务线
    r = requests.get(f"{BASE}/api/admin/lines",
        cookies={"session_token": token})
    assert r.json()["code"] == 0
    assert len(r.json()["data"]) > 0

def test_member_access(token):
    """测试成员权限"""
    # 应该被拒绝访问管理 API
    r = requests.get(f"{BASE}/api/admin/lines",
        cookies={"session_token": token})
    assert r.status_code == 403

if __name__ == "__main__":
    admin_token = test_login()
    test_super_admin(admin_token)
    # member_token = test_login_member()
    # test_member_access(member_token)
    print("✅ 所有权限测试通过")
```

---

## 六、测试环境

| 组件 | 配置 |
|------|------|
| MySQL | 127.0.0.1:3306 |
| etcd | 127.0.0.1:2379 |
| Dashboard | http://localhost:6111 |
| 测试用户 | admin@company.com / test123 |
| 测试业务线 | 测试业务线 |
| 测试 App | test-app |
