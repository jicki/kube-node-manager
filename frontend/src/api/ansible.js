import request from '@/utils/request'

// 任务管理 API

/**
 * 列出任务
 */
export function listTasks(params) {
  return request({
    url: '/api/v1/ansible/tasks',
    method: 'get',
    params
  })
}

/**
 * 获取任务详情
 */
export function getTask(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}`,
    method: 'get'
  })
}

/**
 * 创建并执行任务
 */
export function createTask(data) {
  return request({
    url: '/api/v1/ansible/tasks',
    method: 'post',
    data
  })
}

/**
 * 取消任务
 */
export function cancelTask(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/cancel`,
    method: 'post'
  })
}

/**
 * 重试任务
 */
export function retryTask(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/retry`,
    method: 'post'
  })
}

/**
 * 获取任务日志
 */
export function getTaskLogs(id, params) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/logs`,
    method: 'get',
    params
  })
}

/**
 * 刷新任务状态
 */
export function refreshTaskStatus(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/refresh`,
    method: 'post'
  })
}

/**
 * 获取统计信息
 */
export function getStatistics() {
  return request({
    url: '/api/v1/ansible/statistics',
    method: 'get'
  })
}

// 模板管理 API

/**
 * 列出模板
 */
export function listTemplates(params) {
  return request({
    url: '/api/v1/ansible/templates',
    method: 'get',
    params
  })
}

/**
 * 获取模板详情
 */
export function getTemplate(id) {
  return request({
    url: `/api/v1/ansible/templates/${id}`,
    method: 'get'
  })
}

/**
 * 创建模板
 */
export function createTemplate(data) {
  return request({
    url: '/api/v1/ansible/templates',
    method: 'post',
    data
  })
}

/**
 * 更新模板
 */
export function updateTemplate(id, data) {
  return request({
    url: `/api/v1/ansible/templates/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除模板
 */
export function deleteTemplate(id) {
  return request({
    url: `/api/v1/ansible/templates/${id}`,
    method: 'delete'
  })
}

/**
 * 验证 playbook
 */
export function validateTemplate(data) {
  return request({
    url: '/api/v1/ansible/templates/validate',
    method: 'post',
    data
  })
}

// 主机清单管理 API

/**
 * 列出主机清单
 */
export function listInventories(params) {
  return request({
    url: '/api/v1/ansible/inventories',
    method: 'get',
    params
  })
}

/**
 * 获取主机清单详情
 */
export function getInventory(id) {
  return request({
    url: `/api/v1/ansible/inventories/${id}`,
    method: 'get'
  })
}

/**
 * 创建主机清单
 */
export function createInventory(data) {
  return request({
    url: '/api/v1/ansible/inventories',
    method: 'post',
    data
  })
}

/**
 * 更新主机清单
 */
export function updateInventory(id, data) {
  return request({
    url: `/api/v1/ansible/inventories/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除主机清单
 */
export function deleteInventory(id) {
  return request({
    url: `/api/v1/ansible/inventories/${id}`,
    method: 'delete'
  })
}

/**
 * 从集群生成主机清单
 */
export function generateInventory(data) {
  return request({
    url: '/api/v1/ansible/inventories/generate',
    method: 'post',
    data
  })
}

/**
 * 刷新 K8s 来源的主机清单
 */
export function refreshInventory(id) {
  return request({
    url: `/api/v1/ansible/inventories/${id}/refresh`,
    method: 'post'
  })
}

// WebSocket 连接

/**
 * 创建任务日志 WebSocket 连接
 */
export function connectTaskLogStream(taskId, onMessage, onError) {
  const token = localStorage.getItem('token')
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${wsProtocol}//${window.location.host}/api/v1/ansible/tasks/${taskId}/ws?token=${token}`
  
  const ws = new WebSocket(wsUrl)
  
  ws.onopen = () => {
    console.log('WebSocket connected for task:', taskId)
  }
  
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      if (onMessage) {
        onMessage(data)
      }
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error)
    }
  }
  
  ws.onerror = (error) => {
    console.error('WebSocket error:', error)
    if (onError) {
      onError(error)
    }
  }
  
  ws.onclose = () => {
    console.log('WebSocket closed for task:', taskId)
  }
  
  return ws
}

