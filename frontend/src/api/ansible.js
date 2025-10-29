import request from '@/utils/request'

/**
 * 获取 Playbook 列表
 */
export function listPlaybooks(params) {
  return request({
    url: '/automation/ansible/playbooks',
    method: 'get',
    params
  })
}

/**
 * 获取 Playbook 详情
 */
export function getPlaybook(id) {
  return request({
    url: `/automation/ansible/playbooks/${id}`,
    method: 'get'
  })
}

/**
 * 创建 Playbook
 */
export function createPlaybook(data) {
  return request({
    url: '/automation/ansible/playbooks',
    method: 'post',
    data
  })
}

/**
 * 更新 Playbook
 */
export function updatePlaybook(id, data) {
  return request({
    url: `/automation/ansible/playbooks/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除 Playbook
 */
export function deletePlaybook(id) {
  return request({
    url: `/automation/ansible/playbooks/${id}`,
    method: 'delete'
  })
}

/**
 * 执行 Playbook
 */
export function runPlaybook(data) {
  return request({
    url: '/automation/ansible/run',
    method: 'post',
    data
  })
}

/**
 * 获取执行状态
 */
export function getExecutionStatus(taskId) {
  return request({
    url: `/automation/ansible/status/${taskId}`,
    method: 'get'
  })
}

/**
 * 取消执行
 */
export function cancelExecution(taskId) {
  return request({
    url: `/automation/ansible/cancel/${taskId}`,
    method: 'post'
  })
}

/**
 * 获取执行历史
 */
export function listExecutions(params) {
  return request({
    url: '/automation/ansible/history',
    method: 'get',
    params
  })
}

