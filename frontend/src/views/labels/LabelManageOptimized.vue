<template>
  <div class="label-manage-optimized">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">标签管理 (优化版)</h1>
        <p class="page-description">管理Kubernetes节点标签，支持批量操作进度显示和撤销重做</p>
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
            <div class="stat-label">模板数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.single }}</div>
            <div class="stat-label">单个标签</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.multiple }}</div>
            <div class="stat-label">多个标签组</div>
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
        @clear="handleSearchClear"
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

        <div class="label-content">
          <div v-if="template.labels && Object.keys(template.labels).length > 0" class="label-tags">
            <el-tag
              v-for="(value, key) in template.labels"
              :key="key"
              size="small"
              class="label-tag"
              :title="`${key}=${value}`"
            >
              <span class="tag-key">{{ key }}</span>=<span class="tag-value">{{ value }}</span>
            </el-tag>
          </div>
          <div v-else class="no-labels">
            暂无标签内容
          </div>
        </div>

        <div class="label-footer">
          <div class="template-info">
            <span class="created-time">{{ formatTime(template.created_at) }}</span>
          </div>
          <div class="template-actions">
            <el-button size="small" @click="applyTemplateToNodes(template)">
              应用到节点
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 应用标签对话框 -->
    <el-dialog
      v-model="applyDialogVisible"
      title="应用标签模板到节点"
      width="800px"
      destroy-on-close
    >
      <div class="apply-dialog-content">
        <div class="template-summary">
          <h4>模板信息</h4>
          <div class="template-details">
            <p><strong>名称:</strong> {{ selectedTemplate?.name }}</p>
            <p><strong>标签数量:</strong> {{ Object.keys(selectedTemplate?.labels || {}).length }}</p>
          </div>
        </div>

        <div class="node-selector-section">
          <h4>选择目标节点</h4>
          <NodeSelector
            v-model="selectedNodes"
            :available-nodes="availableNodes"
            :loading="nodeDialogLoading"
            multiple
          />
        </div>

        <div class="selected-summary" v-if="selectedNodes.length > 0">
          <p>将应用到 <strong>{{ selectedNodes.length }}</strong> 个节点</p>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="applyDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            @click="confirmApplyTemplate"
            :loading="applying"
            :disabled="selectedNodes.length === 0"
          >
            <el-icon v-if="!applying"><Check /></el-icon>
            {{ applying ? '应用中...' : '确认应用' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 批量操作进度弹窗 -->
    <BatchProgressDialog
      v-model="progressDialogVisible"
      :operation-id="currentOperationId"
      @cancel="handleProgressCancel"
      @retry="handleProgressRetry"
    />

    <!-- 撤销/重做栏 -->
    <UndoRedoBar
      :auto-hide="true"
      @undo="handleUndo"
      @redo="handleRedo"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useClusterStore } from '@/store/modules/cluster'
import { useAuthStore } from '@/store/modules/auth'
import { useProgressStore } from '@/store/modules/progress'
import { useHistoryStore } from '@/store/modules/history'
import labelApi from '@/api/label'
import nodeApi from '@/api/node'
import SearchBox from '@/components/common/SearchBox.vue'
import NodeSelector from '@/components/common/NodeSelector.vue'
import BatchProgressDialog from '@/components/common/BatchProgressDialog.vue'
import UndoRedoBar from '@/components/common/UndoRedoBar.vue'
import { formatTime } from '@/utils/format'
import {
  Plus,
  Refresh,
  MoreFilled,
  Edit,
  CopyDocument,
  Delete,
  Check
} from '@element-plus/icons-vue'

// Store实例
const clusterStore = useClusterStore()
const authStore = useAuthStore()
const progressStore = useProgressStore()
const historyStore = useHistoryStore()

// 响应式数据
const labelTemplates = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const selectedTemplate = ref(null)
const availableNodes = ref([])
const selectedNodes = ref([])
const applying = ref(false)
const nodeDialogLoading = ref(false)

// 批量操作进度相关
const progressDialogVisible = ref(false)
const currentOperationId = ref('')

// 对话框状态
const applyDialogVisible = ref(false)

// 搜索筛选器
const searchFilters = ref([
  { key: 'name', label: '模板名称', type: 'input' },
  { key: 'labels', label: '标签内容', type: 'input' },
  { key: 'created_at', label: '创建时间', type: 'date' }
])

// 计算属性
const labelStats = computed(() => {
  return {
    total: labelTemplates.value.reduce((total, template) => {
      return total + Object.keys(template.labels || {}).length
    }, 0),
    active: labelTemplates.value.length,
    single: labelTemplates.value.filter(t => Object.keys(t.labels || {}).length === 1).length,
    multiple: labelTemplates.value.filter(t => Object.keys(t.labels || {}).length > 1).length
  }
})

const filteredLabels = computed(() => {
  if (!searchKeyword.value) {
    return labelTemplates.value
  }
  
  const keyword = searchKeyword.value.toLowerCase()
  return labelTemplates.value.filter(template => {
    return template.name?.toLowerCase().includes(keyword) ||
           Object.keys(template.labels || {}).some(key => 
             key.toLowerCase().includes(keyword) ||
             String(template.labels[key]).toLowerCase().includes(keyword)
           )
  })
})

// 方法
const refreshData = async () => {
  try {
    loading.value = true
    const response = await labelApi.getTemplateList({
      page: 1,
      page_size: 100
    })
    labelTemplates.value = response.data.data || []
  } catch (error) {
    ElMessage.error('获取标签模板失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const fetchNodes = async () => {
  try {
    nodeDialogLoading.value = true
    const clusterName = clusterStore.currentClusterName
    if (!clusterName) {
      ElMessage.error('请先选择集群')
      return
    }

    const response = await nodeApi.getNodes({
      page: 1,
      size: 1000,
      cluster_name: clusterName
    })
    availableNodes.value = response.data.data || []
  } catch (error) {
    console.error('获取节点列表失败:', error)
    ElMessage.error('获取节点列表失败: ' + error.message)
  } finally {
    nodeDialogLoading.value = false
  }
}

const handleSearch = (keyword) => {
  searchKeyword.value = keyword
}

const handleSearchClear = () => {
  searchKeyword.value = ''
}

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

const applyTemplateToNodes = async (template) => {
  selectedTemplate.value = template
  await fetchNodes()
  applyDialogVisible.value = true
}

const editTemplate = (template) => {
  // 实现编辑模板逻辑
  ElMessage.info('编辑功能开发中...')
}

const copyTemplate = (template) => {
  // 实现复制模板逻辑
  ElMessage.info('复制功能开发中...')
}

const deleteTemplate = async (template) => {
  try {
    await ElMessageBox.confirm(
      `确认删除模板 "${template.name}" 吗？此操作不可恢复。`,
      '删除确认',
      {
        type: 'warning'
      }
    )
    
    await labelApi.deleteTemplate(template.id)
    ElMessage.success('删除成功')
    refreshData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + error.message)
    }
  }
}

const showAddDialog = () => {
  ElMessage.info('添加标签功能开发中...')
}

// 确认应用标签模板 - 集成批量操作进度
const confirmApplyTemplate = async () => {
  if (selectedNodes.value.length === 0) {
    ElMessage.warning('请选择至少一个节点')
    return
  }
  
  if (!selectedTemplate.value.id) {
    ElMessage.error('未选择有效的模板')
    return
  }
  
  try {
    applying.value = true
    
    const clusterName = clusterStore.currentClusterName
    if (!clusterName) {
      ElMessage.error('请先选择集群')
      return
    }
    
    // 开始批量操作进度跟踪
    const operationId = progressStore.startBatchOperation({
      type: 'batch_apply_labels',
      title: '批量应用标签',
      description: `正在为 ${selectedNodes.value.length} 个节点应用标签模板 "${selectedTemplate.value.name}"...`,
      items: selectedNodes.value.map(nodeName => ({ name: nodeName, id: nodeName }))
    })
    
    currentOperationId.value = operationId
    progressDialogVisible.value = true
    
    const applyData = {
      cluster_name: clusterName,
      node_names: selectedNodes.value,
      template_id: selectedTemplate.value.id,
      operation: 'add',
      labels: selectedTemplate.value.labels || {}
    }
    
    // 执行API调用
    try {
      const result = await labelApi.applyTemplate(applyData)
      
      // 标记所有节点为成功
      selectedNodes.value.forEach(nodeName => {
        progressStore.addSuccessResult(operationId, { name: nodeName }, result)
      })
      
      // 添加到历史记录
      historyStore.addOperation({
        type: 'labels.batch_apply',
        description: `应用标签模板 "${selectedTemplate.value.name}" 到 ${selectedNodes.value.length} 个节点`,
        undoAction: async () => {
          // 实现撤销逻辑 - 批量删除刚添加的标签
          const deleteData = {
            cluster_name: clusterName,
            nodes: selectedNodes.value,
            keys: Object.keys(selectedTemplate.value.labels || {})
          }
          await labelApi.batchDeleteLabels(deleteData)
        },
        redoAction: async () => {
          await labelApi.applyTemplate(applyData)
        }
      })
      
      progressStore.completeOperation(operationId, 'completed')
      ElMessage.success(`成功为 ${selectedNodes.value.length} 个节点应用标签模板`)
      applyDialogVisible.value = false
      
    } catch (error) {
      // 标记所有节点为失败
      selectedNodes.value.forEach(nodeName => {
        progressStore.addFailureResult(operationId, { name: nodeName }, error)
      })
      
      progressStore.completeOperation(operationId, 'failed')
      throw error
    }
    
  } catch (error) {
    console.error('应用标签模板失败:', error)
    ElMessage.error(`应用模板失败: ${error.message}`)
  } finally {
    applying.value = false
  }
}

// 批量进度处理
const handleProgressCancel = (operationId) => {
  progressStore.cancelOperation(operationId)
  progressDialogVisible.value = false
}

const handleProgressRetry = async ({ operationId, items }) => {
  const operation = progressStore.getOperationProgress(operationId)
  if (operation && selectedTemplate.value) {
    const nodeNames = items.map(item => item.name || item.id || item)
    selectedNodes.value = nodeNames
    await confirmApplyTemplate()
  }
}

// 撤销/重做处理
const handleUndo = () => {
  refreshData() // 撤销后刷新数据
}

const handleRedo = () => {
  refreshData() // 重做后刷新数据
}

// 组件挂载
onMounted(() => {
  refreshData()
})
</script>

<style scoped>
.label-manage-optimized {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
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

.stats-row {
  margin-bottom: 24px;
}

.stat-card {
  text-align: center;
}

.stat-content {
  padding: 20px;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
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
}

.label-card {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 20px;
  background: #fff;
  transition: all 0.2s;
  cursor: pointer;
}

.label-card:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.1);
  transform: translateY(-2px);
}

.label-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.label-key {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.label-value {
  font-size: 12px;
  color: #999;
}

.label-content {
  margin-bottom: 16px;
}

.label-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.label-tag {
  font-family: monospace;
  font-size: 12px;
}

.tag-key {
  color: #1890ff;
  font-weight: 500;
}

.tag-value {
  color: #52c41a;
  font-weight: 500;
}

.no-labels {
  color: #999;
  font-style: italic;
  text-align: center;
  padding: 20px 0;
}

.label-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}

.created-time {
  font-size: 12px;
  color: #999;
}

.apply-dialog-content {
  padding: 8px 0;
}

.template-summary {
  margin-bottom: 24px;
  padding: 16px;
  background: #f9f9f9;
  border-radius: 6px;
}

.template-summary h4 {
  margin: 0 0 12px 0;
  color: #333;
}

.template-details p {
  margin: 8px 0;
  color: #666;
}

.node-selector-section {
  margin-bottom: 24px;
}

.node-selector-section h4 {
  margin: 0 0 16px 0;
  color: #333;
}

.selected-summary {
  padding: 12px;
  background: #e6f7ff;
  border: 1px solid #91d5ff;
  border-radius: 6px;
  color: #0958d9;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
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
  
  .label-card {
    padding: 16px;
  }
  
  .label-header {
    flex-direction: column;
    gap: 8px;
  }
  
  .label-footer {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
}
</style>
