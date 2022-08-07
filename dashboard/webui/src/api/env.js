import request from '@/utils/request'

export function listApps() {
  return request({
    url: '/app/list',
    method: 'get',
    params: {}
  })
}

export function listFlowRules(appId) {
  return request({
    url: '/app/rule/flow/list',
    method: 'get',
    params: { app: appId }
  })
}

export function listCircuitbreakerRules(appId) {
  return request({
    url: '/app/rule/circuitbreaker/list',
    method: 'get',
    params: { app: appId }
  })
}

export function deleteFlowRule(data) {
  return request({
    url: '/app/rule/flow/del',
    method: 'post',
    data: JSON.stringify(data),
    headers: {
      'Content-Type': 'application/json'
    }
  })
}

export function saveOrUpdateFlowRule(data) {
  return request({
    url: '/app/rule/flow/update',
    method: 'post',
    data: JSON.stringify(data),
    headers: {
      'Content-Type': 'application/json'
    }
  })
}

export function deleteCircuitbreakerRule(data) {
  return request({
    url: '/app/rule/circuitbreaker/del',
    method: 'post',
    data: JSON.stringify(data),
    headers: {
      'Content-Type': 'application/json'
    }
  })
}

export function saveOrUpdateCircuitbreakerRule(data) {
  return request({
    url: '/app/rule/circuitbreaker/update',
    method: 'post',
    data: JSON.stringify(data),
    headers: {
      'Content-Type': 'application/json'
    }
  })
}

