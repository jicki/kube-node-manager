<template>
  <div class="node-selector">
    <!-- 节点搜索和筛选 -->
    <div class="selector-header">
      <div class="search-section">
        <el-input
          v-model="searchQuery"
          placeholder="搜索节点名称..."
          clearable
          @input="handleSearchDebounced"
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
            <el-select
              v-model="nodeOwnershipFilter"
              placeholder="节点归属"
              clearable
              @change="handleFilter"
            >
              <el-option label="全部归属" value="" />
              <el-option 
                v-for="ownership in nodeOwnershipOptions" 
                :key="ownership" 
                :label="ownership" 
                :value="ownership" 
              />
            </el-select>
          </el-col>

        </el-row>
        
        <!-- 高级搜索区域 -->
        <div class="advanced-search-toggle">
          <el-button type="text" @click="showAdvancedSearch = !showAdvancedSearch">
            <el-icon><Filter /></el-icon>
            高级搜索
            <el-icon><component :is="showAdvancedSearch ? 'ArrowUp' : 'ArrowDown'" /></el-icon>
          </el-button>
        </div>
        
        <div v-show="showAdvancedSearch" class="advanced-search">
          <el-divider content-position="left">标签搜索</el-divider>
          <el-row :gutter="12">
            <el-col :span="12">
              <el-input
                v-model="labelKeyFilter"
                placeholder="输入标签键，如 node-role.kubernetes.io/master"
                clearable
                @input="handleFilter"
              >
                <template #prefix>
                  <el-icon><CollectionTag /></el-icon>
                </template>
              </el-input>
            </el-col>
            <el-col :span="12">
              <el-input
                v-model="labelValueFilter"
                placeholder="输入标签值（可选）"
                clearable
                @input="handleFilter"
              >
                <template #prefix>
                  <el-icon><Edit /></el-icon>
                </template>
              </el-input>
            </el-col>
          </el-row>
          
          <el-divider content-position="left">污点搜索</el-divider>
          <el-row :gutter="12">
            <el-col :span="8">
              <el-input
                v-model="taintKeyFilter"
                placeholder="输入污点键..."
                clearable
                @input="handleFilter"
              >
                <template #prefix>
                  <el-icon><WarningFilled /></el-icon>
                </template>
              </el-input>
            </el-col>
            <el-col :span="8">
              <el-input
                v-model="taintValueFilter"
                placeholder="输入污点值（可选）..."
                clearable
                @input="handleFilter"
              >
                <template #prefix>
                  <el-icon><Edit /></el-icon>
                </template>
              </el-input>
            </el-col>
            <el-col :span="8">
              <el-select
                v-model="taintEffectFilter"
                placeholder="污点效果（可选）"
                clearable
                @change="handleFilter"
              >
                <el-option label="NoSchedule" value="NoSchedule" />
                <el-option label="PreferNoSchedule" value="PreferNoSchedule" />
                <el-option label="NoExecute" value="NoExecute" />
              </el-select>
            </el-col>
          </el-row>
        </div>
      </div>

      <div class="action-section">
        <el-checkbox
          v-model="selectAll"
          :indeterminate="indeterminate"
          @change="handleSelectAll"
        >
          全选 ({{ selectedNodes.length }}/{{ totalFilteredCount }})
        </el-checkbox>
        <div class="action-controls">
          <el-button type="text" size="small" @click="clearSelection">
            清空选择
          </el-button>
          <div class="pagination-info">
            <span class="total-info">
              共 {{ totalFilteredCount }} 个节点
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 节点列表 -->
    <div class="node-list">
      <!-- 加载状态 -->
      <div v-if="isFiltering" class="loading-container">
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>正在过滤节点...</span>
      </div>
      
      <!-- 无数据状态 -->
      <div v-else-if="filteredNodesCache.length === 0" class="empty-container">
        <el-empty description="没有找到匹配的节点" :image-size="100" />
      </div>
      
      <!-- 节点列表内容 -->
      <div v-else>
        <el-scrollbar height="300px">
          <el-checkbox-group v-model="selectedNodes" @change="handleSelectionChange">
            <div class="node-container">
              <div
                v-for="node in filteredNodesCache"
                :key="node?.name || node?.id || Math.random()"
                class="node-item"
                :class="{ 'selected': getNodeName(node) && selectedNodes.includes(getNodeName(node)) }"
              >
                <el-checkbox 
                  :value="getNodeName(node) || `node-${Math.random()}`"
                  :disabled="!getNodeName(node)"
                  class="node-checkbox"
                />
            
            <div class="node-content">
              <div class="node-header">
                <h4 class="node-name">{{ getNodeName(node) || '未知节点' }}</h4>
                
                <div class="node-basic-info">
                  <el-tag 
                    :type="getStatusType(node.status)" 
                    size="small"
                    class="status-tag"
                  >
                    {{ node.status || 'Unknown' }}
                  </el-tag>
                  <span v-if="node.roles?.length" class="node-roles">
                    {{ node.roles.join(', ') }}
                  </span>
                </div>
                
                <div v-if="node.internal_ip" class="node-ip">
                  <el-icon class="ip-icon"><Location /></el-icon>
                  <span class="ip-text">{{ node.internal_ip }}</span>
                </div>
              </div>
              
              <div v-if="shouldShowAttributes(node)" class="node-attributes">
                <!-- 标签区域 -->
                <div v-if="shouldShowLabels(node)" class="labels-section">
                  <div class="section-header">
                    <el-icon class="section-icon"><Collection /></el-icon>
                    <span class="section-title">标签</span>
                  </div>
                  <div class="tags-container">
                    <el-tag
                      v-for="(value, key) in getCompactDisplayLabels(node.labels)"
                      :key="`${node.name}-${key}`"
                      size="small"
                      type="info"
                      class="attribute-tag label-tag"
                      :title="`${key}=${value}`"
                    >
                      <span class="tag-key">{{ smartTruncateLabel(key, value).key }}</span>
                      <span v-if="value" class="tag-separator">=</span>
                      <span v-if="value" class="tag-value">{{ smartTruncateLabel(key, value).value }}</span>
                    </el-tag>
                    
                    <el-dropdown
                      v-if="getTotalLabelsCount(node.labels) > 0"
                      trigger="click"
                      placement="bottom-start"
                      class="more-dropdown"
                    >
                      <el-tag
                        size="small"
                        type="info"
                        class="attribute-tag more-tag"
                        :title="`还有${getTotalLabelsCount(node.labels)}个其他标签，点击查看详情`"
                      >
                        详情({{ getTotalLabelsCount(node.labels) }})
                        <el-icon class="arrow-icon"><ArrowDown /></el-icon>
                      </el-tag>
                      <template #dropdown>
                        <el-dropdown-menu class="attributes-dropdown">
                          <div class="dropdown-header">其他节点标签</div>
                          <div class="dropdown-body">
                            <el-tag
                              v-for="(value, key) in getOtherLabels(node.labels) || {}"
                              :key="`dropdown-${node.name}-${key}`"
                              size="small"
                              type="info"
                              class="dropdown-tag"
                            >
                              {{ key }}={{ value }}
                            </el-tag>
                          </div>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </div>
                </div>
                
                <!-- 污点区域 -->
                <div v-if="node.taints && node.taints.length > 0" class="taints-section">
                  <div class="section-header">
                    <el-icon class="section-icon"><Warning /></el-icon>
                    <span class="section-title">污点</span>
                  </div>
                  <div class="tags-container">
                    <el-tag
                      v-for="(taint, index) in getCompactDisplayTaints(node.taints)"
                      :key="`${node.name}-taint-${index}`"
                      size="small"
                      :type="getTaintType(taint.effect)"
                      class="attribute-tag taint-tag"
                      :title="`${taint.key}${taint.value ? '=' + taint.value : ''}:${taint.effect}`"
                    >
                      <span class="tag-key">{{ smartFormatTaint(taint).key }}</span>
                      <span v-if="smartFormatTaint(taint).value" class="tag-separator">=</span>
                      <span v-if="smartFormatTaint(taint).value" class="tag-value">{{ smartFormatTaint(taint).value }}</span>
                      <span class="taint-effect">:{{ smartFormatTaint(taint).effect }}</span>
                    </el-tag>
                    
                    <el-dropdown
                      v-if="node.taints && node.taints.length > 1"
                      trigger="click"
                      placement="bottom-start"
                      class="more-dropdown"
                    >
                      <el-tag
                        size="small"
                        type="danger"
                        class="attribute-tag more-tag"
                        :title="`共${node.taints.length}个污点，点击查看更多`"
                      >
                        +{{ node.taints.length - 1 }}
                        <el-icon class="arrow-icon"><ArrowDown /></el-icon>
                      </el-tag>
                      <template #dropdown>
                        <el-dropdown-menu class="attributes-dropdown">
                          <div class="dropdown-header">节点污点</div>
                          <div class="dropdown-body">
                            <el-tag
                              v-for="(taint, index) in node.taints || []"
                              :key="`dropdown-${node.name}-taint-${index}`"
                              size="small"
                              :type="getTaintType(taint.effect)"
                              class="dropdown-tag"
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
          </div>
        </div>
        </el-checkbox-group>
        
        <!-- 空状态 -->
        <div v-if="filteredNodes.length === 0 && !loading" class="empty-nodes">
          <el-empty 
            :description="getEmptyDescription()" 
            :image-size="60"
          />
        </div>
        
        </el-scrollbar>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import { Search, Location, Collection, Warning, ArrowDown, Filter, WarningFilled, Edit, ArrowUp, CollectionTag, Loading } from '@element-plus/icons-vue'

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
const debouncedSearchQuery = ref('')
const statusFilter = ref('')
const roleFilter = ref('')
const labelKeyFilter = ref('')
const labelValueFilter = ref('')
const nodeOwnershipFilter = ref('')
const taintKeyFilter = ref('')
const taintValueFilter = ref('')
const taintEffectFilter = ref('')
const showAdvancedSearch = ref(false)
const selectedNodes = ref([...props.modelValue])

// 性能优化相关
const isFiltering = ref(false)
let searchDebounceTimer = null

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

// 获取节点名称的统一函数
const getNodeName = (node) => {
  return node?.name || node?.node_name || node?.nodeName || node?.metadata?.name || null
}

// 数据验证函数 - 宽松但合理的验证
const validateNodeData = (node) => {
  // 宽松验证：有对象且有某种名称字段即可
  const isValid = node && 
         typeof node === 'object' && 
         (node.name || node.node_name || node.nodeName || node.metadata?.name)
  
  return isValid
}

// 节点归属选项计算
const nodeOwnershipOptions = computed(() => {
  if (!props.nodes || props.nodes.length === 0) {
    return []
  }
  
  const ownershipSet = new Set()
  let hasNoOwnership = false
  
  props.nodes.forEach(node => {
    const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
    // 检查标签是否存在且不为空字符串
    if (userTypeLabel && userTypeLabel.trim() !== '') {
      ownershipSet.add(userTypeLabel)
    } else {
      hasNoOwnership = true
    }
  })
  
  const options = Array.from(ownershipSet).sort()
  
  // 如果有节点没有 deeproute.cn/user-type 标签，添加"无归属"选项
  if (hasNoOwnership) {
    options.unshift('无归属')
  }
  
  return options
})

// 计算属性 - 优化过滤逻辑，减少不必要的计算  
// 优化的过滤逻辑 - 支持分页和性能优化
const filteredNodesCache = computed(() => {
  if (!props.nodes || props.nodes.length === 0) {
    return []
  }
  
  // 首先过滤掉无效的节点数据
  let result = props.nodes.filter(validateNodeData)

  // 文本搜索 - 使用防抖后的搜索词
  if (debouncedSearchQuery.value?.trim()) {
    const query = debouncedSearchQuery.value.toLowerCase().trim()
    result = result.filter(node => 
      getNodeName(node)?.toLowerCase().includes(query)
    )
  }

  // 状态筛选
  if (statusFilter.value) {
    result = result.filter(node => node?.status === statusFilter.value)
  }

  // 角色筛选
  if (roleFilter.value) {
    result = result.filter(node => {
      if (!node?.roles || !Array.isArray(node.roles)) {
        return roleFilter.value === 'worker' // 无角色视为worker
      }
      
      if (roleFilter.value === 'master') {
        // 检查是否为master相关角色
        return node.roles.some(role => 
          role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
        )
      } else if (roleFilter.value === 'worker') {
        // 检查是否为worker (不包含master相关角色)
        return !node.roles.some(role => 
          role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')
        )
      }
      
      return false
    })
  }

  // 节点归属筛选
  if (nodeOwnershipFilter.value) {
    result = result.filter(node => {
      const userTypeLabel = node.labels && node.labels['deeproute.cn/user-type']
      
      // 如果选择的是"无归属"，过滤出没有或为空的 deeproute.cn/user-type 标签的节点
      if (nodeOwnershipFilter.value === '无归属') {
        return !userTypeLabel || userTypeLabel.trim() === ''
      }
      
      // 否则过滤具有匹配标签值的节点
      if (!userTypeLabel || userTypeLabel.trim() === '') {
        return false
      }
      return userTypeLabel === nodeOwnershipFilter.value
    })
  }

  // 标签筛选
  if (labelKeyFilter.value?.trim()) {
    result = result.filter(node => {
      if (!node?.labels) return false
      const key = labelKeyFilter.value.trim()
      
      // 检查标签键是否存在
      if (!(key in node.labels)) {
        return false
      }
      
      // 如果指定了标签值，进行精确匹配
      if (labelValueFilter.value?.trim()) {
        return node.labels[key] === labelValueFilter.value.trim()
      }
      
      // 否则只检查标签键是否存在
      return true
    })
  }

  // 污点筛选
  if (taintKeyFilter.value) {
    result = result.filter(node => {
      if (!node.taints || node.taints.length === 0) {
        return false
      }
      return node.taints.some(taint => {
        if (taint.key !== taintKeyFilter.value) {
          return false
        }
        // 如果指定了污点值，进行值匹配
        if (taintValueFilter.value && taint.value !== taintValueFilter.value) {
          return false
        }
        // 如果指定了污点效果，进行效果匹配
        if (taintEffectFilter.value && taint.effect !== taintEffectFilter.value) {
          return false
        }
        return true
      })
    })
  }

  return result
})

// 总过滤节点数量
const totalFilteredCount = computed(() => filteredNodesCache.value.length)

// 直接使用过滤后的节点
const filteredNodes = computed(() => filteredNodesCache.value)

const selectAll = computed({
  get() {
    if (!filteredNodes.value?.length) return false
    
    // 优化检查，避免每次都遍历整个数组
    const filteredNodeNames = filteredNodes.value.map(node => getNodeName(node)).filter(Boolean)
    return filteredNodeNames.length > 0 && 
           filteredNodeNames.every(name => selectedNodes.value.includes(name))
  },
  set(value) {
    if (value) {
      // 全选：将所有过滤的节点添加到选中列表
      const allNodeNames = filteredNodes.value
        .map(node => getNodeName(node))
        .filter(Boolean)
      selectedNodes.value = [...new Set([...selectedNodes.value, ...allNodeNames])]
    } else {
      // 取消选择所有过滤的节点
      const filteredNodeNames = filteredNodes.value
        .map(node => getNodeName(node))
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
    getNodeName(node) && selectedNodes.value.includes(getNodeName(node))
  ).length
  return selectedInFiltered > 0 && selectedInFiltered < filteredNodes.value.length
})

// 方法
const handleSearchDebounced = () => {
  isFiltering.value = true
  
  // 清除之前的定时器
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
  }
  
  // 设置防抖延迟
  searchDebounceTimer = setTimeout(() => {
    debouncedSearchQuery.value = searchQuery.value
    isFiltering.value = false
  }, 300)
}

const handleFilter = () => {
  console.log('Filter changed:', { 
    status: statusFilter.value, 
    role: roleFilter.value, 
    labelKey: labelKeyFilter.value,
    labelValue: labelValueFilter.value,
    nodeOwnership: nodeOwnershipFilter.value,
    taintKey: taintKeyFilter.value,
    taintValue: taintValueFilter.value,
    taintEffect: taintEffectFilter.value,
    totalFiltered: totalFilteredCount.value
  })
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
const smartTruncateLabel = (key, value, maxKeyLength = 35, maxValueLength = 20) => {
  let truncatedKey = key
  let truncatedValue = value
  
  // 重要标签列表：这些标签应该完整显示，不被截断
  const importantKeys = [
    'deeproute.cn/instance-type',
    'deeproute.cn/user-type', 
    'deeproute.cn/node-type',
    'kubernetes.io/hostname',
    'kubernetes.io/arch',
    'kubernetes.io/os',
    'node.kubernetes.io/instance-type',
    'nvidia.com/gpu'
  ]
  
  // 重要标签完全不截断
  if (importantKeys.includes(key)) {
    // 对于重要标签，保持完整显示
    truncatedKey = key
  } else {
    // 其他标签在超出限制时进行智能截断
    if (key.length > maxKeyLength) {
      // 对于带命名空间的标签，尝试保留关键部分
      if (key.includes('/')) {
        const parts = key.split('/')
        if (parts.length === 2) {
          const namespace = parts[0]
          const name = parts[1]
          // 如果命名空间太长，截断命名空间但保留名称
          if (namespace.length > 20) {
            truncatedKey = `${namespace.substring(0, 15)}.../${name}`
          } else {
            truncatedKey = key.substring(0, maxKeyLength) + '..'
          }
        } else {
          truncatedKey = key.substring(0, maxKeyLength) + '..'
        }
      } else {
        truncatedKey = key.substring(0, maxKeyLength) + '..'
      }
    }
  }
  
  // 值的截断也更宽松
  if (truncatedValue && truncatedValue.length > maxValueLength) {
    truncatedValue = truncatedValue.substring(0, maxValueLength) + '..'
  }
  
  return { key: truncatedKey, value: truncatedValue }
}

// 智能格式化污点显示
const smartFormatTaint = (taint) => {
  if (!taint) return { key: '', value: '', effect: '' }
  
  // 重要的污点key不截断
  const importantTaintKeys = [
    'nvidia.com/gpu',
    'node-role.kubernetes.io/master',
    'node-role.kubernetes.io/control-plane',
    'node.kubernetes.io/not-ready',
    'node.kubernetes.io/unreachable',
    'node.kubernetes.io/memory-pressure',
    'node.kubernetes.io/disk-pressure',
    'node.kubernetes.io/pid-pressure',
    'node.kubernetes.io/network-unavailable'
  ]
  
  let formattedKey = taint.key
  let formattedValue = taint.value || ''
  let formattedEffect = taint.effect || ''
  
  // 重要污点key保持完整
  if (!importantTaintKeys.includes(taint.key)) {
    // 只有非重要污点才进行截断，且给更大空间
    if (taint.key.length > 25) {
      if (taint.key.includes('/')) {
        const parts = taint.key.split('/')
        if (parts.length === 2) {
          const namespace = parts[0]
          const name = parts[1]
          if (namespace.length > 15) {
            formattedKey = `${namespace.substring(0, 12)}.../${name}`
          }
        }
      } else {
        formattedKey = taint.key.substring(0, 22) + '...'
      }
    }
  }
  
  // 值的处理更宽松
  if (formattedValue.length > 15) {
    formattedValue = formattedValue.substring(0, 12) + '...'
  }
  
  // 效果不截断，显示完整名称
  const effectMap = {
    'NoSchedule': 'NoSchedule',
    'PreferNoSchedule': 'PreferNoSchedule', 
    'NoExecute': 'NoExecute'
  }
  formattedEffect = effectMap[taint.effect] || taint.effect
  
  return { key: formattedKey, value: formattedValue, effect: formattedEffect }
}

// 判断是否应该显示属性区域
const shouldShowAttributes = (node) => {
  return shouldShowLabels(node) || (node.taints && node.taints.length > 0)
}

// 判断是否应该显示标签
const shouldShowLabels = (node) => {
  return node.labels && props.showLabels && Object.keys(getDisplayLabels(node.labels)).length > 0
}

// 清理定时器
onUnmounted(() => {
  if (emitTimer) {
    clearTimeout(emitTimer)
    emitTimer = null
  }
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
    searchDebounceTimer = null
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

/* 确保checkbox-group不影响布局 */
.node-list :deep(.el-checkbox-group) {
  display: block;
  width: 100%;
}

.node-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
}

.node-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  background: #ffffff;
  transition: all 0.2s ease;
  position: relative;
}

.node-item:hover {
  background-color: #f8f9fa;
  border-color: #1890ff;
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.1);
}

.node-item.selected {
  background-color: #e6f7ff;
  border-color: #1890ff;
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.15);
}

.node-checkbox {
  flex-shrink: 0;
  margin-top: 2px;
}

.node-content {
  flex: 1;
  min-width: 0;
}

.node-header {
  margin-bottom: 12px;
}

.node-name {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px 0;
  line-height: 1.4;
}

.node-basic-info {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.status-tag {
  font-weight: 500;
}

.node-roles {
  font-size: 12px;
  color: #666;
  background: #f5f5f5;
  padding: 4px 8px;
  border-radius: 4px;
  white-space: nowrap;
}

.node-ip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #52c41a;
  background: #f6ffed;
  padding: 6px 10px;
  border-radius: 4px;
  border: 1px solid #b7eb8f;
  width: fit-content;
}

.ip-icon {
  font-size: 12px;
  color: #52c41a;
}

.ip-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 500;
  font-size: 12px;
}

.node-attributes {
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.labels-section,
.taints-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #666;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.section-icon {
  font-size: 12px;
}

.section-title {
  font-size: 11px;
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.attribute-tag {
  font-size: 11px;
  font-family: 'Monaco', 'Menlo', monospace;
  border-radius: 4px;
  transition: all 0.2s ease;
  max-width: none;
  word-wrap: break-word;
  white-space: normal;
}

.attribute-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.label-tag .tag-key {
  font-weight: 600;
  color: #1890ff;
}

.tag-separator {
  margin: 0 2px;
  opacity: 0.7;
  color: #666;
}

.label-tag .tag-value {
  font-weight: 500;
  color: #52c41a;
}

.taint-tag .tag-key {
  font-weight: 600;
}

.taint-tag .tag-value {
  font-weight: 500;
  opacity: 0.9;
}

.taint-effect {
  font-weight: 700;
  margin-left: 2px;
  text-transform: uppercase;
}

.more-tag {
  font-weight: 600;
  cursor: pointer;
  background: #f8f9fa;
  border-color: #dee2e6;
  color: #6c757d;
}

.more-tag:hover {
  background: #e6f7ff;
  border-color: #91d5ff;
  color: #1890ff;
}

.arrow-icon {
  font-size: 8px;
  margin-left: 4px;
  transition: transform 0.2s ease;
}

.more-dropdown {
  position: relative;
}

:deep(.el-dropdown-menu) {
  z-index: 9999;
}

:deep(.el-popper) {
  z-index: 9999;
}

.attributes-dropdown {
  min-width: 260px;
  max-width: 400px;
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

.dropdown-body {
  padding: 10px;
  max-height: 200px;
  overflow-y: auto;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.dropdown-tag {
  font-size: 11px;
  font-family: 'Monaco', 'Menlo', monospace;
  border-radius: 4px;
  margin: 0;
  word-break: break-all;
}

.empty-nodes, .loading-nodes {
  padding: 20px;
  text-align: center;
}

/* 新增样式 - 性能优化相关 */
.loading-container {
  padding: 40px 20px;
  text-align: center;
  color: #666;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.loading-container .el-icon {
  font-size: 24px;
  color: #409eff;
}

.empty-container {
  padding: 20px;
  text-align: center;
}

.action-controls {
  display: flex;
  align-items: center;
  gap: 16px;
}

.pagination-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.total-info {
  font-size: 12px;
  color: #666;
}




/* 响应式设计 */
@media (max-width: 768px) {
  .filter-section .el-col {
    margin-bottom: 8px;
  }
  
  .node-container {
    gap: 12px;
    padding: 12px;
  }
  
  .node-item {
    padding: 12px;
    gap: 8px;
  }
  
  .node-name {
    font-size: 14px;
  }
  
  .node-basic-info {
    gap: 8px;
  }
  
  .node-ip {
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
  
  .labels-section,
  .taints-section {
    gap: 6px;
  }
  
  .tags-container {
    gap: 4px;
  }
  
  .attribute-tag {
    font-size: 10px;
    max-width: none;
  }
  
  .arrow-icon {
    font-size: 7px;
    margin-left: 2px;
  }
  
  .attributes-dropdown {
    min-width: 200px;
    max-width: 280px;
  }
  
  .dropdown-body {
    max-height: 150px;
  }
  
  .section-title {
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
  
  .node-container {
    gap: 8px;
    padding: 8px;
  }
  
  .node-item {
    padding: 10px;
    gap: 6px;
  }
  
  .node-name {
    font-size: 13px;
  }
  
  .node-basic-info {
    gap: 6px;
    flex-wrap: wrap;
  }
  
  .node-roles {
    font-size: 11px;
    padding: 2px 6px;
  }
  
  .node-ip {
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
  
  .labels-section,
  .taints-section {
    gap: 4px;
  }
  
  .tags-container {
    gap: 3px;
  }
  
  .attribute-tag {
    font-size: 9px;
    max-width: none;
    word-break: break-all;
  }
  
  .arrow-icon {
    font-size: 6px;
    margin-left: 1px;
  }
  
  .attributes-dropdown {
    min-width: 180px;
    max-width: 240px;
  }
  
  .dropdown-body {
    max-height: 120px;
  }
  
  .dropdown-tag {
    font-size: 10px;
  }
  
  .section-title {
    font-size: 9px;
  }
  
  .section-icon {
    font-size: 10px;
  }
  
  .node-selector {
    font-size: 12px;
  }
}
.advanced-search-toggle {
  margin-top: 12px;
  text-align: center;
}

.advanced-search {
  margin-top: 16px;
  padding: 16px;
  background: #fafafa;
  border-radius: 6px;
  border: 1px solid #e8e8e8;
}

.advanced-search .el-divider {
  margin: 16px 0 12px 0;
}

.advanced-search .el-divider:first-child {
  margin-top: 0;
}

.advanced-search .el-row {
  margin-bottom: 12px;
}

.advanced-search .el-row:last-child {
  margin-bottom: 0;
}
</style>