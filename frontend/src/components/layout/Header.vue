<template>
  <div class="header">
    <!-- 左侧区域 -->
    <div class="header-left">
      <!-- 侧边栏切换按钮（移动端） -->
      <el-button
        v-if="isMobile"
        type="text"
        class="sidebar-toggle"
        @click="toggleSidebar"
      >
        <el-icon><Menu /></el-icon>
      </el-button>
      
      <!-- 面包屑导航 -->
      <el-breadcrumb separator="/" class="breadcrumb">
        <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
        <el-breadcrumb-item v-if="currentPage.title">
          {{ currentPage.title }}
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>
    
    <!-- 右侧区域 -->
    <div class="header-right">
      <!-- 集群状态指示器 -->
      <div v-if="currentCluster" class="cluster-status">
        <el-tag 
          :type="currentCluster.status === 'active' ? 'success' : 'danger'"
          size="small"
        >
          <el-icon class="cluster-icon">
            <Connection />
          </el-icon>
          {{ currentCluster.name }}
        </el-tag>
      </div>
      
      <!-- 通知中心 -->
      <el-popover placement="bottom" width="320" trigger="click">
        <template #reference>
          <el-badge :value="notificationCount" :hidden="notificationCount === 0">
            <el-button type="text" class="notification-btn">
              <el-icon><Bell /></el-icon>
            </el-button>
          </el-badge>
        </template>
        
        <div class="notification-panel">
          <div class="notification-header">
            <span>通知中心</span>
            <el-button type="text" size="small" @click="clearAllNotifications">
              清空
            </el-button>
          </div>
          
          <div class="notification-list">
            <div 
              v-for="notification in notifications" 
              :key="notification.id"
              class="notification-item"
            >
              <div class="notification-content">
                <div class="notification-title">{{ notification.title }}</div>
                <div class="notification-desc">{{ notification.message }}</div>
                <div class="notification-time">{{ formatTime(notification.time) }}</div>
              </div>
              <el-button 
                type="text" 
                size="small"
                @click="removeNotification(notification.id)"
              >
                <el-icon><Close /></el-icon>
              </el-button>
            </div>
            
            <div v-if="notifications.length === 0" class="no-notifications">
              暂无通知
            </div>
          </div>
        </div>
      </el-popover>
      
      <!-- 用户信息下拉菜单 -->
      <el-dropdown @command="handleUserCommand">
        <div class="user-info">
          <el-avatar :size="32" class="user-avatar">
            <el-icon><User /></el-icon>
          </el-avatar>
          <span v-if="!isMobile" class="username">{{ userInfo.username }}</span>
          <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
        </div>
        
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="profile">
              <el-icon><User /></el-icon>
              个人信息
            </el-dropdown-item>
            <el-dropdown-item command="changePassword">
              <el-icon><Lock /></el-icon>
              修改密码
            </el-dropdown-item>
            <el-dropdown-item divided command="logout">
              <el-icon><SwitchButton /></el-icon>
              退出登录
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
    
    <!-- 修改密码对话框 -->
    <el-dialog
      v-model="changePasswordVisible"
      title="修改密码"
      width="400px"
    >
      <el-form
        ref="changePasswordFormRef"
        :model="changePasswordForm"
        :rules="changePasswordRules"
        label-width="80px"
      >
        <el-form-item label="当前密码" prop="oldPassword">
          <el-input
            v-model="changePasswordForm.oldPassword"
            type="password"
            show-password
          />
        </el-form-item>
        <el-form-item label="新密码" prop="newPassword">
          <el-input
            v-model="changePasswordForm.newPassword"
            type="password"
            show-password
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="changePasswordForm.confirmPassword"
            type="password"
            show-password
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="changePasswordVisible = false">取消</el-button>
        <el-button type="primary" @click="handleChangePassword">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/store/modules/auth'
import { useClusterStore } from '@/store/modules/cluster'
import { formatTime } from '@/utils/format'
import authApi from '@/api/auth'
import {
  Menu,
  Bell,
  User,
  ArrowDown,
  Close,
  Lock,
  SwitchButton,
  Connection
} from '@element-plus/icons-vue'

const props = defineProps({
  collapsed: Boolean
})

const emit = defineEmits(['toggle-sidebar'])

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const clusterStore = useClusterStore()

// 响应式状态
const isMobile = ref(false)
const changePasswordVisible = ref(false)
const changePasswordFormRef = ref()

// 用户信息
const userInfo = computed(() => authStore.userInfo || {})
const currentCluster = computed(() => clusterStore.currentCluster)

// 当前页面信息
const currentPage = computed(() => {
  const meta = route.meta || {}
  return {
    title: meta.title || ''
  }
})

// 通知相关
const notifications = ref([])

const notificationCount = computed(() => notifications.value.length)

// 修改密码表单
const changePasswordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const changePasswordRules = {
  oldPassword: [
    { required: true, message: '请输入当前密码', trigger: 'blur' }
  ],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少为6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== changePasswordForm.newPassword) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

// 切换侧边栏
const toggleSidebar = () => {
  emit('toggle-sidebar')
}

// 处理用户下拉菜单命令
const handleUserCommand = (command) => {
  switch (command) {
    case 'profile':
      // 打开个人信息页面
      break
    case 'changePassword':
      changePasswordVisible.value = true
      break
    case 'logout':
      handleLogout()
      break
  }
}

// 处理修改密码
const handleChangePassword = async () => {
  try {
    await changePasswordFormRef.value.validate()
    
    await authApi.changePassword({
      oldPassword: changePasswordForm.oldPassword,
      newPassword: changePasswordForm.newPassword
    })
    
    ElMessage.success('密码修改成功，请重新登录')
    changePasswordVisible.value = false
    
    // 重置表单
    Object.assign(changePasswordForm, {
      oldPassword: '',
      newPassword: '',
      confirmPassword: ''
    })
    
    // 延迟退出登录
    setTimeout(() => {
      handleLogout()
    }, 1500)
    
  } catch (error) {
    ElMessage.error(error.message || '密码修改失败')
  }
}

// 处理退出登录
const handleLogout = () => {
  ElMessageBox.confirm('确认退出登录吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    authStore.logout()
    router.push('/login')
    ElMessage.success('已退出登录')
  }).catch(() => {
    // 用户取消
  })
}

// 清除所有通知
const clearAllNotifications = () => {
  notifications.value = []
}

// 移除单个通知
const removeNotification = (id) => {
  const index = notifications.value.findIndex(n => n.id === id)
  if (index !== -1) {
    notifications.value.splice(index, 1)
  }
}

// 响应式处理
const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => {
  handleResize()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 24px;
  height: 100%;
  background: #fff;
}

.header-left {
  display: flex;
  align-items: center;
}

.sidebar-toggle {
  margin-right: 16px;
  padding: 8px;
}

.breadcrumb {
  font-size: 14px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.cluster-status {
  display: flex;
  align-items: center;
}

.cluster-icon {
  margin-right: 4px;
}

.notification-btn {
  padding: 8px;
}

.notification-panel {
  padding: 0;
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  font-weight: 600;
}

.notification-list {
  max-height: 300px;
  overflow-y: auto;
}

.notification-item {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.notification-item:hover {
  background: #f5f5f5;
}

.notification-content {
  flex: 1;
}

.notification-title {
  font-weight: 500;
  margin-bottom: 4px;
}

.notification-desc {
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
}

.notification-time {
  font-size: 12px;
  color: #999;
}

.no-notifications {
  text-align: center;
  padding: 24px;
  color: #999;
}

.user-info {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.user-info:hover {
  background-color: #f0f2f5;
}

.user-avatar {
  margin-right: 8px;
}

.username {
  margin-right: 8px;
  font-size: 14px;
  color: #333;
}

.dropdown-icon {
  font-size: 12px;
  color: #999;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .header {
    padding: 0 16px;
  }
  
  .cluster-status {
    display: none;
  }
  
  .username {
    display: none;
  }
}
</style>