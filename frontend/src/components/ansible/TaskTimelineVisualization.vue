<template>
  <div class="task-timeline-visualization" v-loading="loading">
    <!-- 有数据时显示 -->
    <div v-if="!loading && visualization && visualization.timeline && visualization.timeline.length > 0">
      <!-- 时间线头部 -->
      <el-card class="header-card">
        <el-row :gutter="20">
          <el-col :span="6">
            <div class="stat-card-item">
              <div class="stat-label">任务名称</div>
              <div class="stat-value" :title="visualization.task_name || '未命名'">
                {{ visualization.task_name || '未命名任务' }}
              </div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="stat-card-item">
              <div class="stat-label">
                <el-icon><Clock /></el-icon>
                总耗时
              </div>
              <div class="stat-value">
                {{ formatDuration(visualization.total_duration) }}
              </div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="stat-card-item">
              <div class="stat-label">执行状态</div>
              <div class="stat-value">
                <el-tag :type="getStatusType(visualization.status)" size="large">
                  {{ getStatusText(visualization.status) }}
                </el-tag>
              </div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="stat-card-item">
              <div class="stat-label">执行阶段</div>
              <div class="stat-value">
                {{ visualization.timeline.length }} 个
              </div>
            </div>
          </el-col>
        </el-row>
      </el-card>

      <!-- 时间线 -->
      <el-card style="margin-top: 20px">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon><DataLine /></el-icon>
            <span>执行时间线</span>
          </div>
        </template>
        <el-timeline v-if="visualization.timeline && visualization.timeline.length > 0">
          <el-timeline-item
            v-for="(event, index) in visualization.timeline"
            :key="index"
            :timestamp="formatTimestamp(event.timestamp)"
            :type="getPhaseType(event.phase)"
            :icon="getPhaseIcon(event.phase)"
            placement="top"
            :hollow="false"
          >
            <el-card shadow="hover">
              <div style="display: flex; justify-content: space-between; align-items: flex-start">
                <div style="flex: 1">
                  <h4 style="margin: 0 0 8px 0; display: flex; align-items: center; gap: 8px">
                    <component :is="getPhaseIconComponent(event.phase)" />
                    {{ getPhaseLabel(event.phase) }}
                  </h4>
                  <p v-if="event.message" style="color: #606266; margin: 8px 0; line-height: 1.6">
                    {{ event.message }}
                  </p>
                </div>
                <div style="text-align: right; min-width: 140px">
                  <el-tag v-if="event.duration && event.duration > 0" type="info" effect="plain" size="large">
                    <el-icon><Timer /></el-icon>
                    {{ formatDuration(event.duration) }}
                  </el-tag>
                  <el-tag v-else type="info" effect="plain" size="small">
                    <el-icon><Clock /></el-icon>
                    瞬时
                  </el-tag>
                  <div v-if="event.batch_number" style="margin-top: 8px">
                    <el-tag size="small" type="warning">
                      批次 {{ event.batch_number }}
                    </el-tag>
                  </div>
                </div>
              </div>
              
              <!-- 批次详情 -->
              <div v-if="event.host_count" style="margin-top: 12px">
                <el-divider />
                <el-row :gutter="16">
                  <el-col :span="8">
                    <div class="stat-item">
                      <el-icon class="stat-icon"><Monitor /></el-icon>
                      <span>主机总数: <strong>{{ event.host_count }}</strong></span>
                    </div>
                  </el-col>
                  <el-col :span="8">
                    <div class="stat-item success">
                      <el-icon class="stat-icon"><CircleCheck /></el-icon>
                      <span>成功: <strong>{{ event.success_count }}</strong></span>
                    </div>
                  </el-col>
                  <el-col :span="8">
                    <div class="stat-item error">
                      <el-icon class="stat-icon"><CircleClose /></el-icon>
                      <span>失败: <strong>{{ event.fail_count }}</strong></span>
                    </div>
                  </el-col>
                </el-row>
              </div>
              
              <!-- 额外详情 -->
              <div v-if="event.details && Object.keys(event.details).length > 0" style="margin-top: 12px">
                <el-divider />
                <el-descriptions :column="2" size="small" border>
                  <el-descriptions-item
                    v-for="(value, key) in event.details"
                    :key="key"
                    :label="key"
                  >
                    {{ value }}
                  </el-descriptions-item>
                </el-descriptions>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
        <el-empty v-else description="暂无执行时间线数据" />
      </el-card>

      <!-- 阶段耗时分布饼图 -->
      <el-card style="margin-top: 20px" v-if="hasPhaseDistribution">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <el-icon><PieChart /></el-icon>
            <span>阶段耗时分布</span>
          </div>
        </template>
        <div ref="chartRef" style="height: 400px"></div>
      </el-card>
    </div>
    
    <!-- 无数据时显示 -->
    <el-empty 
      v-else-if="!loading && (!visualization || !visualization.timeline || visualization.timeline.length === 0)" 
      description="暂无可视化数据"
    >
      <template #description>
        <div>
          <p>该任务暂无执行时间线数据</p>
          <p style="font-size: 12px; color: #909399; margin-top: 8px">
            可能原因：任务尚未执行或执行过程中未记录时间线
          </p>
        </div>
      </template>
    </el-empty>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { 
  Clock, DataLine, CircleCheck, CircleClose, Loading as LoadingIcon, 
  WarningFilled, SuccessFilled, InfoFilled, Timer, 
  Monitor, PieChart 
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { getTaskVisualization } from '@/api/ansible'

const props = defineProps({
  taskId: {
    type: Number,
    required: true
  }
})

const loading = ref(false)
const visualization = ref(null)
const chartRef = ref(null)
let chart = null

// 计算属性：是否有阶段分布数据
const hasPhaseDistribution = computed(() => {
  return visualization.value?.phase_distribution && 
    Object.keys(visualization.value.phase_distribution).length > 0
})

// 加载可视化数据
const loadVisualization = async () => {
  if (!props.taskId) {
    console.warn('TaskTimelineVisualization: taskId is required')
    return
  }
  
  loading.value = true
  visualization.value = null  // 重置数据
  
  try {
    console.log(`Loading visualization for task ${props.taskId}`)
    const response = await getTaskVisualization(props.taskId)
    
    console.log('Visualization response:', response)
    
    // 检查响应数据结构
    if (response && response.data && response.data.code === 200) {
      visualization.value = response.data.data
      console.log('Visualization data:', visualization.value)
      
      // 渲染图表（需要等待 DOM 更新）
      if (hasPhaseDistribution.value) {
        await nextTick()
        renderChart()
      }
    } else {
      console.warn('Invalid visualization response:', response)
      ElMessage.warning('可视化数据格式不正确')
    }
  } catch (error) {
    console.error('Failed to load visualization:', error)
    ElMessage.error(`加载可视化数据失败: ${error.message || '未知错误'}`)
    visualization.value = null
  } finally {
    // 确保 loading 状态被重置
    loading.value = false
    console.log('Loading complete, visualization:', visualization.value)
  }
}

// 渲染饼图
const renderChart = () => {
  if (!chartRef.value || !hasPhaseDistribution.value) return
  
  if (!chart) {
    chart = echarts.init(chartRef.value)
  }
  
  const data = Object.entries(visualization.value.phase_distribution || {}).map(([name, value]) => ({
    name: getPhaseLabel(name),
    value: value
  }))
  
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c}ms ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      top: 'middle'
    },
    series: [
      {
        name: '阶段耗时',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: true,
          formatter: '{b}: {d}%'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 16,
            fontWeight: 'bold'
          },
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        },
        data: data,
        // 配色方案
        color: [
          '#5470c6', '#91cc75', '#fac858', '#ee6666', 
          '#73c0de', '#3ba272', '#fc8452', '#9a60b4'
        ]
      }
    ]
  }
  
  chart.setOption(option)
  
  // 响应式处理
  window.addEventListener('resize', () => {
    chart?.resize()
  })
}

// 辅助方法
const getPhaseLabel = (phase) => {
  const labels = {
    'queued': '⏰ 入队等待',
    'preflight_check': '🔍 前置检查',
    'executing': '⚙️ 执行中',
    'batch_paused': '⏸️ 批次暂停',
    'completed': '✅ 已完成',
    'failed': '❌ 执行失败',
    'cancelled': '🚫 已取消',
    'timeout': '⏱️ 执行超时'
  }
  return labels[phase] || phase
}

const getPhaseType = (phase) => {
  const types = {
    'queued': 'info',
    'preflight_check': 'warning',
    'executing': 'primary',
    'batch_paused': 'warning',
    'completed': 'success',
    'failed': 'danger',
    'cancelled': 'info',
    'timeout': 'danger'
  }
  return types[phase] || 'info'
}

const getPhaseIcon = (phase) => {
  // Element Plus Timeline 组件的图标
  return null
}

const getPhaseIconComponent = (phase) => {
  const icons = {
    'queued': Clock,
    'preflight_check': InfoFilled,
    'executing': LoadingIcon,
    'batch_paused': Timer,
    'completed': SuccessFilled,
    'failed': CircleClose,
    'cancelled': WarningFilled,
    'timeout': WarningFilled
  }
  return icons[phase] || InfoFilled
}

const formatDuration = (ms) => {
  if (!ms || ms === 0) return '0秒'
  if (ms < 1000) return `${ms}毫秒`
  
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}秒`
  
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  
  if (minutes < 60) {
    return remainingSeconds > 0 
      ? `${minutes}分${remainingSeconds}秒` 
      : `${minutes}分钟`
  }
  
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  
  if (remainingMinutes > 0) {
    return `${hours}小时${remainingMinutes}分钟`
  }
  return `${hours}小时`
}

const formatTimestamp = (timestamp) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const getStatusText = (status) => {
  const texts = {
    'pending': '等待中',
    'running': '运行中',
    'success': '成功',
    'failed': '失败',
    'cancelled': '已取消'
  }
  return texts[status] || status
}

const getStatusType = (status) => {
  const types = {
    'pending': 'info',
    'running': 'warning',
    'success': 'success',
    'failed': 'danger',
    'cancelled': 'info'
  }
  return types[status] || 'info'
}

onMounted(() => {
  loadVisualization()
})

watch(() => props.taskId, () => {
  loadVisualization()
})

// 清理
watch(() => chart, (newChart, oldChart) => {
  if (oldChart && !newChart) {
    oldChart.dispose()
  }
})
</script>

<style scoped>
.task-timeline-visualization {
  padding: 20px;
  min-height: 400px;
  max-height: 80vh;
  overflow-y: auto;
  position: relative;
}

/* 限制 loading 图标的大小 */
.task-timeline-visualization :deep(.el-loading-spinner) {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  margin-top: 0 !important;
}

.task-timeline-visualization :deep(.el-loading-spinner .circular) {
  width: 42px !important;
  height: 42px !important;
}

.task-timeline-visualization :deep(.el-loading-text) {
  font-size: 14px;
  margin-top: 10px;
}

.header-card {
  margin-bottom: 20px;
}

.stat-card-item {
  text-align: center;
  padding: 12px;
}

.stat-card-item .stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.stat-card-item .stat-value {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  word-break: break-word;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.stat-item.success {
  color: #67C23A;
}

.stat-item.error {
  color: #F56C6C;
}

.stat-icon {
  font-size: 18px;
}

/* Timeline 样式增强 */
:deep(.el-timeline-item__wrapper) {
  padding-left: 28px;
}

:deep(.el-timeline-item__tail) {
  border-left: 2px solid #e4e7ed;
}

:deep(.el-timeline-item__node) {
  font-size: 14px;
}

:deep(.el-card__body) {
  padding: 16px;
}

/* 限制所有图标的大小 */
:deep(.el-icon) {
  width: 16px;
  height: 16px;
  font-size: 16px;
}

:deep(.el-card__header .el-icon) {
  width: 18px;
  height: 18px;
  font-size: 18px;
}

:deep(.el-statistic__head .el-icon) {
  width: 20px;
  height: 20px;
  font-size: 20px;
}

/* 限制时间线卡片中的图标 */
:deep(.el-timeline-item .el-card h4 .el-icon),
:deep(.el-timeline-item .el-card h4 svg) {
  width: 18px !important;
  height: 18px !important;
  font-size: 18px !important;
  display: inline-block;
  vertical-align: middle;
}

/* 限制所有SVG元素的尺寸 */
:deep(svg) {
  max-width: 100%;
  max-height: 100%;
}

:deep(.el-icon svg) {
  width: 1em;
  height: 1em;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .task-timeline-visualization {
    padding: 10px;
  }
  
  :deep(.el-col) {
    margin-bottom: 16px;
  }
}
</style>

