// 优化后的节点管理 Store
import { defineStore } from 'pinia'
import nodeApi from '@/api/node'
import { useClusterStore } from './cluster'
import { useCacheStore } from './cache'
import { useHistoryStore } from './history'
import { useProgressStore } from './progress'

// 获取智能调度状态的辅助函数
function getSmartSchedulingStatus(node) {
  if (node.schedulable === false) {
    return 'unschedulable'
  }
  
  const hasSchedulingTaints = node.taints && node.taints.length > 0 && 
    node.taints.some(taint => 
      taint.effect === 'NoSchedule' || 
      taint.effect === 'PreferNoSchedule'
    )
  
  if (hasSchedulingTaints) {
    return 'limited'
  }
  
  return 'schedulable'
}

export const useNodeStore = defineStore('node-optimized', {
  state: () => ({
    nodes: [],
    selectedNodes: [],
    currentClusterName: '',
    // 分页改为真正的后端分页
    pagination: {
      current: 1,
      size: 20,
      total: 0,
      totalPages: 0
    },
    // 过滤条件
    filters: {
      name: '',
      status: '',
      role: '',
      schedulable: '',
      labelKey: '',
      labelValue: '',
      taintKey: '',
      taintValue: '',
      taintEffect: '',
      nodeOwnership: ''
    },
    // 排序状态
    sort: {
      prop: 'name',
      order: 'ascending'
    },
    // 加载状态
    loading: {
      fetching: false,
      refreshing: false,
      operating: false
    },
    // 统计信息
    stats: {
      total: 0,
      ready: 0,
      notReady: 0,
      unknown: 0,
      schedulable: 0,
      limited: 0,
      unschedulable: 0,
      ownership: {}
    },
    // 错误状态
    error: null,
    // 最后更新时间
    lastUpdated: null
  }),

  getters: {
    // 是否有任何加载状态
    isLoading: (state) => Object.values(state.loading).some(loading => loading),
    
    // 是否有数据
    hasData: (state) => state.nodes.length > 0,
    
    // 是否有过滤条件
    hasFilters: (state) => {
      return Object.values(state.filters).some(value => value && value.trim() !== '')
    },
    
    // 分页信息
    paginationInfo: (state) => ({
      current: state.pagination.current,
      size: state.pagination.size,
      total: state.pagination.total,
      totalPages: state.pagination.totalPages,
      hasMore: state.pagination.current < state.pagination.totalPages,
      showingStart: (state.pagination.current - 1) * state.pagination.size + 1,
      showingEnd: Math.min(state.pagination.current * state.pagination.size, state.pagination.total)
    }),
    
    // 节点归属选项
    nodeOwnershipOptions: (state) => {
      const ownershipSet = new Set()
      let hasNoOwnership = false
      
      state.nodes.forEach(node => {
        const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
        if (userTypeLabel && userTypeLabel.trim() !== '') {
          ownershipSet.add(userTypeLabel)
        } else {
          hasNoOwnership = true
        }
      })
      
      const options = Array.from(ownershipSet).sort()
      if (hasNoOwnership) {
        options.unshift('无归属')
      }
      
      return options
    },
    
    // 选中节点的统计信息
    selectedNodesStats: (state) => {
      const stats = {
        total: state.selectedNodes.length,
        schedulable: 0,
        unschedulable: 0,
        masters: 0,
        workers: 0
      }
      
      state.selectedNodes.forEach(node => {
        const status = getSmartSchedulingStatus(node)
        if (status === 'schedulable') stats.schedulable++
        else if (status === 'unschedulable') stats.unschedulable++
        
        if (node.roles?.some(role => 
          role === 'master' || role.includes('master') || role.includes('control-plane')
        )) {
          stats.masters++
        } else {
          stats.workers++
        }
      })
      
      return stats
    }
  },

  actions: {
    // 获取节点数据（后端分页）
    async fetchNodes(params = {}, options = {}) {
      const {
        useCache = true,
        forceRefresh = false,
        silent = false
      } = options
      
      if (!silent) {
        this.loading.fetching = true
      }
      this.error = null
      
      try {
        const cacheStore = useCacheStore()
        const clusterStore = useClusterStore()
        const clusterName = params.cluster_name || this.currentClusterName || clusterStore.currentClusterName
        
        if (!clusterName) {
          throw new Error('未选择集群')
        }
        
        // 构建查询参数
        const queryParams = {
          page: params.page || this.pagination.current,
          size: params.size || this.pagination.size,
          sort_by: this.sort.prop,
          sort_order: this.sort.order === 'ascending' ? 'asc' : 'desc',
          cluster_name: clusterName,
          ...this.filters,
          ...params
        }
        
        // 生成缓存键
        const cacheKey = `nodes_${clusterName}_${JSON.stringify(queryParams)}`
        
        // 检查缓存
        if (useCache && !forceRefresh) {
          const cachedData = cacheStore.getApiCache(cacheKey)
          if (cachedData) {
            this.applyFetchResult(cachedData)
            return cachedData
          }
        }
        
        // 调用API
        const response = await nodeApi.getNodes(queryParams)
        const result = {
          nodes: response.data.data || [],
          total: response.data.total || 0,
          totalPages: response.data.total_pages || 0,
          current: response.data.current_page || 1,
          size: response.data.page_size || this.pagination.size
        }
        
        // 更新状态
        this.applyFetchResult(result)
        
        // 缓存结果
        if (useCache) {
          cacheStore.setApiCache(cacheKey, result)
        }
        
        this.lastUpdated = Date.now()
        
        return result
      } catch (error) {
        this.error = error.message || '获取节点数据失败'
        throw error
      } finally {
        if (!silent) {
          this.loading.fetching = false
        }
      }
    },

    // 应用获取结果
    applyFetchResult(result) {
      this.nodes = result.nodes || []
      this.pagination.total = result.total || 0
      this.pagination.totalPages = result.totalPages || 0
      this.pagination.current = result.current || 1
      this.pagination.size = result.size || this.pagination.size
      this.updateStats()
    },

    // 刷新数据
    async refreshNodes(silent = false) {
      this.loading.refreshing = !silent
      try {
        await this.fetchNodes({}, { forceRefresh: true, silent })
      } finally {
        this.loading.refreshing = false
      }
    },

    // 批量操作节点（带进度条）
    async batchOperateNodes(operation, nodeNames, params = {}) {
      const progressStore = useProgressStore()
      const historyStore = useHistoryStore()
      
      // 创建进度跟踪
      const operationId = progressStore.startBatchOperation({
        type: `batch_${operation}`,
        title: this.getOperationTitle(operation, nodeNames.length),
        description: this.getOperationDescription(operation, nodeNames.length),
        items: nodeNames.map(name => ({ name, id: name }))
      })
      
      this.loading.operating = true
      const results = []
      const errors = []
      
      try {
        // 记录原始状态（用于撤销）
        const originalStates = await this.captureNodesState(nodeNames)
        
        // 逐个处理节点
        for (let i = 0; i < nodeNames.length; i++) {
          const nodeName = nodeNames[i]
          
          try {
            progressStore.updateProgress(operationId, {
              currentItem: nodeName
            })
            
            let result
            switch (operation) {
              case 'cordon':
                result = await this.cordonNode(nodeName, params.reason, { silent: true })
                break
              case 'uncordon':
                result = await this.uncordonNode(nodeName, params.reason, { silent: true })
                break
              case 'drain':
                result = await this.drainNode(nodeName, params.reason, { silent: true })
                break
              default:
                throw new Error(`不支持的操作: ${operation}`)
            }
            
            progressStore.addSuccessResult(operationId, { name: nodeName }, result)
            results.push({ node: nodeName, result })
          } catch (error) {
            progressStore.addFailureResult(operationId, { name: nodeName }, error)
            errors.push({ node: nodeName, error })
          }
        }
        
        // 创建撤销操作
        if (results.length > 0) {
          const undoOperation = this.createUndoOperation(operation, results, originalStates)
          const redoOperation = () => this.batchOperateNodes(operation, 
            results.map(r => r.node), params)
          
          historyStore.addOperation(
            historyStore.createNodeOperation(
              operation,
              results.map(r => ({ name: r.node })),
              originalStates,
              redoOperation,
              undoOperation
            )
          )
        }
        
        // 刷新数据
        await this.refreshNodes(true)
        
        // 完成操作
        progressStore.completeOperation(operationId, errors.length === 0 ? 'completed' : 'completed')
        
        return {
          success: results.length,
          failed: errors.length,
          results,
          errors
        }
      } catch (error) {
        progressStore.completeOperation(operationId, 'failed')
        throw error
      } finally {
        this.loading.operating = false
      }
    },

    // 单个节点操作
    async cordonNode(nodeName, reason = '', options = {}) {
      const { silent = false } = options
      
      try {
        if (!silent) this.loading.operating = true
        
        const clusterStore = useClusterStore()
        const response = await nodeApi.cordonNode(nodeName, clusterStore.currentClusterName, reason)
        
        // 刷新数据
        if (!silent) {
          await this.refreshNodes(true)
        }
        
        return response
      } catch (error) {
        throw error
      } finally {
        if (!silent) this.loading.operating = false
      }
    },

    async uncordonNode(nodeName, reason = '', options = {}) {
      const { silent = false } = options
      
      try {
        if (!silent) this.loading.operating = true
        
        const clusterStore = useClusterStore()
        const response = await nodeApi.uncordonNode(nodeName, clusterStore.currentClusterName, reason)
        
        if (!silent) {
          await this.refreshNodes(true)
        }
        
        return response
      } catch (error) {
        throw error
      } finally {
        if (!silent) this.loading.operating = false
      }
    },

    async drainNode(nodeName, reason = '', options = {}) {
      const { silent = false } = options
      
      try {
        if (!silent) this.loading.operating = true
        
        const clusterStore = useClusterStore()
        const response = await nodeApi.drainNode(nodeName, clusterStore.currentClusterName, reason)
        
        if (!silent) {
          await this.refreshNodes(true)
        }
        
        return response
      } catch (error) {
        throw error
      } finally {
        if (!silent) this.loading.operating = false
      }
    },

    // 设置过滤条件
    setFilters(filters) {
      this.filters = { ...this.filters, ...filters }
      this.pagination.current = 1
      // 清除缓存，因为过滤条件改变了
      const cacheStore = useCacheStore()
      cacheStore.clearAllCache()
    },

    // 设置排序
    setSort({ prop, order }) {
      this.sort.prop = prop
      this.sort.order = order
      this.pagination.current = 1
      // 清除缓存
      const cacheStore = useCacheStore()
      cacheStore.clearAllCache()
    },

    // 设置分页
    setPagination(pagination) {
      this.pagination = { ...this.pagination, ...pagination }
    },

    // 选择节点
    setSelectedNodes(nodes) {
      this.selectedNodes = nodes
    },

    clearSelectedNodes() {
      this.selectedNodes = []
    },

    // 重置所有状态
    resetFilters() {
      this.filters = {
        name: '',
        status: '',
        role: '',
        schedulable: '',
        labelKey: '',
        labelValue: '',
        taintKey: '',
        taintValue: '',
        taintEffect: '',
        nodeOwnership: ''
      }
      this.sort = {
        prop: 'name',
        order: 'ascending'
      }
      this.pagination.current = 1
      
      // 清除缓存
      const cacheStore = useCacheStore()
      cacheStore.clearAllCache()
    },

    // 更新统计信息
    updateStats() {
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
        stats[schedulingStatus]++

        // 归属统计
        const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
        if (userTypeLabel && userTypeLabel.trim() !== '') {
          stats.ownership[userTypeLabel] = (stats.ownership[userTypeLabel] || 0) + 1
        } else {
          stats.ownership['无归属'] = (stats.ownership['无归属'] || 0) + 1
        }
      })

      this.stats = stats
    },

    // 工具方法
    getOperationTitle(operation, count) {
      const operations = {
        cordon: '批量禁止调度',
        uncordon: '批量解除调度',
        drain: '批量驱逐节点'
      }
      return `${operations[operation] || operation} (${count}个节点)`
    },

    getOperationDescription(operation, count) {
      const descriptions = {
        cordon: `正在禁止 ${count} 个节点的调度...`,
        uncordon: `正在解除 ${count} 个节点的调度限制...`,
        drain: `正在驱逐 ${count} 个节点上的Pod...`
      }
      return descriptions[operation] || `正在对 ${count} 个节点执行 ${operation} 操作...`
    },

    // 捕获节点状态（用于撤销）
    async captureNodesState(nodeNames) {
      const states = {}
      
      for (const nodeName of nodeNames) {
        const node = this.nodes.find(n => n.name === nodeName)
        if (node) {
          states[nodeName] = {
            schedulable: node.schedulable,
            taints: node.taints ? [...node.taints] : [],
            labels: node.labels ? { ...node.labels } : {}
          }
        }
      }
      
      return states
    },

    // 创建撤销操作
    createUndoOperation(operation, results, originalStates) {
      return async () => {
        // 实现撤销逻辑
        switch (operation) {
          case 'cordon':
            // 撤销禁止调度 = 解除调度
            await this.batchOperateNodes('uncordon', 
              results.map(r => r.node), 
              { reason: '撤销操作' })
            break
          case 'uncordon':
            // 撤销解除调度 = 禁止调度
            await this.batchOperateNodes('cordon', 
              results.map(r => r.node), 
              { reason: '撤销操作' })
            break
          default:
            throw new Error(`无法撤销操作: ${operation}`)
        }
      }
    }
  }
})
