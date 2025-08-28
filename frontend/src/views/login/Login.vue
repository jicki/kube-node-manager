<template>
  <div class="login-container">
    <div class="login-background">
      <!-- 背景动效 -->
      <div class="background-shapes">
        <div class="shape shape-1"></div>
        <div class="shape shape-2"></div>
        <div class="shape shape-3"></div>
        <div class="shape shape-4"></div>
      </div>
    </div>
    
    <div class="login-content">
      <!-- 左侧信息 -->
      <div class="login-info">
        <div class="info-content">
          <h1 class="system-title">
            <el-icon class="title-icon"><Monitor /></el-icon>
            Kubernetes 节点管理平台
          </h1>
          <p class="system-description">
            集群节点统一管理，标签污点便捷配置
          </p>
          
          <div class="feature-list">
            <div class="feature-item">
              <el-icon class="feature-icon"><Check /></el-icon>
              <span>多集群节点统一管理</span>
            </div>
            <div class="feature-item">
              <el-icon class="feature-icon"><Check /></el-icon>
              <span>标签污点批量操作</span>
            </div>
            <div class="feature-item">
              <el-icon class="feature-icon"><Check /></el-icon>
              <span>节点状态实时监控</span>
            </div>
            <div class="feature-item">
              <el-icon class="feature-icon"><Check /></el-icon>
              <span>操作审计完整记录</span>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 右侧登录表单 -->
      <div class="login-form-container">
        <div class="login-form-wrapper">
          <div class="login-header">
            <h2 class="login-title">用户登录</h2>
            <p class="login-subtitle">欢迎登录 Kubernetes 节点管理平台</p>
          </div>
          
          <el-form
            ref="loginFormRef"
            :model="loginForm"
            :rules="loginRules"
            class="login-form"
            label-position="top"
            size="large"
          >
            <el-form-item label="用户名" prop="username">
              <el-input
                v-model="loginForm.username"
                placeholder="请输入用户名"
                prefix-icon="User"
                @keyup.enter="handleLogin"
              />
            </el-form-item>
            
            <el-form-item label="密码" prop="password">
              <el-input
                v-model="loginForm.password"
                type="password"
                placeholder="请输入密码"
                prefix-icon="Lock"
                show-password
                @keyup.enter="handleLogin"
              />
            </el-form-item>
            

            
            <el-form-item>
              <div class="login-options">
                <el-checkbox v-model="rememberMe">记住我</el-checkbox>
              </div>
            </el-form-item>
            
            <el-form-item>
              <el-button
                type="primary"
                class="login-button"
                :loading="loading"
                @click="handleLogin"
              >
                {{ loading ? '登录中...' : '登录' }}
              </el-button>
            </el-form-item>
          </el-form>
          
          <!-- 其他登录方式 -->
          <div v-if="showLdapLogin" class="alternative-login">
            <el-divider>
              <span class="divider-text">其他登录方式</span>
            </el-divider>
            
            <el-button
              class="ldap-login-button"
              :loading="ldapLoading"
              @click="handleLdapLogin"
            >
              <el-icon class="button-icon"><Connection /></el-icon>
              LDAP 登录
            </el-button>
          </div>
          
          <!-- 系统信息 -->
          <div class="system-info">
            <p class="version-info">版本: {{ systemVersion }}</p>
            <p class="copyright">© 2024 Kubernetes 节点管理平台</p>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 加载遮罩 -->
    <LoadingSpinner
      v-if="loading"
      full-screen
      text="登录中..."
      description="正在验证用户信息"
    />
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/store/modules/auth'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { Monitor, Check, User, Lock, Warning, Connection } from '@element-plus/icons-vue'
import authApi from '@/api/auth'

const router = useRouter()
const authStore = useAuthStore()

// 响应式数据
const loginFormRef = ref()
const loading = ref(false)
const ldapLoading = ref(false)
const rememberMe = ref(false)

// 登录表单
const loginForm = reactive({
  username: '',
  password: ''
})

// 表单验证规则
const loginRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 50, message: '用户名长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少为 6 位', trigger: 'blur' }
  ]
}

// 计算属性
const showLdapLogin = computed(() => {
  // 从配置或环境变量中获取
  return import.meta.env.VITE_ENABLE_LDAP === 'true'
})

// 系统版本信息
const systemVersion = ref('1.0.0')

// 处理登录
const handleLogin = async () => {
  try {
    // 表单验证
    await loginFormRef.value.validate()
    
    loading.value = true
    
    const credentials = {
      username: loginForm.username,
      password: loginForm.password
    }
    

    
    // 调用登录API
    await authStore.login(credentials)
    
    // 记住我功能
    if (rememberMe.value) {
      localStorage.setItem('rememberedUsername', loginForm.username)
    } else {
      localStorage.removeItem('rememberedUsername')
    }
    
    ElMessage.success('登录成功')
    
    // 跳转到首页
    router.push('/dashboard')
    
  } catch (error) {
    console.error('Login error:', error)
    
    // 登录失败处理
    const failedCount = parseInt(localStorage.getItem('loginFailedCount') || '0') + 1
    localStorage.setItem('loginFailedCount', failedCount.toString())
    
    ElMessage.error(error.message || '登录失败，请检查用户名和密码')
  } finally {
    loading.value = false
  }
}

// LDAP登录
const handleLdapLogin = async () => {
  try {
    ldapLoading.value = true
    
    // 跳转到LDAP登录页面或处理LDAP登录逻辑
    ElMessage.info('LDAP登录功能开发中')
    
  } catch (error) {
    ElMessage.error('LDAP登录失败')
  } finally {
    ldapLoading.value = false
  }
}



// 记住用户名
const loadRememberedUsername = () => {
  const remembered = localStorage.getItem('rememberedUsername')
  if (remembered) {
    loginForm.username = remembered
    rememberMe.value = true
  }
}

// 获取系统版本信息
const fetchSystemVersion = async () => {
  try {
    const response = await authApi.getVersion()
    if (response && response.version) {
      systemVersion.value = response.version
    }
  } catch (error) {
    console.warn('Failed to fetch system version:', error)
    // 保持默认版本
  }
}

onMounted(() => {
  loadRememberedUsername()
  fetchSystemVersion()
})
</script>

<style scoped>
.login-container {
  position: relative;
  width: 100%;
  height: 100vh;
  overflow: hidden;
}

.login-background {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.background-shapes {
  position: absolute;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.shape {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  animation: float 6s ease-in-out infinite;
}

.shape-1 {
  width: 200px;
  height: 200px;
  top: 10%;
  left: 10%;
  animation-delay: 0s;
}

.shape-2 {
  width: 150px;
  height: 150px;
  top: 70%;
  left: 20%;
  animation-delay: 2s;
}

.shape-3 {
  width: 100px;
  height: 100px;
  top: 30%;
  right: 20%;
  animation-delay: 4s;
}

.shape-4 {
  width: 80px;
  height: 80px;
  bottom: 20%;
  right: 30%;
  animation-delay: 1s;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0px) rotate(0deg);
    opacity: 0.1;
  }
  50% {
    transform: translateY(-20px) rotate(180deg);
    opacity: 0.2;
  }
}

.login-content {
  position: relative;
  display: flex;
  width: 100%;
  height: 100%;
  z-index: 1;
}

.login-info {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
  color: white;
}

.info-content {
  max-width: 500px;
}

.system-title {
  display: flex;
  align-items: center;
  font-size: 36px;
  font-weight: 700;
  margin-bottom: 20px;
  line-height: 1.2;
}

.title-icon {
  font-size: 40px;
  margin-right: 12px;
  color: #ffd700;
}

.system-description {
  font-size: 18px;
  margin-bottom: 40px;
  opacity: 0.9;
  line-height: 1.6;
}

.feature-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.feature-item {
  display: flex;
  align-items: center;
  font-size: 16px;
}

.feature-icon {
  margin-right: 12px;
  color: #52c41a;
  background: rgba(82, 196, 26, 0.2);
  padding: 4px;
  border-radius: 50%;
}

.login-form-container {
  width: 480px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: -5px 0 15px rgba(0, 0, 0, 0.1);
}

.login-form-wrapper {
  width: 100%;
  max-width: 360px;
  padding: 40px 0;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-title {
  font-size: 28px;
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
}

.login-subtitle {
  font-size: 14px;
  color: #666;
  margin: 0;
}

.login-form {
  width: 100%;
}

.login-form :deep(.el-form-item__label) {
  color: #333;
  font-weight: 500;
  font-size: 14px;
}

.login-form :deep(.el-input__wrapper) {
  border-radius: 8px;
  box-shadow: 0 0 0 1px #d9d9d9;
}

.login-form :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #4096ff;
}

.login-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px rgba(64, 150, 255, 0.2);
}



.login-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}



.login-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: 500;
  border-radius: 8px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
}

.login-button:hover {
  background: linear-gradient(135deg, #5a6fd8 0%, #6a4190 100%);
}

.alternative-login {
  margin-top: 24px;
}

.divider-text {
  color: #999;
  font-size: 12px;
}

.ldap-login-button {
  width: 100%;
  height: 40px;
  border: 1px solid #d9d9d9;
  color: #666;
  background: #fff;
}

.button-icon {
  margin-right: 8px;
}

.system-info {
  text-align: center;
  margin-top: 32px;
  padding-top: 20px;
  border-top: 1px solid #f0f0f0;
}

.version-info,
.copyright {
  font-size: 12px;
  color: #999;
  margin: 4px 0;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .login-info {
    display: none;
  }
  
  .login-form-container {
    width: 100%;
  }
}

@media (max-width: 768px) {
  .login-form-container {
    background: rgba(255, 255, 255, 1);
  }
  
  .login-form-wrapper {
    padding: 20px;
    max-width: none;
  }
  
  .system-title {
    font-size: 24px;
  }
  
  .login-title {
    font-size: 24px;
  }
}
</style>