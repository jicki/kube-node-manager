<template>
  <div class="ansible-inventories">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>主机清单管理</span>
          <div>
            <el-button type="success" @click="showGenerateDialog">
              <el-icon><Download /></el-icon>
              从集群生成
            </el-button>
            <el-button type="primary" @click="showCreateDialog">
              <el-icon><Plus /></el-icon>
              手动创建
            </el-button>
          </div>
        </div>
      </template>

      <!-- 清单列表 -->
      <el-table :data="inventories" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="清单名称" min-width="200" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="来源" width="120">
          <template #default="{ row }">
            <el-tag :type="row.source_type === 'k8s' ? 'success' : ''">
              {{ row.source_type === 'k8s' ? 'K8s集群' : '手动' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cluster.name" label="集群" width="150" />
        <el-table-column label="主机数" width="100">
          <template #default="{ row }">
            {{ row.hosts_data?.total || 0 }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleView(row)">查看</el-button>
            <el-button size="small" type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button 
              size="small" 
              type="success" 
              @click="handleRefresh(row)" 
              v-if="row.source_type === 'k8s'"
            >
              刷新
            </el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="queryParams.page"
        v-model:page-size="queryParams.page_size"
        :page-sizes="[10, 20, 50]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadInventories"
        @current-change="loadInventories"
        style="margin-top: 20px"
      />
    </el-card>

    <!-- 从集群生成对话框 -->
    <el-dialog v-model="generateDialogVisible" title="从集群生成主机清单" width="600px">
      <el-form :model="generateForm" label-width="120px">
        <el-form-item label="清单名称" required>
          <el-input v-model="generateForm.name" placeholder="请输入清单名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="generateForm.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="选择集群" required>
          <el-select 
            v-model="generateForm.cluster_id" 
            placeholder="选择集群" 
            style="width: 100%"
            clearable
            filterable
          >
            <el-option 
              v-for="cluster in clusters" 
              :key="cluster.id" 
              :label="cluster.name" 
              :value="cluster.id" 
            />
          </el-select>
          <div style="color: #999; font-size: 12px; margin-top: 5px;">
            {{ clusters.length > 0 ? `共 ${clusters.length} 个集群` : '暂无集群数据，请先添加集群' }}
          </div>
        </el-form-item>
        <el-form-item label="SSH 密钥">
          <el-select 
            v-model="generateForm.ssh_key_id" 
            placeholder="选择 SSH 密钥（可选）" 
            style="width: 100%"
            clearable
            filterable
          >
            <el-option 
              v-for="key in sshKeys" 
              :key="key.id" 
              :label="`${key.name} (${key.username}@${key.type})`" 
              :value="key.id" 
            />
          </el-select>
          <div style="color: #999; font-size: 12px; margin-top: 5px;">
            选择用于连接主机的 SSH 密钥，不选则使用默认密钥
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="generateDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleGenerate" :loading="generating">生成</el-button>
      </template>
    </el-dialog>

    <!-- 创建/编辑对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="dialogTitle" 
      width="60%"
      :close-on-click-modal="false"
    >
      <el-form :model="inventoryForm" label-width="120px">
        <el-form-item label="清单名称" :required="!isViewMode">
          <el-input v-model="inventoryForm.name" placeholder="请输入清单名称" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="inventoryForm.description" type="textarea" :rows="3" placeholder="请输入描述" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item label="SSH 密钥">
          <el-select 
            v-model="inventoryForm.ssh_key_id" 
            placeholder="选择 SSH 密钥（可选）" 
            style="width: 100%"
            clearable
            filterable
            :disabled="isViewMode"
          >
            <el-option 
              v-for="key in sshKeys" 
              :key="key.id" 
              :label="`${key.name} (${key.username}@${key.type})`" 
              :value="key.id" 
            />
          </el-select>
          <div style="color: #999; font-size: 12px; margin-top: 5px;">
            选择用于连接主机的 SSH 密钥
          </div>
        </el-form-item>
        <el-form-item label="清单内容" :required="!isViewMode">
          <el-input 
            v-model="inventoryForm.content" 
            type="textarea" 
            :rows="15"
            placeholder="请输入 Ansible Inventory 内容（INI 格式）&#10;&#10;示例：&#10;[webservers]&#10;192.168.1.10 ansible_user=root&#10;192.168.1.11 ansible_user=root&#10;&#10;[dbservers]&#10;192.168.1.20 ansible_user=root" 
            style="font-family: 'Courier New', monospace; font-size: 13px;"
            :disabled="isViewMode"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">{{ isViewMode ? '关闭' : '取消' }}</el-button>
          <el-button v-if="!isViewMode" type="primary" @click="handleSave" :loading="saving">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Download } from '@element-plus/icons-vue'
import * as ansibleAPI from '@/api/ansible'
import clusterAPI from '@/api/cluster'

const inventories = ref([])
const total = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const generateDialogVisible = ref(false)
const dialogTitle = ref('')
const generating = ref(false)
const saving = ref(false)
const isViewMode = ref(false) // 是否为查看模式（只读）

const queryParams = reactive({
  page: 1,
  page_size: 20
})

const inventoryForm = reactive({
  id: null,
  name: '',
  description: '',
  content: '',
  ssh_key_id: null
})

const generateForm = reactive({
  name: '',
  description: '',
  cluster_id: null,
  ssh_key_id: null
})

const clusters = ref([])
const sshKeys = ref([])

const loadInventories = async () => {
  loading.value = true
  try {
    const res = await ansibleAPI.listInventories(queryParams)
    console.log('清单列表响应:', res)
    console.log('清单数据:', res.data)
    // axios拦截器返回完整response，所以路径是: res.data.data 和 res.data.total
    inventories.value = res.data?.data || []
    total.value = res.data?.total || 0
    console.log('已加载清单:', inventories.value.length, '个')
  } catch (error) {
    console.error('加载清单失败:', error)
    ElMessage.error('加载清单失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
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

const loadSSHKeys = async () => {
  try {
    const res = await ansibleAPI.listSSHKeys({ page_size: 100 })
    console.log('SSH密钥列表响应:', res)
    // axios拦截器返回完整response，所以路径是: res.data.data
    sshKeys.value = res.data?.data || []
    console.log('已加载SSH密钥:', sshKeys.value.length, '个')
  } catch (error) {
    console.error('加载SSH密钥失败:', error)
  }
}

const showGenerateDialog = () => {
  Object.assign(generateForm, {
    name: '',
    description: '',
    cluster_id: null,
    ssh_key_id: null
  })
  generateDialogVisible.value = true
  loadClusters()
  loadSSHKeys()
}

const handleGenerate = async () => {
  if (!generateForm.name || !generateForm.cluster_id) {
    ElMessage.warning('请填写必填项')
    return
  }

  generating.value = true
  try {
    await ansibleAPI.generateInventory(generateForm)
    ElMessage.success('主机清单已生成')
    generateDialogVisible.value = false
    loadInventories()
  } catch (error) {
    ElMessage.error('生成失败: ' + error.message)
  } finally {
    generating.value = false
  }
}

const showCreateDialog = () => {
  isViewMode.value = false // 创建模式，可编辑
  dialogTitle.value = '手动创建清单'
  Object.assign(inventoryForm, {
    id: null,
    name: '',
    description: '',
    content: '',
    ssh_key_id: null
  })
  dialogVisible.value = true
  loadSSHKeys()
}

const handleSave = async () => {
  if (!inventoryForm.name || !inventoryForm.content) {
    ElMessage.warning('请填写必填项')
    return
  }

  saving.value = true
  try {
    const data = {
      name: inventoryForm.name,
      description: inventoryForm.description,
      source_type: 'manual',
      content: inventoryForm.content,
      ssh_key_id: inventoryForm.ssh_key_id || null
    }

    if (inventoryForm.id) {
      // 更新
      await ansibleAPI.updateInventory(inventoryForm.id, data)
      ElMessage.success('清单已更新')
    } else {
      // 创建
      await ansibleAPI.createInventory(data)
      ElMessage.success('清单已创建')
    }
    
    dialogVisible.value = false
    loadInventories()
  } catch (error) {
    console.error('保存清单失败:', error)
    ElMessage.error('保存失败: ' + (error.message || '未知错误'))
  } finally {
    saving.value = false
  }
}

const handleView = async (row) => {
  isViewMode.value = true // 查看模式，只读
  dialogTitle.value = '查看清单'
  try {
    const res = await ansibleAPI.getInventory(row.id)
    console.log('清单详情响应:', res)
    console.log('清单详情数据:', res.data)
    // axios拦截器返回完整response，所以路径是: res.data.data
    Object.assign(inventoryForm, res.data?.data || {})
    dialogVisible.value = true
    loadSSHKeys() // 加载 SSH 密钥列表
  } catch (error) {
    console.error('加载清单失败:', error)
    ElMessage.error('加载清单失败: ' + (error.message || '未知错误'))
  }
}

const handleEdit = async (row) => {
  isViewMode.value = false // 编辑模式，可编辑
  dialogTitle.value = '编辑清单'
  try {
    const res = await ansibleAPI.getInventory(row.id)
    console.log('编辑清单响应:', res)
    // axios拦截器返回完整response，所以路径是: res.data.data
    Object.assign(inventoryForm, res.data?.data || {})
    dialogVisible.value = true
    loadSSHKeys() // 加载 SSH 密钥列表
  } catch (error) {
    console.error('加载清单失败:', error)
    ElMessage.error('加载清单失败: ' + (error.message || '未知错误'))
  }
}

const handleRefresh = async (row) => {
  try {
    await ansibleAPI.refreshInventory(row.id)
    ElMessage.success('清单已刷新')
    loadInventories()
  } catch (error) {
    ElMessage.error('刷新失败: ' + error.message)
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除此清单吗？删除后无法恢复。',
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await ansibleAPI.deleteInventory(row.id)
    ElMessage.success('清单已删除')
    loadInventories()
  } catch (error) {
    if (error !== 'cancel') {
      const errorMsg = error.message || error.toString()
      if (errorMsg.includes('tasks are using this inventory')) {
        // 提取任务数量
        const match = errorMsg.match(/(\d+) tasks/)
        const taskCount = match ? match[1] : '若干'
        ElMessage.error({
          message: `无法删除：有 ${taskCount} 个任务正在使用此清单。请先删除这些任务后再试。`,
          duration: 5000
        })
      } else {
        ElMessage.error('删除失败: ' + errorMsg)
      }
    }
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadInventories()
})
</script>

<style scoped>
.ansible-inventories {
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
</style>

