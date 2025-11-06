import request from '@/utils/request'

const clusterApi = {
  // 获取集群列表
  getClusters(params) {
    return request({
      url: '/api/v1/clusters',
      method: 'get',
      params
    })
  },

  // 获取集群详情
  getClusterDetail(id) {
    return request({
      url: `/api/v1/clusters/${id}`,
      method: 'get'
    })
  },

  // 添加集群
  addCluster(data) {
    return request({
      url: '/api/v1/clusters',
      method: 'post',
      data
    })
  },

  // 更新集群
  updateCluster(id, data) {
    return request({
      url: `/api/v1/clusters/${id}`,
      method: 'put',
      data
    })
  },

  // 删除集群
  deleteCluster(id) {
    return request({
      url: `/api/v1/clusters/${id}`,
      method: 'delete'
    })
  },

  // 测试集群连接
  testConnection(id) {
    return request({
      url: `/api/v1/clusters/${id}/test`,
      method: 'post'
    })
  },

  // 获取集群状态
  getClusterStatus(id) {
    return request({
      url: `/api/v1/clusters/${id}/status`,
      method: 'get'
    })
  },

  // 获取集群资源使用情况
  getClusterResources(id) {
    return request({
      url: `/api/v1/clusters/${id}/resources`,
      method: 'get'
    })
  },

  // 切换当前集群
  switchCluster(id) {
    return request({
      url: `/api/v1/clusters/${id}/switch`,
      method: 'post'
    })
  },

  // 同步集群信息
  syncCluster(id) {
    return request({
      url: `/api/v1/clusters/${id}/sync`,
      method: 'post'
    })
  },

  // ============== 新增：优化方法 ==============

  // 获取集群列表（带超时）
  getClustersWithTimeout(params, timeout = 10000) {
    return request({
      url: '/api/v1/clusters',
      method: 'get',
      params,
      timeout
    })
  },

  // 获取集群健康状态
  getClusterHealth(clusterName) {
    return request({
      url: '/api/v1/clusters/health',
      method: 'get',
      params: { cluster_name: clusterName },
      timeout: 5000 // 5秒超时
    })
  },

  // 获取所有集群健康状态
  getAllClustersHealth() {
    return request({
      url: '/api/v1/clusters/health/all',
      method: 'get',
      timeout: 5000
    })
  },

  // 重置断路器
  resetCircuitBreaker(clusterName) {
    return request({
      url: '/api/v1/clusters/circuit-breaker/reset',
      method: 'post',
      params: { cluster_name: clusterName }
    })
  },

  // 并行获取多个集群的状态
  async getAllClustersStatusParallel(clusters, timeout = 10000) {
    const promises = clusters.map(cluster =>
      this.getClusterStatus(cluster.id)
        .catch(err => {
          console.warn(`Failed to get status for cluster ${cluster.name}:`, err)
          return { 
            error: err.message || 'Failed to fetch',
            cluster: cluster.name,
            clusterId: cluster.id
          }
        })
    )

    return await Promise.all(promises)
  }
}

export default clusterApi