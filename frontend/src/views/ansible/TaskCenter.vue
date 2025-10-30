<template>
  <div class="ansible-task-center">
    <el-card class="header-card">
      <template #header>
        <div class="card-header">
          <span>Ansible 任务中心</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            启动任务
          </el-button>
        </div>
      </template>
      
      <!-- 统计卡片 -->
      <el-row :gutter="20">
        <el-col :span="6">
          <el-statistic title="总任务数" :value="statistics.total_tasks || 0" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="运行中" :value="statistics.running_tasks || 0" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="成功" :value="statistics.status_counts?.success || 0" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="失败" :value="statistics.status_counts?.failed || 0" />
        </el-col>
      </el-row>
    </el-card>

    <!-- 筛选器 -->
    <el-card style="margin-top: 20px">
      <el-form :inline="true" :model="queryParams">
        <el-form-item label="状态">
          <el-select v-model="queryParams.status" placeholder="全部" clearable style="width: 120px">
            <el-option label="待执行" value="pending" />
            <el-option label="运行中" value="running" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键字">
          <el-input v-model="queryParams.keyword" placeholder="搜索任务名称" clearable style="width: 200px" @keyup.enter="handleQuery" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleQuery">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
          <el-button @click="handleRefresh" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新状态
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 任务列表 -->
    <el-card style="margin-top: 20px">
      <div style="margin-bottom: 16px">
        <el-button 
          type="danger" 
          :disabled="selectedTasks.length === 0"
          @click="handleBatchDelete"
        >
          批量删除 ({{ selectedTasks.length }})
        </el-button>
        <el-text type="info" size="small" style="margin-left: 16px">
          提示：只能删除已完成、失败或取消的任务
        </el-text>
      </div>
      <el-table 
        :data="tasks" 
        v-loading="loading" 
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" :selectable="canSelectTask" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="任务名称" min-width="200" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="150">
          <template #default="{ row }">
            <div v-if="row.status === 'running'">
              <el-progress 
                :percentage="calculateProgress(row)" 
                :status="row.hosts_failed > 0 ? 'exception' : 'success'" 
              />
            </div>
            <div v-else-if="row.status === 'success' || row.status === 'failed'">
              {{ row.hosts_ok }}/{{ row.hosts_total }} 成功
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="duration" label="耗时" width="100">
          <template #default="{ row }">
            {{ row.duration ? `${row.duration}秒` : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleViewLogs(row)">查看日志</el-button>
            <el-button 
              size="small" 
              type="warning" 
              @click="handleCancel(row)" 
              v-if="row.status === 'running'"
            >
              取消
            </el-button>
            <el-button 
              size="small" 
              type="primary" 
              @click="handleRetry(row)" 
              v-if="row.status === 'failed'"
            >
              重试
            </el-button>
            <el-button 
              size="small" 
              type="danger" 
              @click="handleDelete(row)" 
              v-if="row.status !== 'running' && row.status !== 'pending'"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="queryParams.page"
        v-model:page-size="queryParams.page_size"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleQuery"
        @current-change="handleQuery"
        style="margin-top: 20px"
      />
    </el-card>

    <!-- 创建任务对话框 -->
    <el-dialog v-model="createDialogVisible" title="启动 Ansible 任务" width="800px">
      <el-form :model="taskForm" label-width="120px">
        <el-form-item label="任务名称" required>
          <el-input v-model="taskForm.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="选择模板">
          <el-select v-model="taskForm.template_id" placeholder="选择模板（可选）" clearable style="width: 100%">
            <el-option 
              v-for="template in templates" 
              :key="template.id" 
              :label="template.name" 
              :value="template.id" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="主机清单" required>
          <el-select v-model="taskForm.inventory_id" placeholder="选择主机清单" style="width: 100%">
            <el-option 
              v-for="inventory in inventories" 
              :key="inventory.id" 
              :label="inventory.name" 
              :value="inventory.id" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="集群">
          <el-select v-model="taskForm.cluster_id" placeholder="选择集群（可选）" clearable style="width: 100%">
            <el-option 
              v-for="cluster in clusters" 
              :key="cluster.id" 
              :label="cluster.name" 
              :value="cluster.id" 
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="creating">启动任务</el-button>
      </template>
    </el-dialog>

    <!-- 日志对话框 -->
    <el-dialog 
      v-model="logDialogVisible" 
      title="任务日志" 
      width="70%"
      :close-on-click-modal="false"
    >
      <div class="log-container">
        <el-scrollbar max-height="500px">
          <pre class="log-content">{{ logContent || '暂无日志' }}</pre>
        </el-scrollbar>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="logDialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="copyLogs" v-if="logContent">复制日志</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import * as ansibleAPI from '@/api/ansible'
import clusterAPI from '@/api/cluster'

// 数据
const tasks = ref([])
const total = ref(0)
const loading = ref(false)
const statistics = ref({})
const selectedTasks = ref([])

const queryParams = reactive({
  page: 1,
  page_size: 20,
  status: '',
  keyword: ''
})

const createDialogVisible = ref(false)
const logDialogVisible = ref(false)
const creating = ref(false)

const taskForm = reactive({
  name: '',
  template_id: null,
  cluster_id: null,
  inventory_id: null
})

const templates = ref([])
const inventories = ref([])
const clusters = ref([])
const logContent = ref('')

// 方法
const loadTasks = async () => {
  loading.value = true
  try {
    const res = await ansibleAPI.listTasks(queryParams)
    console.log('任务列表响应:', res)
    // axios拦截器返回完整response，所以路径是: res.data.data 和 res.data.total
    tasks.value = res.data?.data || []
    total.value = res.data?.total || 0
  } catch (error) {
    console.error('加载任务列表失败:', error)
    ElMessage.error('加载任务列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const loadStatistics = async () => {
  try {
    const res = await ansibleAPI.getStatistics()
    // axios拦截器返回完整response，所以路径是: res.data.data
    statistics.value = res.data?.data || {}
  } catch (error) {
    console.error('加载统计信息失败:', error)
  }
}

const loadTemplates = async () => {
  try {
    const res = await ansibleAPI.listTemplates({ page_size: 100 })
    // axios拦截器返回完整response，所以路径是: res.data.data
    templates.value = res.data?.data || []
  } catch (error) {
    console.error('加载模板失败:', error)
  }
}

const loadInventories = async () => {
  try {
    const res = await ansibleAPI.listInventories({ page_size: 100 })
    // axios拦截器返回完整response，所以路径是: res.data.data
    inventories.value = res.data?.data || []
  } catch (error) {
    console.error('加载主机清单失败:', error)
  }
}

const loadClusters = async () => {
  try {
    const res = await clusterAPI.getClusters()
    console.log('集群API完整响应:', res)
    console.log('响应数据:', res.data)
    // axios拦截器返回完整response，所以路径是: res.data.data.clusters
    clusters.value = res.data?.data?.clusters || []
    console.log('已加载集群:', clusters.value.length, '个', clusters.value)
  } catch (error) {
    console.error('加载集群失败:', error)
    ElMessage.error('加载集群失败: ' + error.message)
  }
}

const handleQuery = () => {
  queryParams.page = 1
  loadTasks()
}

const handleReset = () => {
  queryParams.status = ''
  queryParams.keyword = ''
  handleQuery()
}

const handleRefresh = () => {
  loadTasks()
  loadStatistics()
}

const showCreateDialog = () => {
  createDialogVisible.value = true
  loadTemplates()
  loadInventories()
  loadClusters()
}

const handleCreate = async () => {
  if (!taskForm.name) {
    ElMessage.warning('请输入任务名称')
    return
  }
  if (!taskForm.inventory_id) {
    ElMessage.warning('请选择主机清单')
    return
  }

  creating.value = true
  try {
    await ansibleAPI.createTask(taskForm)
    ElMessage.success('任务已启动')
    createDialogVisible.value = false
    loadTasks()
    loadStatistics()
  } catch (error) {
    ElMessage.error('启动任务失败: ' + error.message)
  } finally {
    creating.value = false
  }
}

const handleViewLogs = async (row) => {
  logDialogVisible.value = true
  try {
    const res = await ansibleAPI.getTaskLogs(row.id, { full: true })
    console.log('任务日志响应:', res)
    // axios拦截器返回完整response，数据是字符串格式
    logContent.value = res.data?.data || '暂无日志'
  } catch (error) {
    console.error('加载日志失败:', error)
    ElMessage.error('加载日志失败: ' + (error.message || '未知错误'))
  }
}

const handleCancel = async (row) => {
  try {
    await ElMessageBox.confirm('确定要取消此任务吗？', '提示', {
      type: 'warning'
    })
    await ansibleAPI.cancelTask(row.id)
    ElMessage.success('任务已取消')
    loadTasks()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('取消任务失败: ' + error.message)
    }
  }
}

const handleRetry = async (row) => {
  try {
    await ansibleAPI.retryTask(row.id)
    ElMessage.success('任务已重新启动')
    loadTasks()
  } catch (error) {
    ElMessage.error('重试任务失败: ' + error.message)
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除任务 "${row.name}" 吗？此操作不可恢复。`,
      '删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await ansibleAPI.deleteTask(row.id)
    ElMessage.success('删除成功')
    loadTasks()
    loadStatistics()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleBatchDelete = async () => {
  if (selectedTasks.value.length === 0) {
    ElMessage.warning('请先选择要删除的任务')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedTasks.value.length} 个任务吗？此操作不可恢复。`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    const ids = selectedTasks.value.map(task => task.id)
    const res = await ansibleAPI.batchDeleteTasks(ids)
    
    const successCount = res.data?.success_count || 0
    const failedCount = res.data?.failed_count || 0
    
    if (failedCount > 0) {
      ElMessage.warning(`成功删除 ${successCount} 个任务，${failedCount} 个任务删除失败`)
      console.error('删除失败的任务:', res.data?.errors)
    } else {
      ElMessage.success(`成功删除 ${successCount} 个任务`)
    }
    
    selectedTasks.value = []
    loadTasks()
    loadStatistics()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleSelectionChange = (selection) => {
  selectedTasks.value = selection
}

const canSelectTask = (row) => {
  // 只能选择已完成、失败或取消的任务
  return row.status !== 'running' && row.status !== 'pending'
}

const copyLogs = async () => {
  try {
    await navigator.clipboard.writeText(logContent.value)
    ElMessage.success('日志已复制到剪贴板')
  } catch (error) {
    console.error('复制日志失败:', error)
    // 降级方案：使用传统方法
    const textArea = document.createElement('textarea')
    textArea.value = logContent.value
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      ElMessage.success('日志已复制到剪贴板')
    } catch (err) {
      ElMessage.error('复制失败，请手动复制')
    }
    document.body.removeChild(textArea)
  }
}

const getStatusType = (status) => {
  const types = {
    pending: '',
    running: 'warning',
    success: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return types[status] || ''
}

const getStatusText = (status) => {
  const texts = {
    pending: '待执行',
    running: '运行中',
    success: '成功',
    failed: '失败',
    cancelled: '已取消'
  }
  return texts[status] || status
}

const calculateProgress = (task) => {
  if (task.hosts_total === 0) return 0
  return Math.round((task.hosts_ok + task.hosts_failed) / task.hosts_total * 100)
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 生命周期
let refreshTimer = null

onMounted(() => {
  loadTasks()
  loadStatistics()
  
  // 每 5 秒自动刷新
  refreshTimer = setInterval(() => {
    if (tasks.value.some(t => t.status === 'running')) {
      loadTasks()
      loadStatistics()
    }
  }, 5000)
})

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.ansible-task-center {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.log-container {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: 4px;
  border: 1px solid #3e3e3e;
}

.log-content {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  color: #d4d4d4;
}

/* 优化日志中不同类型的文本颜色 */
.log-content :deep(.error) {
  color: #f48771;
}

.log-content :deep(.success) {
  color: #89d185;
}

.log-content :deep(.warning) {
  color: #e5c07b;
}
</style>

