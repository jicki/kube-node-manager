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
              class="node-checkbox"
            >
              <div class="node-content">
                  <div class="node-header">
                    <div class="node-main-info">
                      <div class="node-name">{{ node.name || '未知节点' }}</div>
                      <div class="node-basic-row">
                        <el-tag 
                          :type="getStatusType(node.status)" 
                          size="small"
                          class="node-status-tag"
                        >
                          {{ node.status || 'Unknown' }}
                        </el-tag>
                        <span v-if="node.roles?.length" class="node-roles">
                          {{ node.roles.join(', ') }}
                        </span>
                      </div>
                      <div v-if="node.internal_ip" class="node-ip-row">
                        <el-icon class="ip-icon"><Location /></el-icon>
                        <span class="ip-text">{{ node.internal_ip }}</span>
                      </div>
                    </div>
                  </div>
                  
                  <div v-if="(node.labels && showLabels && Object.keys(getDisplayLabels(node.labels)).length > 0) || (node.taints && node.taints.length > 0)" class="node-attributes">
                    <div v-if="node.labels && showLabels && Object.keys(getDisplayLabels(node.labels)).length > 0" class="attributes-section">
                      <div class="attributes-header">
                        <el-icon class="section-icon"><Collection /></el-icon>
                        <span class="section-label">标签</span>
                      </div>
                      <div class="attributes-content">
                        <el-tag
                          v-for="(value, key) in getDisplayLabels(node.labels)"
                          :key="`${node.name}-${key}`"
                          size="small"
                          type="info"
                          class="label-tag"
                          :title="`${key}=${value}`"
                        >
                          <span class="label-key">{{ truncateText(key, 15) }}</span>
                          <span v-if="value" class="label-separator">=</span>
                          <span v-if="value" class="label-value">{{ truncateText(value, 12) }}</span>
                        </el-tag>
                        <el-tag
                          v-if="getTotalLabelsCount(node.labels) > maxLabelDisplay"
                          size="small"
                          type="warning"
                          class="more-labels-tag"
                          :title="`共${getTotalLabelsCount(node.labels)}个标签，点击查看更多`"
                          @click="showAllLabels(node)"
                        >
                          +{{ getTotalLabelsCount(node.labels) - maxLabelDisplay }}
                        </el-tag>
                      </div>
                    </div>
                    
                    <div v-if="node.taints && node.taints.length > 0" class="attributes-section">
                      <div class="attributes-header">
                        <el-icon class="section-icon"><Warning /></el-icon>
                        <span class="section-label">污点</span>
                      </div>
                      <div class="attributes-content">
                        <el-tag
                          v-for="(taint, index) in getDisplayTaints(node.taints)"
                          :key="`${node.name}-taint-${index}`"
                          size="small"
                          :type="getTaintType(taint.effect)"
                          class="taint-tag"
                          :title="`${taint.key}=${taint.value || ''}:${taint.effect}`"
                        >
                          <span class="taint-key">{{ truncateText(taint.key, 12) }}</span>
                          <span v-if="taint.value" class="taint-separator">=</span>
                          <span v-if="taint.value" class="taint-value">{{ truncateText(taint.value, 10) }}</span>
                          <span class="taint-effect">:{{ taint.effect.substr(0, 2) }}</span>
                        </el-tag>
                        <el-tag
                          v-if="node.taints.length > 2"
                          size="small"
                          type="danger"
                          class="more-taints-tag"
                          :title="`共${node.taints.length}个污点，点击查看更多`"
                          @click="showAllTaints(node)"
                        >
                          +{{ node.taints.length - 2 }}
                        </el-tag>
                      </div>
                    </div>
                  </div>
                </div>
            </el-checkbox>
          </div>
        </el-checkbox-group>
        
        <!-- 空状态 -->
        <div v-if="filteredNodes.length === 0 && !loading" class="empty-nodes">
          <el-empty 
            :description="getEmptyDescription()" 
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
import { Search, Location, Collection, Warning } from '@element-plus/icons-vue'

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

// 数据验证函数
const validateNodeData = (node) => {
  return node && 
         typeof node === 'object' && 
         node.name && 
         typeof node.name === 'string' &&
         node.name.trim().length > 0
}

// 计算属性 - 优化过滤逻辑，减少不必要的计算
const filteredNodes = computed(() => {
  if (!props.nodes || props.nodes.length === 0) {
    return []
  }
  
  // 首先过滤掉无效的节点数据
  let result = props.nodes.filter(validateNodeData)

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

const getEmptyDescription = () => {
  if (!props.nodes || props.nodes.length === 0) {
    return '暂无节点数据'
  }
  
  // 检查是否有无效数据被过滤掉
  const validNodes = props.nodes.filter(validateNodeData)
  if (validNodes.length === 0) {
    return '节点数据格式无效，请刷新重试'
  }
  
  return '没有找到匹配的节点'
}

const getStatusType = (status) => {
  const statusMap = {
    'Ready': 'success',
    'NotReady': 'danger',
    'Unknown': 'warning'
  }
  return statusMap[status] || 'info'
}

const getDisplayLabels = (labels) => {
  if (!labels || typeof labels !== 'object') return {}
  try {
    const entries = Object.entries(labels)
    const systemLabels = entries.filter(([key]) => 
      key.startsWith('kubernetes.io/') || 
      key.startsWith('node.kubernetes.io/') ||
      key.startsWith('topology.kubernetes.io/')
    )
    const customLabels = entries.filter(([key]) => 
      !key.startsWith('kubernetes.io/') && 
      !key.startsWith('node.kubernetes.io/') &&
      !key.startsWith('topology.kubernetes.io/')
    )
    
    // 优先显示自定义标签，再显示系统标签
    const prioritizedEntries = [...customLabels, ...systemLabels]
    const displayEntries = prioritizedEntries.slice(0, props.maxLabelDisplay)
    return Object.fromEntries(displayEntries)
  } catch (error) {
    console.warn('Error filtering labels:', error)
    return {}
  }
}

const getDisplayTaints = (taints) => {
  if (!taints || !Array.isArray(taints)) return []
  return taints.slice(0, 2) // 显示前2个污点
}

const getTotalLabelsCount = (labels) => {
  if (!labels || typeof labels !== 'object') return 0
  return Object.keys(labels).length
}

const getTaintType = (effect) => {
  const typeMap = {
    'NoSchedule': 'danger',
    'PreferNoSchedule': 'warning', 
    'NoExecute': 'error'
  }
  return typeMap[effect] || 'info'
}

const truncateText = (text, maxLength) => {
  if (!text) return ''
  return text.length > maxLength ? text.substring(0, maxLength) + '..' : text
}

const showAllLabels = (node) => {
  // 可以在这里实现显示所有标签的逻辑，比如弹出对话框
  const labelsText = Object.entries(node.labels || {})
    .map(([key, value]) => `${key}=${value}`)
    .join('\n')
  ElMessageBox.alert(labelsText, `节点 ${node.name} 的所有标签`, {
    confirmButtonText: '关闭',
    customStyle: {
      'white-space': 'pre-line',
      'font-family': 'Monaco, Menlo, monospace',
      'font-size': '12px'
    }
  })
}

const showAllTaints = (node) => {
  // 显示所有污点
  const taintsText = (node.taints || [])
    .map(taint => `${taint.key}${taint.value ? '=' + taint.value : ''}:${taint.effect}`)
    .join('\n')
  ElMessageBox.alert(taintsText, `节点 ${node.name} 的所有污点`, {
    confirmButtonText: '关闭',
    customStyle: {
      'white-space': 'pre-line',
      'font-family': 'Monaco, Menlo, monospace',
      'font-size': '12px'
    }
  })
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
  position: relative;
}

.node-item {
  padding: 16px;
  margin-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
  transition: all 0.3s ease;
  position: relative;
  min-height: 60px; /* 确保最小高度 */
}

.node-item:last-child {
  border-bottom: none;
  margin-bottom: 0; /* 最后一个节点不需要下边距 */
}

.node-checkbox {
  display: flex;
  align-items: flex-start;
  width: 100%;
  min-height: 100%; /* 确保checkbox占满整个节点项高度 */
}

/* 确保Element Plus的checkbox组件不会影响布局 */
.node-checkbox :deep(.el-checkbox__label) {
  width: 100%;
  padding-left: 0;
}

.node-checkbox :deep(.el-checkbox) {
  white-space: normal;
  line-height: normal;
}

.node-item:hover {
  background-color: #f8f9fa;
  border-left: 3px solid #1890ff;
  padding-left: 13px;
}

.node-item.selected {
  background-color: #e6f7ff;
  border-left: 3px solid #1890ff;
  padding-left: 13px;
}

.node-content {
  display: flex;
  flex-direction: column;
  margin-left: 8px;
  width: calc(100% - 8px);
  gap: 8px;
  padding: 4px 0;
  flex: 1; /* 确保内容区域占满可用空间 */
}

.node-header {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
}

.node-main-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.node-name {
  font-weight: 600;
  color: #333;
  font-size: 15px;
  line-height: 1.3;
  word-break: break-word;
  margin: 0;
  flex-shrink: 0;
}

.node-basic-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  min-height: 24px;
}

.node-status-tag {
  font-weight: 500;
  flex-shrink: 0;
}

.node-roles {
  font-size: 12px;
  color: #666;
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
  white-space: nowrap;
  flex-shrink: 0;
  height: 20px;
  line-height: 16px;
  display: inline-flex;
  align-items: center;
}

.node-ip-row {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #52c41a;
  background: #f6ffed;
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid #b7eb8f;
  white-space: nowrap;
  flex-shrink: 0;
  height: 20px;
  line-height: 16px;
}

.ip-icon {
  font-size: 12px;
  color: #52c41a;
  flex-shrink: 0;
}

.ip-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 500;
  font-size: 12px;
}

.node-attributes {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 1px solid #f0f0f0;
  padding-top: 8px;
  margin-top: 2px;
}

.attributes-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.attributes-header {
  display: flex;
  align-items: center;
  gap: 4px;
}

.section-icon {
  font-size: 12px;
  color: #999;
  flex-shrink: 0;
}

.section-label {
  font-size: 11px;
  color: #999;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  flex-shrink: 0;
}

.attributes-content {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: flex-start;
  line-height: 1;
}

.label-tag {
  font-size: 10px;
  height: 18px;
  line-height: 16px;
  padding: 0 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  border-radius: 2px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
  display: inline-block;
  vertical-align: top;
  margin: 1px 2px 1px 0;
}

.label-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  border-color: #d9ecff;
  z-index: 10;
  position: relative;
}

.label-key {
  font-weight: 600;
  color: #1890ff;
}

.label-separator {
  margin: 0 1px;
  opacity: 0.7;
  color: #666;
}

.label-value {
  font-weight: 500;
  opacity: 0.9;
  color: #52c41a;
}

.taint-tag {
  font-size: 10px;
  height: 18px;
  line-height: 16px;
  padding: 0 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  border-radius: 2px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
  display: inline-block;
  vertical-align: top;
  margin: 1px 2px 1px 0;
}

.taint-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  z-index: 10;
  position: relative;
}

.taint-key {
  font-weight: 600;
}

.taint-separator {
  margin: 0 1px;
  opacity: 0.7;
}

.taint-value {
  font-weight: 500;
  opacity: 0.9;
}

.taint-effect {
  font-weight: 700;
  margin-left: 1px;
  text-transform: uppercase;
}

.more-labels-tag,
.more-taints-tag {
  font-size: 10px;
  height: 18px;
  line-height: 16px;
  padding: 0 4px;
  cursor: pointer;
  font-weight: 600;
  border-radius: 2px;
  transition: all 0.2s ease;
  display: inline-block;
  vertical-align: top;
  margin: 1px 2px 1px 0;
}

.more-labels-tag:hover,
.more-taints-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
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
  
  .node-item {
    padding: 14px 12px;
  }
  
  .node-content {
    gap: 10px;
  }
  
  .node-name {
    font-size: 14px;
  }
  
  .node-basic-row {
    gap: 8px;
  }
  
  .node-ip-row {
    padding: 4px 8px;
    font-size: 12px;
  }
  
  .ip-text {
    font-size: 12px;
  }
  
  .node-attributes {
    gap: 6px;
    padding-top: 6px;
  }
  
  .attributes-content {
    gap: 2px;
  }
  
  .label-tag,
  .taint-tag {
    font-size: 9px;
    height: 18px;
    line-height: 16px;
    padding: 0 4px;
  }
  
  .more-labels-tag,
  .more-taints-tag {
    font-size: 9px;
    height: 18px;
    line-height: 16px;
    padding: 0 4px;
  }
  
  .section-label {
    font-size: 10px;
  }
  
  .selector-header {
    padding: 12px;
  }
}

@media (max-width: 480px) {
  .filter-section .el-row .el-col {
    margin-bottom: 8px;
  }
  
  .node-item {
    padding: 12px 10px;
  }
  
  .node-content {
    gap: 8px;
  }
  
  .node-name {
    font-size: 13px;
  }
  
  .node-basic-row {
    gap: 6px;
    flex-wrap: wrap;
  }
  
  .node-roles {
    font-size: 11px;
    padding: 2px 5px;
  }
  
  .node-ip-row {
    padding: 3px 6px;
    font-size: 11px;
  }
  
  .ip-text {
    font-size: 11px;
  }
  
  .node-attributes {
    gap: 4px;
    padding-top: 4px;
  }
  
  .attributes-content {
    gap: 2px;
  }
  
  .label-tag,
  .taint-tag {
    font-size: 8px;
    height: 16px;
    line-height: 14px;
    padding: 0 3px;
  }
  
  .more-labels-tag,
  .more-taints-tag {
    font-size: 8px;
    height: 16px;
    line-height: 14px;
    padding: 0 3px;
  }
  
  .section-label {
    font-size: 9px;
  }
  
  .attributes-header .el-icon {
    font-size: 10px;
  }
  
  .node-selector {
    font-size: 12px;
  }
}
</style>