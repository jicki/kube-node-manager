<template>
  <div class="ssh-command-container">
    <el-card>
      <template #header>
        <span>SSH 命令执行</span>
      </template>

      <el-form :model="form" label-width="120px">
        <el-form-item label="集群">
          <el-select v-model="form.cluster_name" placeholder="请选择集群" @change="handleClusterChange">
            <el-option
              v-for="cluster in clusters"
              :key="cluster.name"
              :label="cluster.name"
              :value="cluster.name"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="目标节点">
          <el-select
            v-model="form.target_nodes"
            multiple
            placeholder="请选择节点"
            style="width: 100%"
          >
            <el-option
              v-for="node in nodes"
              :key="node"
              :label="node"
              :value="node"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="SSH 凭据">
          <el-select v-model="form.credential_id" placeholder="请选择 SSH 凭据">
            <el-option label="默认凭据" :value="1" />
          </el-select>
        </el-form-item>

        <el-form-item label="命令">
          <el-input
            v-model="form.command"
            type="textarea"
            :rows="5"
            placeholder="输入要执行的命令，例如: uptime"
          />
        </el-form-item>

        <el-form-item label="超时时间(秒)">
          <el-input-number v-model="form.timeout" :min="10" :max="600" />
        </el-form-item>

        <el-form-item label="并发数">
          <el-input-number v-model="form.concurrent" :min="1" :max="50" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleExecute" :loading="executing">
            <el-icon><Connection /></el-icon>
            执行
          </el-button>
          <el-button @click="handleReset">
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 执行历史 -->
    <el-card style="margin-top: 20px">
      <template #header>
        <span>执行历史</span>
      </template>

      <el-table
        v-loading="loading"
        :data="executions"
        style="width: 100%"
      >
        <el-table-column prop="task_id" label="任务 ID" width="200" show-overflow-tooltip />
        <el-table-column prop="command" label="命令" min-width="200" show-overflow-tooltip />
        <el-table-column prop="cluster_name" label="集群" width="120" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="success_count" label="成功/失败" width="120">
          <template #default="{ row }">
            <span style="color: #67c23a">{{ row.success_count }}</span> /
            <span style="color: #f56c6c">{{ row.failed_count }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="执行时间" width="180" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleViewResult(row)">
              查看结果
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.size"
        :total="pagination.total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @size-change="fetchExecutions"
        @current-change="fetchExecutions"
        style="margin-top: 20px; justify-content: flex-end"
      />
    </el-card>

    <!-- 结果对话框 -->
    <el-dialog
      v-model="resultDialogVisible"
      title="执行结果"
      width="70%"
    >
      <el-descriptions :column="2" border>
        <el-descriptions-item label="任务 ID">{{ currentExecution.task_id }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusType(currentExecution.status)">
            {{ getStatusLabel(currentExecution.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="成功数">{{ currentExecution.success_count }}</el-descriptions-item>
        <el-descriptions-item label="失败数">{{ currentExecution.failed_count }}</el-descriptions-item>
      </el-descriptions>
      
      <div style="margin-top: 20px">
        <h4>执行结果详情</h4>
        <el-input
          v-model="resultOutput"
          type="textarea"
          :rows="15"
          readonly
          style="font-family: monospace"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection } from '@element-plus/icons-vue'
import { executeCommand, listExecutions, getExecutionStatus } from '@/api/ssh'
import { listClusters } from '@/api/cluster'
import { listNodes } from '@/api/node'

const clusters = ref([])
const nodes = ref([])
const executing = ref(false)
const loading = ref(false)

const form = reactive({
  cluster_name: '',
  target_nodes: [],
  command: '',
  credential_id: 1,
  timeout: 30,
  concurrent: 10
})

const executions = ref([])
const pagination = reactive({
  page: 1,
  size: 20,
  total: 0
})

const resultDialogVisible = ref(false)
const currentExecution = ref({})
const resultOutput = ref('')

const getStatusType = (status) => {
  const types = {
    pending: 'info',
    running: 'primary',
    completed: 'success',
    failed: 'danger',
    partial: 'warning'
  }
  return types[status] || 'info'
}

const getStatusLabel = (status) => {
  const labels = {
    pending: '等待中',
    running: '执行中',
    completed: '已完成',
    failed: '失败',
    partial: '部分成功'
  }
  return labels[status] || status
}

const fetchClusters = async () => {
  try {
    const res = await listClusters()
    clusters.value = res.data || []
  } catch (error) {
    console.error('获取集群列表失败', error)
  }
}

const handleClusterChange = async () => {
  form.target_nodes = []
  try {
    const res = await listNodes({ cluster: form.cluster_name })
    nodes.value = res.data?.map(n => n.name) || []
  } catch (error) {
    console.error('获取节点列表失败', error)
  }
}

const handleExecute = async () => {
  if (!form.cluster_name) {
    ElMessage.warning('请选择集群')
    return
  }
  if (form.target_nodes.length === 0) {
    ElMessage.warning('请选择目标节点')
    return
  }
  if (!form.command.trim()) {
    ElMessage.warning('请输入要执行的命令')
    return
  }

  executing.value = true
  try {
    const res = await executeCommand(form)
    ElMessage.success('命令执行已开始')
    handleReset()
    fetchExecutions()
  } catch (error) {
    ElMessage.error(error.message || '执行失败')
  } finally {
    executing.value = false
  }
}

const handleReset = () => {
  form.command = ''
  form.target_nodes = []
}

const fetchExecutions = async () => {
  loading.value = true
  try {
    const res = await listExecutions({
      page: pagination.page,
      size: pagination.size
    })
    executions.value = res.data || []
    pagination.total = res.total || 0
  } catch (error) {
    ElMessage.error('获取执行历史失败')
  } finally {
    loading.value = false
  }
}

const handleViewResult = async (row) => {
  currentExecution.value = row
  resultDialogVisible.value = true
  
  // 格式化结果显示
  try {
    const results = JSON.parse(row.results || '[]')
    resultOutput.value = results.map(r => {
      return `[${r.Host}]\nExit Code: ${r.ExitCode}\nStdout:\n${r.Stdout}\nStderr:\n${r.Stderr}\n${'='.repeat(80)}\n`
    }).join('\n')
  } catch {
    resultOutput.value = row.results || '无结果'
  }
}

onMounted(() => {
  fetchClusters()
  fetchExecutions()
})
</script>

<style scoped>
.ssh-command-container {
  padding: 20px;
}
</style>

