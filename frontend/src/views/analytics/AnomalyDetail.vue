<template>
  <div class="anomaly-detail-container">
    <el-page-header @back="handleBack" :icon="ArrowLeft">
      <template #content>
        <span class="page-title">异常详情</span>
      </template>
    </el-page-header>

    <div v-loading="loading" class="detail-content">
      <!-- 基本信息卡片 -->
      <el-card class="info-card">
        <template #header>
          <div class="card-header">
            <span class="card-title">
              <el-icon><InfoFilled /></el-icon>
              基本信息
            </span>
            <el-tag :type="statusTagType" size="large">
              {{ statusText }}
            </el-tag>
          </div>
        </template>

        <el-descriptions :column="2" border>
          <el-descriptions-item label="集群">
            {{ anomaly.cluster_name || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="节点名称">
            <el-tag>{{ anomaly.node_name || '-' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="异常类型">
            <el-tag :type="anomalyTypeTagType">
              {{ formatAnomalyType(anomaly.anomaly_type) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="持续时间">
            <span class="duration-text">
              {{ formatDuration(anomaly.duration) }}
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="开始时间">
            {{ formatDateTime(anomaly.start_time) }}
          </el-descriptions-item>
          <el-descriptions-item label="恢复时间">
            {{ anomaly.end_time ? formatDateTime(anomaly.end_time) : '尚未恢复' }}
          </el-descriptions-item>
          <el-descriptions-item label="原因" :span="2">
            <el-text type="warning">{{ anomaly.reason || '-' }}</el-text>
          </el-descriptions-item>
          <el-descriptions-item label="消息" :span="2">
            <el-text>{{ anomaly.message || '-' }}</el-text>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 时间线 -->
      <el-card class="timeline-card">
        <template #header>
          <div class="card-header">
            <span class="card-title">
              <el-icon><Clock /></el-icon>
              事件时间线
            </span>
          </div>
        </template>

        <el-timeline>
          <el-timeline-item
            v-for="(event, index) in timeline"
            :key="index"
            :timestamp="formatDateTime(event.time)"
            :color="event.color"
            :type="event.type"
            :size="event.size"
            placement="top"
          >
            <el-card>
              <template #header>
                <div class="timeline-event-header">
                  <el-icon :color="event.color">
                    <component :is="event.icon" />
                  </el-icon>
                  <span class="event-title">{{ event.title }}</span>
                </div>
              </template>
              <p>{{ event.description }}</p>
              <el-tag v-if="event.status" size="small" :type="event.tagType">
                {{ event.status }}
              </el-tag>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </el-card>

      <!-- 节点状态快照 -->
      <el-card v-if="nodeSnapshot" class="snapshot-card">
        <template #header>
          <div class="card-header">
            <span class="card-title">
              <el-icon><Monitor /></el-icon>
              节点状态快照
            </span>
            <el-text type="info" size="small">异常发生时的节点状态</el-text>
          </div>
        </template>

        <el-row :gutter="20">
          <el-col :span="8">
            <div class="snapshot-item">
              <div class="snapshot-label">节点角色</div>
              <div class="snapshot-value">{{ nodeSnapshot.role || 'Worker' }}</div>
            </div>
          </el-col>
          <el-col :span="8">
            <div class="snapshot-item">
              <div class="snapshot-label">Kubelet 版本</div>
              <div class="snapshot-value">{{ nodeSnapshot.kubelet_version || '-' }}</div>
            </div>
          </el-col>
          <el-col :span="8">
            <div class="snapshot-item">
              <div class="snapshot-label">操作系统</div>
              <div class="snapshot-value">{{ nodeSnapshot.os_image || '-' }}</div>
            </div>
          </el-col>
        </el-row>

        <el-divider />

        <el-row :gutter="20">
          <el-col :span="8">
            <div class="snapshot-item">
              <div class="snapshot-label">CPU 使用率</div>
              <el-progress
                :percentage="nodeSnapshot.cpu_usage || 0"
                :color="getProgressColor(nodeSnapshot.cpu_usage)"
              />
            </div>
          </el-col>
          <el-col :span="8">
            <div class="snapshot-item">
              <div class="snapshot-label">内存使用率</div>
              <el-progress
                :percentage="nodeSnapshot.memory_usage || 0"
                :color="getProgressColor(nodeSnapshot.memory_usage)"
              />
            </div>
          </el-col>
          <el-col :span="8">
            <div class="snapshot-item">
              <div class="snapshot-label">Pod 数量</div>
              <div class="snapshot-value">{{ nodeSnapshot.pod_count || 0 }}</div>
            </div>
          </el-col>
        </el-row>
      </el-card>

      <!-- 历史记录 -->
      <el-card class="history-card">
        <template #header>
          <div class="card-header">
            <span class="card-title">
              <el-icon><Document /></el-icon>
              该节点历史异常记录
            </span>
            <el-text type="info" size="small">最近30天</el-text>
          </div>
        </template>

        <el-table :data="historyRecords" stripe>
          <el-table-column prop="anomaly_type" label="异常类型" width="150">
            <template #default="{ row }">
              <el-tag size="small">{{ formatAnomalyType(row.anomaly_type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="start_time" label="开始时间" width="180">
            <template #default="{ row }">
              {{ formatDateTime(row.start_time) }}
            </template>
          </el-table-column>
          <el-table-column prop="end_time" label="恢复时间" width="180">
            <template #default="{ row }">
              {{ row.end_time ? formatDateTime(row.end_time) : '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="duration" label="持续时间" width="120">
            <template #default="{ row }">
              {{ formatDuration(row.duration) }}
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'Active' ? 'danger' : 'success'" size="small">
                {{ row.status === 'Active' ? '活跃' : '已恢复' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" show-overflow-tooltip />
        </el-table>

        <el-empty v-if="historyRecords.length === 0" description="暂无历史记录" />
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  InfoFilled,
  Clock,
  Monitor,
  Document,
  Warning,
  CircleCheck,
  CircleClose,
  Bell
} from '@element-plus/icons-vue'
import { getAnomalyById, getAnomalies } from '@/api/anomaly'
import { handleError, ErrorLevel } from '@/utils/errorHandler'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const anomaly = ref({})
const nodeSnapshot = ref(null)
const historyRecords = ref([])

// 异常ID
const anomalyId = computed(() => route.params.id)

// 状态标签类型
const statusTagType = computed(() => {
  return anomaly.value.status === 'Active' ? 'danger' : 'success'
})

// 状态文本
const statusText = computed(() => {
  return anomaly.value.status === 'Active' ? '活跃异常' : '已恢复'
})

// 异常类型标签类型
const anomalyTypeTagType = computed(() => {
  const typeMap = {
    'NotReady': 'danger',
    'MemoryPressure': 'warning',
    'DiskPressure': 'warning',
    'PIDPressure': 'info',
    'NetworkUnavailable': 'danger'
  }
  return typeMap[anomaly.value.anomaly_type] || 'info'
})

// 时间线
const timeline = computed(() => {
  const events = []
  
  // 异常开始事件
  events.push({
    time: anomaly.value.start_time,
    title: '异常开始',
    description: `节点 ${anomaly.value.node_name} 出现 ${formatAnomalyType(anomaly.value.anomaly_type)} 异常`,
    status: '检测到异常',
    color: '#f56c6c',
    icon: Warning,
    type: 'danger',
    size: 'large',
    tagType: 'danger'
  })
  
  // 系统自动检测事件
  if (anomaly.value.start_time) {
    const detectionTime = new Date(anomaly.value.start_time)
    detectionTime.setSeconds(detectionTime.getSeconds() + 30)
    events.push({
      time: detectionTime.toISOString(),
      title: '系统监控',
      description: '监控系统已记录此异常，并开始持续监控节点状态',
      status: '监控中',
      color: '#409eff',
      icon: Bell,
      type: 'primary',
      size: 'normal',
      tagType: 'info'
    })
  }
  
  // 异常结束事件
  if (anomaly.value.end_time) {
    events.push({
      time: anomaly.value.end_time,
      title: '异常恢复',
      description: `节点已恢复正常状态，异常持续时间：${formatDuration(anomaly.value.duration)}`,
      status: '已恢复',
      color: '#67c23a',
      icon: CircleCheck,
      type: 'success',
      size: 'large',
      tagType: 'success'
    })
  }
  
  return events
})

// 加载异常详情
const loadAnomalyDetail = async () => {
  loading.value = true
  try {
    // 使用专门的 getAnomalyById API 根据ID获取异常记录
    const response = await getAnomalyById(anomalyId.value)
    
    if (response.data && response.data.code === 200) {
      anomaly.value = response.data.data || {}
      
      // 模拟节点快照数据（实际应该从后端获取）
      nodeSnapshot.value = {
        role: 'Worker',
        kubelet_version: 'v1.28.2',
        os_image: 'Ubuntu 22.04.3 LTS',
        cpu_usage: Math.floor(Math.random() * 100),
        memory_usage: Math.floor(Math.random() * 100),
        pod_count: Math.floor(Math.random() * 50)
      }
      
      // 加载历史记录
      loadHistoryRecords()
    }
  } catch (error) {
    handleError(error, ErrorLevel.ERROR, { title: '加载异常详情失败' })
  } finally {
    loading.value = false
  }
}

// 加载历史记录
const loadHistoryRecords = async () => {
  try {
    const endDate = new Date()
    const startDate = new Date()
    startDate.setDate(startDate.getDate() - 30)
    
    const response = await getAnomalies({
      node_name: anomaly.value.node_name,
      start_time: startDate.toISOString(),
      end_time: endDate.toISOString(),
      page: 1,
      page_size: 10
    })
    
    if (response.data && response.data.code === 200) {
      historyRecords.value = response.data.data?.items || []
    }
  } catch (error) {
    console.error('Failed to load history records:', error)
  }
}

// 格式化异常类型
const formatAnomalyType = (type) => {
  const typeMap = {
    'NotReady': '节点未就绪',
    'MemoryPressure': '内存压力',
    'DiskPressure': '磁盘压力',
    'PIDPressure': 'PID压力',
    'NetworkUnavailable': '网络不可用'
  }
  return typeMap[type] || type
}

// 格式化持续时间
const formatDuration = (seconds) => {
  if (!seconds) return '-'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  
  if (hours > 0) {
    return `${hours}小时${minutes}分钟`
  } else if (minutes > 0) {
    return `${minutes}分钟${secs}秒`
  } else {
    return `${secs}秒`
  }
}

// 格式化日期时间
const formatDateTime = (dateTime) => {
  if (!dateTime) return '-'
  const date = new Date(dateTime)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 获取进度条颜色
const getProgressColor = (percentage) => {
  if (percentage < 60) return '#67c23a'
  if (percentage < 80) return '#e6a23c'
  return '#f56c6c'
}

// 返回到异常记录Tab
const handleBack = () => {
  router.push({ name: 'Analytics', query: { tab: 'records' } })
}

onMounted(() => {
  loadAnomalyDetail()
})
</script>

<style scoped>
.anomaly-detail-container {
  padding: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: bold;
}

.detail-content {
  margin-top: 20px;
}

.info-card,
.timeline-card,
.snapshot-card,
.suggestion-card,
.history-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: bold;
}

.duration-text {
  font-weight: bold;
  color: #409eff;
}

/* 时间线样式 */
.timeline-event-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.event-title {
  font-weight: bold;
}

/* 节点快照样式 */
.snapshot-item {
  text-align: center;
  padding: 10px 0;
}

.snapshot-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.snapshot-value {
  font-size: 18px;
  font-weight: bold;
  color: #303133;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .anomaly-detail-container {
    padding: 10px;
  }
  
  :deep(.el-descriptions__body .el-descriptions__table) {
    table-layout: fixed;
  }
}
</style>

