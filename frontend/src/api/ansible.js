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
 * 暂停批次执行
 */
export function pauseBatch(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/pause-batch`,
    method: 'post'
  })
}

/**
 * 继续批次执行
 */
export function continueBatch(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/continue-batch`,
    method: 'post'
  })
}

/**
 * 停止批次执行
 */
export function stopBatch(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/stop-batch`,
    method: 'post'
  })
}

/**
 * 删除任务
 */
export function deleteTask(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}`,
    method: 'delete'
  })
}

/**
 * 批量删除任务
 */
export function batchDeleteTasks(ids) {
  return request({
    url: '/api/v1/ansible/tasks/batch-delete',
    method: 'post',
    data: { ids }
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

// SSH 密钥管理 API

/**
 * 列出 SSH 密钥
 */
export function listSSHKeys(params) {
  return request({
    url: '/api/v1/ansible/ssh-keys',
    method: 'get',
    params
  })
}

/**
 * 获取 SSH 密钥详情
 */
export function getSSHKey(id) {
  return request({
    url: `/api/v1/ansible/ssh-keys/${id}`,
    method: 'get'
  })
}

/**
 * 创建 SSH 密钥
 */
export function createSSHKey(data) {
  return request({
    url: '/api/v1/ansible/ssh-keys',
    method: 'post',
    data
  })
}

/**
 * 更新 SSH 密钥
 */
export function updateSSHKey(id, data) {
  return request({
    url: `/api/v1/ansible/ssh-keys/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除 SSH 密钥
 */
export function deleteSSHKey(id) {
  return request({
    url: `/api/v1/ansible/ssh-keys/${id}`,
    method: 'delete'
  })
}

/**
 * 测试 SSH 连接
 */
export function testSSHConnection(id, data) {
  return request({
    url: `/api/v1/ansible/ssh-keys/${id}/test`,
    method: 'post',
    data
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

// 定时任务调度 API

/**
 * 列出定时任务
 */
export function listSchedules(params) {
  return request({
    url: '/api/v1/ansible/schedules',
    method: 'get',
    params
  })
}

/**
 * 获取定时任务详情
 */
export function getSchedule(id) {
  return request({
    url: `/api/v1/ansible/schedules/${id}`,
    method: 'get'
  })
}

/**
 * 创建定时任务
 */
export function createSchedule(data) {
  return request({
    url: '/api/v1/ansible/schedules',
    method: 'post',
    data
  })
}

/**
 * 更新定时任务
 */
export function updateSchedule(id, data) {
  return request({
    url: `/api/v1/ansible/schedules/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除定时任务
 */
export function deleteSchedule(id) {
  return request({
    url: `/api/v1/ansible/schedules/${id}`,
    method: 'delete'
  })
}

/**
 * 启用/禁用定时任务
 */
export function toggleSchedule(id, enabled) {
  return request({
    url: `/api/v1/ansible/schedules/${id}/toggle`,
    method: 'post',
    data: { enabled }
  })
}

/**
 * 立即执行定时任务
 */
export function runScheduleNow(id) {
  return request({
    url: `/api/v1/ansible/schedules/${id}/run-now`,
    method: 'post'
  })
}

// 收藏管理 API

/**
 * 添加收藏
 */
export function addFavorite(data) {
  return request({
    url: '/api/v1/ansible/favorites',
    method: 'post',
    data
  })
}

/**
 * 移除收藏
 */
export function removeFavorite(targetType, targetId) {
  return request({
    url: '/api/v1/ansible/favorites',
    method: 'delete',
    params: { target_type: targetType, target_id: targetId }
  })
}

/**
 * 列出收藏
 */
export function listFavorites(targetType) {
  return request({
    url: '/api/v1/ansible/favorites',
    method: 'get',
    params: { target_type: targetType }
  })
}

/**
 * 获取最近使用的任务
 */
export function getRecentTasks(limit = 10) {
  return request({
    url: '/api/v1/ansible/recent-tasks',
    method: 'get',
    params: { limit }
  })
}

/**
 * 获取任务历史详情
 */
export function getTaskHistory(id) {
  return request({
    url: `/api/v1/ansible/task-history/${id}`,
    method: 'get'
  })
}

/**
 * 删除任务历史
 */
export function deleteTaskHistory(id) {
  return request({
    url: `/api/v1/ansible/task-history/${id}`,
    method: 'delete'
  })
}

// 前置检查 API

/**
 * 执行前置检查
 */
export function runPreflightChecks(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/preflight-checks`,
    method: 'post'
  })
}

/**
 * 获取前置检查结果
 */
export function getPreflightChecks(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/preflight-checks`,
    method: 'get'
  })
}

// 任务执行预估 API

/**
 * 基于模板预估任务执行时间
 */
export function estimateByTemplate(templateId) {
  return request({
    url: '/api/v1/ansible/estimate/template',
    method: 'get',
    params: { template_id: templateId }
  })
}

/**
 * 基于清单预估任务执行时间
 */
export function estimateByInventory(inventoryId) {
  return request({
    url: '/api/v1/ansible/estimate/inventory',
    method: 'get',
    params: { inventory_id: inventoryId }
  })
}

/**
 * 基于模板和清单组合预估任务执行时间
 */
export function estimateByTemplateAndInventory(templateId, inventoryId) {
  return request({
    url: '/api/v1/ansible/estimate/combined',
    method: 'get',
    params: { 
      template_id: templateId, 
      inventory_id: inventoryId 
    }
  })
}

// 任务队列统计 API

/**
 * 获取任务队列统计信息
 */
export function getQueueStats() {
  return request({
    url: '/api/v1/ansible/queue/stats',
    method: 'get'
  })
}

// 任务执行可视化 API

/**
 * 获取任务执行可视化数据
 */
export function getTaskVisualization(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/visualization`,
    method: 'get'
  })
}

/**
 * 获取任务时间线摘要
 */
export function getTaskTimelineSummary(id) {
  return request({
    url: `/api/v1/ansible/tasks/${id}/timeline-summary`,
    method: 'get'
  })
}

