<template>
  <div class="workflow-list">
    <div class="header">
      <h2>Ansible 工作流管理</h2>
      <div class="actions">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索工作流名称或描述"
          class="search-input"
          prefix-icon="el-icon-search"
          clearable
          @change="loadWorkflows"
        />
        <el-button type="primary" icon="el-icon-plus" @click="handleCreate">
          创建工作流
        </el-button>
      </div>
    </div>

    <!-- 工作流列表 -->
    <el-card class="workflow-card">
      <template #header>
        <span>工作流列表</span>
      </template>
      <el-table :data="workflows" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="name" label="工作流名称" min-width="180" show-overflow-tooltip />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="节点数" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small">{{ row.dag?.nodes?.length || 0 }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="边数" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.dag?.edges?.length || 0 }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ formatTime(row.updated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="success" icon="el-icon-video-play" @click="handleExecute(row)">
              执行
            </el-button>
            <el-button size="small" icon="el-icon-edit" @click="handleEdit(row)">
              编辑
            </el-button>
            <el-button size="small" type="info" icon="el-icon-view" @click="handleView(row)">
              查看
            </el-button>
            <el-button size="small" type="danger" icon="el-icon-delete" @click="handleDelete(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadWorkflows"
          @current-change="loadWorkflows"
        />
      </div>
    </el-card>

    <!-- 执行记录列表 -->
    <el-card class="execution-card">
      <template #header>
        <div class="execution-header">
          <span>工作流执行监控</span>
          <div class="execution-filters">
            <el-button 
              v-if="selectedExecutionIds.length > 0"
              type="danger" 
              icon="el-icon-delete"
              @click="handleBatchDelete"
              style="margin-right: 12px"
            >
              批量删除 ({{ selectedExecutionIds.length }})
            </el-button>
            <el-select 
              v-model="executionFilters.workflow_id" 
              clearable 
              placeholder="选择工作流"
              style="width: 200px; margin-right: 12px"
              @change="loadExecutions"
            >
              <el-option
                v-for="workflow in workflows"
                :key="workflow.id"
                :label="workflow.name"
                :value="workflow.id"
              />
            </el-select>
            <el-select 
              v-model="executionFilters.status" 
              clearable 
              placeholder="选择状态"
              style="width: 150px; margin-right: 12px"
              @change="loadExecutions"
            >
              <el-option label="等待中" value="pending" />
              <el-option label="运行中" value="running" />
              <el-option label="成功" value="success" />
              <el-option label="失败" value="failed" />
              <el-option label="已取消" value="cancelled" />
            </el-select>
            <el-button 
              type="primary" 
              icon="el-icon-refresh" 
              @click="loadExecutions"
              :loading="executionLoading"
            >
              刷新
            </el-button>
          </div>
        </div>
      </template>
      
      <el-table 
        :data="executions" 
        v-loading="executionLoading" 
        stripe
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" :selectable="isExecutionSelectable" />
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
                {{ row.completed_tasks || 0 }}/{{ row.total_tasks || 0 }} 任务
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
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" icon="el-icon-view" @click="handleViewExecution(row)">
              查看详情
            </el-button>
            <el-button 
              v-if="row.status === 'running'"
              size="small" 
              type="warning" 
              icon="el-icon-close"
              @click="handleCancelExecution(row)"
            >
              取消
            </el-button>
            <el-button 
              v-else
              size="small" 
              type="danger" 
              icon="el-icon-delete"
              @click="handleDeleteExecution(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="executionCurrentPage"
          v-model:page-size="executionPageSize"
          :total="executionTotal"
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
  listWorkflows, 
  deleteWorkflow, 
  executeWorkflow,
  listWorkflowExecutions,
  cancelWorkflowExecution,
  deleteWorkflowExecution,
  batchDeleteWorkflowExecutions
} from '@/api/workflow'
import { formatTime } from '@/utils/format'

const router = useRouter()

// 工作流相关状态
const workflows = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 执行记录相关状态
const executions = ref([])
const executionLoading = ref(false)
const executionCurrentPage = ref(1)
const executionPageSize = ref(20)
const executionTotal = ref(0)
const autoRefreshTimer = ref(null)
const selectedExecutionIds = ref([])

const executionFilters = ref({
  workflow_id: null,
  status: null
})

// 加载工作流列表
const loadWorkflows = async () => {
  loading.value = true
  try {
    const response = await listWorkflows({
      page: currentPage.value,
      page_size: pageSize.value,
      keyword: searchKeyword.value
    })
    workflows.value = response.data.workflows || []
    total.value = response.data.total || 0
  } catch (error) {
    console.error('Failed to load workflows:', error)
    ElMessage.error(error.response?.data?.error || '加载工作流列表失败')
  } finally {
    loading.value = false
  }
}

// 加载执行记录列表
const loadExecutions = async () => {
  executionLoading.value = true
  try {
    const response = await listWorkflowExecutions({
      page: executionCurrentPage.value,
      page_size: executionPageSize.value,
      workflow_id: executionFilters.value.workflow_id || undefined,
      status: executionFilters.value.status || undefined
    })
    executions.value = response.data.executions || []
    executionTotal.value = response.data.total || 0
  } catch (error) {
    console.error('Failed to load executions:', error)
    ElMessage.error(error.response?.data?.error || '加载执行记录失败')
  } finally {
    executionLoading.value = false
  }
}

// 创建工作流
const handleCreate = () => {
  router.push('/ansible/workflows/create')
}

// 编辑工作流
const handleEdit = (workflow) => {
  router.push(`/ansible/workflows/${workflow.id}/edit`)
}

// 查看工作流
const handleView = (workflow) => {
  router.push(`/ansible/workflows/${workflow.id}`)
}

// 删除工作流
const handleDelete = async (workflow) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除工作流"${workflow.name}"吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteWorkflow(workflow.id)
    ElMessage.success('工作流删除成功')
    loadWorkflows()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete workflow:', error)
      ElMessage.error(error.response?.data?.error || '删除工作流失败')
    }
  }
}

// 执行工作流
const handleExecute = async (workflow) => {
  try {
    await ElMessageBox.confirm(
      `确定要执行工作流"${workflow.name}"吗？`,
      '确认执行',
      {
        confirmButtonText: '执行',
        cancelButtonText: '取消',
        type: 'info'
      }
    )

    console.log('开始执行工作流，ID:', workflow.id)
    const response = await executeWorkflow(workflow.id)
    console.log('执行工作流响应:', response)
    
    // 提取执行 ID
    const executionId = response.data?.data?.id || response.data?.id
    
    if (executionId) {
      ElMessage.success(`工作流开始执行，执行 ID: ${executionId}`)
      // 刷新执行列表
      loadExecutions()
      // 跳转到详情页
      router.push(`/ansible/workflow-executions/${executionId}`)
    } else {
      console.error('无法获取执行 ID，完整响应:', response)
      ElMessage.warning('工作流已提交执行，但无法获取执行 ID')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to execute workflow:', error)
      ElMessage.error(error.response?.data?.error || '执行工作流失败')
    }
  }
}

// 查看执行详情
const handleViewExecution = (execution) => {
  router.push(`/ansible/workflow-executions/${execution.id}`)
}

// 取消执行
const handleCancelExecution = async (execution) => {
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

// 删除执行记录
const handleDeleteExecution = async (execution) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除执行记录 #${execution.id} 吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteWorkflowExecution(execution.id)
    ElMessage.success('执行记录已删除')
    selectedExecutionIds.value = []
    await loadExecutions()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete execution:', error)
      ElMessage.error(error.response?.data?.error || '删除执行记录失败')
    }
  }
}

// 批量删除执行记录
const handleBatchDelete = async () => {
  if (selectedExecutionIds.value.length === 0) {
    ElMessage.warning('请选择要删除的执行记录')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedExecutionIds.value.length} 条执行记录吗？此操作不可恢复。`,
      '确认批量删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await batchDeleteWorkflowExecutions(selectedExecutionIds.value)
    ElMessage.success(`成功删除 ${selectedExecutionIds.value.length} 条执行记录`)
    selectedExecutionIds.value = []
    await loadExecutions()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to batch delete executions:', error)
      ElMessage.error(error.response?.data?.error || '批量删除执行记录失败')
    }
  }
}

// 选择变更处理
const handleSelectionChange = (selection) => {
  selectedExecutionIds.value = selection.map(item => item.id)
}

// 判断执行记录是否可选择（正在运行的不能选）
const isExecutionSelectable = (row) => {
  return row.status !== 'running' && row.status !== 'pending'
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
.workflow-list {
  padding: 20px;
}

.workflow-list .header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.workflow-list .header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.workflow-list .header .actions {
  display: flex;
  gap: 12px;
}

.workflow-list .header .actions .search-input {
  width: 300px;
}

.workflow-card {
  margin-bottom: 20px;
}

.execution-card {
  min-height: 400px;
}

.execution-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.execution-filters {
  display: flex;
  align-items: center;
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
