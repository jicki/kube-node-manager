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

    <!-- 标签模板列表 -->
    <div class="label-grid">
      <div
        v-for="template in filteredLabels"
        :key="template.id"
        class="label-card"
      >
        <div class="label-header">
          <div class="label-key-value">
            <div class="label-key">{{ template.name }}</div>
            <div class="label-value">{{ Object.keys(template.labels || {}).length }} 个标签</div>
          </div>
          <div class="label-actions">
            <el-dropdown @command="(cmd) => handleLabelAction(cmd, template)">
              <el-button type="text" size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">
                    <el-icon><Edit /></el-icon>
                    编辑
                  </el-dropdown-item>
                  <el-dropdown-item command="copy">
                    <el-icon><CopyDocument /></el-icon>
                    复制
                  </el-dropdown-item>
                  <el-dropdown-item command="apply">
                    <el-icon><Plus /></el-icon>
                    应用到节点
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided>
                    <el-icon><Delete /></el-icon>
                    删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <div class="label-meta">
          <el-tag type="primary" size="small">
            标签模板
          </el-tag>
          <span class="create-time">{{ template.created_at }}</span>
        </div>

        <div v-if="template.description" class="label-description">
          {{ template.description }}
        </div>

        <div class="label-content">
          <div class="labels-title">包含标签:</div>
          <div class="labels-list">
            <el-tag
              v-for="(value, key) in template.labels || {}"
              :key="key"
              size="small"
              class="label-item-tag"
            >
              {{ key }}{{ value ? `=${value}` : '' }}
            </el-tag>
          </div>
        </div>

        <div class="label-footer">
          <el-button-group size="small">
            <el-button @click="applyTemplateToNodes(template)">
              <el-icon><Plus /></el-icon>
              应用到节点
            </el-button>
            <el-button @click="editTemplate(template)">
              <el-icon><Edit /></el-icon>
              编辑模板
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
      :title="isEditing ? '编辑模板' : '创建模板'"
      width="800px"
      class="template-dialog"
    >
              <el-form
        ref="labelFormRef"
        :model="labelForm"
        :rules="labelRules"
        label-width="110px"
        style="margin-top: 20px;"
      >
        <el-form-item label="模板名称" prop="name">
          <el-input
            v-model="labelForm.name"
            placeholder="输入模板名称，如：Web应用标签、生产环境标签"
          />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="labelForm.description"
            type="textarea"
            :rows="2"
            placeholder="模板用途描述"
          />
        </el-form-item>

        <el-divider content-position="left">标签配置</el-divider>

        <div class="labels-config">
          <div 
            v-for="(label, index) in labelForm.labels" 
            :key="index"
            class="label-config-item"
          >
            <el-row :gutter="12" align="middle" class="label-row">
              <el-col :xs="24" :sm="11">
                <el-form-item 
                  :prop="`labels.${index}.key`" 
                  :rules="[{ required: true, message: '请输入标签键', trigger: 'blur' }]"
                  style="margin-bottom: 12px;"
                >
                  <el-input
                    v-model="label.key"
                    placeholder="标签键，如：app、version、environment"
                    size="large"
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="11">
                <el-form-item style="margin-bottom: 12px;">
                  <el-input
                    v-model="label.value"
                    placeholder="标签值，如：web、v1.0、production（可为空）"
                    size="large"
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="2" class="delete-col">
                <el-button
                  type="danger"
                  size="large"
                  :icon="Delete"
                  circle
                  @click="removeLabel(index)"
                  :disabled="labelForm.labels.length === 1"
                />
              </el-col>
            </el-row>
          </div>
          
          <el-button
            type="dashed"
            block
            size="large"
            @click="addLabel"
            :icon="Plus"
            class="add-label-btn"
          >
            添加标签
          </el-button>
        </div>
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

    <!-- 应用模板对话框 -->
    <el-dialog
      v-model="applyDialogVisible"
      :title="`应用模板: ${selectedTemplate?.name}`"
      width="600px"
    >
      <div class="template-info">
        <h4>模板包含的标签:</h4>
        <div class="template-labels">
          <el-tag
            v-for="(value, key) in selectedTemplate?.labels || {}"
            :key="key"
            class="label-tag"
            type="primary"
          >
            {{ key }}{{ value ? `=${value}` : '' }}
          </el-tag>
        </div>
      </div>

      <el-divider />

      <el-form label-width="100px">
        <el-form-item label="选择节点" required>
          <el-select
            v-model="selectedNodes"
            multiple
            filterable
            placeholder="选择要应用模板的节点"
            style="width: 100%"
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
                >
                  {{ node.status }}
                </el-tag>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="applyDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="applying"
          @click="handleApplyTemplate"
        >
          应用到节点
        </el-button>
      </template>
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
const applying = ref(false)
const searchKeyword = ref('')
const labelDialogVisible = ref(false)
const nodesDialogVisible = ref(false)
const applyDialogVisible = ref(false)
const isEditing = ref(false)
const labelFormRef = ref()

// 数据
const labels = ref([])
const availableNodes = ref([])
const selectedLabel = ref(null)
const selectedLabelNodes = ref([])
const selectedTemplate = ref(null)
const selectedNodes = ref([])

// 分页
const pagination = reactive({
  current: 1,
  size: 24,
  total: 0
})

// 表单数据
const labelForm = reactive({
  name: '',
  description: '',
  labels: [{ key: '', value: '' }]
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
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' }
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
    const response = await labelApi.getTemplateList({
      page: pagination.current,
      page_size: pagination.size
    })
    if (response.data && response.data.code === 200) {
      const data = response.data.data
      labels.value = data.templates || []
      pagination.total = data.total || 0
    } else {
      labels.value = []
      pagination.total = 0
    }
  } catch (error) {
    console.warn('获取标签模板失败:', error)
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
    name: '',
    description: '',
    labels: [{ key: '', value: '' }]
  })
}

// 添加标签
const addLabel = () => {
  labelForm.labels.push({ key: '', value: '' })
}

// 移除标签
const removeLabel = (index) => {
  if (labelForm.labels.length > 1) {
    labelForm.labels.splice(index, 1)
  }
}

// 处理标签操作
const handleLabelAction = (command, template) => {
  switch (command) {
    case 'edit':
      editTemplate(template)
      break
    case 'copy':
      copyTemplate(template)
      break
    case 'apply':
      applyTemplateToNodes(template)
      break
    case 'delete':
      deleteTemplate(template)
      break
  }
}

// 编辑模板
const editTemplate = (template) => {
  isEditing.value = true
  
  // 转换标签对象为数组
  const labelsArray = Object.entries(template.labels || {}).map(([key, value]) => ({ key, value }))
  
  Object.assign(labelForm, {
    name: template.name,
    description: template.description || '',
    labels: labelsArray.length > 0 ? labelsArray : [{ key: '', value: '' }]
  })
  
  // 保存当前编辑的模板ID
  labelForm.id = template.id
  
  labelDialogVisible.value = true
}

// 复制模板
const copyTemplate = (template) => {
  const labelsText = Object.entries(template.labels || {})
    .map(([key, value]) => value ? `${key}=${value}` : key)
    .join(', ')
  const text = `${template.name}: ${labelsText}`
  
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('模板信息已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

// 删除模板
const deleteTemplate = (template) => {
  ElMessageBox.confirm(
    `确认删除模板 "${template.name}" 吗？此操作不可撤销。`,
    '删除模板',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await labelApi.deleteTemplate(template.id)
      ElMessage.success('模板已删除')
      refreshData()
    } catch (error) {
      ElMessage.error(`删除模板失败: ${error.message}`)
    }
  }).catch(() => {
    // 用户取消
  })
}

// 保存模板
const handleSaveLabel = async () => {
  try {
    await labelFormRef.value.validate()
    saving.value = true
    
    // 验证标签配置
    const validLabels = labelForm.labels.filter(label => label.key.trim())
    if (validLabels.length === 0) {
      ElMessage.error('请至少添加一个有效的标签')
      return
    }
    
    // 转换标签数组为对象
    const labelsObj = {}
    validLabels.forEach(label => {
      labelsObj[label.key.trim()] = label.value.trim()
    })
    
    const templateData = {
      name: labelForm.name,
      description: labelForm.description,
      labels: labelsObj
    }
    
    if (isEditing.value) {
      // 更新模板
      await labelApi.updateTemplate(labelForm.id, templateData)
      ElMessage.success('模板更新成功')
    } else {
      // 创建新模板
      await labelApi.createTemplate(templateData)
      ElMessage.success('模板创建成功')
    }
    
    labelDialogVisible.value = false
    refreshData()
    
  } catch (error) {
    ElMessage.error(error.message || '保存模板失败')
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

// 应用模板到节点
const applyTemplateToNodes = (template) => {
  // 显示节点选择对话框
  showApplyDialog(template)
}

// 显示应用对话框
const showApplyDialog = (template) => {
  selectedTemplate.value = template
  applyDialogVisible.value = true
  fetchNodes() // 获取节点列表
}

// 应用模板
const handleApplyTemplate = async () => {
  try {
    if (selectedNodes.value.length === 0) {
      ElMessage.error('请选择要应用的节点')
      return
    }
    
    const clusterStore = useClusterStore()
    const clusterName = clusterStore.currentClusterName
    
    if (!clusterName) {
      ElMessage.error('请先选择集群')
      return
    }
    
    applying.value = true
    
    const applyData = {
      cluster_name: clusterName,
      node_names: selectedNodes.value,
      template_id: selectedTemplate.value.id,
      operation: 'add'
    }
    
    await labelApi.applyTemplate(applyData)
    ElMessage.success('模板应用成功')
    applyDialogVisible.value = false
    
  } catch (error) {
    ElMessage.error(`应用模板失败: ${error.message}`)
  } finally {
    applying.value = false
  }
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

/* 新增样式 */
.labels-config {
  margin-top: 16px;
}

.label-config-item {
  margin-bottom: 16px;
}

.create-time {
  color: #999;
  font-size: 12px;
}

.label-content {
  margin-bottom: 16px;
}

.labels-title {
  font-size: 13px;
  color: #666;
  font-weight: 500;
  margin-bottom: 8px;
}

.labels-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.label-item-tag {
  font-size: 11px;
  height: 20px;
  line-height: 18px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.template-info {
  margin-bottom: 16px;
}

.template-info h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #333;
}

.template-labels {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.label-tag {
  font-family: 'Monaco', 'Menlo', monospace;
}

/* 表单优化样式 */
.template-dialog {
  --el-dialog-border-radius: 8px;
}

.template-dialog :deep(.el-dialog__body) {
  padding: 20px 30px 30px;
}

.label-row {
  margin-bottom: 8px;
}

.delete-col {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding-top: 4px;
}

.add-label-btn {
  margin-top: 12px;
  height: 44px;
  border-style: dashed;
  border-color: #d9d9d9;
  color: #666;
  font-size: 14px;
}

.add-label-btn:hover {
  border-color: #1890ff;
  color: #1890ff;
}

.labels-config {
  background-color: #fafafa;
  border-radius: 6px;
  padding: 16px;
  border: 1px solid #f0f0f0;
}

.label-config-item {
  background-color: white;
  border-radius: 4px;
  padding: 12px;
  margin-bottom: 12px;
  border: 1px solid #e8e8e8;
}

.label-config-item:last-child {
  margin-bottom: 0;
}

/* 响应式优化 */
@media (max-width: 768px) {
  .template-dialog {
    --el-dialog-width: 95vw !important;
    --el-dialog-margin-top: 5vh !important;
  }
  
  .delete-col {
    justify-content: flex-start;
    padding-top: 0;
    margin-top: 8px;
  }
  
  .label-row .el-col {
    margin-bottom: 8px;
  }
}
</style>