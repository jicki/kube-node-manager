<template>
  <div class="task-timeline-visualization" v-loading="loading">
    <!-- 有数据时显示 -->
    <div v-if="!loading && visualization && visualization.timeline && visualization.timeline.length > 0">
      <!-- 时间线头部 -->
      <el-card class="header-card" shadow="hover">
        <el-row :gutter="20">
          <el-col :xs="24" :sm="12" :md="6">
            <div class="stat-card-item stat-card-primary">
              <div class="stat-icon-wrapper">
                <el-icon><DocumentCopy /></el-icon>
              </div>
              <div class="stat-content">
                <div class="stat-label">任务名称</div>
                <div class="stat-value" :title="visualization.task_name || '未命名'">
                  {{ visualization.task_name || '未命名任务' }}
                </div>
              </div>
            </div>
          </el-col>
          <el-col :xs="24" :sm="12" :md="6">
            <div class="stat-card-item stat-card-success">
              <div class="stat-icon-wrapper">
                <el-icon><Clock /></el-icon>
              </div>
              <div class="stat-content">
                <div class="stat-label">总耗时</div>
                <div class="stat-value">
                  {{ formatDuration(visualization.total_duration) }}
                </div>
              </div>
            </div>
          </el-col>
          <el-col :xs="24" :sm="12" :md="6">
            <div class="stat-card-item stat-card-info">
              <div class="stat-icon-wrapper">
                <el-icon><InfoFilled /></el-icon>
              </div>
              <div class="stat-content">
                <div class="stat-label">执行状态</div>
                <div class="stat-value">
                  <el-tag :type="getStatusType(visualization.status)" size="large" effect="dark">
                    {{ getStatusText(visualization.status) }}
                  </el-tag>
                </div>
              </div>
            </div>
          </el-col>
          <el-col :xs="24" :sm="12" :md="6">
            <div class="stat-card-item stat-card-warning">
              <div class="stat-icon-wrapper">
                <el-icon><DataLine /></el-icon>
              </div>
              <div class="stat-content">
                <div class="stat-label">执行阶段</div>
                <div class="stat-value">
                  {{ visualization.timeline.length }} 个
                </div>
              </div>
            </div>
          </el-col>
        </el-row>
      </el-card>

      <!-- 时间线 -->
      <el-card style="margin-top: 20px" shadow="hover">
        <template #header>
          <div style="display: flex; align-items: center; justify-content: space-between">
            <div style="display: flex; align-items: center; gap: 8px">
              <el-icon><DataLine /></el-icon>
              <span>执行时间线</span>
            </div>
            <el-tag type="info" size="small">
              {{ visualization.timeline.length }} 个事件
            </el-tag>
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

      <!-- 阶段耗时分布 -->
      <el-card style="margin-top: 20px">
        <template #header>
          <div style="display: flex; align-items: center; justify-content: space-between">
            <div style="display: flex; align-items: center; gap: 8px">
              <el-icon><PieChart /></el-icon>
              <span>阶段耗时分布</span>
            </div>
            <el-tag v-if="hasPhaseDistribution" type="success" size="small">
              {{ Object.keys(visualization.phase_distribution).length }} 个阶段
            </el-tag>
          </div>
        </template>
        
        <!-- 有分布数据时显示饼图 -->
        <div v-if="hasPhaseDistribution">
          <div ref="chartRef" style="height: 400px"></div>
          
          <!-- 阶段详细统计 -->
          <el-divider />
          <div class="phase-stats">
            <h4 style="margin: 0 0 16px 0; font-size: 14px; color: #606266">阶段耗时详情</h4>
            <el-row :gutter="16">
              <el-col 
                v-for="(duration, phase) in visualization.phase_distribution" 
                :key="phase"
                :xs="24" :sm="12" :md="8" :lg="6"
                style="margin-bottom: 16px"
              >
                <div class="phase-stat-card">
                  <div class="phase-stat-label">{{ getDetailedPhaseLabel(phase) }}</div>
                  <div class="phase-stat-value">{{ formatDuration(duration) }}</div>
                  <div class="phase-stat-percent">
                    {{ calculatePercentage(duration) }}%
                  </div>
                </div>
              </el-col>
            </el-row>
          </div>
        </div>
        
        <!-- 无分布数据时的友好提示 -->
        <el-empty 
          v-else 
          description="暂无阶段耗时分布数据" 
          :image-size="120"
        >
          <template #description>
            <div style="color: #909399; font-size: 14px; padding: 0 20px">
              <p style="margin: 8px 0; line-height: 1.6">
                该任务的执行时间线尚未包含详细的阶段耗时数据
              </p>
              <div style="margin-top: 16px; text-align: left; display: inline-block">
                <p style="margin: 8px 0; font-size: 13px; font-weight: 500; color: #606266">
                  可能原因：
                </p>
                <ul style="margin: 0; padding-left: 20px; font-size: 12px; line-height: 2">
                  <li>任务执行时间极短（几乎瞬时完成）</li>
                  <li>任务尚未开始执行或仍在队列中</li>
                  <li>执行过程中未记录详细的阶段时间戳</li>
                  <li>任务已被取消或超时</li>
                </ul>
              </div>
              <el-alert 
                v-if="visualization && visualization.total_duration > 0"
                type="info" 
                :closable="false"
                style="margin-top: 16px"
              >
                <template #title>
                  <div style="font-size: 13px">
                    任务总耗时: <strong>{{ formatDuration(visualization.total_duration) }}</strong>
                  </div>
                </template>
              </el-alert>
            </div>
          </template>
        </el-empty>
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
import { ref, onMounted, onBeforeUnmount, watch, computed, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { 
  Clock, DataLine, CircleCheck, CircleClose, Loading as LoadingIcon, 
  WarningFilled, SuccessFilled, InfoFilled, Timer, 
  Monitor, PieChart, DocumentCopy
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import { getTaskVisualization } from '@/api/ansible'

const props = defineProps({
  taskId: {
    type: Number,
    required: true
  }
})

// 暴露给父组件的方法
defineExpose({
  refreshChart: () => {
    console.log('refreshChart called from parent')
    if (hasPhaseDistribution.value && chartRef.value) {
      renderChartTimer = setTimeout(() => {
        renderChart()
      }, 200)
    }
  }
})

const loading = ref(false)
const visualization = ref(null)
const chartRef = ref(null)
let chart = null
let renderChartTimer = null // 防抖定时器
let isRendering = ref(false) // 是否正在渲染

// 计算属性：是否有阶段分布数据
const hasPhaseDistribution = computed(() => {
  return visualization.value?.phase_distribution && 
    Object.keys(visualization.value.phase_distribution).length > 0
})

// 清理 ECharts 实例
const disposeChart = () => {
  if (chart) {
    console.log('Disposing existing chart instance')
    try {
      chart.dispose()
    } catch (error) {
      console.error('Error disposing chart:', error)
    }
    chart = null
  }
}

// 加载可视化数据
const loadVisualization = async () => {
  if (!props.taskId) {
    console.warn('TaskTimelineVisualization: taskId is required')
    return
  }
  
  loading.value = true
  visualization.value = null  // 重置数据
  disposeChart() // 清理旧的图表实例
  
  try {
    console.log(`Loading visualization for task ${props.taskId}`)
    const response = await getTaskVisualization(props.taskId)
    
    console.log('Visualization response:', response)
    
    // 检查响应数据结构
    if (response && response.data && response.data.code === 200) {
      visualization.value = response.data.data
      console.log('Visualization data:', visualization.value)
      
      // 不在这里直接渲染，而是让 watch 来触发渲染
      if (hasPhaseDistribution.value) {
        console.log('Has phase distribution, will render via watch')
      } else {
        console.warn('No phase distribution data available')
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

// 渲染饼图（带防抖）
const renderChart = () => {
  // 防抖：清除之前的定时器
  if (renderChartTimer) {
    clearTimeout(renderChartTimer)
    renderChartTimer = null
  }
  
  // 防止重复渲染
  if (isRendering.value) {
    console.log('Chart is already rendering, skipping')
    return
  }
  
  console.log('renderChart called', {
    hasChartRef: !!chartRef.value,
    hasPhaseDistribution: hasPhaseDistribution.value,
    phaseDistribution: visualization.value?.phase_distribution,
    loading: loading.value
  })
  
  // 如果正在加载，则延迟渲染
  if (loading.value) {
    console.log('Still loading, deferring chart render')
    renderChartTimer = setTimeout(() => renderChart(), 200)
    return
  }
  
  if (!hasPhaseDistribution.value) {
    console.warn('No phase distribution data')
    return
  }
  
  if (!chartRef.value) {
    console.warn('chartRef.value is null, will retry after nextTick')
    // 使用 nextTick 确保 DOM 已更新
    nextTick(() => {
      renderChartTimer = setTimeout(() => {
        if (chartRef.value) {
          console.log('chartRef available after nextTick, retrying')
          renderChart()
        } else {
          console.error('chartRef still null after nextTick')
        }
      }, 100)
    })
    return
  }
  
  // 检查元素是否可见
  const rect = chartRef.value.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) {
    console.warn('Chart container has zero size, will retry', rect)
    renderChartTimer = setTimeout(() => renderChart(), 200)
    return
  }
  
  isRendering.value = true
  
  try {
    // 清理旧实例
    if (chart) {
      console.log('Disposing old chart before re-init')
      try {
        chart.dispose()
      } catch (e) {
        console.warn('Error disposing old chart:', e)
      }
      chart = null
    }
    
    // 创建新实例
    console.log('Initializing echarts with container size:', rect.width, rect.height)
    chart = echarts.init(chartRef.value)
  } catch (error) {
    console.error('Failed to initialize echarts:', error)
    isRendering.value = false
    return
  }
  
  // 准备数据并排序（按耗时从大到小）
  const data = Object.entries(visualization.value.phase_distribution || {})
    .map(([name, value]) => ({
      name: getDetailedPhaseLabel(name), // 使用详细标签，包含 TASK 名称
      value: value,
      rawPhase: name
    }))
    .sort((a, b) => b.value - a.value)
  
  console.log('Chart data prepared:', data)
  
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: (params) => {
        const duration = formatDuration(params.value)
        return `${params.seriesName}<br/>${params.marker}${params.name}<br/>耗时: ${duration}<br/>占比: ${params.percent}%`
      },
      backgroundColor: 'rgba(50, 50, 50, 0.9)',
      borderColor: '#777',
      borderWidth: 1,
      textStyle: {
        color: '#fff',
        fontSize: 13
      },
      padding: 12
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      top: 'middle',
      itemGap: 16,
      itemWidth: 16,
      itemHeight: 16,
      textStyle: {
        fontSize: 13,
        color: '#606266'
      },
      formatter: (name) => {
        const item = data.find(d => d.name === name)
        if (item) {
          return `${name} (${formatDuration(item.value)})`
        }
        return name
      }
    },
    series: [
      {
        name: '执行阶段',
        type: 'pie',
        radius: ['45%', '75%'],
        center: ['65%', '50%'],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: 12,
          borderColor: '#fff',
          borderWidth: 3,
          shadowBlur: 10,
          shadowColor: 'rgba(0, 0, 0, 0.1)'
        },
        label: {
          show: true,
          position: 'outside',
          formatter: '{b}\n{d}%',
          fontSize: 13,
          fontWeight: 'bold',
          color: '#606266',
          lineHeight: 18
        },
        labelLine: {
          show: true,
          length: 15,
          length2: 30,
          smooth: true,
          lineStyle: {
            width: 2
          }
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 16,
            fontWeight: 'bold'
          },
          itemStyle: {
            shadowBlur: 20,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.3)'
          },
          scaleSize: 10
        },
        data: data,
        // 精选配色方案 - 使用现代化的渐变色
        color: [
          '#667eea', '#91cc75', '#fac858', '#ee6666', 
          '#73c0de', '#3ba272', '#fc8452', '#9a60b4',
          '#f093fb', '#4facfe', '#43e97b', '#fa709a'
        ],
        // 动画配置
        animationType: 'scale',
        animationEasing: 'elasticOut',
        animationDelay: (idx) => idx * 100
      }
    ]
  }
  
  console.log('Setting chart option')
  try {
    chart.setOption(option, true)
    console.log('Chart rendered successfully')
  } catch (error) {
    console.error('Failed to set chart option:', error)
    isRendering.value = false
    return
  }
  
  // 清理旧的事件监听器
  const resizeHandler = () => {
    if (chart && !chart.isDisposed()) {
      chart.resize()
    }
  }
  
  // 响应式处理
  window.removeEventListener('resize', resizeHandler)
  window.addEventListener('resize', resizeHandler)
  
  // 渲染完成
  isRendering.value = false
}

// 辅助方法
const getPhaseLabel = (phase) => {
  const labels = {
    'queued': '⏰ 入队等待',
    'preflight_check': '🔍 前置检查',
    'executing': '⚙️ 执行中',
    'task_execution': '📋 任务执行',
    'batch_paused': '⏸️ 批次暂停',
    'completed': '✅ 已完成',
    'failed': '❌ 执行失败',
    'cancelled': '🚫 已取消',
    'timeout': '⏱️ 执行超时'
  }
  
  // 处理动态 TASK 阶段 (task_1, task_2, etc.)
  if (phase && phase.startsWith('task_')) {
    return '📋 任务执行'
  }
  
  return labels[phase] || phase
}

// 获取详细的阶段标签（包含 TASK 名称）
const getDetailedPhaseLabel = (phase) => {
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
  
  // 处理动态 TASK 阶段 (task_1, task_2, etc.)
  if (phase && phase.startsWith('task_')) {
    // 从时间线中查找对应的任务名称
    if (visualization.value?.timeline) {
      const event = visualization.value.timeline.find(e => e.phase === phase)
      if (event && event.details && event.details.task_name) {
        return `📋 ${event.details.task_name}`
      }
      // 如果有 message 字段，从中提取任务名称
      if (event && event.message) {
        const match = event.message.match(/执行任务:\s*(.+)/)
        if (match && match[1]) {
          return `📋 ${match[1]}`
        }
      }
    }
    return '📋 任务执行'
  }
  
  return labels[phase] || phase
}

const getPhaseType = (phase) => {
  const types = {
    'queued': 'info',
    'preflight_check': 'warning',
    'executing': 'primary',
    'task_execution': '',
    'batch_paused': 'warning',
    'completed': 'success',
    'failed': 'danger',
    'cancelled': 'info',
    'timeout': 'danger'
  }
  
  // 处理动态 TASK 阶段
  if (phase && phase.startsWith('task_')) {
    return ''
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
    'task_execution': DocumentCopy,
    'batch_paused': Timer,
    'completed': SuccessFilled,
    'failed': CircleClose,
    'cancelled': WarningFilled,
    'timeout': WarningFilled
  }
  
  // 处理动态 TASK 阶段
  if (phase && phase.startsWith('task_')) {
    return DocumentCopy
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

// 计算百分比
const calculatePercentage = (duration) => {
  if (!visualization.value?.phase_distribution) return 0
  
  const total = Object.values(visualization.value.phase_distribution).reduce((sum, val) => sum + val, 0)
  if (total === 0) return 0
  
  return ((duration / total) * 100).toFixed(1)
}

onMounted(() => {
  console.log('TaskTimelineVisualization mounted')
  loadVisualization()
})

onBeforeUnmount(() => {
  console.log('TaskTimelineVisualization unmounting, cleaning up')
  
  // 清理定时器
  if (renderChartTimer) {
    clearTimeout(renderChartTimer)
    renderChartTimer = null
  }
  
  // 清理图表实例
  disposeChart()
  
  // 移除事件监听
  window.removeEventListener('resize', null)
})

watch(() => props.taskId, (newId, oldId) => {
  if (newId !== oldId) {
    console.log('Task ID changed:', oldId, '->', newId)
    loadVisualization()
  }
})

// 监听 phase distribution 变化，自动渲染图表
watch(() => hasPhaseDistribution.value, (newValue, oldValue) => {
  console.log('hasPhaseDistribution changed:', oldValue, '->', newValue)
  if (newValue && !oldValue) {
    // 只在从 false 变为 true 时触发
    console.log('Phase distribution became available, scheduling chart render')
    // 确保 DOM 已更新并且不在 loading 状态
    nextTick(() => {
      if (!loading.value) {
        renderChartTimer = setTimeout(() => {
          console.log('Auto-rendering chart after hasPhaseDistribution became true')
          renderChart()
        }, 300) // 增加延迟确保 DOM 稳定
      } else {
        console.log('Still loading, will retry when loading completes')
      }
    })
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
  border-radius: 12px;
  overflow: hidden;
}

.stat-card-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border-radius: 8px;
  transition: all 0.3s ease;
  background: linear-gradient(135deg, #f5f7fa 0%, #ffffff 100%);
  height: 100%;
  min-height: 100px;
}

.stat-card-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.stat-icon-wrapper {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: white;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.stat-card-primary .stat-icon-wrapper {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-card-success .stat-icon-wrapper {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.stat-card-info .stat-icon-wrapper {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.stat-card-warning .stat-icon-wrapper {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
}

.stat-content {
  flex: 1;
  text-align: left;
  min-width: 0;
}

.stat-card-item .stat-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.stat-card-item .stat-value {
  font-size: 20px;
  font-weight: 700;
  color: #303133;
  word-break: break-word;
  overflow: hidden;
  text-overflow: ellipsis;
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

/* 阶段统计卡片 */
.phase-stats {
  padding: 16px 0;
}

.phase-stat-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  padding: 20px;
  color: white;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.15);
  transition: all 0.3s ease;
  text-align: center;
}

.phase-stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(102, 126, 234, 0.25);
}

.phase-stat-label {
  font-size: 14px;
  opacity: 0.9;
  margin-bottom: 12px;
  font-weight: 500;
}

.phase-stat-value {
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 8px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.phase-stat-percent {
  font-size: 12px;
  opacity: 0.85;
  background: rgba(255, 255, 255, 0.2);
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  backdrop-filter: blur(10px);
}

/* 为不同阶段使用不同的渐变色 */
.phase-stat-card:nth-child(1) {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.phase-stat-card:nth-child(2) {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.phase-stat-card:nth-child(3) {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.phase-stat-card:nth-child(4) {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.phase-stat-card:nth-child(5) {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
}

.phase-stat-card:nth-child(6) {
  background: linear-gradient(135deg, #30cfd0 0%, #330867 100%);
}

.phase-stat-card:nth-child(7) {
  background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
}

.phase-stat-card:nth-child(8) {
  background: linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .task-timeline-visualization {
    padding: 10px;
  }
  
  :deep(.el-col) {
    margin-bottom: 16px;
  }
  
  .stat-card-item {
    min-height: 80px;
    padding: 12px;
  }
  
  .stat-icon-wrapper {
    width: 48px;
    height: 48px;
    font-size: 20px;
  }
  
  .stat-card-item .stat-label {
    font-size: 12px;
  }
  
  .stat-card-item .stat-value {
    font-size: 16px;
  }
  
  .phase-stat-card {
    padding: 16px;
  }
  
  .phase-stat-value {
    font-size: 20px;
  }
  
  /* 移动端饼图调整 */
  :deep(.header-card) {
    margin-bottom: 16px;
  }
}

@media (max-width: 576px) {
  .stat-card-item {
    flex-direction: row;
    text-align: left;
    min-height: 70px;
  }
  
  .stat-icon-wrapper {
    width: 42px;
    height: 42px;
    font-size: 18px;
  }
  
  .stat-card-item .stat-value {
    font-size: 15px;
  }
  
  .phase-stat-card {
    padding: 14px;
  }
  
  .phase-stat-value {
    font-size: 18px;
  }
}
</style>

