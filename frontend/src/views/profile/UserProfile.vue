<template>
  <div class="profile-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">个人信息</h1>
        <p class="page-description">查看和管理您的个人账户信息</p>
      </div>
    </div>

    <div class="profile-content">
      <el-row :gutter="24">
        <!-- 个人信息卡片 -->
        <el-col :span="16">
          <el-card class="profile-card">
            <template #header>
              <div class="card-header">
                <span class="card-title">基本信息</span>
                <el-button
                  v-if="!editing"
                  type="primary"
                  size="small"
                  @click="startEdit"
                >
                  编辑信息
                </el-button>
                <div v-else class="edit-buttons">
                  <el-button size="small" @click="cancelEdit">取消</el-button>
                  <el-button
                    type="primary"
                    size="small"
                    :loading="updateLoading"
                    @click="saveProfile"
                  >
                    保存
                  </el-button>
                </div>
              </div>
            </template>

            <el-form
              ref="profileFormRef"
              :model="profileForm"
              :rules="profileRules"
              label-width="100px"
            >
              <el-form-item label="用户名">
                <el-input
                  v-model="profileForm.username"
                  disabled
                  placeholder="用户名不可修改"
                />
              </el-form-item>

              <el-form-item label="邮箱" prop="email">
                <el-input
                  v-model="profileForm.email"
                  :disabled="!editing"
                  placeholder="请输入邮箱地址"
                />
              </el-form-item>

              <el-form-item label="角色">
                <el-tag
                  :type="getRoleTagType(profileForm.role)"
                  size="large"
                >
                  {{ getRoleDisplayName(profileForm.role) }}
                </el-tag>
              </el-form-item>

              <el-form-item label="创建时间">
                <span class="info-text">{{ formatTime(profileForm.created_at) }}</span>
              </el-form-item>

              <el-form-item label="最后登录">
                <span class="info-text">{{ formatTime(profileForm.last_login) }}</span>
              </el-form-item>
            </el-form>
          </el-card>
        </el-col>

        <!-- 操作面板 -->
        <el-col :span="8">
          <el-card class="action-card">
            <template #header>
              <span class="card-title">账户操作</span>
            </template>

            <div class="action-list">
              <div class="action-item" @click="showChangePasswordDialog">
                <div class="action-icon">
                  <el-icon color="#409eff"><Lock /></el-icon>
                </div>
                <div class="action-content">
                  <div class="action-title">修改密码</div>
                  <div class="action-desc">更改您的登录密码</div>
                </div>
              </div>
            </div>
          </el-card>

          <!-- 统计信息卡片 -->
          <el-card class="stats-card">
            <template #header>
              <span class="card-title">统计信息</span>
            </template>

            <div class="stats-list">
              <div class="stat-item">
                <div class="stat-label">登录次数</div>
                <div class="stat-value">{{ profileStats.loginCount || '-' }}</div>
              </div>
              <div class="stat-item">
                <div class="stat-label">操作记录</div>
                <div class="stat-value">{{ profileStats.operationCount || '-' }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
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
            placeholder="请输入当前密码"
          />
        </el-form-item>
        <el-form-item label="新密码" prop="newPassword">
          <el-input
            v-model="changePasswordForm.newPassword"
            type="password"
            show-password
            placeholder="请输入新密码"
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="changePasswordForm.confirmPassword"
            type="password"
            show-password
            placeholder="请再次输入新密码"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="changePasswordVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="changePasswordLoading"
          @click="handleChangePassword"
        >
          确认修改
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/modules/auth'
import { formatTime } from '@/utils/format'
import authApi from '@/api/auth'
import {
  Lock,
  User,
  Edit
} from '@element-plus/icons-vue'

const authStore = useAuthStore()

// 响应式数据
const editing = ref(false)
const updateLoading = ref(false)
const changePasswordVisible = ref(false)
const changePasswordLoading = ref(false)
const profileFormRef = ref()
const changePasswordFormRef = ref()

// 个人信息表单
const profileForm = reactive({
  username: '',
  email: '',
  role: '',
  created_at: '',
  last_login: ''
})

// 原始数据（用于取消编辑）
let originalProfileData = {}

// 统计信息
const profileStats = reactive({
  loginCount: 0,
  operationCount: 0
})

// 修改密码表单
const changePasswordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

// 表单验证规则
const profileRules = {
  email: [
    { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }
  ]
}

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

// 计算属性
const userInfo = computed(() => authStore.userInfo || {})

// 获取角色显示名称
const getRoleDisplayName = (role) => {
  const roleMap = {
    admin: '管理员',
    user: '普通用户',
    viewer: '只读用户'
  }
  return roleMap[role] || role
}

// 获取角色标签类型
const getRoleTagType = (role) => {
  const typeMap = {
    admin: 'danger',
    user: 'primary',
    viewer: 'info'
  }
  return typeMap[role] || 'info'
}

// 开始编辑
const startEdit = () => {
  originalProfileData = { ...profileForm }
  editing.value = true
}

// 取消编辑
const cancelEdit = () => {
  Object.assign(profileForm, originalProfileData)
  editing.value = false
}

// 保存个人信息
const saveProfile = async () => {
  try {
    await profileFormRef.value.validate()
    updateLoading.value = true
    
    const updateData = {
      email: profileForm.email
    }
    
    await authApi.updateProfile(updateData)
    
    // 更新store中的用户信息
    await authStore.getUserInfo()
    loadProfileData()
    
    editing.value = false
    ElMessage.success('个人信息更新成功')
  } catch (error) {
    ElMessage.error(`更新失败: ${error.message || '系统错误'}`)
  } finally {
    updateLoading.value = false
  }
}

// 显示修改密码对话框
const showChangePasswordDialog = () => {
  changePasswordForm.oldPassword = ''
  changePasswordForm.newPassword = ''
  changePasswordForm.confirmPassword = ''
  changePasswordVisible.value = true
}

// 修改密码
const handleChangePassword = async () => {
  try {
    await changePasswordFormRef.value.validate()
    changePasswordLoading.value = true
    
    await authApi.changePassword({
      oldPassword: changePasswordForm.oldPassword,
      newPassword: changePasswordForm.newPassword
    })
    
    changePasswordVisible.value = false
    ElMessage.success('密码修改成功')
    
    // 重置表单
    changePasswordForm.oldPassword = ''
    changePasswordForm.newPassword = ''
    changePasswordForm.confirmPassword = ''
  } catch (error) {
    ElMessage.error(`修改密码失败: ${error.message || '系统错误'}`)
  } finally {
    changePasswordLoading.value = false
  }
}

// 加载个人信息
const loadProfileData = () => {
  const user = userInfo.value
  profileForm.username = user.username || ''
  profileForm.email = user.email || ''
  profileForm.role = user.role || ''
  profileForm.created_at = user.created_at || ''
  profileForm.last_login = user.last_login || ''
}

// 加载统计信息
const loadProfileStats = async () => {
  try {
    const response = await authApi.getProfileStats()
    const stats = response.data.data || response.data
    profileStats.loginCount = stats.loginCount || 0
    profileStats.operationCount = stats.operationCount || 0
  } catch (error) {
    console.error('Failed to load profile stats:', error)
    ElMessage.warning('获取统计信息失败')
  }
}

onMounted(() => {
  loadProfileData()
  loadProfileStats()
})
</script>

<style scoped>
.profile-container {
  padding: 20px;
  min-height: 100vh;
  background-color: #f5f7fa;
}

.page-header {
  margin-bottom: 20px;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 8px;
}

.page-description {
  color: #909399;
  font-size: 14px;
}

.profile-content {
  max-width: 1200px;
}

.profile-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.edit-buttons {
  display: flex;
  gap: 8px;
}

.info-text {
  color: #606266;
  font-size: 14px;
}

.action-card {
  margin-bottom: 20px;
}

.action-list {
  display: flex;
  flex-direction: column;
}

.action-item {
  display: flex;
  align-items: center;
  padding: 12px 0;
  cursor: pointer;
  transition: all 0.3s;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 8px;
}

.action-item:hover {
  background-color: #f5f7fa;
}

.action-icon {
  margin-right: 12px;
  font-size: 20px;
}

.action-content {
  flex: 1;
}

.action-title {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 4px;
}

.action-desc {
  font-size: 12px;
  color: #909399;
}

.stats-card .stats-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}

.stat-item:last-child {
  border-bottom: none;
}

.stat-label {
  color: #606266;
  font-size: 14px;
}

.stat-value {
  color: #303133;
  font-weight: 600;
  font-size: 16px;
}

@media (max-width: 768px) {
  .profile-container {
    padding: 12px;
  }
  
  .profile-content :deep(.el-col) {
    width: 100% !important;
    margin-bottom: 20px;
  }
}
</style>
