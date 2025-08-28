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
  }
}

export default clusterApi