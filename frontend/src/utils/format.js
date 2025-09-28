/**
 * 格式化工具函数
 */

/**
 * 格式化文件大小
 */
export function formatFileSize(bytes, decimals = 2) {
  if (bytes === 0) return '0 Bytes'
  
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
  
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

/**
 * 格式化内存大小（Kubernetes格式）
 */
export function formatMemory(value, unit = 'Ki') {
  if (!value) return '0'
  
  const num = parseFloat(value)
  if (isNaN(num)) return value
  
  const unitMap = {
    Ki: 1024,
    Mi: 1024 * 1024,
    Gi: 1024 * 1024 * 1024,
    Ti: 1024 * 1024 * 1024 * 1024,
    K: 1000,
    M: 1000 * 1000,
    G: 1000 * 1000 * 1000,
    T: 1000 * 1000 * 1000 * 1000
  }
  
  const bytes = num * (unitMap[unit] || 1)
  return formatFileSize(bytes)
}

/**
 * 格式化CPU（统一格式，处理各种CPU数值）
 */
export function formatCPU(value, unit = 'm') {
  if (!value) return '0'
  
  const cpuStr = String(value).trim()
  
  // 处理纳秒cores (如: "2033181390n")  
  if (cpuStr.endsWith('n')) {
    const nanoStr = cpuStr.slice(0, -1)
    const nano = parseFloat(nanoStr)
    if (!isNaN(nano)) {
      const cores = nano / 1000000000  // 1 core = 1,000,000,000 nanoseconds
      return formatCPUValue(cores)
    }
  }
  
  // 处理微秒cores (如: "2033181u")  
  if (cpuStr.endsWith('u')) {
    const microStr = cpuStr.slice(0, -1)
    const micro = parseFloat(microStr)
    if (!isNaN(micro)) {
      const cores = micro / 1000000  // 1 core = 1,000,000 microseconds
      return formatCPUValue(cores)
    }
  }
  
  // 处理毫cores (如: "256m")
  if (cpuStr.endsWith('m')) {
    const milliStr = cpuStr.slice(0, -1)
    const milli = parseFloat(milliStr)
    if (!isNaN(milli)) {
      const cores = milli / 1000  // 1 core = 1000 millicores
      return formatCPUValue(cores)
    }
  }
  
  // 处理纯数字cores (如: "2.5")
  const num = parseFloat(cpuStr)
  if (!isNaN(num)) {
    return formatCPUValue(num)
  }
  
  return value
}

/**
 * 格式化CPU值为统一格式
 */
function formatCPUValue(cores) {
  if (cores >= 1) {
    return cores.toFixed(2) + ' 核'
  } else {
    const millicores = Math.round(cores * 1000)
    return millicores + 'm'
  }
}


/**
 * 格式化时间
 */
export function formatTime(time, format = 'YYYY-MM-DD HH:mm:ss') {
  if (!time) return '-'
  
  const date = new Date(time)
  if (isNaN(date.getTime())) return time
  
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  const second = String(date.getSeconds()).padStart(2, '0')
  
  return format
    .replace('YYYY', year)
    .replace('MM', month)
    .replace('DD', day)
    .replace('HH', hour)
    .replace('mm', minute)
    .replace('ss', second)
}

/**
 * 格式化相对时间
 */
export function formatRelativeTime(time) {
  if (!time) return '-'
  
  const now = new Date()
  const target = new Date(time)
  const diff = now - target
  
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (days > 0) {
    return `${days}天前`
  } else if (hours > 0) {
    return `${hours}小时前`
  } else if (minutes > 0) {
    return `${minutes}分钟前`
  } else {
    return `${seconds}秒前`
  }
}

/**
 * 格式化时长（秒）
 */
export function formatDuration(seconds) {
  if (!seconds || seconds < 0) return '0秒'
  
  const days = Math.floor(seconds / (24 * 3600))
  const hours = Math.floor((seconds % (24 * 3600)) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)
  
  const parts = []
  if (days > 0) parts.push(`${days}天`)
  if (hours > 0) parts.push(`${hours}小时`)
  if (minutes > 0) parts.push(`${minutes}分钟`)
  if (secs > 0 || parts.length === 0) parts.push(`${secs}秒`)
  
  return parts.join('')
}

/**
 * 格式化百分比
 */
export function formatPercentage(value, total, decimals = 1) {
  if (!total || total === 0) return '0%'
  const percent = (value / total) * 100
  return percent.toFixed(decimals) + '%'
}

/**
 * 格式化节点状态
 */
export function formatNodeStatus(status) {
  const statusMap = {
    Ready: { text: '就绪', type: 'success' },
    NotReady: { text: '未就绪', type: 'danger' },
    Unknown: { text: '未知', type: 'warning' },
    SchedulingDisabled: { text: '禁止调度', type: 'info' }
  }
  
  return statusMap[status] || { text: status, type: 'info' }
}

/**
 * 格式化节点角色
 */
export function formatNodeRoles(roles) {
  if (!Array.isArray(roles)) return '-'
  
  const roleMap = {
    master: '主节点',
    worker: '工作节点',
    control: '控制节点',
    'control-plane': '主节点'
  }
  
  return roles.map(role => roleMap[role] || role).join(', ')
}

/**
 * 格式化标签键值对
 */
export function formatLabel(key, value) {
  if (!value || value === 'true') {
    return key
  }
  return `${key}=${value}`
}

/**
 * 格式化污点
 */
export function formatTaint(taint) {
  const { key, value, effect } = taint
  let result = key
  
  if (value) {
    result += `=${value}`
  }
  
  if (effect) {
    result += `:${effect}`
  }
  
  return result
}

/**
 * 格式化污点效果
 */
export function formatTaintEffect(effect) {
  const effectMap = {
    NoSchedule: '禁止调度',
    PreferNoSchedule: '尽量不调度',
    NoExecute: '禁止执行'
  }
  
  return effectMap[effect] || effect
}

/**
 * 高亮搜索关键字
 */
export function highlightKeyword(text, keyword) {
  if (!keyword || !text) return text
  
  const regex = new RegExp(`(${keyword})`, 'gi')
  return text.replace(regex, '<mark>$1</mark>')
}

/**
 * 截断文本
 */
export function truncateText(text, maxLength = 50) {
  if (!text) return ''
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

/**
 * 格式化数字
 */
export function formatNumber(number, decimals = 0) {
  if (number == null || isNaN(number)) return '0'
  return Number(number).toLocaleString('zh-CN', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals
  })
}

/**
 * 格式化JSON
 */
export function formatJSON(obj, indent = 2) {
  try {
    return JSON.stringify(obj, null, indent)
  } catch (error) {
    return String(obj)
  }
}

/**
 * 驼峰转短横线
 */
export function camelToKebab(str) {
  return str.replace(/([A-Z])/g, '-$1').toLowerCase()
}

/**
 * 短横线转驼峰
 */
export function kebabToCamel(str) {
  return str.replace(/-([a-z])/g, (match, letter) => letter.toUpperCase())
}

/**
 * 首字母大写
 */
export function capitalize(str) {
  if (!str) return ''
  return str.charAt(0).toUpperCase() + str.slice(1)
}