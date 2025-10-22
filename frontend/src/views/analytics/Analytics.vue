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
      <!-- 概览面板 -->
      <el-tab-pane label="数据概览" name="overview">
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
      </el-tab-pane>

      <!-- 趋势分析面板 -->
      <el-tab-pane label="趋势分析" name="trend">
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
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, computed, watch } from 'vue'
import {
  Search,
  Refresh,
  RefreshRight,
  Download,
  DataAnalysis,
  Warning,
  CircleCheck,
  Monitor
} from '@element-plus/icons-vue'
import { getAnomalies, getActiveAnomalies, triggerCheck } from '@/api/anomaly'
import clusterApi from '@/api/cluster'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/store/modules/auth'
import { usePageVisibility } from '@/composables/usePageVisibility'
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

// 初始化
onMounted(() => {
  loadClusters()
  loadAnomalies()
  loadActiveSummary()
  
  // 启动智能轮询
  startPolling()
})

// 清理
onUnmounted(() => {
  stopPolling()
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
  margin-bottom: 20px;
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
</style>
