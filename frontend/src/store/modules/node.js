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
    masterNodes: (state) => state.nodes.filter(node => node.roles.includes('master')),
    workerNodes: (state) => state.nodes.filter(node => node.roles.includes('worker')),
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
        result = result.filter(node => node.roles.includes(state.filters.role))
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
        this.nodes = response.data.items || response.data
        this.pagination.total = response.data.total || this.nodes.length
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