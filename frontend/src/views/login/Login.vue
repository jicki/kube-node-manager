<template>
  <div class="login-container">
    <div class="login-content">
      <!-- 左侧信息 -->
      <div class="login-info">
        <div class="info-content">
          <!-- 系统图标 -->
          <div class="system-icon">
            <svg viewBox="0 0 1024 1024" class="logo-icon">
              <path d="M512 42.666667C252.8 42.666667 42.666667 252.8 42.666667 512s210.133333 469.333333 469.333333 469.333333 469.333333-210.133333 469.333333-469.333333S771.2 42.666667 512 42.666667z m213.333333 469.333333c0 23.466667-19.2 42.666667-42.666666 42.666667h-128v128c0 23.466667-19.2 42.666667-42.666667 42.666666s-42.666667-19.2-42.666667-42.666666v-128H341.333333c-23.466667 0-42.666667-19.2-42.666666-42.666667s19.2-42.666667 42.666666-42.666666h128V341.333333c0-23.466667 19.2-42.666667 42.666667-42.666666s42.666667 19.2 42.666667 42.666666v128h128c23.466667 0 42.666667 19.2 42.666666 42.666667z" />
            </svg>
          </div>
          
          <!-- 系统标题 -->
          <h1 class="system-title">Kubernetes节点管理器</h1>
          <p class="system-subtitle">实时监控・安全可靠・快速部署</p>
        </div>
      </div>
      
      <!-- 右侧登录表单 -->
      <div class="login-form-container">
        <div class="login-form-wrapper">
          <div class="login-header">
            <h2 class="login-title">欢迎回来</h2>
            <p class="login-subtitle">请登录以继续使用 Kubernetes节点管理器</p>
          </div>
          
          <el-form
            ref="loginFormRef"
            :model="loginForm"
            :rules="loginRules"
            class="login-form"
            label-position="top"
            size="large"
          >
            <el-form-item prop="username">
              <el-input
                v-model="loginForm.username"
                placeholder="用户名"
                prefix-icon="User"
                @keyup.enter="handleLogin"
              />
            </el-form-item>
            
            <el-form-item prop="password">
              <el-input
                v-model="loginForm.password"
                type="password"
                placeholder="密码"
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
                登录
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
            <p class="copyright">© 2024 Kubernetes节点管理器</p>
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

// 系统版本信息 - 优先使用构建时注入的版本，后备为API获取
const systemVersion = ref(__APP_VERSION__ || 'dev')

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
    
    // 显示更友好的错误信息
    let errorMessage = '登录失败，请检查用户名和密码'
    if (error.message) {
      errorMessage = error.message
    } else if (error.response && error.response.status === 401) {
      errorMessage = '用户名或密码错误'
    } else if (error.response && error.response.status >= 500) {
      errorMessage = '服务器错误，请稍后重试'
    }
    
    ElMessage.error(errorMessage)
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
  background: #1a1a1a;
}

.login-content {
  display: flex;
  width: 100%;
  height: 100%;
}

.login-info {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
  color: white;
  background: #1a1a1a;
}

.info-content {
  text-align: center;
}

.system-icon {
  margin-bottom: 32px;
  display: flex;
  justify-content: center;
}

.logo-icon {
  width: 80px;
  height: 80px;
  fill: #ffffff;
  opacity: 0.9;
}

.system-title {
  font-size: 32px;
  font-weight: 600;
  margin-bottom: 16px;
  line-height: 1.2;
  color: #ffffff;
}

.system-subtitle {
  font-size: 16px;
  color: rgba(255, 255, 255, 0.7);
  margin: 0;
  line-height: 1.5;
}

.login-form-container {
  width: 420px;
  background: #ffffff;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 24px 0 0 24px;
  box-shadow: -4px 0 20px rgba(0, 0, 0, 0.1);
}

.login-form-wrapper {
  width: 100%;
  max-width: 320px;
  padding: 40px 0;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-title {
  font-size: 24px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 8px;
}

.login-subtitle {
  font-size: 14px;
  color: #666;
  margin: 0;
  line-height: 1.4;
}

.login-form {
  width: 100%;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.login-form :deep(.el-input__wrapper) {
  height: 48px;
  border-radius: 12px;
  box-shadow: 0 0 0 1px #e5e5e5;
  background: #fafafa;
  transition: all 0.3s ease;
}

.login-form :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #1a1a1a;
  background: #ffffff;
}

.login-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px rgba(26, 26, 26, 0.2);
  background: #ffffff;
}

.login-form :deep(.el-input__inner) {
  font-size: 14px;
  color: #333;
}

.login-form :deep(.el-input__inner::placeholder) {
  color: #999;
}

.login-options {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  margin-bottom: 8px;
}

.login-options :deep(.el-checkbox__label) {
  font-size: 14px;
  color: #666;
}

.login-button {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 500;
  border-radius: 12px;
  background: #1a1a1a;
  border: none;
  color: #ffffff;
  transition: all 0.3s ease;
}

.login-button:hover {
  background: #333333;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(26, 26, 26, 0.3);
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
  height: 44px;
  border: 1px solid #e5e5e5;
  color: #666;
  background: #fff;
  border-radius: 12px;
  transition: all 0.3s ease;
}

.ldap-login-button:hover {
  border-color: #1a1a1a;
  color: #1a1a1a;
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
    border-radius: 0;
  }
}

@media (max-width: 768px) {
  .login-container {
    background: #ffffff;
  }
  
  .login-form-container {
    background: #ffffff;
    box-shadow: none;
  }
  
  .login-form-wrapper {
    padding: 20px;
    max-width: none;
  }
  
  .login-title {
    font-size: 20px;
  }
  
  .login-subtitle {
    font-size: 13px;
  }
}
</style>