<template>
  <div class="network-topology">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">网络拓扑</h1>
        <p class="page-description">查看集群网络拓扑结构和连通性</p>
      </div>
      <div class="header-right">
        <el-button @click="refreshTopology" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新拓扑
        </el-button>
        <el-button @click="testConnectivity" :loading="testingConnectivity">
          <el-icon><Connection /></el-icon>
          连通性测试
        </el-button>
      </div>
    </div>

    <!-- 监控状态检查 -->
    <div v-if="!monitoringConfigured" class="monitoring-not-configured">
      <el-empty
        description="当前集群未配置监控"
        :image-size="100"
      >
        <template #description>
          <p>您还没有为当前集群配置监控系统</p>
          <p>请在集群管理中开启监控功能</p>
        </template>
        <el-button type="primary" @click="goToClusterManage">
          <el-icon><Setting /></el-icon>
          配置监控
        </el-button>
      </el-empty>
    </div>

    <!-- 网络拓扑内容 -->
    <div v-else class="topology-content">
      <!-- 网络统计 -->
      <div class="network-stats">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon nodes">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ networkStats.totalNodes }}</div>
              <div class="stat-label">节点总数</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon connections">
              <el-icon><Connection /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ networkStats.activeConnections }}</div>
              <div class="stat-label">活跃连接</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon latency">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ networkStats.avgLatency }}ms</div>
              <div class="stat-label">平均延迟</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon throughput">
              <el-icon><DataLine /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ formatBytes(networkStats.throughput) }}/s</div>
              <div class="stat-label">网络吞吐量</div>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 拓扑图和连通性测试结果 -->
      <div class="topology-main">
        <!-- 拓扑图 -->
        <el-card class="topology-card">
          <template #header>
            <div class="card-header">
              <span>网络拓扑图</span>
              <div class="topology-controls">
                <el-select
                  v-model="viewMode"
                  placeholder="视图模式"
                  style="width: 120px; margin-right: 12px;"
                  size="small"
                >
                  <el-option label="物理视图" value="physical" />
                  <el-option label="逻辑视图" value="logical" />
                </el-select>
                <el-button size="small" @click="exportTopology">导出</el-button>
              </div>
            </div>
          </template>

          <div class="topology-container" ref="topologyRef">
            <div v-loading="loading" class="topology-svg-container">
              <!-- 简化的SVG拓扑图 -->
              <svg width="100%" height="400" viewBox="0 0 800 400">
                <!-- 定义渐变 -->
                <defs>
                  <radialGradient id="nodeGradient" cx="50%" cy="50%" r="50%">
                    <stop offset="0%" style="stop-color:#1890ff;stop-opacity:1" />
                    <stop offset="100%" style="stop-color:#096dd9;stop-opacity:1" />
                  </radialGradient>
                  <radialGradient id="masterGradient" cx="50%" cy="50%" r="50%">
                    <stop offset="0%" style="stop-color:#52c41a;stop-opacity:1" />
                    <stop offset="100%" style="stop-color:#389e0d;stop-opacity:1" />
                  </radialGradient>
                </defs>

                <!-- 连接线 -->
                <g class="connections">
                  <line
                    v-for="connection in connections"
                    :key="`${connection.from}-${connection.to}`"
                    :x1="connection.x1"
                    :y1="connection.y1"
                    :x2="connection.x2"
                    :y2="connection.y2"
                    :stroke="getConnectionColor(connection.status)"
                    stroke-width="2"
                    :stroke-dasharray="connection.status === 'error' ? '5,5' : 'none'"
                  />
                </g>

                <!-- 节点 -->
                <g class="nodes">
                  <g
                    v-for="node in topologyNodes"
                    :key="node.id"
                    class="node-group"
                    :transform="`translate(${node.x}, ${node.y})`"
                  >
                    <circle
                      r="20"
                      :fill="node.type === 'master' ? 'url(#masterGradient)' : 'url(#nodeGradient)'"
                      :stroke="getNodeBorderColor(node.status)"
                      stroke-width="3"
                      class="node-circle"
                      @click="selectNode(node)"
                    />
                    <text
                      y="5"
                      text-anchor="middle"
                      fill="white"
                      font-size="12"
                      font-weight="bold"
                    >
                      {{ node.type === 'master' ? 'M' : 'W' }}
                    </text>
                    <text
                      y="35"
                      text-anchor="middle"
                      fill="#333"
                      font-size="10"
                      class="node-label"
                    >
                      {{ node.name }}
                    </text>
                  </g>
                </g>

                <!-- 图例 -->
                <g class="legend" transform="translate(10, 350)">
                  <rect x="0" y="0" width="200" height="40" fill="rgba(255,255,255,0.9)" stroke="#ddd" rx="4"/>
                  <circle cx="15" cy="15" r="8" fill="url(#masterGradient)" />
                  <text x="30" y="19" font-size="12" fill="#333">Master节点</text>
                  <circle cx="15" cy="30" r="8" fill="url(#nodeGradient)" />
                  <text x="30" y="34" font-size="12" fill="#333">Worker节点</text>
                  <line x1="120" y1="15" x2="140" y2="15" stroke="#52c41a" stroke-width="2"/>
                  <text x="145" y="19" font-size="12" fill="#333">正常连接</text>
                  <line x1="120" y1="30" x2="140" y2="30" stroke="#ff4d4f" stroke-width="2" stroke-dasharray="3,3"/>
                  <text x="145" y="34" font-size="12" fill="#333">异常连接</text>
                </g>
              </svg>
            </div>
          </div>
        </el-card>

        <!-- 连通性测试结果 -->
        <el-card class="connectivity-card">
          <template #header>
            <div class="card-header">
              <span>连通性测试结果</span>
              <el-button size="small" @click="clearTestResults">清空结果</el-button>
            </div>
          </template>

          <div v-if="connectivityResults.length === 0" class="no-results">
            <el-result
              icon="info"
              title="暂无测试结果"
              sub-title="点击上方按钮开始连通性测试"
            />
          </div>

          <div v-else class="connectivity-results">
            <div
              v-for="result in connectivityResults"
              :key="`${result.from}-${result.to}`"
              class="result-item"
              :class="result.status"
            >
              <div class="result-status">
                <el-icon v-if="result.status === 'success'"><Check /></el-icon>
                <el-icon v-else-if="result.status === 'error'"><Close /></el-icon>
                <el-icon v-else><Loading /></el-icon>
              </div>
              <div class="result-content">
                <div class="result-path">{{ result.from }} → {{ result.to }}</div>
                <div class="result-details">
                  <span v-if="result.status === 'success'">
                    延迟: {{ result.latency }}ms, 丢包率: {{ result.packetLoss }}%
                  </span>
                  <span v-else-if="result.status === 'error'">
                    连接失败: {{ result.error }}
                  </span>
                  <span v-else>测试中...</span>
                </div>
              </div>
              <div class="result-time">{{ formatTime(result.timestamp) }}</div>
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <!-- 节点详情对话框 -->
    <el-dialog
      v-model="nodeDetailVisible"
      :title="`节点详情 - ${selectedNodeDetail?.name}`"
      width="600px"
    >
      <div v-if="selectedNodeDetail" class="node-detail-content">
        <div class="detail-item">
          <label>节点名称:</label>
          <span>{{ selectedNodeDetail.name }}</span>
        </div>
        <div class="detail-item">
          <label>节点类型:</label>
          <el-tag :type="selectedNodeDetail.type === 'master' ? 'success' : 'primary'">
            {{ selectedNodeDetail.type === 'master' ? 'Master' : 'Worker' }}
          </el-tag>
        </div>
        <div class="detail-item">
          <label>IP地址:</label>
          <span>{{ selectedNodeDetail.ip }}</span>
        </div>
        <div class="detail-item">
          <label>状态:</label>
          <el-tag :type="selectedNodeDetail.status === 'healthy' ? 'success' : 'danger'">
            {{ selectedNodeDetail.status === 'healthy' ? '正常' : '异常' }}
          </el-tag>
        </div>
        <div class="detail-item">
          <label>连接数:</label>
          <span>{{ selectedNodeDetail.connections }}</span>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useClusterStore } from '@/store/modules/cluster'
import { formatTime } from '@/utils/format'
import {
  Refresh,
  Connection,
  Setting,
  Monitor,
  DataLine,
  Check,
  Close,
  Loading
} from '@element-plus/icons-vue'

const router = useRouter()
const clusterStore = useClusterStore()

// 响应式数据
const loading = ref(false)
const testingConnectivity = ref(false)
const viewMode = ref('physical')
const nodeDetailVisible = ref(false)
const selectedNodeDetail = ref(null)
const topologyRef = ref()

const networkStats = ref({
  totalNodes: 0,
  activeConnections: 0,
  avgLatency: 0,
  throughput: 0
})

const topologyNodes = ref([])
const connections = ref([])
const connectivityResults = ref([])

// 计算属性
const currentCluster = computed(() => clusterStore.currentCluster)
const monitoringConfigured = computed(() => {
  return currentCluster.value?.monitoring_enabled || false
})

// 生成拓扑图数据
const generateTopologyData = () => {
  const nodeCount = currentCluster.value?.node_count || 3
  const nodes = []
  const conns = []

  // 创建master节点
  nodes.push({
    id: 'master-1',
    name: 'master-1',
    type: 'master',
    status: 'healthy',
    ip: '192.168.1.10',
    connections: nodeCount,
    x: 400,
    y: 100
  })

  // 创建worker节点
  const angleStep = (2 * Math.PI) / nodeCount
  for (let i = 0; i < nodeCount; i++) {
    const angle = i * angleStep
    const x = 400 + Math.cos(angle) * 150
    const y = 250 + Math.sin(angle) * 100

    nodes.push({
      id: `worker-${i + 1}`,
      name: `worker-${i + 1}`,
      type: 'worker',
      status: Math.random() > 0.8 ? 'error' : 'healthy',
      ip: `192.168.1.${20 + i}`,
      connections: Math.floor(Math.random() * 50) + 10,
      x,
      y
    })

    // 创建master到worker的连接
    conns.push({
      from: 'master-1',
      to: `worker-${i + 1}`,
      status: Math.random() > 0.9 ? 'error' : 'healthy',
      x1: 400,
      y1: 100,
      x2: x,
      y2: y
    })
  }

  // 创建worker之间的连接
  for (let i = 0; i < nodeCount; i++) {
    for (let j = i + 1; j < nodeCount; j++) {
      if (Math.random() > 0.5) { // 50%概率存在worker间连接
        const node1 = nodes[i + 1]
        const node2 = nodes[j + 1]
        conns.push({
          from: node1.id,
          to: node2.id,
          status: Math.random() > 0.8 ? 'error' : 'healthy',
          x1: node1.x,
          y1: node1.y,
          x2: node2.x,
          y2: node2.y
        })
      }
    }
  }

  return { nodes, connections: conns }
}

// 获取连接颜色
const getConnectionColor = (status) => {
  switch (status) {
    case 'healthy': return '#52c41a'
    case 'error': return '#ff4d4f'
    default: return '#d9d9d9'
  }
}

// 获取节点边框颜色
const getNodeBorderColor = (status) => {
  switch (status) {
    case 'healthy': return '#52c41a'
    case 'error': return '#ff4d4f'
    default: return '#d9d9d9'
  }
}

// 格式化字节数
const formatBytes = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 刷新拓扑
const refreshTopology = async () => {
  if (!monitoringConfigured.value) {
    return
  }

  try {
    loading.value = true

    // 模拟API调用延迟
    await new Promise(resolve => setTimeout(resolve, 1000))

    // 生成拓扑数据
    const { nodes, connections: conns } = generateTopologyData()
    topologyNodes.value = nodes
    connections.value = conns

    // 更新网络统计
    networkStats.value = {
      totalNodes: nodes.length,
      activeConnections: conns.filter(c => c.status === 'healthy').length,
      avgLatency: Math.floor(Math.random() * 50) + 10,
      throughput: Math.floor(Math.random() * 100) * 1024 * 1024 // MB/s
    }

    console.log('Network topology refreshed')
    ElMessage.success('拓扑图刷新成功')

  } catch (error) {
    console.error('Failed to refresh topology:', error)
    ElMessage.error('拓扑图刷新失败')
  } finally {
    loading.value = false
  }
}

// 连通性测试
const testConnectivity = async () => {
  if (!monitoringConfigured.value) {
    ElMessage.warning('请先配置监控系统')
    return
  }

  try {
    testingConnectivity.value = true
    connectivityResults.value = []

    // 生成测试任务
    const testTasks = []
    topologyNodes.value.forEach(from => {
      topologyNodes.value.forEach(to => {
        if (from.id !== to.id) {
          testTasks.push({
            from: from.name,
            to: to.name,
            status: 'testing',
            timestamp: new Date()
          })
        }
      })
    })

    connectivityResults.value = testTasks.slice(0, 6) // 只显示前6个测试

    // 模拟测试过程
    for (let i = 0; i < connectivityResults.value.length; i++) {
      await new Promise(resolve => setTimeout(resolve, 500))

      const success = Math.random() > 0.2 // 80%成功率
      connectivityResults.value[i] = {
        ...connectivityResults.value[i],
        status: success ? 'success' : 'error',
        latency: success ? Math.floor(Math.random() * 100) + 10 : null,
        packetLoss: success ? Math.floor(Math.random() * 5) : null,
        error: success ? null : '连接超时'
      }
    }

    ElMessage.success('连通性测试完成')

  } catch (error) {
    console.error('Connectivity test failed:', error)
    ElMessage.error('连通性测试失败')
  } finally {
    testingConnectivity.value = false
  }
}

// 选择节点
const selectNode = (node) => {
  selectedNodeDetail.value = node
  nodeDetailVisible.value = true
}

// 跳转到集群管理
const goToClusterManage = () => {
  router.push('/clusters')
}

// 导出拓扑
const exportTopology = () => {
  try {
    // 这里可以实现拓扑图导出功能
    ElMessage.success('拓扑图导出功能开发中')
  } catch (error) {
    console.error('Failed to export topology:', error)
    ElMessage.error('导出失败')
  }
}

// 清空测试结果
const clearTestResults = () => {
  connectivityResults.value = []
  ElMessage.success('测试结果已清空')
}

onMounted(() => {
  if (monitoringConfigured.value) {
    refreshTopology()
  }
})
</script>

<style scoped>
.network-topology {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left {
  flex: 1;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px 0;
}

.page-description {
  color: #666;
  margin: 0;
  font-size: 14px;
}

.header-right {
  display: flex;
  gap: 12px;
}

.monitoring-not-configured {
  background: #fff;
  border-radius: 8px;
  padding: 40px;
  text-align: center;
}

.topology-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.network-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  cursor: pointer;
  transition: all 0.3s ease;
}

.stat-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.stat-content {
  display: flex;
  align-items: center;
  padding: 8px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 20px;
}

.stat-icon.nodes {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.stat-icon.connections {
  background: linear-gradient(135deg, #52c41a 0%, #389e0d 100%);
  color: white;
}

.stat-icon.latency {
  background: linear-gradient(135deg, #faad14 0%, #d48806 100%);
  color: white;
}

.stat-icon.throughput {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  color: white;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  color: #333;
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  color: #666;
  margin-top: 4px;
}

.topology-main {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 24px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.topology-controls {
  display: flex;
  align-items: center;
}

.topology-svg-container {
  min-height: 400px;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
}

.node-circle {
  cursor: pointer;
  transition: all 0.3s ease;
}

.node-circle:hover {
  stroke-width: 4;
  filter: brightness(1.1);
}

.node-label {
  cursor: pointer;
}

.no-results {
  padding: 40px 0;
}

.connectivity-results {
  max-height: 400px;
  overflow-y: auto;
}

.result-item {
  display: flex;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.3s ease;
}

.result-item:hover {
  background-color: #fafafa;
}

.result-item.success {
  border-left: 3px solid #52c41a;
}

.result-item.error {
  border-left: 3px solid #ff4d4f;
}

.result-item.testing {
  border-left: 3px solid #1890ff;
}

.result-status {
  width: 24px;
  margin-right: 12px;
  font-size: 16px;
}

.result-status .el-icon {
  color: #52c41a;
}

.result-item.error .result-status .el-icon {
  color: #ff4d4f;
}

.result-item.testing .result-status .el-icon {
  color: #1890ff;
}

.result-content {
  flex: 1;
}

.result-path {
  font-weight: 500;
  color: #333;
  margin-bottom: 4px;
}

.result-details {
  font-size: 12px;
  color: #666;
}

.result-time {
  font-size: 12px;
  color: #999;
  margin-left: 12px;
}

.node-detail-content {
  padding: 16px 0;
}

.detail-item {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}

.detail-item label {
  min-width: 80px;
  color: #666;
  margin-right: 12px;
}

.detail-item span {
  color: #333;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .topology-main {
    grid-template-columns: 1fr;
    gap: 16px;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .network-stats {
    grid-template-columns: 1fr;
  }

  .topology-controls {
    flex-direction: column;
    gap: 8px;
  }
}
</style>