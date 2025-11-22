<template>
  <el-dialog
    v-model="dialogVisible"
    :title="`Web Terminal - ${nodeName} (${clusterName})`"
    width="80%"
    :close-on-click-modal="false"
    @opened="initTerminal"
    @closed="closeTerminal"
    class="terminal-dialog"
    top="5vh"
  >
    <div class="terminal-toolbar">
      <div class="status-indicator">
        <div class="status-dot" :class="connectionStatus"></div>
        <span>{{ connectionStatusText }}</span>
      </div>
      <el-button size="small" @click="showConfig = true">
        <el-icon><Setting /></el-icon> SSH配置
      </el-button>
    </div>
    
    <div ref="terminalContainer" class="terminal-container"></div>
    
    <!-- SSH Config Dialog -->
    <el-dialog
      v-model="showConfig"
      title="SSH连接配置"
      width="500px"
      append-to-body
    >
      <el-form :model="sshConfig" label-width="100px">
        <el-form-item label="SSH端口">
          <el-input-number v-model="sshConfig.ssh_port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="SSH用户">
          <el-input v-model="sshConfig.ssh_user" placeholder="默认为System Key用户" />
        </el-form-item>
        <el-form-item label="使用密钥">
           <el-select v-model="sshConfig.system_ssh_key_id" placeholder="默认使用系统默认密钥" clearable style="width: 100%">
              <el-option 
                v-for="key in sshKeys" 
                :key="key.id" 
                :label="key.name" 
                :value="key.id" 
              />
           </el-select>
           <div class="help-text">如果不选择，将使用标记为"默认"的系统密钥</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showConfig = false">取消</el-button>
        <el-button type="primary" @click="saveConfig">保存并重连</el-button>
      </template>
    </el-dialog>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'
import { Setting } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import axios from '@/utils/request' // Correct import path

const props = defineProps({
  modelValue: Boolean,
  clusterName: String,
  nodeName: String
})

const emit = defineEmits(['update:modelValue'])

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const terminalContainer = ref(null)
const showConfig = ref(false)
const sshConfig = ref({
  ssh_port: 22,
  ssh_user: '',
  system_ssh_key_id: null
})
const sshKeys = ref([])

const connectionStatus = ref('disconnected') // disconnected, connecting, connected, error
const connectionStatusText = computed(() => {
  switch(connectionStatus.value) {
    case 'disconnected': return '未连接'
    case 'connecting': return '连接中...'
    case 'connected': return '已连接'
    case 'error': return '连接错误'
    default: return '未知'
  }
})

let term = null
let socket = null
let fitAddon = null
let resizeObserver = null

// 获取SSH配置和密钥列表
const fetchConfig = async () => {
  try {
    // 获取SSH Keys
    const keysRes = await axios.get('/api/v1/ssh-keys')
    sshKeys.value = keysRes.data.data || []

    // 获取当前节点配置
    const configRes = await axios.get(`/api/v1/nodes/ssh-config/${props.nodeName}?cluster_name=${props.clusterName}`)
    if (configRes.data.data) {
      const data = configRes.data.data
      sshConfig.value = {
        ssh_port: data.ssh_port || 22,
        ssh_user: data.ssh_user || '',
        system_ssh_key_id: data.system_ssh_key_id || null
      }
    }
  } catch (err) {
    console.error('Failed to load config:', err)
    ElMessage.error('加载配置失败')
  }
}

const saveConfig = async () => {
  try {
    await axios.put(`/api/v1/nodes/ssh-config/${props.nodeName}?cluster_name=${props.clusterName}`, sshConfig.value)
    ElMessage.success('配置保存成功')
    showConfig.value = false
    // Reconnect
    closeTerminal()
    setTimeout(() => initTerminal(), 500)
  } catch (err) {
    ElMessage.error('保存配置失败: ' + err.message)
  }
}

const initTerminal = async () => {
  await fetchConfig()

  if (term) {
    term.dispose()
  }

  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#f0f0f0'
    }
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.loadAddon(new WebLinksAddon())

  term.open(terminalContainer.value)
  fitAddon.fit()

  // 监听 Resize
  resizeObserver = new ResizeObserver(() => {
      if (fitAddon) {
          fitAddon.fit()
          if (socket && socket.readyState === WebSocket.OPEN) {
              socket.send(JSON.stringify({
                  type: 'resize',
                  cols: term.cols,
                  rows: term.rows
              }))
          }
      }
  })
  resizeObserver.observe(terminalContainer.value)

  connectWebSocket()
}

const connectWebSocket = () => {
  connectionStatus.value = 'connecting'
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const wsUrl = `${protocol}//${host}/api/v1/terminal/ws?cluster_name=${props.clusterName}&node_name=${props.nodeName}`

  socket = new WebSocket(wsUrl)

  socket.onopen = () => {
    connectionStatus.value = 'connected'
    term.write('\r\n\x1b[32mConnected to server...\x1b[0m\r\n')
    // Send initial resize
    socket.send(JSON.stringify({
        type: 'resize',
        cols: term.cols,
        rows: term.rows
    }))
  }

  socket.onmessage = (event) => {
    term.write(event.data)
  }

  socket.onclose = () => {
    connectionStatus.value = 'disconnected'
    term.write('\r\n\x1b[31mConnection closed.\x1b[0m\r\n')
  }

  socket.onerror = () => {
    connectionStatus.value = 'error'
    term.write('\r\n\x1b[31mConnection error.\x1b[0m\r\n')
  }

  term.onData(data => {
    if (socket && socket.readyState === WebSocket.OPEN) {
        // 包装为 input 类型 JSON
        socket.send(JSON.stringify({
            type: 'input',
            data: data
        }))
    }
  })
}

const closeTerminal = () => {
  if (socket) {
    socket.close()
    socket = null
  }
  if (term) {
    term.dispose()
    term = null
  }
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  connectionStatus.value = 'disconnected'
}

onBeforeUnmount(() => {
  closeTerminal()
})
</script>

<style scoped>
.terminal-dialog :deep(.el-dialog__body) {
  padding: 0;
  height: 70vh;
  display: flex;
  flex-direction: column;
  background-color: #1e1e1e;
}

.terminal-toolbar {
  height: 40px;
  background-color: #2d2d2d;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 15px;
  border-bottom: 1px solid #333;
}

.terminal-container {
  flex: 1;
  width: 100%;
  height: calc(100% - 40px);
  overflow: hidden;
  padding: 5px;
}

.status-indicator {
  display: flex;
  align-items: center;
  font-size: 12px;
  color: #ccc;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
  background-color: #666;
}

.status-dot.connected {
  background-color: #67c23a;
}
.status-dot.connecting {
  background-color: #e6a23c;
}
.status-dot.error {
  background-color: #f56c6c;
}

.help-text {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
