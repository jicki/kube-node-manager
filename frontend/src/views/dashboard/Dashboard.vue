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

    <!-- 图表和详细信息 -->
    <el-row :gutter="24" class="content-row">
      <!-- 节点状态图表 -->
      <el-col :xs="24" :lg="12">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">节点状态分布</span>
              <el-button type="text" size="small" @click="refreshNodeStats">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>
          
          <div class="chart-container" style="height: 300px;">
            <!-- 这里可以集成图表库如ECharts -->
            <div class="pie-chart-placeholder">
              <div class="chart-legend">
                <div class="legend-item">
                  <span class="legend-color legend-success"></span>
                  <span class="legend-text">正常 ({{ nodeStats.ready }})</span>
                </div>
                <div class="legend-item">
                  <span class="legend-color legend-warning"></span>
                  <span class="legend-text">异常 ({{ nodeStats.notReady }})</span>
                </div>
                <div class="legend-item">
                  <span class="legend-color legend-info"></span>
                  <span class="legend-text">未知 ({{ nodeStats.unknown }})</span>
                </div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 集群列表 -->
      <el-col :xs="24" :lg="12">
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

// 模拟最近操作数据
const recentActions = ref([
  {
    id: 1,
    type: 'label_add',
    description: '为节点 worker-01 添加标签 env=production',
    user: 'admin',
    time: new Date(Date.now() - 5 * 60 * 1000),
    status: 'success'
  },
  {
    id: 2,
    type: 'node_cordon',
    description: '封锁节点 worker-02',
    user: 'operator',
    time: new Date(Date.now() - 15 * 60 * 1000),
    status: 'success'
  },
  {
    id: 3,
    type: 'taint_add',
    description: '为节点 master-01 添加污点',
    user: 'admin',
    time: new Date(Date.now() - 30 * 60 * 1000),
    status: 'failure'
  }
])

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
    user_update: Edit
  }
  return iconMap[type] || Edit
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
    user_update: 'action-icon-info'
  }
  return classMap[type] || 'action-icon-default'
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

.pie-chart-placeholder {
  text-align: center;
}

.chart-legend {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.legend-color {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.legend-success {
  background: #52c41a;
}

.legend-warning {
  background: #faad14;
}

.legend-info {
  background: #1890ff;
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
}
</style>