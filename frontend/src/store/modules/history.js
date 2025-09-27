// 操作历史管理 Store - 用于撤销/重做功能
import { defineStore } from 'pinia'

export const useHistoryStore = defineStore('history', {
  state: () => ({
    // 操作历史栈
    history: [],
    // 当前位置指针
    currentIndex: -1,
    // 最大历史记录数
    maxHistorySize: 50,
    // 是否正在执行撤销/重做操作
    isUndoRedoInProgress: false
  }),

  getters: {
    // 是否可以撤销
    canUndo: (state) => state.currentIndex >= 0 && !state.isUndoRedoInProgress,
    
    // 是否可以重做
    canRedo: (state) => state.currentIndex < state.history.length - 1 && !state.isUndoRedoInProgress,
    
    // 获取当前操作
    currentOperation: (state) => {
      return state.currentIndex >= 0 ? state.history[state.currentIndex] : null
    },
    
    // 获取下一个操作（用于重做）
    nextOperation: (state) => {
      return state.currentIndex < state.history.length - 1 ? state.history[state.currentIndex + 1] : null
    }
  },

  actions: {
    // 添加操作到历史记录
    addOperation(operation) {
      if (this.isUndoRedoInProgress) {
        return // 在执行撤销/重做时不记录操作
      }

      // 验证操作对象
      if (!this.validateOperation(operation)) {
        console.warn('Invalid operation object:', operation)
        return
      }

      // 如果当前不在历史记录的最后位置，删除当前位置之后的所有记录
      if (this.currentIndex < this.history.length - 1) {
        this.history.splice(this.currentIndex + 1)
      }

      // 添加新操作
      this.history.push({
        ...operation,
        timestamp: Date.now(),
        id: this.generateOperationId()
      })

      // 限制历史记录大小
      if (this.history.length > this.maxHistorySize) {
        this.history.shift()
      } else {
        this.currentIndex++
      }
    },

    // 撤销操作
    async undo() {
      if (!this.canUndo) {
        return false
      }

      this.isUndoRedoInProgress = true
      
      try {
        const operation = this.history[this.currentIndex]
        
        if (operation && operation.undoAction) {
          await operation.undoAction()
          this.currentIndex--
          return true
        }
      } catch (error) {
        console.error('撤销操作失败:', error)
        return false
      } finally {
        this.isUndoRedoInProgress = false
      }
    },

    // 重做操作
    async redo() {
      if (!this.canRedo) {
        return false
      }

      this.isUndoRedoInProgress = true
      
      try {
        const operation = this.history[this.currentIndex + 1]
        
        if (operation && operation.redoAction) {
          await operation.redoAction()
          this.currentIndex++
          return true
        }
      } catch (error) {
        console.error('重做操作失败:', error)
        return false
      } finally {
        this.isUndoRedoInProgress = false
      }
    },

    // 清除历史记录
    clearHistory() {
      this.history = []
      this.currentIndex = -1
    },

    // 验证操作对象
    validateOperation(operation) {
      return operation && 
             typeof operation.type === 'string' &&
             typeof operation.description === 'string' &&
             typeof operation.undoAction === 'function'
    },

    // 生成操作ID
    generateOperationId() {
      return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
    },

    // 创建节点操作记录
    createNodeOperation(type, nodes, originalData, redoAction, undoAction) {
      return {
        type: `node.${type}`,
        description: this.getNodeOperationDescription(type, nodes),
        nodes: Array.isArray(nodes) ? nodes : [nodes],
        originalData,
        redoAction,
        undoAction
      }
    },

    // 获取节点操作描述
    getNodeOperationDescription(type, nodes) {
      const nodeCount = Array.isArray(nodes) ? nodes.length : 1
      const nodeText = nodeCount === 1 ? nodes[0]?.name || '节点' : `${nodeCount}个节点`
      
      const descriptions = {
        'cordon': `禁止调度 ${nodeText}`,
        'uncordon': `解除调度 ${nodeText}`,
        'drain': `驱逐 ${nodeText}`,
        'label.add': `为 ${nodeText} 添加标签`,
        'label.remove': `从 ${nodeText} 移除标签`,
        'taint.add': `为 ${nodeText} 添加污点`,
        'taint.remove': `从 ${nodeText} 移除污点`
      }
      
      return descriptions[type] || `对 ${nodeText} 执行 ${type} 操作`
    }
  }
})
