import request from '@/utils/request'

const nodeApi = {
  // 获取节点列表
  getNodes(params) {
    return request({
      url: '/api/v1/nodes',
      method: 'get',
      params
    })
  },

  // 获取节点详情
  getNodeDetail(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}`,
      method: 'get'
    })
  },

  // 封锁节点
  cordonNode(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}/cordon`,
      method: 'post'
    })
  },

  // 取消封锁节点
  uncordonNode(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}/uncordon`,
      method: 'post'
    })
  },

  // 驱逐节点
  drainNode(nodeName, options = {}) {
    return request({
      url: `/api/v1/nodes/${nodeName}/drain`,
      method: 'post',
      data: options
    })
  },

  // 批量封锁节点
  batchCordon(nodeNames) {
    return request({
      url: '/api/v1/nodes/batch-cordon',
      method: 'post',
      data: { nodes: nodeNames }
    })
  },

  // 批量取消封锁节点
  batchUncordon(nodeNames) {
    return request({
      url: '/api/v1/nodes/batch-uncordon',
      method: 'post',
      data: { nodes: nodeNames }
    })
  },

  // 批量驱逐节点
  batchDrain(nodeNames, options = {}) {
    return request({
      url: '/api/v1/nodes/batch-drain',
      method: 'post',
      data: { nodes: nodeNames, options }
    })
  },

  // 获取节点资源使用情况
  getNodeResources(nodeName) {
    return request({
      url: `/api/v1/nodes/${nodeName}/resources`,
      method: 'get'
    })
  },

  // 获取节点上的Pods
  getNodePods(nodeName, params) {
    return request({
      url: `/api/v1/nodes/${nodeName}/pods`,
      method: 'get',
      params
    })
  },

  // 获取节点事件
  getNodeEvents(nodeName, params) {
    return request({
      url: `/api/v1/nodes/${nodeName}/events`,
      method: 'get',
      params
    })
  },

  // 获取节点统计信息
  getNodeStats() {
    return request({
      url: '/api/v1/nodes/stats',
      method: 'get'
    })
  }
}

export default nodeApi