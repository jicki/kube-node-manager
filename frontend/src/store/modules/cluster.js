import { defineStore } from 'pinia'
import clusterApi from '@/api/cluster'

export const useClusterStore = defineStore('cluster', {
  state: () => ({
    clusters: [],
    currentCluster: null,
    clusterStats: {
      total: 0,
      active: 0,
      inactive: 0
    },
    loading: false
  }),

  getters: {
    activeClusters: (state) => state.clusters.filter(cluster => cluster.status === 'active'),
    inactiveClusters: (state) => state.clusters.filter(cluster => cluster.status !== 'active'),
    currentClusterName: (state) => state.currentCluster?.name || null,
    hasCluster: (state) => state.clusters.length > 0,
    hasCurrentCluster: (state) => !!state.currentCluster
  },

  actions: {
    async fetchClusters() {
      this.loading = true
      try {
        const response = await clusterApi.getClusters()
        // 后端返回格式: { code, message, data: [...] }
        this.clusters = response.data.data || []
        this.updateStats()
        return response
      } catch (error) {
        throw error
      } finally {
        this.loading = false
      }
    },

    async addCluster(clusterData) {
      try {
        const response = await clusterApi.addCluster(clusterData)
        // 后端返回格式: { code, message, data: {...} }
        if (response.data.data) {
          // 确保clusters是数组
          this.ensureClustersArray()
          this.clusters.push(response.data.data)
          this.updateStats()
        }
        return response
      } catch (error) {
        throw error
      }
    },

    async updateCluster(id, clusterData) {
      try {
        const response = await clusterApi.updateCluster(id, clusterData)
        // 后端返回格式: { code, message, data: {...} }
        const index = this.clusters.findIndex(cluster => cluster.id === id)
        if (index !== -1 && response.data.data) {
          this.clusters[index] = response.data.data
        }
        this.updateStats()
        return response
      } catch (error) {
        throw error
      }
    },

    async deleteCluster(id) {
      try {
        const response = await clusterApi.deleteCluster(id)
        this.clusters = this.clusters.filter(cluster => cluster.id !== id)
        this.updateStats()
        return response
      } catch (error) {
        throw error
      }
    },

    async testClusterConnection(id) {
      try {
        return await clusterApi.testConnection(id)
      } catch (error) {
        throw error
      }
    },

    setCurrentCluster(cluster) {
      this.currentCluster = cluster
      // 保存到本地存储
      localStorage.setItem('currentCluster', JSON.stringify(cluster))
    },

    loadCurrentCluster() {
      const saved = localStorage.getItem('currentCluster')
      if (saved) {
        this.currentCluster = JSON.parse(saved)
      }
    },

    updateStats() {
      // 确保clusters是数组
      if (!Array.isArray(this.clusters)) {
        this.clusters = []
      }
      this.clusterStats.total = this.clusters.length
      this.clusterStats.active = this.activeClusters.length
      this.clusterStats.inactive = this.inactiveClusters.length
    },

    // 确保clusters始终是数组的辅助方法
    ensureClustersArray() {
      if (!Array.isArray(this.clusters)) {
        this.clusters = []
      }
    }
  }
})