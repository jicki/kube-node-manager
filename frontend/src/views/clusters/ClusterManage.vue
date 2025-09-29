<template>
  <div class="cluster-manage">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">集群管理</h1>
        <p class="page-description">管理Kubernetes集群连接配置</p>
      </div>
      <div class="header-right">
        <el-button 
          v-if="isAdmin || authStore.role === 'admin'" 
          type="primary" 
          @click="showAddDialog"
        >
          <el-icon><Plus /></el-icon>
          添加集群
        </el-button>
        <el-button @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 集群统计卡片 -->
    <div class="stats-cards">
      <el-card class="stat-card">
        <div class="stat-content">
          <div class="stat-icon total">
            <el-icon><Monitor /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ clusterStats.total }}</div>
            <div class="stat-label">总集群数</div>
          </div>
        </div>
      </el-card>
      
      <el-card class="stat-card">
        <div class="stat-content">
          <div class="stat-icon active">
            <el-icon><Check /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ clusterStats.active }}</div>
            <div class="stat-label">正常集群</div>
          </div>
        </div>
      </el-card>
      
      <el-card class="stat-card">
        <div class="stat-content">
          <div class="stat-icon inactive">
            <el-icon><Warning /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ clusterStats.inactive }}</div>
            <div class="stat-label">异常集群</div>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 集群列表 -->
    <el-card class="table-card">
      <el-table
        v-loading="loading"
        :data="clusters"
        style="width: 100%"
      >
        <!-- 空状态 -->
        <template #empty>
          <div class="empty-content">
            <el-empty
              description="暂无集群配置"
              :image-size="100"
            >
              <template #description>
                <p>您还没有配置任何Kubernetes集群</p>
                <p>请添加集群配置以开始管理节点</p>
              </template>
              <el-button 
                v-if="isAdmin || authStore.role === 'admin'" 
                type="primary" 
                @click="showAddDialog"
              >
                <el-icon><Plus /></el-icon>
                添加集群
              </el-button>
            </el-empty>
          </div>
        </template>

        <el-table-column prop="name" label="集群名称" min-width="150">
          <template #default="{ row }">
            <div class="cluster-name-cell">
              <el-icon class="cluster-icon"><Monitor /></el-icon>
              <span class="cluster-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />

        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 'active' ? 'success' : 'danger'"
              size="small"
            >
              {{ row.status === 'active' ? '正常' : '异常' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="version" label="版本" width="120">
          <template #default="{ row }">
            <span class="version-text">{{ row.version || 'N/A' }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="monitoring" label="监控状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.monitoring_enabled ? 'success' : 'info'"
              size="small"
            >
              {{ row.monitoring_enabled ? '已启用' : '未配置' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="current" label="当前集群" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.id === currentCluster?.id" type="primary" size="small">
              当前
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            <span class="time-text">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button 
                type="text" 
                size="small" 
                disabled
                title="测试连接功能暂时不可用"
              >
                <el-icon><Connection /></el-icon>
                测试连接
              </el-button>
              
              <el-button
                v-if="row.id !== currentCluster?.id"
                type="text"
                size="small"
                @click="switchCluster(row)"
              >
                <el-icon><Switch /></el-icon>
                切换
              </el-button>
              
              <el-button 
                v-if="isAdmin || authStore.role === 'admin'" 
                type="text" 
                size="small" 
                @click="showEditDialog(row)"
              >
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              
              <el-button 
                v-if="isAdmin || authStore.role === 'admin'"
                type="text" 
                size="small" 
                class="danger-button"
                @click="handleDelete(row)"
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑集群对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑集群' : '添加集群'"
      width="600px"
      @close="resetForm"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-width="120px"
      >
        <el-form-item label="集群名称" prop="name">
          <el-input
            v-model="form.name"
            placeholder="请输入集群名称"
          />
        </el-form-item>
        
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="2"
            placeholder="请输入集群描述"
          />
        </el-form-item>
        
        <el-form-item label="Kubeconfig" prop="kube_config">
          <el-input
            v-model="form.kube_config"
            type="textarea"
            :rows="8"
            placeholder="请粘贴Kubeconfig文件内容"
          />
          <div class="form-help-text">
            <el-alert
              title="权限要求"
              type="info"
              :closable="false"
              show-icon
            >
              <template #default>
                <p>为了正常使用节点管理功能，您的Kubeconfig需要包含以下权限：</p>
                <ul>
                  <li><strong>必需权限</strong>：API服务器连接权限</li>
                  <li><strong>推荐权限</strong>：nodes资源的get、list、patch权限（用于节点管理）</li>
                  <li><strong>可选权限</strong>：namespaces资源的list权限</li>
                </ul>
                <p><strong>建议</strong>：使用具有cluster-admin角色的Kubeconfig以获得完整功能。</p>
              </template>
            </el-alert>
          </div>
        </el-form-item>

        <!-- 监控配置 -->
        <el-form-item label="监控设置">
          <el-switch
            v-model="form.monitoring_enabled"
            active-text="启用监控"
            inactive-text="禁用监控"
          />
        </el-form-item>

        <template v-if="form.monitoring_enabled">
          <el-form-item label="监控类型" prop="monitoring_type">
            <el-radio-group v-model="form.monitoring_type">
              <el-radio label="prometheus">Prometheus</el-radio>
              <el-radio label="victoriametrics">VictoriaMetrics</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="监控地址" prop="monitoring_endpoint">
            <el-input
              v-model="form.monitoring_endpoint"
              placeholder="请输入监控系统地址，如: http://prometheus.example.com:9090"
            />
            <div class="form-help-text">
              <p class="help-tip">请确保此地址可以访问到 node-exporter 的指标数据</p>
            </div>
          </el-form-item>
        </template>
      </el-form>
      
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ isEdit ? '保存' : '添加' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useClusterStore } from '@/store/modules/cluster'
import { useAuthStore } from '@/store/modules/auth'
import { formatTime } from '@/utils/format'
import {
  Plus,
  Refresh,
  Monitor,
  Check,
  Warning,
  Connection,
  Switch,
  Edit,
  Delete
} from '@element-plus/icons-vue'

const clusterStore = useClusterStore()
const authStore = useAuthStore()

// 响应式数据
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref()
const testingClusters = ref({})

// 表单数据
const form = reactive({
  name: '',
  description: '',
  kube_config: '',
  monitoring_enabled: false,
  monitoring_type: 'prometheus',
  monitoring_endpoint: ''
})

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入集群名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  kube_config: [
    { required: true, message: '请输入Kubeconfig内容', trigger: 'blur' }
  ],
  monitoring_endpoint: [
    {
      validator: (rule, value, callback) => {
        if (form.monitoring_enabled && (!value || value.trim() === '')) {
          callback(new Error('请输入监控系统地址'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

// 计算属性
const clusters = computed(() => clusterStore.clusters)
const clusterStats = computed(() => clusterStore.clusterStats)
const currentCluster = computed(() => clusterStore.currentCluster)
const isAdmin = computed(() => {
  const user = authStore.user
  const userRole = user?.role
  const isAdminUser = userRole === 'admin'
  
  console.log('=== 权限检查调试信息 ===')
  console.log('用户信息:', user)
  console.log('用户角色:', userRole)
  console.log('是否为管理员:', isAdminUser)
  console.log('authStore.role getter:', authStore.role)
  console.log('========================')
  
  return isAdminUser
})

// 获取集群数据
const fetchClusters = async () => {
  try {
    loading.value = true
    await clusterStore.fetchClusters()
  } catch (error) {
    ElMessage.error('获取集群列表失败')
  } finally {
    loading.value = false
  }
}

// 刷新数据
const refreshData = () => {
  fetchClusters()
}

// 显示添加对话框
const showAddDialog = () => {
  isEdit.value = false
  dialogVisible.value = true
  resetForm()
}

// 显示编辑对话框
const showEditDialog = (cluster) => {
  isEdit.value = true
  dialogVisible.value = true

  // 填充表单数据
  Object.assign(form, {
    id: cluster.id,
    name: cluster.name,
    description: cluster.description || '',
    kube_config: cluster.kube_config || '',
    monitoring_enabled: cluster.monitoring_enabled || false,
    monitoring_type: cluster.monitoring_type || 'prometheus',
    monitoring_endpoint: cluster.monitoring_endpoint || ''
  })
}

// 重置表单
const resetForm = () => {
  Object.assign(form, {
    id: null,
    name: '',
    description: '',
    kube_config: '',
    monitoring_enabled: false,
    monitoring_type: 'prometheus',
    monitoring_endpoint: ''
  })

  if (formRef.value) {
    formRef.value.resetFields()
  }
}

// 处理提交
const handleSubmit = async () => {
  try {
    await formRef.value.validate()
    
    submitting.value = true
    
    // 添加调试日志
    console.log('提交集群数据:', form)
    
    if (isEdit.value) {
      await clusterStore.updateCluster(form.id, form)
      ElMessage.success('集群更新成功')
    } else {
      await clusterStore.addCluster(form)
      ElMessage.success('集群添加成功')
    }
    
    dialogVisible.value = false
    fetchClusters()
    
  } catch (error) {
    console.error('集群操作失败:', error)
    
    // 解析不同类型的错误并提供友好的错误信息
    let errorMessage = '操作失败'
    
    if (error.response?.data?.message) {
      const serverMessage = error.response.data.message
      
      // 检查是否是权限相关错误
      if (serverMessage.includes('forbidden') && serverMessage.includes('nodes')) {
        errorMessage = '集群连接失败：您的Kubeconfig没有列出节点的权限。请确保使用具有cluster-admin权限或包含nodes资源读取权限的Kubeconfig。'
      } else if (serverMessage.includes('invalid kubeconfig')) {
        errorMessage = '集群连接失败：Kubeconfig格式无效或权限不足。请检查Kubeconfig内容。'
      } else if (serverMessage.includes('failed to connect')) {
        errorMessage = '集群连接失败：无法连接到Kubernetes集群。请检查网络连接和API地址是否正确。'
      } else if (serverMessage.includes('already exists')) {
        errorMessage = '集群名称已存在，请使用不同的名称。'
      } else {
        errorMessage = serverMessage
      }
    } else if (error.message) {
      errorMessage = error.message
    }
    
    ElMessage.error(errorMessage)
  } finally {
    submitting.value = false
  }
}

// 测试连接
const testConnection = async (cluster) => {
  try {
    testingClusters.value[cluster.id] = true
    await clusterStore.testClusterConnection(cluster.id)
    ElMessage.success(`集群 ${cluster.name} 连接正常`)
  } catch (error) {
    ElMessage.error(`集群 ${cluster.name} 连接失败: ${error.message}`)
  } finally {
    testingClusters.value[cluster.id] = false
  }
}

// 切换集群
const switchCluster = async (cluster) => {
  try {
    clusterStore.setCurrentCluster(cluster)
    ElMessage.success(`已切换到集群: ${cluster.name}`)
  } catch (error) {
    ElMessage.error(`切换集群失败: ${error.message}`)
  }
}

// 处理删除
const handleDelete = (cluster) => {
  ElMessageBox.confirm(
    `确认删除集群 "${cluster.name}" 吗？`,
    '删除确认',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'el-button--danger'
    }
  ).then(async () => {
    try {
      await clusterStore.deleteCluster(cluster.id)
      ElMessage.success('集群删除成功')
      fetchClusters()
    } catch (error) {
      ElMessage.error(`删除集群失败: ${error.message}`)
    }
  }).catch(() => {
    // 用户取消
  })
}

onMounted(() => {
  fetchClusters()
})
</script>

<style scoped>
.cluster-manage {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left {
  flex: 1;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px 0;
}

.page-description {
  color: #666;
  margin: 0;
  font-size: 14px;
}

.header-right {
  display: flex;
  gap: 12px;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  cursor: pointer;
  transition: all 0.3s ease;
}

.stat-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.stat-content {
  display: flex;
  align-items: center;
  padding: 8px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 20px;
}

.stat-icon.total {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.stat-icon.active {
  background: linear-gradient(135deg, #52c41a 0%, #389e0d 100%);
  color: white;
}

.stat-icon.inactive {
  background: linear-gradient(135deg, #ff7875 0%, #ff4d4f 100%);
  color: white;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  color: #666;
  margin-top: 4px;
}

.table-card :deep(.el-card__body) {
  padding: 0;
}

.cluster-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cluster-icon {
  color: #1890ff;
}

.cluster-name {
  font-weight: 500;
}

.version-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #666;
}

.time-text {
  font-size: 13px;
  color: #666;
}

.action-buttons {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.danger-button:hover {
  color: #ff4d4f;
}

/* 表单帮助文本样式 */
.form-help-text {
  margin-top: 8px;
}

.form-help-text .el-alert {
  margin-top: 8px;
}

.form-help-text ul {
  margin: 8px 0;
  padding-left: 20px;
}

.form-help-text li {
  margin: 4px 0;
  line-height: 1.4;
}

.help-tip {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }
  
  .stats-cards {
    grid-template-columns: 1fr;
  }
  
  .action-buttons {
    flex-direction: column;
    gap: 2px;
  }
}
</style>
