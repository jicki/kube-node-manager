<template>
  <div class="workflow-execution-list">
    <div class="header">
      <h2>工作流执行监控</h2>
      <div class="actions">
        <el-button @click="handleBack">返回</el-button>
        <el-button 
          type="primary" 
          icon="el-icon-refresh" 
          @click="loadExecutions"
          :loading="loading"
        >
          刷新
        </el-button>
      </div>
    </div>

    <!-- 筛选器 -->
    <el-card class="filter-card">
      <el-form :inline="true" :model="filters">
        <el-form-item label="工作流">
          <el-select 
            v-model="filters.workflow_id" 
            clearable 
            placeholder="选择工作流"
            style="width: 200px"
            @change="loadExecutions"
          >
            <el-option
              v-for="workflow in workflows"
              :key="workflow.id"
              :label="workflow.name"
              :value="workflow.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select 
            v-model="filters.status" 
            clearable 
            placeholder="选择状态"
            style="width: 150px"
            @change="loadExecutions"
          >
            <el-option label="等待中" value="pending" />
            <el-option label="运行中" value="running" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 执行列表 -->
    <el-card class="list-card">
      <el-table :data="executions" v-loading="loading" stripe>
        <el-table-column prop="id" label="执行 ID" width="100" align="center" />
        <el-table-column label="工作流" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.workflow?.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" :effect="row.status === 'running' ? 'dark' : 'plain'">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="200">
          <template #default="{ row }">
            <div v-if="row.status === 'running'" class="progress-info">
              <el-progress 
                :percentage="calculateProgress(row)" 
                :status="row.failed_tasks > 0 ? 'exception' : undefined"
              />
              <div class="progress-text">
                {{ row.completed_tasks || 0 }}/{{ row.total_tasks || 0 }} 任务完成
              </div>
            </div>
            <div v-else class="progress-text">
              {{ row.completed_tasks || 0 }}/{{ row.total_tasks || 0 }} 任务
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="started_at" label="开始时间" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.started_at ? formatTime(row.started_at) : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="finished_at" label="完成时间" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.finished_at ? formatTime(row.finished_at) : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="duration" label="耗时" width="100">
          <template #default="{ row }">
            {{ row.duration ? formatDuration(row.duration) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" icon="el-icon-view" @click="handleView(row)">
              查看详情
            </el-button>
            <el-button 
              v-if="row.status === 'running'"
              size="small" 
              type="danger" 
              icon="el-icon-close"
              @click="handleCancel(row)"
            >
              取消
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadExecutions"
          @current-change="loadExecutions"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  listWorkflowExecutions, 
  listWorkflows,
  cancelWorkflowExecution 
} from '@/api/workflow'
import { formatTime } from '@/utils/format'

const router = useRouter()

const executions = ref([])
const workflows = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const autoRefreshTimer = ref(null)

const filters = ref({
  workflow_id: null,
  status: null
})

// 加载工作流列表（用于筛选）
const loadWorkflows = async () => {
  try {
    const response = await listWorkflows({ page: 1, page_size: 1000 })
    workflows.value = response.data.workflows || []
  } catch (error) {
    console.error('Failed to load workflows:', error)
  }
}

// 加载执行列表
const loadExecutions = async () => {
  loading.value = true
  try {
    const response = await listWorkflowExecutions({
      page: currentPage.value,
      page_size: pageSize.value,
      workflow_id: filters.value.workflow_id || undefined,
      status: filters.value.status || undefined
    })
    executions.value = response.data.executions || []
    total.value = response.data.total || 0
  } catch (error) {
    console.error('Failed to load executions:', error)
    ElMessage.error(error.response?.data?.error || '加载执行列表失败')
  } finally {
    loading.value = false
  }
}

// 查看执行详情
const handleView = (execution) => {
  router.push(`/ansible/workflow-executions/${execution.id}`)
}

// 取消执行
const handleCancel = async (execution) => {
  try {
    await ElMessageBox.confirm(
      '确定要取消该工作流执行吗？',
      '确认取消',
      {
        confirmButtonText: '取消执行',
        cancelButtonText: '返回',
        type: 'warning'
      }
    )

    await cancelWorkflowExecution(execution.id)
    ElMessage.success('工作流执行已取消')
    await loadExecutions()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to cancel execution:', error)
      ElMessage.error(error.response?.data?.error || '取消执行失败')
    }
  }
}

// 返回
const handleBack = () => {
  router.push('/ansible/workflows')
}

// 获取状态类型
const getStatusType = (status) => {
  const types = {
    pending: 'info',
    running: 'warning',
    success: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return types[status] || 'info'
}

// 获取状态文本
const getStatusText = (status) => {
  const texts = {
    pending: '等待中',
    running: '运行中',
    success: '成功',
    failed: '失败',
    cancelled: '已取消'
  }
  return texts[status] || status
}

// 计算进度百分比
const calculateProgress = (execution) => {
  if (!execution.total_tasks || execution.total_tasks === 0) {
    return 0
  }
  return Math.round((execution.completed_tasks || 0) / execution.total_tasks * 100)
}

// 格式化时长
const formatDuration = (seconds) => {
  if (!seconds) return '-'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  
  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`
  } else if (minutes > 0) {
    return `${minutes}m ${secs}s`
  } else {
    return `${secs}s`
  }
}

// 启动自动刷新
const startAutoRefresh = () => {
  autoRefreshTimer.value = setInterval(() => {
    // 如果有运行中的执行，才自动刷新
    const hasRunning = executions.value.some(e => e.status === 'running')
    if (hasRunning) {
      loadExecutions()
    }
  }, 5000) // 每 5 秒刷新一次
}

// 停止自动刷新
const stopAutoRefresh = () => {
  if (autoRefreshTimer.value) {
    clearInterval(autoRefreshTimer.value)
    autoRefreshTimer.value = null
  }
}

// 初始化
onMounted(async () => {
  await loadWorkflows()
  await loadExecutions()
  startAutoRefresh()
})

// 清理
onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.workflow-execution-list {
  padding: 20px;
}

.workflow-execution-list .header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.workflow-execution-list .header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.workflow-execution-list .header .actions {
  display: flex;
  gap: 12px;
}

.workflow-execution-list .filter-card {
  margin-bottom: 20px;
}

.workflow-execution-list .list-card {
  min-height: 400px;
}

.progress-info {
  width: 100%;
}

.progress-text {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>

