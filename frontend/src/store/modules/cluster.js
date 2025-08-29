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
        this.clusters = response.data
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
        this.clusters.push(response.data)
        this.updateStats()
        return response
      } catch (error) {
        throw error
      }
    },

    async updateCluster(id, clusterData) {
      try {
        const response = await clusterApi.updateCluster(id, clusterData)
        const index = this.clusters.findIndex(cluster => cluster.id === id)
        if (index !== -1) {
          this.clusters[index] = response.data
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
      this.clusterStats.total = this.clusters.length
      this.clusterStats.active = this.activeClusters.length
      this.clusterStats.inactive = this.inactiveClusters.length
    }
  }
})