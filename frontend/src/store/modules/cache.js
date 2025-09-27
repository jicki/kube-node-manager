// 缓存管理 Store
import { defineStore } from 'pinia'

export const useCacheStore = defineStore('cache', {
  state: () => ({
    // 节点数据缓存，按集群名称分组
    nodeCache: new Map(),
    // 缓存时间戳，用于判断缓存是否过期
    nodeCacheTime: new Map(),
    // 缓存过期时间（毫秒）
    cacheExpiry: 5 * 60 * 1000, // 5分钟
    // 最大缓存大小
    maxCacheSize: 10,
    // API请求缓存
    apiCache: new Map(),
    apiCacheTime: new Map()
  }),

  getters: {
    // 获取节点缓存
    getNodeCache: (state) => (clusterName) => {
      const cacheKey = clusterName
      const cacheTime = state.nodeCacheTime.get(cacheKey)
      
      // 检查缓存是否过期
      if (!cacheTime || Date.now() - cacheTime > state.cacheExpiry) {
        state.nodeCache.delete(cacheKey)
        state.nodeCacheTime.delete(cacheKey)
        return null
      }
      
      return state.nodeCache.get(cacheKey) || null
    },

    // 获取API缓存
    getApiCache: (state) => (cacheKey) => {
      const cacheTime = state.apiCacheTime.get(cacheKey)
      
      if (!cacheTime || Date.now() - cacheTime > state.cacheExpiry) {
        state.apiCache.delete(cacheKey)
        state.apiCacheTime.delete(cacheKey)
        return null
      }
      
      return state.apiCache.get(cacheKey) || null
    }
  },

  actions: {
    // 设置节点缓存
    setNodeCache(clusterName, data) {
      const cacheKey = clusterName
      
      // 清理过期缓存
      this.cleanExpiredCache()
      
      // 如果缓存超过最大大小，删除最旧的缓存
      if (this.nodeCache.size >= this.maxCacheSize) {
        const oldestKey = this.nodeCache.keys().next().value
        this.nodeCache.delete(oldestKey)
        this.nodeCacheTime.delete(oldestKey)
      }
      
      this.nodeCache.set(cacheKey, data)
      this.nodeCacheTime.set(cacheKey, Date.now())
    },

    // 设置API缓存
    setApiCache(cacheKey, data, customExpiry = null) {
      this.cleanExpiredApiCache()
      
      this.apiCache.set(cacheKey, data)
      this.apiCacheTime.set(cacheKey, Date.now())
    },

    // 清理过期的节点缓存
    cleanExpiredCache() {
      const now = Date.now()
      for (const [key, time] of this.nodeCacheTime.entries()) {
        if (now - time > this.cacheExpiry) {
          this.nodeCache.delete(key)
          this.nodeCacheTime.delete(key)
        }
      }
    },

    // 清理过期的API缓存
    cleanExpiredApiCache() {
      const now = Date.now()
      for (const [key, time] of this.apiCacheTime.entries()) {
        if (now - time > this.cacheExpiry) {
          this.apiCache.delete(key)
          this.apiCacheTime.delete(key)
        }
      }
    },

    // 清除所有缓存
    clearAllCache() {
      this.nodeCache.clear()
      this.nodeCacheTime.clear()
      this.apiCache.clear()
      this.apiCacheTime.clear()
    },

    // 清除特定集群的缓存
    clearClusterCache(clusterName) {
      this.nodeCache.delete(clusterName)
      this.nodeCacheTime.delete(clusterName)
    }
  }
})
