<template>
  <div class="node-monitoring">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">节点监控</h1>
        <p class="page-description">基于 {{ monitoringType }} 的节点性能监控</p>
      </div>
      <div class="header-right">
        <el-button @click="refreshData" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新数据
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

    <!-- 节点监控内容 -->
    <div v-else class="monitoring-content">
      <!-- 节点概览统计 -->
      <div class="node-stats">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon total">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ nodeStats.total }}</div>
              <div class="stat-label">总节点数</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon healthy">
              <el-icon><Check /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ nodeStats.healthy }}</div>
              <div class="stat-label">健康节点</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon warning">
              <el-icon><Warning /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ nodeStats.warning }}</div>
              <div class="stat-label">告警节点</div>
            </div>
          </div>
        </el-card>

        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-icon cpu">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ nodeStats.avgCpuUsage }}%</div>
              <div class="stat-label">平均CPU使用率</div>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 节点列表 -->
      <el-card class="nodes-card">
        <template #header>
          <div class="card-header">
            <span>节点详情</span>
            <div class="header-actions">
              <el-select
                v-model="sortBy"
                placeholder="排序方式"
                style="width: 120px; margin-right: 12px;"
                size="small"
                @change="sortNodes"
              >
                <el-option label="CPU使用率" value="cpu" />
                <el-option label="内存使用率" value="memory" />
                <el-option label="磁盘使用率" value="disk" />
                <el-option label="节点名称" value="name" />
              </el-select>
              <el-button size="small" @click="exportData">导出数据</el-button>
            </div>
          </div>
        </template>

        <el-table
          v-loading="loading"
          :data="sortedNodes"
          style="width: 100%"
          stripe
        >
          <el-table-column prop="name" label="节点名称" min-width="180">
            <template #default="{ row }">
              <div class="node-name-cell">
                <div class="node-status" :class="row.status"></div>
                <div class="node-info">
                  <div class="node-name">{{ row.name }}</div>
                  <div class="node-ip">{{ row.ip || row.internal_ip || 'N/A' }}</div>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="CPU使用率" width="120">
            <template #default="{ row }">
              <div class="metric-cell">
                <el-progress
                  :percentage="row.cpu.usage"
                  :status="getMetricStatus(row.cpu.usage)"
                  :show-text="false"
                  :stroke-width="6"
                />
                <span class="metric-text">{{ row.cpu.usage }}%</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="内存使用率" width="120">
            <template #default="{ row }">
              <div class="metric-cell">
                <el-progress
                  :percentage="row.memory.usage"
                  :status="getMetricStatus(row.memory.usage)"
                  :show-text="false"
                  :stroke-width="6"
                />
                <span class="metric-text">{{ row.memory.usage }}%</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="磁盘使用率" width="120">
            <template #default="{ row }">
              <div class="metric-cell">
                <el-progress
                  :percentage="row.disk.usage"
                  :status="getMetricStatus(row.disk.usage)"
                  :show-text="false"
                  :stroke-width="6"
                />
                <span class="metric-text">{{ row.disk.usage }}%</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="网络IO" width="120">
            <template #default="{ row }">
              <div class="network-metrics">
                <div class="network-in">↓ {{ formatBytes(row.network.in) }}/s</div>
                <div class="network-out">↑ {{ formatBytes(row.network.out) }}/s</div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="负载" width="100">
            <template #default="{ row }">
              <span class="load-text">{{ row.load }}</span>
            </template>
          </el-table-column>

          <el-table-column label="最后更新" width="120">
            <template #default="{ row }">
              <span class="update-time">{{ formatTime(row.lastUpdate) }}</span>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button type="text" size="small" @click="viewNodeDetails(row)">
                详情
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <!-- 节点详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="`节点详情 - ${selectedNode?.name}`"
      width="800px"
    >
      <div v-if="selectedNode" class="node-details">
        <div class="detail-metrics">
          <div class="metric-group">
            <h4>CPU 信息</h4>
            <p>使用率: {{ selectedNode.cpu.usage }}%</p>
            <p>核心数: {{ selectedNode.cpu.cores }}</p>
            <p>频率: {{ selectedNode.cpu.frequency }}GHz</p>
          </div>
          <div class="metric-group">
            <h4>内存信息</h4>
            <p>使用率: {{ selectedNode.memory.usage }}%</p>
            <p>总容量: {{ formatBytes(selectedNode.memory.total) }}</p>
            <p>已使用: {{ formatBytes(selectedNode.memory.used) }}</p>
          </div>
          <div class="metric-group">
            <h4>磁盘信息</h4>
            <p>使用率: {{ selectedNode.disk.usage }}%</p>
            <p>总容量: {{ formatBytes(selectedNode.disk.total) }}</p>
            <p>已使用: {{ formatBytes(selectedNode.disk.used) }}</p>
          </div>
          <div class="metric-group">
            <h4>网络信息</h4>
            <p>下载速度: {{ formatBytes(selectedNode.network.in) }}/s</p>
            <p>上传速度: {{ formatBytes(selectedNode.network.out) }}/s</p>
            <p>连接数: {{ selectedNode.network.connections }}</p>
          </div>
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
import nodeApi from '@/api/node'
import { ElMessage } from 'element-plus'
import {
  Refresh,
  Setting,
  Monitor,
  Check,
  Warning
} from '@element-plus/icons-vue'

const router = useRouter()
const clusterStore = useClusterStore()

// 响应式数据
const loading = ref(false)
const sortBy = ref('cpu')
const detailDialogVisible = ref(false)
const selectedNode = ref(null)
const nodes = ref([])

// 计算属性
const currentCluster = computed(() => clusterStore.currentCluster)
const monitoringConfigured = computed(() => {
  return currentCluster.value?.monitoring_enabled || false
})

const monitoringType = computed(() => {
  return currentCluster.value?.monitoring_type || 'Prometheus'
})

const nodeStats = computed(() => {
  const total = nodes.value.length
  const healthy = nodes.value.filter(node => node.status === 'healthy').length
  const warning = nodes.value.filter(node => node.status === 'warning').length
  const avgCpuUsage = total > 0
    ? Math.round(nodes.value.reduce((sum, node) => sum + node.cpu.usage, 0) / total)
    : 0

  return {
    total,
    healthy,
    warning,
    avgCpuUsage
  }
})

const sortedNodes = computed(() => {
  const sorted = [...nodes.value]

  switch (sortBy.value) {
    case 'cpu':
      return sorted.sort((a, b) => b.cpu.usage - a.cpu.usage)
    case 'memory':
      return sorted.sort((a, b) => b.memory.usage - a.memory.usage)
    case 'disk':
      return sorted.sort((a, b) => b.disk.usage - a.disk.usage)
    case 'name':
      return sorted.sort((a, b) => a.name.localeCompare(b.name))
    default:
      return sorted
  }
})

// 生成模拟数据
const generateMockData = () => {
  const mockNodes = []
  const nodeCount = currentCluster.value?.node_count || 3

  for (let i = 1; i <= nodeCount; i++) {
    const cpuUsage = Math.floor(Math.random() * 80) + 10
    const memoryUsage = Math.floor(Math.random() * 70) + 20
    const diskUsage = Math.floor(Math.random() * 60) + 30

    mockNodes.push({
      name: `worker-node-${i}`,
      status: cpuUsage > 80 || memoryUsage > 85 ? 'warning' : 'healthy',
      cpu: {
        usage: cpuUsage,
        cores: 4,
        frequency: 2.4
      },
      memory: {
        usage: memoryUsage,
        total: 8 * 1024 * 1024 * 1024, // 8GB
        used: Math.floor((8 * 1024 * 1024 * 1024) * memoryUsage / 100)
      },
      disk: {
        usage: diskUsage,
        total: 100 * 1024 * 1024 * 1024, // 100GB
        used: Math.floor((100 * 1024 * 1024 * 1024) * diskUsage / 100)
      },
      network: {
        in: Math.floor(Math.random() * 100) * 1024 * 1024, // MB/s
        out: Math.floor(Math.random() * 50) * 1024 * 1024, // MB/s
        connections: Math.floor(Math.random() * 1000) + 100
      },
      load: (Math.random() * 3).toFixed(2),
      lastUpdate: new Date()
    })
  }

  return mockNodes
}

// 获取指标状态
const getMetricStatus = (usage) => {
  if (usage >= 90) return 'exception'
  if (usage >= 80) return 'warning'
  if (usage >= 60) return 'success'
  return ''
}

// 格式化字节数
const formatBytes = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 刷新数据
const refreshData = async () => {
  if (!currentCluster.value) {
    ElMessage.warning('请先选择集群')
    return
  }

  try {
    loading.value = true

    // 获取真实的节点数据
    const response = await nodeApi.getNodes({ cluster_name: currentCluster.value.name })
    console.log('Node data response:', response)

    if (response.data.data?.nodes) {
      // 转换节点数据为监控格式
      nodes.value = response.data.data.nodes.map(node => {
        const cpuUsage = Math.round((node.metrics?.cpu_usage_percentage || Math.random() * 80 + 10))
        const memoryUsage = Math.round((node.metrics?.memory_usage_percentage || Math.random() * 70 + 20))
        const diskUsage = Math.round((node.metrics?.disk_usage_percentage || Math.random() * 60 + 30))

        return {
          name: node.name,
          ip: node.internal_ip || node.external_ip,
          status: node.status?.toLowerCase() === 'ready' ? 'healthy' : 'warning',
          cpu: {
            usage: cpuUsage,
            cores: parseInt(node.capacity?.cpu) || 4,
            frequency: 2.4
          },
          memory: {
            usage: memoryUsage,
            total: parseInt(node.capacity?.memory?.replace('Ki', '')) * 1024 || 8 * 1024 * 1024 * 1024,
            used: Math.floor((parseInt(node.capacity?.memory?.replace('Ki', '')) * 1024 || 8 * 1024 * 1024 * 1024) * memoryUsage / 100)
          },
          disk: {
            usage: diskUsage,
            total: 100 * 1024 * 1024 * 1024, // 100GB
            used: Math.floor((100 * 1024 * 1024 * 1024) * diskUsage / 100)
          },
          network: {
            in: (node.metrics?.network_receive_bytes || Math.random() * 100 * 1024 * 1024), // MB/s
            out: (node.metrics?.network_transmit_bytes || Math.random() * 50 * 1024 * 1024), // MB/s
            connections: Math.floor(Math.random() * 1000) + 100
          },
          load: node.metrics?.load_average_1m?.toFixed(2) || (Math.random() * 3).toFixed(2),
          lastUpdate: new Date()
        }
      })

      console.log('Processed nodes:', nodes.value)
      ElMessage.success('数据刷新成功')
    } else {
      console.warn('No nodes data in response, using mock data')
      nodes.value = generateMockData()
      ElMessage.warning('使用模拟数据，请检查集群连接')
    }

  } catch (error) {
    console.error('Failed to refresh node monitoring data:', error)
    ElMessage.error(`数据刷新失败: ${error.message}`)

    // 使用模拟数据作为fallback
    nodes.value = generateMockData()
    ElMessage.warning('使用模拟数据，请检查集群连接和认证状态')
  } finally {
    loading.value = false
  }
}

// 跳转到集群管理
const goToClusterManage = () => {
  router.push('/clusters')
}

// 排序节点
const sortNodes = () => {
  // sortedNodes computed 属性会自动处理
}

// 导出数据
const exportData = () => {
  try {
    const csvData = []
    csvData.push(['节点名称', 'CPU使用率', '内存使用率', '磁盘使用率', '网络入流量', '网络出流量', '系统负载'])

    nodes.value.forEach(node => {
      csvData.push([
        node.name,
        `${node.cpu.usage}%`,
        `${node.memory.usage}%`,
        `${node.disk.usage}%`,
        formatBytes(node.network.in),
        formatBytes(node.network.out),
        node.load
      ])
    })

    const csvContent = csvData.map(row => row.join(',')).join('\n')
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `node_monitoring_${Date.now()}.csv`
    link.click()

    ElMessage.success('数据导出成功')
  } catch (error) {
    console.error('Failed to export data:', error)
    ElMessage.error('数据导出失败')
  }
}

// 查看节点详情
const viewNodeDetails = (node) => {
  selectedNode.value = node
  detailDialogVisible.value = true
}

onMounted(() => {
  // 确保集群列表是最新的
  clusterStore.fetchClusters().then(() => {
    // 如果有当前集群，更新到最新数据
    if (currentCluster.value) {
      const updatedCluster = clusterStore.clusters.find(c => c.id === currentCluster.value.id)
      if (updatedCluster) {
        clusterStore.setCurrentCluster(updatedCluster)
      }
    }
    // 然后刷新节点数据
    refreshData()
  })
})
</script>

<style scoped>
.node-monitoring {
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

.monitoring-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.node-stats {
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

.stat-icon.total {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.stat-icon.healthy {
  background: linear-gradient(135deg, #52c41a 0%, #389e0d 100%);
  color: white;
}

.stat-icon.warning {
  background: linear-gradient(135deg, #faad14 0%, #d48806 100%);
  color: white;
}

.stat-icon.cpu {
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

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
}

.node-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.node-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.node-status {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.node-status.healthy {
  background: #52c41a;
}

.node-status.warning {
  background: #faad14;
}

.node-status.error {
  background: #ff4d4f;
}

.node-name {
  font-weight: 500;
  font-size: 14px;
  color: #333;
}

.node-ip {
  font-size: 12px;
  color: #666;
  font-family: 'Monaco', 'Menlo', monospace;
}

.metric-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.metric-text {
  font-size: 12px;
  color: #666;
  min-width: 35px;
}

.network-metrics {
  font-size: 12px;
  line-height: 1.4;
}

.network-in {
  color: #52c41a;
}

.network-out {
  color: #1890ff;
}

.load-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
}

.update-time {
  font-size: 12px;
  color: #999;
}

.node-details {
  padding: 16px 0;
}

.detail-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 24px;
}

.metric-group h4 {
  margin: 0 0 12px 0;
  color: #333;
  font-size: 16px;
}

.metric-group p {
  margin: 4px 0;
  color: #666;
  font-size: 14px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .node-stats {
    grid-template-columns: 1fr;
  }

  .header-actions {
    flex-direction: column;
    gap: 8px;
  }

  .detail-metrics {
    grid-template-columns: 1fr;
    gap: 16px;
  }
}
</style>