<template>
  <el-dialog
    v-model="visible"
    title="批量操作进度"
    width="500px"
    :close-on-click-modal="true"
    :close-on-press-escape="true"
    :show-close="true"
    @close="handleClose"
  >
    <div class="progress-content">
      <!-- 进度条 -->
      <div class="progress-bar-container">
        <el-progress
          :percentage="progressData.progress"
          :status="progressStatus"
          :stroke-width="8"
          :show-text="true"
        />
        <div class="progress-text">
          {{ progressData.current || 0 }} / {{ progressData.total || 0 }}
        </div>
      </div>

      <!-- 当前操作信息 -->
      <div class="current-operation" v-if="progressData.current_node">
        <el-icon><Loading /></el-icon>
        <span>正在处理节点: {{ progressData.current_node }}</span>
      </div>

      <!-- 状态消息 -->
      <div class="status-message">
        <div :class="messageClass">
          {{ progressData.message || '准备开始批量操作...' }}
        </div>
        <div v-if="progressData.error" class="error-message">
          错误: {{ progressData.error }}
        </div>
      </div>

      <!-- 操作详情 -->
      <div class="operation-details" v-if="progressData.action">
        <div class="detail-item">
          <span class="label">操作类型:</span>
          <span class="value">{{ getActionText(progressData.action) }}</span>
        </div>
        <!-- 任务ID已隐藏，优化用户界面 -->
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button v-if="!isCompleted && !isError" @click="handleCancel" type="danger">取消任务</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { getToken } from '@/utils/auth'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  taskId: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue', 'completed', 'error', 'cancelled'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const progressData = ref({
  task_id: '',
  type: '',
  action: '',
  current: 0,
  total: 0,
  progress: 0,
  current_node: '',
  message: '',
  error: '',
  timestamp: null
})

const isCompleted = ref(false)
const isError = ref(false)
const websocket = ref(null)

// 计算进度状态
const progressStatus = computed(() => {
  if (isError.value) return 'exception'
  if (isCompleted.value) return 'success'
  return undefined
})

// 消息样式
const messageClass = computed(() => ({
  'message': true,
  'success-message': isCompleted.value,
  'error-message': isError.value,
  'processing-message': !isCompleted.value && !isError.value
}))

// 获取操作类型文本
const getActionText = (action) => {
  switch (action) {
    case 'batch_label':
      return '批量标签操作'
    case 'batch_taint':
      return '批量污点操作'
    default:
      return '批量操作'
  }
}

// 建立WebSocket连接
const connectWebSocket = () => {
  if (!props.taskId) {
    console.log('No taskId provided, skipping WebSocket connection')
    return
  }

  // 先关闭已有连接（防止重复连接）
  if (websocket.value) {
    console.log('关闭已有的WebSocket连接以建立新连接')
    websocket.value.close()
    websocket.value = null
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const token = getToken()
  
  const wsUrl = `${protocol}//${host}/api/v1/progress/ws?token=${token}`
  
  console.log('Attempting to connect WebSocket:', wsUrl)
  console.log('TaskId:', props.taskId)
  console.log('Token exists:', !!token)
  
  try {
    websocket.value = new WebSocket(wsUrl)

    websocket.value.onopen = () => {
      console.log('WebSocket连接已建立')
    }

    websocket.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        handleProgressUpdate(data)
      } catch (error) {
        console.error('解析WebSocket消息失败:', error)
      }
    }

    websocket.value.onclose = (event) => {
      console.log('WebSocket连接已关闭', { code: event.code, reason: event.reason, wasClean: event.wasClean })
      
      // 只在非正常关闭且任务未完成时才重连
      if (!event.wasClean && !isCompleted.value && !isError.value && visible.value && props.taskId) {
        console.log('检测到异常关闭，2秒后尝试重连')
        setTimeout(() => {
          // 重连前再次检查状态，避免不必要的重连
          if (!isCompleted.value && !isError.value && visible.value && props.taskId) {
            connectWebSocket()
          }
        }, 2000)
      }
    }

    websocket.value.onerror = (error) => {
      console.error('WebSocket错误:', error)
      console.error('WebSocket URL:', wsUrl)
      console.error('WebSocket状态:', websocket.value?.readyState)
      
      // 不要因为连接错误就立即显示错误消息，因为可能是正常的连接替换
      console.warn('WebSocket连接遇到错误，可能是连接替换导致')
    }
  } catch (error) {
    console.error('创建WebSocket连接失败:', error)
    ElMessage.error('无法建立WebSocket连接')
  }
}

// 处理进度更新
const handleProgressUpdate = (data) => {
  // 只处理当前任务的消息
  if (data.task_id && data.task_id !== props.taskId) {
    return
  }

  switch (data.type) {
    case 'connected':
      console.log('WebSocket连接确认:', data.message)
      break
      
    case 'progress':
      progressData.value = { ...data }
      break
      
    case 'complete':
      progressData.value = { ...data }
      isCompleted.value = true
      ElMessage.success(data.message || '批量操作完成')
      emit('completed', data)
      
      // 3秒后自动关闭弹窗
      setTimeout(() => {
        if (isCompleted.value) {
          emit('update:modelValue', false)
        }
      }, 3000)
      break
      
    case 'error':
      progressData.value = { ...data }
      isError.value = true
      ElMessage.error(data.message || '批量操作失败')
      emit('error', data)
      break
      
    default:
      console.log('收到未知类型的进度消息:', data)
  }
}

// 关闭对话框
const handleClose = () => {
  console.log('关闭进度对话框')
  
  // 如果任务正在进行中，给用户提示
  if (!isCompleted.value && !isError.value && progressData.value.task_id) {
    console.log('任务仍在进行中，但用户选择关闭弹窗')
  }
  
  // 关闭弹窗
  emit('update:modelValue', false)
  
  // 清理资源
  closeWebSocket()
  resetState()
}

// 取消操作
const handleCancel = () => {
  // TODO: 实现取消操作的API调用
  ElMessage.warning('取消功能正在开发中')
  emit('cancelled')
  handleClose()
}

// 关闭WebSocket连接
const closeWebSocket = () => {
  if (websocket.value) {
    console.log('正在关闭WebSocket连接')
    try {
      // 移除事件监听器，防止触发不必要的重连
      websocket.value.onclose = null
      websocket.value.onerror = null
      websocket.value.onmessage = null
      websocket.value.onopen = null
      
      // 正常关闭连接
      websocket.value.close(1000, 'Normal closure')
    } catch (error) {
      console.error('关闭WebSocket时出错:', error)
    }
    websocket.value = null
  }
}

// 重置状态
const resetState = () => {
  progressData.value = {
    task_id: '',
    type: '',
    action: '',
    current: 0,
    total: 0,
    progress: 0,
    current_node: '',
    message: '',
    error: '',
    timestamp: null
  }
  isCompleted.value = false
  isError.value = false
}

// 监听taskId变化
watch(() => props.taskId, (newTaskId, oldTaskId) => {
  console.log('ProgressDialog taskId changed:', { oldTaskId, newTaskId, visible: props.modelValue })
  
  if (newTaskId && props.modelValue) {
    // 新任务开始时，先关闭旧连接，重置状态，再建立新连接
    closeWebSocket()
    resetState()
    // 延迟一点时间确保旧连接完全关闭
    nextTick(() => {
      connectWebSocket()
    })
  } else if (!newTaskId) {
    // 如果taskId为空，关闭连接
    closeWebSocket()
  }
})

// 监听对话框显示状态
watch(() => props.modelValue, (newVal) => {
  console.log('ProgressDialog visibility changed:', newVal, 'taskId:', props.taskId)
  
  if (newVal && props.taskId) {
    // 对话框打开且有taskId时，确保状态清理后再连接
    closeWebSocket()
    resetState()
    nextTick(() => {
      connectWebSocket()
    })
  } else if (!newVal) {
    // 对话框关闭时，立即关闭连接
    closeWebSocket()
  }
})

onMounted(() => {
  console.log('ProgressDialog mounted, visible:', visible.value, 'taskId:', props.taskId)
  if (visible.value && props.taskId) {
    connectWebSocket()
  }
})

onUnmounted(() => {
  closeWebSocket()
})
</script>

<style scoped>
.progress-content {
  padding: 10px 0;
}

.progress-bar-container {
  margin-bottom: 20px;
}

.progress-text {
  text-align: center;
  margin-top: 8px;
  font-size: 14px;
  color: #666;
}

.current-operation {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 15px;
  padding: 10px;
  background-color: #f0f9ff;
  border-radius: 4px;
  border-left: 4px solid #409eff;
}

.current-operation .el-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.status-message {
  margin-bottom: 15px;
}

.message {
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 14px;
  line-height: 1.4;
}

.success-message {
  background-color: #f0f9ff;
  color: #067f23;
  border: 1px solid #b3d8ff;
}

.error-message {
  background-color: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

.processing-message {
  background-color: #f8fafc;
  color: #374151;
  border: 1px solid #e5e7eb;
}

.operation-details {
  padding: 12px;
  background-color: #f9fafb;
  border-radius: 4px;
  border: 1px solid #e5e7eb;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 6px;
}

.detail-item:last-child {
  margin-bottom: 0;
}

.label {
  font-weight: 500;
  color: #6b7280;
}

.value {
  color: #374151;
  font-family: monospace;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
