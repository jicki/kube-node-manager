<template>
  <div class="page-container">
    <div class="card-container">
      <h2>GitLab 配置</h2>
      <el-divider />

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
        style="max-width: 600px"
      >
        <el-form-item label="启用 GitLab" prop="enabled">
          <el-switch v-model="form.enabled" />
          <span style="margin-left: 12px; color: #909399; font-size: 14px">
            启用后将显示 GitLab 相关功能
          </span>
        </el-form-item>

        <el-form-item label="GitLab 域名" prop="domain">
          <el-input
            v-model="form.domain"
            placeholder="https://gitlab.example.com"
            :disabled="!form.enabled"
          />
        </el-form-item>

        <el-form-item label="访问令牌" prop="token">
          <el-input
            v-model="form.token"
            type="password"
            placeholder="输入新的访问令牌"
            :disabled="!form.enabled"
            show-password
          >
            <template #append>
              <el-button
                :icon="View"
                @click="testConnection"
                :disabled="!canTest"
                :loading="testing"
              >
                测试连接
              </el-button>
            </template>
          </el-input>
          <div v-if="hasToken && !form.token" style="margin-top: 4px; color: #67c23a; font-size: 12px">
            已配置令牌，留空则不修改
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSave" :loading="saving">
            保存配置
          </el-button>
          <el-button @click="handleCancel">取消</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 配置说明 -->
    <div class="card-container" style="margin-top: 20px">
      <h3>配置说明</h3>
      <el-divider />
      <div class="info-content">
        <p><strong>1. 获取访问令牌：</strong></p>
        <ul>
          <li>登录 GitLab</li>
          <li>进入 User Settings → Access Tokens</li>
          <li>创建新的 Personal Access Token</li>
          <li>权限需要选择: <code>api</code>, <code>read_api</code>, <code>read_repository</code></li>
        </ul>

        <p style="margin-top: 16px"><strong>2. GitLab 域名格式：</strong></p>
        <ul>
          <li>需要包含协议 (http:// 或 https://)</li>
          <li>例如: <code>https://gitlab.example.com</code></li>
          <li>不需要包含 /api/v4 路径</li>
        </ul>

        <p style="margin-top: 16px"><strong>3. 权限说明：</strong></p>
        <ul>
          <li>仅管理员可以配置 GitLab 集成</li>
          <li>配置后所有用户都可以查看 Runners 和 Pipelines</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { View } from '@element-plus/icons-vue'
import { useGitlabStore } from '@/store/modules/gitlab'
import { useAuthStore } from '@/store/modules/auth'

const gitlabStore = useGitlabStore()
const authStore = useAuthStore()

const formRef = ref(null)
const saving = ref(false)
const testing = ref(false)

const form = ref({
  enabled: false,
  domain: '',
  token: ''
})

const rules = {
  domain: [
    {
      required: true,
      message: '请输入 GitLab 域名',
      trigger: 'blur'
    },
    {
      pattern: /^https?:\/\/.+/,
      message: '域名必须以 http:// 或 https:// 开头',
      trigger: 'blur'
    }
  ]
}

const hasToken = computed(() => gitlabStore.hasToken)
const canTest = computed(() => {
  return form.value.enabled && form.value.domain && (form.value.token || hasToken.value)
})

// Load settings
const loadSettings = async () => {
  try {
    await gitlabStore.fetchSettings()
    if (gitlabStore.settings) {
      form.value.enabled = gitlabStore.settings.enabled
      form.value.domain = gitlabStore.settings.domain || ''
      form.value.token = ''
    }
  } catch (error) {
    ElMessage.error('加载配置失败')
  }
}

// Test connection
const testConnection = async () => {
  if (!canTest.value) {
    return
  }

  testing.value = true
  try {
    await gitlabStore.testConnection({
      domain: form.value.domain,
      token: form.value.token || undefined
    })
    ElMessage.success('连接测试成功')
  } catch (error) {
    ElMessage.error(gitlabStore.error || '连接测试失败')
  } finally {
    testing.value = false
  }
}

// Save settings
const handleSave = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    // If enabled, domain is required
    if (form.value.enabled && !form.value.domain) {
      ElMessage.error('启用 GitLab 时必须配置域名')
      return
    }

    saving.value = true
    try {
      await gitlabStore.updateSettings({
        enabled: form.value.enabled,
        domain: form.value.domain,
        token: form.value.token
      })
      ElMessage.success('配置保存成功')
      form.value.token = ''
      await loadSettings()
    } catch (error) {
      ElMessage.error(gitlabStore.error || '保存配置失败')
    } finally {
      saving.value = false
    }
  })
}

// Cancel
const handleCancel = () => {
  loadSettings()
}

onMounted(() => {
  // Check admin permission
  if (!authStore.hasPermission('admin')) {
    ElMessage.error('只有管理员可以配置 GitLab')
    return
  }

  loadSettings()
})
</script>

<style scoped>
.info-content {
  color: #606266;
  line-height: 1.8;
}

.info-content ul {
  margin: 8px 0;
  padding-left: 24px;
}

.info-content li {
  margin: 4px 0;
}

.info-content code {
  background-color: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  color: #e6a23c;
}
</style>
