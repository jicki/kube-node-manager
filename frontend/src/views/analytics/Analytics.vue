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

        <el-form-item label="异常类型">
          <el-select
            v-model="filterForm.anomaly_type"
            placeholder="请选择异常类型"
            clearable
            style="width: 180px"
            @change="handleFilterChange"
          >
            <el-option label="NotReady" value="NotReady" />
            <el-option label="MemoryPressure" value="MemoryPressure" />
            <el-option label="DiskPressure" value="DiskPressure" />
            <el-option label="PIDPressure" value="PIDPressure" />
            <el-option label="NetworkUnavailable" value="NetworkUnavailable" />
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

        <el-form-item label="统计维度">
          <el-radio-group v-model="filterForm.dimension" @change="handleFilterChange">
            <el-radio-button label="day">按天</el-radio-button>
            <el-radio-button label="week">按周</el-radio-button>
          </el-radio-group>
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
              <div class="stat-label">总异常次数</div>
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

    <!-- 异常记录列表 -->
    <el-card class="table-card">
      <template #header>
        <div class="card-header">
          <span>异常记录列表</span>
          <el-button type="primary" :icon="Download" size="small" @click="handleExport">
            导出
          </el-button>
        </div>
      </template>

      <el-table
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
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
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
import { listClusters } from '@/api/cluster'
import { useAuthStore } from '@/store/modules/auth'

const authStore = useAuthStore()
const userRole = computed(() => authStore.userInfo?.role || '')

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

// 加载集群列表
const loadClusters = async () => {
  try {
    const response = await listClusters()
    if (response.data && response.data.code === 200) {
      clusters.value = response.data.data || []
    }
  } catch (error) {
    console.error('Failed to load clusters:', error)
  }
}

// 加载活跃异常统计
const loadActiveSummary = async () => {
  try {
    const response = await getActiveAnomalies(filterForm.cluster_id)
    if (response.data && response.data.code === 200) {
      const activeAnomalies = response.data.data || []
      summary.activeCount = activeAnomalies.length
      
      // 计算受影响节点数
      const nodeSet = new Set(activeAnomalies.map(a => a.node_name))
      summary.affectedNodes = nodeSet.size
    }
  } catch (error) {
    console.error('Failed to load active anomalies:', error)
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
      params.start_time = new Date(filterForm.dateRange[0] + 'T00:00:00Z').toISOString()
      params.end_time = new Date(filterForm.dateRange[1] + 'T23:59:59Z').toISOString()
    }

    const response = await getAnomalies(params)
    if (response.data && response.data.code === 200) {
      const data = response.data.data
      tableData.value = data.items || []
      pagination.total = data.total || 0
      
      // 更新统计摘要
      summary.totalCount = data.total || 0
      
      // 计算已恢复异常数
      summary.resolvedCount = (data.items || []).filter(item => item.status === 'Resolved').length
    }
  } catch (error) {
    console.error('Failed to load anomalies:', error)
    ElMessage.error('加载异常记录失败')
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
      ElMessage.success('检测任务已触发，请稍后刷新查看结果')
      // 延迟5秒后自动刷新
      setTimeout(() => {
        loadAnomalies()
        loadActiveSummary()
      }, 5000)
    }
  } catch (error) {
    console.error('Failed to trigger check:', error)
    ElMessage.error('触发检测失败：' + (error.response?.data?.message || error.message))
  } finally {
    checkLoading.value = false
  }
}

// 导出数据
const handleExport = () => {
  if (tableData.value.length === 0) {
    ElMessage.warning('暂无数据可导出')
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
    row.reason || '-',
    row.message || '-'
  ])

  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
  ].join('\n')

  const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `node-anomalies-${new Date().getTime()}.csv`
  link.click()
  URL.revokeObjectURL(link.href)
  
  ElMessage.success('导出成功')
}

// 格式化日期时间
const formatDateTime = (dateString) => {
  if (!dateString) return '-'
  const date = new Date(dateString)
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
  let seconds = row.duration
  
  // 如果是活跃状态，计算到当前时间的持续时长
  if (row.status === 'Active' && row.start_time) {
    const startTime = new Date(row.start_time)
    const now = new Date()
    seconds = Math.floor((now - startTime) / 1000)
  }
  
  if (seconds < 60) {
    return `${seconds}秒`
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60)
    return `${minutes}分钟`
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    return `${hours}小时${minutes}分钟`
  } else {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    return `${days}天${hours}小时`
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

// 初始化
onMounted(() => {
  loadClusters()
  loadAnomalies()
  loadActiveSummary()
  
  // 每30秒自动刷新活跃异常统计
  setInterval(() => {
    loadActiveSummary()
  }, 30000)
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
  display: flex;
  flex-wrap: wrap;
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
  padding: 10px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  margin-right: 20px;
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
  margin-bottom: 10px;
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

.el-pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>

