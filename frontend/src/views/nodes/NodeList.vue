<template>
  <div class="node-list">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">节点管理</h1>
        <p class="page-description">管理Kubernetes集群节点</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <SearchBox
        v-model="searchKeyword"
        placeholder="搜索节点名称..."
        :advanced-search="true"
        :filters="searchFilters"
        :realtime="true"
        :debounce="500"
        @search="handleSearch"
        @filter-change="handleFilterChange"
      />
    </el-card>

    <!-- 批量操作栏 -->
    <div v-if="selectedNodes.length > 0" class="batch-actions">
      <div class="batch-info">
        <span>已选择 {{ selectedNodes.length }} 个节点</span>
        <el-button type="text" @click="clearSelection">清空选择</el-button>
      </div>
      <div class="batch-buttons">
        <el-button @click="batchCordon" :loading="batchLoading.cordon">
          <el-icon><Lock /></el-icon>
          批量封锁
        </el-button>
        <el-button @click="batchUncordon" :loading="batchLoading.uncordon">
          <el-icon><Unlock /></el-icon>
          批量解封
        </el-button>
        <el-button type="danger" @click="batchDrain" :loading="batchLoading.drain">
          <el-icon><Download /></el-icon>
          批量驱逐
        </el-button>
      </div>
    </div>

    <!-- 节点表格 -->
    <el-card class="table-card">
      <el-table
        v-loading="loading"
        :data="filteredNodes"
        style="width: 100%"
        @selection-change="handleSelectionChange"
        @sort-change="handleSortChange"
      >
        <!-- 空状态 -->
        <template #empty>
          <div class="empty-content">
            <el-empty
              v-if="!clusterStore.hasCluster"
              description="暂无集群配置"
              :image-size="100"
            >
              <template #description>
                <p>您还没有配置任何Kubernetes集群</p>
                <p>请先添加集群配置以开始管理节点</p>
              </template>
              <el-button type="primary" @click="$router.push('/clusters')">
                <el-icon><Plus /></el-icon>
                添加集群
              </el-button>
            </el-empty>
            
            <el-empty
              v-else
              description="当前集群暂无节点数据"
              :image-size="80"
            >
              <template #description>
                <p>当前集群中没有找到节点</p>
                <p>请检查集群连接状态或稍后重试</p>
              </template>
              <el-button @click="refreshData">
                <el-icon><Refresh /></el-icon>
                刷新数据
              </el-button>
            </el-empty>
          </div>
        </template>
        <el-table-column type="selection" width="55" />
        
        <el-table-column
          prop="name"
          label="节点名称"
          sortable="custom"
          min-width="150"
        >
          <template #default="{ row }">
            <div class="node-name-cell">
              <el-button
                type="text"
                class="node-name-link"
                @click="viewNodeDetail(row)"
              >
                {{ row.name }}
              </el-button>
              <div class="node-labels">
                <el-tag
                  v-for="role in row.roles"
                  :key="role"
                  :type="role === 'master' ? 'danger' : 'primary'"
                  size="small"
                  class="role-tag"
                >
                  {{ formatNodeRoles([role]) }}
                </el-tag>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column
          prop="status"
          label="状态"
          sortable="custom"
          width="100"
        >
          <template #default="{ row }">
            <el-tag
              :type="formatNodeStatus(row.status).type"
              size="small"
            >
              {{ formatNodeStatus(row.status).text }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column
          prop="schedulable"
          label="可调度"
          width="80"
        >
          <template #default="{ row }">
            <el-tag
              :type="row.schedulable ? 'success' : 'info'"
              size="small"
            >
              {{ row.schedulable ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="资源使用" min-width="200">
          <template #default="{ row }">
            <div class="resource-usage">
              <div class="resource-item">
                <span class="resource-label">CPU:</span>
                <el-progress
                  :percentage="row.resources?.cpu?.usage || 0"
                  :stroke-width="4"
                  size="small"
                />
                <span class="resource-text">
                  {{ formatCPU(row.resources?.cpu?.used) }} / {{ formatCPU(row.resources?.cpu?.total) }}
                </span>
              </div>
              <div class="resource-item">
                <span class="resource-label">内存:</span>
                <el-progress
                  :percentage="row.resources?.memory?.usage || 0"
                  :stroke-width="4"
                  size="small"
                />
                <span class="resource-text">
                  {{ formatMemory(row.resources?.memory?.used) }} / {{ formatMemory(row.resources?.memory?.total) }}
                </span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column
          prop="version"
          label="版本"
          width="120"
        >
          <template #default="{ row }">
            <span class="version-text">{{ row.version || 'N/A' }}</span>
          </template>
        </el-table-column>

        <el-table-column
          prop="createdAt"
          label="创建时间"
          sortable="custom"
          width="180"
        >
          <template #default="{ row }">
            <span class="time-text">{{ formatTime(row.createdAt) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button type="text" size="small" @click="viewNodeDetail(row)">
                <el-icon><View /></el-icon>
                详情
              </el-button>
              
              <el-button
                v-if="row.schedulable"
                type="text"
                size="small"
                @click="cordonNode(row)"
              >
                <el-icon><Lock /></el-icon>
                封锁
              </el-button>
              
              <el-button
                v-else
                type="text"
                size="small"
                @click="uncordonNode(row)"
              >
                <el-icon><Unlock /></el-icon>
                解封
              </el-button>
              
              <el-dropdown @command="(cmd) => handleNodeAction(cmd, row)">
                <el-button type="text" size="small">
                  <el-icon><MoreFilled /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="drain">
                      <el-icon><Download /></el-icon>
                      驱逐Pod
                    </el-dropdown-item>
                    <el-dropdown-item command="labels">
                      <el-icon><CollectionTag /></el-icon>
                      管理标签
                    </el-dropdown-item>
                    <el-dropdown-item command="taints">
                      <el-icon><WarningFilled /></el-icon>
                      管理污点
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 节点详情对话框 -->
    <NodeDetailDialog
      v-model="detailDialogVisible"
      :node="selectedNode"
      @refresh="refreshData"
    />

    <!-- 驱逐确认对话框 -->
    <ConfirmDialog
      v-model="drainConfirmVisible"
      title="确认驱逐操作"
      :message="drainConfirmMessage"
      :details="drainDetails"
      dangerous
      confirm-text="确认驱逐"
      @confirm="confirmDrain"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useNodeStore } from '@/store/modules/node'
import { useClusterStore } from '@/store/modules/cluster'
import { formatTime, formatNodeStatus, formatNodeRoles, formatCPU, formatMemory } from '@/utils/format'
import SearchBox from '@/components/common/SearchBox.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import NodeDetailDialog from './components/NodeDetailDialog.vue'
import {
  Refresh,
  Lock,
  Unlock,
  Download,
  View,
  MoreFilled,
  CollectionTag,
  WarningFilled,
  Plus
} from '@element-plus/icons-vue'

const nodeStore = useNodeStore()
const clusterStore = useClusterStore()

// 响应式数据
const loading = ref(false)
const searchKeyword = ref('')
const selectedNode = ref(null)
const detailDialogVisible = ref(false)
const drainConfirmVisible = ref(false)
const drainConfirmMessage = ref('')
const drainDetails = ref([])

// 批量操作加载状态
const batchLoading = reactive({
  cordon: false,
  uncordon: false,
  drain: false
})

// 搜索筛选配置
const searchFilters = ref([
  {
    key: 'status',
    label: '状态',
    type: 'select',
    placeholder: '选择状态',
    options: [
      { label: '正常', value: 'Ready' },
      { label: '异常', value: 'NotReady' },
      { label: '未知', value: 'Unknown' }
    ]
  },
  {
    key: 'role',
    label: '角色',
    type: 'select',
    placeholder: '选择角色',
    options: [
      { label: '主节点', value: 'master' },
      { label: '工作节点', value: 'worker' }
    ]
  },
  {
    key: 'schedulable',
    label: '可调度',
    type: 'select',
    placeholder: '选择调度状态',
    options: [
      { label: '可调度', value: 'true' },
      { label: '不可调度', value: 'false' }
    ]
  }
])

// 计算属性
const nodes = computed(() => nodeStore.nodes)
const selectedNodes = computed(() => nodeStore.selectedNodes)
const pagination = computed(() => nodeStore.pagination)

const filteredNodes = computed(() => {
  return nodeStore.filteredNodes
})

// 处理搜索
const handleSearch = (params) => {
  nodeStore.setFilters({ name: params.keyword, ...params.filters })
  fetchNodes()
}

// 处理筛选变化
const handleFilterChange = (filters) => {
  nodeStore.setFilters(filters)
}

// 处理选择变化
const handleSelectionChange = (selection) => {
  nodeStore.setSelectedNodes(selection)
}

// 处理排序
const handleSortChange = ({ prop, order }) => {
  const sortBy = prop
  const sortOrder = order === 'ascending' ? 'asc' : 'desc'
  fetchNodes({ sortBy, sortOrder })
}

// 分页处理
const handleSizeChange = (size) => {
  nodeStore.setPagination({ size, current: 1 })
  fetchNodes()
}

const handleCurrentChange = (current) => {
  nodeStore.setPagination({ current })
  fetchNodes()
}

// 获取节点数据
const fetchNodes = async (params = {}) => {
  try {
    loading.value = true
    await nodeStore.fetchNodes(params)
  } catch (error) {
    ElMessage.error('获取节点数据失败')
  } finally {
    loading.value = false
  }
}

// 刷新数据
const refreshData = () => {
  fetchNodes()
}

// 查看节点详情
const viewNodeDetail = (node) => {
  selectedNode.value = node
  detailDialogVisible.value = true
}

// 封锁节点
const cordonNode = async (node) => {
  try {
    await nodeStore.cordonNode(node.name)
    ElMessage.success(`节点 ${node.name} 已封锁`)
    refreshData()
  } catch (error) {
    ElMessage.error(`封锁节点失败: ${error.message}`)
  }
}

// 解封节点
const uncordonNode = async (node) => {
  try {
    await nodeStore.uncordonNode(node.name)
    ElMessage.success(`节点 ${node.name} 已解封`)
    refreshData()
  } catch (error) {
    ElMessage.error(`解封节点失败: ${error.message}`)
  }
}

// 驱逐节点
const drainNode = (node) => {
  drainConfirmMessage.value = `确认要驱逐节点 "${node.name}" 上的所有Pod吗？`
  drainDetails.value = [
    '此操作将会:',
    '1. 将节点标记为不可调度',
    '2. 驱逐节点上的所有Pod',
    '3. 等待Pod优雅终止',
    '请确保已经做好相应的准备工作'
  ]
  selectedNode.value = node
  drainConfirmVisible.value = true
}

// 确认驱逐
const confirmDrain = async () => {
  try {
    await nodeStore.drainNode(selectedNode.value.name)
    ElMessage.success(`节点 ${selectedNode.value.name} 驱逐操作已开始`)
    refreshData()
  } catch (error) {
    ElMessage.error(`驱逐节点失败: ${error.message}`)
  }
}

// 处理节点操作
const handleNodeAction = (command, node) => {
  switch (command) {
    case 'drain':
      drainNode(node)
      break
    case 'labels':
      // 跳转到标签管理页面，传递节点信息
      break
    case 'taints':
      // 跳转到污点管理页面，传递节点信息
      break
  }
}

// 批量操作
const batchCordon = async () => {
  if (selectedNodes.value.length === 0) return
  
  try {
    batchLoading.cordon = true
    const nodeNames = selectedNodes.value.map(node => node.name)
    await nodeStore.batchCordon(nodeNames)
    ElMessage.success(`成功封锁 ${nodeNames.length} 个节点`)
    clearSelection()
    refreshData()
  } catch (error) {
    ElMessage.error(`批量封锁失败: ${error.message}`)
  } finally {
    batchLoading.cordon = false
  }
}

const batchUncordon = async () => {
  if (selectedNodes.value.length === 0) return
  
  try {
    batchLoading.uncordon = true
    const nodeNames = selectedNodes.value.map(node => node.name)
    await nodeStore.batchUncordon(nodeNames)
    ElMessage.success(`成功解封 ${nodeNames.length} 个节点`)
    clearSelection()
    refreshData()
  } catch (error) {
    ElMessage.error(`批量解封失败: ${error.message}`)
  } finally {
    batchLoading.uncordon = false
  }
}

const batchDrain = async () => {
  if (selectedNodes.value.length === 0) return
  
  ElMessageBox.confirm(
    `确认要驱逐选中的 ${selectedNodes.value.length} 个节点上的所有Pod吗？`,
    '批量驱逐确认',
    {
      confirmButtonText: '确认驱逐',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'el-button--danger'
    }
  ).then(async () => {
    try {
      batchLoading.drain = true
      const nodeNames = selectedNodes.value.map(node => node.name)
      await nodeStore.batchDrain(nodeNames)
      ElMessage.success(`成功开始驱逐 ${nodeNames.length} 个节点`)
      clearSelection()
      refreshData()
    } catch (error) {
      ElMessage.error(`批量驱逐失败: ${error.message}`)
    } finally {
      batchLoading.drain = false
    }
  }).catch(() => {
    // 用户取消
  })
}

// 清空选择
const clearSelection = () => {
  nodeStore.clearSelectedNodes()
}

onMounted(() => {
  fetchNodes()
})
</script>

<style scoped>
.node-list {
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

.search-card {
  margin-bottom: 16px;
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

.node-name-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.node-name-link {
  font-weight: 500;
  padding: 0;
  height: auto;
}

.node-labels {
  display: flex;
  gap: 4px;
}

.role-tag {
  font-size: 11px;
  height: 18px;
  line-height: 16px;
}

.resource-usage {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.resource-label {
  width: 40px;
  color: #666;
}

.resource-item :deep(.el-progress) {
  flex: 1;
  max-width: 60px;
}

.resource-text {
  color: #666;
  white-space: nowrap;
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
  
  .table-card :deep(.el-table) {
    font-size: 12px;
  }
  
  .action-buttons {
    flex-direction: column;
    gap: 2px;
  }
}
</style>