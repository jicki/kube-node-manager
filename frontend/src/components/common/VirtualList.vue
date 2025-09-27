<template>
  <div 
    ref="containerRef"
    class="virtual-list"
    :style="{ height: containerHeight + 'px' }"
    @scroll="handleScroll"
  >
    <!-- 占位空间，用于保持滚动条正确 -->
    <div class="virtual-spacer" :style="{ height: totalHeight + 'px' }">
      <!-- 可见区域的项目 -->
      <div
        class="virtual-content"
        :style="{ 
          transform: `translateY(${offsetY}px)`,
          height: visibleHeight + 'px'
        }"
      >
        <div
          v-for="(item, index) in visibleItems"
          :key="getItemKey(item, visibleStartIndex + index)"
          class="virtual-item"
          :style="{ height: itemHeight + 'px' }"
        >
          <slot :item="item" :index="visibleStartIndex + index"></slot>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'

const props = defineProps({
  // 数据项数组
  items: {
    type: Array,
    required: true,
    default: () => []
  },
  // 每项的高度
  itemHeight: {
    type: Number,
    default: 50
  },
  // 容器高度
  containerHeight: {
    type: Number,
    default: 300
  },
  // 缓冲区大小（渲染可见区域外的项目数量）
  bufferSize: {
    type: Number,
    default: 5
  },
  // 获取项目的唯一键
  keyField: {
    type: String,
    default: 'id'
  },
  // 是否启用平滑滚动
  smoothScroll: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['scroll', 'reach-bottom', 'reach-top'])

// 响应式数据
const containerRef = ref(null)
const scrollTop = ref(0)
const isScrolling = ref(false)
const scrollingTimer = ref(null)

// 计算属性
const totalHeight = computed(() => {
  return props.items.length * props.itemHeight
})

const visibleCount = computed(() => {
  return Math.ceil(props.containerHeight / props.itemHeight)
})

const visibleStartIndex = computed(() => {
  return Math.max(0, Math.floor(scrollTop.value / props.itemHeight) - props.bufferSize)
})

const visibleEndIndex = computed(() => {
  return Math.min(
    props.items.length - 1,
    visibleStartIndex.value + visibleCount.value + props.bufferSize * 2
  )
})

const visibleItems = computed(() => {
  return props.items.slice(visibleStartIndex.value, visibleEndIndex.value + 1)
})

const offsetY = computed(() => {
  return visibleStartIndex.value * props.itemHeight
})

const visibleHeight = computed(() => {
  return visibleItems.value.length * props.itemHeight
})

// 滚动处理
const handleScroll = (event) => {
  const target = event.target
  scrollTop.value = target.scrollTop

  // 设置滚动状态
  isScrolling.value = true
  clearTimeout(scrollingTimer.value)
  scrollingTimer.value = setTimeout(() => {
    isScrolling.value = false
  }, 150)

  // 发出滚动事件
  emit('scroll', {
    scrollTop: scrollTop.value,
    scrollLeft: target.scrollLeft,
    isScrolling: isScrolling.value
  })

  // 检查是否到达顶部或底部
  checkScrollBoundaries(target)
}

// 检查滚动边界
const checkScrollBoundaries = (target) => {
  const { scrollTop, scrollHeight, clientHeight } = target
  
  // 到达顶部
  if (scrollTop <= 0) {
    emit('reach-top')
  }
  
  // 到达底部（允许10px的误差）
  if (scrollTop + clientHeight >= scrollHeight - 10) {
    emit('reach-bottom')
  }
}

// 获取项目的唯一键
const getItemKey = (item, index) => {
  if (typeof item === 'object' && item !== null) {
    return item[props.keyField] || index
  }
  return index
}

// 滚动到指定索引
const scrollToIndex = (index, behavior = 'smooth') => {
  if (!containerRef.value || index < 0 || index >= props.items.length) {
    return
  }

  const targetScrollTop = index * props.itemHeight
  
  if (props.smoothScroll && behavior === 'smooth') {
    containerRef.value.scrollTo({
      top: targetScrollTop,
      behavior: 'smooth'
    })
  } else {
    containerRef.value.scrollTop = targetScrollTop
  }
}

// 滚动到指定项目
const scrollToItem = (item, behavior = 'smooth') => {
  const index = props.items.findIndex(i => 
    typeof item === 'object' 
      ? i[props.keyField] === item[props.keyField]
      : i === item
  )
  
  if (index !== -1) {
    scrollToIndex(index, behavior)
  }
}

// 滚动到顶部
const scrollToTop = (behavior = 'smooth') => {
  scrollToIndex(0, behavior)
}

// 滚动到底部
const scrollToBottom = (behavior = 'smooth') => {
  scrollToIndex(props.items.length - 1, behavior)
}

// 获取可见区域的项目
const getVisibleRange = () => {
  return {
    startIndex: visibleStartIndex.value,
    endIndex: visibleEndIndex.value,
    items: visibleItems.value
  }
}

// 监听数据变化，重新计算滚动位置
watch(() => props.items.length, (newLength, oldLength) => {
  // 如果数据长度变化，可能需要调整滚动位置
  if (newLength < oldLength && containerRef.value) {
    const maxScrollTop = Math.max(0, newLength * props.itemHeight - props.containerHeight)
    if (scrollTop.value > maxScrollTop) {
      containerRef.value.scrollTop = maxScrollTop
    }
  }
})

// 组件挂载时初始化
onMounted(() => {
  // 确保滚动位置正确
  nextTick(() => {
    if (containerRef.value) {
      containerRef.value.scrollTop = 0
    }
  })
})

// 清理定时器
onUnmounted(() => {
  if (scrollingTimer.value) {
    clearTimeout(scrollingTimer.value)
  }
})

// 暴露方法给父组件
defineExpose({
  scrollToIndex,
  scrollToItem,
  scrollToTop,
  scrollToBottom,
  getVisibleRange,
  container: containerRef
})
</script>

<style scoped>
.virtual-list {
  overflow: auto;
  position: relative;
  width: 100%;
}

.virtual-spacer {
  position: relative;
  width: 100%;
}

.virtual-content {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  will-change: transform;
}

.virtual-item {
  width: 100%;
  position: relative;
  display: flex;
  align-items: center;
  box-sizing: border-box;
}

/* 自定义滚动条样式 */
.virtual-list::-webkit-scrollbar {
  width: 6px;
}

.virtual-list::-webkit-scrollbar-track {
  background: #f5f5f5;
  border-radius: 3px;
}

.virtual-list::-webkit-scrollbar-thumb {
  background: #d9d9d9;
  border-radius: 3px;
  transition: background-color 0.2s;
}

.virtual-list::-webkit-scrollbar-thumb:hover {
  background: #bfbfbf;
}

/* 滚动中的样式 */
.virtual-list.scrolling {
  pointer-events: none;
}

.virtual-list.scrolling .virtual-item {
  opacity: 0.8;
}

/* 优化渲染性能 */
.virtual-content {
  contain: layout style paint;
}

.virtual-item {
  contain: layout style paint size;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .virtual-list::-webkit-scrollbar {
    width: 4px;
  }
  
  .virtual-item {
    font-size: 14px;
  }
}

/* 无障碍支持 */
.virtual-list:focus {
  outline: 2px solid #1890ff;
  outline-offset: 2px;
}

/* 性能优化提示 */
.virtual-list {
  /* 使用 GPU 加速 */
  transform: translateZ(0);
  /* 优化滚动性能 */
  -webkit-overflow-scrolling: touch;
  /* 减少重排 */
  contain: layout style paint;
}
</style>
