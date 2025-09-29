<template>
  <div class="monitoring-overview">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">监控概览</h1>
        <p class="page-description">查看集群监控状态和基础指标信息</p>
      </div>
      <div class="header-right">
        <el-button @click="checkMonitoringStatus">
          <el-icon><Refresh /></el-icon>
          刷新状态
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

    <!-- 监控概览内容 -->
    <div v-else class="monitoring-content">
      <!-- 监控系统状态卡片 -->
      <div class="status-cards">
        <el-card class="status-card">
          <div class="status-content">
            <div class="status-icon success">
              <el-icon><Check /></el-icon>
            </div>
            <div class="status-info">
              <div class="status-title">监控系统状态</div>
              <div class="status-value">正常运行</div>
              <div class="status-detail">{{ currentCluster?.monitoring_type || 'Prometheus' }}</div>
            </div>
          </div>
        </el-card>

        <el-card class="status-card">
          <div class="status-content">
            <div class="status-icon info">
              <el-icon><Monitor /></el-icon>
            </div>
            <div class="status-info">
              <div class="status-title">监控节点数量</div>
              <div class="status-value">{{ nodeMetrics.total || 0 }}</div>
              <div class="status-detail">已配置 node-exporter</div>
            </div>
          </div>
        </el-card>

        <el-card class="status-card">
          <div class="status-content">
            <div class="status-icon warning">
              <el-icon><DataLine /></el-icon>
            </div>
            <div class="status-info">
              <div class="status-title">采集间隔</div>
              <div class="status-value">15s</div>
              <div class="status-detail">数据更新频率</div>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 快速操作 -->
      <div class="quick-actions">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>快速操作</span>
            </div>
          </template>
          <div class="action-buttons">
            <el-button
              type="primary"
              @click="$router.push('/monitoring/nodes')"
            >
              <el-icon><Monitor /></el-icon>
              查看节点监控
            </el-button>
            <el-button
              type="success"
              @click="$router.push('/monitoring/network')"
            >
              <el-icon><Connection /></el-icon>
              网络拓扑图
            </el-button>
            <el-button
              @click="testNetworkConnectivity"
              :loading="testingNetwork"
            >
              <el-icon><Monitor /></el-icon>
              网络连通性测试
            </el-button>
          </div>
        </el-card>
      </div>

      <!-- 最近告警 -->
      <div class="recent-alerts">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>最近告警</span>
              <el-button type="text" size="small">查看全部</el-button>
            </div>
          </template>
          <div v-if="alerts.length === 0" class="no-alerts">
            <el-result
              icon="success"
              title="暂无告警"
              sub-title="系统运行正常"
            />
          </div>
          <div v-else class="alerts-list">
            <div v-for="alert in alerts" :key="alert.id" class="alert-item">
              <div class="alert-severity" :class="alert.severity">
                <el-icon><Warning /></el-icon>
              </div>
              <div class="alert-content">
                <div class="alert-title">{{ alert.title }}</div>
                <div class="alert-time">{{ formatTime(alert.time) }}</div>
              </div>
            </div>
          </div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useClusterStore } from '@/store/modules/cluster'
import { formatTime } from '@/utils/format'
import {
  Refresh,
  Setting,
  Check,
  Monitor,
  DataLine,
  Connection,
  Warning
} from '@element-plus/icons-vue'

const router = useRouter()
const clusterStore = useClusterStore()

// 响应式数据
const loading = ref(false)
const testingNetwork = ref(false)
const nodeMetrics = ref({
  total: 0,
  healthy: 0,
  warning: 0
})
const alerts = ref([])

// 计算属性
const currentCluster = computed(() => clusterStore.currentCluster)
const monitoringConfigured = computed(() => {
  const cluster = currentCluster.value
  console.log('Computing monitoringConfigured for cluster:', cluster)
  return cluster?.monitoring_enabled === true
})

// 检查监控状态
const checkMonitoringStatus = async () => {
  // 调试输出当前集群信息
  console.log('Current cluster:', currentCluster.value)
  console.log('Monitoring configured:', monitoringConfigured.value)

  if (!currentCluster.value) {
    console.log('No current cluster selected')
    return
  }

  try {
    loading.value = true

    // 始终调用监控状态 API，不管配置如何
    const response = await fetch(`/api/v1/clusters/${currentCluster.value.id}/monitoring/status`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      }
    })

    if (response.ok) {
      const data = await response.json()
      console.log('Monitoring status response:', data)

      if (data.data && data.data.enabled) {
        // 更新节点指标数据
        nodeMetrics.value = {
          total: currentCluster.value?.node_count || 0,
          healthy: currentCluster.value?.node_count || 0,
          warning: 0
        }
      }
    } else {
      console.error('Failed to fetch monitoring status:', response.status)
    }

  } catch (error) {
    console.error('Failed to check monitoring status:', error)
    ElMessage.error('获取监控状态失败')
  } finally {
    loading.value = false
  }
}

// 跳转到集群管理
const goToClusterManage = () => {
  router.push('/clusters')
}

// 测试网络连通性
const testNetworkConnectivity = async () => {
  if (!monitoringConfigured.value) {
    ElMessage.warning('请先配置监控系统')
    return
  }

  try {
    testingNetwork.value = true
    // 这里可以调用API进行网络连通性测试
    console.log('Testing network connectivity...')

    // 模拟测试延迟
    await new Promise(resolve => setTimeout(resolve, 2000))

    ElMessage.success('网络连通性测试完成')

  } catch (error) {
    console.error('Network connectivity test failed:', error)
    ElMessage.error('网络连通性测试失败')
  } finally {
    testingNetwork.value = false
  }
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
    // 然后检查监控状态
    checkMonitoringStatus()
  })
})
</script>

<style scoped>
.monitoring-overview {
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

.status-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.status-card {
  cursor: pointer;
  transition: all 0.3s ease;
}

.status-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.status-content {
  display: flex;
  align-items: center;
  padding: 8px;
}

.status-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 24px;
}

.status-icon.success {
  background: linear-gradient(135deg, #52c41a 0%, #389e0d 100%);
  color: white;
}

.status-icon.info {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  color: white;
}

.status-icon.warning {
  background: linear-gradient(135deg, #faad14 0%, #d48806 100%);
  color: white;
}

.status-info {
  flex: 1;
}

.status-title {
  font-size: 14px;
  color: #666;
  margin-bottom: 4px;
}

.status-value {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  line-height: 1;
  margin-bottom: 4px;
}

.status-detail {
  font-size: 12px;
  color: #999;
}

.quick-actions,
.recent-alerts {
  margin-bottom: 24px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.no-alerts {
  padding: 20px 0;
}

.alerts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.alert-item {
  display: flex;
  align-items: center;
  padding: 12px;
  background: #fafafa;
  border-radius: 6px;
}

.alert-severity {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  font-size: 16px;
}

.alert-severity.warning {
  background: #faad14;
  color: white;
}

.alert-severity.error {
  background: #ff4d4f;
  color: white;
}

.alert-content {
  flex: 1;
}

.alert-title {
  font-size: 14px;
  color: #333;
  margin-bottom: 4px;
}

.alert-time {
  font-size: 12px;
  color: #999;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .status-cards {
    grid-template-columns: 1fr;
  }

  .action-buttons {
    flex-direction: column;
    gap: 8px;
  }
}
</style>