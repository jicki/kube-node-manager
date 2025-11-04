<template>
  <div class="workflow-execution">
    <div class="header">
      <h2>工作流执行监控</h2>
      <div class="actions">
        <el-button @click="handleBack">返回</el-button>
        <el-button 
          v-if="execution?.status === 'running'"
          type="danger"
          icon="el-icon-close"
          @click="handleCancel"
        >
          取消执行
        </el-button>
        <el-button 
          type="primary" 
          icon="el-icon-refresh" 
          @click="loadExecution"
          :loading="loading"
        >
          刷新
        </el-button>
      </div>
    </div>

    <div v-if="execution" class="content">
      <!-- 执行信息 -->
      <el-card class="info-card">
        <template #header>
          <div class="card-header">
            <span>执行信息</span>
            <el-tag :type="getStatusType(execution.status)">
              {{ getStatusText(execution.status) }}
            </el-tag>
          </div>
        </template>
        <el-descriptions :column="3" border>
          <el-descriptions-item label="执行 ID">{{ execution.id }}</el-descriptions-item>
          <el-descriptions-item label="工作流名称">{{ execution.workflow?.name }}</el-descriptions-item>
          <el-descriptions-item label="工作流 ID">{{ execution.workflow_id }}</el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ formatTime(execution.started_at) }}</el-descriptions-item>
          <el-descriptions-item label="完成时间">
            {{ execution.finished_at ? formatTime(execution.finished_at) : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="执行时长">
            {{ getElapsedTime(execution) }}
          </el-descriptions-item>
          <el-descriptions-item v-if="execution.error_message" label="错误信息" :span="3">
            <el-alert type="error" :closable="false">
              {{ execution.error_message }}
            </el-alert>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- DAG 可视化 -->
      <el-card class="dag-card">
        <template #header>
          <div class="card-header">
            <span>工作流 DAG（实时状态）</span>
            <el-button size="small" @click="refreshStatus">刷新状态</el-button>
          </div>
        </template>
        <div class="dag-view" ref="dagViewRef">
          <svg width="100%" height="100%">
            <!-- 绘制边 -->
            <g v-for="edge in execution.workflow?.dag?.edges || []" :key="edge.id">
              <path
                :d="getEdgePath(edge)"
                stroke="#999"
                stroke-width="2"
                fill="none"
                marker-end="url(#arrowhead-exec)"
              />
            </g>
            
            <!-- 箭头标记 -->
            <defs>
              <marker
                id="arrowhead-exec"
                markerWidth="10"
                markerHeight="10"
                refX="9"
                refY="3"
                orient="auto"
              >
                <polygon points="0 0, 10 3, 0 6" fill="#999" />
              </marker>
            </defs>
          </svg>

          <!-- 节点 -->
          <div
            v-for="node in execution.workflow?.dag?.nodes || []"
            :key="node.id"
            class="node"
            :class="[node.type, getNodeStatusClass(node.id)]"
            :style="{
              left: node.position.x + 'px',
              top: node.position.y + 'px'
            }"
            @click="selectNode(node)"
          >
            <div class="node-header">
              <span class="node-label">{{ node.label }}</span>
              <el-tag size="small" :type="getNodeStatusType(node.id)">
                {{ getNodeStatusText(node.id) }}
              </el-tag>
            </div>
            <div v-if="node.type === 'task'" class="node-body">
              <div class="node-info">{{ node.task_config?.name || '未命名任务' }}</div>
            </div>
            <!-- 状态指示器 -->
            <div class="status-indicator" :class="getNodeStatusClass(node.id)"></div>
          </div>
        </div>
      </el-card>

      <!-- 任务列表 -->
      <el-card class="tasks-card">
        <template #header>
          <span>任务列表</span>
        </template>
        <el-table :data="execution.tasks || []" stripe>
          <el-table-column prop="node_id" label="节点 ID" width="150" />
          <el-table-column prop="name" label="任务名称" min-width="200" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="started_at" label="开始时间" width="180">
            <template #default="{ row }">
              {{ row.started_at ? formatTime(row.started_at) : '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="finished_at" label="完成时间" width="180">
            <template #default="{ row }">
              {{ row.finished_at ? formatTime(row.finished_at) : '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="duration" label="时长" width="100">
            <template #default="{ row }">
              {{ row.duration ? row.duration + 's' : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" @click="viewTaskLog(row)">
                查看日志
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <div v-else-if="loading" class="loading">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>加载中...</span>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { 
  getWorkflowExecution, 
  getWorkflowExecutionStatus,
  cancelWorkflowExecution 
} from '@/api/workflow'
import { formatTime } from '@/utils/format'

const route = useRoute()
const router = useRouter()

const execution = ref(null)
const nodeStatus = ref({})
const loading = ref(false)
const autoRefreshTimer = ref(null)

// 加载执行详情
const loadExecution = async () => {
  loading.value = true
  try {
    const executionId = parseInt(route.params.id)
    const response = await getWorkflowExecution(executionId)
    execution.value = response.data
    
    // 如果正在运行，加载实时状态
    if (execution.value.status === 'running') {
      await refreshStatus()
    }
  } catch (error) {
    console.error('Failed to load execution:', error)
    ElMessage.error(error.response?.data?.error || '加载执行详情失败')
  } finally {
    loading.value = false
  }
}

// 刷新节点状态
const refreshStatus = async () => {
  try {
    const executionId = parseInt(route.params.id)
    const response = await getWorkflowExecutionStatus(executionId)
    nodeStatus.value = response.data.node_status || {}
  } catch (error) {
    // 忽略错误（可能是执行已完成）
    console.log('Status refresh skipped:', error.response?.status)
  }
}

// 获取节点状态
const getNodeStatus = (nodeId) => {
  return nodeStatus.value[nodeId] || 'pending'
}

// 获取节点状态样式类
const getNodeStatusClass = (nodeId) => {
  const status = getNodeStatus(nodeId)
  return `status-${status}`
}

// 获取节点状态类型
const getNodeStatusType = (nodeId) => {
  const statusMap = {
    pending: '',
    running: 'warning',
    success: 'success',
    failed: 'danger',
    skipped: 'info'
  }
  return statusMap[getNodeStatus(nodeId)] || ''
}

// 获取节点状态文本
const getNodeStatusText = (nodeId) => {
  const statusMap = {
    pending: '等待中',
    running: '执行中',
    success: '成功',
    failed: '失败',
    skipped: '跳过'
  }
  return statusMap[getNodeStatus(nodeId)] || '未知'
}

// 获取状态类型
const getStatusType = (status) => {
  const typeMap = {
    running: 'warning',
    success: 'success',
    failed: 'danger',
    cancelled: 'info',
    pending: ''
  }
  return typeMap[status] || ''
}

// 获取状态文本
const getStatusText = (status) => {
  const textMap = {
    running: '执行中',
    success: '成功',
    failed: '失败',
    cancelled: '已取消',
    pending: '等待中'
  }
  return textMap[status] || status
}

// 获取执行时长
const getElapsedTime = (execution) => {
  if (!execution.started_at) return '-'
  
  const start = new Date(execution.started_at)
  const end = execution.finished_at ? new Date(execution.finished_at) : new Date()
  const diff = Math.floor((end - start) / 1000)
  
  const hours = Math.floor(diff / 3600)
  const minutes = Math.floor((diff % 3600) / 60)
  const seconds = diff % 60
  
  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`
  } else if (minutes > 0) {
    return `${minutes}m ${seconds}s`
  } else {
    return `${seconds}s`
  }
}

// 获取边的路径（与编辑器保持一致）
const getEdgePath = (edge) => {
  const sourceNode = execution.value.workflow?.dag?.nodes.find(n => n.id === edge.source)
  const targetNode = execution.value.workflow?.dag?.nodes.find(n => n.id === edge.target)
  
  if (!sourceNode || !targetNode) return ''

  const x1 = sourceNode.position.x + 100
  const y1 = sourceNode.position.y + 40
  const x2 = targetNode.position.x + 100
  const y2 = targetNode.position.y

  const cx = (x1 + x2) / 2
  return `M ${x1} ${y1} Q ${cx} ${y1}, ${cx} ${(y1 + y2) / 2} T ${x2} ${y2}`
}

// 选择节点
const selectNode = (node) => {
  if (node.type === 'task') {
    const task = execution.value.tasks?.find(t => t.node_id === node.id)
    if (task) {
      viewTaskLog(task)
    }
  }
}

// 查看任务日志
const viewTaskLog = (task) => {
  router.push(`/ansible/tasks/${task.id}`)
}

// 取消执行
const handleCancel = async () => {
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

    await cancelWorkflowExecution(execution.value.id)
    ElMessage.success('工作流执行已取消')
    await loadExecution()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to cancel execution:', error)
      ElMessage.error(error.response?.data?.error || '取消执行失败')
    }
  }
}

// 返回
const handleBack = () => {
  router.back()
}

// 启动自动刷新
const startAutoRefresh = () => {
  autoRefreshTimer.value = setInterval(async () => {
    if (execution.value?.status === 'running') {
      await refreshStatus()
      await loadExecution()
    } else {
      stopAutoRefresh()
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
  await loadExecution()
  if (execution.value?.status === 'running') {
    startAutoRefresh()
  }
})

// 清理
onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.workflow-execution {
  padding: 20px;
}

.workflow-execution .header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.workflow-execution .header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.workflow-execution .header .actions {
  display: flex;
  gap: 12px;
}

.workflow-execution .content .el-card {
  margin-bottom: 20px;
}

.workflow-execution .content .el-card .card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.workflow-execution .content .dag-card .dag-view {
  position: relative;
  height: 600px;
  background: 
    linear-gradient(90deg, #e5e5e5 1px, transparent 1px),
    linear-gradient(#e5e5e5 1px, transparent 1px);
  background-size: 20px 20px;
  overflow: hidden;
}

.workflow-execution .content .dag-card .dag-view svg {
  position: absolute;
  top: 0;
  left: 0;
  pointer-events: none;
}

.workflow-execution .content .dag-card .dag-view .node {
  position: absolute;
  width: 200px;
  background: white;
  border: 2px solid #ddd;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  transition: all 0.2s;
}

.workflow-execution .content .dag-card .dag-view .node:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.workflow-execution .content .dag-card .dag-view .node.status-pending {
  border-color: #909399;
}

.workflow-execution .content .dag-card .dag-view .node.status-running {
  border-color: #e6a23c;
  animation: pulse 1.5s infinite;
}

.workflow-execution .content .dag-card .dag-view .node.status-success {
  border-color: #67c23a;
}

.workflow-execution .content .dag-card .dag-view .node.status-failed {
  border-color: #f56c6c;
}

.workflow-execution .content .dag-card .dag-view .node.status-skipped {
  border-color: #c0c4cc;
  opacity: 0.6;
}

.workflow-execution .content .dag-card .dag-view .node .node-header {
  padding: 8px 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #f5f5f5;
  border-radius: 6px 6px 0 0;
  font-weight: 600;
}

.workflow-execution .content .dag-card .dag-view .node .node-header .node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workflow-execution .content .dag-card .dag-view .node .node-body {
  padding: 12px;
}

.workflow-execution .content .dag-card .dag-view .node .node-body .node-info {
  font-size: 13px;
  color: #666;
}

.workflow-execution .content .dag-card .dag-view .node .status-indicator {
  position: absolute;
  top: -5px;
  right: -5px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid white;
}

.workflow-execution .content .dag-card .dag-view .node .status-indicator.status-pending {
  background: #909399;
}

.workflow-execution .content .dag-card .dag-view .node .status-indicator.status-running {
  background: #e6a23c;
  animation: blink 1s infinite;
}

.workflow-execution .content .dag-card .dag-view .node .status-indicator.status-success {
  background: #67c23a;
}

.workflow-execution .content .dag-card .dag-view .node .status-indicator.status-failed {
  background: #f56c6c;
}

.workflow-execution .content .dag-card .dag-view .node .status-indicator.status-skipped {
  background: #c0c4cc;
}

.workflow-execution .loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 400px;
  gap: 16px;
  font-size: 16px;
  color: #666;
}

.workflow-execution .loading .el-icon {
  font-size: 48px;
}

@keyframes pulse {
  0%, 100% {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  }
  50% {
    box-shadow: 0 4px 16px rgba(230, 162, 60, 0.4);
  }
}

@keyframes blink {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}
</style>

