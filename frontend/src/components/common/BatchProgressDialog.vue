<template>
  <el-dialog
    v-model="visible"
    :title="operation?.title || '批量操作进度'"
    width="800px"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    destroy-on-close
  >
    <div v-if="operation" class="progress-content">
      <!-- 操作概览 -->
      <div class="operation-overview">
        <div class="overview-item">
          <el-icon class="overview-icon"><Operation /></el-icon>
          <div class="overview-info">
            <div class="overview-title">{{ operation.title }}</div>
            <div class="overview-desc">{{ operation.description }}</div>
          </div>
          <div class="overview-status">
            <el-tag 
              :type="getStatusType(operation.status)"
              :icon="getStatusIcon(operation.status)"
            >
              {{ getStatusText(operation.status) }}
            </el-tag>
          </div>
        </div>
      </div>

      <!-- 进度条 -->
      <div class="progress-section">
        <el-progress
          :percentage="operation.progress || 0"
          :status="getProgressStatus(operation.status)"
          :stroke-width="8"
          :show-text="true"
        />
        <div class="progress-info">
          <span class="progress-text">
            {{ operation.completedCount }} / {{ operation.totalCount }} 已完成
          </span>
          <span v-if="operation.status === 'running' && operation.estimatedTimeRemaining" class="time-remaining">
            预计剩余: {{ formatEstimatedTime(operation.estimatedTimeRemaining) }}
          </span>
          <span v-if="operation.speed" class="speed-info">
            速度: {{ operation.speed.toFixed(1) }} 项/秒
          </span>
        </div>
      </div>

      <!-- 统计信息 -->
      <div class="stats-section">
        <div class="stat-item success">
          <el-icon class="stat-icon"><Check /></el-icon>
          <span class="stat-number">{{ operation.successCount }}</span>
          <span class="stat-label">成功</span>
        </div>
        <div class="stat-item failed">
          <el-icon class="stat-icon"><Close /></el-icon>
          <span class="stat-number">{{ operation.failedCount }}</span>
          <span class="stat-label">失败</span>
        </div>
        <div class="stat-item total">
          <el-icon class="stat-icon"><Grid /></el-icon>
          <span class="stat-number">{{ operation.totalCount }}</span>
          <span class="stat-label">总计</span>
        </div>
      </div>

      <!-- 当前处理项 -->
      <div v-if="operation.currentItem && operation.status === 'running'" class="current-item">
        <div class="current-label">
          <el-icon class="processing-icon"><Loading /></el-icon>
          正在处理:
        </div>
        <div class="current-name">{{ operation.currentItem }}</div>
      </div>

      <!-- 详细结果 -->
      <div class="results-section">
        <el-collapse v-model="activeCollapse">
          <el-collapse-item name="results" title="操作详情">
            <template #title>
              <span class="collapse-title">
                <el-icon><List /></el-icon>
                操作详情 ({{ operation.results.length }})
              </span>
            </template>
            
            <div class="results-list">
              <virtual-list
                v-if="operation.results.length > 0"
                :items="sortedResults"
                :item-height="60"
                :container-height="300"
              >
                <template #default="{ item }">
                  <div class="result-item" :class="item.status">
                    <div class="result-status">
                      <el-icon 
                        :class="item.status === 'success' ? 'success-icon' : 'error-icon'"
                      >
                        <Check v-if="item.status === 'success'" />
                        <Close v-else />
                      </el-icon>
                    </div>
                    <div class="result-content">
                      <div class="result-item-name">{{ getItemName(item.item) }}</div>
                      <div v-if="item.status === 'failed'" class="result-error">
                        {{ item.error }}
                      </div>
                      <div v-else class="result-success">
                        操作成功完成
                      </div>
                    </div>
                    <div class="result-time">
                      {{ formatTime(item.timestamp) }}
                    </div>
                  </div>
                </template>
              </virtual-list>
              
              <el-empty v-else description="暂无操作记录" :image-size="80" />
            </div>
          </el-collapse-item>
          
          <el-collapse-item v-if="operation.errors.length > 0" name="errors" title="错误详情">
            <template #title>
              <span class="collapse-title error">
                <el-icon><Warning /></el-icon>
                错误详情 ({{ operation.errors.length }})
              </span>
            </template>
            
            <div class="errors-list">
              <div 
                v-for="(error, index) in operation.errors" 
                :key="index" 
                class="error-item"
              >
                <div class="error-header">
                  <el-icon class="error-icon"><Close /></el-icon>
                  <span class="error-item-name">{{ getItemName(error.item) }}</span>
                  <span class="error-time">{{ formatTime(error.timestamp) }}</span>
                </div>
                <div class="error-message">{{ error.error }}</div>
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button 
          v-if="operation?.status === 'running'"
          type="danger"
          plain
          @click="handleCancel"
        >
          <el-icon><Close /></el-icon>
          取消操作
        </el-button>
        
        <el-button 
          v-if="operation?.status !== 'running'"
          @click="handleClose"
        >
          关闭
        </el-button>
        
        <el-button 
          v-if="operation?.status === 'completed' && operation.failedCount > 0"
          type="warning"
          @click="handleRetryFailed"
        >
          <el-icon><Refresh /></el-icon>
          重试失败项
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useProgressStore } from '@/store/modules/progress'
import VirtualList from './VirtualList.vue'
import {
  Operation,
  Check,
  Close,
  Grid,
  Loading,
  List,
  Warning,
  Refresh
} from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  operationId: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['update:modelValue', 'cancel', 'retry'])

const progressStore = useProgressStore()

const visible = ref(false)
const activeCollapse = ref(['results'])

// 计算属性
const operation = computed(() => {
  return progressStore.getOperationProgress(props.operationId)
})

const sortedResults = computed(() => {
  if (!operation.value?.results) return []
  return [...operation.value.results].sort((a, b) => {
    // 失败的结果排在前面
    if (a.status === 'failed' && b.status !== 'failed') return -1
    if (b.status === 'failed' && a.status !== 'failed') return 1
    // 按时间倒序排序
    return b.timestamp - a.timestamp
  })
})

// 监听 props 变化
watch(() => props.modelValue, (value) => {
  visible.value = value
})

watch(visible, (value) => {
  emit('update:modelValue', value)
})

// 状态相关方法
const getStatusType = (status) => {
  const types = {
    'running': 'primary',
    'completed': 'success',
    'failed': 'danger',
    'cancelled': 'warning'
  }
  return types[status] || 'info'
}

const getStatusIcon = (status) => {
  const icons = {
    'running': Loading,
    'completed': Check,
    'failed': Close,
    'cancelled': Warning
  }
  return icons[status] || Operation
}

const getStatusText = (status) => {
  const texts = {
    'running': '进行中',
    'completed': '已完成',
    'failed': '已失败',
    'cancelled': '已取消'
  }
  return texts[status] || '未知'
}

const getProgressStatus = (status) => {
  if (status === 'failed') return 'exception'
  if (status === 'completed') return 'success'
  return undefined
}

// 工具方法
const getItemName = (item) => {
  if (typeof item === 'string') return item
  if (item?.name) return item.name
  if (item?.id) return item.id
  return JSON.stringify(item)
}

const formatTime = (timestamp) => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString('zh-CN')
}

const formatEstimatedTime = (milliseconds) => {
  return progressStore.formatEstimatedTime(milliseconds)
}

// 事件处理
const handleCancel = async () => {
  try {
    await ElMessageBox.confirm('确认要取消当前操作吗？', '确认取消', {
      type: 'warning'
    })
    
    emit('cancel', props.operationId)
    progressStore.cancelOperation(props.operationId)
    ElMessage.info('操作已取消')
  } catch (error) {
    // 用户取消确认
  }
}

const handleClose = () => {
  visible.value = false
}

const handleRetryFailed = () => {
  if (!operation.value) return
  
  const failedItems = operation.value.results
    .filter(result => result.status === 'failed')
    .map(result => result.item)
  
  emit('retry', {
    operationId: props.operationId,
    items: failedItems
  })
}
</script>

<style scoped>
.progress-content {
  padding: 8px 0;
}

.operation-overview {
  margin-bottom: 24px;
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  border-left: 4px solid #1890ff;
}

.overview-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.overview-icon {
  color: #1890ff;
  font-size: 20px;
  margin-top: 2px;
}

.overview-info {
  flex: 1;
}

.overview-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.overview-desc {
  font-size: 14px;
  color: #666;
  line-height: 1.5;
}

.overview-status {
  margin-left: auto;
}

.progress-section {
  margin-bottom: 24px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
  font-size: 14px;
}

.progress-text {
  color: #333;
  font-weight: 500;
}

.time-remaining {
  color: #1890ff;
}

.speed-info {
  color: #52c41a;
}

.stats-section {
  display: flex;
  gap: 24px;
  justify-content: center;
  margin-bottom: 24px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 12px 16px;
  border-radius: 6px;
  background: white;
  box-shadow: 0 2px 4px rgba(0,0,0,0.05);
}

.stat-item.success .stat-icon {
  color: #52c41a;
}

.stat-item.failed .stat-icon {
  color: #ff4d4f;
}

.stat-item.total .stat-icon {
  color: #1890ff;
}

.stat-icon {
  font-size: 18px;
}

.stat-number {
  font-size: 20px;
  font-weight: 600;
  color: #333;
}

.stat-label {
  font-size: 12px;
  color: #666;
}

.current-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
  padding: 12px 16px;
  background: #e6f7ff;
  border-radius: 6px;
  border-left: 3px solid #1890ff;
}

.current-label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
  color: #1890ff;
  font-weight: 500;
}

.processing-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.current-name {
  font-size: 14px;
  color: #333;
  font-weight: 500;
}

.results-section {
  margin-top: 20px;
}

.collapse-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
}

.collapse-title.error {
  color: #ff4d4f;
}

.results-list {
  max-height: 300px;
  overflow: hidden;
}

.result-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.2s;
}

.result-item:hover {
  background: #fafafa;
}

.result-item.success .result-status {
  color: #52c41a;
}

.result-item.failed .result-status {
  color: #ff4d4f;
}

.result-status {
  font-size: 16px;
  margin-top: 2px;
}

.result-content {
  flex: 1;
  min-width: 0;
}

.result-item-name {
  font-size: 14px;
  color: #333;
  font-weight: 500;
  margin-bottom: 2px;
  word-break: break-all;
}

.result-error {
  font-size: 12px;
  color: #ff4d4f;
  line-height: 1.4;
}

.result-success {
  font-size: 12px;
  color: #52c41a;
}

.result-time {
  font-size: 12px;
  color: #999;
  white-space: nowrap;
}

.errors-list {
  max-height: 200px;
  overflow-y: auto;
}

.error-item {
  margin-bottom: 12px;
  padding: 12px;
  background: #fff2f0;
  border-radius: 6px;
  border-left: 3px solid #ff4d4f;
}

.error-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.error-icon {
  color: #ff4d4f;
  font-size: 14px;
}

.error-item-name {
  font-size: 14px;
  color: #333;
  font-weight: 500;
  flex: 1;
  word-break: break-all;
}

.error-time {
  font-size: 12px;
  color: #999;
}

.error-message {
  font-size: 12px;
  color: #a8071a;
  line-height: 1.4;
  padding-left: 22px;
  word-break: break-word;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .stats-section {
    flex-direction: column;
    gap: 12px;
  }
  
  .stat-item {
    flex-direction: row;
    justify-content: space-between;
    padding: 8px 12px;
  }
  
  .progress-info {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }
  
  .current-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>
