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
                          v-for="(value, key) in getCompactDisplayLabels(node.labels)"
                          :key="`${node.name}-${key}`"
                          size="small"
                          type="info"
                          class="label-tag"
                          :title="`${key}=${value}`"
                        >
                          <span class="label-key">{{ smartTruncateLabel(key, value).key }}</span>
                          <span v-if="value" class="label-separator">=</span>
                          <span v-if="value" class="label-value">{{ smartTruncateLabel(key, value).value }}</span>
                        </el-tag>
                        <el-dropdown
                          v-if="getTotalLabelsCount(node.labels) > 0"
                          trigger="click"
                          placement="bottom-start"
                        >
                          <el-tag
                            size="small"
                            type="info"
                            class="more-labels-tag"
                            :title="`还有${getTotalLabelsCount(node.labels)}个其他标签，点击查看详情`"
                          >
                            详情({{ getTotalLabelsCount(node.labels) }})
                            <el-icon class="more-icon"><ArrowDown /></el-icon>
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu class="labels-dropdown">
                              <div class="dropdown-header">其他节点标签</div>
                              <div class="dropdown-content">
                                <el-tag
                                  v-for="(value, key) in getOtherLabels(node.labels) || {}"
                                  :key="`dropdown-${node.name}-${key}`"
                                  size="small"
                                  type="info"
                                  class="dropdown-label-tag"
                                >
                                  {{ key }}={{ value }}
                                </el-tag>
                              </div>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                      </div>
                    </div>
                    
                    <div v-if="node.taints && node.taints.length > 0" class="attributes-section">
                      <div class="attributes-header">
                        <el-icon class="section-icon"><Warning /></el-icon>
                        <span class="section-label">污点</span>
                      </div>
                      <div class="attributes-content">
                        <el-tag
                          v-for="(taint, index) in getCompactDisplayTaints(node.taints)"
                          :key="`${node.name}-taint-${index}`"
                          size="small"
                          :type="getTaintType(taint.effect)"
                          class="taint-tag"
                          :title="`${taint.key}=${taint.value || ''}:${taint.effect}`"
                        >
                          <span class="taint-key">{{ truncateText(taint.key, 10) }}</span>
                          <span v-if="taint.value" class="taint-separator">=</span>
                          <span v-if="taint.value" class="taint-value">{{ truncateText(taint.value, 8) }}</span>
                          <span class="taint-effect">:{{ taint.effect.substr(0, 2) }}</span>
                        </el-tag>
                        <el-dropdown
                          v-if="node.taints && node.taints.length > 1"
                          trigger="click"
                          placement="bottom-start"
                        >
                          <el-tag
                            size="small"
                            type="danger"
                            class="more-taints-tag"
                            :title="`共${node.taints.length}个污点，点击查看更多`"
                          >
                            +{{ node.taints.length - 1 }}
                            <el-icon class="more-icon"><ArrowDown /></el-icon>
                          </el-tag>
                          <template #dropdown>
                            <el-dropdown-menu class="taints-dropdown">
                              <div class="dropdown-header">节点污点</div>
                              <div class="dropdown-content">
                                <el-tag
                                  v-for="(taint, index) in node.taints || []"
                                  :key="`dropdown-${node.name}-taint-${index}`"
                                  size="small"
                                  :type="getTaintType(taint.effect)"
                                  class="dropdown-taint-tag"
                                >
                                  {{ taint.key }}{{ taint.value ? '=' + taint.value : '' }}:{{ taint.effect }}
                                </el-tag>
                              </div>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
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
import { Search, Location, Collection, Warning, ArrowDown } from '@element-plus/icons-vue'

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
    // 只显示 cluster 和 deeproute.cn/user-type 标签
    const priorityKeys = ['cluster', 'deeproute.cn/user-type']
    const priorityEntries = entries.filter(([key]) => priorityKeys.includes(key))
    return Object.fromEntries(priorityEntries)
  } catch (error) {
    console.warn('Error filtering labels:', error)
    return {}
  }
}

const getCompactDisplayLabels = (labels) => {
  if (!labels || typeof labels !== 'object') return {}
  try {
    const entries = Object.entries(labels)
    // 按优先级显示关键标签
    const priorityKeys = ['cluster', 'deeproute.cn/user-type', 'deeproute.cn/instance-type']
    const priorityEntries = entries.filter(([key]) => priorityKeys.includes(key)).slice(0, 2)
    return Object.fromEntries(priorityEntries)
  } catch (error) {
    console.warn('Error filtering labels:', error)
    return {}
  }
}

const getDisplayTaints = (taints) => {
  if (!taints || !Array.isArray(taints)) return []
  return taints.slice(0, 2) // 显示前2个污点
}

const getCompactDisplayTaints = (taints) => {
  if (!taints || !Array.isArray(taints)) return []
  return taints.slice(0, 1) // 只显示第1个污点，其余折叠
}

const getTotalLabelsCount = (labels) => {
  if (!labels || typeof labels !== 'object') return 0
  // 返回除了优先显示标签外的总数
  const priorityKeys = ['cluster', 'deeproute.cn/user-type', 'deeproute.cn/instance-type']
  const allKeys = Object.keys(labels)
  const otherLabelsCount = allKeys.filter(key => !priorityKeys.includes(key)).length
  return otherLabelsCount
}

const getOtherLabels = (labels) => {
  if (!labels || typeof labels !== 'object') return {}
  try {
    const entries = Object.entries(labels)
    // 返回除了优先显示标签外的所有其他标签
    const priorityKeys = ['cluster', 'deeproute.cn/user-type', 'deeproute.cn/instance-type']
    const otherEntries = entries.filter(([key]) => !priorityKeys.includes(key))
    // 按重要性排序：系统标签优先，自定义标签在后
    const systemLabels = otherEntries.filter(([key]) => 
      key.startsWith('kubernetes.io/') || 
      key.startsWith('node.kubernetes.io/') ||
      key.startsWith('topology.kubernetes.io/')
    )
    const customLabels = otherEntries.filter(([key]) => 
      !key.startsWith('kubernetes.io/') && 
      !key.startsWith('node.kubernetes.io/') &&
      !key.startsWith('topology.kubernetes.io/')
    )
    const sortedEntries = [...systemLabels, ...customLabels]
    return Object.fromEntries(sortedEntries)
  } catch (error) {
    console.warn('Error filtering other labels:', error)
    return {}
  }
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

// 智能截断标签键值，保留关键信息
const smartTruncateLabel = (key, value, maxKeyLength = 15, maxValueLength = 12) => {
  let truncatedKey = key
  let truncatedValue = value
  
  // 对于deeproute相关的键，保留关键部分
  if (key.includes('deeproute.cn/')) {
    const parts = key.split('/')
    if (parts.length > 1) {
      truncatedKey = parts[parts.length - 1] // 只显示最后一部分
    }
  }
  
  // 截断键和值
  if (truncatedKey.length > maxKeyLength) {
    truncatedKey = truncatedKey.substring(0, maxKeyLength) + '..'
  }
  
  if (truncatedValue && truncatedValue.length > maxValueLength) {
    truncatedValue = truncatedValue.substring(0, maxValueLength) + '..'
  }
  
  return { key: truncatedKey, value: truncatedValue }
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

/* 强制垂直布局，防止层叠 */
.node-list :deep(.el-scrollbar__view) {
  display: block !important;
  position: static !important;
}

.node-list :deep(.el-checkbox-group) {
  display: block !important;
  position: static !important;
}

.node-list :deep(.el-checkbox-group .el-checkbox) {
  display: block !important;
  width: 100% !important;
  position: static !important;
  margin: 0 !important;
}

.node-item {
  display: block !important;
  width: 100% !important;
  padding: 18px 16px;
  margin: 0 0 12px 0 !important;
  border-bottom: 1px solid #f0f0f0;
  transition: all 0.3s ease;
  position: static !important; /* 强制静态定位 */
  min-height: 80px; /* 增加最小高度 */
  height: auto;
  box-sizing: border-box;
  clear: both;
  float: none !important;
}

.node-item:last-child {
  border-bottom: none;
  margin-bottom: 0; /* 最后一个节点不需要下边距 */
}

.node-checkbox {
  display: block !important;
  width: 100% !important;
  min-height: 100%;
  position: static !important;
}

/* 确保Element Plus的checkbox组件不会影响布局 */
.node-checkbox :deep(.el-checkbox__label) {
  width: 100% !important;
  padding-left: 0 !important;
  display: block !important;
  position: static !important;
}

.node-checkbox :deep(.el-checkbox) {
  white-space: normal !important;
  line-height: normal !important;
  display: block !important;
  position: static !important;
  float: none !important;
}

.node-checkbox :deep(.el-checkbox__input) {
  position: static !important;
  float: left;
  margin-right: 8px;
}

/* 确保Element Plus dropdown组件的z-index正确 */
:deep(.el-dropdown) {
  position: relative;
  z-index: 3;
}

:deep(.el-dropdown-menu) {
  z-index: 9999 !important;
}

:deep(.el-popper) {
  z-index: 9999 !important;
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
  display: block;
  margin-left: 32px; /* 为checkbox留出空间 */
  width: calc(100% - 40px);
  padding: 8px 0;
  position: static;
  clear: left;
  line-height: 1.5; /* 增加行高 */
}

.node-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  margin-bottom: 8px; /* 增加与下方的间距 */
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
  gap: 12px;
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
  margin-top: 8px;
  width: 100%;
  box-sizing: border-box;
}

.attributes-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  box-sizing: border-box;
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
  flex-wrap: wrap; /* 允许换行但控制行数 */
  gap: 8px 6px; /* 行间距8px，列间距6px */
  align-items: flex-start;
  line-height: 1.4;
  width: 100%;
  box-sizing: border-box;
  min-height: 24px; /* 确保最小高度 */
}

.label-tag {
  font-size: 11px;
  height: 22px;
  line-height: 20px;
  padding: 0 8px;
  font-family: 'Monaco', 'Menlo', monospace;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
  display: inline-flex;
  align-items: center;
  vertical-align: top;
  white-space: nowrap;
  flex-shrink: 0; /* 防止被压缩 */
  max-width: 200px; /* 限制最大宽度 */
  overflow: hidden;
  text-overflow: ellipsis;
  margin: 2px 0; /* 增加上下边距 */
}

.label-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  border-color: #d9ecff;
  z-index: 5; /* 降低z-index避免遮挡其他节点 */
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
  height: 20px;
  line-height: 18px;
  padding: 0 6px;
  font-family: 'Monaco', 'Menlo', monospace;
  border-radius: 3px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid transparent;
  display: inline-flex; /* 保证垂直居中且不叠压 */
  align-items: center;
  vertical-align: top;
}

.taint-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  z-index: 5; /* 降低z-index避免遮挡其他节点 */
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
  font-size: 11px;
  height: 22px;
  line-height: 20px;
  padding: 0 8px;
  cursor: pointer;
  font-weight: 600;
  border-radius: 4px;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  vertical-align: top;
  background: #f0f0f0 !important;
  border: 1px solid #d9d9d9 !important;
  color: #666 !important;
  flex-shrink: 0;
  white-space: nowrap;
  margin: 2px 0; /* 增加上下边距 */
}

.more-labels-tag:hover,
.more-taints-tag:hover {
  background: #e6f7ff !important;
  border-color: #91d5ff !important;
  color: #1890ff !important;
}

.more-icon {
  font-size: 8px;
  margin-left: 2px;
  transition: transform 0.2s ease;
}

/* 下拉菜单样式 */
.labels-dropdown,
.taints-dropdown {
  min-width: 260px;
  max-width: 400px;
  z-index: 9999 !important; /* 确保下拉菜单在最顶层 */
}

/* 确保下拉菜单触发器不会遮挡其他节点 */
.more-labels-tag.el-dropdown__trigger,
.more-taints-tag.el-dropdown__trigger {
  z-index: 6 !important;
}

.dropdown-header {
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 600;
  color: #666;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.dropdown-content {
  padding: 10px;
  max-height: 200px;
  overflow-y: auto;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.dropdown-label-tag,
.dropdown-taint-tag {
  font-size: 11px;
  height: 22px;
  line-height: 20px;
  padding: 0 8px;
  border-radius: 11px;
  font-weight: 500;
  margin: 0;
  font-family: 'Monaco', 'Menlo', monospace;
  word-break: break-all;
  max-width: 100%;
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
    padding: 16px 12px;
    min-height: 70px;
  }
  
  .node-content {
    padding: 6px 0;
    line-height: 1.4;
  }
  
  .node-header {
    gap: 6px;
    margin-bottom: 6px;
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
    gap: 8px;
    padding-top: 8px;
  }
  
  .attributes-section {
    gap: 6px;
  }
  
  .attributes-content {
    gap: 6px 4px;
    min-height: 20px;
  }
  
  .label-tag,
  .taint-tag {
    font-size: 10px;
    height: 20px;
    line-height: 18px;
    padding: 0 6px;
    margin: 1px 0;
  }
  
  .more-labels-tag,
  .more-taints-tag {
    font-size: 10px;
    height: 20px;
    line-height: 18px;
    padding: 0 6px;
    margin: 1px 0;
  }
  
  .more-icon {
    font-size: 7px;
    margin-left: 1px;
  }
  
  .labels-dropdown,
  .taints-dropdown {
    min-width: 200px;
    max-width: 280px;
  }
  
  .dropdown-content {
    max-height: 150px;
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
    padding: 14px 10px;
    min-height: 60px;
  }
  
  .node-content {
    padding: 4px 0;
    margin-left: 28px;
    width: calc(100% - 32px);
  }
  
  .node-header {
    gap: 4px;
    margin-bottom: 4px;
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
    gap: 6px;
    padding-top: 6px;
  }
  
  .attributes-section {
    gap: 4px;
  }
  
  .attributes-content {
    gap: 4px 3px;
    min-height: 18px;
  }
  
  .label-tag,
  .taint-tag {
    font-size: 9px;
    height: 18px;
    line-height: 16px;
    padding: 0 4px;
    margin: 1px 0;
    max-width: 120px;
  }
  
  .more-labels-tag,
  .more-taints-tag {
    font-size: 9px;
    height: 18px;
    line-height: 16px;
    padding: 0 4px;
    margin: 1px 0;
  }
  
  .more-icon {
    font-size: 6px;
    margin-left: 1px;
  }
  
  .labels-dropdown,
  .taints-dropdown {
    min-width: 180px;
    max-width: 240px;
  }
  
  .dropdown-content {
    max-height: 120px;
  }
  
  .dropdown-label-tag,
  .dropdown-taint-tag {
    font-size: 10px;
    height: 20px;
    line-height: 18px;
    padding: 0 6px;
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