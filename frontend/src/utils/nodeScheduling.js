/**
 * 节点调度状态相关工具函数
 */

/**
 * 获取节点的智能调度状态
 * @param {Object} node 节点对象
 * @returns {string} 调度状态：'schedulable' | 'limited' | 'unschedulable'
 */
export function getSmartSchedulingStatus(node) {
  // 如果节点被cordon（不可调度）
  if (node.schedulable === false) {
    return 'unschedulable'
  }
  
  // 检查是否有影响调度的污点
  const hasSchedulingTaints = node.taints && node.taints.length > 0 && 
    node.taints.some(taint => 
      taint.effect === 'NoSchedule' || 
      taint.effect === 'PreferNoSchedule'
    )
  
  if (hasSchedulingTaints) {
    return 'limited'
  }
  
  // 没有污点且可调度
  return 'schedulable'
}

/**
 * 获取调度状态的显示信息
 * @param {Object} node 节点对象
 * @returns {Object} 包含text, type, icon的显示信息对象
 */
export function getSchedulingStatusDisplay(node) {
  const status = getSmartSchedulingStatus(node)
  
  switch (status) {
    case 'schedulable':
      return {
        text: '可调度',
        type: 'success',
        icon: 'Check',
        value: 'schedulable'
      }
    case 'limited':
      return {
        text: '有限调度',
        type: 'warning', 
        icon: 'QuestionFilled',
        value: 'limited'
      }
    case 'unschedulable':
      return {
        text: '不可调度',
        type: 'danger',
        icon: 'Lock',
        value: 'unschedulable'
      }
    default:
      return {
        text: '未知',
        type: 'info',
        icon: 'QuestionFilled',
        value: 'unknown'
      }
  }
}

/**
 * 调度状态选项常量
 */
export const SCHEDULING_STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '可调度', value: 'schedulable' },
  { label: '有限调度', value: 'limited' },
  { label: '不可调度', value: 'unschedulable' }
]

/**
 * 获取调度状态的中文名称
 * @param {string} status 调度状态值
 * @returns {string} 中文名称
 */
export function getSchedulingStatusText(status) {
  const statusMap = {
    'schedulable': '可调度',
    'limited': '有限调度',
    'unschedulable': '不可调度'
  }
  return statusMap[status] || '未知'
}

/**
 * 检查节点是否有影响调度的污点
 * @param {Object} node 节点对象
 * @returns {boolean} 是否有影响调度的污点
 */
export function hasSchedulingTaints(node) {
  return node.taints && node.taints.length > 0 && 
    node.taints.some(taint => 
      taint.effect === 'NoSchedule' || 
      taint.effect === 'PreferNoSchedule'
    )
}

/**
 * 检查节点是否被禁止调度（cordon）
 * @param {Object} node 节点对象
 * @returns {boolean} 是否被禁止调度
 */
export function isCordoned(node) {
  return node.schedulable === false
}

/**
 * 获取影响调度的污点列表
 * @param {Object} node 节点对象
 * @returns {Array} 影响调度的污点列表
 */
export function getSchedulingTaints(node) {
  if (!node.taints || node.taints.length === 0) {
    return []
  }
  
  return node.taints.filter(taint => 
    taint.effect === 'NoSchedule' || 
    taint.effect === 'PreferNoSchedule'
  )
}

/**
 * 节点调度统计信息
 * @param {Array} nodes 节点列表
 * @returns {Object} 统计信息
 */
export function getSchedulingStats(nodes) {
  if (!Array.isArray(nodes)) {
    return {
      total: 0,
      schedulable: 0,
      limited: 0,
      unschedulable: 0
    }
  }
  
  const stats = {
    total: nodes.length,
    schedulable: 0,
    limited: 0,
    unschedulable: 0
  }
  
  nodes.forEach(node => {
    const status = getSmartSchedulingStatus(node)
    stats[status] = (stats[status] || 0) + 1
  })
  
  return stats
}
