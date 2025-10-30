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
          <el-select v-model="generateForm.cluster_id" placeholder="选择集群" style="width: 100%">
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
        <el-button @click="generateDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleGenerate" :loading="generating">生成</el-button>
      </template>
    </el-dialog>

    <!-- 创建/编辑对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="dialogTitle" 
      width="80%"
      :fullscreen="true"
    >
      <el-form :model="inventoryForm" label-width="120px">
        <el-form-item label="清单名称" required>
          <el-input v-model="inventoryForm.name" placeholder="请输入清单名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="inventoryForm.description" type="textarea" :rows="3" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="清单内容" required>
          <el-input 
            v-model="inventoryForm.content" 
            type="textarea" 
            :rows="20"
            placeholder="请输入 Ansible Inventory 内容（INI 格式）" 
            style="font-family: monospace"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Download } from '@element-plus/icons-vue'
import * as ansibleAPI from '@/api/ansible'
import * as clusterAPI from '@/api/cluster'

const inventories = ref([])
const total = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const generateDialogVisible = ref(false)
const dialogTitle = ref('')
const generating = ref(false)

const queryParams = reactive({
  page: 1,
  page_size: 20
})

const inventoryForm = reactive({
  id: null,
  name: '',
  description: '',
  content: ''
})

const generateForm = reactive({
  name: '',
  description: '',
  cluster_id: null
})

const clusters = ref([])

const loadInventories = async () => {
  loading.value = true
  try {
    const res = await ansibleAPI.listInventories(queryParams)
    inventories.value = res.data || []
    total.value = res.total || 0
  } catch (error) {
    ElMessage.error('加载清单失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const loadClusters = async () => {
  try {
    const res = await clusterAPI.listClusters()
    clusters.value = res.data || []
  } catch (error) {
    console.error('加载集群失败:', error)
  }
}

const showGenerateDialog = () => {
  Object.assign(generateForm, {
    name: '',
    description: '',
    cluster_id: null
  })
  generateDialogVisible.value = true
  loadClusters()
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
  dialogTitle.value = '手动创建清单'
  Object.assign(inventoryForm, {
    id: null,
    name: '',
    description: '',
    content: ''
  })
  dialogVisible.value = true
}

const handleView = async (row) => {
  dialogTitle.value = '查看清单'
  try {
    const res = await ansibleAPI.getInventory(row.id)
    Object.assign(inventoryForm, res.data)
    dialogVisible.value = true
  } catch (error) {
    ElMessage.error('加载清单失败: ' + error.message)
  }
}

const handleEdit = async (row) => {
  handleView(row)
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
    await ElMessageBox.confirm('确定要删除此清单吗？', '提示', {
      type: 'warning'
    })
    await ansibleAPI.deleteInventory(row.id)
    ElMessage.success('清单已删除')
    loadInventories()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + error.message)
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
</style>

