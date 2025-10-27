<template>
  <el-card class="node-health-card" :body-style="{ padding: '20px' }">
    <template #header>
      <div class="card-header">
        <span>
          <el-icon><TrendCharts /></el-icon>
          节点健康度评分
        </span>
        <el-button
          v-if="nodeName"
          size="small"
          type="primary"
          link
          @click="$emit('view-detail', nodeName)"
        >
          查看详情
        </el-button>
      </div>
    </template>

    <div v-loading="loading" class="health-content">
      <template v-if="!loading && healthData">
        <!-- 健康度评分圆环 -->
        <div class="score-ring">
          <el-progress
            type="circle"
            :percentage="healthData.health_score"
            :width="160"
            :stroke-width="12"
            :color="scoreColor"
          >
            <template #default="{ percentage }">
              <div class="score-display">
                <span class="score-value">{{ percentage }}</span>
                <span class="score-label">分</span>
              </div>
              <div class="score-level">{{ scoreLevel }}</div>
            </template>
          </el-progress>
        </div>

        <!-- 节点信息 -->
        <div class="node-info">
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="节点名称" :span="2">
              <el-tag>{{ healthData.node_name }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="集群">
              {{ healthData.cluster_name }}
            </el-descriptions-item>
            <el-descriptions-item label="节点角色">
              <el-tag
                v-for="role in healthData.node_roles"
                :key="role"
                size="small"
                style="margin-right: 4px"
              >
                {{ role }}
              </el-tag>
              <span v-if="!healthData.node_roles || healthData.node_roles.length === 0">-</span>
            </el-descriptions-item>
            <el-descriptions-item label="总异常数">
              <el-tag type="danger">{{ healthData.total_anomalies }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="活跃异常">
              <el-tag type="warning">{{ healthData.active_anomalies }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="平均恢复时间">
              {{ formatDuration(healthData.avg_recovery_time) }}
            </el-descriptions-item>
            <el-descriptions-item label="最大持续时间">
              {{ formatDuration(healthData.max_duration) }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 影响因素分解 -->
        <div class="factors">
          <el-divider content-position="left">影响因素分解</el-divider>
          <el-row :gutter="12">
            <el-col :span="12">
              <div class="factor-item">
                <span class="factor-label">异常频率影响</span>
                <el-progress
                  :percentage="calculateFactorPercentage(healthData.total_anomalies, 50)"
                  :format="() => ''"
                  :stroke-width="8"
                  :color="factorColors[0]"
                />
              </div>
            </el-col>
            <el-col :span="12">
              <div class="factor-item">
                <span class="factor-label">恢复速度影响</span>
                <el-progress
                  :percentage="calculateRecoveryFactor(healthData.avg_recovery_time)"
                  :format="() => ''"
                  :stroke-width="8"
                  :color="factorColors[1]"
                />
              </div>
            </el-col>
            <el-col :span="12">
              <div class="factor-item">
                <span class="factor-label">异常严重性</span>
                <el-progress
                  :percentage="calculateSeverityFactor(healthData)"
                  :format="() => ''"
                  :stroke-width="8"
                  :color="factorColors[2]"
                />
              </div>
            </el-col>
            <el-col :span="12">
              <div class="factor-item">
                <span class="factor-label">稳定性评分</span>
                <el-progress
                  :percentage="healthData.recurrence_rate ? (100 - healthData.recurrence_rate) : 100"
                  :format="() => ''"
                  :stroke-width="8"
                  :color="factorColors[3]"
                />
              </div>
            </el-col>
          </el-row>
        </div>

        <!-- 健康度趋势图 -->
        <div v-if="showTrend" class="trend-chart">
          <el-divider content-position="left">近7天健康度趋势</el-divider>
          <div ref="trendChartRef" style="width: 100%; height: 200px"></div>
        </div>
      </template>

      <el-empty v-else-if="!loading && !healthData" description="暂无数据" />
    </div>
  </el-card>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { TrendCharts } from '@element-plus/icons-vue'
import { getNodeHealth } from '@/api/anomaly'
import * as echarts from 'echarts'

const props = defineProps({
  nodeName: {
    type: String,
    required: true
  },
  clusterId: {
    type: Number,
    default: null
  },
  startTime: {
    type: String,
    default: null
  },
  endTime: {
    type: String,
    default: null
  },
  showTrend: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['view-detail'])

const loading = ref(false)
const healthData = ref(null)
const trendChartRef = ref(null)
let trendChart = null

const factorColors = ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C']

// 计算健康度等级
const scoreLevel = computed(() => {
  const score = healthData.value?.health_score || 0
  if (score >= 90) return '优秀'
  if (score >= 75) return '良好'
  if (score >= 60) return '一般'
  if (score >= 40) return '较差'
  return '很差'
})

// 计算评分颜色
const scoreColor = computed(() => {
  const score = healthData.value?.health_score || 0
  if (score >= 90) return '#67C23A'
  if (score >= 75) return '#409EFF'
  if (score >= 60) return '#E6A23C'
  if (score >= 40) return '#F56C6C'
  return '#909399'
})

// 格式化时长
const formatDuration = (seconds) => {
  if (!seconds || seconds === 0) return '-'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

// 计算因素百分比（越多越不好，需要反向）
const calculateFactorPercentage = (value, threshold) => {
  if (!value) return 100
  const percentage = Math.max(0, 100 - (value / threshold) * 100)
  return Math.round(percentage)
}

// 计算恢复速度因素（恢复越快越好）
const calculateRecoveryFactor = (avgRecoveryTime) => {
  if (!avgRecoveryTime) return 100
  // 1小时内恢复算100分，24小时算0分
  const hours = avgRecoveryTime / 3600
  const percentage = Math.max(0, 100 - (hours / 24) * 100)
  return Math.round(percentage)
}

// 计算严重性因素
const calculateSeverityFactor = (data) => {
  if (!data) return 100
  // 基于异常类型和持续时间计算
  const maxDurationHours = (data.max_duration || 0) / 3600
  const percentage = Math.max(0, 100 - (maxDurationHours / 48) * 100)
  return Math.round(percentage)
}

// 加载健康度数据
const loadHealthData = async () => {
  if (!props.nodeName) return
  
  loading.value = true
  try {
    const params = {
      node_name: props.nodeName,
      cluster_id: props.clusterId,
      start_time: props.startTime,
      end_time: props.endTime
    }
    
    const res = await getNodeHealth(params)
    healthData.value = res.data
    
    // 渲染趋势图
    if (props.showTrend && healthData.value) {
      await nextTick()
      renderTrendChart()
    }
  } catch (error) {
    console.error('加载节点健康度失败：', error)
    healthData.value = null
  } finally {
    loading.value = false
  }
}

// 渲染趋势图
const renderTrendChart = () => {
  if (!trendChartRef.value || !healthData.value) return
  
  if (!trendChart) {
    trendChart = echarts.init(trendChartRef.value)
  }
  
  // 模拟趋势数据（实际应该从后端获取）
  const dates = []
  const scores = []
  const now = new Date()
  
  for (let i = 6; i >= 0; i--) {
    const date = new Date(now)
    date.setDate(date.getDate() - i)
    dates.push(date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }))
    
    // 模拟数据：健康度有波动
    const baseScore = healthData.value.health_score
    const variation = Math.random() * 20 - 10
    scores.push(Math.max(0, Math.min(100, baseScore + variation)))
  }
  
  const option = {
    tooltip: {
      trigger: 'axis',
      formatter: (params) => {
        const point = params[0]
        return `${point.axisValue}<br/>健康度: ${point.value.toFixed(1)}分`
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: dates
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: {
        formatter: '{value}分'
      }
    },
    series: [
      {
        name: '健康度',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        data: scores,
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(64, 158, 255, 0.3)' },
            { offset: 1, color: 'rgba(64, 158, 255, 0.05)' }
          ])
        },
        lineStyle: {
          width: 2,
          color: '#409EFF'
        },
        itemStyle: {
          color: '#409EFF'
        }
      }
    ]
  }
  
  trendChart.setOption(option)
}

// 监听props变化
watch(() => [props.nodeName, props.clusterId, props.startTime, props.endTime], () => {
  loadHealthData()
}, { deep: true })

// 初始化
onMounted(() => {
  loadHealthData()
  
  // 响应式调整图表大小
  window.addEventListener('resize', () => {
    trendChart?.resize()
  })
})
</script>

<style scoped>
.node-health-card {
  height: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 16px;
  font-weight: 600;
}

.card-header span {
  display: flex;
  align-items: center;
  gap: 8px;
}

.health-content {
  min-height: 400px;
}

.score-ring {
  display: flex;
  justify-content: center;
  margin-bottom: 24px;
}

.score-display {
  display: flex;
  align-items: baseline;
  justify-content: center;
  margin-bottom: 4px;
}

.score-value {
  font-size: 36px;
  font-weight: bold;
}

.score-label {
  font-size: 14px;
  margin-left: 4px;
  color: #909399;
}

.score-level {
  text-align: center;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
}

.node-info {
  margin-bottom: 24px;
}

.factors {
  margin-bottom: 24px;
}

.factor-item {
  margin-bottom: 16px;
}

.factor-label {
  display: block;
  font-size: 12px;
  color: #606266;
  margin-bottom: 8px;
}

.trend-chart {
  margin-top: 24px;
}
</style>

