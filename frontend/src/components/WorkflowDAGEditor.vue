<template>
  <div class="dag-editor">
    <div class="toolbar">
      <div class="toolbar-left">
        <el-button-group>
          <el-button size="small" icon="el-icon-plus" @click="addNode('task')">添加任务节点</el-button>
          <el-button size="small" icon="el-icon-connection" @click="toggleConnectionMode">
            {{ connectionMode ? '取消连线' : '连线模式' }}
          </el-button>
          <el-button size="small" icon="el-icon-delete" @click="deleteSelected">删除选中</el-button>
        </el-button-group>
        <el-tag style="margin-left: 15px;">节点数: {{ dag.nodes.length }}</el-tag>
        <el-alert
          title="提示：双击节点可以配置任务详情，配置完整后节点边框会变为实线"
          type="info"
          :closable="false"
          style="margin-left: 15px; padding: 8px 12px;"
        />
      </div>
      <el-button size="small" type="primary" @click="saveDAG">保存</el-button>
    </div>

    <div class="canvas" ref="canvasRef" @click="handleCanvasClick">
      <!-- 调试信息 -->
      <div style="position: absolute; top: 10px; left: 10px; background: rgba(255,255,255,0.9); padding: 10px; border: 1px solid #ccc; z-index: 1000; font-size: 12px;">
        <div>节点数量: {{ dag.nodes.length }}</div>
        <div>边数量: {{ dag.edges.length }}</div>
        <div v-for="(node, index) in dag.nodes" :key="node.id" style="margin-top: 5px;">
          节点{{index}}: {{ node.label }} ({{ node.type }}) at [{{ node.position.x }}, {{ node.position.y }}]
        </div>
      </div>
      
      <svg width="100%" height="100%">
        <!-- 绘制边 -->
        <g v-for="edge in dag.edges" :key="edge.id">
          <path
            :d="getEdgePath(edge)"
            stroke="#999"
            stroke-width="2"
            fill="none"
            marker-end="url(#arrowhead)"
            :class="{ selected: selectedEdge === edge.id }"
            @click.stop="selectEdge(edge.id)"
          />
        </g>
        
        <!-- 箭头标记定义 -->
        <defs>
          <marker
            id="arrowhead"
            markerWidth="10"
            markerHeight="10"
            refX="9"
            refY="3"
            orient="auto"
          >
            <polygon points="0 0, 10 3, 0 6" fill="#999" />
          </marker>
        </defs>
      </svg>

      <!-- 绘制节点 -->
      <div
        v-for="node in dag.nodes"
        :key="node.id"
        class="node"
        :class="[
          node.type, 
          { 
            selected: selectedNode === node.id,
            'not-configured': node.type === 'task' && !isNodeConfigured(node)
          }
        ]"
        :style="{
          left: node.position.x + 'px',
          top: node.position.y + 'px'
        }"
        @mousedown="startDrag(node, $event)"
        @click.stop="selectNode(node.id)"
        @dblclick.stop="editNode(node)"
      >
        <div class="node-header">
          <span class="node-type-icon">
            <i v-if="node.type === 'start'" class="el-icon-video-play"></i>
            <i v-else-if="node.type === 'end'" class="el-icon-check"></i>
            <i v-else class="el-icon-document"></i>
          </span>
          <span class="node-label">{{ node.label }}</span>
          <i v-if="node.type === 'task' && !isNodeConfigured(node)" 
             class="el-icon-warning" 
             style="color: #f56c6c; margin-left: 5px;"
             title="节点未配置完整，请双击编辑"
          ></i>
        </div>
        <div v-if="node.type === 'task' && node.task_config" class="node-body">
          <div class="node-info">
            {{ node.task_config.name || '未配置任务名称' }}
          </div>
          <div class="node-hint" v-if="!isNodeConfigured(node)">
            <small style="color: #f56c6c;">双击配置</small>
          </div>
        </div>
        <div v-if="connectionMode" class="connection-points">
          <div class="point input" @click.stop="connectTo(node.id)" />
          <div class="point output" @click.stop="connectFrom(node.id)" />
        </div>
      </div>
    </div>

    <!-- 节点编辑对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑节点" width="600px">
      <el-form ref="editFormRef" :model="editingNode" :rules="editFormRules" label-width="100px">
        <el-form-item label="节点标签" prop="label">
          <el-input v-model="editingNode.label" placeholder="为节点设置一个描述性的标签" />
        </el-form-item>
        <template v-if="editingNode.type === 'task'">
          <el-form-item label="任务名称" prop="task_config.name">
            <el-input 
              v-model="editingNode.task_config.name" 
              placeholder="输入任务名称，用于标识该任务"
            />
          </el-form-item>
          <el-form-item label="主机清单" prop="task_config.inventory_id">
            <el-select 
              v-model="editingNode.task_config.inventory_id" 
              placeholder="选择要执行任务的主机清单"
              style="width: 100%;"
            >
              <el-option
                v-for="inv in inventories"
                :key="inv.id"
                :label="inv.name"
                :value="inv.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="Playbook" prop="task_config.playbook_content">
            <el-input
              v-model="editingNode.task_config.playbook_content"
              type="textarea"
              :rows="10"
              placeholder="输入 Ansible Playbook YAML 内容，例如：&#10;- hosts: all&#10;  tasks:&#10;    - name: ping&#10;      ping:"
            />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveNodeEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({ nodes: [], edges: [] })
  },
  inventories: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue', 'save'])

// 初始化 DAG 数据
const initializeDAG = () => {
  const initialNodes = props.modelValue?.nodes || []
  const initialEdges = props.modelValue?.edges || []
  
  // 如果没有节点，添加开始和结束节点
  if (initialNodes.length === 0) {
    return {
      nodes: [
        {
          id: 'start',
          type: 'start',
          label: '开始',
          position: { x: 100, y: 50 }
        },
        {
          id: 'end',
          type: 'end',
          label: '结束',
          position: { x: 100, y: 400 }
        }
      ],
      edges: []
    }
  }
  
  return {
    nodes: initialNodes,
    edges: initialEdges
  }
}

const dag = reactive(initializeDAG())

// 工具栏状态
const connectionMode = ref(false)
const selectedNode = ref(null)
const selectedEdge = ref(null)
const connectingFrom = ref(null)

// 编辑对话框
const editDialogVisible = ref(false)
const editingNode = ref(null)

// 拖拽状态
const dragging = ref(false)
const dragNode = ref(null)
const dragOffset = ref({ x: 0, y: 0 })

// 画布引用
const canvasRef = ref(null)
const editFormRef = ref(null)

// 表单验证规则
const editFormRules = {
  label: [
    { required: true, message: '节点标签不能为空', trigger: 'blur' }
  ],
  'task_config.name': [
    { required: true, message: '任务名称不能为空', trigger: 'blur' },
    { min: 2, max: 100, message: '任务名称长度在 2 到 100 个字符', trigger: 'blur' }
  ],
  'task_config.inventory_id': [
    { required: true, message: '请选择主机清单', trigger: 'change' }
  ],
  'task_config.playbook_content': [
    { required: true, message: 'Playbook 内容不能为空', trigger: 'blur' }
  ]
}

// 标记是否正在同步，避免循环更新
const syncing = ref(false)

// 监听 props.modelValue 变化，更新 DAG
watch(
  () => props.modelValue,
  (newValue) => {
    if (syncing.value) return
    
    if (newValue && (newValue.nodes || newValue.edges)) {
      // 只在有实际数据且与当前不同时更新
      const nodesChanged = JSON.stringify(newValue.nodes) !== JSON.stringify(dag.nodes)
      const edgesChanged = JSON.stringify(newValue.edges) !== JSON.stringify(dag.edges)
      
      if (nodesChanged || edgesChanged) {
        syncing.value = true
        if (newValue.nodes) dag.nodes = [...newValue.nodes]
        if (newValue.edges) dag.edges = [...newValue.edges]
        syncing.value = false
      }
    }
  },
  { deep: true, immediate: false }
)

// 监听 DAG 变化，同步到父组件
watch(
  dag,
  (newDag) => {
    if (syncing.value) return
    
    syncing.value = true
    emit('update:modelValue', { 
      nodes: [...newDag.nodes], 
      edges: [...newDag.edges] 
    })
    syncing.value = false
  },
  { deep: true }
)

// 添加节点
const addNode = (type) => {
  const id = `node-${Date.now()}`
  const node = {
    id,
    type,
    label: type === 'start' ? '开始' : type === 'end' ? '结束' : '新任务',
    position: { x: 100 + dag.nodes.length * 50, y: 100 + dag.nodes.length * 30 },
    task_config: type === 'task' ? {
      name: '',
      inventory_id: null,
      playbook_content: ''
    } : undefined
  }
  dag.nodes.push(node)
}

// 切换连线模式
const toggleConnectionMode = () => {
  connectionMode.value = !connectionMode.value
  if (!connectionMode.value) {
    connectingFrom.value = null
  }
}

// 从节点开始连线
const connectFrom = (nodeId) => {
  if (!connectionMode.value) return
  connectingFrom.value = nodeId
  ElMessage.info('请点击目标节点完成连线')
}

// 连接到目标节点
const connectTo = (nodeId) => {
  if (!connectionMode.value || !connectingFrom.value) return
  
  if (connectingFrom.value === nodeId) {
    ElMessage.warning('不能连接到自己')
    return
  }

  // 检查是否已存在相同的边
  const exists = dag.edges.some(
    e => e.source === connectingFrom.value && e.target === nodeId
  )
  if (exists) {
    ElMessage.warning('该连接已存在')
    return
  }

  // 添加边
  const edgeId = `edge-${Date.now()}`
  dag.edges.push({
    id: edgeId,
    source: connectingFrom.value,
    target: nodeId
  })

  connectingFrom.value = null
  ElMessage.success('连线成功')
}

// 获取边的路径
const getEdgePath = (edge) => {
  const sourceNode = dag.nodes.find(n => n.id === edge.source)
  const targetNode = dag.nodes.find(n => n.id === edge.target)
  
  if (!sourceNode || !targetNode) return ''

  const x1 = sourceNode.position.x + 100 // 节点宽度的一半
  const y1 = sourceNode.position.y + 40  // 节点高度
  const x2 = targetNode.position.x + 100
  const y2 = targetNode.position.y

  // 简单的贝塞尔曲线
  const cx = (x1 + x2) / 2
  return `M ${x1} ${y1} Q ${cx} ${y1}, ${cx} ${(y1 + y2) / 2} T ${x2} ${y2}`
}

// 选择节点
const selectNode = (nodeId) => {
  selectedNode.value = nodeId
  selectedEdge.value = null
}

// 选择边
const selectEdge = (edgeId) => {
  selectedEdge.value = edgeId
  selectedNode.value = null
}

// 编辑节点
const editNode = (node) => {
  editingNode.value = JSON.parse(JSON.stringify(node))
  editDialogVisible.value = true
  // 清除表单验证状态
  setTimeout(() => {
    if (editFormRef.value) {
      editFormRef.value.clearValidate()
    }
  }, 0)
}

// 保存节点编辑
const saveNodeEdit = async () => {
  if (!editFormRef.value) return
  
  try {
    await editFormRef.value.validate()
    const index = dag.nodes.findIndex(n => n.id === editingNode.value.id)
    if (index !== -1) {
      dag.nodes[index] = editingNode.value
    }
    editDialogVisible.value = false
    ElMessage.success('节点配置保存成功')
  } catch (error) {
    console.log('Validation failed:', error)
  }
}

// 删除选中
const deleteSelected = () => {
  if (selectedNode.value) {
    // 删除节点及相关的边
    dag.nodes = dag.nodes.filter(n => n.id !== selectedNode.value)
    dag.edges = dag.edges.filter(
      e => e.source !== selectedNode.value && e.target !== selectedNode.value
    )
    selectedNode.value = null
  } else if (selectedEdge.value) {
    // 删除边
    dag.edges = dag.edges.filter(e => e.id !== selectedEdge.value)
    selectedEdge.value = null
  }
}

// 开始拖拽
const startDrag = (node, event) => {
  dragging.value = true
  dragNode.value = node
  const rect = event.target.getBoundingClientRect()
  dragOffset.value = {
    x: event.clientX - rect.left,
    y: event.clientY - rect.top
  }
  
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

// 拖拽中
const onDrag = (event) => {
  if (!dragging.value || !dragNode.value) return
  
  const canvas = canvasRef.value
  const rect = canvas.getBoundingClientRect()
  
  dragNode.value.position.x = event.clientX - rect.left - dragOffset.value.x
  dragNode.value.position.y = event.clientY - rect.top - dragOffset.value.y
}

// 停止拖拽
const stopDrag = () => {
  dragging.value = false
  dragNode.value = null
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

// 画布点击
const handleCanvasClick = () => {
  selectedNode.value = null
  selectedEdge.value = null
}

// 检查节点是否已配置
const isNodeConfigured = (node) => {
  if (node.type !== 'task') return true
  if (!node.task_config) return false
  
  return !!(
    node.task_config.name && 
    node.task_config.name.trim() !== '' &&
    node.task_config.inventory_id &&
    node.task_config.playbook_content && 
    node.task_config.playbook_content.trim() !== ''
  )
}

// 验证 DAG
const validateDAG = () => {
  const errors = []
  
  // 检查是否有节点
  if (dag.nodes.length === 0) {
    errors.push('至少需要添加一个节点')
    return errors
  }
  
  // 检查任务节点配置
  dag.nodes.forEach(node => {
    if (node.type === 'task') {
      if (!node.task_config) {
        errors.push(`节点"${node.label}"缺少任务配置，请双击节点进行配置`)
        return
      }
      
      if (!node.task_config.name || node.task_config.name.trim() === '') {
        errors.push(`节点"${node.label}"的任务名称不能为空，请双击节点配置任务名称`)
      }
      
      if (!node.task_config.inventory_id) {
        errors.push(`节点"${node.label}"未选择主机清单，请双击节点配置主机清单`)
      }
      
      if (!node.task_config.playbook_content || node.task_config.playbook_content.trim() === '') {
        errors.push(`节点"${node.label}"的 Playbook 内容不能为空，请双击节点配置 Playbook`)
      }
    }
  })
  
  return errors
}

// 保存 DAG
const saveDAG = () => {
  // 验证 DAG
  const errors = validateDAG()
  if (errors.length > 0) {
    // 使用通知组件显示多个错误
    ElNotification({
      title: 'DAG 配置不完整',
      message: errors.map((err, index) => `${index + 1}. ${err}`).join('\n'),
      type: 'error',
      duration: 8000,
      dangerouslyUseHTMLString: false
    })
    return
  }
  
  emit('save', { ...dag })
}

// 初始化
onMounted(() => {
  console.log('=== WorkflowDAGEditor mounted ===')
  console.log('props.modelValue:', props.modelValue)
  console.log('dag.nodes count:', dag.nodes.length)
  console.log('dag.nodes:', JSON.stringify(dag.nodes, null, 2))
  console.log('dag.edges:', JSON.stringify(dag.edges, null, 2))
  
  // 强制触发一次更新
  if (dag.nodes.length > 0) {
    console.log('Emitting initial update to parent')
    emit('update:modelValue', { 
      nodes: [...dag.nodes], 
      edges: [...dag.edges] 
    })
  }
})
</script>

<style scoped>
.dag-editor {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f5f5f5;
}

.dag-editor .toolbar {
  padding: 12px;
  background: white;
  border-bottom: 1px solid #ddd;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dag-editor .toolbar .toolbar-left {
  display: flex;
  align-items: center;
  flex: 1;
}

.dag-editor .canvas {
  flex: 1;
  position: relative;
  overflow: hidden;
  background: 
    linear-gradient(90deg, #e5e5e5 1px, transparent 1px),
    linear-gradient(#e5e5e5 1px, transparent 1px);
  background-size: 20px 20px;
}

.dag-editor .canvas svg {
  position: absolute;
  top: 0;
  left: 0;
  pointer-events: none;
}

.dag-editor .canvas svg path {
  pointer-events: stroke;
  cursor: pointer;
  transition: stroke 0.2s;
}

.dag-editor .canvas svg path:hover,
.dag-editor .canvas svg path.selected {
  stroke: #409eff;
  stroke-width: 3;
}

.dag-editor .canvas .node {
  position: absolute;
  width: 200px;
  background: white;
  border: 2px solid #ddd;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  cursor: move;
  transition: all 0.2s;
}

.dag-editor .canvas .node:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.dag-editor .canvas .node.selected {
  border-color: #409eff;
  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.2);
}

.dag-editor .canvas .node.start {
  border-color: #67c23a;
}

.dag-editor .canvas .node.start .node-header {
  background: #67c23a;
  color: white;
}

.dag-editor .canvas .node.end {
  border-color: #909399;
}

.dag-editor .canvas .node.end .node-header {
  background: #909399;
  color: white;
}

.dag-editor .canvas .node.task {
  border-color: #409eff;
}

.dag-editor .canvas .node.task .node-header {
  background: #409eff;
  color: white;
}

.dag-editor .canvas .node.not-configured {
  border-color: #f56c6c;
  border-style: dashed;
  animation: pulse 2s ease-in-out infinite;
}

.dag-editor .canvas .node.not-configured .node-header {
  background: #f56c6c;
}

@keyframes pulse {
  0%, 100% {
    box-shadow: 0 2px 8px rgba(245, 108, 108, 0.2);
  }
  50% {
    box-shadow: 0 4px 12px rgba(245, 108, 108, 0.4);
  }
}

.dag-editor .canvas .node .node-header {
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  border-radius: 6px 6px 0 0;
  font-weight: 600;
}

.dag-editor .canvas .node .node-header .node-type-icon {
  font-size: 16px;
}

.dag-editor .canvas .node .node-header .node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dag-editor .canvas .node .node-body {
  padding: 12px;
}

.dag-editor .canvas .node .node-body .node-info {
  font-size: 13px;
  color: #666;
}

.dag-editor .canvas .node .connection-points .point {
  position: absolute;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #409eff;
  border: 2px solid white;
  cursor: pointer;
}

.dag-editor .canvas .node .connection-points .point:hover {
  transform: scale(1.3);
}

.dag-editor .canvas .node .connection-points .point.input {
  top: -6px;
  left: 50%;
  transform: translateX(-50%);
}

.dag-editor .canvas .node .connection-points .point.output {
  bottom: -6px;
  left: 50%;
  transform: translateX(-50%);
}
</style>

