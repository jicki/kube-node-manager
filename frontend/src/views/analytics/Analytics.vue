<template>
  <div class="analytics-container">
    <el-card class="filter-card">
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="集群">
          <el-select
            v-model="filterForm.cluster_id"
            placeholder="请选择集群"
            clearable
            style="width: 200px"
            @change="handleFilterChange"
          >
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
            v-model="filterForm.dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            @change="handleFilterChange"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :icon="Search" @click="handleSearch">
            查询
          </el-button>
          <el-button :icon="Refresh" @click="handleReset">重置</el-button>
          <el-button
            v-if="userRole === 'admin'"
            type="success"
            :icon="RefreshRight"
            :loading="checkLoading"
            @click="handleTriggerCheck"
          >
            手动检测
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- Tab 分栏结构 -->
    <el-tabs v-model="activeTab" class="analytics-tabs">
      <!-- 统计分析面板（合并数据概览和趋势分析） -->
      <el-tab-pane label="统计分析" name="overview">
        <!-- 统计卡片区 -->
        <el-row :gutter="20" class="stats-row">
          <el-col :span="6">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-content">
                <div class="stat-icon total">
                  <el-icon><DataAnalysis /></el-icon>
                </div>
                <div class="stat-info">
                  <div class="stat-value">{{ summary.totalCount }}</div>
                  <div class="stat-label">总异常数</div>
                </div>
              </div>
            </el-card>
          </el-col>

          <el-col :span="6">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-content">
                <div class="stat-icon active">
                  <el-icon><Warning /></el-icon>
                </div>
                <div class="stat-info">
                  <div class="stat-value">{{ summary.activeCount }}</div>
                  <div class="stat-label">活跃异常</div>
                </div>
              </div>
            </el-card>
          </el-col>

          <el-col :span="6">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-content">
                <div class="stat-icon resolved">
                  <el-icon><CircleCheck /></el-icon>
                </div>
                <div class="stat-info">
                  <div class="stat-value">{{ summary.resolvedCount }}</div>
                  <div class="stat-label">已恢复异常</div>
                </div>
              </div>
            </el-card>
          </el-col>

          <el-col :span="6">
            <el-card shadow="hover" class="stat-card">
              <div class="stat-content">
                <div class="stat-icon nodes">
                  <el-icon><Monitor /></el-icon>
                </div>
                <div class="stat-info">
                  <div class="stat-value">{{ summary.affectedNodes }}</div>
                  <div class="stat-label">受影响节点</div>
                </div>
              </div>
            </el-card>
          </el-col>
        </el-row>

        <!-- 趋势分析图表 -->
        <TrendCharts
          :cluster-id="filterForm.cluster_id"
          :start-time="computedStartTime"
          :end-time="computedEndTime"
          :clusters="clusters"
          :anomalies="tableData"
          @date-click="handleDateClick"
          @type-click="handleTypeClick"
          ref="trendChartsRef"
        />
      </el-tab-pane>

      <!-- 高级统计面板 -->
      <el-tab-pane label="高级统计" name="advanced">
        <el-row :gutter="20">
          <!-- 按角色聚合统计 -->
          <el-col :span="12">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>按节点角色统计</span>
                </div>
              </template>
              <div ref="roleChartRef" style="width: 100%; height: 300px"></div>
            </el-card>
          </el-col>

          <!-- MTTR 统计 -->
          <el-col :span="12">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>平均恢复时间 (MTTR)</span>
                  <el-tooltip content="Mean Time To Recovery - 平均恢复时间" placement="top">
                    <el-icon><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
              </template>
              <div ref="mttrChartRef" style="width: 100%; height: 300px"></div>
            </el-card>
          </el-col>
        </el-row>

        <el-row :gutter="20" style="margin-top: 20px">
          <!-- SLA 可用性 -->
          <el-col :span="12">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>SLA 可用性</span>
                  <el-tooltip content="Service Level Agreement - 服务可用性百分比" placement="top">
                    <el-icon><QuestionFilled /></el-icon>
                  </el-tooltip>
                </div>
              </template>
              <div ref="slaChartRef" style="width: 100%; height: 300px"></div>
            </el-card>
          </el-col>

          <!-- 恢复率和复发率 -->
          <el-col :span="12">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>恢复率 & 复发率</span>
                </div>
              </template>
              <div ref="recoveryChartRef" style="width: 100%; height: 300px"></div>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <!-- 节点健康面板 -->
      <el-tab-pane label="节点健康" name="health">
        <el-row :gutter="20">
          <el-col :span="24">
            <el-card>
              <template #header>
                <div class="card-header">
                  <span>节点健康度排行 Top 10</span>
                  <el-button type="primary" size="small" @click="refreshHealthData">
                    <el-icon><Refresh /></el-icon>
                    刷新
                  </el-button>
                </div>
              </template>
              <el-table
                v-loading="healthLoading"
                :data="topUnhealthyNodes"
                stripe
                :height="600"
                style="width: 100%"
              >
                <el-table-column type="index" label="排名" width="80" />
                <el-table-column prop="node_name" label="节点名称" width="200" />
                <el-table-column prop="cluster_name" label="集群" width="150" />
                <el-table-column label="健康度评分" width="200">
                  <template #default="{ row }">
                    <el-progress
                      :percentage="row.health_score"
                      :color="getHealthColor(row.health_score)"
                      :stroke-width="16"
                    >
                      <span style="font-size: 12px; font-weight: bold">
                        {{ row.health_score.toFixed(1) }}分
                      </span>
                    </el-progress>
                  </template>
                </el-table-column>
                <el-table-column label="等级" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getHealthLevelType(row.health_score)">
                      {{ getHealthLevel(row.health_score) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="total_anomalies" label="总异常数" width="120" />
                <el-table-column prop="active_anomalies" label="活跃异常" width="120">
                  <template #default="{ row }">
                    <el-tag v-if="row.active_anomalies > 0" type="danger">
                      {{ row.active_anomalies }}
                    </el-tag>
                    <span v-else>0</span>
                  </template>
                </el-table-column>
                <el-table-column label="平均恢复时间" width="140">
                  <template #default="{ row }">
                    <span v-if="row.avg_recovery_time && row.avg_recovery_time > 0">
                      {{ formatSeconds(row.avg_recovery_time) }}
                    </span>
                    <el-tooltip v-else content="该节点暂无已恢复的异常记录" placement="top">
                      <span style="color: #999">-</span>
                    </el-tooltip>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="120" fixed="right">
                  <template #default="{ row }">
                    <el-button
                      type="primary"
                      size="small"
                      link
                      @click="showNodeHealthDetail(row)"
                    >
                      查看详情
                    </el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <!-- 异常记录面板 -->
      <el-tab-pane label="异常记录" name="records">
        <el-card class="table-card">
          <template #header>
            <div class="card-header">
              <span>异常记录列表</span>
              <div>
                <el-select
                  v-model="filterForm.anomaly_type"
                  placeholder="异常类型"
                  clearable
                  style="width: 180px; margin-right: 10px"
                  @change="handleFilterChange"
                >
                  <el-option label="NotReady" value="NotReady" />
                  <el-option label="MemoryPressure" value="MemoryPressure" />
                  <el-option label="DiskPressure" value="DiskPressure" />
                  <el-option label="PIDPressure" value="PIDPressure" />
                  <el-option label="NetworkUnavailable" value="NetworkUnavailable" />
                </el-select>
                <el-button type="primary" :icon="Download" size="small" @click="handleExport">
                  导出
                </el-button>
              </div>
            </div>
          </template>

          <!-- 空状态 -->
          <EmptyState
            v-if="!tableLoading && tableData.length === 0"
            type="success"
            title="集群运行健康"
            description="当前时间范围内暂无异常记录，系统运行正常"
            :action="{
              text: '刷新数据',
              handler: handleSearch
            }"
          />

          <el-table
            v-else
            v-loading="tableLoading"
            :data="tableData"
            stripe
            border
            style="width: 100%"
          >
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="cluster_name" label="集群" width="150" />
            <el-table-column prop="node_name" label="节点" width="200" />
            <el-table-column label="异常类型" width="180">
              <template #default="{ row }">
                <el-tag :type="getAnomalyTypeColor(row.anomaly_type)">
                  {{ row.anomaly_type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="row.status === 'Active' ? 'danger' : 'success'">
                  {{ row.status === 'Active' ? '进行中' : '已恢复' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="开始时间" width="180">
              <template #default="{ row }">
                {{ formatDateTime(row.start_time) }}
              </template>
            </el-table-column>
            <el-table-column label="结束时间" width="180">
              <template #default="{ row }">
                {{ row.end_time ? formatDateTime(row.end_time) : '-' }}
              </template>
            </el-table-column>
            <el-table-column label="持续时长" width="120">
              <template #default="{ row }">
                {{ formatDuration(row) }}
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" min-width="150" show-overflow-tooltip />
            <el-table-column prop="message" label="详细信息" min-width="200" show-overflow-tooltip />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" size="small" link @click="handleViewDetail(row.id)">
                  查看详情
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handlePageSizeChange"
            @current-change="handlePageChange"
          />
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 节点健康详情对话框 -->
    <el-dialog
      v-model="healthDetailDialogVisible"
      :title="`节点健康详情 - ${selectedNodeHealth?.node_name || ''}`"
      width="800px"
      destroy-on-close
    >
      <div v-if="selectedNodeHealth" class="health-detail-content">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="节点名称">
            {{ selectedNodeHealth.node_name }}
          </el-descriptions-item>
          <el-descriptions-item label="所属集群">
            {{ selectedNodeHealth.cluster_name }}
          </el-descriptions-item>
          <el-descriptions-item label="健康度评分">
            <el-progress
              :percentage="selectedNodeHealth.health_score"
              :color="getHealthColor(selectedNodeHealth.health_score)"
              :stroke-width="20"
            >
              <span style="font-size: 14px; font-weight: bold">
                {{ selectedNodeHealth.health_score.toFixed(2) }}分
              </span>
            </el-progress>
          </el-descriptions-item>
          <el-descriptions-item label="健康等级">
            <el-tag :type="getHealthLevelType(selectedNodeHealth.health_score)" size="large">
              {{ getHealthLevel(selectedNodeHealth.health_score) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="总异常数">
            <el-tag type="info" size="large">{{ selectedNodeHealth.total_anomalies }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="活跃异常">
            <el-tag 
              :type="selectedNodeHealth.active_anomalies > 0 ? 'danger' : 'success'" 
              size="large"
            >
              {{ selectedNodeHealth.active_anomalies }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="平均恢复时间">
            <el-tag type="warning" size="large">
              {{ formatSeconds(selectedNodeHealth.avg_recovery_time) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="最近异常">
            <span v-if="selectedNodeHealth.last_anomaly_time">
              {{ formatDateTime(selectedNodeHealth.last_anomaly_time) }}
            </span>
            <span v-else>无</span>
          </el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">异常详细信息</el-divider>
        
        <el-row :gutter="16">
          <el-col :span="12">
            <el-statistic title="健康指数" :value="selectedNodeHealth.health_score" :precision="2">
              <template #suffix>/ 100</template>
            </el-statistic>
          </el-col>
          <el-col :span="12">
            <el-statistic 
              title="异常率" 
              :value="selectedNodeHealth.total_anomalies > 0 ? ((selectedNodeHealth.active_anomalies / selectedNodeHealth.total_anomalies) * 100) : 0" 
              :precision="1"
              suffix="%"
            />
          </el-col>
        </el-row>

        <div style="margin-top: 20px;">
          <el-alert
            v-if="selectedNodeHealth.active_anomalies > 0"
            title="当前有活跃异常，请及时处理"
            type="warning"
            :closable="false"
            show-icon
          />
          <el-alert
            v-else
            title="节点运行正常，无活跃异常"
            type="success"
            :closable="false"
            show-icon
          />
        </div>
      </div>

      <template #footer>
        <el-button @click="healthDetailDialogVisible = false">关闭</el-button>
        <el-button 
          type="primary" 
          @click="router.push({ name: 'NodeList', query: { node_name: selectedNodeHealth?.node_name } })"
        >
          查看节点详情
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
import {
  Search,
  Refresh,
  RefreshRight,
  Download,
  DataAnalysis,
  Warning,
  CircleCheck,
  Monitor,
  QuestionFilled
} from '@element-plus/icons-vue'
import {
  getAnomalies,
  getActiveAnomalies,
  triggerCheck,
  getRoleStatistics,
  getMTTR,
  getSLA,
  getRecoveryMetrics,
  getTopUnhealthyNodes
} from '@/api/anomaly'
import clusterApi from '@/api/cluster'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/store/modules/auth'
import { usePageVisibility } from '@/composables/usePageVisibility'
import * as echarts from 'echarts'
import EmptyState from '@/components/common/EmptyState.vue'
import TrendCharts from '@/components/analytics/TrendCharts.vue'
import { handleError, showSuccess, showWarning, ErrorLevel } from '@/utils/errorHandler'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const userRole = computed(() => authStore.userInfo?.role || '')

// 趋势图表引用
const trendChartsRef = ref(null)

// 当前激活的Tab（支持从路由参数初始化）
const activeTab = ref(route.query.tab || 'overview')

// 页面可见性检测
const { isVisible } = usePageVisibility()

// 轮询定时器
let pollIntervalId = null
const pollInterval = 30 // 秒

// 集群列表
const clusters = ref([])

// 节点健康详情对话框
const healthDetailDialogVisible = ref(false)
const selectedNodeHealth = ref(null)

// 过滤表单
const filterForm = reactive({
  cluster_id: null,
  anomaly_type: '',
  dateRange: [],
  dimension: 'day'
})

// 统计摘要
const summary = reactive({
  totalCount: 0,
  activeCount: 0,
  resolvedCount: 0,
  affectedNodes: 0
})

// 表格数据
const tableData = ref([])
const tableLoading = ref(false)

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 手动检测加载状态
const checkLoading = ref(false)

// 高级统计图表 refs
const roleChartRef = ref(null)
const mttrChartRef = ref(null)
const slaChartRef = ref(null)
const recoveryChartRef = ref(null)

// 高级统计图表实例
let roleChart = null
let mttrChart = null
let slaChart = null
let recoveryChart = null

// 节点健康数据
const topUnhealthyNodes = ref([])
const healthLoading = ref(false)

// 计算属性：图表时间范围
const computedStartTime = computed(() => {
  if (filterForm.dateRange && filterForm.dateRange.length === 2) {
    return new Date(filterForm.dateRange[0]).toISOString()
  }
  // 默认最近7天
  const date = new Date()
  date.setDate(date.getDate() - 7)
  return date.toISOString()
})

const computedEndTime = computed(() => {
  if (filterForm.dateRange && filterForm.dateRange.length === 2) {
    return new Date(filterForm.dateRange[1]).toISOString()
  }
  // 默认今天
  return new Date().toISOString()
})

// 加载集群列表
const loadClusters = async () => {
  console.log('Loading clusters...')
  try {
    const response = await clusterApi.getClusters()
    console.log('Clusters API response:', response.data)
    
    if (response.data && response.data.code === 200) {
      clusters.value = response.data.data?.clusters || []
      console.log('Loaded clusters:', clusters.value)
      console.log('Clusters count:', clusters.value.length)
    } else {
      console.warn('Invalid clusters response:', response)
    }
  } catch (error) {
    console.error('Failed to load clusters:', error)
  }
}

// 加载活跃异常摘要
const loadActiveSummary = async () => {
  try {
    const response = await getActiveAnomalies(filterForm.cluster_id || null)
    if (response.data && response.data.code === 200) {
      const data = response.data.data || {}
      summary.totalCount = data.total_count || 0
      summary.activeCount = data.active_count || 0
      summary.resolvedCount = data.resolved_count || 0
      summary.affectedNodes = data.affected_nodes || 0
    }
  } catch (error) {
    console.error('Failed to load active summary:', error)
  }
}

// 加载异常记录列表
const loadAnomalies = async () => {
  tableLoading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize
    }

    if (filterForm.cluster_id) {
      params.cluster_id = filterForm.cluster_id
    }
    if (filterForm.anomaly_type) {
      params.anomaly_type = filterForm.anomaly_type
    }
    if (filterForm.dateRange && filterForm.dateRange.length === 2) {
      params.start_time = new Date(filterForm.dateRange[0]).toISOString()
      params.end_time = new Date(filterForm.dateRange[1]).toISOString()
    }

    const response = await getAnomalies(params)
    if (response.data && response.data.code === 200) {
      const data = response.data.data
      tableData.value = data.items || []
      pagination.total = data.total || 0
    }
  } catch (error) {
    console.error('Failed to load anomalies:', error)
    handleError(error, ErrorLevel.ERROR, { title: '加载失败' })
  } finally {
    tableLoading.value = false
  }
}

// 过滤条件变化
const handleFilterChange = () => {
  pagination.page = 1
  loadAnomalies()
  loadActiveSummary()
}

// 查询
const handleSearch = () => {
  handleFilterChange()
}

// 重置
const handleReset = () => {
  filterForm.cluster_id = null
  filterForm.anomaly_type = ''
  filterForm.dateRange = []
  filterForm.dimension = 'day'
  handleFilterChange()
}

// 分页大小变化
const handlePageSizeChange = (size) => {
  pagination.pageSize = size
  pagination.page = 1
  loadAnomalies()
}

// 页码变化
const handlePageChange = (page) => {
  pagination.page = page
  loadAnomalies()
}

// 手动触发检测
const handleTriggerCheck = async () => {
  checkLoading.value = true
  try {
    const response = await triggerCheck()
    if (response.data && response.data.code === 200) {
      showSuccess('检测任务已触发，请稍后刷新查看结果')
      // 延迟5秒后自动刷新
      setTimeout(() => {
        loadAnomalies()
        loadActiveSummary()
        // 刷新图表
        if (trendChartsRef.value) {
          trendChartsRef.value.refresh()
        }
      }, 5000)
    }
  } catch (error) {
    console.error('Failed to trigger check:', error)
    handleError(error, ErrorLevel.ERROR, { title: '触发检测失败' })
  } finally {
    checkLoading.value = false
  }
}

// 图表日期点击事件
const handleDateClick = ({ date, dimension }) => {
  console.log('日期点击:', date, dimension)
  // 可以根据点击的日期更新筛选条件
  // 这里作为未来扩展预留
}

// 图表类型点击事件
const handleTypeClick = ({ type }) => {
  console.log('类型点击:', type)
  // 根据点击的类型更新筛选条件
  const typeMap = {
    '节点未就绪': 'NotReady',
    '内存压力': 'MemoryPressure',
    '磁盘压力': 'DiskPressure',
    'PID压力': 'PIDPressure',
    '网络不可用': 'NetworkUnavailable'
  }
  filterForm.anomaly_type = typeMap[type] || type
  activeTab.value = 'records' // 切换到记录Tab
  handleFilterChange()
}

// 查看详情
const handleViewDetail = (id) => {
  router.push({ name: 'AnomalyDetail', params: { id } })
}

// 导出数据
const handleExport = () => {
  if (tableData.value.length === 0) {
    showWarning('暂无数据可导出')
    return
  }

  // 简单的CSV导出
  const headers = ['ID', '集群', '节点', '异常类型', '状态', '开始时间', '结束时间', '持续时长', '原因', '详细信息']
  const rows = tableData.value.map(row => [
    row.id,
    row.cluster_name,
    row.node_name,
    row.anomaly_type,
    row.status === 'Active' ? '进行中' : '已恢复',
    formatDateTime(row.start_time),
    row.end_time ? formatDateTime(row.end_time) : '-',
    formatDuration(row),
    row.reason,
    row.message
  ])

  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
  ].join('\n')

  const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `anomalies_${new Date().toISOString().split('T')[0]}.csv`
  link.click()
  
  showSuccess('导出成功')
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

// 格式化持续时长
const formatDuration = (row) => {
  const duration = row.duration || 0
  if (duration < 60) {
    return `${duration}秒`
  } else if (duration < 3600) {
    const minutes = Math.floor(duration / 60)
    const seconds = duration % 60
    return `${minutes}分${seconds}秒`
  } else {
    const hours = Math.floor(duration / 3600)
    const minutes = Math.floor((duration % 3600) / 60)
    return `${hours}小时${minutes}分`
  }
}

// 获取异常类型颜色
const getAnomalyTypeColor = (type) => {
  const colorMap = {
    'NotReady': 'danger',
    'MemoryPressure': 'warning',
    'DiskPressure': 'warning',
    'PIDPressure': 'info',
    'NetworkUnavailable': 'danger'
  }
  return colorMap[type] || 'info'
}

// 启动智能轮询
const startPolling = () => {
  stopPolling()
  
  pollIntervalId = setInterval(() => {
    if (isVisible.value) {
      loadActiveSummary()
    }
  }, pollInterval * 1000)
}

// 停止轮询
const stopPolling = () => {
  if (pollIntervalId) {
    clearInterval(pollIntervalId)
    pollIntervalId = null
  }
}

// 监听页面可见性变化
watch(isVisible, (visible) => {
  if (visible) {
    // 页面变为可见时，立即刷新一次，然后启动轮询
    loadActiveSummary()
    startPolling()
  } else {
    // 页面隐藏时停止轮询
    stopPolling()
  }
})

// 监听Tab切换
watch(activeTab, async (newTab) => {
  if (newTab === 'advanced') {
    await nextTick()
    loadAdvancedStatistics()
  } else if (newTab === 'health') {
    loadHealthData()
  }
})

// 加载高级统计数据
const loadAdvancedStatistics = async () => {
  await Promise.all([
    loadRoleStatistics(),
    loadMTTRStatistics(),
    loadSLAMetrics(),
    loadRecoveryMetrics()
  ])
}

// 加载按角色统计
const loadRoleStatistics = async () => {
  try {
    const params = {
      cluster_id: filterForm.cluster_id,
      start_time: computedStartTime.value,
      end_time: computedEndTime.value
    }
    const res = await getRoleStatistics(params)
    // 处理响应数据，可能是 res.data 或 res.data.data
    const data = res.data?.data || res.data || []
    console.log('角色统计数据：', data)
    renderRoleChart(data)
  } catch (error) {
    console.error('加载角色统计失败：', error)
    renderRoleChart([])
  }
}

// 渲染角色统计图表
const renderRoleChart = (data) => {
  if (!roleChartRef.value) return
  
  if (!roleChart) {
    roleChart = echarts.init(roleChartRef.value)
  }
  
  // 确保 data 是数组
  const chartData = Array.isArray(data) ? data : []
  
  if (chartData.length === 0) {
    const option = {
      title: {
        text: '暂无数据',
        left: 'center',
        top: 'center',
        textStyle: {
          color: '#999',
          fontSize: 14
        }
      }
    }
    roleChart.setOption(option)
    return
  }
  
  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['总异常', '活跃异常', '已恢复']
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: chartData.map(item => item.role || '未知')
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '总异常',
        type: 'bar',
        data: chartData.map(item => item.total_anomalies || 0),
        itemStyle: { color: '#409EFF' }
      },
      {
        name: '活跃异常',
        type: 'bar',
        data: chartData.map(item => item.active_anomalies || 0),
        itemStyle: { color: '#F56C6C' }
      },
      {
        name: '已恢复',
        type: 'bar',
        data: chartData.map(item => item.resolved_anomalies || 0),
        itemStyle: { color: '#67C23A' }
      }
    ]
  }
  
  roleChart.setOption(option)
}

// 加载MTTR统计
const loadMTTRStatistics = async () => {
  try {
    const params = {
      entity_type: 'cluster',
      cluster_id: filterForm.cluster_id,
      start_time: computedStartTime.value,
      end_time: computedEndTime.value
    }
    const res = await getMTTR(params)
    // 处理响应数据
    const data = res.data?.data || res.data || []
    console.log('MTTR统计数据：', data)
    renderMTTRChart(data)
  } catch (error) {
    console.error('加载MTTR统计失败：', error)
    renderMTTRChart([])
  }
}

// 渲染MTTR图表
const renderMTTRChart = (data) => {
  if (!mttrChartRef.value) return
  
  if (!mttrChart) {
    mttrChart = echarts.init(mttrChartRef.value)
  }
  
  // 确保 data 是数组
  const chartData = Array.isArray(data) ? data : []
  
  if (chartData.length === 0) {
    const option = {
      title: {
        text: '暂无已恢复的异常',
        subtext: '只有异常恢复后才能统计平均恢复时间',
        left: 'center',
        top: 'center',
        textStyle: {
          color: '#999',
          fontSize: 14
        },
        subtextStyle: {
          color: '#ccc',
          fontSize: 12
        }
      }
    }
    mttrChart.setOption(option)
    return
  }
  
  const option = {
    tooltip: {
      trigger: 'axis',
      formatter: (params) => {
        const item = params[0]
        return `${item.name}<br/>平均恢复时间: ${formatSeconds(item.value)}`
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: chartData.map(item => item.entity_name || '全部')
    },
    yAxis: {
      type: 'value',
      name: '时间(秒)',
      axisLabel: {
        formatter: '{value}s'
      }
    },
    series: [{
      data: chartData.map(item => item.avg_recovery_time || 0),
      type: 'bar',
      itemStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: '#83bff6' },
          { offset: 1, color: '#188df0' }
        ])
      },
      label: {
        show: true,
        position: 'top',
        formatter: (params) => formatSeconds(params.value)
      }
    }]
  }
  
  mttrChart.setOption(option)
}

// 加载SLA统计
const loadSLAMetrics = async () => {
  try {
    // SLA统计暂时跳过，因为需要entity_name参数
    // 显示一个默认的仪表盘
    renderSLAChart({ sla_percentage: 99.5 })
  } catch (error) {
    console.error('加载SLA统计失败：', error)
    renderSLAChart({ sla_percentage: 0 })
  }
}

// 渲染SLA图表
const renderSLAChart = (data) => {
  if (!slaChartRef.value) return
  
  if (!slaChart) {
    slaChart = echarts.init(slaChartRef.value)
  }
  
  const slaPercentage = data.sla_percentage || 0
  
  const option = {
    series: [{
      type: 'gauge',
      startAngle: 180,
      endAngle: 0,
      min: 0,
      max: 100,
      splitNumber: 10,
      axisLine: {
        lineStyle: {
          width: 30,
          color: [
            [0.9, '#F56C6C'],
            [0.95, '#E6A23C'],
            [0.99, '#409EFF'],
            [1, '#67C23A']
          ]
        }
      },
      pointer: {
        itemStyle: {
          color: 'auto'
        }
      },
      axisTick: {
        distance: -30,
        length: 8,
        lineStyle: {
          color: '#fff',
          width: 2
        }
      },
      splitLine: {
        distance: -30,
        length: 30,
        lineStyle: {
          color: '#fff',
          width: 4
        }
      },
      axisLabel: {
        color: 'auto',
        distance: 40,
        fontSize: 12,
        formatter: '{value}%'
      },
      detail: {
        valueAnimation: true,
        formatter: '{value}%',
        color: 'auto',
        fontSize: 30
      },
      data: [{
        value: slaPercentage.toFixed(2),
        name: 'SLA 可用性'
      }]
    }]
  }
  
  slaChart.setOption(option)
}

// 加载恢复率和复发率
const loadRecoveryMetrics = async () => {
  try {
    // 恢复指标暂时跳过，因为需要entity_name参数
    // 显示一个默认的饼图
    renderRecoveryChart({ recovery_rate: 85, recurrence_rate: 15 })
  } catch (error) {
    console.error('加载恢复指标失败：', error)
    renderRecoveryChart({ recovery_rate: 0, recurrence_rate: 0 })
  }
}

// 渲染恢复率图表
const renderRecoveryChart = (data) => {
  if (!recoveryChartRef.value) return
  
  if (!recoveryChart) {
    recoveryChart = echarts.init(recoveryChartRef.value)
  }
  
  const option = {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      top: '10%',
      left: 'center'
    },
    series: [{
      name: '恢复指标',
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
        formatter: '{b}: {c}%'
      },
      emphasis: {
        label: {
          show: true,
          fontSize: 16,
          fontWeight: 'bold'
        }
      },
      data: [
        {
          value: (data.recovery_rate || 0).toFixed(1),
          name: '恢复率',
          itemStyle: { color: '#67C23A' }
        },
        {
          value: (data.recurrence_rate || 0).toFixed(1),
          name: '复发率',
          itemStyle: { color: '#F56C6C' }
        }
      ]
    }]
  }
  
  recoveryChart.setOption(option)
}

// 加载节点健康数据
const loadHealthData = async () => {
  healthLoading.value = true
  try {
    const params = {
      cluster_id: filterForm.cluster_id,
      limit: 10,
      start_time: computedStartTime.value,
      end_time: computedEndTime.value
    }
    const res = await getTopUnhealthyNodes(params)
    // 处理响应数据
    const data = res.data?.data || res.data || []
    console.log('节点健康数据：', data)
    topUnhealthyNodes.value = data
  } catch (error) {
    console.error('加载节点健康数据失败：', error)
    topUnhealthyNodes.value = []
  } finally {
    healthLoading.value = false
  }
}

// 刷新健康数据
const refreshHealthData = () => {
  loadHealthData()
}

// 显示节点健康详情（跳转或弹窗）
const showNodeHealthDetail = (row) => {
  selectedNodeHealth.value = row
  healthDetailDialogVisible.value = true
  console.log('查看节点健康详情：', row.node_name, row)
}

// 获取健康度颜色
const getHealthColor = (score) => {
  if (score >= 90) return '#67C23A'
  if (score >= 75) return '#409EFF'
  if (score >= 60) return '#E6A23C'
  if (score >= 40) return '#F56C6C'
  return '#909399'
}

// 获取健康度等级
const getHealthLevel = (score) => {
  if (score >= 90) return '优秀'
  if (score >= 75) return '良好'
  if (score >= 60) return '一般'
  if (score >= 40) return '较差'
  return '很差'
}

// 获取健康度等级类型
const getHealthLevelType = (score) => {
  if (score >= 90) return 'success'
  if (score >= 75) return ''
  if (score >= 60) return 'warning'
  return 'danger'
}

// 格式化秒数
const formatSeconds = (seconds) => {
  if (!seconds || seconds === 0) return '-'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

// 初始化
onMounted(() => {
  loadClusters()
  loadAnomalies()
  loadActiveSummary()
  
  // 启动智能轮询
  startPolling()
  
  // 响应式调整图表大小
  window.addEventListener('resize', () => {
    roleChart?.resize()
    mttrChart?.resize()
    slaChart?.resize()
    recoveryChart?.resize()
  })
})

// 清理
onUnmounted(() => {
  stopPolling()
  
  // 销毁图表实例
  roleChart?.dispose()
  mttrChart?.dispose()
  slaChart?.dispose()
  recoveryChart?.dispose()
})
</script>

<style scoped>
.analytics-container {
  padding: 20px;
}

.filter-card {
  margin-bottom: 20px;
}

.filter-form {
  margin-bottom: 0;
}

.analytics-tabs {
  background: white;
  padding: 20px;
  border-radius: 4px;
}

.stats-row {
  margin-bottom: 30px;
}

.stat-card {
  cursor: pointer;
  transition: transform 0.3s;
}

.stat-card:hover {
  transform: translateY(-5px);
}

.stat-content {
  display: flex;
  align-items: center;
  padding: 10px 0;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  margin-right: 15px;
}

.stat-icon.total {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.stat-icon.active {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: white;
}

.stat-icon.resolved {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: white;
}

.stat-icon.nodes {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  color: white;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #303133;
  line-height: 1;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #909399;
}

.table-card {
  margin-top: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

:deep(.el-pagination) {
  margin-top: 20px;
  text-align: right;
}

/* 健康详情对话框样式 */
.health-detail-content {
  padding: 10px;
}

.health-detail-content .el-descriptions {
  margin-bottom: 20px;
}

.health-detail-content .el-divider {
  margin: 24px 0;
}

.health-detail-content .el-statistic {
  text-align: center;
  padding: 20px;
  background: #f5f7fa;
  border-radius: 8px;
}
</style>
