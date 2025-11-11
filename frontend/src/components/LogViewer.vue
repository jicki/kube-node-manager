<template>
  <div class="log-viewer">
    <div class="log-toolbar">
      <el-space wrap>
        <el-input
          v-model="searchKeyword"
          placeholder="搜索日志内容"
          clearable
          style="width: 300px"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        
        <el-select
          v-model="filterLevel"
          placeholder="日志级别"
          clearable
          style="width: 150px"
          @change="applyFilter"
        >
          <el-option label="全部" value="" />
          <el-option label="Info" value="info" />
          <el-option label="Warning" value="warning" />
          <el-option label="Error" value="error" />
          <el-option label="Debug" value="debug" />
        </el-select>
        
        <el-checkbox v-model="autoScroll" @change="toggleAutoScroll">
          自动滚动
        </el-checkbox>
        
        <el-checkbox v-model="showTimestamp">
          显示时间戳
        </el-checkbox>
        
        <el-checkbox v-model="showLineNumbers">
          显示行号
        </el-checkbox>
        
        <el-button @click="handleCopy" :loading="copying">
          <el-icon><DocumentCopy /></el-icon>
          复制日志
        </el-button>
        
        <el-button @click="handleDownload">
          <el-icon><Download /></el-icon>
          下载日志
        </el-button>
        
        <el-button @click="handleClear" type="danger">
          <el-icon><Delete /></el-icon>
          清空
        </el-button>
      </el-space>
    </div>
    
    <div 
      ref="logContainerRef" 
      class="log-container" 
      :class="{ 'line-numbers': showLineNumbers }"
      @scroll="handleScroll"
    >
      <div 
        v-for="(line, index) in filteredLines" 
        :key="index" 
        class="log-line"
        :class="getLineClass(line)"
      >
        <span v-if="showLineNumbers" class="line-number">{{ index + 1 }}</span>
        <span v-if="showTimestamp && line.timestamp" class="timestamp">{{ line.timestamp }}</span>
        <span class="line-content" v-html="highlightText(line.content)"></span>
      </div>
      <div v-if="filteredLines.length === 0" class="empty-log">
        <el-empty description="暂无日志" />
      </div>
    </div>
    
    <div class="log-footer">
      <el-text size="small" type="info">
        总计 {{ totalLines }} 行
        <span v-if="searchKeyword"> | 匹配 {{ filteredLines.length }} 行</span>
      </el-text>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, DocumentCopy, Download, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  logs: {
    type: String,
    default: ''
  },
  realtime: {
    type: Boolean,
    default: false
  },
  maxLines: {
    type: Number,
    default: 10000
  }
})

const emit = defineEmits(['clear'])

// 数据
const searchKeyword = ref('')
const filterLevel = ref('')
const autoScroll = ref(true)
const showTimestamp = ref(false)
const showLineNumbers = ref(true)
const copying = ref(false)
const logContainerRef = ref(null)
const isUserScrolling = ref(false)
const scrollTimeout = ref(null)

// 解析日志行
const parseLogLines = (logText) => {
  if (!logText) return []
  
  const lines = logText.split('\n')
  return lines.map(line => {
    const parsed = {
      content: line,
      timestamp: null,
      level: 'info'
    }
    
    // 尝试解析时间戳 (ISO 8601 格式)
    const timestampMatch = line.match(/(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d{3})?(?:Z|[+-]\d{2}:\d{2})?)/)
    if (timestampMatch) {
      parsed.timestamp = timestampMatch[1]
    }
    
    // 检测日志级别
    const lowerLine = line.toLowerCase()
    if (lowerLine.includes('error') || lowerLine.includes('err:') || lowerLine.includes('failed')) {
      parsed.level = 'error'
    } else if (lowerLine.includes('warn') || lowerLine.includes('warning')) {
      parsed.level = 'warning'
    } else if (lowerLine.includes('debug')) {
      parsed.level = 'debug'
    }
    
    return parsed
  })
  // 不过滤空行，保留所有行以便完整显示日志
  // }).filter(line => line.content.trim() !== '')
}

// 计算属性
const logLines = computed(() => {
  const lines = parseLogLines(props.logs)
  // 限制最大行数，防止性能问题
  if (lines.length > props.maxLines) {
    return lines.slice(-props.maxLines)
  }
  return lines
})

const totalLines = computed(() => logLines.value.length)

const filteredLines = computed(() => {
  let lines = logLines.value
  
  // 按级别过滤
  if (filterLevel.value) {
    lines = lines.filter(line => line.level === filterLevel.value)
  }
  
  // 按关键字搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    lines = lines.filter(line => 
      line.content.toLowerCase().includes(keyword)
    )
  }
  
  return lines
})

// 方法
const getLineClass = (line) => {
  return `level-${line.level}`
}

const highlightText = (text) => {
  if (!searchKeyword.value) {
    return escapeHtml(text)
  }
  
  const escapedText = escapeHtml(text)
  const regex = new RegExp(`(${escapeRegex(searchKeyword.value)})`, 'gi')
  return escapedText.replace(regex, '<mark>$1</mark>')
}

const escapeHtml = (text) => {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

const escapeRegex = (str) => {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

const handleSearch = () => {
  // 搜索时自动滚动到第一个匹配项
  if (searchKeyword.value && filteredLines.value.length > 0) {
    nextTick(() => {
      const firstMatch = logContainerRef.value?.querySelector('.log-line')
      if (firstMatch) {
        firstMatch.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }
    })
  }
}

const applyFilter = () => {
  // 过滤后滚动到顶部
  if (logContainerRef.value) {
    logContainerRef.value.scrollTop = 0
  }
}

const scrollToBottom = () => {
  if (autoScroll.value && !isUserScrolling.value && logContainerRef.value) {
    nextTick(() => {
      logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
    })
  }
}

const toggleAutoScroll = () => {
  if (autoScroll.value) {
    scrollToBottom()
  }
}

const handleScroll = () => {
  if (!logContainerRef.value) return
  
  // 检测用户是否主动滚动
  const { scrollTop, scrollHeight, clientHeight } = logContainerRef.value
  const isAtBottom = Math.abs(scrollHeight - clientHeight - scrollTop) < 10
  
  // 如果用户滚动到底部，恢复自动滚动
  if (isAtBottom) {
    isUserScrolling.value = false
    autoScroll.value = true
  } else {
    // 用户主动滚动时暂停自动滚动
    isUserScrolling.value = true
    autoScroll.value = false
    
    // 清除之前的定时器
    if (scrollTimeout.value) {
      clearTimeout(scrollTimeout.value)
    }
    
    // 5秒后恢复自动滚动
    scrollTimeout.value = setTimeout(() => {
      isUserScrolling.value = false
    }, 5000)
  }
}

const handleCopy = async () => {
  copying.value = true
  try {
    const textToCopy = filteredLines.value.map(line => line.content).join('\n')
    await navigator.clipboard.writeText(textToCopy)
    ElMessage.success('日志已复制到剪贴板')
  } catch (error) {
    console.error('复制日志失败:', error)
    // 降级方案
    const textArea = document.createElement('textarea')
    textArea.value = filteredLines.value.map(line => line.content).join('\n')
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      ElMessage.success('日志已复制到剪贴板')
    } catch (err) {
      ElMessage.error('复制失败，请手动复制')
    }
    document.body.removeChild(textArea)
  } finally {
    copying.value = false
  }
}

const handleDownload = () => {
  const content = filteredLines.value.map(line => line.content).join('\n')
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `ansible-log-${new Date().getTime()}.txt`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  ElMessage.success('日志已下载')
}

const handleClear = () => {
  emit('clear')
}

// 监听日志变化，自动滚动
watch(() => props.logs, () => {
  if (props.realtime) {
    scrollToBottom()
  }
}, { immediate: true })

// 生命周期
onMounted(() => {
  scrollToBottom()
})

onBeforeUnmount(() => {
  if (scrollTimeout.value) {
    clearTimeout(scrollTimeout.value)
  }
})
</script>

<style scoped>
.log-viewer {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1e1e1e;
  border-radius: 4px;
  overflow: hidden;
}

.log-toolbar {
  padding: 12px;
  background: #2d2d2d;
  border-bottom: 1px solid #3e3e3e;
}

.log-container {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #d4d4d4;
  background: #1e1e1e;
}

.log-container.line-numbers {
  padding-left: 0;
}

.log-line {
  display: flex;
  padding: 2px 0;
  word-wrap: break-word;
  white-space: pre-wrap;
}

.log-line:hover {
  background: #2d2d2d;
}

.line-number {
  display: inline-block;
  min-width: 50px;
  padding-right: 12px;
  padding-left: 12px;
  text-align: right;
  color: #858585;
  user-select: none;
  flex-shrink: 0;
}

.timestamp {
  display: inline-block;
  margin-right: 8px;
  color: #608b4e;
  flex-shrink: 0;
}

.line-content {
  flex: 1;
  word-break: break-all;
}

.log-line.level-error {
  color: #f48771;
}

.log-line.level-warning {
  color: #dcdcaa;
}

.log-line.level-debug {
  color: #858585;
}

.log-line.level-info {
  color: #d4d4d4;
}

.empty-log {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.log-footer {
  padding: 8px 12px;
  background: #2d2d2d;
  border-top: 1px solid #3e3e3e;
  text-align: right;
}

/* 高亮搜索结果 */
:deep(mark) {
  background-color: #f59e0b;
  color: #000;
  padding: 0 2px;
  border-radius: 2px;
}

/* 滚动条样式 */
.log-container::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

.log-container::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.log-container::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 5px;
}

.log-container::-webkit-scrollbar-thumb:hover {
  background: #4e4e4e;
}
</style>

