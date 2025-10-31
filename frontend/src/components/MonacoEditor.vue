<template>
  <div class="monaco-editor-container">
    <div class="editor-toolbar" v-if="showToolbar">
      <el-space>
        <el-button size="small" @click="formatCode" :disabled="!canFormat">
          <el-icon><DocumentCopy /></el-icon>
          格式化
        </el-button>
        <el-button size="small" @click="handleUndo" :disabled="!canUndo">
          <el-icon><RefreshLeft /></el-icon>
          撤销
        </el-button>
        <el-button size="small" @click="handleRedo" :disabled="!canRedo">
          <el-icon><RefreshRight /></el-icon>
          重做
        </el-button>
        <el-button size="small" @click="toggleFullscreen">
          <el-icon><FullScreen /></el-icon>
          {{ isFullscreen ? '退出全屏' : '全屏' }}
        </el-button>
        <el-text type="info" size="small" v-if="showLineInfo">
          行 {{ lineNumber }} : 列 {{ column }}
        </el-text>
      </el-space>
    </div>
    
    <div 
      ref="editorContainer" 
      class="editor-wrapper"
      :class="{ 'fullscreen': isFullscreen }"
      :style="{ height: isFullscreen ? '100vh' : height }"
    >
      <vue-monaco-editor
        v-model:value="editorValue"
        :language="language"
        :theme="theme"
        :options="editorOptions"
        @mount="handleMount"
        style="height: 100%;"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import { DocumentCopy, RefreshLeft, RefreshRight, FullScreen } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  language: {
    type: String,
    default: 'yaml'
  },
  theme: {
    type: String,
    default: 'vs-dark' // vs, vs-dark, hc-black
  },
  height: {
    type: String,
    default: '500px'
  },
  readonly: {
    type: Boolean,
    default: false
  },
  showToolbar: {
    type: Boolean,
    default: true
  },
  showLineInfo: {
    type: Boolean,
    default: true
  },
  minimap: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['update:modelValue', 'change', 'mounted'])

// 数据
const editorInstance = ref(null)
const editorValue = ref(props.modelValue)
const lineNumber = ref(1)
const column = ref(1)
const canUndo = ref(false)
const canRedo = ref(false)
const canFormat = ref(true)
const isFullscreen = ref(false)
const editorContainer = ref(null)

// 计算属性
const computedHeight = computed(() => {
  return isFullscreen.value ? '100vh' : props.height
})

const editorOptions = computed(() => ({
  automaticLayout: true,
  fontSize: 14,
  tabSize: 2,
  insertSpaces: true,
  wordWrap: 'on',
  minimap: {
    enabled: props.minimap
  },
  scrollBeyondLastLine: false,
  readOnly: props.readonly,
  folding: true,
  lineNumbers: 'on',
  renderWhitespace: 'boundary',
  scrollbar: {
    vertical: 'visible',
    horizontal: 'visible',
    useShadows: false,
    verticalScrollbarSize: 10,
    horizontalScrollbarSize: 10
  },
  suggest: {
    showWords: true,
    showSnippets: true
  },
  quickSuggestions: {
    other: true,
    comments: false,
    strings: true
  }
}))

// 方法
const handleMount = (editor) => {
  editorInstance.value = editor
  
  // 监听光标位置变化
  editor.onDidChangeCursorPosition((e) => {
    lineNumber.value = e.position.lineNumber
    column.value = e.position.column
  })
  
  // 监听编辑器内容变化
  editor.onDidChangeModelContent(() => {
    const model = editor.getModel()
    if (model) {
      canUndo.value = model.canUndo()
      canRedo.value = model.canRedo()
    }
  })
  
  // 添加 Ansible 相关的自动补全（如果是 YAML）
  if (props.language === 'yaml') {
    setupAnsibleCompletion(editor)
  }
  
  emit('mounted', editor)
}

// 监听编辑器内容变化
watch(editorValue, (newValue) => {
  emit('update:modelValue', newValue)
  emit('change', newValue)
})

const formatCode = () => {
  if (!editorInstance.value) return
  
  try {
    editorInstance.value.getAction('editor.action.formatDocument').run()
    ElMessage.success('代码已格式化')
  } catch (error) {
    console.error('格式化失败:', error)
    ElMessage.error('格式化失败')
  }
}

const handleUndo = () => {
  if (!editorInstance.value) return
  editorInstance.value.trigger('keyboard', 'undo')
}

const handleRedo = () => {
  if (!editorInstance.value) return
  editorInstance.value.trigger('keyboard', 'redo')
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  
  if (isFullscreen.value) {
    editorContainer.value?.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
}

const setupAnsibleCompletion = (editor) => {
  // Ansible 模块建议列表
  const ansibleModules = [
    'ping', 'copy', 'file', 'template', 'service', 'systemd',
    'command', 'shell', 'script', 'raw',
    'apt', 'yum', 'dnf', 'package',
    'user', 'group',
    'git', 'get_url',
    'debug', 'set_fact', 'include_vars',
    'lineinfile', 'blockinfile', 'replace',
    'cron', 'at',
    'docker_container', 'docker_image',
    'mysql_db', 'mysql_user',
    'postgresql_db', 'postgresql_user'
  ]
  
  // Ansible 关键字
  const ansibleKeywords = [
    'hosts', 'tasks', 'name', 'become', 'become_user', 'vars',
    'handlers', 'notify', 'when', 'with_items', 'loop',
    'register', 'changed_when', 'failed_when',
    'tags', 'block', 'rescue', 'always',
    'gather_facts', 'connection', 'remote_user'
  ]
  
  // 注册自动补全提供者
  const monaco = window.monaco
  if (monaco) {
    monaco.languages.registerCompletionItemProvider('yaml', {
      provideCompletionItems: (model, position) => {
        const suggestions = []
        
        // 添加模块建议
        ansibleModules.forEach(module => {
          suggestions.push({
            label: module,
            kind: monaco.languages.CompletionItemKind.Module,
            insertText: `${module}:`,
            documentation: `Ansible module: ${module}`
          })
        })
        
        // 添加关键字建议
        ansibleKeywords.forEach(keyword => {
          suggestions.push({
            label: keyword,
            kind: monaco.languages.CompletionItemKind.Keyword,
            insertText: `${keyword}: `,
            documentation: `Ansible keyword: ${keyword}`
          })
        })
        
        return { suggestions }
      }
    })
  }
}

// 监听 modelValue 变化
watch(() => props.modelValue, (newValue) => {
  if (newValue !== editorValue.value) {
    editorValue.value = newValue
  }
})

// 监听全屏状态变化
const handleFullscreenChange = () => {
  if (!document.fullscreenElement) {
    isFullscreen.value = false
  }
}

// 生命周期
onMounted(() => {
  document.addEventListener('fullscreenchange', handleFullscreenChange)
})

onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
})

// 暴露方法给父组件
defineExpose({
  getEditor: () => editorInstance.value,
  getValue: () => editorValue.value,
  setValue: (value) => {
    editorValue.value = value
  },
  format: formatCode,
  undo: handleUndo,
  redo: handleRedo
})
</script>

<style scoped>
.monaco-editor-container {
  display: flex;
  flex-direction: column;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  overflow: hidden;
}

.editor-toolbar {
  padding: 8px 12px;
  background: #f5f7fa;
  border-bottom: 1px solid #dcdfe6;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.editor-wrapper {
  flex: 1;
  overflow: hidden;
  position: relative;
  min-height: 300px;
}

.editor-wrapper > div {
  height: 100%;
}

.editor-wrapper.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
  background: #1e1e1e;
}

:deep(.monaco-editor) {
  width: 100%;
  height: 100%;
}

/* 自定义滚动条 */
:deep(.monaco-scrollable-element > .scrollbar) {
  background: transparent;
}

:deep(.monaco-scrollable-element > .scrollbar > .slider) {
  background: rgba(100, 100, 100, 0.4);
}

:deep(.monaco-scrollable-element > .scrollbar > .slider:hover) {
  background: rgba(100, 100, 100, 0.6);
}
</style>

