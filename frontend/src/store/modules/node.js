import { defineStore } from 'pinia'
import nodeApi from '@/api/node'
import { useClusterStore } from './cluster'

// 获取智能调度状态的辅助函数
function getSmartSchedulingStatus(node) {
  // 如果节点被cordon（不可调度）
  if (!node.schedulable) {
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
    nodeStats: {
      total: 0,
      ready: 0,
      notReady: 0,
      unknown: 0,
      schedulable: 0,
      limited: 0,
      unschedulable: 0
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
      
      state.nodes.forEach(node => {
        if (node.labels && node.labels['deeproute.cn/user-type']) {
          ownershipSet.add(node.labels['deeproute.cn/user-type'])
        } else {
          hasNoOwnership = true
        }
      })
      
      const options = Array.from(ownershipSet).sort()
      
      // 如果有节点没有 deeproute.cn/user-type 标签，添加“无归属”选项
      if (hasNoOwnership) {
        options.unshift('无归属') // 添加到数组开头
      }
      
      return options
    },
    filteredNodes: (state) => {
      let result = state.nodes
      
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
          // 如果选择的是“无归属”，过滤出没有 deeproute.cn/user-type 标签的节点
          if (state.filters.nodeOwnership === '无归属') {
            return !node.labels || !node.labels['deeproute.cn/user-type']
          }
          
          // 否则过滤具有匹配标签值的节点
          if (!node.labels || !node.labels['deeproute.cn/user-type']) {
            return false
          }
          return node.labels['deeproute.cn/user-type'] === state.filters.nodeOwnership
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
          this.pagination.total = 0
          this.updateStats()
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
        this.pagination.total = this.nodes.length
        this.updateStats()
        
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

    async cordonNode(nodeName) {
      try {
        const clusterStore = useClusterStore()
        const clusterName = clusterStore.currentClusterName
        if (!clusterName) {
          throw new Error('请先选择集群')
        }
        const response = await nodeApi.cordonNode(nodeName, clusterName)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async uncordonNode(nodeName) {
      try {
        const clusterStore = useClusterStore()
        const clusterName = clusterStore.currentClusterName
        if (!clusterName) {
          throw new Error('请先选择集群')
        }
        const response = await nodeApi.uncordonNode(nodeName, clusterName)
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

    async batchCordon(nodeNames) {
      try {
        const clusterStore = useClusterStore()
        const response = await nodeApi.batchCordon(nodeNames, clusterStore.currentClusterName)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async batchUncordon(nodeNames) {
      try {
        const clusterStore = useClusterStore()
        const response = await nodeApi.batchUncordon(nodeNames, clusterStore.currentClusterName)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    setFilters(filters) {
      this.filters = { ...this.filters, ...filters }
      this.pagination.current = 1
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
      this.nodeStats.total = this.nodes.length
      this.nodeStats.ready = this.readyNodes.length
      this.nodeStats.notReady = this.notReadyNodes.length
      this.nodeStats.unknown = this.unknownNodes.length
      this.nodeStats.schedulable = this.schedulableNodes.length
      this.nodeStats.limited = this.limitedNodes.length
      this.nodeStats.unschedulable = this.unschedulableNodes.length
      
      // 调试日志，帮助诊断统计不匹配问题
      if (this.nodeStats.ready + this.nodeStats.notReady + this.nodeStats.unknown !== this.nodeStats.total) {
        console.warn('节点统计不匹配:', {
          total: this.nodeStats.total,
          ready: this.nodeStats.ready,
          notReady: this.nodeStats.notReady,
          unknown: this.nodeStats.unknown,
          sum: this.nodeStats.ready + this.nodeStats.notReady + this.nodeStats.unknown,
          nodeStatuses: this.nodes.map(node => ({ name: node.name, status: node.status }))
        })
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
    }
  }
})