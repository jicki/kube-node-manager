<template>
  <div class="virtual-table-container">
    <el-table-v2
      :columns="computedColumns"
      :data="filteredData"
      :width="tableWidth"
      :height="tableHeight"
      :row-height="rowHeight"
      :header-height="headerHeight"
      :fixed="true"
      :class="tableClass"
      @row-click="handleRowClick"
    >
      <!-- 自定义单元格插槽 -->
      <template v-for="column in columns" #[`cell-${column.key}`]="{ rowData, column: col }">
        <slot :name="`cell-${column.key}`" :row="rowData" :column="col">
          {{ rowData[column.key] }}
        </slot>
      </template>
    </el-table-v2>
    
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-overlay">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>加载中...</span>
    </div>
    
    <!-- 空数据提示 -->
    <div v-if="!loading && filteredData.length === 0" class="empty-data">
      <el-empty :description="emptyText || '暂无数据'" />
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { Loading } from '@element-plus/icons-vue'

// Props定义
const props = defineProps({
  // 数据源
  data: {
    type: Array,
    required: true,
    default: () => []
  },
  // 列配置
  columns: {
    type: Array,
    required: true,
    default: () => []
  },
  // 表格宽度
  width: {
    type: [Number, String],
    default: '100%'
  },
  // 表格高度
  height: {
    type: [Number, String],
    default: 600
  },
  // 行高
  rowHeight: {
    type: Number,
    default: 50
  },
  // 表头高度
  headerHeight: {
    type: Number,
    default: 50
  },
  // 加载状态
  loading: {
    type: Boolean,
    default: false
  },
  // 空数据文本
  emptyText: {
    type: String,
    default: '暂无数据'
  },
  // 自定义表格类名
  tableClass: {
    type: String,
    default: ''
  },
  // 搜索关键字（用于过滤）
  searchKeyword: {
    type: String,
    default: ''
  },
  // 过滤函数
  filterFn: {
    type: Function,
    default: null
  }
})

// Emits定义
const emit = defineEmits(['row-click', 'selection-change'])

// 计算表格宽度
const tableWidth = computed(() => {
  if (typeof props.width === 'number') {
    return props.width
  }
  if (props.width === '100%') {
    return '100%'
  }
  return parseInt(props.width) || 1200
})

// 计算表格高度
const tableHeight = computed(() => {
  if (typeof props.height === 'number') {
    return props.height
  }
  return parseInt(props.height) || 600
})

// 处理列配置，确保兼容el-table-v2格式
const computedColumns = computed(() => {
  return props.columns.map(col => {
    const column = {
      key: col.key || col.prop,
      dataKey: col.dataKey || col.prop || col.key,
      title: col.title || col.label,
      width: col.width || 150,
      align: col.align || 'left',
      fixed: col.fixed || undefined,
      sortable: col.sortable || false,
    }
    
    // 自定义渲染器
    if (col.cellRenderer) {
      column.cellRenderer = col.cellRenderer
    }
    
    // 表头渲染器
    if (col.headerCellRenderer) {
      column.headerCellRenderer = col.headerCellRenderer
    }
    
    return column
  })
})

// 过滤后的数据
const filteredData = computed(() => {
  let data = props.data || []
  
  // 自定义过滤函数
  if (props.filterFn && typeof props.filterFn === 'function') {
    data = data.filter(props.filterFn)
  }
  
  // 关键字搜索过滤
  if (props.searchKeyword && props.searchKeyword.trim()) {
    const keyword = props.searchKeyword.toLowerCase().trim()
    data = data.filter(item => {
      return Object.values(item).some(val => {
        if (val === null || val === undefined) return false
        return String(val).toLowerCase().includes(keyword)
      })
    })
  }
  
  return data
})

// 行点击事件
const handleRowClick = ({ rowData, rowIndex }) => {
  emit('row-click', rowData, rowIndex)
}

// 监听数据变化
watch(() => props.data, (newData) => {
  console.log(`VirtualTable: Data updated, ${newData?.length || 0} rows`)
}, { immediate: false })
</script>

<style scoped lang="scss">
.virtual-table-container {
  position: relative;
  width: 100%;
  height: 100%;
  
  // 加载遮罩
  .loading-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.8);
    z-index: 1000;
    gap: 10px;
    
    .el-icon {
      font-size: 32px;
      color: var(--el-color-primary);
    }
    
    span {
      font-size: 14px;
      color: var(--el-text-color-secondary);
    }
  }
  
  // 空数据
  .empty-data {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
  }
}

// 自定义滚动条样式
:deep(.el-table-v2__main) {
  &::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }
  
  &::-webkit-scrollbar-track {
    background: #f1f1f1;
    border-radius: 4px;
  }
  
  &::-webkit-scrollbar-thumb {
    background: #888;
    border-radius: 4px;
    
    &:hover {
      background: #555;
    }
  }
}

// 表格单元格样式优化
:deep(.el-table-v2__row-cell) {
  padding: 0 12px;
  font-size: 14px;
  color: var(--el-text-color-primary);
  
  // 单元格内容超出省略
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

// 表头样式
:deep(.el-table-v2__header-cell) {
  padding: 0 12px;
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
}

// 行悬停效果
:deep(.el-table-v2__row:hover) {
  background: var(--el-fill-color-light);
  cursor: pointer;
}
</style>

