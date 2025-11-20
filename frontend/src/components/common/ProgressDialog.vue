<template>
  <el-dialog
    v-model="visible"
    title="批量操作进度"
    width="800px"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
    @close="handleClose"
  >
    <div class="progress-content">
      <!-- 顶部进度条区域 -->
      <div class="progress-bar-container">
        <el-progress
          :percentage="roundedProgress"
          :status="progressStatus"
          :stroke-width="10"
          :show-text="true"
        />
        <div class="progress-text">
          {{ progressData.current || 0 }} / {{ progressData.total || 0 }} 个节点已完成
        </div>
      </div>

      <!-- 三栏节点状态展示 -->
      <el-row :gutter="16" class="nodes-status-row">
        <!-- 处理中 -->
        <el-col :span="8">
          <el-card class="status-card processing-card" shadow="hover">
            <template #header>
              <div class="card-header">
                <el-icon class="header-icon processing-icon"><Loading /></el-icon>
                <span class="header-title">处理中 ({{ processingNodes.length }})</span>
              </div>
            </template>
            <div class="node-list">
              <div v-if="processingNodes.length === 0" class="empty-text">
                暂无处理中的节点
              </div>
              <transition-group name="node-list">
                <div
                  v-for="node in processingNodes"
                  :key="node"
                  class="node-item processing-item"
                >
                  <el-icon class="node-icon rotating"><Loading /></el-icon>
                  <span class="node-name">{{ node }}</span>
                </div>
              </transition-group>
            </div>
          </el-card>
        </el-col>

        <!-- 已成功 -->
        <el-col :span="8">
          <el-card class="status-card success-card" shadow="hover">
            <template #header>
              <div class="card-header">
                <el-icon class="header-icon success-icon"><CircleCheck /></el-icon>
                <span class="header-title">已成功 ({{ successNodes.length }})</span>
              </div>
            </template>
            <div class="node-list">
              <div v-if="successNodes.length === 0" class="empty-text">
                暂无成功节点
              </div>
              <transition-group name="node-list">
                <div
                  v-for="node in successNodes"
                  :key="node"
                  class="node-item success-item"
                >
                  <el-icon class="node-icon"><CircleCheck /></el-icon>
                  <span class="node-name">{{ node }}</span>
                </div>
              </transition-group>
            </div>
          </el-card>
        </el-col>

        <!-- 已失败 -->
        <el-col :span="8">
          <el-card class="status-card failed-card" shadow="hover">
            <template #header>
              <div class="card-header">
                <el-icon class="header-icon failed-icon"><CircleClose /></el-icon>
                <span class="header-title">已失败 ({{ failedNodes.length }})</span>
              </div>
            </template>
            <div class="node-list">
              <div v-if="failedNodes.length === 0" class="empty-text">
                暂无失败节点
              </div>
              <transition-group name="node-list">
                <div
                  v-for="(nodeError, index) in failedNodes"
                  :key="nodeError.node_name || index"
                  class="node-item failed-item"
                >
                  <el-icon class="node-icon"><CircleClose /></el-icon>
                  <el-popover
                    placement="top"
                    :width="300"
                    trigger="hover"
                  >
                    <template #reference>
                      <span class="node-name error-trigger">{{ nodeError.node_name }}</span>
                    </template>
                    <div class="error-detail">
                      <div class="error-title">错误详情:</div>
                      <div class="error-message">{{ nodeError.error }}</div>
                    </div>
                  </el-popover>
                </div>
              </transition-group>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 底部统计信息 -->
      <div class="summary-info" v-if="isCompleted || isError">
        <el-alert
          :title="summaryTitle"
          :type="isError ? 'warning' : 'success'"
          :closable="false"
          show-icon
        >
          <template #default>
            <div class="summary-detail">
              总计 {{ progressData.total }} 个节点：
              <span class="success-count">成功 {{ successNodes.length }} 个</span>
              <span v-if="failedNodes.length > 0" class="failed-count">，失败 {{ failedNodes.length }} 个</span>
            </div>
          </template>
        </el-alert>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button
          v-if="!isCompleted && !isError"
          @click="handleCancel"
          :disabled="true"
        >
          取消任务
        </el-button>
        <el-button
          type="primary"
          @click="handleClose"
          :disabled="!isCompleted && !isError"
        >
          关闭
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading, CircleCheck, CircleClose } from '@element-plus/icons-vue'
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
  success_nodes: [],
  failed_nodes: [],
  message: '',
  error: '',
  timestamp: null
})

const isCompleted = ref(false)
const isError = ref(false)
const websocket = ref(null)
const completionTimer = ref(null)
const reconnectCount = ref(0)
const maxReconnectAttempts = 5

// 计算进度百分比
const roundedProgress = computed(() => {
  return Math.round(progressData.value.progress || 0)
})

// 计算进度状态
const progressStatus = computed(() => {
  if (isError.value) return 'exception'
  if (isCompleted.value) return 'success'
  return undefined
})

// 成功节点列表
const successNodes = computed(() => {
  const nodes = progressData.value.success_nodes || []
  console.log('✅ Success nodes:', nodes, 'Type:', typeof nodes, 'IsArray:', Array.isArray(nodes))
  
  if (!Array.isArray(nodes)) {
    console.error('❌ success_nodes is not an array:', nodes)
    return []
  }
  return nodes
})

// 失败节点列表
const failedNodes = computed(() => {
  const nodes = progressData.value.failed_nodes || []
  console.log('❌ Failed nodes:', nodes, 'Type:', typeof nodes, 'IsArray:', Array.isArray(nodes))
  
  if (!Array.isArray(nodes)) {
    console.error('❌ failed_nodes is not an array:', nodes)
    return []
  }
  
  // 确保返回正确格式的对象数组
  return nodes.map(node => {
    if (typeof node === 'string') {
      return { node_name: node, error: '未知错误' }
    }
    return node
  })
})

// 处理中节点列表
const processingNodes = computed(() => {
  const current = progressData.value.current_node
  if (!current || isCompleted.value || isError.value) {
    return []
  }
  // 如果当前节点不在成功或失败列表中，则认为正在处理
  if (!successNodes.value.includes(current) && 
      !failedNodes.value.some(n => n.node_name === current)) {
    return [current]
  }
  return []
})

// 汇总标题
const summaryTitle = computed(() => {
  if (isError.value && failedNodes.value.length > 0) {
    return `批量操作完成（部分失败）`
  }
  return '批量操作成功完成'
})

// 建立WebSocket连接
const connectWebSocket = () => {
  if (!props.taskId) {
    console.log('No taskId provided, skipping WebSocket connection')
    return
  }

  if (reconnectCount.value >= maxReconnectAttempts) {
    console.log('已达到最大重连次数限制，停止重连')
    ElMessage.warning('WebSocket 连接超时，请刷新页面重试')
    return
  }

  if (websocket.value) {
    console.log('关闭已有的WebSocket连接以建立新连接')
    closeWebSocket()
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const token = getToken()
  
  const wsUrl = `${protocol}//${host}/api/v1/progress/ws?token=${token}`
  
  console.log(`尝试连接 WebSocket (${reconnectCount.value + 1}/${maxReconnectAttempts}):`, wsUrl)
  console.log('TaskId:', props.taskId)
  
  try {
    websocket.value = new WebSocket(wsUrl)

    websocket.value.onopen = () => {
      console.log('WebSocket连接已建立')
      reconnectCount.value = 0
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

      if (isCompleted.value || isError.value) {
        console.log('任务已完成或出错，不再重连')
        return
      }

      const shouldReconnect = !isCompleted.value && !isError.value && visible.value && props.taskId
      const isNearCompletion = progressData.value.progress >= 100 && !isCompleted.value

      if ((shouldReconnect || isNearCompletion) && reconnectCount.value < maxReconnectAttempts) {
        reconnectCount.value++
        const reconnectDelay = Math.min(1000 * reconnectCount.value, 3000)
        console.log(`任务未完成，${reconnectDelay}ms 后尝试第 ${reconnectCount.value} 次重连`)
        setTimeout(() => {
          if ((!isCompleted.value && !isError.value && visible.value && props.taskId) ||
              (progressData.value.progress >= 100 && !isCompleted.value)) {
            connectWebSocket()
          }
        }, reconnectDelay)
      }
    }

    websocket.value.onerror = (error) => {
      console.error('WebSocket错误:', error)
      console.warn('WebSocket连接遇到错误')
    }
  } catch (error) {
    console.error('创建WebSocket连接失败:', error)
    ElMessage.error('无法建立WebSocket连接')
  }
}

// 处理进度更新
const handleProgressUpdate = (data) => {
  if (data.task_id && data.task_id !== props.taskId) {
    return
  }

  switch (data.type) {
    case 'connected':
      console.log('WebSocket连接确认:', data.message)
      break

    case 'progress':
      progressData.value = { ...data }

      if (data.progress >= 100 && !isCompleted.value && !isError.value) {
        console.log('进度达到100%，启动完成检查定时器')
        if (completionTimer.value) {
          clearTimeout(completionTimer.value)
        }
        completionTimer.value = setTimeout(() => {
          if (!isCompleted.value && !isError.value) {
            console.log('进度100%后未收到完成消息，尝试重连获取状态')
            connectWebSocket()
          }
        }, 3000)
      }
      break

    case 'complete':
      console.log('📦 Complete message received:', {
        task_id: data.task_id,
        success_count: data.success_nodes?.length,
        failed_count: data.failed_nodes?.length,
        success_nodes: data.success_nodes,
        failed_nodes: data.failed_nodes,
        progress: data.progress,
        current: data.current,
        total: data.total
      })
      
      progressData.value = { ...data }
      isCompleted.value = true

      if (completionTimer.value) {
        clearTimeout(completionTimer.value)
        completionTimer.value = null
      }

      console.log('任务完成，关闭 WebSocket 连接')
      closeWebSocket()

      const successCount = data.success_nodes?.length || 0
      const failedCount = data.failed_nodes?.length || 0
      
      console.log(`✅ 批量操作完成统计: 成功=${successCount}, 失败=${failedCount}`)
      
      if (failedCount > 0) {
        ElMessage.warning(`批量操作完成：${successCount}个成功，${failedCount}个失败`)
      } else {
        ElMessage.success(data.message || '批量操作完成')
      }
      
      emit('completed', data)
      break

    case 'error':
      console.log('❌ Error message received:', {
        task_id: data.task_id,
        success_count: data.success_nodes?.length,
        failed_count: data.failed_nodes?.length,
        error: data.error,
        message: data.message
      })
      
      progressData.value = { ...data }
      isError.value = true

      if (completionTimer.value) {
        clearTimeout(completionTimer.value)
        completionTimer.value = null
      }

      const successCnt = data.success_nodes?.length || 0
      const failedCnt = data.failed_nodes?.length || 0
      console.log(`❌ 批量操作错误统计: 成功=${successCnt}, 失败=${failedCnt}`)
      ElMessage.error(`批量操作完成：${successCnt}个成功，${failedCnt}个失败`)
      emit('error', data)
      break

    default:
      console.log('收到未知类型的进度消息:', data)
  }
}

// 关闭对话框
const handleClose = () => {
  console.log('关闭进度对话框')
  
  if (!isCompleted.value && !isError.value && progressData.value.task_id) {
    console.log('任务仍在进行中，但用户选择关闭弹窗')
  }
  
  emit('update:modelValue', false)
  closeWebSocket()
  resetState()
}

// 取消操作
const handleCancel = () => {
  ElMessage.warning('取消功能正在开发中')
  emit('cancelled')
  handleClose()
}

// 关闭WebSocket连接
const closeWebSocket = () => {
  if (websocket.value) {
    console.log('正在关闭WebSocket连接')
    try {
      websocket.value.onclose = null
      websocket.value.onerror = null
      websocket.value.onmessage = null
      websocket.value.onopen = null
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
    success_nodes: [],
    failed_nodes: [],
    message: '',
    error: '',
    timestamp: null
  }
  isCompleted.value = false
  isError.value = false
  reconnectCount.value = 0

  if (completionTimer.value) {
    clearTimeout(completionTimer.value)
    completionTimer.value = null
  }
}

watch([() => props.taskId, () => props.modelValue], ([newTaskId, newVisible], [oldTaskId, oldVisible]) => {
  console.log('ProgressDialog state changed:', { newTaskId, newVisible, oldTaskId, oldVisible })
  
  if (newVisible && newTaskId) {
    if (newTaskId !== oldTaskId || !oldVisible) {
      closeWebSocket()
      resetState()
      nextTick(() => {
        connectWebSocket()
      })
    }
  } else if (!newVisible) {
    closeWebSocket()
  } else if (!newTaskId) {
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
  margin-bottom: 24px;
}

.progress-text {
  text-align: center;
  margin-top: 8px;
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.nodes-status-row {
  margin-bottom: 20px;
}

.status-card {
  height: 320px;
  display: flex;
  flex-direction: column;
}

.status-card :deep(.el-card__header) {
  padding: 12px 16px;
  background-color: #f5f7fa;
  border-bottom: 1px solid #ebeef5;
}

.status-card :deep(.el-card__body) {
  flex: 1;
  padding: 0;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  font-size: 20px;
}

.processing-icon {
  color: #409eff;
}

.success-icon {
  color: #67c23a;
}

.failed-icon {
  color: #f56c6c;
}

.header-title {
  font-weight: 600;
  font-size: 14px;
  color: #303133;
}

.node-list {
  height: 100%;
  padding: 12px;
  overflow-y: auto;
  max-height: 240px;
}

.empty-text {
  text-align: center;
  color: #909399;
  font-size: 13px;
  padding: 40px 0;
}

.node-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  margin-bottom: 6px;
  border-radius: 4px;
  font-size: 13px;
  transition: all 0.3s ease;
}

.node-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.node-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.processing-item {
  background-color: #ecf5ff;
  border-left: 3px solid #409eff;
  color: #409eff;
}

.success-item {
  background-color: #f0f9ff;
  border-left: 3px solid #67c23a;
  color: #67c23a;
}

.failed-item {
  background-color: #fef0f0;
  border-left: 3px solid #f56c6c;
  color: #f56c6c;
}

.error-trigger {
  cursor: pointer;
  text-decoration: underline;
  text-decoration-style: dotted;
}

.error-detail {
  padding: 4px;
}

.error-title {
  font-weight: 600;
  margin-bottom: 8px;
  color: #303133;
}

.error-message {
  color: #606266;
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
}

.rotating {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.node-list-enter-active,
.node-list-leave-active {
  transition: all 0.3s ease;
}

.node-list-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}

.node-list-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

.summary-info {
  margin-top: 16px;
}

.summary-detail {
  font-size: 14px;
  line-height: 1.6;
}

.success-count {
  color: #67c23a;
  font-weight: 600;
}

.failed-count {
  color: #f56c6c;
  font-weight: 600;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 自定义滚动条 */
.node-list::-webkit-scrollbar {
  width: 6px;
}

.node-list::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.node-list::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.node-list::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>
