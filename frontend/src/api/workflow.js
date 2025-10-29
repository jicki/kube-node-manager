import request from '@/utils/request'

/**
 * 获取工作流列表
 */
export function listWorkflows(params) {
  return request({
    url: '/automation/workflows',
    method: 'get',
    params
  })
}

/**
 * 获取工作流详情
 */
export function getWorkflow(id) {
  return request({
    url: `/automation/workflows/${id}`,
    method: 'get'
  })
}

/**
 * 创建工作流
 */
export function createWorkflow(data) {
  return request({
    url: '/automation/workflows',
    method: 'post',
    data
  })
}

/**
 * 更新工作流
 */
export function updateWorkflow(id, data) {
  return request({
    url: `/automation/workflows/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除工作流
 */
export function deleteWorkflow(id) {
  return request({
    url: `/automation/workflows/${id}`,
    method: 'delete'
  })
}

/**
 * 执行工作流
 */
export function executeWorkflow(data) {
  return request({
    url: '/automation/workflows/execute',
    method: 'post',
    data
  })
}

/**
 * 获取执行状态
 */
export function getExecutionStatus(taskId) {
  return request({
    url: `/automation/workflows/status/${taskId}`,
    method: 'get'
  })
}

/**
 * 获取执行历史
 */
export function listExecutions(params) {
  return request({
    url: '/automation/workflows/history',
    method: 'get',
    params
  })
}

