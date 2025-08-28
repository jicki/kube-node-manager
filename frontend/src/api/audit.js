import request from '@/utils/request'

const auditApi = {
  // 获取审计日志
  getAuditLogs(params) {
    return request({
      url: '/api/v1/audit/logs',
      method: 'get',
      params
    })
  },

  // 获取审计日志详情
  getAuditLogDetail(id) {
    return request({
      url: `/api/v1/audit/logs/${id}`,
      method: 'get'
    })
  },

  // 导出审计日志
  exportAuditLogs(params) {
    return request({
      url: '/api/v1/audit/logs/export',
      method: 'get',
      params,
      responseType: 'blob'
    })
  },

  // 获取审计统计
  getAuditStats(params) {
    return request({
      url: '/api/v1/audit/stats',
      method: 'get',
      params
    })
  },

  // 获取用户操作日志
  getUserOperations(userId, params) {
    return request({
      url: `/api/v1/audit/users/${userId}/operations`,
      method: 'get',
      params
    })
  },

  // 获取节点操作日志
  getNodeOperations(nodeName, params) {
    return request({
      url: `/api/v1/audit/nodes/${nodeName}/operations`,
      method: 'get',
      params
    })
  },

  // 获取操作类型统计
  getOperationStats(params) {
    return request({
      url: '/api/v1/audit/operations/stats',
      method: 'get',
      params
    })
  },

  // 搜索审计日志
  searchAuditLogs(keyword, params) {
    return request({
      url: '/api/v1/audit/logs/search',
      method: 'get',
      params: { keyword, ...params }
    })
  },

  // 清理过期日志
  cleanupExpiredLogs(days) {
    return request({
      url: '/api/v1/audit/logs/cleanup',
      method: 'post',
      data: { days }
    })
  }
}

export default auditApi