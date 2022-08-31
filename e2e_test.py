#!/usr/bin/env python3
"""端到端测试脚本"""
import requests
import json
import time

BASE = "http://localhost:6111"
APP_ID = "d2a9d2a4c717bd13f290045816ef66d9"
ENV = "prod"

pass_count = 0
fail_count = 0

def test(name, resp):
    global pass_count, fail_count
    try:
        data = resp.json()
        code = data.get("code", -1)
        if code == 0:
            print(f"  ✅ {name}")
            pass_count += 1
            return data
        else:
            msg = data.get("message", "")[:50]
            print(f"  ❌ {name} (code:{code} {msg})")
            fail_count += 1
            return data
    except:
        print(f"  ❌ {name} (invalid response)")
        fail_count += 1
        return {}

def api(method, path, **kwargs):
    url = f"{BASE}{path}"
    if method == "GET":
        return requests.get(url, **kwargs)
    elif method == "POST":
        return requests.post(url, **kwargs)
    elif method == "PUT":
        return requests.put(url, **kwargs)
    elif method == "DELETE":
        return requests.delete(url, **kwargs)

time.sleep(2)

print("🧪 端到端完整测试")
print("=" * 50)

# 1. 模块管理
print("\n【模块管理】")
r = test("创建模块", api("POST", "/api/groups", json={"app_id": APP_ID, "env": ENV, "name": "E2E测试", "description": "test"}))
gid = r.get("data", {}).get("id", "")
print(f"    模块ID: {gid}")

test("编辑模块", api("PUT", f"/api/groups/{gid}", json={"app_id": APP_ID, "env": ENV, "name": "E2E测试(改)", "description": "mod"}))
test("模块列表", api("GET", "/api/groups", params={"app": APP_ID, "env": ENV}))
test("模块详情", api("GET", f"/api/groups/{gid}", params={"app": APP_ID, "env": ENV}))

# 2. 资源管理
print("\n【资源管理】")
test("资源列表(含计数)", api("GET", "/api/resources", params={"app": APP_ID, "env": ENV}))
test("添加资源", api("PUT", "/api/resource/e2e-api", json={"group_id": "default"}, params={"app": APP_ID, "env": ENV}))
test("变更模块", api("PUT", "/api/resource/e2e-api", json={"group_id": gid}, params={"app": APP_ID, "env": ENV}))
test("资源规则", api("GET", "/api/resource/e2e-api/rules", params={"app": APP_ID, "env": ENV}))

# 3. 成员管理
print("\n【成员管理】")
test("添加成员", api("POST", f"/api/groups/{gid}/members", json={"resource": "e2e-api"}, params={"app": APP_ID}))
test("成员列表", api("GET", f"/api/groups/{gid}/members", params={"app": APP_ID, "env": ENV}))

# 4. 流控规则
print("\n【流控规则】")
test("创建流控", api("POST", "/api/app/rule/flow/update", json={
    "appId": APP_ID, "id": "e2e-flow", "resource": "e2e-api",
    "threshold": 100, "controlBehavior": 0, "metricType": 1
}))
test("流控列表", api("GET", "/api/app/rule/flow/list", params={"app": APP_ID}))
test("流控切换", api("PUT", "/api/resource/e2e-api/flow/e2e-flow/toggle", params={"app": APP_ID, "env": ENV}))
test("流控删除", api("POST", "/api/app/rule/flow/del", json={"id": "e2e-flow"}))

# 5. 熔断规则
print("\n【熔断规则】")
test("创建熔断", api("POST", "/api/app/rule/circuitbreaker/update", json={
    "appId": APP_ID, "id": "e2e-cb", "resource": "e2e-api",
    "strategy": 0, "threshold": 0.5, "retryTimeoutMs": 10000, "minRequestAmount": 5
}))
test("熔断列表", api("GET", "/api/app/rule/circuitbreaker/list", params={"app": APP_ID}))
test("熔断删除", api("POST", "/api/app/rule/circuitbreaker/del", json={"id": "e2e-cb"}))

# 6. 发布
print("\n【发布功能】")
test("发布", api("POST", "/api/publish", json={"app_key": APP_ID, "env": ENV, "rule_type": "all"}))
test("发布记录", api("GET", "/api/publish/records", params={"app": APP_ID, "env": ENV}))

# 7. 清理
print("\n【清理】")
test("删除资源", api("DELETE", "/api/resource/e2e-api", params={"app": APP_ID, "env": ENV}))
test("删除模块", api("DELETE", f"/api/groups/{gid}", params={"app": APP_ID, "env": ENV}))

# 8. 前端
print("\n【前端页面】")
for page in [("首页", "/web/"), ("资源中心", "/web/resources"), ("模块管理", "/web/modules")]:
    r = requests.get(f"{BASE}{page[1]}")
    status = "✅" if r.status_code == 200 else "❌"
    print(f"  {status} {page[0]} (HTTP {r.status_code})")

# Summary
print("\n" + "=" * 50)
total = pass_count + fail_count
rate = pass_count * 100 // total if total > 0 else 0
print(f"📊 通过: {pass_count} 失败: {fail_count} 总计: {total} 通过率: {rate}%")
print("=" * 50)

# Version management tests
print("\n【版本管理】")

# Create rules and publish v1
api("POST", "/api/app/rule/flow/update", json={
    "appId": APP_ID, "id": "vtest-1", "resource": "vtest-res",
    "threshold": 100, "controlBehavior": 0
})
test("发布v1", api("POST", "/api/publish", json={"app_key": APP_ID, "env": ENV, "rule_type": "all"}))

# Create another rule and publish v2
api("POST", "/api/app/rule/flow/update", json={
    "appId": APP_ID, "id": "vtest-2", "resource": "vtest-res",
    "threshold": 200, "controlBehavior": 0
})
test("发布v2", api("POST", "/api/publish", json={"app_key": APP_ID, "env": ENV, "rule_type": "all"}))

# List versions
r = test("版本列表", api("GET", "/api/versions", params={"app": APP_ID, "env": ENV}))
if r.get("data"):
    print(f"    版本数: {len(r['data'])}")
    for v in r['data'][:3]:
        print(f"    - v{v['version_number']}: {v['description']}")

# Get version detail
if r.get("data"):
    vid = r['data'][0]['id']
    r2 = test("版本详情", api("GET", f"/api/versions/{vid}"))
    if r2.get("data"):
        snap = r2['data'].get('snapshot', {})
        print(f"    流控规则: {len(snap.get('flow_rules',[]))}")
        print(f"    熔断规则: {len(snap.get('circuit_breaker_rules',[]))}")

# Rollback
if r.get("data") and len(r['data']) > 1:
    vid = r['data'][1]['id']
    test("回滚", api("POST", f"/api/versions/{vid}/rollback", json={"app_key": APP_ID, "env": ENV}))

# Cleanup
api("POST", "/api/app/rule/flow/del", json={"id": "vtest-1"})
api("POST", "/api/app/rule/flow/del", json={"id": "vtest-2"})
