<template>
  <el-dialog
    v-model="dialogVisible"
    title="对比分析"
    width="90%"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <!-- 时间段对比 -->
      <el-tab-pane label="时间段对比" name="time">
        <div class="compare-form">
          <el-form :inline="true">
            <el-form-item label="集群">
              <el-select v-model="timeCompare.clusterId" placeholder="选择集群" clearable style="width: 200px">
                <el-option
                  v-for="cluster in clusters"
                  :key="cluster.id"
                  :label="cluster.name"
                  :value="cluster.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="时间段 A">
              <el-date-picker
                v-model="timeCompare.periodA"
                type="daterange"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                value-format="YYYY-MM-DD"
              />
            </el-form-item>
            <el-form-item label="时间段 B">
              <el-date-picker
                v-model="timeCompare.periodB"
                type="daterange"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                value-format="YYYY-MM-DD"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :icon="Search" :loading="timeCompareLoading" @click="handleTimeCompare">
                对比分析
              </el-button>
            </el-form-item>
          </el-form>
        </div>

        <div v-if="timeCompareResult" class="compare-result">
          <v-chart class="chart" :option="timeCompareChartOption" :autoresize="true" />
          
          <el-row :gutter="20" style="margin-top: 20px">
            <el-col :span="12">
              <el-card>
                <template #header>时间段 A 统计</template>
                <el-descriptions :column="2" border>
                  <el-descriptions-item label="时间范围">
                    {{ formatDateRange(timeCompare.periodA) }}
                  </el-descriptions-item>
                  <el-descriptions-item label="总异常数">
                    <el-tag type="danger">{{ timeCompareResult.periodA.total }}</el-tag>
                  </el-descriptions-item>
                  <el-descriptions-item label="活跃异常">
                    {{ timeCompareResult.periodA.active }}
                  </el-descriptions-item>
                  <el-descriptions-item label="已恢复">
                    {{ timeCompareResult.periodA.resolved }}
                  </el-descriptions-item>
                  <el-descriptions-item label="平均持续时间">
                    {{ formatDuration(timeCompareResult.periodA.avg_duration) }}
                  </el-descriptions-item>
                  <el-descriptions-item label="受影响节点">
                    {{ timeCompareResult.periodA.affected_nodes }}
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card>
                <template #header>时间段 B 统计</template>
                <el-descriptions :column="2" border>
                  <el-descriptions-item label="时间范围">
                    {{ formatDateRange(timeCompare.periodB) }}
                  </el-descriptions-item>
                  <el-descriptions-item label="总异常数">
                    <el-tag type="danger">{{ timeCompareResult.periodB.total }}</el-tag>
                  </el-descriptions-item>
                  <el-descriptions-item label="活跃异常">
                    {{ timeCompareResult.periodB.active }}
                  </el-descriptions-item>
                  <el-descriptions-item label="已恢复">
                    {{ timeCompareResult.periodB.resolved }}
                  </el-descriptions-item>
                  <el-descriptions-item label="平均持续时间">
                    {{ formatDuration(timeCompareResult.periodB.avg_duration) }}
                  </el-descriptions-item>
                  <el-descriptions-item label="受影响节点">
                    {{ timeCompareResult.periodB.affected_nodes }}
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
            </el-col>
          </el-row>

          <el-card style="margin-top: 20px">
            <template #header>对比分析结论</template>
            <el-alert
              v-for="(conclusion, index) in timeCompareConclusions"
              :key="index"
              :title="conclusion.title"
              :type="conclusion.type"
              :description="conclusion.description"
              show-icon
              style="margin-bottom: 10px"
            />
          </el-card>
        </div>
      </el-tab-pane>

      <!-- 集群对比 -->
      <el-tab-pane label="集群对比" name="cluster">
        <div class="compare-form">
          <el-form :inline="true">
            <el-form-item label="集群 A">
              <el-select v-model="clusterCompare.clusterA" placeholder="选择集群 A" style="width: 200px">
                <el-option
                  v-for="cluster in clusters"
                  :key="cluster.id"
                  :label="cluster.name"
                  :value="cluster.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="集群 B">
              <el-select v-model="clusterCompare.clusterB" placeholder="选择集群 B" style="width: 200px">
                <el-option
                  v-for="cluster in clusters"
                  :key="cluster.id"
                  :label="cluster.name"
                  :value="cluster.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="时间范围">
              <el-date-picker
                v-model="clusterCompare.dateRange"
                type="daterange"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                value-format="YYYY-MM-DD"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :icon="Search" :loading="clusterCompareLoading" @click="handleClusterCompare">
                对比分析
              </el-button>
            </el-form-item>
          </el-form>
        </div>

        <div v-if="clusterCompareResult" class="compare-result">
          <v-chart class="chart" :option="clusterCompareChartOption" :autoresize="true" />
          
          <el-row :gutter="20" style="margin-top: 20px">
            <el-col :span="12">
              <el-card>
                <template #header>
                  集群 A: {{ getClusterName(clusterCompare.clusterA) }}
                </template>
                <el-descriptions :column="2" border>
                  <el-descriptions-item label="总异常数">
                    <el-tag type="danger">{{ clusterCompareResult.clusterA.total }}</el-tag>
                  </el-descriptions-item>
                  <el-descriptions-item label="活跃异常">
                    {{ clusterCompareResult.clusterA.active }}
                  </el-descriptions-item>
                  <el-descriptions-item label="已恢复">
                    {{ clusterCompareResult.clusterA.resolved }}
                  </el-descriptions-item>
                  <el-descriptions-item label="节点数">
                    {{ clusterCompareResult.clusterA.node_count }}
                  </el-descriptions-item>
                  <el-descriptions-item label="异常率">
                    {{ calculateAnomalyRate(clusterCompareResult.clusterA) }}
                  </el-descriptions-item>
                  <el-descriptions-item label="最常见异常">
                    {{ clusterCompareResult.clusterA.most_common_type || '-' }}
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card>
                <template #header>
                  集群 B: {{ getClusterName(clusterCompare.clusterB) }}
                </template>
                <el-descriptions :column="2" border>
                  <el-descriptions-item label="总异常数">
                    <el-tag type="danger">{{ clusterCompareResult.clusterB.total }}</el-tag>
                  </el-descriptions-item>
                  <el-descriptions-item label="活跃异常">
                    {{ clusterCompareResult.clusterB.active }}
                  </el-descriptions-item>
                  <el-descriptions-item label="已恢复">
                    {{ clusterCompareResult.clusterB.resolved }}
                  </el-descriptions-item>
                  <el-descriptions-item label="节点数">
                    {{ clusterCompareResult.clusterB.node_count }}
                  </el-descriptions-item>
                  <el-descriptions-item label="异常率">
                    {{ calculateAnomalyRate(clusterCompareResult.clusterB) }}
                  </el-descriptions-item>
                  <el-descriptions-item label="最常见异常">
                    {{ clusterCompareResult.clusterB.most_common_type || '-' }}
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
            </el-col>
          </el-row>
        </div>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart, RadarChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent
} from 'echarts/components'
import { Search } from '@element-plus/icons-vue'
import { getAnomalies, getAnomalyStatistics } from '@/api/anomaly'
import { handleError, ErrorLevel } from '@/utils/errorHandler'

// 注册 ECharts 组件
use([
  CanvasRenderer,
  BarChart,
  LineChart,
  RadarChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent
])

const props = defineProps({
  clusters: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['close'])

const dialogVisible = ref(false)
const activeTab = ref('time')

// 时间段对比
const timeCompare = ref({
  clusterId: null,
  periodA: [],
  periodB: []
})
const timeCompareLoading = ref(false)
const timeCompareResult = ref(null)

// 集群对比
const clusterCompare = ref({
  clusterA: null,
  clusterB: null,
  dateRange: []
})
const clusterCompareLoading = ref(false)
const clusterCompareResult = ref(null)

// 时间段对比图表
const timeCompareChartOption = computed(() => {
  if (!timeCompareResult.value) return {}

  const categories = ['总异常', '活跃异常', '已恢复', '受影响节点']
  const dataA = [
    timeCompareResult.value.periodA.total,
    timeCompareResult.value.periodA.active,
    timeCompareResult.value.periodA.resolved,
    timeCompareResult.value.periodA.affected_nodes
  ]
  const dataB = [
    timeCompareResult.value.periodB.total,
    timeCompareResult.value.periodB.active,
    timeCompareResult.value.periodB.resolved,
    timeCompareResult.value.periodB.affected_nodes
  ]

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['时间段 A', '时间段 B'],
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
      data: categories
    },
    yAxis: {
      type: 'value',
      minInterval: 1
    },
    series: [
      {
        name: '时间段 A',
        type: 'bar',
        data: dataA,
        itemStyle: { color: '#409eff' },
        label: {
          show: true,
          position: 'top'
        }
      },
      {
        name: '时间段 B',
        type: 'bar',
        data: dataB,
        itemStyle: { color: '#67c23a' },
        label: {
          show: true,
          position: 'top'
        }
      }
    ]
  }
})

// 集群对比图表
const clusterCompareChartOption = computed(() => {
  if (!clusterCompareResult.value) return {}

  const categories = ['总异常', '活跃异常', '已恢复', '节点数']
  const dataA = [
    clusterCompareResult.value.clusterA.total,
    clusterCompareResult.value.clusterA.active,
    clusterCompareResult.value.clusterA.resolved,
    clusterCompareResult.value.clusterA.node_count
  ]
  const dataB = [
    clusterCompareResult.value.clusterB.total,
    clusterCompareResult.value.clusterB.active,
    clusterCompareResult.value.clusterB.resolved,
    clusterCompareResult.value.clusterB.node_count
  ]

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: [
        `集群 A: ${getClusterName(clusterCompare.value.clusterA)}`,
        `集群 B: ${getClusterName(clusterCompare.value.clusterB)}`
      ],
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
      data: categories
    },
    yAxis: {
      type: 'value',
      minInterval: 1
    },
    series: [
      {
        name: `集群 A: ${getClusterName(clusterCompare.value.clusterA)}`,
        type: 'bar',
        data: dataA,
        itemStyle: { color: '#409eff' },
        label: {
          show: true,
          position: 'top'
        }
      },
      {
        name: `集群 B: ${getClusterName(clusterCompare.value.clusterB)}`,
        type: 'bar',
        data: dataB,
        itemStyle: { color: '#67c23a' },
        label: {
          show: true,
          position: 'top'
        }
      }
    ]
  }
})

// 时间段对比结论
const timeCompareConclusions = computed(() => {
  if (!timeCompareResult.value) return []

  const conclusions = []
  const a = timeCompareResult.value.periodA
  const b = timeCompareResult.value.periodB

  // 总异常数对比
  const totalDiff = ((b.total - a.total) / (a.total || 1) * 100).toFixed(1)
  if (Math.abs(totalDiff) > 10) {
    conclusions.push({
      title: '异常总数变化明显',
      type: totalDiff > 0 ? 'warning' : 'success',
      description: `时间段 B 比时间段 A 异常总数${totalDiff > 0 ? '增加' : '减少'}了 ${Math.abs(totalDiff)}%`
    })
  }

  // 平均持续时间对比
  const durationDiff = b.avg_duration - a.avg_duration
  if (Math.abs(durationDiff) > 60) {
    conclusions.push({
      title: '异常持续时间变化',
      type: durationDiff > 0 ? 'warning' : 'success',
      description: `时间段 B 的平均异常持续时间比时间段 A ${durationDiff > 0 ? '延长' : '缩短'}了 ${formatDuration(Math.abs(durationDiff))}`
    })
  }

  // 活跃异常对比
  if (b.active > a.active * 1.5) {
    conclusions.push({
      title: '活跃异常激增',
      type: 'error',
      description: `时间段 B 的活跃异常数是时间段 A 的 ${(b.active / (a.active || 1)).toFixed(1)} 倍，需要关注`
    })
  }

  if (conclusions.length === 0) {
    conclusions.push({
      title: '整体稳定',
      type: 'success',
      description: '两个时间段的异常情况差异不大，集群运行整体稳定'
    })
  }

  return conclusions
})

// 获取集群名称
const getClusterName = (clusterId) => {
  const cluster = props.clusters.find(c => c.id === clusterId)
  return cluster ? cluster.name : `集群 ${clusterId}`
}

// 计算异常率
const calculateAnomalyRate = (data) => {
  if (!data.node_count || data.node_count === 0) return '0%'
  return ((data.total / data.node_count) * 100).toFixed(1) + '%'
}

// 格式化日期范围
const formatDateRange = (range) => {
  if (!range || range.length !== 2) return '-'
  return `${range[0]} ~ ${range[1]}`
}

// 格式化持续时间
const formatDuration = (seconds) => {
  if (!seconds) return '0秒'
  
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

// 时间段对比分析
const handleTimeCompare = async () => {
  if (!timeCompare.value.periodA || timeCompare.value.periodA.length !== 2) {
    handleError('请选择时间段 A', ErrorLevel.WARNING)
    return
  }
  if (!timeCompare.value.periodB || timeCompare.value.periodB.length !== 2) {
    handleError('请选择时间段 B', ErrorLevel.WARNING)
    return
  }

  timeCompareLoading.value = true
  try {
    // 获取时间段 A 的数据
    const responseA = await getAnomalies({
      cluster_id: timeCompare.value.clusterId,
      start_time: new Date(timeCompare.value.periodA[0]).toISOString(),
      end_time: new Date(timeCompare.value.periodA[1]).toISOString(),
      page: 1,
      page_size: 1000
    })

    // 获取时间段 B 的数据
    const responseB = await getAnomalies({
      cluster_id: timeCompare.value.clusterId,
      start_time: new Date(timeCompare.value.periodB[0]).toISOString(),
      end_time: new Date(timeCompare.value.periodB[1]).toISOString(),
      page: 1,
      page_size: 1000
    })

    const dataA = responseA.data.data?.items || []
    const dataB = responseB.data.data?.items || []

    // 计算统计数据
    timeCompareResult.value = {
      periodA: calculatePeriodStats(dataA),
      periodB: calculatePeriodStats(dataB)
    }
  } catch (error) {
    handleError(error, ErrorLevel.ERROR, { title: '对比分析失败' })
  } finally {
    timeCompareLoading.value = false
  }
}

// 计算时间段统计
const calculatePeriodStats = (data) => {
  const active = data.filter(item => item.status === 'Active').length
  const resolved = data.filter(item => item.status === 'Resolved').length
  const affectedNodes = new Set(data.map(item => item.node_name)).size
  const totalDuration = data.reduce((sum, item) => sum + (item.duration || 0), 0)
  const avgDuration = data.length > 0 ? Math.floor(totalDuration / data.length) : 0

  return {
    total: data.length,
    active,
    resolved,
    affected_nodes: affectedNodes,
    avg_duration: avgDuration
  }
}

// 集群对比分析
const handleClusterCompare = async () => {
  if (!clusterCompare.value.clusterA) {
    handleError('请选择集群 A', ErrorLevel.WARNING)
    return
  }
  if (!clusterCompare.value.clusterB) {
    handleError('请选择集群 B', ErrorLevel.WARNING)
    return
  }
  if (clusterCompare.value.clusterA === clusterCompare.value.clusterB) {
    handleError('请选择不同的集群进行对比', ErrorLevel.WARNING)
    return
  }
  if (!clusterCompare.value.dateRange || clusterCompare.value.dateRange.length !== 2) {
    handleError('请选择时间范围', ErrorLevel.WARNING)
    return
  }

  clusterCompareLoading.value = true
  try {
    // 获取集群 A 的数据
    const responseA = await getAnomalies({
      cluster_id: clusterCompare.value.clusterA,
      start_time: new Date(clusterCompare.value.dateRange[0]).toISOString(),
      end_time: new Date(clusterCompare.value.dateRange[1]).toISOString(),
      page: 1,
      page_size: 1000
    })

    // 获取集群 B 的数据
    const responseB = await getAnomalies({
      cluster_id: clusterCompare.value.clusterB,
      start_time: new Date(clusterCompare.value.dateRange[0]).toISOString(),
      end_time: new Date(clusterCompare.value.dateRange[1]).toISOString(),
      page: 1,
      page_size: 1000
    })

    const dataA = responseA.data.data?.items || []
    const dataB = responseB.data.data?.items || []

    // 计算统计数据
    clusterCompareResult.value = {
      clusterA: calculateClusterStats(dataA),
      clusterB: calculateClusterStats(dataB)
    }
  } catch (error) {
    handleError(error, ErrorLevel.ERROR, { title: '集群对比失败' })
  } finally {
    clusterCompareLoading.value = false
  }
}

// 计算集群统计
const calculateClusterStats = (data) => {
  const active = data.filter(item => item.status === 'Active').length
  const resolved = data.filter(item => item.status === 'Resolved').length
  const nodeCount = new Set(data.map(item => item.node_name)).size
  
  // 统计最常见的异常类型
  const typeCount = {}
  data.forEach(item => {
    typeCount[item.anomaly_type] = (typeCount[item.anomaly_type] || 0) + 1
  })
  const mostCommonType = Object.keys(typeCount).length > 0
    ? Object.keys(typeCount).reduce((a, b) => typeCount[a] > typeCount[b] ? a : b)
    : null

  return {
    total: data.length,
    active,
    resolved,
    node_count: nodeCount,
    most_common_type: mostCommonType
  }
}

// 标签切换
const handleTabChange = () => {
  // 清空之前的结果
  timeCompareResult.value = null
  clusterCompareResult.value = null
}

// 打开对话框
const open = () => {
  dialogVisible.value = true
  activeTab.value = 'time'
  timeCompareResult.value = null
  clusterCompareResult.value = null
}

// 关闭对话框
const handleClose = () => {
  dialogVisible.value = false
  emit('close')
}

defineExpose({
  open
})
</script>

<style scoped>
.compare-form {
  padding: 20px;
  background-color: #f5f7fa;
  border-radius: 4px;
  margin-bottom: 20px;
}

.compare-result {
  padding: 20px 0;
}

.chart {
  width: 100%;
  height: 400px;
}
</style>

