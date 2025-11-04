import request from '@/utils/request'

// 创建工作流
export function createWorkflow(data) {
  return request({
    url: '/api/v1/ansible/workflows',
    method: 'post',
    data
  })
}

// 获取工作流列表
export function listWorkflows(params) {
  return request({
    url: '/api/v1/ansible/workflows',
    method: 'get',
    params
  })
}

// 获取工作流详情
export function getWorkflow(id) {
  return request({
    url: `/api/v1/ansible/workflows/${id}`,
    method: 'get'
  })
}

// 更新工作流
export function updateWorkflow(id, data) {
  return request({
    url: `/api/v1/ansible/workflows/${id}`,
    method: 'put',
    data
  })
}

// 删除工作流
export function deleteWorkflow(id) {
  return request({
    url: `/api/v1/ansible/workflows/${id}`,
    method: 'delete'
  })
}

// 执行工作流
export function executeWorkflow(id) {
  return request({
    url: `/api/v1/ansible/workflows/${id}/execute`,
    method: 'post'
  })
}

// 获取工作流执行列表
export function listWorkflowExecutions(params) {
  return request({
    url: '/api/v1/ansible/workflow-executions',
    method: 'get',
    params
  })
}

// 获取工作流执行详情
export function getWorkflowExecution(id) {
  return request({
    url: `/api/v1/ansible/workflow-executions/${id}`,
    method: 'get'
  })
}

// 取消工作流执行
export function cancelWorkflowExecution(id) {
  return request({
    url: `/api/v1/ansible/workflow-executions/${id}/cancel`,
    method: 'post'
  })
}

// 删除工作流执行记录
export function deleteWorkflowExecution(id) {
  return request({
    url: `/api/v1/ansible/workflow-executions/${id}`,
    method: 'delete'
  })
}

// 获取工作流执行状态（实时）
export function getWorkflowExecutionStatus(id) {
  return request({
    url: `/api/v1/ansible/workflow-executions/${id}/status`,
    method: 'get'
  })
}

