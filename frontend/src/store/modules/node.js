import { defineStore } from 'pinia'
import nodeApi from '@/api/node'
import { useClusterStore } from './cluster'

export const useNodeStore = defineStore('node', {
  state: () => ({
    nodes: [],
    selectedNodes: [],
    nodeStats: {
      total: 0,
      ready: 0,
      notReady: 0,
      unknown: 0
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
      cluster_name: ''
    },
    loading: false
  }),

  getters: {
    readyNodes: (state) => state.nodes.filter(node => node.status === 'Ready'),
    notReadyNodes: (state) => state.nodes.filter(node => node.status === 'NotReady'),
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
        const response = await nodeApi.cordonNode(nodeName)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async uncordonNode(nodeName) {
      try {
        const response = await nodeApi.uncordonNode(nodeName)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async drainNode(nodeName, options = {}) {
      try {
        const response = await nodeApi.drainNode(nodeName, options)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async batchCordon(nodeNames) {
      try {
        const response = await nodeApi.batchCordon(nodeNames)
        await this.fetchNodes()
        return response
      } catch (error) {
        throw error
      }
    },

    async batchUncordon(nodeNames) {
      try {
        const response = await nodeApi.batchUncordon(nodeNames)
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
    },

    resetFilters() {
      this.filters = {
        name: '',
        status: '',
        role: '',
        cluster_name: ''
      }
      this.pagination.current = 1
    }
  }
})