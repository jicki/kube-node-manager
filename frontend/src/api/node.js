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

  // 禁止调度节点
  cordonNode(nodeName, clusterName, reason = '') {
    return request({
      url: `/api/v1/nodes/${nodeName}/cordon`,
      method: 'post',
      data: {
        cluster_name: clusterName,
        reason: reason
      }
    })
  },

  // 解除调度节点
  uncordonNode(nodeName, clusterName, reason = '') {
    return request({
      url: `/api/v1/nodes/${nodeName}/uncordon`,
      method: 'post',
      data: {
        cluster_name: clusterName,
        reason: reason
      }
    })
  },

  // 驱逐节点
  drainNode(nodeName, clusterName, reason = '') {
    return request({
      url: `/api/v1/nodes/${nodeName}/drain`,
      method: 'post',
      data: {
        cluster_name: clusterName,
        reason: reason
      }
    })
  },


  // 批量禁止调度节点
  batchCordon(nodeNames, clusterName, reason = '') {
    return request({
      url: '/api/v1/nodes/batch-cordon',
      method: 'post',
      data: { 
        nodes: nodeNames,
        cluster_name: clusterName,
        reason: reason
      }
    })
  },

  // 批量解除调度节点
  batchUncordon(nodeNames, clusterName, reason = '') {
    return request({
      url: '/api/v1/nodes/batch-uncordon',
      method: 'post',
      data: { 
        nodes: nodeNames,
        cluster_name: clusterName,
        reason: reason
      }
    })
  },

  // 批量驱逐节点
  batchDrain(nodeNames, clusterName, reason = '') {
    return request({
      url: '/api/v1/nodes/batch-drain',
      method: 'post',
      data: { 
        nodes: nodeNames,
        cluster_name: clusterName,
        reason: reason
      }
    })
  },

  // 批量禁止调度（带进度）
  batchCordonWithProgress(nodeNames, clusterName, reason = '') {
    return request({
      url: '/api/v1/nodes/batch-cordon-progress',
      method: 'post',
      data: { 
        nodes: nodeNames,
        cluster_name: clusterName,
        reason: reason
      }
    })
  },

  // 批量解除调度（带进度）
  batchUncordonWithProgress(nodeNames, clusterName, reason = '') {
    return request({
      url: '/api/v1/nodes/batch-uncordon-progress',
      method: 'post',
      data: { 
        nodes: nodeNames,
        cluster_name: clusterName,
        reason: reason
      }
    })
  },

  // 批量驱逐节点（带进度）
  batchDrainWithProgress(nodeNames, clusterName, reason = '') {
    return request({
      url: '/api/v1/nodes/batch-drain-progress',
      method: 'post',
      data: { 
        nodes: nodeNames,
        cluster_name: clusterName,
        reason: reason
      }
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
  },

  // 获取节点禁止调度历史
  getCordonHistory(nodeName, clusterName) {
    return request({
      url: '/api/v1/nodes/cordon-history',
      method: 'post',
      data: {
        node_name: nodeName,
        cluster_name: clusterName
      }
    })
  },

  // 批量获取节点禁止调度历史
  getBatchCordonHistory(data) {
    return request({
      url: '/api/v1/nodes/batch-cordon-history',
      method: 'post',
      data
    })
  },

  // ============== 新增：优化方法 ==============

  // 获取节点列表（带超时）
  getNodesWithTimeout(params, timeout = 15000) {
    return request({
      url: '/api/v1/nodes',
      method: 'get',
      params,
      timeout
    })
  },

  // 并行获取多个集群的节点列表
  async getAllClustersNodesParallel(clusterNames, timeout = 15000) {
    const promises = clusterNames.map(clusterName =>
      this.getNodesWithTimeout({ cluster_name: clusterName }, timeout)
        .then(res => ({
          clusterName,
          success: true,
          data: res.data || []
        }))
        .catch(err => {
          console.warn(`Failed to load cluster ${clusterName}:`, err)
          return {
            clusterName,
            success: false,
            error: err.message || 'Failed to fetch',
            data: []
          }
        })
    )

    return await Promise.all(promises)
  },

  // 快速检查集群节点可用性（超时时间更短）
  async quickCheckClusterNodes(clusterName) {
    try {
      const res = await this.getNodesWithTimeout({ cluster_name: clusterName }, 5000)
      return {
        clusterName,
        available: true,
        nodeCount: res.data?.length || 0
      }
    } catch (err) {
      return {
        clusterName,
        available: false,
        error: err.message
      }
    }
  }
}

export default nodeApi