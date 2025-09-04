import { defineStore } from 'pinia'
import nodeApi from '@/api/node'
import { useClusterStore } from './cluster'

// 获取智能调度状态的辅助函数
function getSmartSchedulingStatus(node) {
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

export const useNodeStore = defineStore('node', {
  state: () => ({
    nodes: [],
    selectedNodes: [],
    currentClusterName: '', // 当前集群名称，用于缓存识别
    nodeStats: {
      total: 0,
      ready: 0,
      notReady: 0,
      unknown: 0,
      schedulable: 0,
      limited: 0,
      unschedulable: 0,
      ownership: {} // 节点归属统计 { "归属名称": 数量, "无归属": 数量 }
    },
    pagination: {
      current: 1,
      size: 20,
      total: 0
    },
    filters: {
      name: '',
      status: '',
      role: '',
      cluster_name: '',
      labelKey: '',
      labelValue: '',
      taintKey: '',
      taintValue: '',
      taintEffect: '',
      nodeOwnership: '' // deeproute.cn/user-type 标签过滤
    },
    loading: false
  }),

  getters: {
    readyNodes: (state) => state.nodes.filter(node => node.status === 'Ready'),
    notReadyNodes: (state) => state.nodes.filter(node => 
      node.status === 'NotReady' || node.status === 'SchedulingDisabled'
    ),
    unknownNodes: (state) => state.nodes.filter(node => node.status === 'Unknown'),
    masterNodes: (state) => state.nodes.filter(node => {
      if (!node.roles || !Array.isArray(node.roles)) return false
      return node.roles.some(role => 
        role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
      )
    }),
    workerNodes: (state) => state.nodes.filter(node => {
      if (!node.roles || !Array.isArray(node.roles)) return true // 无角色默认为worker
      // 如果不包含master相关角色，则为worker
      return !node.roles.some(role => 
        role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
      )
    }),
    // 调度状态统计
    schedulableNodes: (state) => state.nodes.filter(node => getSmartSchedulingStatus(node) === 'schedulable'),
    limitedNodes: (state) => state.nodes.filter(node => getSmartSchedulingStatus(node) === 'limited'),
    unschedulableNodes: (state) => state.nodes.filter(node => getSmartSchedulingStatus(node) === 'unschedulable'),
    // 节点归属选项 (从所有节点的 deeproute.cn/user-type 标签中提取)
    nodeOwnershipOptions: (state) => {
      const ownershipSet = new Set()
      let hasNoOwnership = false
      let noOwnershipNodes = []
      
      state.nodes.forEach(node => {
        const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
        // 检查标签是否存在且不为空字符串
        if (userTypeLabel && userTypeLabel.trim() !== '') {
          ownershipSet.add(userTypeLabel)
        } else {
          hasNoOwnership = true
          noOwnershipNodes.push({
            name: node.name,
            hasLabels: !!node.labels,
            userTypeValue: userTypeLabel
          })
        }
      })
      
      // 调试信息
      console.log('节点总数:', state.nodes.length)
      console.log('有deeproute.cn/user-type标签的节点数量:', ownershipSet.size)
      console.log('无deeproute.cn/user-type标签的节点:', noOwnershipNodes)
      console.log('hasNoOwnership:', hasNoOwnership)
      
      const options = Array.from(ownershipSet).sort()
      
      // 如果有节点没有 deeproute.cn/user-type 标签，添加“无归属”选项
      if (hasNoOwnership) {
        options.unshift('无归属') // 添加到数组开头
      }
      
      console.log('最终选项数组:', options)
      return options
    },
    filteredNodes: (state) => {
      let result = state.nodes
      const originalCount = result.length
      
      // 检查是否有任何过滤条件
      const hasFilters = !!(state.filters.name || state.filters.status || state.filters.role || 
                           state.filters.schedulable || state.filters.labelKey || state.filters.taintKey || 
                           state.filters.nodeOwnership)
      
      // 只在有过滤条件且结果数量异常时输出调试信息
      if (hasFilters && originalCount > 0) {
        console.log('开始过滤节点:', {
          原始节点数: originalCount,
          过滤条件: state.filters
        })
      }
      
      if (state.filters.name) {
        result = result.filter(node => 
          node.name.toLowerCase().includes(state.filters.name.toLowerCase())
        )
      }
      
      if (state.filters.status) {
        result = result.filter(node => node.status === state.filters.status)
      }
      
      if (state.filters.role) {
        result = result.filter(node => {
          if (!node.roles || !Array.isArray(node.roles)) {
            return state.filters.role === 'worker' // 无角色视为worker
          }
          
          if (state.filters.role === 'master') {
            // 检查是否为master相关角色
            return node.roles.some(role => 
              role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
            )
          } else if (state.filters.role === 'worker') {
            // 检查是否为worker (不包含master相关角色)
            return !node.roles.some(role => 
              role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
            )
          }
          
          return false
        })
      }
      
      // 调度状态筛选
      if (state.filters.schedulable) {
        result = result.filter(node => {
          const schedulingStatus = getSmartSchedulingStatus(node)
          return schedulingStatus === state.filters.schedulable
        })
      }
      
      // 标签筛选
      if (state.filters.labelKey) {
        result = result.filter(node => {
          if (!node.labels || !node.labels[state.filters.labelKey]) {
            return false
          }
          // 如果指定了标签值，进行精确匹配
          if (state.filters.labelValue) {
            return node.labels[state.filters.labelKey] === state.filters.labelValue
          }
          // 否则只检查标签键是否存在
          return true
        })
      }
      
      // 污点筛选
      if (state.filters.taintKey) {
        result = result.filter(node => {
          if (!node.taints || node.taints.length === 0) {
            return false
          }
          return node.taints.some(taint => {
            if (taint.key !== state.filters.taintKey) {
              return false
            }
            // 如果指定了污点值，进行值匹配
            if (state.filters.taintValue && taint.value !== state.filters.taintValue) {
              return false
            }
            // 如果指定了污点效果，进行效果匹配
            if (state.filters.taintEffect && taint.effect !== state.filters.taintEffect) {
              return false
            }
            return true
          })
        })
      }
      
      // 节点归属筛选 (deeproute.cn/user-type)
      if (state.filters.nodeOwnership) {
        result = result.filter(node => {
          const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
          
          // 如果选择的是"无归属"，过滤出没有或为空的 deeproute.cn/user-type 标签的节点
          if (state.filters.nodeOwnership === '无归属') {
            return !userTypeLabel || userTypeLabel.trim() === ''
          }
          
          // 否则过滤具有匹配标签值的节点
          if (!userTypeLabel || userTypeLabel.trim() === '') {
            return false
          }
          return userTypeLabel === state.filters.nodeOwnership
        })
      }
      
      // 只在有过滤条件或结果为空时输出最终结果
      if (hasFilters || result.length === 0) {
        console.log(`过滤结果: ${originalCount} → ${result.length}`)
      }
      
      return result
    },
    // 添加分页后的节点列表
    paginatedNodes: (state) => {
      // 重新实现过滤逻辑，避免 getter 循环依赖
      let filtered = state.nodes
      
      // 应用所有过滤条件
      if (state.filters.name) {
        filtered = filtered.filter(node => 
          node.name.toLowerCase().includes(state.filters.name.toLowerCase())
        )
      }
      
      if (state.filters.status) {
        filtered = filtered.filter(node => node.status === state.filters.status)
      }
      
      if (state.filters.role) {
        filtered = filtered.filter(node => {
          if (!node.roles || !Array.isArray(node.roles)) {
            return state.filters.role === 'worker'
          }
          
          if (state.filters.role === 'master') {
            return node.roles.some(role => 
              role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
            )
          } else if (state.filters.role === 'worker') {
            return !node.roles.some(role => 
              role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
            )
          }
          
          return false
        })
      }
      
      if (state.filters.schedulable) {
        filtered = filtered.filter(node => {
          const schedulingStatus = getSmartSchedulingStatus(node)
          return schedulingStatus === state.filters.schedulable
        })
      }
      
      if (state.filters.labelKey) {
        filtered = filtered.filter(node => {
          if (!node.labels || !node.labels[state.filters.labelKey]) {
            return false
          }
          if (state.filters.labelValue) {
            return node.labels[state.filters.labelKey] === state.filters.labelValue
          }
          return true
        })
      }
      
      if (state.filters.taintKey) {
        filtered = filtered.filter(node => {
          if (!node.taints || node.taints.length === 0) {
            return false
          }
          return node.taints.some(taint => {
            if (taint.key !== state.filters.taintKey) {
              return false
            }
            if (state.filters.taintValue && taint.value !== state.filters.taintValue) {
              return false
            }
            if (state.filters.taintEffect && taint.effect !== state.filters.taintEffect) {
              return false
            }
            return true
          })
        })
      }
      
      if (state.filters.nodeOwnership) {
        filtered = filtered.filter(node => {
          const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
          
          if (state.filters.nodeOwnership === '无归属') {
            return !userTypeLabel || userTypeLabel.trim() === ''
          }
          
          if (!userTypeLabel || userTypeLabel.trim() === '') {
            return false
          }
          return userTypeLabel === state.filters.nodeOwnership
        })
      }
      
      // 分页计算
      const start = (state.pagination.current - 1) * state.pagination.size
      const end = start + state.pagination.size
      const result = filtered.slice(start, end)
      
      // 只在结果为空但应该有数据时输出调试信息
      if (result.length === 0 && filtered.length > 0) {
        console.warn('分页异常:', {
          filteredCount: filtered.length,
          currentPage: state.pagination.current,
          pageSize: state.pagination.size,
          start,
          end,
          paginatedCount: result.length
        })
      }
      
      return result
    }
  },

  actions: {
    async fetchNodes(params = {}) {
      this.loading = true
      try {
        const clusterStore = useClusterStore()
        const clusterName = params.cluster_name || this.filters.cluster_name || clusterStore.currentClusterName
        
        // 如果没有集群名称，直接返回空结果
        if (!clusterName) {
          this.nodes = []
          this.updateStats()
          this.updatePaginationTotal()
          return { data: [] }
        }
        
        const queryParams = {
          page: this.pagination.current,
          size: this.pagination.size,
          ...this.filters,
          cluster_name: clusterName,
          ...params
        }
        
        const response = await nodeApi.getNodes(queryParams)
        // 后端返回格式: { code, message, data: [...] } - data直接是节点数组
        this.nodes = response.data.data || []
        this.currentClusterName = clusterName
        this.updateStats()
        // 重新计算分页总数（基于过滤后的结果）
        this.updatePaginationTotal()
        
        return response
      } catch (error) {
        throw error
      } finally {
        this.loading = false
      }
    },

    async getNodeDetail(nodeName) {
      try {
        return await nodeApi.getNodeDetail(nodeName)
      } catch (error) {
        throw error
      }
    },

    async cordonNode(nodeName, reason = '') {
      try {
        const clusterStore = useClusterStore()
        const clusterName = clusterStore.currentClusterName
        if (!clusterName) {
          throw new Error('请先选择集群')
        }
        const response = await nodeApi.cordonNode(nodeName, clusterName, reason)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async uncordonNode(nodeName, reason = '') {
      try {
        const clusterStore = useClusterStore()
        const clusterName = clusterStore.currentClusterName
        if (!clusterName) {
          throw new Error('请先选择集群')
        }
        const response = await nodeApi.uncordonNode(nodeName, clusterName, reason)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async drainNode(nodeName, options = {}) {
      try {
        const clusterStore = useClusterStore()
        const clusterName = clusterStore.currentClusterName
        if (!clusterName) {
          throw new Error('请先选择集群')
        }
        const response = await nodeApi.drainNode(nodeName, clusterName, options)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async batchCordon(nodeNames, reason = '') {
      try {
        const clusterStore = useClusterStore()
        const response = await nodeApi.batchCordon(nodeNames, clusterStore.currentClusterName, reason)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async batchUncordon(nodeNames, reason = '') {
      try {
        const clusterStore = useClusterStore()
        const response = await nodeApi.batchUncordon(nodeNames, clusterStore.currentClusterName, reason)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    setFilters(filters) {
      this.filters = { ...this.filters, ...filters }
      this.pagination.current = 1
      // 重新计算分页总数（基于过滤后的结果）
      this.updatePaginationTotal()
    },

    setPagination(pagination) {
      this.pagination = { ...this.pagination, ...pagination }
    },

    setSelectedNodes(nodes) {
      this.selectedNodes = nodes
    },

    clearSelectedNodes() {
      this.selectedNodes = []
    },


    updateStats() {
      // 统计节点状态
      const stats = {
        total: this.nodes.length,
        ready: 0,
        notReady: 0,
        unknown: 0,
        schedulable: 0,
        limited: 0,
        unschedulable: 0,
        ownership: {}
      }

      // 统计各种状态的节点
      this.nodes.forEach(node => {
        // 状态统计
        switch (node.status) {
          case 'Ready':
            stats.ready++
            break
          case 'NotReady':
          case 'SchedulingDisabled':
            stats.notReady++
            break
          case 'Unknown':
            stats.unknown++
            break
        }

        // 调度状态统计
        const schedulingStatus = getSmartSchedulingStatus(node)
        switch (schedulingStatus) {
          case 'schedulable':
            stats.schedulable++
            break
          case 'limited':
            stats.limited++
            break
          case 'unschedulable':
            stats.unschedulable++
            break
        }

        // 节点归属统计
        const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
        if (userTypeLabel && userTypeLabel.trim() !== '') {
          stats.ownership[userTypeLabel] = (stats.ownership[userTypeLabel] || 0) + 1
        } else {
          stats.ownership['无归属'] = (stats.ownership['无归属'] || 0) + 1
        }
      })

      // 更新状态
      this.nodeStats = stats
    },

    updatePaginationTotal() {
      // 计算过滤后的节点数量（避免使用 getter）
      let filteredCount = this.nodes.length
      
      // 如果有过滤条件，重新计算过滤后的数量
      if (this.filters.name || this.filters.status || this.filters.role || 
          this.filters.schedulable || this.filters.labelKey || this.filters.taintKey || 
          this.filters.nodeOwnership) {
        
        let filtered = this.nodes
        
        if (this.filters.name) {
          filtered = filtered.filter(node => 
            node.name.toLowerCase().includes(this.filters.name.toLowerCase())
          )
        }
        
        if (this.filters.status) {
          filtered = filtered.filter(node => node.status === this.filters.status)
        }
        
        if (this.filters.role) {
          filtered = filtered.filter(node => {
            if (!node.roles || !Array.isArray(node.roles)) {
              return this.filters.role === 'worker'
            }
            
            if (this.filters.role === 'master') {
              return node.roles.some(role => 
                role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
              )
            } else if (this.filters.role === 'worker') {
              return !node.roles.some(role => 
                role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
              )
            }
            
            return false
          })
        }
        
        if (this.filters.schedulable) {
          filtered = filtered.filter(node => {
            const schedulingStatus = getSmartSchedulingStatus(node)
            return schedulingStatus === this.filters.schedulable
          })
        }
        
        if (this.filters.labelKey) {
          filtered = filtered.filter(node => {
            if (!node.labels || !node.labels[this.filters.labelKey]) {
              return false
            }
            if (this.filters.labelValue) {
              return node.labels[this.filters.labelKey] === this.filters.labelValue
            }
            return true
          })
        }
        
        if (this.filters.taintKey) {
          filtered = filtered.filter(node => {
            if (!node.taints || node.taints.length === 0) {
              return false
            }
            return node.taints.some(taint => {
              if (taint.key !== this.filters.taintKey) {
                return false
              }
              if (this.filters.taintValue && taint.value !== this.filters.taintValue) {
                return false
              }
              if (this.filters.taintEffect && taint.effect !== this.filters.taintEffect) {
                return false
              }
              return true
            })
          })
        }
        
        if (this.filters.nodeOwnership) {
          filtered = filtered.filter(node => {
            const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
            
            if (this.filters.nodeOwnership === '无归属') {
              return !userTypeLabel || userTypeLabel.trim() === ''
            }
            
            if (!userTypeLabel || userTypeLabel.trim() === '') {
              return false
            }
            return userTypeLabel === this.filters.nodeOwnership
          })
        }
        
        filteredCount = filtered.length
      }
      
      const oldTotal = this.pagination.total
      this.pagination.total = filteredCount
      
      // 只在数量变化时输出日志，减少控制台噪音
      if (oldTotal !== filteredCount) {
        console.log('更新分页总数:', {
          totalNodes: this.nodes.length,
          filteredNodes: filteredCount,
          paginationTotal: this.pagination.total,
          currentPage: this.pagination.current,
          pageSize: this.pagination.size
        })
      }
      
      // 如果当前页超出范围，重置到第一页
      const maxPage = Math.ceil(filteredCount / this.pagination.size) || 1
      if (this.pagination.current > maxPage) {
        this.pagination.current = 1
      }
    },

    resetFilters() {
      this.filters = {
        name: '',
        status: '',
        role: '',
        cluster_name: '',
        labelKey: '',
        labelValue: '',
        taintKey: '',
        taintValue: '',
        taintEffect: '',
        nodeOwnership: ''
      }
      this.pagination.current = 1
    },

    // 直接设置节点数据（用于其他组件缓存节点数据）
    setNodes(nodes, clusterName = '') {
      this.nodes = nodes || []
      if (clusterName) {
        this.currentClusterName = clusterName
      }
      this.updateStats()
      this.updatePaginationTotal()
    }
  }
})