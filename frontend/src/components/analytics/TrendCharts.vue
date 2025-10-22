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
                <el-icon><PieChartIcon /></el-icon>
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
            class="chart small-chart" 
            :option="nodeChartOption" 
            :autoresize="true"
          />
        </el-card>
      </el-col>
    </el-row>

    <!-- 集群对比柱状图 -->
    <el-card v-if="props.clusterId === null && clusters.length > 1" class="chart-card">
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
import { getStatistics, getTypeStatistics } from '@/api/anomaly'
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

// Props
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
  },
  anomalies: {
    type: Array,
    default: () => []
  }
})

// Emits
const emit = defineEmits(['date-click', 'type-click'])

// 数据
const trendDimension = ref('day')
const trendData = ref([])
const typeData = ref([])
const clusterData = ref([])

const trendLoading = ref(false)
const typeLoading = ref(false)
const clusterLoading = ref(false)

// 类型名称映射
const typeNameMap = {
  'NotReady': '节点未就绪',
  'MemoryPressure': '内存压力',
  'DiskPressure': '磁盘压力',
  'PIDPressure': 'PID压力',
  'NetworkUnavailable': '网络不可用'
}

// ==================== 加载数据 ====================
const loadTrendData = async () => {
  trendLoading.value = true
  try {
    const params = {
      start_time: props.startTime,
      end_time: props.endTime,
      dimension: trendDimension.value
    }
    if (props.clusterId) {
      params.cluster_id = props.clusterId
    }

    const response = await getStatistics(params)
    if (response.data && response.data.code === 200) {
      trendData.value = response.data.data || []
    }
  } catch (error) {
    console.error('Failed to load trend data:', error)
    handleError(error, ErrorLevel.WARNING)
  } finally {
    trendLoading.value = false
  }
}

const loadTypeData = async () => {
  console.log('Loading type statistics...')
  console.log('Params:', {
    cluster_id: props.clusterId,
    start_time: props.startTime,
    end_time: props.endTime
  })
  
  typeLoading.value = true
  try {
    const params = {
      start_time: props.startTime,
      end_time: props.endTime
    }
    if (props.clusterId) {
      params.cluster_id = props.clusterId
    }

    const response = await getTypeStatistics(params)
    console.log('Type statistics response:', response.data)
    
    if (response.data && response.data.code === 200) {
      typeData.value = response.data.data || []
      console.log('Type statistics data:', typeData.value)
      console.log('Mapped data for chart:', typeData.value.map(item => ({
        type: item.anomaly_type,
        mapped_name: typeNameMap[item.anomaly_type] || item.anomaly_type,
        count: item.total_count
      })))
    } else {
      console.warn('Invalid type statistics response:', response)
    }
  } catch (error) {
    console.error('Failed to load type data:', error)
    handleError(error, ErrorLevel.WARNING)
  } finally {
    typeLoading.value = false
  }
}

const loadClusterData = async () => {
  if (props.clusterId !== null || props.clusters.length === 0) {
    console.log('Skip cluster comparison:', props.clusterId !== null ? 'Single cluster view' : 'No clusters')
    clusterData.value = []
    return
  }

  console.log('Loading cluster comparison data for', props.clusters.length, 'clusters')
  console.log('Time range:', props.startTime, 'to', props.endTime)
  
  clusterLoading.value = true
  try {
    // 为每个集群获取统计数据
    const promises = props.clusters.map(cluster => {
      console.log(`Fetching data for cluster: ${cluster.name} (ID: ${cluster.id})`)
      return getStatistics({
        cluster_id: cluster.id,
        start_time: props.startTime,
        end_time: props.endTime,
        dimension: 'day'
      }).then(response => {
        console.log(`Response for cluster ${cluster.name}:`, response.data)
        return response
      }).catch(err => {
        console.error(`Failed to load cluster ${cluster.name} (ID: ${cluster.id}) data:`, err)
        return { data: { code: 500, data: [] } }
      })
    })

    const responses = await Promise.all(promises)
    console.log('All responses received:', responses.length)
    
    // 聚合每个集群的总异常数
    const results = props.clusters.map((cluster, index) => {
      const response = responses[index]
      let totalCount = 0
      
      if (response.data && response.data.code === 200) {
        const data = response.data.data || []
        console.log(`Cluster ${cluster.name} has ${data.length} days of data`)
        
        // 累加所有天的异常数
        totalCount = data.reduce((sum, item) => {
          const count = item.total_count || 0
          console.log(`  - ${item.date}: ${count} anomalies`)
          return sum + count
        }, 0)
        
        console.log(`Cluster ${cluster.name} total: ${totalCount}`)
      } else {
        console.warn(`Invalid response for cluster ${cluster.name}:`, response)
      }
      
      return {
        cluster_id: cluster.id,
        cluster_name: cluster.name,
        total_count: totalCount
      }
    })
    
    console.log('All results before filtering:', results)
    
    // 只保留有数据的集群
    clusterData.value = results.filter(item => item.total_count > 0)
    
    // 如果所有集群都没有数据，保留所有集群但数量为0
    if (clusterData.value.length === 0) {
      console.warn('No clusters with data, showing all clusters with 0 count')
      clusterData.value = results
    }
    
    console.log('Final cluster comparison data:', clusterData.value)
  } catch (error) {
    console.error('Failed to load cluster data (outer catch):', error)
    handleError(error, ErrorLevel.WARNING)
    clusterData.value = []
  } finally {
    clusterLoading.value = false
  }
}

// ==================== 异常趋势折线图 ====================
const trendChartOption = computed(() => {
  const dates = trendData.value.map(item => item.date)
  const totals = trendData.value.map(item => item.total_count || 0)
  const actives = trendData.value.map(item => item.active_count || 0)
  const resolveds = trendData.value.map(item => item.resolved_count || 0)

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985'
        }
      },
      formatter: function(params) {
        // 只显示日期，不显示时间
        let date = params[0].axisValue
        if (date && date.includes('T')) {
          date = date.split('T')[0]
        }
        
        let result = `${date}<br/>`
        params.forEach(param => {
          result += `${param.marker} ${param.seriesName}: ${param.value}<br/>`
        })
        return result
      }
    },
    legend: {
      data: ['总异常数', '活跃异常', '已恢复'],
      bottom: 10
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      top: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: dates,
      axisLabel: {
        rotate: dates.length > 10 ? 45 : 0,
        formatter: function(value) {
          // 只显示日期，不显示时间
          if (value && value.includes('T')) {
            return value.split('T')[0]
          }
          return value
        }
      }
    },
    yAxis: {
      type: 'value',
      name: '异常次数',
      minInterval: 1
    },
    series: [
      {
        name: '总异常数',
        type: 'line',
        data: totals,
        smooth: true,
        itemStyle: { color: '#409eff' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(64, 158, 255, 0.3)' },
              { offset: 1, color: 'rgba(64, 158, 255, 0.05)' }
            ]
          }
        }
      },
      {
        name: '活跃异常',
        type: 'line',
        data: actives,
        smooth: true,
        itemStyle: { color: '#f56c6c' }
      },
      {
        name: '已恢复',
        type: 'line',
        data: resolveds,
        smooth: true,
        itemStyle: { color: '#67c23a' }
      }
    ],
    dataZoom: dates.length > 10 ? [
      {
        type: 'slider',
        start: 0,
        end: 100,
        height: 20,
        bottom: 40
      }
    ] : []
  }
})

// ==================== 异常类型分布饼图 ====================
const typeChartOption = computed(() => {
  const data = typeData.value.map(item => ({
    name: typeNameMap[item.anomaly_type] || item.anomaly_type,
    value: item.total_count || 0
  }))

  return {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      data: data.map(item => item.name)
    },
    series: [
      {
        name: '异常类型',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: true,
          formatter: '{b}: {c}'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 16,
            fontWeight: 'bold'
          }
        },
        data: data,
        color: ['#f56c6c', '#e6a23c', '#909399', '#67c23a', '#409eff']
      }
    ]
  }
})

// ==================== 节点异常排行榜 ====================
const nodeChartOption = computed(() => {
  // 从 anomalies prop 中聚合节点统计
  const nodeStats = {}
  
  props.anomalies.forEach(anomaly => {
    const nodeName = anomaly.node_name
    if (nodeName) {
      nodeStats[nodeName] = (nodeStats[nodeName] || 0) + 1
    }
  })

  const sortedNodes = Object.entries(nodeStats)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)

  if (sortedNodes.length === 0) {
    // 空数据处理
    return {
      title: {
        text: '暂无数据',
        left: 'center',
        top: 'center',
        textStyle: {
          color: '#909399',
          fontSize: 14
        }
      }
    }
  }

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
  if (clusterData.value.length === 0) {
    return {
      title: {
        text: '暂无数据',
        left: 'center',
        top: 'center',
        textStyle: {
          color: '#909399',
          fontSize: 14
        }
      }
    }
  }

  const clusterNames = clusterData.value.map(item => item.cluster_name || `集群 ${item.cluster_id}`)
  const counts = clusterData.value.map(item => item.total_count || 0)

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
      type: 'category',
      data: clusterNames,
      axisLabel: {
        interval: 0,
        rotate: 30
      }
    },
    yAxis: {
      type: 'value',
      name: '异常总数',
      minInterval: 1
    },
    series: [
      {
        name: '异常总数',
        type: 'bar',
        data: counts,
        itemStyle: {
          color: {
            type: 'linear',
            x: 0, y: 1, x2: 0, y2: 0,
            colorStops: [
              { offset: 0, color: '#409eff' },
              { offset: 1, color: '#67c23a' }
            ]
          },
          borderRadius: [5, 5, 0, 0]
        },
        label: {
          show: true,
          position: 'top',
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

// ==================== 事件处理 ====================
const handleDimensionChange = () => {
  loadTrendData()
}

const handleTrendClick = (params) => {
  if (params.componentType === 'series') {
    emit('date-click', {
      date: params.name,
      dimension: trendDimension.value
    })
  }
}

const handleTypeClick = (params) => {
  if (params.componentType === 'series') {
    emit('type-click', {
      type: params.name
    })
  }
}

// ==================== 刷新方法 ====================
const refresh = () => {
  loadTrendData()
  loadTypeData()
  loadClusterData()
}

// 暴露方法给父组件
defineExpose({
  refresh
})

// ==================== 监听 props 变化 ====================
watch(() => [props.clusterId, props.startTime, props.endTime], () => {
  loadTrendData()
  loadTypeData()
  loadClusterData()
}, { immediate: false })

// ==================== 初始化 ====================
onMounted(() => {
  loadTrendData()
  loadTypeData()
  loadClusterData()
})
</script>

<style scoped>
.trend-charts {
  width: 100%;
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
  font-weight: 600;
}

.chart {
  width: 100%;
  height: 400px;
}

.small-chart {
  height: 350px;
}
</style>
