import request from '@/utils/request'

/**
 * 执行 SSH 命令
 */
export function executeCommand(data) {
  return request({
    url: '/automation/ssh/execute',
    method: 'post',
    data
  })
}

/**
 * 获取执行状态
 */
export function getExecutionStatus(taskId) {
  return request({
    url: `/automation/ssh/status/${taskId}`,
    method: 'get'
  })
}

/**
 * 获取执行历史
 */
export function listExecutions(params) {
  return request({
    url: '/automation/ssh/history',
    method: 'get',
    params
  })
}

