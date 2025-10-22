<template>
  <div class="trend-charts">
    <!-- 异常趋势折线图 -->
    <el-card class="chart-card">
      <template #header>
        <div class="chart-header">
          <span class="chart-title">
            <el-icon><TrendCharts /></el-icon>
            异常趋势分析
          </span>
          <el-radio-group v-model="trendDimension" size="small" @change="handleDimensionChange">
            <el-radio-button label="day">按天</el-radio-button>
            <el-radio-button label="week">按周</el-radio-button>
            <el-radio-button label="month">按月</el-radio-button>
          </el-radio-group>
        </div>
      </template>
      <v-chart 
        v-loading="trendLoading"
        class="chart" 
        :option="trendChartOption" 
        :autoresize="true"
        @click="handleTrendClick"
      />
    </el-card>

    <el-row :gutter="20">
      <!-- 异常类型分布饼图 -->
      <el-col :xs="24" :sm="24" :md="12" :lg="12">
        <el-card class="chart-card">
          <template #header>
            <div class="chart-header">
              <span class="chart-title">
                <el-icon><PieChart /></el-icon>
                异常类型分布
              </span>
            </div>
          </template>
          <v-chart 
            v-loading="typeLoading"
            class="chart small-chart" 
            :option="typeChartOption" 
            :autoresize="true"
            @click="handleTypeClick"
          />
        </el-card>
      </el-col>

      <!-- 节点异常排行榜 -->
      <el-col :xs="24" :sm="24" :md="12" :lg="12">
        <el-card class="chart-card">
          <template #header>
            <div class="chart-header">
              <span class="chart-title">
                <el-icon><DataLine /></el-icon>
                节点异常Top 10
              </span>
            </div>
          </template>
          <v-chart 
            v-loading="nodeLoading"
            class="chart small-chart" 
            :option="nodeChartOption" 
            :autoresize="true"
          />
        </el-card>
      </el-col>
    </el-row>

    <!-- 集群对比柱状图 -->
    <el-card v-if="clusterChartVisible" class="chart-card">
      <template #header>
        <div class="chart-header">
          <span class="chart-title">
            <el-icon><Histogram /></el-icon>
            集群异常对比
          </span>
        </div>
      </template>
      <v-chart 
        v-loading="clusterLoading"
        class="chart" 
        :option="clusterChartOption" 
        :autoresize="true"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart, BarChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DataZoomComponent
} from 'echarts/components'
import { TrendCharts, PieChart as PieChartIcon, DataLine, Histogram } from '@element-plus/icons-vue'
import { getAnomalyStatistics, getAnomalyTypeStatistics } from '@/api/anomaly'
import { handleError, ErrorLevel } from '@/utils/errorHandler'

// 注册 ECharts 组件
use([
  CanvasRenderer,
  LineChart,
  PieChart,
  BarChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
  DataZoomComponent
])

const props = defineProps({
  clusterId: {
    type: Number,
    default: null
  },
  startTime: {
    type: String,
    required: true
  },
  endTime: {
    type: String,
    required: true
  },
  clusters: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['dateClick', 'typeClick'])

// 趋势维度
const trendDimension = ref('day')

// 加载状态
const trendLoading = ref(false)
const typeLoading = ref(false)
const nodeLoading = ref(false)
const clusterLoading = ref(false)

// 数据
const trendData = ref([])
const typeData = ref([])
const nodeData = ref([])
const clusterData = ref([])

// 集群对比是否可见（多集群时显示）
const clusterChartVisible = computed(() => {
  return !props.clusterId && props.clusters.length > 1
})

// ==================== 异常趋势折线图 ====================
const trendChartOption = computed(() => {
  const dates = trendData.value.map(item => item.date)
  const activeCount = trendData.value.map(item => item.active_count || 0)
  const resolvedCount = trendData.value.map(item => item.resolved_count || 0)
  const totalCount = trendData.value.map(item => item.total_count || 0)

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985'
        }
      }
    },
    legend: {
      data: ['活跃异常', '已恢复', '总数'],
      bottom: 10
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: dates,
      axisLabel: {
        rotate: dates.length > 15 ? 45 : 0,
        interval: dates.length > 30 ? Math.floor(dates.length / 20) : 0
      }
    },
    yAxis: {
      type: 'value',
      name: '异常数量',
      minInterval: 1
    },
    series: [
      {
        name: '活跃异常',
        type: 'line',
        data: activeCount,
        smooth: true,
        itemStyle: { color: '#f56c6c' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(245, 108, 108, 0.3)' },
              { offset: 1, color: 'rgba(245, 108, 108, 0.05)' }
            ]
          }
        },
        emphasis: { focus: 'series' }
      },
      {
        name: '已恢复',
        type: 'line',
        data: resolvedCount,
        smooth: true,
        itemStyle: { color: '#67c23a' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(103, 194, 58, 0.3)' },
              { offset: 1, color: 'rgba(103, 194, 58, 0.05)' }
            ]
          }
        },
        emphasis: { focus: 'series' }
      },
      {
        name: '总数',
        type: 'line',
        data: totalCount,
        smooth: true,
        itemStyle: { color: '#409eff' },
        lineStyle: { width: 2, type: 'dashed' },
        emphasis: { focus: 'series' }
      }
    ],
    dataZoom: dates.length > 30 ? [
      {
        type: 'inside',
        start: 70,
        end: 100
      },
      {
        start: 70,
        end: 100,
        height: 20,
        bottom: 50
      }
    ] : []
  }
})

// ==================== 异常类型分布饼图 ====================
const typeChartOption = computed(() => {
  const data = typeData.value.map(item => ({
    name: formatAnomalyType(item.anomaly_type),
    value: item.total_count || 0
  }))

  return {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      textStyle: { fontSize: 12 }
    },
    series: [
      {
        name: '异常类型',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['40%', '50%'],
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
          }
        },
        labelLine: {
          show: true
        },
        data: data,
        color: ['#f56c6c', '#e6a23c', '#909399', '#67c23a', '#409eff']
      }
    ]
  }
})

// ==================== 节点异常排行榜 ====================
const nodeChartOption = computed(() => {
  // 从 trendData 中提取节点统计（这里简化处理，实际可能需要后端提供专门的节点排行接口）
  const nodeStats = {}
  
  // 模拟数据处理（实际应该从后端获取）
  trendData.value.forEach(item => {
    if (item.node_stats) {
      Object.entries(item.node_stats).forEach(([node, count]) => {
        nodeStats[node] = (nodeStats[node] || 0) + count
      })
    }
  })

  const sortedNodes = Object.entries(nodeStats)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)

  const nodes = sortedNodes.map(([node]) => node.length > 20 ? node.substring(0, 20) + '...' : node)
  const counts = sortedNodes.map(([, count]) => count)

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'value',
      name: '异常次数',
      minInterval: 1
    },
    yAxis: {
      type: 'category',
      data: nodes.reverse(),
      axisLabel: {
        interval: 0,
        fontSize: 11
      }
    },
    series: [
      {
        name: '异常次数',
        type: 'bar',
        data: counts.reverse(),
        itemStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 1, y2: 0,
            colorStops: [
              { offset: 0, color: '#409eff' },
              { offset: 1, color: '#67c23a' }
            ]
          },
          borderRadius: [0, 5, 5, 0]
        },
        label: {
          show: true,
          position: 'right',
          formatter: '{c}'
        },
        emphasis: {
          itemStyle: {
            color: '#f56c6c'
          }
        }
      }
    ]
  }
})

// ==================== 集群对比柱状图 ====================
const clusterChartOption = computed(() => {
  const clusterNames = clusterData.value.map(item => {
    const cluster = props.clusters.find(c => c.id === item.cluster_id)
    return cluster ? cluster.name : `集群 ${item.cluster_id}`
  })
  const activeCount = clusterData.value.map(item => item.active_count || 0)
  const resolvedCount = clusterData.value.map(item => item.resolved_count || 0)

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['活跃异常', '已恢复'],
      bottom: 10
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: clusterNames,
      axisLabel: {
        interval: 0,
        rotate: clusterNames.length > 5 ? 30 : 0
      }
    },
    yAxis: {
      type: 'value',
      name: '异常数量',
      minInterval: 1
    },
    series: [
      {
        name: '活跃异常',
        type: 'bar',
        data: activeCount,
        itemStyle: { color: '#f56c6c' },
        label: {
          show: true,
          position: 'top'
        }
      },
      {
        name: '已恢复',
        type: 'bar',
        data: resolvedCount,
        itemStyle: { color: '#67c23a' },
        label: {
          show: true,
          position: 'top'
        }
      }
    ]
  }
})

// ==================== 数据加载 ====================
const loadTrendData = async () => {
  trendLoading.value = true
  try {
    const response = await getAnomalyStatistics({
      cluster_id: props.clusterId,
      start_time: props.startTime,
      end_time: props.endTime,
      dimension: trendDimension.value
    })
    trendData.value = response.data.data || []
  } catch (error) {
    handleError(error, ErrorLevel.ERROR, { title: '加载趋势数据失败' })
    trendData.value = []
  } finally {
    trendLoading.value = false
  }
}

const loadTypeData = async () => {
  typeLoading.value = true
  try {
    const response = await getAnomalyTypeStatistics({
      cluster_id: props.clusterId,
      start_time: props.startTime,
      end_time: props.endTime
    })
    typeData.value = response.data.data || []
  } catch (error) {
    handleError(error, ErrorLevel.ERROR, { title: '加载类型分布数据失败' })
    typeData.value = []
  } finally {
    typeLoading.value = false
  }
}

const loadClusterData = async () => {
  if (!clusterChartVisible.value) return
  
  clusterLoading.value = true
  try {
    // 为每个集群获取统计数据
    const promises = props.clusters.map(cluster =>
      getAnomalyStatistics({
        cluster_id: cluster.id,
        start_time: props.startTime,
        end_time: props.endTime,
        dimension: 'total'
      })
    )
    
    const results = await Promise.all(promises)
    clusterData.value = results.map((response, index) => {
      const stats = response.data.data?.[0] || {}
      return {
        cluster_id: props.clusters[index].id,
        active_count: stats.active_count || 0,
        resolved_count: stats.resolved_count || 0
      }
    })
  } catch (error) {
    handleError(error, ErrorLevel.ERROR, { title: '加载集群对比数据失败' })
    clusterData.value = []
  } finally {
    clusterLoading.value = false
  }
}

const loadAllData = () => {
  loadTrendData()
  loadTypeData()
  loadClusterData()
}

// ==================== 事件处理 ====================
const handleDimensionChange = () => {
  loadTrendData()
}

const handleTrendClick = (params) => {
  if (params.componentType === 'series') {
    emit('dateClick', {
      date: params.name,
      dimension: trendDimension.value
    })
  }
}

const handleTypeClick = (params) => {
  if (params.componentType === 'series') {
    emit('typeClick', {
      type: params.data.name
    })
  }
}

// ==================== 工具函数 ====================
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

// ==================== 监听和生命周期 ====================
watch(() => [props.clusterId, props.startTime, props.endTime], () => {
  loadAllData()
}, { deep: true })

onMounted(() => {
  loadAllData()
})

// 暴露刷新方法
defineExpose({
  refresh: loadAllData
})
</script>

<style scoped>
.trend-charts {
  margin-top: 20px;
}

.chart-card {
  margin-bottom: 20px;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chart-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: bold;
}

.chart {
  width: 100%;
  height: 400px;
}

.small-chart {
  height: 350px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .chart {
    height: 300px;
  }
  
  .small-chart {
    height: 280px;
  }
  
  .chart-header {
    flex-direction: column;
    gap: 10px;
    align-items: flex-start;
  }
}
</style>

