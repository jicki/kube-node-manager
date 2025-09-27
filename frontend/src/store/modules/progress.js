// 批量操作进度管理 Store
import { defineStore } from 'pinia'

export const useProgressStore = defineStore('progress', {
  state: () => ({
    // 活跃的批量操作
    activeOperations: new Map(),
    // 操作历史
    operationHistory: [],
    // 最大历史记录数
    maxHistorySize: 100
  }),

  getters: {
    // 获取所有活跃的操作
    getActiveOperations: (state) => Array.from(state.activeOperations.values()),
    
    // 获取特定操作的进度
    getOperationProgress: (state) => (operationId) => {
      return state.activeOperations.get(operationId) || null
    },
    
    // 是否有活跃的操作
    hasActiveOperations: (state) => state.activeOperations.size > 0,
    
    // 获取最近的操作历史
    getRecentHistory: (state) => state.operationHistory.slice(-20)
  },

  actions: {
    // 开始批量操作
    startBatchOperation(config) {
      const operationId = this.generateOperationId()
      const operation = {
        id: operationId,
        type: config.type,
        title: config.title || '批量操作',
        description: config.description || '',
        items: config.items || [],
        totalCount: config.items?.length || 0,
        completedCount: 0,
        failedCount: 0,
        successCount: 0,
        status: 'running', // running, completed, failed, cancelled
        startTime: Date.now(),
        endTime: null,
        results: [],
        errors: [],
        currentItem: null,
        estimatedTimeRemaining: null,
        speed: null // 操作速度 (items/second)
      }

      this.activeOperations.set(operationId, operation)
      return operationId
    },

    // 更新操作进度
    updateProgress(operationId, updates) {
      const operation = this.activeOperations.get(operationId)
      if (!operation) {
        return false
      }

      // 更新操作状态
      Object.assign(operation, updates)

      // 计算进度百分比
      if (operation.totalCount > 0) {
        operation.progress = Math.round((operation.completedCount / operation.totalCount) * 100)
      }

      // 计算预计剩余时间
      if (operation.completedCount > 0 && operation.status === 'running') {
        const elapsedTime = Date.now() - operation.startTime
        const speed = operation.completedCount / (elapsedTime / 1000)
        operation.speed = speed
        
        const remainingItems = operation.totalCount - operation.completedCount
        operation.estimatedTimeRemaining = remainingItems / speed * 1000
      }

      return true
    },

    // 添加成功结果
    addSuccessResult(operationId, item, result) {
      const operation = this.activeOperations.get(operationId)
      if (!operation) {
        return false
      }

      operation.results.push({
        item,
        result,
        status: 'success',
        timestamp: Date.now()
      })
      
      operation.successCount++
      operation.completedCount++
      
      this.updateProgress(operationId, {})
      return true
    },

    // 添加失败结果
    addFailureResult(operationId, item, error) {
      const operation = this.activeOperations.get(operationId)
      if (!operation) {
        return false
      }

      operation.results.push({
        item,
        error: error?.message || error || '操作失败',
        status: 'failed',
        timestamp: Date.now()
      })
      
      operation.errors.push({
        item,
        error: error?.message || error || '操作失败',
        timestamp: Date.now()
      })
      
      operation.failedCount++
      operation.completedCount++
      
      this.updateProgress(operationId, {})
      return true
    },

    // 完成操作
    completeOperation(operationId, status = 'completed') {
      const operation = this.activeOperations.get(operationId)
      if (!operation) {
        return false
      }

      operation.status = status
      operation.endTime = Date.now()
      operation.duration = operation.endTime - operation.startTime

      // 移到历史记录
      this.addToHistory(operation)
      
      // 从活跃操作中移除
      this.activeOperations.delete(operationId)
      
      return true
    },

    // 取消操作
    cancelOperation(operationId) {
      return this.completeOperation(operationId, 'cancelled')
    },

    // 添加到历史记录
    addToHistory(operation) {
      this.operationHistory.push({
        ...operation,
        // 只保留必要的信息，减少内存占用
        items: operation.items.length,
        results: operation.results.length,
        errors: operation.errors.length
      })

      // 限制历史记录大小
      if (this.operationHistory.length > this.maxHistorySize) {
        this.operationHistory.shift()
      }
    },

    // 清除已完成的操作
    clearCompletedOperations() {
      const activeOps = Array.from(this.activeOperations.entries())
      activeOps.forEach(([id, operation]) => {
        if (operation.status === 'completed' || operation.status === 'failed') {
          this.activeOperations.delete(id)
        }
      })
    },

    // 清除所有操作
    clearAllOperations() {
      this.activeOperations.clear()
    },

    // 生成操作ID
    generateOperationId() {
      return `batch_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
    },

    // 格式化时间
    formatDuration(milliseconds) {
      if (!milliseconds) return '0秒'
      
      const seconds = Math.floor(milliseconds / 1000)
      const minutes = Math.floor(seconds / 60)
      const hours = Math.floor(minutes / 60)
      
      if (hours > 0) {
        return `${hours}小时${minutes % 60}分${seconds % 60}秒`
      } else if (minutes > 0) {
        return `${minutes}分${seconds % 60}秒`
      } else {
        return `${seconds}秒`
      }
    },

    // 格式化预计剩余时间
    formatEstimatedTime(milliseconds) {
      if (!milliseconds || milliseconds < 1000) return '即将完成'
      return this.formatDuration(milliseconds)
    }
  }
})
