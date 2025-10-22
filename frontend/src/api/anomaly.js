import request from '@/utils/request'

/**
 * 获取异常记录列表
 * @param {Object} params - 查询参数
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.node_name - 节点名称
 * @param {string} params.anomaly_type - 异常类型
 * @param {string} params.status - 异常状态
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 * @param {number} params.page - 页码
 * @param {number} params.page_size - 每页数量
 */
export function getAnomalies(params) {
  return request({
    url: '/api/v1/anomalies',
    method: 'get',
    params
  })
}

/**
 * 获取异常统计数据
 * @param {Object} params - 查询参数
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.anomaly_type - 异常类型
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 * @param {string} params.dimension - 统计维度 (day/week)
 */
export function getStatistics(params) {
  return request({
    url: '/api/v1/anomalies/statistics',
    method: 'get',
    params
  })
}

/**
 * 获取当前活跃的异常
 * @param {number} cluster_id - 集群ID（可选）
 */
export function getActiveAnomalies(cluster_id) {
  return request({
    url: '/api/v1/anomalies/active',
    method: 'get',
    params: cluster_id ? { cluster_id } : {}
  })
}

/**
 * 获取异常类型统计
 * @param {Object} params - 查询参数
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getTypeStatistics(params) {
  return request({
    url: '/api/v1/anomalies/type-statistics',
    method: 'get',
    params
  })
}

/**
 * 手动触发异常检测
 */
export function triggerCheck() {
  return request({
    url: '/api/v1/anomalies/check',
    method: 'post'
  })
}

