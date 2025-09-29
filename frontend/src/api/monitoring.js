import request from '@/utils/request'

const monitoringApi = {
  // 获取监控状态
  getMonitoringStatus(clusterId) {
    return request({
      url: `/api/v1/clusters/${clusterId}/monitoring/status`,
      method: 'get'
    })
  },

  // 获取节点指标数据
  getNodeMetrics(clusterId, params = {}) {
    return request({
      url: `/api/v1/clusters/${clusterId}/monitoring/nodes`,
      method: 'get',
      params
    })
  },

  // 获取网络拓扑数据
  getNetworkTopology(clusterId) {
    return request({
      url: `/api/v1/clusters/${clusterId}/monitoring/topology`,
      method: 'get'
    })
  },

  // 执行网络连通性测试
  testNetworkConnectivity(clusterId, params = {}) {
    return request({
      url: `/api/v1/clusters/${clusterId}/monitoring/connectivity`,
      method: 'post',
      data: params
    })
  },

  // 获取告警信息
  getAlerts(clusterId, params = {}) {
    return request({
      url: `/api/v1/clusters/${clusterId}/monitoring/alerts`,
      method: 'get',
      params
    })
  },

  // 测试监控端点连接
  testMonitoringEndpoint(endpoint, type) {
    return request({
      url: '/api/v1/monitoring/test',
      method: 'post',
      data: {
        endpoint,
        type
      }
    })
  },

  // 获取 Prometheus 查询结果
  prometheusQuery(clusterId, query, params = {}) {
    return request({
      url: `/api/v1/clusters/${clusterId}/monitoring/query`,
      method: 'post',
      data: {
        query,
        ...params
      }
    })
  }
}

export default monitoringApi