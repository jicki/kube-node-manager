<template>
  <div class="search-box" :class="{ 'expanded': expanded }">
    <div class="search-input-container">
      <el-input
        v-model="searchValue"
        :placeholder="placeholder"
        :size="size"
        :clearable="clearable"
        :disabled="disabled"
        class="search-input"
        @input="handleInput"
        @clear="handleClear"
        @focus="handleFocus"
        @blur="handleBlur"
        @keyup.enter="handleSearch"
      >
        <template #prefix>
          <el-icon class="search-icon">
            <Search />
          </el-icon>
        </template>
        <template #suffix>
          <el-button
            v-if="showSearchButton"
            type="primary"
            :size="size"
            :loading="loading"
            @click="handleSearch"
          >
            搜索
          </el-button>
        </template>
      </el-input>
    </div>
    
    <!-- 高级搜索 -->
    <div v-if="advancedSearch" class="advanced-search">
      <el-button
        type="text"
        :icon="expanded ? ArrowUp : ArrowDown"
        @click="toggleExpanded"
      >
        高级搜索
      </el-button>
      
      <div v-show="expanded" class="advanced-filters">
        <el-row :gutter="16">
          <el-col
            v-for="filter in filters"
            :key="filter.key"
            :span="filter.span || 8"
          >
            <div class="filter-item">
              <label class="filter-label">{{ filter.label }}:</label>
              
              <!-- 输入框 -->
              <el-input
                v-if="filter.type === 'input'"
                v-model="filterValues[filter.key]"
                :placeholder="filter.placeholder"
                :size="size"
                @input="handleFilterChange"
              />
              
              <!-- 选择器 -->
              <el-select
                v-else-if="filter.type === 'select'"
                v-model="filterValues[filter.key]"
                :placeholder="filter.placeholder"
                :size="size"
                :multiple="filter.multiple"
                :clearable="true"
                @change="handleFilterChange"
              >
                <el-option
                  v-for="option in filter.options"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
              
              <!-- 日期选择 -->
              <el-date-picker
                v-else-if="filter.type === 'date'"
                v-model="filterValues[filter.key]"
                :type="filter.dateType || 'date'"
                :placeholder="filter.placeholder"
                :size="size"
                @change="handleFilterChange"
              />
              
              <!-- 日期范围选择 -->
              <el-date-picker
                v-else-if="filter.type === 'daterange'"
                v-model="filterValues[filter.key]"
                type="daterange"
                range-separator="至"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                :size="size"
                @change="handleFilterChange"
              />
            </div>
          </el-col>
        </el-row>
        
        <div class="filter-actions">
          <el-button :size="size" @click="handleReset">
            重置
          </el-button>
          <el-button
            type="primary"
            :size="size"
            :loading="loading"
            @click="handleAdvancedSearch"
          >
            搜索
          </el-button>
        </div>
      </div>
    </div>
    
    <!-- 搜索历史 -->
    <div v-if="showHistory && searchHistory.length > 0" class="search-history">
      <div class="history-header">
        <span>搜索历史</span>
        <el-button type="text" size="small" @click="clearHistory">
          清空
        </el-button>
      </div>
      <div class="history-items">
        <el-tag
          v-for="(item, index) in searchHistory"
          :key="index"
          :closable="true"
          @click="handleHistoryClick(item)"
          @close="removeHistoryItem(index)"
        >
          {{ item }}
        </el-tag>
      </div>
    </div>
    
    <!-- 搜索建议 -->
    <div v-if="showSuggestions && suggestions.length > 0" class="search-suggestions">
      <div
        v-for="(suggestion, index) in suggestions"
        :key="index"
        class="suggestion-item"
        @click="handleSuggestionClick(suggestion)"
      >
        <el-icon class="suggestion-icon"><Search /></el-icon>
        <span class="suggestion-text" v-html="highlightKeyword(suggestion, searchValue)"></span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { Search, ArrowUp, ArrowDown } from '@element-plus/icons-vue'
import { highlightKeyword } from '@/utils/format'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  placeholder: {
    type: String,
    default: '请输入搜索关键字'
  },
  size: {
    type: String,
    default: 'default',
    validator: (value) => ['large', 'default', 'small'].includes(value)
  },
  clearable: {
    type: Boolean,
    default: true
  },
  disabled: {
    type: Boolean,
    default: false
  },
  loading: {
    type: Boolean,
    default: false
  },
  showSearchButton: {
    type: Boolean,
    default: false
  },
  // 高级搜索
  advancedSearch: {
    type: Boolean,
    default: false
  },
  filters: {
    type: Array,
    default: () => []
  },
  // 搜索历史
  showHistory: {
    type: Boolean,
    default: false
  },
  maxHistory: {
    type: Number,
    default: 10
  },
  // 搜索建议
  showSuggestions: {
    type: Boolean,
    default: false
  },
  suggestions: {
    type: Array,
    default: () => []
  },
  // 实时搜索
  realtime: {
    type: Boolean,
    default: false
  },
  debounce: {
    type: Number,
    default: 300
  }
})

const emit = defineEmits([
  'update:modelValue',
  'search',
  'clear',
  'filter-change',
  'focus',
  'blur'
])

// 响应式数据
const searchValue = ref('')
const expanded = ref(false)
const filterValues = ref({})
const searchHistory = ref([])
const debounceTimer = ref(null)

// 计算属性
const historyKey = computed(() => `search_history_${window.location.pathname}`)

// 监听modelValue变化
watch(() => props.modelValue, (newValue) => {
  searchValue.value = newValue
}, { immediate: true })

watch(searchValue, (newValue) => {
  emit('update:modelValue', newValue)
  
  if (props.realtime) {
    handleRealtimeSearch()
  }
})

// 初始化过滤器值
watch(() => props.filters, (newFilters) => {
  const newFilterValues = {}
  newFilters.forEach(filter => {
    if (filterValues.value[filter.key] === undefined) {
      newFilterValues[filter.key] = filter.multiple ? [] : ''
    } else {
      newFilterValues[filter.key] = filterValues.value[filter.key]
    }
  })
  filterValues.value = newFilterValues
}, { immediate: true, deep: true })

// 方法
const handleInput = (value) => {
  // 由watch处理
}

const handleClear = () => {
  emit('clear')
}

const handleFocus = () => {
  emit('focus')
}

const handleBlur = () => {
  emit('blur')
}

const handleSearch = () => {
  const keyword = searchValue.value.trim()
  if (!keyword) return
  
  // 添加到搜索历史
  if (props.showHistory) {
    addToHistory(keyword)
  }
  
  emit('search', {
    keyword,
    filters: { ...filterValues.value }
  })
}

const handleAdvancedSearch = () => {
  const keyword = searchValue.value.trim()
  
  emit('search', {
    keyword,
    filters: { ...filterValues.value }
  })
}

const handleRealtimeSearch = () => {
  if (debounceTimer.value) {
    clearTimeout(debounceTimer.value)
  }
  
  debounceTimer.value = setTimeout(() => {
    const keyword = searchValue.value.trim()
    emit('search', {
      keyword,
      filters: { ...filterValues.value }
    })
  }, props.debounce)
}

const toggleExpanded = () => {
  expanded.value = !expanded.value
}

const handleFilterChange = () => {
  emit('filter-change', { ...filterValues.value })
}

const handleReset = () => {
  searchValue.value = ''
  
  // 重置所有过滤器
  Object.keys(filterValues.value).forEach(key => {
    const filter = props.filters.find(f => f.key === key)
    filterValues.value[key] = filter?.multiple ? [] : ''
  })
  
  emit('filter-change', { ...filterValues.value })
  
  // 触发搜索
  if (props.realtime) {
    handleRealtimeSearch()
  }
}

// 搜索历史相关
const loadHistory = () => {
  const saved = localStorage.getItem(historyKey.value)
  if (saved) {
    searchHistory.value = JSON.parse(saved)
  }
}

const saveHistory = () => {
  localStorage.setItem(historyKey.value, JSON.stringify(searchHistory.value))
}

const addToHistory = (keyword) => {
  // 移除重复项
  const index = searchHistory.value.indexOf(keyword)
  if (index !== -1) {
    searchHistory.value.splice(index, 1)
  }
  
  // 添加到开头
  searchHistory.value.unshift(keyword)
  
  // 限制数量
  if (searchHistory.value.length > props.maxHistory) {
    searchHistory.value = searchHistory.value.slice(0, props.maxHistory)
  }
  
  saveHistory()
}

const handleHistoryClick = (keyword) => {
  searchValue.value = keyword
  handleSearch()
}

const removeHistoryItem = (index) => {
  searchHistory.value.splice(index, 1)
  saveHistory()
}

const clearHistory = () => {
  searchHistory.value = []
  localStorage.removeItem(historyKey.value)
}

// 搜索建议相关
const handleSuggestionClick = (suggestion) => {
  searchValue.value = suggestion
  handleSearch()
}

onMounted(() => {
  if (props.showHistory) {
    loadHistory()
  }
})
</script>

<style scoped>
.search-box {
  width: 100%;
}

.search-input-container {
  width: 100%;
}

.search-input {
  width: 100%;
}

.search-icon {
  color: #999;
}

.advanced-search {
  margin-top: 12px;
}

.advanced-filters {
  margin-top: 16px;
  padding: 16px;
  background: #fafafa;
  border-radius: 4px;
  border: 1px solid #e8e8e8;
}

.filter-item {
  margin-bottom: 16px;
}

.filter-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: #666;
  font-weight: 500;
}

.filter-actions {
  text-align: right;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e8e8e8;
}

.search-history {
  margin-top: 12px;
  padding: 12px;
  background: #f9f9f9;
  border-radius: 4px;
  border: 1px solid #e8e8e8;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 13px;
  color: #666;
}

.history-items {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.history-items .el-tag {
  cursor: pointer;
}

.search-suggestions {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: #fff;
  border: 1px solid #e8e8e8;
  border-top: none;
  border-radius: 0 0 4px 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  z-index: 1000;
  max-height: 200px;
  overflow-y: auto;
}

.suggestion-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  transition: background-color 0.3s;
}

.suggestion-item:hover {
  background-color: #f5f5f5;
}

.suggestion-icon {
  margin-right: 8px;
  color: #999;
  font-size: 14px;
}

.suggestion-text {
  flex: 1;
  font-size: 14px;
}

.suggestion-text :deep(mark) {
  background: #fff3cd;
  color: #856404;
  padding: 0 2px;
  border-radius: 2px;
}

/* 展开状态样式 */
.search-box.expanded {
  position: relative;
}
</style>