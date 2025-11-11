import request from '@/utils/request'

/**
 * 根据ID获取单个异常记录
 * @param {number} id - 异常记录ID
 */
export function getAnomalyById(id) {
  return request({
    url: `/api/v1/anomalies/${id}`,
    method: 'get'
  })
}

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

// ==================== 高级统计 API ====================

/**
 * 获取按节点角色聚合的统计
 * @param {Object} params - 查询参数
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getRoleStatistics(params) {
  return request({
    url: '/api/v1/anomalies/role-statistics',
    method: 'get',
    params
  })
}

/**
 * 获取按集群聚合的统计
 * @param {Object} params - 查询参数
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getClusterAggregate(params) {
  return request({
    url: '/api/v1/anomalies/cluster-aggregate',
    method: 'get',
    params
  })
}

/**
 * 获取单个节点的历史异常趋势
 * @param {Object} params - 查询参数
 * @param {string} params.node_name - 节点名称（必填）
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 * @param {string} params.dimension - 时间维度 (hour/day/week)
 */
export function getNodeTrend(params) {
  return request({
    url: '/api/v1/anomalies/node-trend',
    method: 'get',
    params
  })
}

/**
 * 获取 MTTR（平均恢复时间）统计
 * @param {Object} params - 查询参数
 * @param {string} params.entity_type - 实体类型 (node/cluster)
 * @param {string} params.entity_name - 实体名称（节点名或集群名）
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getMTTR(params) {
  return request({
    url: '/api/v1/anomalies/mttr',
    method: 'get',
    params
  })
}

/**
 * 获取 SLA 可用性统计
 * @param {Object} params - 查询参数
 * @param {string} params.entity_type - 实体类型 (node/cluster)
 * @param {string} params.entity_name - 实体名称（节点名或集群名）
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getSLA(params) {
  return request({
    url: '/api/v1/anomalies/sla',
    method: 'get',
    params
  })
}

/**
 * 获取异常恢复率和复发率统计
 * @param {Object} params - 查询参数
 * @param {string} params.entity_type - 实体类型 (node/cluster)
 * @param {string} params.entity_name - 实体名称（节点名或集群名）
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getRecoveryMetrics(params) {
  return request({
    url: '/api/v1/anomalies/recovery-metrics',
    method: 'get',
    params
  })
}

/**
 * 获取节点健康度评分
 * @param {Object} params - 查询参数
 * @param {string} params.node_name - 节点名称（必填）
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getNodeHealth(params) {
  return request({
    url: '/api/v1/anomalies/node-health',
    method: 'get',
    params
  })
}

/**
 * 获取热力图数据（时间 × 节点矩阵）
 * @param {Object} params - 查询参数
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getHeatmapData(params) {
  return request({
    url: '/api/v1/anomalies/heatmap',
    method: 'get',
    params
  })
}

/**
 * 获取日历热力图数据（按日期聚合）
 * @param {Object} params - 查询参数
 * @param {number} params.cluster_id - 集群ID
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getCalendarData(params) {
  return request({
    url: '/api/v1/anomalies/calendar',
    method: 'get',
    params
  })
}

/**
 * 获取健康度最低的节点列表
 * @param {Object} params - 查询参数
 * @param {number} params.cluster_id - 集群ID
 * @param {number} params.limit - 返回数量限制，默认10
 * @param {string} params.start_time - 开始时间
 * @param {string} params.end_time - 结束时间
 */
export function getTopUnhealthyNodes(params) {
  return request({
    url: '/api/v1/anomalies/top-unhealthy-nodes',
    method: 'get',
    params
  })
}

// ==================== 异常报告配置 API ====================

/**
 * 获取报告配置列表
 */
export function getReportConfigs() {
  return request({
    url: '/api/v1/anomaly-reports/configs',
    method: 'get'
  })
}

/**
 * 获取单个报告配置
 * @param {number} id - 配置ID
 */
export function getReportConfig(id) {
  return request({
    url: `/api/v1/anomaly-reports/configs/${id}`,
    method: 'get'
  })
}

/**
 * 创建报告配置
 * @param {Object} data - 配置数据
 */
export function createReportConfig(data) {
  return request({
    url: '/api/v1/anomaly-reports/configs',
    method: 'post',
    data
  })
}

/**
 * 更新报告配置
 * @param {number} id - 配置ID
 * @param {Object} data - 配置数据
 */
export function updateReportConfig(id, data) {
  return request({
    url: `/api/v1/anomaly-reports/configs/${id}`,
    method: 'put',
    data
  })
}

/**
 * 删除报告配置
 * @param {number} id - 配置ID
 */
export function deleteReportConfig(id) {
  return request({
    url: `/api/v1/anomaly-reports/configs/${id}`,
    method: 'delete'
  })
}

/**
 * 测试报告发送
 * @param {number} id - 配置ID
 */
export function testReportSend(id) {
  return request({
    url: `/api/v1/anomaly-reports/configs/${id}/test`,
    method: 'post'
  })
}

/**
 * 手动执行报告生成
 * @param {number} id - 配置ID
 */
export function runReportNow(id) {
  return request({
    url: `/api/v1/anomaly-reports/configs/${id}/run`,
    method: 'post'
  })
}

