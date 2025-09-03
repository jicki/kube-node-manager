<template>
  <div class="dashboard">
    <!-- 页面标题 -->
    <div class="page-header">
      <h1 class="page-title">概览</h1>
      <p class="page-description">查看集群节点状态和系统运行情况</p>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="24" class="stats-row">
      <el-col :xs="24" :sm="12" :md="6" :lg="6">
        <el-card class="stat-card stat-card-primary">
          <div class="stat-content">
            <div class="stat-icon">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ nodeStats.total }}</div>
              <div class="stat-label">总节点数</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6" :lg="6">
        <el-card class="stat-card stat-card-success">
          <div class="stat-content">
            <div class="stat-icon">
              <el-icon><Check /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ nodeStats.ready }}</div>
              <div class="stat-label">正常节点</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6" :lg="6">
        <el-card class="stat-card stat-card-warning">
          <div class="stat-content">
            <div class="stat-icon">
              <el-icon><Warning /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ nodeStats.notReady }}</div>
              <div class="stat-label">异常节点</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="12" :md="6" :lg="6">
        <el-card class="stat-card stat-card-info">
          <div class="stat-content">
            <div class="stat-icon">
              <el-icon><Connection /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ clusterStats.total }}</div>
              <div class="stat-label">管理集群</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 节点概览分布与集群状态 -->
    <el-row :gutter="24" class="content-row">
      <!-- 节点概览分布 -->
      <el-col :xs="24" :lg="16">
        <el-card class="chart-card node-overview-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">节点概览分布</span>
              <el-button type="text" size="small" @click="refreshNodeStats">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>
          
          <div class="node-overview-container">
            <!-- 左侧：节点状态环形图 -->
            <div class="status-section">
              <div class="section-title">节点状态</div>
              
              <!-- 节点状态环形图 -->
              <div class="node-status-chart">
                <div class="pie-chart">
                  <svg width="180" height="180" viewBox="0 0 180 180" class="pie-svg">
                    <!-- 背景圆 -->
                    <circle
                      cx="90"
                      cy="90"
                      r="65"
                      fill="none"
                      stroke="#f0f0f0"
                      stroke-width="16"
                    />
                    
                    <!-- 正常节点段 -->
                    <circle
                      v-if="nodeStats.ready > 0"
                      cx="90"
                      cy="90"
                      r="65"
                      fill="none"
                      stroke="#67c23a"
                      stroke-width="16"
                      :stroke-dasharray="readyCircumference"
                      :stroke-dashoffset="readyOffset"
                      transform="rotate(-90 90 90)"
                      class="status-arc ready-arc"
                    />
                    
                    <!-- 异常节点段 -->
                    <circle
                      v-if="nodeStats.notReady > 0"
                      cx="90"
                      cy="90"
                      r="65"
                      fill="none"
                      stroke="#f56c6c"
                      stroke-width="16"
                      :stroke-dasharray="notReadyCircumference"
                      :stroke-dashoffset="notReadyOffset"
                      transform="rotate(-90 90 90)"
                      class="status-arc notready-arc"
                    />
                    
                    <!-- 未知节点段 -->
                    <circle
                      v-if="nodeStats.unknown > 0"
                      cx="90"
                      cy="90"
                      r="65"
                      fill="none"
                      stroke="#909399"
                      stroke-width="16"
                      :stroke-dasharray="unknownCircumference"
                      :stroke-dashoffset="unknownOffset"
                      transform="rotate(-90 90 90)"
                      class="status-arc unknown-arc"
                    />
                    
                    <!-- 中心文字 -->
                    <text x="90" y="85" text-anchor="middle" class="center-number">
                      {{ nodeStats.total }}
                    </text>
                    <text x="90" y="105" text-anchor="middle" class="center-label">
                      总节点
                    </text>
                  </svg>
                </div>
                
                <!-- 状态图例 -->
                <div class="chart-legend">
                  <div class="legend-item" :class="{ 'legend-empty': nodeStats.ready === 0 }">
                    <div class="legend-indicator">
                      <span class="legend-color legend-success"></span>
                      <span class="legend-text">正常</span>
                    </div>
                    <span class="legend-value">{{ nodeStats.ready }}</span>
                    <span class="legend-percentage">({{ readyPercentage }}%)</span>
                  </div>
                  <div class="legend-item" :class="{ 'legend-empty': nodeStats.notReady === 0 }">
                    <div class="legend-indicator">
                      <span class="legend-color legend-warning"></span>
                      <span class="legend-text">异常</span>
                    </div>
                    <span class="legend-value">{{ nodeStats.notReady }}</span>
                    <span class="legend-percentage">({{ notReadyPercentage }}%)</span>
                  </div>
                  <div class="legend-item" :class="{ 'legend-empty': nodeStats.unknown === 0 }">
                    <div class="legend-indicator">
                      <span class="legend-color legend-info"></span>
                      <span class="legend-text">未知</span>
                    </div>
                    <span class="legend-value">{{ nodeStats.unknown }}</span>
                    <span class="legend-percentage">({{ unknownPercentage }}%)</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 分隔线 -->
            <el-divider direction="vertical" class="section-divider" />

            <!-- 右侧：节点归属分布 -->
            <div class="ownership-section">
              <div class="section-title">节点归属</div>
              
              <!-- 节点归属图表 -->
              <div class="node-ownership-chart">
                <div v-if="ownershipChartData.length > 0" class="ownership-list">
                  <div 
                    v-for="(item, index) in ownershipChartData" 
                    :key="item.name"
                    class="ownership-item"
                    :style="{ animationDelay: `${index * 0.1}s` }"
                  >
                    <div class="ownership-header">
                      <div class="ownership-indicator">
                        <span 
                          class="ownership-color" 
                          :style="{ backgroundColor: item.color }"
                        ></span>
                        <span class="ownership-name">{{ item.name }}</span>
                      </div>
                      <div class="ownership-stats">
                        <span class="ownership-count">{{ item.count }}</span>
                        <span class="ownership-percentage">{{ item.percentage }}%</span>
                      </div>
                    </div>
                    
                    <div class="ownership-progress">
                      <div 
                        class="ownership-bar"
                        :style="{ 
                          width: `${item.percentage}%`,
                          backgroundColor: item.color 
                        }"
                      ></div>
                    </div>
                  </div>
                </div>
                
                <!-- 空状态 -->
                <div v-else class="empty-ownership">
                  <el-empty description="暂无节点归属数据" :image-size="60">
                    <template #description>
                      <p>节点未配置归属标签</p>
                      <p style="font-size: 12px; color: #999;">标签：deeproute.cn/user-type</p>
                    </template>
                  </el-empty>
                </div>
              </div>
            </div>

            <!-- 空状态 (整体无数据) -->
            <div v-if="nodeStats.total === 0" class="empty-overview">
              <el-empty description="暂无节点数据" :image-size="100">
                <template #description>
                  <p>当前集群中没有节点信息</p>
                  <p>请检查集群配置或添加节点</p>
                </template>
              </el-empty>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 集群状态 -->
      <el-col :xs="24" :lg="8">
        <el-card class="cluster-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">集群状态</span>
              <el-button type="text" size="small" @click="refreshClusters">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>
          
          <div class="cluster-list">
            <div
              v-for="cluster in clusters"
              :key="cluster.id"
              class="cluster-item"
              :class="{ 'active': cluster.id === currentCluster?.id }"
            >
              <div class="cluster-info">
                <div class="cluster-name">{{ cluster.name }}</div>
                <div class="cluster-desc">{{ cluster.description || '无描述' }}</div>
              </div>
              <div class="cluster-status">
                <el-tag
                  :type="cluster.status === 'active' ? 'success' : 'danger'"
                  size="small"
                >
                  {{ cluster.status === 'active' ? '正常' : '异常' }}
                </el-tag>
              </div>
            </div>
            
            <div v-if="clusters.length === 0" class="empty-clusters">
              <el-empty description="暂无集群配置" :image-size="60">
                <template #description>
                  <p>您还没有配置任何Kubernetes集群</p>
                  <p>请先添加集群配置以开始管理节点</p>
                </template>
                <el-button 
                  v-if="isAdmin" 
                  type="primary" 
                  @click="$router.push('/clusters')"
                >
                  <el-icon><Plus /></el-icon>
                  添加集群
                </el-button>
              </el-empty>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 最近操作和快捷入口 -->
    <el-row :gutter="24" class="content-row">
      <!-- 最近操作 -->
      <el-col :xs="24" :lg="14">
        <el-card class="recent-actions-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">最近操作</span>
              <router-link to="/audit" class="more-link">
                查看全部
                <el-icon><ArrowRight /></el-icon>
              </router-link>
            </div>
          </template>
          
          <div class="recent-actions">
            <div
              v-for="action in recentActions"
              :key="action.id"
              class="action-item"
            >
              <div class="action-icon">
                <el-icon
                  :class="getActionIconClass(action.type)"
                >
                  <component :is="getActionIcon(action.type)" />
                </el-icon>
              </div>
              <div class="action-content">
                <div class="action-title">{{ action.description }}</div>
                <div class="action-meta">
                  <span class="action-user">{{ action.user }}</span>
                  <span class="action-time">{{ formatRelativeTime(action.time) }}</span>
                </div>
              </div>
              <div class="action-status">
                <el-tag
                  :type="action.status === 'success' ? 'success' : 'danger'"
                  size="small"
                >
                  {{ action.status === 'success' ? '成功' : '失败' }}
                </el-tag>
              </div>
            </div>
            
            <div v-if="recentActions.length === 0" class="empty-actions">
              <el-empty description="暂无操作记录" :image-size="60" />
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 快捷操作 -->
      <el-col :xs="24" :lg="10">
        <el-card class="quick-actions-card">
          <template #header>
            <span class="card-title">快捷操作</span>
          </template>
          
          <div class="quick-actions">
            <div class="quick-action-item" @click="$router.push('/nodes')">
              <div class="quick-action-icon">
                <el-icon><Monitor /></el-icon>
              </div>
              <div class="quick-action-content">
                <div class="quick-action-title">节点管理</div>
                <div class="quick-action-desc">查看和管理集群节点</div>
              </div>
            </div>
            
            <div class="quick-action-item" @click="$router.push('/labels')">
              <div class="quick-action-icon">
                <el-icon><CollectionTag /></el-icon>
              </div>
              <div class="quick-action-content">
                <div class="quick-action-title">标签管理</div>
                <div class="quick-action-desc">批量添加和修改节点标签</div>
              </div>
            </div>
            
            <div class="quick-action-item" @click="$router.push('/taints')">
              <div class="quick-action-icon">
                <el-icon><WarningFilled /></el-icon>
              </div>
              <div class="quick-action-content">
                <div class="quick-action-title">污点管理</div>
                <div class="quick-action-desc">配置节点调度策略</div>
              </div>
            </div>
            
            <div
              v-if="hasPermission('admin')"
              class="quick-action-item"
              @click="$router.push('/users')"
            >
              <div class="quick-action-icon">
                <el-icon><User /></el-icon>
              </div>
              <div class="quick-action-content">
                <div class="quick-action-title">用户管理</div>
                <div class="quick-action-desc">管理系统用户和权限</div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/store/modules/auth'
import { useClusterStore } from '@/store/modules/cluster'
import { useNodeStore } from '@/store/modules/node'
import { formatRelativeTime } from '@/utils/format'
import auditApi from '@/api/audit'
import {
  Monitor,
  Check,
  Warning,
  Connection,
  Refresh,
  ArrowRight,
  CollectionTag,
  WarningFilled,
  User,
  Edit,
  Delete,
  Plus
} from '@element-plus/icons-vue'

const authStore = useAuthStore()
const clusterStore = useClusterStore()
const nodeStore = useNodeStore()

// 响应式数据
const loading = ref(false)

// 计算属性
const nodeStats = computed(() => nodeStore.nodeStats)
const clusterStats = computed(() => clusterStore.clusterStats)
const clusters = computed(() => clusterStore.clusters)
const currentCluster = computed(() => clusterStore.currentCluster)
const isAdmin = computed(() => authStore.user?.role === 'admin')

// 节点状态百分比计算
const readyPercentage = computed(() => {
  if (nodeStats.value.total === 0) return 0
  return Math.round((nodeStats.value.ready / nodeStats.value.total) * 100)
})

const notReadyPercentage = computed(() => {
  if (nodeStats.value.total === 0) return 0
  return Math.round((nodeStats.value.notReady / nodeStats.value.total) * 100)
})

const unknownPercentage = computed(() => {
  if (nodeStats.value.total === 0) return 0
  return Math.round((nodeStats.value.unknown / nodeStats.value.total) * 100)
})

// 环形图计算
const radius = 65
const circumference = 2 * Math.PI * radius

// 正常节点段
const readyCircumference = computed(() => {
  if (nodeStats.value.total === 0) return '0 ' + circumference
  const ratio = nodeStats.value.ready / nodeStats.value.total
  const arcLength = ratio * circumference
  return `${arcLength} ${circumference}`
})

const readyOffset = computed(() => 0)

// 异常节点段
const notReadyCircumference = computed(() => {
  if (nodeStats.value.total === 0) return '0 ' + circumference
  const ratio = nodeStats.value.notReady / nodeStats.value.total
  const arcLength = ratio * circumference
  return `${arcLength} ${circumference}`
})

const notReadyOffset = computed(() => {
  if (nodeStats.value.total === 0) return 0
  const readyRatio = nodeStats.value.ready / nodeStats.value.total
  return -(readyRatio * circumference)
})

// 未知节点段
const unknownCircumference = computed(() => {
  if (nodeStats.value.total === 0) return '0 ' + circumference
  const ratio = nodeStats.value.unknown / nodeStats.value.total
  const arcLength = ratio * circumference
  return `${arcLength} ${circumference}`
})

const unknownOffset = computed(() => {
  if (nodeStats.value.total === 0) return 0
  const readyRatio = nodeStats.value.ready / nodeStats.value.total
  const notReadyRatio = nodeStats.value.notReady / nodeStats.value.total
  return -((readyRatio + notReadyRatio) * circumference)
})

// 节点归属图表数据计算
const ownershipChartData = computed(() => {
  const ownership = nodeStats.value.ownership || {}
  const total = nodeStats.value.total
  
  if (total === 0) return []
  
  // 定义颜色数组
  const colors = [
    '#409EFF', // 蓝色
    '#67C23A', // 绿色
    '#E6A23C', // 橙色
    '#F56C6C', // 红色
    '#909399', // 灰色
    '#722ED1', // 紫色
    '#13CE66', // 青绿色
    '#FF6B6B', // 粉红色
    '#4DABF7', // 浅蓝色
    '#69DB7C'  // 浅绿色
  ]
  
  // 转换为图表数据
  const data = Object.entries(ownership).map(([name, count], index) => ({
    name,
    count,
    percentage: Math.round((count / total) * 100),
    color: colors[index % colors.length]
  }))
  
  // 按数量排序
  return data.sort((a, b) => b.count - a.count)
})

// 最近操作数据
const recentActions = ref([])

// 获取最近操作数据
const fetchRecentActions = async () => {
  try {
    const response = await auditApi.getAuditLogs({
      page: 1,
      page_size: 5 // 只获取最近5条记录
    })
    if (response.data && response.data.data && response.data.data.logs) {
      recentActions.value = response.data.data.logs.map(log => ({
        id: log.id,
        type: getActionType(log.action, log.resource_type),
        description: log.details || `${log.action} ${log.resource_type}`,
        user: log.user?.username || log.user?.name || 'Unknown',
        time: new Date(log.created_at),
        status: log.status === 'success' ? 'success' : 'failure'
      }))
    }
  } catch (error) {
    console.error('Failed to fetch recent actions:', error)
    // 如果获取失败，保持空数组
    recentActions.value = []
  }
}

// 根据action和resource_type生成操作类型
const getActionType = (action, resourceType) => {
  const key = `${action}_${resourceType}`.toLowerCase()
  const typeMap = {
    'create_label': 'label_add',
    'update_label': 'label_add', 
    'delete_label': 'label_delete',
    'cordon_node': 'node_cordon',
    'uncordon_node': 'node_cordon',
    'drain_node': 'node_drain',
    'create_taint': 'taint_add',
    'update_taint': 'taint_add',
    'delete_taint': 'taint_delete',
    'create_user': 'user_create',
    'update_user': 'user_update'
  }
  return typeMap[key] || 'default'
}

// 权限检查
const hasPermission = (permission) => {
  return authStore.hasPermission(permission)
}

// 获取操作图标
const getActionIcon = (type) => {
  const iconMap = {
    label_add: Plus,
    label_delete: Delete,
    node_cordon: Warning,
    node_drain: Monitor,
    taint_add: WarningFilled,
    taint_delete: Delete,
    user_create: User,
    user_update: Edit,
    default: Monitor
  }
  return iconMap[type] || Monitor
}

// 获取操作图标样式类
const getActionIconClass = (type) => {
  const classMap = {
    label_add: 'action-icon-success',
    label_delete: 'action-icon-danger',
    node_cordon: 'action-icon-warning',
    node_drain: 'action-icon-info',
    taint_add: 'action-icon-warning',
    taint_delete: 'action-icon-danger',
    user_create: 'action-icon-success',
    user_update: 'action-icon-info',
    default: 'action-icon-info'
  }
  return classMap[type] || 'action-icon-info'
}

// 刷新节点统计
const refreshNodeStats = async () => {
  try {
    await nodeStore.fetchNodes()
    ElMessage.success('节点数据已刷新')
  } catch (error) {
    ElMessage.error('刷新节点数据失败')
  }
}

// 刷新集群数据
const refreshClusters = async () => {
  try {
    await clusterStore.fetchClusters()
    ElMessage.success('集群数据已刷新')
  } catch (error) {
    ElMessage.error('刷新集群数据失败')
  }
}

// 初始化数据
onMounted(async () => {
  try {
    loading.value = true
    
    // 先加载保存的当前集群
    clusterStore.loadCurrentCluster()
    
    // 获取集群列表
    await clusterStore.fetchClusters()
    
    // 如果没有当前集群但有可用集群，自动设置第一个活跃集群为当前集群
    if (!clusterStore.currentCluster && clusterStore.activeClusters.length > 0) {
      clusterStore.setCurrentCluster(clusterStore.activeClusters[0])
    }
    
    // 如果有当前集群，则获取节点数据
    if (clusterStore.currentCluster) {
      await nodeStore.fetchNodes()
    }
    
    // 获取最近操作数据
    await fetchRecentActions()
  } catch (error) {
    console.error('Dashboard data loading error:', error)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.dashboard {
  padding: 0;
}

.page-header {
  margin-bottom: 24px;
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

.stats-row {
  margin-bottom: 24px;
}

.stat-card {
  height: 120px;
  cursor: pointer;
  transition: all 0.3s;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.stat-card-primary {
  border-left: 4px solid #1890ff;
}

.stat-card-success {
  border-left: 4px solid #52c41a;
}

.stat-card-warning {
  border-left: 4px solid #faad14;
}

.stat-card-info {
  border-left: 4px solid #722ed1;
}

.stat-content {
  display: flex;
  align-items: center;
  height: 100%;
}

.stat-icon {
  font-size: 32px;
  margin-right: 16px;
  padding: 12px;
  border-radius: 50%;
  background: rgba(24, 144, 255, 0.1);
  color: #1890ff;
}

.stat-card-success .stat-icon {
  background: rgba(82, 196, 26, 0.1);
  color: #52c41a;
}

.stat-card-warning .stat-icon {
  background: rgba(250, 173, 20, 0.1);
  color: #faad14;
}

.stat-card-info .stat-icon {
  background: rgba(114, 46, 209, 0.1);
  color: #722ed1;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  color: #333;
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  color: #666;
  margin-top: 4px;
}

.content-row {
  margin-bottom: 24px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.more-link {
  color: #1890ff;
  text-decoration: none;
  font-size: 14px;
  display: flex;
  align-items: center;
}

.more-link:hover {
  color: #40a9ff;
}

.chart-container {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 节点状态环形图样式 */
.chart-card {
  height: 380px;
}

.node-status-chart {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 40px;
  padding: 20px;
}

.pie-chart {
  display: flex;
  align-items: center;
  justify-content: center;
}

.pie-svg {
  max-width: 200px;
  max-height: 200px;
}

.status-arc {
  transition: all 0.3s ease;
  cursor: pointer;
}

.status-arc:hover {
  stroke-width: 24;
}

.ready-arc {
  stroke: #67c23a;
}

.notready-arc {
  stroke: #f56c6c;
}

.unknown-arc {
  stroke: #909399;
}

.center-number {
  font-size: 24px;
  font-weight: 600;
  fill: #333;
}

.center-label {
  font-size: 12px;
  fill: #666;
}

.chart-legend {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 120px;
}

.legend-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: 6px;
  background: #fafafa;
  transition: all 0.3s ease;
}

.legend-item:hover {
  background: #f0f0f0;
  transform: translateX(2px);
}

.legend-item.legend-empty {
  opacity: 0.5;
}

.legend-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
}

.legend-color {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.legend-success {
  background-color: #67c23a;
}

.legend-warning {
  background-color: #f56c6c;
}

.legend-info {
  background-color: #909399;
}

.legend-text {
  font-size: 14px;
  color: #333;
  font-weight: 500;
}

.legend-value {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-left: auto;
  margin-right: 4px;
}

.legend-percentage {
  font-size: 12px;
  color: #666;
  font-weight: normal;
}

.empty-chart {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 节点概览容器样式 */
.node-overview-card {
  height: 380px;
}

.node-overview-container {
  display: flex;
  min-height: 280px;
  padding: 20px;
  gap: 20px;
  position: relative;
}

.status-section {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 250px;
  max-width: 280px;
}

.ownership-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 200px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 20px;
  text-align: center;
}

.ownership-section .section-title {
  text-align: left;
  margin-left: 20px;
}

.section-divider {
  height: auto !important;
  min-height: 260px;
  margin: 0 !important;
}

/* 空状态样式 */
.empty-overview {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.9);
  z-index: 10;
}

/* 节点归属图表样式 */
.node-ownership-chart {
  height: 100%;
  flex: 1;
}

.ownership-list {
  height: 220px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
  padding: 0 20px;
}

.ownership-item {
  padding: 12px 16px;
  border-radius: 8px;
  background: #fafafa;
  border: 1px solid #e8e8e8;
  transition: all 0.3s ease;
  animation: fadeInUp 0.5s ease-out;
}

.ownership-item:hover {
  background: #f0f0f0;
  border-color: #d9d9d9;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.ownership-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.ownership-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
}

.ownership-color {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.ownership-name {
  font-size: 13px;
  font-weight: 500;
  color: #333;
}

.ownership-stats {
  display: flex;
  align-items: center;
  gap: 6px;
}

.ownership-count {
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.ownership-percentage {
  font-size: 11px;
  color: #666;
  background: #e8e8e8;
  padding: 1px 5px;
  border-radius: 8px;
}

.ownership-progress {
  height: 6px;
  background: #e8e8e8;
  border-radius: 3px;
  overflow: hidden;
}

.ownership-bar {
  height: 100%;
  transition: width 0.8s ease-out;
  border-radius: 3px;
}

.empty-ownership {
  height: 220px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 20px;
}

/* 响应式设计 - 保持一行显示 */
@media (max-width: 1200px) {
  .status-section {
    min-width: 220px;
    max-width: 250px;
  }
  
  .ownership-section {
    min-width: 180px;
  }
  
  .node-overview-container {
    gap: 16px; /* 减小间距 */
    padding: 16px;
  }
  
  .ownership-section .section-title {
    margin-left: 10px; /* 调整边距 */
  }
}

@media (max-width: 992px) {
  .node-overview-card,
  .cluster-card {
    height: auto; /* 在中等屏幕取消固定高度 */
  }
  
  .node-overview-container {
    min-height: 240px;
    padding: 15px;
    gap: 12px;
  }
  
  .status-section {
    min-width: 200px;
    max-width: 220px;
  }
  
  .ownership-section {
    min-width: 160px;
  }
  
  .ownership-list {
    padding: 0 10px;
    height: 180px; /* 减小高度以适应紧凑布局 */
  }
  
  .section-title {
    font-size: 14px;
    margin-bottom: 15px;
  }
  
  .cluster-list {
    max-height: 280px;
  }
}

@media (max-width: 768px) {
  .node-overview-container {
    flex-direction: column; /* 小屏幕堆叠显示 */
    gap: 15px;
    padding: 12px;
    min-height: auto;
  }
  
  .section-divider {
    display: none;
  }
  
  .status-section,
  .ownership-section {
    min-width: auto;
    max-width: none;
  }
  
  .ownership-section .section-title {
    text-align: center;
    margin-left: 0;
  }
  
  .ownership-list {
    height: 160px;
  }
  
  .cluster-list {
    max-height: 240px;
  }
}

/* 动画 */
@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.cluster-card {
  height: 380px;
}

.cluster-list {
  max-height: 300px;
  overflow-y: auto;
}

.cluster-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-radius: 6px;
  margin-bottom: 8px;
  transition: background-color 0.3s;
}

.cluster-item:hover {
  background: #f5f5f5;
}

.cluster-item.active {
  background: #e6f7ff;
  border: 1px solid #91d5ff;
}

.cluster-info {
  flex: 1;
}

.cluster-name {
  font-weight: 500;
  color: #333;
  margin-bottom: 4px;
}

.cluster-desc {
  font-size: 12px;
  color: #666;
}

.recent-actions {
  max-height: 400px;
  overflow-y: auto;
}

.action-item {
  display: flex;
  align-items: center;
  padding: 12px;
  border-radius: 6px;
  margin-bottom: 8px;
  transition: background-color 0.3s;
}

.action-item:hover {
  background: #f5f5f5;
}

.action-icon {
  font-size: 16px;
  margin-right: 12px;
  padding: 8px;
  border-radius: 50%;
}

.action-icon-success {
  background: rgba(82, 196, 26, 0.1);
  color: #52c41a;
}

.action-icon-danger {
  background: rgba(255, 77, 79, 0.1);
  color: #ff4d4f;
}

.action-icon-warning {
  background: rgba(250, 173, 20, 0.1);
  color: #faad14;
}

.action-icon-info {
  background: rgba(24, 144, 255, 0.1);
  color: #1890ff;
}

.action-icon-default {
  background: rgba(0, 0, 0, 0.06);
  color: #666;
}

.action-content {
  flex: 1;
}

.action-title {
  font-weight: 500;
  color: #333;
  margin-bottom: 4px;
}

.action-meta {
  font-size: 12px;
  color: #666;
  display: flex;
  gap: 12px;
}

.quick-actions {
  display: grid;
  gap: 12px;
}

.quick-action-item {
  display: flex;
  align-items: center;
  padding: 16px;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s;
}

.quick-action-item:hover {
  border-color: #1890ff;
  background: #fafafa;
  transform: translateY(-1px);
}

.quick-action-icon {
  font-size: 20px;
  color: #1890ff;
  margin-right: 12px;
}

.quick-action-content {
  flex: 1;
}

.quick-action-title {
  font-weight: 500;
  color: #333;
  margin-bottom: 4px;
}

.quick-action-desc {
  font-size: 12px;
  color: #666;
}

.empty-clusters,
.empty-actions {
  padding: 20px;
  text-align: center;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .stats-row .el-col {
    margin-bottom: 16px;
  }
  
  .content-row .el-col {
    margin-bottom: 16px;
  }
  
  /* 环形图响应式 */
  .node-status-chart {
    flex-direction: column;
    gap: 20px;
    padding: 10px;
  }
  
  .pie-svg {
    max-width: 160px;
    max-height: 160px;
  }
  
  .center-number {
    font-size: 20px;
  }
  
  .center-label {
    font-size: 11px;
  }
  
  .chart-legend {
    min-width: auto;
    width: 100%;
  }
}
</style>