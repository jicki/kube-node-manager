const TOKEN_KEY = 'kube_node_manager_token'
const USER_KEY = 'kube_node_manager_user'
const CLUSTER_KEY = 'kube_node_manager_cluster'

/**
 * Token管理
 */
export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function removeToken() {
  localStorage.removeItem(TOKEN_KEY)
}

/**
 * 用户信息管理
 */
export function getUserInfo() {
  const userStr = localStorage.getItem(USER_KEY)
  return userStr ? JSON.parse(userStr) : null
}

export function setUserInfo(user) {
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function removeUserInfo() {
  localStorage.removeItem(USER_KEY)
}

/**
 * 当前集群管理
 */
export function getCurrentCluster() {
  const clusterStr = localStorage.getItem(CLUSTER_KEY)
  return clusterStr ? JSON.parse(clusterStr) : null
}

export function setCurrentCluster(cluster) {
  localStorage.setItem(CLUSTER_KEY, JSON.stringify(cluster))
}

export function removeCurrentCluster() {
  localStorage.removeItem(CLUSTER_KEY)
}

/**
 * 权限检查
 */
export function hasPermission(user, permission) {
  if (!user) return false
  if (user.role === 'admin') return true
  return user.permissions && user.permissions.includes(permission)
}

/**
 * 角色检查
 */
export function hasRole(user, role) {
  if (!user) return false
  return user.role === role
}

/**
 * 清理所有认证信息
 */
export function clearAuthData() {
  removeToken()
  removeUserInfo()
  removeCurrentCluster()
}

/**
 * 检查Token是否过期
 */
export function isTokenExpired(token) {
  if (!token) return true
  
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    const currentTime = Date.now() / 1000
    return payload.exp < currentTime
  } catch (error) {
    console.error('Token parsing error:', error)
    return true
  }
}

/**
 * 获取Token剩余时间（秒）
 */
export function getTokenRemainingTime(token) {
  if (!token) return 0
  
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    const currentTime = Date.now() / 1000
    return Math.max(0, payload.exp - currentTime)
  } catch (error) {
    console.error('Token parsing error:', error)
    return 0
  }
}

/**
 * 格式化权限列表
 */
export function formatPermissions(permissions) {
  if (!Array.isArray(permissions)) return []
  
  const permissionMap = {
    'node:read': '节点查看',
    'node:write': '节点操作',
    'label:read': '标签查看',
    'label:write': '标签管理',
    'taint:read': '污点查看',
    'taint:write': '污点管理',
    'user:read': '用户查看',
    'user:write': '用户管理',
    'cluster:read': '集群查看',
    'cluster:write': '集群管理',
    'audit:read': '审计查看'
  }
  
  return permissions.map(p => ({
    key: p,
    label: permissionMap[p] || p
  }))
}

/**
 * 生成随机字符串（用于状态参数等）
 */
export function generateRandomString(length = 32) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return result
}