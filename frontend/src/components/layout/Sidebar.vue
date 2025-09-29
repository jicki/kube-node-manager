<template>
  <div class="sidebar">
    <!-- Logo区域 -->
    <div class="logo-container">
      <div class="logo">
        <el-icon v-if="collapsed" class="logo-icon">
          <Monitor />
        </el-icon>
        <template v-else>
          <el-icon class="logo-icon">
            <Monitor />
          </el-icon>
          <span class="logo-text">K8s节点管理</span>
        </template>
      </div>
    </div>
    
    <!-- 集群切换器 -->
    <div v-if="!collapsed" class="cluster-selector">
      <el-select
        v-model="currentClusterId"
        placeholder="选择集群"
        size="small"
        filterable
        clearable
        remote
        :remote-method="handleClusterSearch"
        :loading="clusterSearchLoading"
        @change="handleClusterChange"
      >
        <el-option
          v-for="cluster in filteredClusters"
          :key="cluster.id"
          :label="cluster.name"
          :value="cluster.id"
        >
          <div class="cluster-option">
            <span class="cluster-name">{{ cluster.name }}</span>
            <el-tag
              :type="cluster.status === 'active' ? 'success' : 'danger'"
              size="small"
            >
              {{ cluster.status === 'active' ? '正常' : '异常' }}
            </el-tag>
          </div>
        </el-option>
      </el-select>
    </div>
    
    <!-- 导航菜单 -->
    <el-menu
      :default-active="activeMenu"
      :default-openeds="defaultOpeneds"
      :collapse="collapsed"
      :unique-opened="false"
      background-color="#001529"
      text-color="rgba(255, 255, 255, 0.65)"
      active-text-color="#1890ff"
      class="sidebar-menu"
      router
    >
      <!-- 节点管理 -->
      <el-sub-menu index="node-management">
        <template #title>
          <el-icon><Monitor /></el-icon>
          <span>节点管理</span>
        </template>

        <el-menu-item index="/dashboard">
          <el-icon><Monitor /></el-icon>
          <template #title>概览</template>
        </el-menu-item>

        <el-menu-item index="/nodes">
          <el-icon><Monitor /></el-icon>
          <template #title>节点管理</template>
        </el-menu-item>

        <el-menu-item index="/labels">
          <el-icon><CollectionTag /></el-icon>
          <template #title>标签管理</template>
        </el-menu-item>

        <el-menu-item index="/taints">
          <el-icon><WarningFilled /></el-icon>
          <template #title>污点管理</template>
        </el-menu-item>
      </el-sub-menu>

      <!-- 监控信息 -->
      <el-sub-menu index="monitoring">
        <template #title>
          <el-icon><DataLine /></el-icon>
          <span>监控信息</span>
        </template>

        <el-menu-item index="/monitoring/overview">
          <el-icon><Monitor /></el-icon>
          <template #title>监控概览</template>
        </el-menu-item>

        <el-menu-item index="/monitoring/nodes">
          <el-icon><Monitor /></el-icon>
          <template #title>节点监控</template>
        </el-menu-item>

        <el-menu-item index="/monitoring/network">
          <el-icon><Connection /></el-icon>
          <template #title>网络检测</template>
        </el-menu-item>
      </el-sub-menu>

      <!-- 系统配置 -->
      <el-sub-menu index="system-config">
        <template #title>
          <el-icon><Setting /></el-icon>
          <span>系统配置</span>
        </template>

        <el-menu-item index="/clusters">
          <el-icon><Connection /></el-icon>
          <template #title>集群管理</template>
        </el-menu-item>

        <el-menu-item index="/audit">
          <el-icon><DocumentCopy /></el-icon>
          <template #title>审计日志</template>
        </el-menu-item>

        <el-menu-item
          v-if="hasPermission('admin')"
          index="/users"
        >
          <el-icon><User /></el-icon>
          <template #title>用户管理</template>
        </el-menu-item>
      </el-sub-menu>
    </el-menu>
    
    <!-- 折叠按钮 -->
    <div class="collapse-toggle" @click="toggleCollapse">
      <el-icon>
        <ArrowLeft v-if="!collapsed" />
        <ArrowRight v-if="collapsed" />
      </el-icon>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/store/modules/auth'
import { useClusterStore } from '@/store/modules/cluster'
import {
  Monitor,
  CollectionTag,
  WarningFilled,
  User,
  ArrowLeft,
  ArrowRight,
  Connection,
  DocumentCopy,
  Setting,
  DataLine
} from '@element-plus/icons-vue'

const props = defineProps({
  collapsed: Boolean
})

const emit = defineEmits(['toggle-collapse'])

const route = useRoute()
const authStore = useAuthStore()
const clusterStore = useClusterStore()

// 当前选中的菜单
const activeMenu = computed(() => route.path)

// 默认展开的子菜单
const defaultOpeneds = computed(() => {
  const path = route.path
  const openedMenus = []

  // 根据当前路径确定应该展开的子菜单
  if (['/dashboard', '/nodes', '/labels', '/taints'].includes(path)) {
    openedMenus.push('node-management')
  }

  if (path.startsWith('/monitoring')) {
    openedMenus.push('monitoring')
  }

  if (['/clusters', '/audit', '/users'].includes(path)) {
    openedMenus.push('system-config')
  }

  return openedMenus
})

// 集群列表和当前集群
const clusters = computed(() => clusterStore.clusters)
const currentClusterId = ref(clusterStore.currentCluster?.id)
const filteredClusters = ref([])
const clusterSearchLoading = ref(false)

// 权限检查
const hasPermission = (permission) => {
  return authStore.hasPermission(permission)
}

// 切换折叠状态
const toggleCollapse = () => {
  emit('toggle-collapse')
}

// 处理集群搜索
const handleClusterSearch = (query) => {
  clusterSearchLoading.value = true
  setTimeout(() => {
    if (query) {
      filteredClusters.value = clusters.value.filter(cluster =>
        cluster.name.toLowerCase().includes(query.toLowerCase())
      )
    } else {
      filteredClusters.value = [...clusters.value]
    }
    clusterSearchLoading.value = false
  }, 200)
}

// 处理集群切换
const handleClusterChange = async (clusterId) => {
  const cluster = clusters.value.find(c => c.id === clusterId)
  if (cluster) {
    clusterStore.setCurrentCluster(cluster)
    ElMessage.success(`已切换到集群: ${cluster.name}`)
    
    // 切换集群后立即刷新相关数据
    try {
      // 导入nodeStore来刷新节点数据
      const { useNodeStore } = await import('@/store/modules/node')
      const nodeStore = useNodeStore()
      
      // 清空当前节点数据和禁止调度历史，然后重新获取
      nodeStore.clearCordonHistories()
      await nodeStore.fetchNodes()
      
      // 如果当前页面是Dashboard，触发Dashboard数据刷新
      if (route.path === '/dashboard') {
        // 通过事件总线通知Dashboard刷新
        window.dispatchEvent(new CustomEvent('cluster-changed', { 
          detail: { cluster } 
        }))
      }
    } catch (error) {
      console.warn('Failed to refresh data after cluster switch:', error)
    }
  }
}

// 监听当前集群变化
watch(
  () => clusterStore.currentCluster,
  (newCluster) => {
    if (newCluster) {
      currentClusterId.value = newCluster.id
    }
  },
  { immediate: true }
)

// 监听集群列表变化，初始化过滤列表
watch(
  () => clusters.value,
  (newClusters) => {
    filteredClusters.value = [...newClusters]
  },
  { immediate: true }
)

onMounted(() => {
  // 加载集群列表
  clusterStore.fetchClusters()
})
</script>

<style scoped>
.sidebar {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #001529;
}

.logo-container {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid #002140;
}

.logo {
  display: flex;
  align-items: center;
  color: #fff;
  font-size: 18px;
  font-weight: 600;
}

.logo-icon {
  font-size: 24px;
  color: #1890ff;
  margin-right: 12px;
}

.logo-text {
  white-space: nowrap;
}

.cluster-selector {
  padding: 16px 12px;
  border-bottom: 1px solid #002140;
}

.cluster-selector :deep(.el-select) {
  width: 100%;
}

.cluster-selector :deep(.el-input__wrapper) {
  background: #002140;
  border-color: #0050b3;
  color: rgba(255, 255, 255, 0.85);
}

.cluster-selector :deep(.el-input__inner) {
  color: rgba(255, 255, 255, 0.85);
}

.cluster-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cluster-name {
  margin-right: 8px;
}

.sidebar-menu {
  flex: 1;
  border: none;
}

.sidebar-menu :deep(.el-menu-item) {
  height: 50px;
  line-height: 50px;
}

.sidebar-menu :deep(.el-menu-item:hover) {
  background-color: #1890ff !important;
  color: #fff !important;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background-color: #1890ff !important;
  color: #fff !important;
}

.sidebar-menu :deep(.el-sub-menu__title) {
  height: 50px;
  line-height: 50px;
  color: rgba(255, 255, 255, 0.65);
}

.sidebar-menu :deep(.el-sub-menu__title:hover) {
  background-color: #1890ff !important;
  color: #fff !important;
}

.sidebar-menu :deep(.el-sub-menu.is-active .el-sub-menu__title) {
  color: #1890ff !important;
}

.sidebar-menu :deep(.el-sub-menu .el-menu-item) {
  background-color: #000c17 !important;
  padding-left: 50px !important;
}

.sidebar-menu :deep(.el-sub-menu .el-menu-item:hover) {
  background-color: #1890ff !important;
  color: #fff !important;
}

.collapse-toggle {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.65);
  cursor: pointer;
  border-top: 1px solid #002140;
  transition: all 0.3s;
}

.collapse-toggle:hover {
  color: #1890ff;
  background-color: #002140;
}

/* 折叠状态下的样式调整 */
.sidebar.collapsed .cluster-selector {
  display: none;
}
</style>