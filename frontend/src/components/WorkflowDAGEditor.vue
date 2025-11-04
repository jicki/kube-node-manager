<template>
  <div class="dag-editor">
    <div class="toolbar">
      <el-button-group>
        <el-button size="small" icon="el-icon-plus" @click="addNode('task')">添加任务节点</el-button>
        <el-button size="small" icon="el-icon-connection" @click="toggleConnectionMode">
          {{ connectionMode ? '取消连线' : '连线模式' }}
        </el-button>
        <el-button size="small" icon="el-icon-delete" @click="deleteSelected">删除选中</el-button>
      </el-button-group>
      <el-button size="small" type="primary" @click="saveDAG">保存</el-button>
    </div>

    <div class="canvas" ref="canvasRef" @click="handleCanvasClick">
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
        :class="[node.type, { selected: selectedNode === node.id }]"
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
        </div>
        <div v-if="node.type === 'task' && node.task_config" class="node-body">
          <div class="node-info">{{ node.task_config.name }}</div>
        </div>
        <div v-if="connectionMode" class="connection-points">
          <div class="point input" @click.stop="connectTo(node.id)" />
          <div class="point output" @click.stop="connectFrom(node.id)" />
        </div>
      </div>
    </div>

    <!-- 节点编辑对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑节点" width="600px">
      <el-form :model="editingNode" label-width="100px">
        <el-form-item label="节点标签">
          <el-input v-model="editingNode.label" />
        </el-form-item>
        <el-form-item v-if="editingNode.type === 'task'" label="任务名称">
          <el-input v-model="editingNode.task_config.name" />
        </el-form-item>
        <el-form-item v-if="editingNode.type === 'task'" label="主机清单">
          <el-select v-model="editingNode.task_config.inventory_id" placeholder="选择主机清单">
            <el-option
              v-for="inv in inventories"
              :key="inv.id"
              :label="inv.name"
              :value="inv.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item v-if="editingNode.type === 'task'" label="Playbook">
          <el-input
            v-model="editingNode.task_config.playbook_content"
            type="textarea"
            :rows="10"
            placeholder="输入 Ansible Playbook YAML 内容"
          />
        </el-form-item>
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
import { ElMessage } from 'element-plus'

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

const dag = reactive({
  nodes: props.modelValue?.nodes || [],
  edges: props.modelValue?.edges || []
})

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

// 监听 DAG 变化，同步到父组件
watch(
  dag,
  (newDag) => {
    emit('update:modelValue', { ...newDag })
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
}

// 保存节点编辑
const saveNodeEdit = () => {
  const index = dag.nodes.findIndex(n => n.id === editingNode.value.id)
  if (index !== -1) {
    dag.nodes[index] = editingNode.value
  }
  editDialogVisible.value = false
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

// 保存 DAG
const saveDAG = () => {
  emit('save', { ...dag })
}

// 初始化
onMounted(() => {
  // 确保至少有开始和结束节点
  if (dag.nodes.length === 0) {
    dag.nodes.push(
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
    )
  }
})
</script>

<style scoped lang="scss">
.dag-editor {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f5f5f5;

  .toolbar {
    padding: 12px;
    background: white;
    border-bottom: 1px solid #ddd;
    display: flex;
    justify-content: space-between;
  }

  .canvas {
    flex: 1;
    position: relative;
    overflow: hidden;
    background: 
      linear-gradient(90deg, #e5e5e5 1px, transparent 1px),
      linear-gradient(#e5e5e5 1px, transparent 1px);
    background-size: 20px 20px;

    svg {
      position: absolute;
      top: 0;
      left: 0;
      pointer-events: none;

      path {
        pointer-events: stroke;
        cursor: pointer;
        transition: stroke 0.2s;

        &:hover, &.selected {
          stroke: #409eff;
          stroke-width: 3;
        }
      }
    }

    .node {
      position: absolute;
      width: 200px;
      background: white;
      border: 2px solid #ddd;
      border-radius: 8px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      cursor: move;
      transition: all 0.2s;

      &:hover {
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
      }

      &.selected {
        border-color: #409eff;
        box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.2);
      }

      &.start {
        border-color: #67c23a;
        .node-header {
          background: #67c23a;
          color: white;
        }
      }

      &.end {
        border-color: #909399;
        .node-header {
          background: #909399;
          color: white;
        }
      }

      &.task {
        border-color: #409eff;
        .node-header {
          background: #409eff;
          color: white;
        }
      }

      .node-header {
        padding: 8px 12px;
        display: flex;
        align-items: center;
        gap: 8px;
        border-radius: 6px 6px 0 0;
        font-weight: 600;

        .node-type-icon {
          font-size: 16px;
        }

        .node-label {
          flex: 1;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }

      .node-body {
        padding: 12px;

        .node-info {
          font-size: 13px;
          color: #666;
        }
      }

      .connection-points {
        .point {
          position: absolute;
          width: 12px;
          height: 12px;
          border-radius: 50%;
          background: #409eff;
          border: 2px solid white;
          cursor: pointer;

          &:hover {
            transform: scale(1.3);
          }

          &.input {
            top: -6px;
            left: 50%;
            transform: translateX(-50%);
          }

          &.output {
            bottom: -6px;
            left: 50%;
            transform: translateX(-50%);
          }
        }
      }
    }
  }
}
</style>

