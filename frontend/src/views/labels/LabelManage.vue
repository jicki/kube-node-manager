<template>
  <div class="label-manage">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">标签管理</h1>
        <p class="page-description">管理Kubernetes节点标签，支持批量操作</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="showAddDialog">
          <el-icon><Plus /></el-icon>
          添加标签
        </el-button>
        <el-button @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="24" class="stats-row">
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.total }}</div>
            <div class="stat-label">总标签数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.active }}</div>
            <div class="stat-label">使用中</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.system }}</div>
            <div class="stat-label">系统标签</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.custom }}</div>
            <div class="stat-label">自定义</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <SearchBox
        v-model="searchKeyword"
        placeholder="搜索标签键值..."
        :advanced-search="true"
        :filters="searchFilters"
        :realtime="true"
        @search="handleSearch"
      />
    </el-card>

    <!-- 标签卡片列表 -->
    <div class="label-grid">
      <div
        v-for="label in filteredLabels"
        :key="label.key"
        class="label-card"
        :class="{ 'system-label': label.isSystem }"
      >
        <div class="label-header">
          <div class="label-key-value">
            <div class="label-key">{{ label.key }}</div>
            <div class="label-value">{{ label.value || '(空值)' }}</div>
          </div>
          <div class="label-actions">
            <el-dropdown @command="(cmd) => handleLabelAction(cmd, label)">
              <el-button type="text" size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit" :disabled="label.isSystem">
                    <el-icon><Edit /></el-icon>
                    编辑
                  </el-dropdown-item>
                  <el-dropdown-item command="copy">
                    <el-icon><CopyDocument /></el-icon>
                    复制
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" :disabled="label.isSystem">
                    <el-icon><Delete /></el-icon>
                    删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <div class="label-meta">
          <el-tag
            :type="label.isSystem ? 'info' : 'primary'"
            size="small"
          >
            {{ label.isSystem ? '系统标签' : '自定义标签' }}
          </el-tag>
          <span class="node-count">{{ label.nodeCount || 0 }} 个节点</span>
        </div>

        <div v-if="label.description" class="label-description">
          {{ label.description }}
        </div>

        <div class="label-nodes">
          <div class="nodes-header">
            <span class="nodes-title">关联节点:</span>
            <el-button
              type="text"
              size="small"
              @click="showLabelNodes(label)"
            >
              查看全部
            </el-button>
          </div>
          <div class="nodes-list">
            <el-tag
              v-for="node in (label.nodes || []).slice(0, 5)"
              :key="node"
              size="small"
              class="node-tag"
            >
              {{ node }}
            </el-tag>
            <span
              v-if="(label.nodes || []).length > 5"
              class="more-nodes"
            >
              +{{ (label.nodes || []).length - 5 }} 个
            </span>
          </div>
        </div>

        <div class="label-footer">
          <el-button-group size="small">
            <el-button @click="applyToNodes(label)">
              <el-icon><Plus /></el-icon>
              应用到节点
            </el-button>
            <el-button @click="removeFromNodes(label)">
              <el-icon><Minus /></el-icon>
              从节点移除
            </el-button>
          </el-button-group>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="filteredLabels.length === 0" class="empty-state">
        <el-empty description="暂无标签数据" :image-size="80">
          <el-button type="primary" @click="showAddDialog">
            <el-icon><Plus /></el-icon>
            添加第一个标签
          </el-button>
        </el-empty>
      </div>
    </div>

    <!-- 分页 -->
    <div class="pagination-container">
      <el-pagination
        v-model:current-page="pagination.current"
        v-model:page-size="pagination.size"
        :page-sizes="[12, 24, 48, 96]"
        :total="pagination.total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <!-- 添加/编辑标签对话框 -->
    <el-dialog
      v-model="labelDialogVisible"
      :title="isEditing ? '编辑标签' : '添加标签'"
      width="600px"
    >
              <el-form
        ref="labelFormRef"
        :model="labelForm"
        :rules="labelRules"
        label-width="110px"
        style="margin-top: 20px;"
      >
        <el-form-item label="标签键" prop="key">
          <el-input
            v-model="labelForm.key"
            placeholder="例如: app, version, environment"
            :disabled="isEditing"
          />
        </el-form-item>

        <el-form-item label="标签值" prop="value">
          <el-input
            v-model="labelForm.value"
            placeholder="例如: web, v1.0, production（可为空）"
          />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="labelForm.description"
            type="textarea"
            :rows="3"
            placeholder="标签用途描述"
          />
        </el-form-item>

        <el-form-item label="应用到节点" style="margin-top: 24px;">
          <el-select
            v-model="labelForm.selectedNodes"
            multiple
            filterable
            placeholder="选择要应用此标签的节点"
            style="width: 100%; font-size: 14px;"
            size="large"
          >
            <el-option
              v-for="node in availableNodes"
              :key="node.name"
              :label="`${node.name} (${node.status})`"
              :value="node.name"
            >
              <div class="node-option">
                <span class="node-name">{{ node.name }}</span>
                <el-tag 
                  :type="node.status === 'Ready' ? 'success' : 'danger'" 
                  size="small"
                  style="margin-left: auto;"
                >
                  {{ node.status }}
                </el-tag>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="labelDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="saving"
          @click="handleSaveLabel"
        >
          {{ isEditing ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 节点列表对话框 -->
    <el-dialog
      v-model="nodesDialogVisible"
      :title="`标签 ${selectedLabel?.key} 关联的节点`"
      width="800px"
    >
      <el-table :data="selectedLabelNodes" style="width: 100%">
        <el-table-column prop="name" label="节点名称" />
        <el-table-column prop="status" label="状态">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Ready' ? 'success' : 'danger'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="roles" label="角色">
          <template #default="{ row }">
            <el-tag
              v-for="role in row.roles"
              :key="role"
              :type="role === 'master' ? 'danger' : 'primary'"
              size="small"
            >
              {{ role }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="{ row }">
            <el-button
              type="text"
              size="small"
              @click="removeLabelFromNode(row)"
            >
              移除标签
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import labelApi from '@/api/label'
import nodeApi from '@/api/node'
import { useClusterStore } from '@/store/modules/cluster'
import SearchBox from '@/components/common/SearchBox.vue'
import {
  Plus,
  Refresh,
  MoreFilled,
  Edit,
  Delete,
  CopyDocument,
  Minus
} from '@element-plus/icons-vue'

// 响应式数据
const loading = ref(false)
const saving = ref(false)
const searchKeyword = ref('')
const labelDialogVisible = ref(false)
const nodesDialogVisible = ref(false)
const isEditing = ref(false)
const labelFormRef = ref()

// 数据
const labels = ref([])
const availableNodes = ref([])
const selectedLabel = ref(null)
const selectedLabelNodes = ref([])

// 分页
const pagination = reactive({
  current: 1,
  size: 24,
  total: 0
})

// 表单数据
const labelForm = reactive({
  key: '',
  value: '',
  description: '',
  selectedNodes: []
})

// 搜索筛选
const searchFilters = ref([
  {
    key: 'type',
    label: '类型',
    type: 'select',
    placeholder: '选择标签类型',
    options: [
      { label: '全部', value: '' },
      { label: '系统标签', value: 'system' },
      { label: '自定义标签', value: 'custom' }
    ]
  },
  {
    key: 'usage',
    label: '使用状态',
    type: 'select',
    placeholder: '选择使用状态',
    options: [
      { label: '全部', value: '' },
      { label: '使用中', value: 'used' },
      { label: '未使用', value: 'unused' }
    ]
  }
])

// 表单验证规则
const labelRules = {
  key: [
    { required: true, message: '请输入标签键', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9._/-]+$/, message: '标签键只能包含字母、数字、点、下划线、斜杠和横线', trigger: 'blur' }
  ]
}

// 计算属性
const labelStats = computed(() => {
  const total = labels.value.length
  const system = labels.value.filter(l => l.isSystem).length
  const custom = total - system
  const active = labels.value.filter(l => l.nodeCount > 0).length
  
  return { total, system, custom, active }
})

const filteredLabels = computed(() => {
  // 这里可以添加搜索和筛选逻辑
  return labels.value.slice(
    (pagination.current - 1) * pagination.size,
    pagination.current * pagination.size
  )
})

// 方法
const fetchLabels = async () => {
  try {
    loading.value = true
    // 暂时使用空数据，避免404错误
    // TODO: 实现正确的标签数据获取逻辑
    labels.value = []
    pagination.total = 0
  } catch (error) {
    console.warn('获取标签数据失败，使用空列表')
    labels.value = []
    pagination.total = 0
  } finally {
    loading.value = false
  }
}

const fetchNodes = async () => {
  try {
    const clusterStore = useClusterStore()
    const clusterName = clusterStore.currentClusterName
    
    // 如果没有集群，直接设置为空数组
    if (!clusterName) {
      availableNodes.value = []
      return
    }
    
    const response = await nodeApi.getNodes({
      cluster_name: clusterName
    })
    // 后端返回格式: { code, message, data: [...] } - data直接是节点数组
    availableNodes.value = response.data.data || []
  } catch (error) {
    console.error('获取节点数据失败:', error)
    availableNodes.value = []
  }
}

const refreshData = () => {
  fetchLabels()
  fetchNodes()
}

const handleSearch = (params) => {
  // 实现搜索逻辑
  console.log('Search params:', params)
}

const handleSizeChange = (size) => {
  pagination.size = size
  pagination.current = 1
}

const handleCurrentChange = (current) => {
  pagination.current = current
}

// 显示添加对话框
const showAddDialog = () => {
  isEditing.value = false
  resetLabelForm()
  labelDialogVisible.value = true
}

// 重置表单
const resetLabelForm = () => {
  Object.assign(labelForm, {
    key: '',
    value: '',
    description: '',
    selectedNodes: []
  })
}

// 处理标签操作
const handleLabelAction = (command, label) => {
  switch (command) {
    case 'edit':
      editLabel(label)
      break
    case 'copy':
      copyLabel(label)
      break
    case 'delete':
      deleteLabel(label)
      break
  }
}

// 编辑标签
const editLabel = (label) => {
  isEditing.value = true
  Object.assign(labelForm, {
    key: label.key,
    value: label.value || '',
    description: label.description || '',
    selectedNodes: label.nodes || []
  })
  labelDialogVisible.value = true
}

// 复制标签
const copyLabel = (label) => {
  const text = label.value ? `${label.key}=${label.value}` : label.key
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('标签已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

// 删除标签
const deleteLabel = (label) => {
  ElMessageBox.confirm(
    `确认删除标签 "${label.key}" 吗？此操作将从所有关联节点中移除该标签。`,
    '删除标签',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      // 获取当前集群名称
      const clusterStore = useClusterStore()
      const clusterName = clusterStore.currentClusterName
      
      if (!clusterName) {
        ElMessage.error('请先选择集群')
        return
      }
      
      // 构建删除请求数据
      const deleteData = {
        nodes: label.nodes || [],
        keys: [label.key],
        cluster: clusterName
      }
      
      await labelApi.batchDeleteLabels(deleteData)
      ElMessage.success('标签已删除')
      refreshData()
    } catch (error) {
      ElMessage.error(`删除标签失败: ${error.message}`)
    }
  }).catch(() => {
    // 用户取消
  })
}

// 保存标签
const handleSaveLabel = async () => {
  try {
    await labelFormRef.value.validate()
    saving.value = true
    
    // 获取当前集群名称
    const clusterStore = useClusterStore()
    const clusterName = clusterStore.currentClusterName
    
    if (!clusterName) {
      ElMessage.error('请先选择集群')
      return
    }
    
    if (labelForm.selectedNodes.length === 0) {
      ElMessage.error('请选择要应用标签的节点')
      return
    }
    
    const labelData = {
      key: labelForm.key,
      value: labelForm.value,
      description: labelForm.description
    }
    
    // 构建请求数据，包含集群名称
    const requestData = {
      nodes: labelForm.selectedNodes,
      labels: [labelData],
      cluster: clusterName
    }
    
    if (isEditing.value) {
      // 更新标签
      await labelApi.batchAddLabels(requestData)
    } else {
      // 创建新标签
      await labelApi.batchAddLabels(requestData)
    }
    
    ElMessage.success(isEditing.value ? '标签更新成功' : '标签创建成功')
    labelDialogVisible.value = false
    refreshData()
    
  } catch (error) {
    ElMessage.error(error.message || '保存标签失败')
  } finally {
    saving.value = false
  }
}

// 显示标签关联的节点
const showLabelNodes = async (label) => {
  selectedLabel.value = label
  try {
    // 这里应该获取真实的节点数据
    selectedLabelNodes.value = (label.nodes || []).map(nodeName => ({
      name: nodeName,
      status: 'Ready', // 模拟数据
      roles: ['worker'] // 模拟数据
    }))
    nodesDialogVisible.value = true
  } catch (error) {
    ElMessage.error('获取节点信息失败')
  }
}

// 应用标签到节点
const applyToNodes = (label) => {
  // 实现应用标签逻辑
  ElMessage.info('功能开发中')
}

// 从节点移除标签
const removeFromNodes = (label) => {
  // 实现移除标签逻辑
  ElMessage.info('功能开发中')
}

// 从单个节点移除标签
const removeLabelFromNode = async (node) => {
  try {
    // 获取当前集群名称
    const clusterStore = useClusterStore()
    const clusterName = clusterStore.currentClusterName
    
    if (!clusterName) {
      ElMessage.error('请先选择集群')
      return
    }
    
    // 使用URL参数传递cluster_name
    const response = await labelApi.deleteNodeLabel(node.name, selectedLabel.value.key, { cluster_name: clusterName })
    ElMessage.success(`已从节点 ${node.name} 移除标签`)
    // 刷新数据
    showLabelNodes(selectedLabel.value)
    refreshData()
  } catch (error) {
    ElMessage.error(`移除标签失败: ${error.message}`)
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped>
.node-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  font-size: 14px;
}

.node-name {
  font-weight: 500;
  color: #333;
  letter-spacing: 0.3px;
}

:deep(.el-form-item) {
  margin-bottom: 22px;
}

:deep(.el-form-item__label) {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  letter-spacing: 0.2px;
  line-height: 1.6;
}

:deep(.el-input__wrapper) {
  font-size: 14px;
  padding: 12px 15px;
}

:deep(.el-textarea__inner) {
  font-size: 14px;
  padding: 12px 15px;
  line-height: 1.6;
}

:deep(.el-select) {
  font-size: 14px;
}

:deep(.el-select__wrapper) {
  padding: 8px 15px;
  min-height: 44px;
}

.label-manage {
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

.stats-row {
  margin-bottom: 24px;
}

.stat-card {
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.stat-content {
  padding: 16px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1890ff;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #666;
}

.search-card {
  margin-bottom: 24px;
}

.label-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.label-card {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 20px;
  background: #fff;
  transition: all 0.3s;
  position: relative;
}

.label-card:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.label-card.system-label {
  border-left: 4px solid #722ed1;
}

.label-card:not(.system-label) {
  border-left: 4px solid #1890ff;
}

.label-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.label-key-value {
  flex: 1;
}

.label-key {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.label-value {
  font-size: 14px;
  color: #666;
  font-family: 'Monaco', 'Menlo', monospace;
}

.label-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 13px;
}

.node-count {
  color: #666;
}

.label-description {
  font-size: 13px;
  color: #666;
  margin-bottom: 16px;
  line-height: 1.5;
}

.label-nodes {
  margin-bottom: 16px;
}

.nodes-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.nodes-title {
  font-size: 13px;
  color: #666;
  font-weight: 500;
}

.nodes-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.node-tag {
  font-size: 11px;
  height: 20px;
  line-height: 18px;
}

.more-nodes {
  font-size: 12px;
  color: #999;
}

.label-footer {
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
}

.empty-state {
  grid-column: 1 / -1;
  padding: 40px;
  text-align: center;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

.node-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.node-name {
  flex: 1;
  text-align: left;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }
  
  .label-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .stats-row .el-col {
    margin-bottom: 16px;
  }
}

@media (max-width: 480px) {
  .label-card {
    padding: 16px;
  }
  
  .label-header {
    flex-direction: column;
    gap: 8px;
  }
}
</style>