import request from '@/utils/request'

const taintApi = {
  // 获取节点污点
  getNodeTaints(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}/taints`,
      method: 'get'
    })
  },

  // 添加节点污点
  addNodeTaint(nodeName, data) {
    return request({
      url: `/api/v1/nodes/${nodeName}/taints`,
      method: 'post',
      data
    })
  },

  // 更新节点污点
  updateNodeTaint(nodeName, key, data) {
    return request({
      url: `/api/v1/nodes/${nodeName}/taints/${key}`,
      method: 'put',
      data
    })
  },

  // 删除节点污点
  deleteNodeTaint(nodeName, key) {
    return request({
      url: `/api/v1/nodes/${nodeName}/taints/${key}`,
      method: 'delete'
    })
  },

  // 批量添加污点
  batchAddTaints(requestData) {
    return request({
      url: '/api/v1/nodes/taints/batch-add',
      method: 'post',
      data: requestData
    })
  },

  // 批量删除污点
  batchDeleteTaints(requestData, config = {}) {
    return request({
      url: '/api/v1/nodes/taints/batch-delete',
      method: 'post',
      data: requestData,
      ...config
    })
  },

  // 获取所有污点
  getAllTaints(params) {
    return request({
      url: '/api/v1/taints',
      method: 'get',
      params
    })
  },

  // 搜索污点
  searchTaints(keyword) {
    return request({
      url: '/api/v1/taints/search',
      method: 'get',
      params: { keyword }
    })
  },

  // 获取污点使用统计
  getTaintStats() {
    return request({
      url: '/api/v1/taints/stats',
      method: 'get'
    })
  },

  // 获取污点效果选项
  getTaintEffects() {
    return request({
      url: '/api/v1/taints/effects',
      method: 'get'
    })
  },

  // 验证污点格式
  validateTaint(data) {
    return request({
      url: '/api/v1/taints/validate',
      method: 'post',
      data
    })
  },

  // 获取推荐污点
  getRecommendedTaints(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}/taints/recommendations`,
      method: 'get'
    })
  },

  // 导入污点模板
  importTaints(data) {
    return request({
      url: '/api/v1/taints/import',
      method: 'post',
      data
    })
  },

  // 导出污点
  exportTaints(params) {
    return request({
      url: '/api/v1/taints/export',
      method: 'get',
      params,
      responseType: 'blob'
    })
  },

  // 清除所有污点
  clearAllTaints(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}/taints/clear`,
      method: 'post'
    })
  },

  // 污点模板相关API
  // 获取污点模板列表
  getTemplateList(params = {}) {
    return request({
      url: '/api/v1/taints/templates',
      method: 'get',
      params
    })
  },

  // 创建污点模板
  createTemplate(data) {
    return request({
      url: '/api/v1/taints/templates',
      method: 'post',
      data
    })
  },

  // 更新污点模板
  updateTemplate(id, data) {
    return request({
      url: `/api/v1/taints/templates/${id}`,
      method: 'put',
      data
    })
  },

  // 删除污点模板
  deleteTemplate(id) {
    return request({
      url: `/api/v1/taints/templates/${id}`,
      method: 'delete'
    })
  },

  // 应用污点模板到节点
  applyTemplate(data) {
    return request({
      url: '/api/v1/taints/templates/apply',
      method: 'post',
      data
    })
  }
}

export default taintApi