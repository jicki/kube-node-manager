import request from '@/utils/request'

/**
 * 获取脚本列表
 */
export function listScripts(params) {
  return request({
    url: '/automation/scripts',
    method: 'get',
    params
  })
}

/**
 * 获取脚本详情
 */
export function getScript(id) {
  return request({
    url: `/automation/scripts/${id}`,
    method: 'get'
  })
}

/**
 * 创建脚本
 */
export function createScript(data) {
  return request({
    url: '/automation/scripts',
    method: 'post',
    data
  })
}

/**
 * 更新脚本
 */
export function updateScript(id, data) {
  return request({
    url: `/automation/scripts/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除脚本
 */
export function deleteScript(id) {
  return request({
    url: `/automation/scripts/${id}`,
    method: 'delete'
  })
}

/**
 * 执行脚本
 */
export function executeScript(data) {
  return request({
    url: '/automation/scripts/execute',
    method: 'post',
    data
  })
}

/**
 * 获取执行状态
 */
export function getExecutionStatus(taskId) {
  return request({
    url: `/automation/scripts/status/${taskId}`,
    method: 'get'
  })
}

/**
 * 获取执行历史
 */
export function listExecutions(params) {
  return request({
    url: '/automation/scripts/history',
    method: 'get',
    params
  })
}

