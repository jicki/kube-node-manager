<template>
  <div class="node-list-optimized">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">节点管理 (优化版)</h1>
        <p class="page-description">管理Kubernetes集群节点 - 支持虚拟滚动、批量操作进度、撤销/重做</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="refreshData" :loading="nodeStore.isLoading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <div class="search-section">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索节点名称..."
          clearable
          @input="$debounce(handleSearch, 300)"
          @clear="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      
      <div class="filter-section">
        <el-row :gutter="12">
          <el-col :span="4">
            <el-select
              v-model="statusFilter"
              placeholder="状态筛选"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="全部状态" value="" />
              <el-option label="Ready" value="Ready" />
              <el-option label="NotReady" value="NotReady" />
              <el-option label="Unknown" value="Unknown" />
            </el-select>
          </el-col>
          <el-col :span="4">
            <el-select
              v-model="roleFilter"
              placeholder="角色筛选"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="全部角色" value="" />
              <el-option label="Master" value="master" />
              <el-option label="Worker" value="worker" />
            </el-select>
          </el-col>
          <el-col :span="5">
            <el-select
              v-model="schedulableFilter"
              placeholder="调度状态"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="全部状态" value="" />
              <el-option label="可调度" value="schedulable" />
              <el-option label="有限调度" value="limited" />
              <el-option label="不可调度" value="unschedulable" />
            </el-select>
          </el-col>
        </el-row>
      </div>
    </el-card>

    <!-- 批量操作栏 -->
    <div v-if="nodeStore.selectedNodes.length > 0" class="batch-actions">
      <div class="batch-info">
        <span>已选择 {{ nodeStore.selectedNodes.length }} 个节点</span>
        <el-button type="text" @click="clearSelection">清空选择</el-button>
      </div>
      <div class="batch-buttons">
        <el-button @click="showBatchCordonDialog" :loading="nodeStore.isLoading">
          <el-icon><Lock /></el-icon>
          批量禁止调度
        </el-button>
        <el-button @click="batchUncordon" :loading="nodeStore.isLoading">
          <el-icon><Unlock /></el-icon>
          批量解除调度
        </el-button>
        <el-button 
          v-if="authStore.role === 'admin'"
          type="danger"
          @click="showBatchDrainDialog"
          :loading="nodeStore.isLoading"
        >
          <el-icon><VideoPlay /></el-icon>
          批量驱逐
        </el-button>
      </div>
    </div>

    <!-- 虚拟表格 -->
    <el-card class="table-card">
      <div class="table-header">
        <div class="table-info">
          <span>共 {{ nodeStore.pagination.total }} 个节点</span>
          <span v-if="nodeStore.hasFilters">
            (已过滤，显示 {{ nodeStore.nodes.length }} 个)
          </span>
        </div>
        <div class="table-controls">
          <el-tooltip content="启用虚拟滚动提升性能">
            <el-switch
              v-model="useVirtualScroll"
              active-text="虚拟滚动"
              @change="handleVirtualScrollToggle"
            />
          </el-tooltip>
        </div>
      </div>

      <!-- 虚拟滚动表格 -->
      <div v-if="useVirtualScroll" class="virtual-table-container">
        <VirtualList
          ref="virtualListRef"
          :items="nodeStore.nodes"
          :item-height="80"
          :container-height="600"
          @reach-bottom="handleLoadMore"
        >
          <template #default="{ item: node }">
            <div class="virtual-node-item">
              <div class="node-checkbox">
                <el-checkbox
                  :model-value="isNodeSelected(node)"
                  @change="handleNodeSelect(node, $event)"
                />
              </div>
              
              <div class="node-info">
                <div class="node-name">
                  <el-button type="text" @click="viewNodeDetail(node)">
                    {{ node.name }}
                  </el-button>
                </div>
                
                <div class="node-tags">
                  <el-tag
                    v-for="role in node.roles"
                    :key="role"
                    :type="getNodeRoleType(role)"
                    size="small"
                  >
                    {{ role }}
                  </el-tag>
                  
                  <el-tag
                    v-if="node.taints && node.taints.length > 0"
                    type="warning"
                    size="small"
                  >
                    {{ node.taints.length }} 个污点
                  </el-tag>
                </div>
              </div>
              
              <div class="node-status">
                <el-tag
                  :type="getNodeStatusType(node.status)"
                  size="small"
                >
                  {{ node.status }}
                </el-tag>
                
                <el-tag
                  :type="getSchedulingStatusType(node)"
                  size="small"
                  style="margin-left: 8px;"
                >
                  {{ getSchedulingStatusText(node) }}
                </el-tag>
              </div>
              
              <div class="node-resources">
                <div class="resource-item">
                  <span class="resource-label">CPU:</span>
                  <span class="resource-value">{{ node.capacity?.cpu || 'N/A' }}</span>
                </div>
                <div class="resource-item">
                  <span class="resource-label">内存:</span>
                  <span class="resource-value">{{ formatMemory(node.capacity?.memory) }}</span>
                </div>
              </div>
              
              <div class="node-actions">
                <el-button
                  v-if="node.schedulable"
                  type="text"
                  size="small"
                  @click="cordonNode(node)"
                >
                  禁止调度
                </el-button>
                <el-button
                  v-else
                  type="text"
                  size="small"
                  @click="uncordonNode(node)"
                >
                  解除调度
                </el-button>
                
                <el-dropdown @command="(cmd) => handleNodeAction(cmd, node)">
                  <el-button type="text" size="small">
                    更多<el-icon class="el-icon--right"><ArrowDown /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="detail">查看详情</el-dropdown-item>
                      <el-dropdown-item command="labels">管理标签</el-dropdown-item>
                      <el-dropdown-item command="taints">管理污点</el-dropdown-item>
                      <el-dropdown-item v-if="authStore.role === 'admin'" command="drain" divided>
                        驱逐节点
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </div>
          </template>
        </VirtualList>
      </div>

      <!-- 传统表格 -->
      <el-table
        v-else
        v-loading="nodeStore.loading.fetching"
        :data="nodeStore.nodes"
        style="width: 100%"
        @selection-change="handleSelectionChange"
        @sort-change="handleSortChange"
        :default-sort="{ prop: nodeStore.sort.prop, order: nodeStore.sort.order }"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="name" label="节点名称" sortable="custom" min-width="200" />
        <el-table-column prop="status" label="状态" sortable="custom" width="100" />
        <el-table-column label="调度状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getSchedulingStatusType(row)" size="small">
              {{ getSchedulingStatusText(row) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="资源" min-width="200">
          <template #default="{ row }">
            <div class="resource-summary">
              <span>CPU: {{ row.capacity?.cpu || 'N/A' }}</span>
              <span>内存: {{ formatMemory(row.capacity?.memory) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="text" size="small" @click="viewNodeDetail(row)">详情</el-button>
            <el-button
              v-if="row.schedulable"
              type="text"
              size="small"
              @click="cordonNode(row)"
            >
              禁止调度
            </el-button>
            <el-button
              v-else
              type="text"
              size="small"
              @click="uncordonNode(row)"
            >
              解除调度
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="nodeStore.pagination.current"
          v-model:page-size="nodeStore.pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="nodeStore.pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 批量操作进度弹窗 -->
    <BatchProgressDialog
      v-model="progressDialogVisible"
      :operation-id="currentOperationId"
      @cancel="handleProgressCancel"
      @retry="handleProgressRetry"
    />

    <!-- 增强确认弹窗 -->
    <EnhancedConfirmDialog
      v-model="confirmDialogVisible"
      :config="confirmConfig"
      @confirm="handleConfirmAction"
      @cancel="handleConfirmCancel"
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
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useNodeStore } from '@/store/modules/node-optimized'
import { useClusterStore } from '@/store/modules/cluster'
import { useAuthStore } from '@/store/modules/auth'
import { useProgressStore } from '@/store/modules/progress'
import { useHistoryStore } from '@/store/modules/history'
import VirtualList from '@/components/common/VirtualList.vue'
import BatchProgressDialog from '@/components/common/BatchProgressDialog.vue'
import EnhancedConfirmDialog from '@/components/common/EnhancedConfirmDialog.vue'
import UndoRedoBar from '@/components/common/UndoRedoBar.vue'
import { formatMemory } from '@/utils/format'
import {
  Refresh,
  Search,
  Lock,
  Unlock,
  VideoPlay,
  ArrowDown
} from '@element-plus/icons-vue'

const router = useRouter()
const nodeStore = useNodeStore()
const clusterStore = useClusterStore()
const authStore = useAuthStore()
const progressStore = useProgressStore()
const historyStore = useHistoryStore()

// 响应式数据
const searchKeyword = ref('')
const statusFilter = ref('')
const roleFilter = ref('')
const schedulableFilter = ref('')
const useVirtualScroll = ref(true)
const virtualListRef = ref(null)

// 弹窗状态
const progressDialogVisible = ref(false)
const confirmDialogVisible = ref(false)
const currentOperationId = ref('')
const confirmConfig = ref({})
const pendingAction = ref(null)

// 监听搜索关键词
watch(searchKeyword, () => {
  nodeStore.setFilters({ name: searchKeyword.value })
})

// 计算属性
const selectedNodes = computed(() => nodeStore.selectedNodes)

// 方法
const refreshData = async () => {
  try {
    await nodeStore.refreshNodes()
    ElMessage.success('数据刷新成功')
  } catch (error) {
    ElMessage.error('数据刷新失败: ' + error.message)
  }
}

const handleSearch = () => {
  nodeStore.setFilters({ name: searchKeyword.value })
  fetchNodes()
}

const handleFilterChange = () => {
  nodeStore.setFilters({
    status: statusFilter.value,
    role: roleFilter.value,
    schedulable: schedulableFilter.value
  })
  fetchNodes()
}

const handleSelectionChange = (selection) => {
  nodeStore.setSelectedNodes(selection)
}

const handleSortChange = ({ prop, order }) => {
  nodeStore.setSort({ prop, order })
  fetchNodes()
}

const handleSizeChange = (size) => {
  nodeStore.setPagination({ size, current: 1 })
  fetchNodes()
}

const handleCurrentChange = (current) => {
  nodeStore.setPagination({ current })
  fetchNodes()
}

const handleLoadMore = async () => {
  if (nodeStore.pagination.current < nodeStore.pagination.totalPages) {
    const nextPage = nodeStore.pagination.current + 1
    nodeStore.setPagination({ current: nextPage })
    await fetchNodes()
  }
}

const fetchNodes = async () => {
  try {
    await nodeStore.fetchNodes()
  } catch (error) {
    ElMessage.error('获取节点数据失败: ' + error.message)
  }
}

const handleVirtualScrollToggle = (enabled) => {
  if (enabled && nodeStore.nodes.length > 100) {
    ElMessage.info('已启用虚拟滚动，提升大数据量性能')
  }
}

// 节点操作
const cordonNode = (node) => {
  showCordonConfirmDialog([node])
}

const uncordonNode = async (node) => {
  try {
    await nodeStore.uncordonNode(node.name)
    ElMessage.success(`节点 ${node.name} 已解除调度限制`)
  } catch (error) {
    ElMessage.error(`解除调度失败: ${error.message}`)
  }
}

const viewNodeDetail = (node) => {
  // 跳转到节点详情页面
  router.push(`/nodes/${node.name}`)
}

const handleNodeAction = (command, node) => {
  switch (command) {
    case 'detail':
      viewNodeDetail(node)
      break
    case 'labels':
      router.push(`/labels?node=${node.name}`)
      break
    case 'taints':
      router.push(`/taints?node=${node.name}`)
      break
    case 'drain':
      showDrainConfirmDialog([node])
      break
  }
}

// 选择相关
const isNodeSelected = (node) => {
  return nodeStore.selectedNodes.some(n => n.name === node.name)
}

const handleNodeSelect = (node, checked) => {
  const currentSelection = [...nodeStore.selectedNodes]
  if (checked) {
    if (!currentSelection.find(n => n.name === node.name)) {
      currentSelection.push(node)
    }
  } else {
    const index = currentSelection.findIndex(n => n.name === node.name)
    if (index !== -1) {
      currentSelection.splice(index, 1)
    }
  }
  nodeStore.setSelectedNodes(currentSelection)
}

const clearSelection = () => {
  nodeStore.clearSelectedNodes()
}

// 批量操作
const showBatchCordonDialog = () => {
  showCordonConfirmDialog(nodeStore.selectedNodes)
}

const batchUncordon = async () => {
  const nodeNames = nodeStore.selectedNodes.map(n => n.name)
  await performBatchOperation('uncordon', nodeNames)
}

const showBatchDrainDialog = () => {
  showDrainConfirmDialog(nodeStore.selectedNodes)
}

// 确认弹窗
const showCordonConfirmDialog = (nodes) => {
  const isMultiple = nodes.length > 1
  
  confirmConfig.value = {
    type: 'warning',
    title: isMultiple ? '批量禁止调度' : '禁止调度',
    message: `确认要禁止调度${isMultiple ? `以下 ${nodes.length} 个节点` : `节点 "${nodes[0].name}"`}吗？`,
    input: {
      reason: {
        label: '禁止调度原因',
        type: 'textarea',
        placeholder: '请输入禁止调度的原因（可选）',
        maxlength: 200,
        showWordLimit: true
      }
    },
    risk: {
      title: '操作风险',
      message: '禁止调度后，新的Pod将不会被调度到这些节点上',
      level: 'warning'
    }
  }
  
  pendingAction.value = {
    type: 'cordon',
    nodes
  }
  
  confirmDialogVisible.value = true
}

const showDrainConfirmDialog = (nodes) => {
  const isMultiple = nodes.length > 1
  
  confirmConfig.value = {
    type: 'error',
    title: isMultiple ? '批量驱逐节点' : '驱逐节点',
    message: `确认要驱逐${isMultiple ? `以下 ${nodes.length} 个节点` : `节点 "${nodes[0].name}"`}上的Pod吗？`,
    input: {
      reason: {
        label: '驱逐原因',
        type: 'textarea',
        placeholder: '请输入驱逐原因（必填）',
        required: true,
        maxlength: 200,
        showWordLimit: true
      }
    },
    risk: {
      title: '高风险操作',
      message: '驱逐操作将删除节点上的所有Pod（除DaemonSet），并可能影响服务可用性',
      level: 'error'
    },
    impact: {
      title: '影响范围',
      items: [
        '节点将被标记为不可调度',
        '节点上的Pod将被强制删除',
        'EmptyDir数据将丢失',
        '可能影响服务的高可用性'
      ]
    },
    checklist: [
      '我已了解此操作的风险',
      '我确认要执行此操作'
    ],
    doubleConfirm: {
      text: 'DRAIN'
    }
  }
  
  pendingAction.value = {
    type: 'drain',
    nodes
  }
  
  confirmDialogVisible.value = true
}

const handleConfirmAction = async (result) => {
  if (!pendingAction.value) return
  
  const { type, nodes } = pendingAction.value
  const nodeNames = nodes.map(n => n.name)
  const reason = result.input?.reason || ''
  
  await performBatchOperation(type, nodeNames, { reason })
  
  pendingAction.value = null
}

const handleConfirmCancel = () => {
  pendingAction.value = null
}

// 执行批量操作
const performBatchOperation = async (operation, nodeNames, params = {}) => {
  try {
    const result = await nodeStore.batchOperateNodes(operation, nodeNames, params)
    
    // 显示进度弹窗
    const operationId = progressStore.getActiveOperations()[0]?.id
    if (operationId) {
      currentOperationId.value = operationId
      progressDialogVisible.value = true
    }
    
    // 清除选择
    nodeStore.clearSelectedNodes()
    
    // 显示结果
    if (result.success > 0) {
      ElMessage.success(`成功处理 ${result.success} 个节点`)
    }
    if (result.failed > 0) {
      ElMessage.warning(`${result.failed} 个节点处理失败`)
    }
  } catch (error) {
    ElMessage.error(`批量操作失败: ${error.message}`)
  }
}

// 进度弹窗处理
const handleProgressCancel = (operationId) => {
  progressStore.cancelOperation(operationId)
  progressDialogVisible.value = false
}

const handleProgressRetry = async ({ operationId, items }) => {
  const operation = progressStore.getOperationProgress(operationId)
  if (operation && pendingAction.value) {
    const nodeNames = items.map(item => item.name || item.id || item)
    await performBatchOperation(pendingAction.value.type, nodeNames)
  }
}

// 撤销/重做处理
const handleUndo = () => {
  refreshData() // 撤销后刷新数据
}

const handleRedo = () => {
  refreshData() // 重做后刷新数据
}

// 工具方法
const getNodeRoleType = (role) => {
  if (role === 'master' || role.includes('master') || role.includes('control-plane')) {
    return 'danger'
  }
  return 'primary'
}

const getNodeStatusType = (status) => {
  const types = {
    'Ready': 'success',
    'NotReady': 'danger',
    'Unknown': 'warning'
  }
  return types[status] || 'info'
}

const getSchedulingStatusType = (node) => {
  if (!node.schedulable) return 'danger'
  if (node.taints?.some(t => t.effect === 'NoSchedule' || t.effect === 'PreferNoSchedule')) {
    return 'warning'
  }
  return 'success'
}

const getSchedulingStatusText = (node) => {
  if (!node.schedulable) return '不可调度'
  if (node.taints?.some(t => t.effect === 'NoSchedule' || t.effect === 'PreferNoSchedule')) {
    return '有限调度'
  }
  return '可调度'
}

// 组件挂载
onMounted(async () => {
  await fetchNodes()
})
</script>

<style scoped>
.node-list-optimized {
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

.search-card {
  margin-bottom: 16px;
}

.search-section {
  margin-bottom: 12px;
}

.batch-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #f0f9ff;
  border: 1px solid #bae7ff;
  border-radius: 6px;
  padding: 12px 16px;
  margin-bottom: 16px;
}

.batch-info {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 14px;
  color: #0958d9;
}

.batch-buttons {
  display: flex;
  gap: 8px;
}

.table-card :deep(.el-card__body) {
  padding: 0;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

.table-info {
  font-size: 14px;
  color: #666;
}

.table-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.virtual-table-container {
  padding: 16px 0;
}

.virtual-node-item {
  display: flex;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.2s;
}

.virtual-node-item:hover {
  background: #fafafa;
}

.node-checkbox {
  margin-right: 16px;
}

.node-info {
  flex: 1;
  min-width: 200px;
}

.node-name {
  margin-bottom: 8px;
}

.node-name .el-button {
  font-size: 15px;
  font-weight: 600;
  padding: 0;
  height: auto;
}

.node-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.node-status {
  width: 120px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
}

.node-resources {
  width: 150px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-item {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}

.resource-label {
  color: #666;
}

.resource-value {
  color: #333;
  font-weight: 500;
  font-family: monospace;
}

.resource-summary {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}

.node-actions {
  width: 120px;
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.pagination-container {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }
  
  .batch-actions {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
  
  .batch-info {
    justify-content: center;
  }
  
  .batch-buttons {
    justify-content: center;
  }
  
  .virtual-node-item {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
    padding: 12px 16px;
  }
  
  .node-status,
  .node-resources,
  .node-actions {
    width: auto;
  }
  
  .node-actions {
    justify-content: center;
  }
}
</style>
