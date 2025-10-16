<template>
  <div class="page-container">
    <div class="card-container">
      <h2>飞书配置</h2>
      <el-divider />

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="150px"
        style="max-width: 700px"
      >
        <el-form-item label="启用飞书" prop="enabled">
          <el-switch v-model="form.enabled" />
          <span style="margin-left: 12px; color: #909399; font-size: 14px">
            启用后将显示飞书相关功能
          </span>
        </el-form-item>

        <el-form-item label="App ID" prop="app_id">
          <el-input
            v-model="form.app_id"
            placeholder="输入飞书应用的 App ID"
            :disabled="!form.enabled"
          />
        </el-form-item>

        <el-form-item label="App Secret" prop="app_secret">
          <el-input
            v-model="form.app_secret"
            type="password"
            placeholder="输入新的 App Secret"
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
          <div v-if="hasAppSecret && !form.app_secret" style="margin-top: 4px; color: #67c23a; font-size: 12px">
            已配置 App Secret，留空则不修改
          </div>
        </el-form-item>

        <el-divider content-position="left">机器人配置（长连接模式）</el-divider>

        <el-form-item label="启用机器人功能" prop="bot_enabled">
          <el-switch v-model="form.bot_enabled" :disabled="!form.enabled" />
          <span style="margin-left: 12px; color: #909399; font-size: 14px">
            启用后自动建立长连接，接收飞书消息
          </span>
        </el-form-item>

        <el-form-item label="连接状态" v-if="form.bot_enabled">
          <el-tag :type="botConnected ? 'success' : 'info'" :icon="botConnected ? SuccessFilled : Loading">
            {{ botConnected ? '已连接' : '未连接' }}
          </el-tag>
          <span style="margin-left: 12px; color: #909399; font-size: 14px">
            长连接模式无需配置 Webhook URL
          </span>
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
        <p><strong>1. 创建飞书应用：</strong></p>
        <ul>
          <li>登录飞书开放平台：<a href="https://open.feishu.cn/app" target="_blank">https://open.feishu.cn/app</a></li>
          <li>创建企业自建应用</li>
          <li>在"凭证与基础信息"页面获取 App ID 和 App Secret</li>
        </ul>

        <p style="margin-top: 16px"><strong>2. 配置应用权限：</strong></p>
        <ul>
          <li>进入应用的"权限管理"页面</li>
          <li>添加以下权限：
            <ul>
              <li><code>im:chat</code> - 获取群组信息</li>
              <li><code>im:chat:readonly</code> - 查看群组信息</li>
              <li><code>im:message</code> - 发送消息（机器人功能）</li>
              <li><code>im:message:receive_v1</code> - 接收消息（机器人功能）</li>
            </ul>
          </li>
          <li>发布应用版本并启用</li>
        </ul>

        <p style="margin-top: 16px"><strong>3. 启用机器人功能：</strong></p>
        <ul>
          <li>在本页面启用"机器人功能"开关</li>
          <li>系统将自动使用<strong>长连接模式</strong>接收消息</li>
          <li><span style="color: #67c23a">✓ 无需配置公网域名和 Webhook URL</span></li>
          <li><span style="color: #67c23a">✓ 无需配置加密策略</span></li>
          <li>保存配置后，连接状态会自动更新</li>
        </ul>

        <p style="margin-top: 16px"><strong>4. 将机器人添加到群组：</strong></p>
        <ul>
          <li>在飞书群组中，点击群设置</li>
          <li>选择"群机器人" → "添加机器人"</li>
          <li>搜索并添加您的应用</li>
          <li>在群组中向机器人发送命令（如 <code>/help</code>）</li>
        </ul>

        <p style="margin-top: 16px"><strong>5. 权限说明：</strong></p>
        <ul>
          <li>仅管理员可以配置飞书集成</li>
          <li>配置后所有用户都可以查询群组信息</li>
          <li>使用机器人功能需要先绑定飞书账号</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { View, SuccessFilled, Loading } from '@element-plus/icons-vue'
import { useFeishuStore } from '@/store/modules/feishu'
import { useAuthStore } from '@/store/modules/auth'

const feishuStore = useFeishuStore()
const authStore = useAuthStore()

const formRef = ref(null)
const saving = ref(false)
const testing = ref(false)

const form = ref({
  enabled: false,
  app_id: '',
  app_secret: '',
  bot_enabled: false
})

const rules = {
  app_id: [
    {
      required: true,
      message: '请输入 App ID',
      trigger: 'blur'
    }
  ]
}

const hasAppSecret = computed(() => feishuStore.hasAppSecret)
const botConnected = computed(() => feishuStore.settings?.bot_connected || false)

const canTest = computed(() => {
  return form.value.enabled && form.value.app_id && (form.value.app_secret || hasAppSecret.value)
})

// Load settings
const loadSettings = async () => {
  try {
    await feishuStore.fetchSettings()
    if (feishuStore.settings) {
      form.value.enabled = feishuStore.settings.enabled
      form.value.app_id = feishuStore.settings.app_id || ''
      form.value.app_secret = ''
      form.value.bot_enabled = feishuStore.settings.bot_enabled || false
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
    await feishuStore.testConnection({
      app_id: form.value.app_id,
      app_secret: form.value.app_secret || undefined
    })
    ElMessage.success('连接测试成功')
  } catch (error) {
    ElMessage.error(feishuStore.error || '连接测试失败')
  } finally {
    testing.value = false
  }
}

// Save settings
const handleSave = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    // If enabled, app_id is required
    if (form.value.enabled && !form.value.app_id) {
      ElMessage.error('启用飞书时必须配置 App ID')
      return
    }

    saving.value = true
    try {
      await feishuStore.updateSettings({
        enabled: form.value.enabled,
        app_id: form.value.app_id,
        app_secret: form.value.app_secret,
        bot_enabled: form.value.bot_enabled
      })
      ElMessage.success('配置保存成功')
      form.value.app_secret = ''
      await loadSettings()
    } catch (error) {
      ElMessage.error(feishuStore.error || '保存配置失败')
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
    ElMessage.error('只有管理员可以配置飞书')
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

.info-content a {
  color: #409eff;
  text-decoration: none;
}

.info-content a:hover {
  text-decoration: underline;
}
</style>

