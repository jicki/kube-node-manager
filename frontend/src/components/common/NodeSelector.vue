<template>
  <div class="node-selector">
    <!-- 节点搜索和筛选 -->
    <div class="selector-header">
      <div class="search-section">
        <el-input
          v-model="searchQuery"
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
          <el-col :span="8">
            <el-select
              v-model="statusFilter"
              placeholder="状态筛选"
              clearable
              @change="handleFilter"
            >
              <el-option label="全部状态" value="" />
              <el-option label="Ready" value="Ready" />
              <el-option label="NotReady" value="NotReady" />
            </el-select>
          </el-col>
          <el-col :span="8">
            <el-select
              v-model="roleFilter"
              placeholder="角色筛选"
              clearable
              @change="handleFilter"
            >
              <el-option label="全部角色" value="" />
              <el-option label="Master" value="master" />
              <el-option label="Worker" value="worker" />
            </el-select>
          </el-col>
          <el-col :span="8">
            <el-input
              v-model="labelFilter"
              placeholder="标签筛选 (key=value)"
              clearable
              @input="handleFilter"
            />
          </el-col>
        </el-row>
      </div>

      <div class="action-section">
        <el-checkbox
          v-model="selectAll"
          :indeterminate="indeterminate"
          @change="handleSelectAll"
        >
          全选 ({{ selectedNodes.length }}/{{ filteredNodes.length }})
        </el-checkbox>
        <el-button type="text" size="small" @click="clearSelection">
          清空选择
        </el-button>
      </div>
    </div>

    <!-- 节点列表 -->
    <div class="node-list">
      <el-scrollbar height="300px">
        <el-checkbox-group v-model="selectedNodes" @change="handleSelectionChange">
          <div
            v-for="node in filteredNodes"
            :key="node?.name || node?.id || Math.random()"
            class="node-item"
            :class="{ 'selected': node?.name && selectedNodes.includes(node.name) }"
          >
            <el-checkbox 
              v-if="node?.name" 
              :value="node.name"
              :disabled="!node.name"
            >
              <div class="node-content">
                <div class="node-info">
                  <div class="node-name">{{ node.name || '未知节点' }}</div>
                  <div class="node-meta">
                    <el-tag 
                      :type="getStatusType(node.status)" 
                      size="small"
                    >
                      {{ node.status || 'Unknown' }}
                    </el-tag>
                    <span v-if="node.roles?.length" class="node-roles">
                      {{ node.roles.join(', ') }}
                    </span>
                  </div>
                </div>
                <div class="node-details">
                  <div v-if="node.internal_ip" class="node-ip">
                    IP: {{ node.internal_ip }}
                  </div>
                  <div v-if="node.labels && showLabels" class="node-labels">
                    <el-tag
                      v-for="(value, key) in getFilteredLabels(node.labels)"
                      :key="`${node.name}-${key}`"
                      size="small"
                      type="info"
                      class="label-tag"
                    >
                      {{ key }}={{ value }}
                    </el-tag>
                  </div>
                </div>
              </div>
            </el-checkbox>
          </div>
        </el-checkbox-group>
        
        <!-- 空状态 -->
        <div v-if="filteredNodes.length === 0 && !loading" class="empty-nodes">
          <el-empty 
            :description="nodes.length === 0 ? '暂无节点数据' : '没有找到匹配的节点'" 
            :image-size="60"
          />
        </div>
        
        <!-- 加载状态 -->
        <div v-if="loading" class="loading-nodes">
          <el-skeleton :rows="3" animated />
        </div>
      </el-scrollbar>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import { Search } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  },
  nodes: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  },
  showLabels: {
    type: Boolean,
    default: true
  },
  maxLabelDisplay: {
    type: Number,
    default: 3
  }
})

const emit = defineEmits(['update:modelValue', 'selection-change'])

// 响应式数据
const searchQuery = ref('')
const statusFilter = ref('')
const roleFilter = ref('')
const labelFilter = ref('')
const selectedNodes = ref([...props.modelValue])

// 监听props变化，避免深度监听造成性能问题
watch(() => props.modelValue, (newValue) => {
  // 只在值确实改变时才更新
  if (JSON.stringify(newValue) !== JSON.stringify(selectedNodes.value)) {
    selectedNodes.value = [...newValue]
  }
}, { immediate: true })

let emitTimer = null
watch(selectedNodes, (newValue) => {
  // 防抖处理，避免频繁触发事件
  if (emitTimer) {
    clearTimeout(emitTimer)
  }
  emitTimer = setTimeout(() => {
    emit('update:modelValue', newValue)
    emit('selection-change', newValue)
  }, 10)
})

// 计算属性 - 优化过滤逻辑，减少不必要的计算
const filteredNodes = computed(() => {
  if (!props.nodes || props.nodes.length === 0) {
    return []
  }
  
  let result = props.nodes

  // 文本搜索
  if (searchQuery.value?.trim()) {
    const query = searchQuery.value.toLowerCase().trim()
    result = result.filter(node => 
      node?.name?.toLowerCase().includes(query)
    )
  }

  // 状态筛选
  if (statusFilter.value) {
    result = result.filter(node => node?.status === statusFilter.value)
  }

  // 角色筛选
  if (roleFilter.value) {
    result = result.filter(node => {
      if (!node?.roles?.length) return false
      return node.roles.some(role => 
        role?.toLowerCase().includes(roleFilter.value.toLowerCase())
      )
    })
  }

  // 标签筛选
  if (labelFilter.value?.trim()) {
    const labelQuery = labelFilter.value.trim()
    const [key, value] = labelQuery.split('=')
    if (key) {
      result = result.filter(node => {
        if (!node?.labels) return false
        if (value !== undefined) {
          return node.labels[key] === value
        } else {
          return key in node.labels
        }
      })
    }
  }

  return result
})

const selectAll = computed({
  get() {
    if (!filteredNodes.value?.length) return false
    
    // 优化检查，避免每次都遍历整个数组
    const filteredNodeNames = filteredNodes.value.map(node => node?.name).filter(Boolean)
    return filteredNodeNames.length > 0 && 
           filteredNodeNames.every(name => selectedNodes.value.includes(name))
  },
  set(value) {
    if (value) {
      // 全选：将筛选后的节点添加到选中列表
      const allNodeNames = filteredNodes.value
        .map(node => node?.name)
        .filter(Boolean)
      selectedNodes.value = [...new Set([...selectedNodes.value, ...allNodeNames])]
    } else {
      // 取消选择筛选后的节点
      const filteredNodeNames = filteredNodes.value
        .map(node => node?.name)
        .filter(Boolean)
      selectedNodes.value = selectedNodes.value.filter(name => 
        !filteredNodeNames.includes(name)
      )
    }
  }
})

const indeterminate = computed(() => {
  if (!filteredNodes.value?.length) return false
  
  const selectedInFiltered = filteredNodes.value.filter(node => 
    node?.name && selectedNodes.value.includes(node.name)
  ).length
  return selectedInFiltered > 0 && selectedInFiltered < filteredNodes.value.length
})

// 方法
const handleSearch = () => {
  // 搜索是响应式的，不需要额外处理
}

const handleFilter = () => {
  // 筛选是响应式的，不需要额外处理
}

const handleSelectAll = (checked) => {
  selectAll.value = checked
}

const handleSelectionChange = () => {
  // 由watch处理
}

const clearSelection = () => {
  selectedNodes.value = []
}

const getStatusType = (status) => {
  const statusMap = {
    'Ready': 'success',
    'NotReady': 'danger',
    'Unknown': 'warning'
  }
  return statusMap[status] || 'info'
}

const getFilteredLabels = (labels) => {
  if (!labels || typeof labels !== 'object') return {}
  try {
    const entries = Object.entries(labels)
    if (entries.length <= props.maxLabelDisplay) {
      return labels
    }
    // 显示前几个标签
    const filteredEntries = entries.slice(0, props.maxLabelDisplay)
    return Object.fromEntries(filteredEntries)
  } catch (error) {
    console.warn('Error filtering labels:', error)
    return {}
  }
}

// 清理定时器
onUnmounted(() => {
  if (emitTimer) {
    clearTimeout(emitTimer)
    emitTimer = null
  }
})
</script>

<style scoped>
.node-selector {
  width: 100%;
}

.selector-header {
  margin-bottom: 16px;
  padding: 16px;
  background: #fafafa;
  border-radius: 6px;
}

.search-section {
  margin-bottom: 12px;
}

.filter-section {
  margin-bottom: 12px;
}

.action-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.node-list {
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  background: #fff;
}

.node-item {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.3s;
}

.node-item:last-child {
  border-bottom: none;
}

.node-item:hover {
  background-color: #f8f9fa;
}

.node-item.selected {
  background-color: #e6f7ff;
}

.node-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-left: 8px;
  width: 100%;
  min-height: 60px;
}

.node-info {
  flex: 1;
  min-width: 0; /* 防止flex子项溢出 */
}

.node-name {
  font-weight: 500;
  color: #333;
  margin-bottom: 6px;
  font-size: 14px;
  line-height: 1.4;
  word-break: break-all; /* 防止长节点名称溢出 */
}

.node-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap; /* 允许换行 */
  margin-bottom: 4px;
}

.node-roles {
  font-size: 12px;
  color: #666;
  white-space: nowrap;
}

.node-details {
  max-width: 250px;
  min-width: 150px;
  text-align: right;
  flex-shrink: 0; /* 防止被压缩 */
}

.node-ip {
  font-size: 12px;
  color: #666;
  margin-bottom: 6px;
  white-space: nowrap;
}

.node-labels {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  justify-content: flex-end;
  max-height: 60px;
  overflow-y: auto;
}

.label-tag {
  font-size: 11px;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.2;
}

.empty-nodes, .loading-nodes {
  padding: 20px;
  text-align: center;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .filter-section .el-col {
    margin-bottom: 8px;
  }
  
  .node-content {
    flex-direction: column;
    gap: 8px;
    min-height: auto;
  }
  
  .node-details {
    max-width: 100%;
    min-width: 0;
    text-align: left;
  }
  
  .node-labels {
    justify-content: flex-start;
    max-height: 40px;
  }
  
  .label-tag {
    max-width: 120px;
  }
  
  .node-item {
    padding: 16px 12px;
  }
  
  .selector-header {
    padding: 12px;
  }
}

@media (max-width: 480px) {
  .filter-section .el-row .el-col {
    margin-bottom: 8px;
  }
  
  .node-name {
    font-size: 13px;
  }
  
  .node-meta {
    gap: 6px;
  }
  
  .label-tag {
    max-width: 100px;
    font-size: 10px;
  }
  
  .node-selector {
    font-size: 13px;
  }
}
</style>