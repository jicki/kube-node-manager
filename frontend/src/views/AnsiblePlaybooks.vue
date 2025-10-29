<template>
  <div class="playbooks-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Ansible Playbook 管理</span>
          <el-button
            v-if="hasPermission('admin')"
            type="primary"
            @click="handleCreate"
          >
            <el-icon><Plus /></el-icon>
            创建 Playbook
          </el-button>
        </div>
      </template>

      <!-- 搜索筛选 -->
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="分类">
          <el-select v-model="searchForm.category" placeholder="全部" clearable>
            <el-option label="系统" value="system" />
            <el-option label="Docker" value="docker" />
            <el-option label="内核" value="kernel" />
            <el-option label="安全" value="security" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchPlaybooks">
            <el-icon><Search /></el-icon>
            查询
          </el-button>
        </el-form-item>
      </el-form>

      <!-- Playbook 列表 -->
      <el-table
        v-loading="loading"
        :data="playbooks"
        style="width: 100%"
      >
        <el-table-column prop="name" label="名称" min-width="200" />
        <el-table-column prop="description" label="描述" min-width="250" show-overflow-tooltip />
        <el-table-column prop="category" label="分类" width="100">
          <template #default="{ row }">
            <el-tag :type="getCategoryType(row.category)">
              {{ getCategoryLabel(row.category) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="version" label="版本" width="80" />
        <el-table-column prop="is_builtin" label="类型" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.is_builtin" type="info">内置</el-tag>
            <el-tag v-else type="success">自定义</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleView(row)">
              查看
            </el-button>
            <el-button size="small" type="primary" @click="handleExecute(row)">
              执行
            </el-button>
            <el-button
              v-if="!row.is_builtin && hasPermission('admin')"
              size="small"
              @click="handleEdit(row)"
            >
              编辑
            </el-button>
            <el-button
              v-if="!row.is_builtin && hasPermission('admin')"
              size="small"
              type="danger"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.size"
        :total="pagination.total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="fetchPlaybooks"
        @current-change="fetchPlaybooks"
        style="margin-top: 20px; justify-content: flex-end"
      />
    </el-card>

    <!-- 查看/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="70%"
      :close-on-click-modal="false"
    >
      <el-form :model="currentPlaybook" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="currentPlaybook.name" :disabled="dialogMode === 'view'" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="currentPlaybook.description" type="textarea" :disabled="dialogMode === 'view'" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="currentPlaybook.category" :disabled="dialogMode === 'view'">
            <el-option label="系统" value="system" />
            <el-option label="Docker" value="docker" />
            <el-option label="内核" value="kernel" />
            <el-option label="安全" value="security" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="Playbook 内容">
          <el-input
            v-model="currentPlaybook.content"
            type="textarea"
            :rows="15"
            :disabled="dialogMode === 'view'"
            style="font-family: monospace"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button v-if="dialogMode !== 'view'" type="primary" @click="handleSave" :loading="saving">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 执行对话框 -->
    <el-dialog
      v-model="executeDialogVisible"
      title="执行 Playbook"
      width="50%"
      :close-on-click-modal="false"
    >
      <el-form :model="executeForm" label-width="120px">
        <el-form-item label="Playbook">
          <el-input v-model="currentPlaybook.name" disabled />
        </el-form-item>
        <el-form-item label="集群">
          <el-select v-model="executeForm.cluster_name" placeholder="请选择集群">
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
            v-model="executeForm.target_nodes"
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
          <el-select v-model="executeForm.credential_id" placeholder="请选择 SSH 凭据">
            <el-option label="默认凭据" :value="1" />
          </el-select>
        </el-form-item>
        <el-form-item label="额外变量">
          <el-input
            v-model="executeForm.extra_vars_str"
            type="textarea"
            :rows="3"
            placeholder='{"key": "value"}'
          />
        </el-form-item>
        <el-form-item label="检查模式">
          <el-switch v-model="executeForm.check_mode" />
          <span style="margin-left: 10px; color: #909399; font-size: 12px">
            仅检查不实际执行
          </span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="executeDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleConfirmExecute" :loading="executing">
          执行
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { listPlaybooks, getPlaybook, createPlaybook, updatePlaybook, deletePlaybook, runPlaybook } from '@/api/ansible'
import { listClusters } from '@/api/cluster'
import { listNodes } from '@/api/node'
import { useAuthStore } from '@/store/modules/auth'

const authStore = useAuthStore()

const hasPermission = (permission) => {
  return authStore.hasPermission(permission)
}

// 数据
const loading = ref(false)
const playbooks = ref([])
const clusters = ref([])
const nodes = ref([])

const searchForm = reactive({
  category: ''
})

const pagination = reactive({
  page: 1,
  size: 20,
  total: 0
})

// 对话框
const dialogVisible = ref(false)
const dialogMode = ref('view') // view, create, edit
const dialogTitle = ref('')
const saving = ref(false)

const currentPlaybook = ref({
  name: '',
  description: '',
  category: 'custom',
  content: ''
})

// 执行对话框
const executeDialogVisible = ref(false)
const executing = ref(false)
const executeForm = reactive({
  playbook_id: 0,
  cluster_name: '',
  target_nodes: [],
  extra_vars_str: '',
  credential_id: 1,
  check_mode: false
})

// 分类标签
const getCategoryType = (category) => {
  const types = {
    system: 'primary',
    docker: 'success',
    kernel: 'warning',
    security: 'danger',
    custom: 'info'
  }
  return types[category] || 'info'
}

const getCategoryLabel = (category) => {
  const labels = {
    system: '系统',
    docker: 'Docker',
    kernel: '内核',
    security: '安全',
    custom: '自定义'
  }
  return labels[category] || category
}

// 获取 Playbook 列表
const fetchPlaybooks = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      size: pagination.size
    }
    if (searchForm.category) {
      params.category = searchForm.category
    }
    
    const res = await listPlaybooks(params)
    // 后端返回格式: res.data = { code, message, data: [...], total, page, size }
    const responseData = res.data || {}
    const playbookList = responseData.data || []
    playbooks.value = Array.isArray(playbookList) ? playbookList : []
    pagination.total = responseData.total || 0
  } catch (error) {
    ElMessage.error('获取 Playbook 列表失败')
  } finally {
    loading.value = false
  }
}

// 获取集群列表
const fetchClusters = async () => {
  try {
    const res = await listClusters()
    // 后端返回格式: res.data = { code, message, data: { clusters: [...], total, page, page_size } }
    const responseData = res.data?.data || {}
    const clusterList = responseData.clusters || []
    clusters.value = Array.isArray(clusterList) ? clusterList : []
  } catch (error) {
    console.error('获取集群列表失败', error)
  }
}

// 获取节点列表
const fetchNodes = async (clusterName) => {
  try {
    const res = await listNodes({ cluster: clusterName })
    // 后端返回格式: res.data = { code, message, data: [...] }
    const responseData = res.data || {}
    const nodeList = responseData.data || []
    nodes.value = Array.isArray(nodeList) ? nodeList.map(n => n.name || n) : []
  } catch (error) {
    console.error('获取节点列表失败', error)
  }
}

// 查看
const handleView = (row) => {
  dialogMode.value = 'view'
  dialogTitle.value = '查看 Playbook'
  currentPlaybook.value = { ...row }
  dialogVisible.value = true
}

// 创建
const handleCreate = () => {
  dialogMode.value = 'create'
  dialogTitle.value = '创建 Playbook'
  currentPlaybook.value = {
    name: '',
    description: '',
    category: 'custom',
    content: '---\n- name: Example Playbook\n  hosts: all\n  tasks:\n    - name: Ping\n      ping:\n'
  }
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row) => {
  dialogMode.value = 'edit'
  dialogTitle.value = '编辑 Playbook'
  currentPlaybook.value = { ...row }
  dialogVisible.value = true
}

// 保存
const handleSave = async () => {
  saving.value = true
  try {
    if (dialogMode.value === 'create') {
      await createPlaybook(currentPlaybook.value)
      ElMessage.success('创建成功')
    } else {
      await updatePlaybook(currentPlaybook.value.id, currentPlaybook.value)
      ElMessage.success('更新成功')
    }
    dialogVisible.value = false
    fetchPlaybooks()
  } catch (error) {
    ElMessage.error(error.message || '保存失败')
  } finally {
    saving.value = false
  }
}

// 删除
const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定要删除 Playbook "${row.name}" 吗？`, '确认删除', {
      type: 'warning'
    })
    await deletePlaybook(row.id)
    ElMessage.success('删除成功')
    fetchPlaybooks()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 执行
const handleExecute = async (row) => {
  currentPlaybook.value = { ...row }
  executeForm.playbook_id = row.id
  executeForm.cluster_name = ''
  executeForm.target_nodes = []
  executeForm.extra_vars_str = ''
  executeForm.check_mode = false
  executeDialogVisible.value = true
}

// 确认执行
const handleConfirmExecute = async () => {
  if (!executeForm.cluster_name) {
    ElMessage.warning('请选择集群')
    return
  }
  if (executeForm.target_nodes.length === 0) {
    ElMessage.warning('请选择目标节点')
    return
  }

  executing.value = true
  try {
    let extraVars = {}
    if (executeForm.extra_vars_str) {
      try {
        extraVars = JSON.parse(executeForm.extra_vars_str)
      } catch {
        ElMessage.warning('额外变量格式不正确，应为 JSON 格式')
        executing.value = false
        return
      }
    }

    const res = await runPlaybook({
      playbook_id: executeForm.playbook_id,
      cluster_name: executeForm.cluster_name,
      target_nodes: executeForm.target_nodes,
      extra_vars: extraVars,
      credential_id: executeForm.credential_id,
      check_mode: executeForm.check_mode
    })

    ElMessage.success('Playbook 执行已开始')
    executeDialogVisible.value = false
    
    // 可以跳转到执行历史页面查看进度
    console.log('Task ID:', res.data.task_id)
  } catch (error) {
    ElMessage.error(error.message || '执行失败')
  } finally {
    executing.value = false
  }
}

onMounted(() => {
  fetchPlaybooks()
  fetchClusters()
})
</script>

<style scoped>
.playbooks-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-form {
  margin-bottom: 20px;
}
</style>

