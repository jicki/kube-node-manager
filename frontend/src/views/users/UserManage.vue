<template>
  <div class="user-manage">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">用户管理</h1>
        <p class="page-description">管理系统用户和权限配置</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="showAddDialog">
          <el-icon><Plus /></el-icon>
          添加用户
        </el-button>
        <el-button @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="24" class="stats-row">
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ userStats.total }}</div>
            <div class="stat-label">总用户数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ userStats.admin }}</div>
            <div class="stat-label">管理员</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ userStats.active }}</div>
            <div class="stat-label">活跃用户</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ userStats.online }}</div>
            <div class="stat-label">在线用户</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <SearchBox
        v-model="searchKeyword"
        placeholder="搜索用户名、邮箱..."
        :advanced-search="true"
        :filters="searchFilters"
        @search="handleSearch"
      />
    </el-card>

    <!-- 用户表格 -->
    <el-card class="table-card">
      <el-table
        v-loading="loading"
        :data="filteredUsers"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        
        <el-table-column prop="username" label="用户名" min-width="120">
          <template #default="{ row }">
            <div class="user-info">
              <el-avatar :size="32" class="user-avatar">
                <el-icon><User /></el-icon>
              </el-avatar>
              <div class="user-details">
                <div class="username">{{ row.username }}</div>
                <div class="user-email">{{ row.email }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="role" label="角色" width="100">
          <template #default="{ row }">
            <el-tag
              :type="getRoleType(row.role)"
              size="small"
            >
              {{ getRoleText(row.role) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 'active' ? 'success' : 'danger'"
              size="small"
            >
              {{ row.status === 'active' ? '正常' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="last_login" label="最后登录" width="180">
          <template #default="{ row }">
            <span class="time-text">
              {{ row.last_login ? formatTime(row.last_login) : '从未登录' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            <span class="time-text">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button type="text" size="small" @click="editUser(row)">
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              
              <el-button
                type="text"
                size="small"
                @click="resetPassword(row)"
              >
                <el-icon><Key /></el-icon>
                重置密码
              </el-button>
              
              <el-dropdown @command="(cmd) => handleUserAction(cmd, row)">
                <el-button type="text" size="small">
                  <el-icon><MoreFilled /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      :command="row.status === 'active' ? 'disable' : 'enable'"
                    >
                      <el-icon>
                        <component :is="row.status === 'active' ? 'Lock' : 'Unlock'" />
                      </el-icon>
                      {{ row.status === 'active' ? '禁用' : '启用' }}
                    </el-dropdown-item>
                    <el-dropdown-item command="delete" style="color: #f56c6c">
                      <el-icon><Delete /></el-icon>
                      删除
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 添加/编辑用户对话框 -->
    <el-dialog
      v-model="userDialogVisible"
      :title="isEditing ? '编辑用户' : '添加用户'"
      width="600px"
    >
      <el-form
        ref="userFormRef"
        :model="userForm"
        :rules="userRules"
        label-width="100px"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="userForm.username"
            placeholder="请输入用户名"
            :disabled="isEditing"
          />
        </el-form-item>

        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="userForm.email"
            type="email"
            placeholder="请输入邮箱地址"
          />
        </el-form-item>

        <el-form-item label="角色" prop="role">
          <el-select v-model="userForm.role" placeholder="选择用户角色">
            <el-option label="管理员" value="admin" />
            <el-option label="操作员" value="operator" />
            <el-option label="只读用户" value="viewer" />
          </el-select>
        </el-form-item>

        <el-form-item v-if="!isEditing" label="密码" prop="password">
          <el-input
            v-model="userForm.password"
            type="password"
            placeholder="请输入密码"
            show-password
          />
        </el-form-item>

        <el-form-item label="状态">
          <el-radio-group v-model="userForm.status">
            <el-radio label="active">启用</el-radio>
            <el-radio label="inactive">禁用</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="userForm.description"
            type="textarea"
            :rows="3"
            placeholder="用户描述信息"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="userDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="saving"
          @click="handleSaveUser"
        >
          {{ isEditing ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog
      v-model="passwordDialogVisible"
      title="重置密码"
      width="400px"
    >
      <el-form
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordRules"
        label-width="80px"
      >
        <el-form-item label="新密码" prop="password">
          <el-input
            v-model="passwordForm.password"
            type="password"
            placeholder="请输入新密码"
            show-password
          />
        </el-form-item>

        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="passwordForm.confirmPassword"
            type="password"
            placeholder="请确认新密码"
            show-password
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="passwordDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="saving"
          @click="handleResetPassword"
        >
          重置密码
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import userApi from '@/api/user'
import { formatTime } from '@/utils/format'
import SearchBox from '@/components/common/SearchBox.vue'
import {
  Plus,
  Refresh,
  User,
  Edit,
  Delete,
  Key,
  Lock,
  Unlock,
  MoreFilled
} from '@element-plus/icons-vue'

// 响应式数据
const loading = ref(false)
const saving = ref(false)
const searchKeyword = ref('')
const userDialogVisible = ref(false)
const passwordDialogVisible = ref(false)
const isEditing = ref(false)
const userFormRef = ref()
const passwordFormRef = ref()

// 数据
const users = ref([])
const selectedUsers = ref([])
const selectedUser = ref(null)

// 分页
const pagination = reactive({
  current: 1,
  size: 20,
  total: 0
})

// 表单数据
const userForm = reactive({
  username: '',
  email: '',
  role: '',
  password: '',
  status: 'active',
  description: ''
})

const passwordForm = reactive({
  password: '',
  confirmPassword: ''
})

// 搜索筛选
const searchFilters = ref([
  {
    key: 'role',
    label: '角色',
    type: 'select',
    placeholder: '选择用户角色',
    options: [
      { label: '全部', value: '' },
      { label: '管理员', value: 'admin' },
      { label: '操作员', value: 'operator' },
      { label: '只读用户', value: 'viewer' }
    ]
  },
  {
    key: 'status',
    label: '状态',
    type: 'select',
    placeholder: '选择用户状态',
    options: [
      { label: '全部', value: '' },
      { label: '启用', value: 'active' },
      { label: '禁用', value: 'inactive' }
    ]
  }
])

// 表单验证规则
const userRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度在 3 到 20 个字符', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }
  ],
  role: [
    { required: true, message: '请选择用户角色', trigger: 'change' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少为 6 位', trigger: 'blur' }
  ]
}

const passwordRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少为 6 位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== passwordForm.password) {
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
const userStats = computed(() => {
  // 确保users.value是数组
  const userList = Array.isArray(users.value) ? users.value : []
  const total = userList.length
  const admin = userList.filter(u => u.role === 'admin').length
  const active = userList.filter(u => u.status === 'active').length
  const online = userList.filter(u => u.isOnline).length
  
  return { total, admin, active, online }
})

const filteredUsers = computed(() => {
  // 确保users.value是数组
  const userList = Array.isArray(users.value) ? users.value : []
  return userList.slice(
    (pagination.current - 1) * pagination.size,
    pagination.current * pagination.size
  )
})

// 获取角色类型
const getRoleType = (role) => {
  const typeMap = {
    admin: 'danger',
    operator: 'warning',
    viewer: 'info'
  }
  return typeMap[role] || 'info'
}

// 获取角色文本
const getRoleText = (role) => {
  const textMap = {
    admin: '管理员',
    operator: '操作员',
    viewer: '只读用户'
  }
  return textMap[role] || role
}

// 方法
const fetchUsers = async () => {
  try {
    loading.value = true
    const response = await userApi.getUsers()
    
    // 处理响应数据
    let userData = []
    let totalCount = 0
    
    if (response.data) {
      // 新的统一响应格式: {code: 200, message: "Success", data: {users: [], total: n}}
      if (response.data.data && response.data.data.users) {
        userData = Array.isArray(response.data.data.users) ? response.data.data.users : []
        totalCount = response.data.data.total || userData.length
      }
      // 直接的响应格式: {users: [], total: n}
      else if (response.data.users) {
        userData = Array.isArray(response.data.users) ? response.data.users : []
        totalCount = response.data.total || userData.length
      }
      // 如果返回的是数组
      else if (Array.isArray(response.data)) {
        userData = response.data
        totalCount = userData.length
      }
      // 如果返回的是单个用户对象（向后兼容）
      else if (response.data.username) {
        userData = [response.data]
        totalCount = 1
      }
      // 处理包装在data字段中的单个用户对象
      else if (response.data.data && response.data.data.username) {
        userData = [response.data.data]
        totalCount = 1
      }
      else {
        console.warn('未识别的响应格式:', response.data)
        userData = []
        totalCount = 0
      }
    }
    
    users.value = userData
    pagination.total = totalCount
    
  } catch (error) {
    console.error('获取用户数据失败:', error)
    ElMessage.error(`获取用户数据失败: ${error.message || '系统错误'}`)
    users.value = []
    pagination.total = 0
  } finally {
    loading.value = false
  }
}

const refreshData = () => {
  fetchUsers()
}

const handleSearch = (params) => {
  console.log('Search params:', params)
}

const handleSelectionChange = (selection) => {
  selectedUsers.value = selection
}

const handleSizeChange = (size) => {
  pagination.size = size
  pagination.current = 1
}

const handleCurrentChange = (current) => {
  pagination.current = current
}

// 显示添加对话框
const showAddDialog = () => {
  isEditing.value = false
  resetUserForm()
  userDialogVisible.value = true
}

// 重置表单
const resetUserForm = () => {
  Object.assign(userForm, {
    username: '',
    email: '',
    role: '',
    password: '',
    status: 'active',
    description: ''
  })
}

// 编辑用户
const editUser = (user) => {
  isEditing.value = true
  Object.assign(userForm, {
    username: user.username,
    email: user.email,
    role: user.role,
    status: user.status,
    description: user.description || ''
  })
  selectedUser.value = user
  userDialogVisible.value = true
}

// 重置密码
const resetPassword = (user) => {
  selectedUser.value = user
  passwordForm.password = ''
  passwordForm.confirmPassword = ''
  passwordDialogVisible.value = true
}

// 处理用户操作
const handleUserAction = (command, user) => {
  switch (command) {
    case 'enable':
    case 'disable':
      toggleUserStatus(user)
      break
    case 'delete':
      deleteUser(user)
      break
  }
}

// 切换用户状态
const toggleUserStatus = async (user) => {
  const newStatus = user.status === 'active' ? 'inactive' : 'active'
  const action = newStatus === 'active' ? '启用' : '禁用'
  
  try {
    await userApi.toggleUserStatus(user.id, newStatus === 'active')
    ElMessage.success(`用户已${action}`)
    refreshData()
  } catch (error) {
    ElMessage.error(`${action}用户失败: ${error.message}`)
  }
}

// 删除用户
const deleteUser = (user) => {
  ElMessageBox.confirm(
    `确认删除用户 "${user.username}" 吗？此操作不可恢复。`,
    '删除用户',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await userApi.deleteUser(user.id)
      ElMessage.success('用户已删除')
      refreshData()
    } catch (error) {
      ElMessage.error(`删除用户失败: ${error.message}`)
    }
  }).catch(() => {
    // 用户取消
  })
}

// 保存用户
const handleSaveUser = async () => {
  try {
    await userFormRef.value.validate()
    saving.value = true
    
    if (isEditing.value) {
      await userApi.updateUser(selectedUser.value.id, userForm)
      ElMessage.success('用户更新成功')
    } else {
      await userApi.createUser(userForm)
      ElMessage.success('用户创建成功')
    }
    
    userDialogVisible.value = false
    refreshData()
    
  } catch (error) {
    ElMessage.error(error.message || '保存用户失败')
  } finally {
    saving.value = false
  }
}

// 处理重置密码
const handleResetPassword = async () => {
  try {
    await passwordFormRef.value.validate()
    saving.value = true
    
    await userApi.resetPassword(selectedUser.value.id, {
      password: passwordForm.password
    })
    
    ElMessage.success('密码重置成功')
    passwordDialogVisible.value = false
    
  } catch (error) {
    ElMessage.error(error.message || '重置密码失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped>
.user-manage {
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

.stats-row {
  margin-bottom: 24px;
}

.stat-card {
  text-align: center;
  cursor: pointer;
  transition: all 0.3s;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.stat-content {
  padding: 16px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1890ff;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #666;
}

.search-card {
  margin-bottom: 16px;
}

.table-card :deep(.el-card__body) {
  padding: 0;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-details {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.username {
  font-weight: 500;
  color: #333;
}

.user-email {
  font-size: 12px;
  color: #666;
}

.time-text {
  font-size: 13px;
  color: #666;
}

.action-buttons {
  display: flex;
  gap: 4px;
}

.pagination-container {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }
  
  .stats-row .el-col {
    margin-bottom: 16px;
  }
  
  .action-buttons {
    flex-direction: column;
    gap: 2px;
  }
}
</style>