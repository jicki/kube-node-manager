<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useAuthStore } from '@/store/modules/auth'
import { useClusterStore } from '@/store/modules/cluster'

const authStore = useAuthStore()
const clusterStore = useClusterStore()

onMounted(async () => {
  // 初始化认证信息
  if (authStore.token) {
    try {
      await authStore.getUserInfo()
      // 加载当前集群信息
      clusterStore.loadCurrentCluster()
    } catch (error) {
      console.error('Initialize auth failed:', error)
      authStore.logout()
    }
  }
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background-color: #f0f2f5;
}

#app {
  min-height: 100vh;
}

.page-container {
  padding: 20px;
}

.card-container {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  padding: 24px;
  margin-bottom: 16px;
}

.el-table {
  margin-top: 20px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  gap: 16px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.status-active {
  background-color: #f0f9ff;
  color: #0369a1;
}

.status-inactive {
  background-color: #fef2f2;
  color: #dc2626;
}

.role-badge {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
}

.role-admin {
  background-color: #fef3c7;
  color: #d97706;
}

.role-user {
  background-color: #dbeafe;
  color: #2563eb;
}

.role-viewer {
  background-color: #f3f4f6;
  color: #6b7280;
}

@media (max-width: 768px) {
  .page-container {
    padding: 12px;
  }
  
  .card-container {
    padding: 16px;
  }
  
  .toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }
  
  .toolbar-left,
  .toolbar-right {
    justify-content: center;
  }
}
</style>