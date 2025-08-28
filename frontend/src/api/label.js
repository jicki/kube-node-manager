import request from '@/utils/request'

const labelApi = {
  // 获取节点标签
  getNodeLabels(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}/labels`,
      method: 'get'
    })
  },

  // 添加节点标签
  addNodeLabel(nodeName, data) {
    return request({
      url: `/api/v1/nodes/${nodeName}/labels`,
      method: 'post',
      data
    })
  },

  // 更新节点标签
  updateNodeLabel(nodeName, key, data) {
    return request({
      url: `/api/v1/nodes/${nodeName}/labels/${key}`,
      method: 'put',
      data
    })
  },

  // 删除节点标签
  deleteNodeLabel(nodeName, key) {
    return request({
      url: `/api/v1/nodes/${nodeName}/labels/${key}`,
      method: 'delete'
    })
  },

  // 批量添加标签
  batchAddLabels(nodeNames, labels) {
    return request({
      url: '/api/v1/nodes/labels/batch-add',
      method: 'post',
      data: { nodes: nodeNames, labels }
    })
  },

  // 批量删除标签
  batchDeleteLabels(nodeNames, keys) {
    return request({
      url: '/api/v1/nodes/labels/batch-delete',
      method: 'post',
      data: { nodes: nodeNames, keys }
    })
  },

  // 获取所有标签
  getAllLabels(params) {
    return request({
      url: '/api/v1/labels',
      method: 'get',
      params
    })
  },

  // 搜索标签
  searchLabels(keyword) {
    return request({
      url: '/api/v1/labels/search',
      method: 'get',
      params: { keyword }
    })
  },

  // 获取标签使用统计
  getLabelStats() {
    return request({
      url: '/api/v1/labels/stats',
      method: 'get'
    })
  },

  // 获取推荐标签
  getRecommendedLabels(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}/labels/recommendations`,
      method: 'get'
    })
  },

  // 验证标签格式
  validateLabel(data) {
    return request({
      url: '/api/v1/labels/validate',
      method: 'post',
      data
    })
  },

  // 导入标签模板
  importLabels(data) {
    return request({
      url: '/api/v1/labels/import',
      method: 'post',
      data
    })
  },

  // 导出标签
  exportLabels(params) {
    return request({
      url: '/api/v1/labels/export',
      method: 'get',
      params,
      responseType: 'blob'
    })
  }
}

export default labelApi