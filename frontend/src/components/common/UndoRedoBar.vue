<template>
  <transition name="undo-bar" appear>
    <div v-if="visible && historyStore.canUndo" class="undo-redo-bar">
      <div class="undo-content">
        <div class="operation-info">
          <el-icon class="operation-icon"><Operation /></el-icon>
          <span class="operation-text">{{ historyStore.currentOperation?.description || '已执行操作' }}</span>
        </div>
        
        <div class="action-buttons">
          <el-button
            type="primary"
            size="small"
            plain
            @click="handleUndo"
            :loading="undoing"
            :disabled="!historyStore.canUndo || undoing || redoing"
          >
            <el-icon><Back /></el-icon>
            撤销
          </el-button>
          
          <el-button
            v-if="historyStore.canRedo"
            type="success"
            size="small"
            plain
            @click="handleRedo"
            :loading="redoing"
            :disabled="!historyStore.canRedo || undoing || redoing"
          >
            <el-icon><Right /></el-icon>
            重做
          </el-button>
          
          <el-button
            type="text"
            size="small"
            @click="handleDismiss"
            class="dismiss-btn"
          >
            <el-icon><Close /></el-icon>
          </el-button>
        </div>
      </div>
      
      <!-- 自动消失倒计时 -->
      <div v-if="autoHide" class="countdown-bar">
        <div 
          class="countdown-progress"
          :style="{ width: countdownPercent + '%' }"
        />
      </div>
    </div>
  </transition>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useHistoryStore } from '@/store/modules/history'
import {
  Operation,
  Back,
  Right,
  Close
} from '@element-plus/icons-vue'

const props = defineProps({
  // 是否自动隐藏
  autoHide: {
    type: Boolean,
    default: true
  },
  // 自动隐藏延迟（毫秒）
  autoHideDelay: {
    type: Number,
    default: 8000
  },
  // 显示位置
  position: {
    type: String,
    default: 'bottom', // top, bottom
    validator: (value) => ['top', 'bottom'].includes(value)
  }
})

const emit = defineEmits(['undo', 'redo', 'dismiss'])

const historyStore = useHistoryStore()

// 响应式数据
const visible = ref(false)
const undoing = ref(false)
const redoing = ref(false)
const countdownPercent = ref(100)
const hideTimer = ref(null)
const countdownTimer = ref(null)

// 计算属性
const shouldShow = computed(() => {
  return historyStore.canUndo && !historyStore.isUndoRedoInProgress
})

// 监听是否应该显示
watch(shouldShow, (show) => {
  if (show) {
    showBar()
  } else {
    hideBar()
  }
})

// 显示撤销栏
const showBar = () => {
  visible.value = true
  
  if (props.autoHide) {
    startCountdown()
  }
}

// 隐藏撤销栏
const hideBar = () => {
  visible.value = false
  clearTimers()
}

// 开始倒计时
const startCountdown = () => {
  clearTimers()
  
  countdownPercent.value = 100
  
  const interval = 50 // 更新间隔
  const totalSteps = props.autoHideDelay / interval
  let currentStep = 0
  
  countdownTimer.value = setInterval(() => {
    currentStep++
    countdownPercent.value = Math.max(0, 100 - (currentStep / totalSteps * 100))
    
    if (currentStep >= totalSteps) {
      clearInterval(countdownTimer.value)
      hideBar()
    }
  }, interval)
}

// 清理定时器
const clearTimers = () => {
  if (hideTimer.value) {
    clearTimeout(hideTimer.value)
    hideTimer.value = null
  }
  
  if (countdownTimer.value) {
    clearInterval(countdownTimer.value)
    countdownTimer.value = null
  }
}

// 处理撤销
const handleUndo = async () => {
  if (undoing.value || !historyStore.canUndo) {
    return
  }
  
  undoing.value = true
  
  try {
    const success = await historyStore.undo()
    
    if (success) {
      ElMessage.success('撤销成功')
      emit('undo')
    } else {
      ElMessage.error('撤销失败')
    }
  } catch (error) {
    console.error('撤销操作失败:', error)
    ElMessage.error('撤销操作失败')
  } finally {
    undoing.value = false
  }
}

// 处理重做
const handleRedo = async () => {
  if (redoing.value || !historyStore.canRedo) {
    return
  }
  
  redoing.value = true
  
  try {
    const success = await historyStore.redo()
    
    if (success) {
      ElMessage.success('重做成功')
      emit('redo')
    } else {
      ElMessage.error('重做失败')
    }
  } catch (error) {
    console.error('重做操作失败:', error)
    ElMessage.error('重做操作失败')
  } finally {
    redoing.value = false
  }
}

// 处理手动关闭
const handleDismiss = () => {
  emit('dismiss')
  hideBar()
}

// 鼠标悬停暂停倒计时
const handleMouseEnter = () => {
  if (props.autoHide && countdownTimer.value) {
    clearInterval(countdownTimer.value)
  }
}

// 鼠标离开继续倒计时
const handleMouseLeave = () => {
  if (props.autoHide && visible.value) {
    startCountdown()
  }
}

// 组件挂载时检查是否需要显示
onMounted(() => {
  if (shouldShow.value) {
    showBar()
  }
})

// 组件卸载时清理定时器
onUnmounted(() => {
  clearTimers()
})

// 键盘快捷键支持
const handleKeydown = (event) => {
  if ((event.ctrlKey || event.metaKey) && event.key === 'z' && !event.shiftKey) {
    event.preventDefault()
    handleUndo()
  } else if (((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'Z') ||
             ((event.ctrlKey || event.metaKey) && event.key === 'y')) {
    event.preventDefault()
    handleRedo()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.undo-redo-bar {
  position: fixed;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2000;
  min-width: 300px;
  max-width: 600px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
  border: 1px solid #e8e8e8;
  overflow: hidden;
  transition: all 0.3s ease;
}

.undo-redo-bar.top {
  top: 20px;
}

.undo-redo-bar.bottom {
  bottom: 80px;
}

.undo-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  gap: 16px;
}

.operation-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.operation-icon {
  color: #1890ff;
  font-size: 16px;
  flex-shrink: 0;
}

.operation-text {
  color: #333;
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.dismiss-btn {
  padding: 4px 8px !important;
  min-width: 32px;
  color: #999 !important;
}

.dismiss-btn:hover {
  color: #666 !important;
  background: #f5f5f5 !important;
}

.countdown-bar {
  height: 3px;
  background: #f0f0f0;
  position: relative;
  overflow: hidden;
}

.countdown-progress {
  height: 100%;
  background: linear-gradient(90deg, #1890ff, #40a9ff);
  transition: width 50ms linear;
  border-radius: 0 2px 2px 0;
}

/* 过渡动画 */
.undo-bar-enter-active {
  transition: all 0.3s ease;
}

.undo-bar-leave-active {
  transition: all 0.25s ease;
}

.undo-bar-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(20px);
}

.undo-bar-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-20px);
}

/* 悬停效果 */
.undo-redo-bar:hover {
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
  transform: translateX(-50%) translateY(-2px);
}

.undo-redo-bar:hover .countdown-progress {
  animation-play-state: paused;
}

/* 不同位置的样式 */
.undo-redo-bar[data-position="top"] {
  top: 20px;
}

.undo-redo-bar[data-position="bottom"] {
  bottom: 80px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .undo-redo-bar {
    left: 16px;
    right: 16px;
    transform: none;
    min-width: auto;
    max-width: none;
  }
  
  .undo-redo-bar[data-position="top"] {
    top: 16px;
  }
  
  .undo-redo-bar[data-position="bottom"] {
    bottom: 16px;
  }
  
  .undo-content {
    padding: 10px 12px;
    gap: 12px;
  }
  
  .operation-text {
    font-size: 13px;
  }
  
  .action-buttons .el-button {
    padding: 6px 12px;
    font-size: 12px;
  }
  
  .dismiss-btn {
    padding: 3px 6px !important;
    min-width: 28px;
  }
}

/* 深色主题适配 */
@media (prefers-color-scheme: dark) {
  .undo-redo-bar {
    background: #1f1f1f;
    border-color: #434343;
    color: #fff;
  }
  
  .operation-text {
    color: #fff;
  }
  
  .countdown-bar {
    background: #333;
  }
}

/* 高对比度支持 */
@media (prefers-contrast: high) {
  .undo-redo-bar {
    border: 2px solid #000;
  }
  
  .operation-icon {
    color: #0066cc;
  }
  
  .countdown-progress {
    background: #0066cc;
  }
}

/* 减少动画支持 */
@media (prefers-reduced-motion: reduce) {
  .undo-redo-bar,
  .undo-bar-enter-active,
  .undo-bar-leave-active,
  .countdown-progress {
    transition: none;
  }
}
</style>
