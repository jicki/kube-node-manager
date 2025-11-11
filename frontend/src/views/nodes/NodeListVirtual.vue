<template>
  <div class="node-list-virtual">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">节点管理（虚拟滚动版）</h1>
        <p class="page-description">使用虚拟滚动优化大数据量渲染</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索筛选 -->
    <el-card class="search-card">
      <div class="search-section">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索节点名称..."
          clearable
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      
      <div class="filter-section">
        <el-row :gutter="12">
          <el-col :span="6">
            <el-select
              v-model="statusFilter"
              placeholder="状态筛选"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="全部状态" value="" />
              <el-option label="Ready" value="Ready" />
              <el-option label="NotReady" value="NotReady" />
            </el-select>
          </el-col>
          <el-col :span="6">
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
        </el-row>
      </div>
    </el-card>

    <!-- 批量操作 -->
    <div class="batch-actions" v-if="selectedNodes.length > 0">
      <div class="selected-info">
        已选择 <strong>{{ selectedNodes.length }}</strong> 个节点
      </div>
      <div class="action-buttons">
        <el-button type="primary" @click="batchCordon">
          批量Cordon
        </el-button>
        <el-button type="success" @click="batchUncordon">
          批量Uncordon
        </el-button>
      </div>
    </div>

    <!-- 虚拟滚动表格 -->
    <el-card class="table-card">
      <VirtualTable
        :data="filteredNodes"
        :columns="tableColumns"
        :height="600"
        :row-height="80"
        :loading="loading"
        :search-keyword="searchKeyword"
        empty-text="暂无节点数据"
        @row-click="handleRowClick"
      >
        <!-- 节点名称列 -->
        <template #cell-name="{ row }">
          <div class="node-name-cell">
            <div class="node-name">{{ row.name }}</div>
            
            <!-- 角色标签 -->
            <div class="node-roles" v-if="row.roles && row.roles.length > 0">
              <el-tag
                v-for="role in row.roles"
                :key="role"
                :type="getRoleType(role)"
                size="small"
              >
                {{ role }}
              </el-tag>
            </div>
            
            <!-- 标签 -->
            <div class="node-labels" v-if="hasImportantLabels(row)">
              <el-tag
                v-for="[key, value] in getImportantLabels(row)"
                :key="key"
                size="small"
                type="info"
              >
                {{ key }}: {{ value }}
              </el-tag>
            </div>
            
            <!-- IP地址 -->
            <div class="node-ip" v-if="row.internal_ip || row.external_ip">
              <span class="ip-label">IP:</span>
              <span class="ip-text" v-if="row.internal_ip">{{ row.internal_ip }}</span>
              <span class="ip-text" v-if="row.external_ip">({{ row.external_ip }})</span>
            </div>
          </div>
        </template>
        
        <!-- 状态列 -->
        <template #cell-status="{ row }">
          <el-tag
            :type="row.status === 'Ready' ? 'success' : 'danger'"
            size="small"
          >
            {{ row.status }}
          </el-tag>
        </template>
        
        <!-- 调度状态列 -->
        <template #cell-schedulable="{ row }">
          <el-tag
            :type="row.unschedulable ? 'warning' : 'success'"
            size="small"
          >
            {{ row.unschedulable ? '不可调度' : '可调度' }}
          </el-tag>
        </template>
        
        <!-- 资源列 -->
        <template #cell-resources="{ row }">
          <div class="resource-info">
            <div class="resource-item">
              CPU: {{ row.capacity?.cpu || 'N/A' }}
            </div>
            <div class="resource-item">
              Memory: {{ formatMemory(row.capacity?.memory) }}
            </div>
          </div>
        </template>
        
        <!-- 操作列 -->
        <template #cell-actions="{ row }">
          <el-button-group>
            <el-button
              size="small"
              @click.stop="viewDetail(row)"
            >
              详情
            </el-button>
            <el-button
              size="small"
              :type="row.unschedulable ? 'success' : 'warning'"
              @click.stop="toggleCordon(row)"
            >
              {{ row.unschedulable ? 'Uncordon' : 'Cordon' }}
            </el-button>
          </el-button-group>
        </template>
      </VirtualTable>
    </el-card>

    <!-- 节点详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="节点详情"
      width="800px"
    >
      <div v-if="selectedNode">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="节点名称">
            {{ selectedNode.name }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            {{ selectedNode.status }}
          </el-descriptions-item>
          <el-descriptions-item label="版本">
            {{ selectedNode.version }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ formatTime(selectedNode.created_at) }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useNodeStore } from '@/store/modules/node'
import { useClusterStore } from '@/store/modules/cluster'
import VirtualTable from '@/components/common/VirtualTable.vue'
import { debounce } from '@/utils/debounce'
import { ElMessage } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'

// Store
const nodeStore = useNodeStore()
const clusterStore = useClusterStore()

// State
const loading = ref(false)
const searchKeyword = ref('')
const statusFilter = ref('')
const roleFilter = ref('')
const selectedNodes = ref([])
const detailDialogVisible = ref(false)
const selectedNode = ref(null)

// 表格列配置
const tableColumns = [
  {
    key: 'name',
    title: '节点名称',
    width: 300,
    fixed: 'left'
  },
  {
    key: 'status',
    title: '状态',
    width: 120
  },
  {
    key: 'schedulable',
    title: '调度状态',
    width: 120
  },
  {
    key: 'resources',
    title: '资源',
    width: 200
  },
  {
    key: 'version',
    title: '版本',
    width: 150
  },
  {
    key: 'created_at',
    title: '创建时间',
    width: 180
  },
  {
    key: 'actions',
    title: '操作',
    width: 180,
    fixed: 'right'
  }
]

// 过滤后的节点列表
const filteredNodes = computed(() => {
  let nodes = nodeStore.nodes || []
  
  // 搜索过滤
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    nodes = nodes.filter(node => 
      node.name.toLowerCase().includes(keyword)
    )
  }
  
  // 状态过滤
  if (statusFilter.value) {
    nodes = nodes.filter(node => node.status === statusFilter.value)
  }
  
  // 角色过滤
  if (roleFilter.value) {
    nodes = nodes.filter(node => 
      node.roles && node.roles.includes(roleFilter.value)
    )
  }
  
  return nodes
})

// 搜索去抖动
const debouncedSearch = debounce(() => {
  // 触发过滤
  console.log('Searching:', searchKeyword.value)
}, 300)

const handleSearch = () => {
  debouncedSearch()
}

const handleFilterChange = () => {
  // 触发过滤
  console.log('Filter changed')
}

// 刷新数据
const refreshData = async () => {
  loading.value = true
  try {
    await nodeStore.fetchNodes(clusterStore.currentCluster?.name)
    ElMessage.success('刷新成功')
  } catch (error) {
    ElMessage.error('刷新失败：' + error.message)
  } finally {
    loading.value = false
  }
}

// 查看详情
const viewDetail = (node) => {
  selectedNode.value = node
  detailDialogVisible.value = true
}

// 行点击
const handleRowClick = (row) => {
  console.log('Row clicked:', row)
}

// Toggle Cordon
const toggleCordon = async (node) => {
  try {
    if (node.unschedulable) {
      await nodeStore.uncordonNode(clusterStore.currentCluster?.name, node.name)
      ElMessage.success(`节点 ${node.name} 已解除禁止调度`)
    } else {
      await nodeStore.cordonNode(clusterStore.currentCluster?.name, node.name)
      ElMessage.success(`节点 ${node.name} 已禁止调度`)
    }
    await refreshData()
  } catch (error) {
    ElMessage.error('操作失败：' + error.message)
  }
}

// 批量操作
const batchCordon = async () => {
  // 实现批量Cordon
  ElMessage.info('批量Cordon功能开发中')
}

const batchUncordon = async () => {
  // 实现批量Uncordon
  ElMessage.info('批量Uncordon功能开发中')
}

// 工具函数
const getRoleType = (role) => {
  return role === 'master' ? 'danger' : 'primary'
}

const hasImportantLabels = (node) => {
  const importantKeys = ['node-ownership', 'node.kubernetes.io/instance-type']
  return node.labels && Object.keys(node.labels).some(key => 
    importantKeys.includes(key)
  )
}

const getImportantLabels = (node) => {
  const importantKeys = ['node-ownership', 'node.kubernetes.io/instance-type']
  return Object.entries(node.labels || {}).filter(([key]) => 
    importantKeys.includes(key)
  )
}

const formatMemory = (memory) => {
  if (!memory) return 'N/A'
  const match = memory.match(/^(\d+)Ki$/)
  if (match) {
    const gb = (parseInt(match[1]) / (1024 * 1024)).toFixed(1)
    return `${gb}Gi`
  }
  return memory
}

const formatTime = (time) => {
  if (!time) return 'N/A'
  return new Date(time).toLocaleString('zh-CN')
}

// 初始化
onMounted(async () => {
  if (clusterStore.currentCluster) {
    await refreshData()
  }
})
</script>

<style scoped lang="scss">
.node-list-virtual {
  padding: 20px;
  
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    
    .page-title {
      margin: 0;
      font-size: 24px;
      color: var(--el-text-color-primary);
    }
    
    .page-description {
      margin: 4px 0 0 0;
      font-size: 14px;
      color: var(--el-text-color-secondary);
    }
  }
  
  .search-card,
  .table-card {
    margin-bottom: 20px;
  }
  
  .search-section {
    margin-bottom: 16px;
  }
  
  .filter-section {
    .el-select {
      width: 100%;
    }
  }
  
  .batch-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    margin-bottom: 16px;
    background: var(--el-color-primary-light-9);
    border-radius: 4px;
    
    .selected-info {
      color: var(--el-color-primary);
      
      strong {
        font-size: 18px;
      }
    }
  }
  
  .node-name-cell {
    .node-name {
      font-weight: 500;
      margin-bottom: 8px;
    }
    
    .node-roles,
    .node-labels {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      margin-top: 4px;
    }
    
    .node-ip {
      display: flex;
      align-items: center;
      gap: 6px;
      margin-top: 4px;
      font-size: 12px;
      
      .ip-label {
        color: var(--el-text-color-secondary);
        font-weight: 600;
        font-size: 11px;
      }
      
      .ip-text {
        font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
        color: var(--el-color-primary);
        font-weight: 500;
        margin-right: 4px;
      }
    }
  }
  
  .resource-info {
    .resource-item {
      font-size: 12px;
      line-height: 1.6;
    }
  }
}
</style>

