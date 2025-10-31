<template>
  <div class="task-timeline">
    <el-timeline>
      <el-timeline-item
        v-for="(step, index) in timelineSteps"
        :key="index"
        :timestamp="step.timestamp"
        :type="getStepType(step.status)"
        :hollow="step.status === 'pending'"
        :icon="getStepIcon(step.status)"
        placement="top"
      >
        <el-card :body-style="{ padding: '12px' }">
          <div class="step-header">
            <span class="step-title">{{ step.title }}</span>
            <el-tag :type="getStatusType(step.status)" size="small">
              {{ getStatusText(step.status) }}
            </el-tag>
          </div>
          
          <div v-if="step.description" class="step-description">
            {{ step.description }}
          </div>
          
          <div v-if="step.duration" class="step-duration">
            <el-icon><Timer /></el-icon>
            耗时: {{ step.duration }}
          </div>
          
          <div v-if="step.details && step.details.length > 0" class="step-details">
            <el-collapse accordion>
              <el-collapse-item title="详细信息" name="1">
                <div v-for="(detail, idx) in step.details" :key="idx" class="detail-item">
                  <el-text :type="detail.type || 'info'" size="small">
                    {{ detail.text }}
                  </el-text>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
          
          <div v-if="step.error" class="step-error">
            <el-alert :title="step.error" type="error" :closable="false" />
          </div>
        </el-card>
      </el-timeline-item>
    </el-timeline>
    
    <div v-if="timelineSteps.length === 0" class="empty-timeline">
      <el-empty description="暂无执行记录" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Timer, Check, Close, Loading, Clock, Warning } from '@element-plus/icons-vue'
import { markRaw } from 'vue'

const props = defineProps({
  task: {
    type: Object,
    required: true
  }
})

// 根据任务状态生成时间线步骤
const timelineSteps = computed(() => {
  const steps = []
  const task = props.task
  
  // 1. 任务创建
  steps.push({
    title: '任务创建',
    status: 'success',
    timestamp: formatTimestamp(task.created_at),
    description: `创建人: ${task.user?.username || '未知'}`,
    duration: null
  })
  
  // 2. 任务排队
  if (task.status !== 'pending') {
    steps.push({
      title: '等待执行',
      status: 'success',
      timestamp: formatTimestamp(task.created_at),
      description: '任务已加入执行队列',
      duration: calculateDuration(task.created_at, task.started_at || task.created_at)
    })
  } else {
    steps.push({
      title: '等待执行',
      status: 'pending',
      timestamp: null,
      description: '任务正在等待执行...'
    })
  }
  
  // 3. 任务执行
  if (task.started_at) {
    const executionStep = {
      title: '执行任务',
      status: task.status === 'running' ? 'running' : task.status === 'success' ? 'success' : 'error',
      timestamp: formatTimestamp(task.started_at),
      description: `模板: ${task.template?.name || '直接执行'} | 清单: ${task.inventory?.name || '-'}`,
      details: []
    }
    
    // 添加执行详情
    if (task.cluster) {
      executionStep.details.push({
        text: `集群: ${task.cluster.name}`,
        type: 'info'
      })
    }
    
    if (task.inventory?.hosts_data?.total) {
      executionStep.details.push({
        text: `主机数: ${task.inventory.hosts_data.total}`,
        type: 'info'
      })
    }
    
    if (task.finished_at) {
      executionStep.duration = calculateDuration(task.started_at, task.finished_at)
    } else if (task.status === 'running') {
      executionStep.duration = calculateDuration(task.started_at, new Date().toISOString()) + ' (进行中)'
    }
    
    steps.push(executionStep)
  }
  
  // 4. 任务完成/失败
  if (task.finished_at) {
    const finalStep = {
      title: task.status === 'success' ? '任务完成' : task.status === 'failed' ? '任务失败' : '任务取消',
      status: task.status,
      timestamp: formatTimestamp(task.finished_at),
      description: task.status === 'success' 
        ? '所有操作已成功完成' 
        : task.status === 'failed'
        ? '任务执行过程中发生错误'
        : '任务已被取消'
    }
    
    if (task.error_msg) {
      finalStep.error = task.error_msg
    }
    
    // 重试信息
    if (task.retry_count > 0) {
      finalStep.details = finalStep.details || []
      finalStep.details.push({
        text: `重试次数: ${task.retry_count}/${task.max_retries}`,
        type: 'warning'
      })
    }
    
    steps.push(finalStep)
  } else if (task.status === 'running') {
    steps.push({
      title: '执行中',
      status: 'running',
      timestamp: null,
      description: '任务正在执行...'
    })
  }
  
  return steps
})

// 辅助方法
const formatTimestamp = (timestamp) => {
  if (!timestamp) return null
  return new Date(timestamp).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const calculateDuration = (start, end) => {
  if (!start || !end) return null
  
  const startTime = new Date(start)
  const endTime = new Date(end)
  const diff = endTime - startTime
  
  if (diff < 1000) {
    return `${diff}ms`
  } else if (diff < 60000) {
    return `${(diff / 1000).toFixed(1)}s`
  } else if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    const seconds = Math.floor((diff % 60000) / 1000)
    return `${minutes}m ${seconds}s`
  } else {
    const hours = Math.floor(diff / 3600000)
    const minutes = Math.floor((diff % 3600000) / 60000)
    return `${hours}h ${minutes}m`
  }
}

const getStepType = (status) => {
  const typeMap = {
    success: 'success',
    error: 'danger',
    failed: 'danger',
    cancelled: 'info',
    running: 'primary',
    pending: 'info'
  }
  return typeMap[status] || 'info'
}

const getStepIcon = (status) => {
  const iconMap = {
    success: markRaw(Check),
    error: markRaw(Close),
    failed: markRaw(Close),
    cancelled: markRaw(Warning),
    running: markRaw(Loading),
    pending: markRaw(Clock)
  }
  return iconMap[status] || markRaw(Clock)
}

const getStatusType = (status) => {
  const typeMap = {
    success: 'success',
    error: 'danger',
    failed: 'danger',
    cancelled: 'info',
    running: 'warning',
    pending: ''
  }
  return typeMap[status] || 'info'
}

const getStatusText = (status) => {
  const textMap = {
    success: '已完成',
    error: '失败',
    failed: '失败',
    cancelled: '已取消',
    running: '执行中',
    pending: '等待中'
  }
  return textMap[status] || status
}
</script>

<style scoped>
.task-timeline {
  padding: 20px 0;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.step-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.step-description {
  margin-top: 8px;
  color: #606266;
  font-size: 14px;
}

.step-duration {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  color: #909399;
  font-size: 13px;
}

.step-details {
  margin-top: 12px;
}

.detail-item {
  padding: 4px 0;
}

.step-error {
  margin-top: 12px;
}

.empty-timeline {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}

:deep(.el-timeline-item__timestamp) {
  font-size: 13px;
  color: #909399;
}

:deep(.el-timeline-item__wrapper) {
  padding-left: 24px;
}
</style>

