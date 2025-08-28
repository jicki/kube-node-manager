<template>
  <div class="layout">
    <el-container class="layout-container">
      <!-- 侧边栏 -->
      <el-aside 
        :width="sidebarCollapsed ? '64px' : '250px'" 
        class="layout-sidebar"
      >
        <Sidebar 
          :collapsed="sidebarCollapsed" 
          @toggle-collapse="toggleSidebar"
        />
      </el-aside>
      
      <!-- 主体内容 -->
      <el-container>
        <!-- 头部 -->
        <el-header class="layout-header">
          <Header 
            :collapsed="sidebarCollapsed"
            @toggle-sidebar="toggleSidebar"
          />
        </el-header>
        
        <!-- 内容区域 -->
        <el-main class="layout-main">
          <div class="main-content">
            <router-view />
          </div>
        </el-main>
      </el-container>
    </el-container>
    
    <!-- 全局加载遮罩 -->
    <el-overlay v-if="loading" class="global-loading">
      <div class="loading-content">
        <el-icon class="loading-icon">
          <Loading />
        </el-icon>
        <p>加载中...</p>
      </div>
    </el-overlay>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '@/store/modules/auth'
import { useClusterStore } from '@/store/modules/cluster'
import Sidebar from './Sidebar.vue'
import Header from './Header.vue'
import { Loading } from '@element-plus/icons-vue'

const authStore = useAuthStore()
const clusterStore = useClusterStore()

// 侧边栏折叠状态
const sidebarCollapsed = ref(false)

// 全局加载状态
const loading = computed(() => {
  return clusterStore.loading
})

// 切换侧边栏折叠状态
const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  // 保存到本地存储
  localStorage.setItem('sidebarCollapsed', sidebarCollapsed.value)
}

// 响应式处理
const handleResize = () => {
  if (window.innerWidth <= 768) {
    sidebarCollapsed.value = true
  }
}

onMounted(() => {
  // 从本地存储恢复侧边栏状态
  const saved = localStorage.getItem('sidebarCollapsed')
  if (saved !== null) {
    sidebarCollapsed.value = JSON.parse(saved)
  }
  
  // 响应式处理
  handleResize()
  window.addEventListener('resize', handleResize)
  
  // 加载当前集群信息
  clusterStore.loadCurrentCluster()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.layout {
  height: 100vh;
  overflow: hidden;
}

.layout-container {
  height: 100%;
}

.layout-sidebar {
  background: #001529;
  transition: width 0.3s ease;
  overflow: hidden;
}

.layout-header {
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  padding: 0;
  height: 64px;
  display: flex;
  align-items: center;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
  z-index: 10;
}

.layout-main {
  background: #f0f2f5;
  padding: 24px;
  overflow-y: auto;
  height: calc(100vh - 64px);
}

.main-content {
  min-height: 100%;
}

.global-loading {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(2px);
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #409eff;
}

.loading-icon {
  font-size: 32px;
  margin-bottom: 12px;
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 响应式布局 */
@media (max-width: 768px) {
  .layout-main {
    padding: 16px;
  }
  
  .layout-sidebar {
    position: fixed;
    top: 0;
    left: 0;
    height: 100vh;
    z-index: 1000;
  }
  
  .layout-sidebar:not(.collapsed) + .el-container {
    margin-left: 250px;
  }
}

@media (max-width: 480px) {
  .layout-main {
    padding: 12px;
  }
}
</style>