<template>
  <div class="page-container">
    <div class="card-container">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2><el-icon><Setting /></el-icon> 自动化配置</h2>
        </div>
        <div class="toolbar-right">
          <el-button type="primary" @click="saveSettings" :loading="saving">
            <el-icon><Check /></el-icon>
            保存配置
          </el-button>
        </div>
      </div>

      <el-alert
        title="提示"
        type="info"
        :closable="false"
        style="margin-bottom: 20px;"
      >
        自动化功能允许您通过 Ansible、SSH 命令、脚本和工作流来管理 Kubernetes 节点。启用后，侧边栏将显示"自动化"菜单。
      </el-alert>

      <el-form
        ref="formRef"
        :model="formData"
        label-width="180px"
        label-position="left"
      >
        <!-- 主开关 -->
        <el-card shadow="hover" style="margin-bottom: 20px;">
          <template #header>
            <div class="card-header">
              <span>功能主开关</span>
            </div>
          </template>

          <el-form-item label="启用自动化功能">
            <el-switch
              v-model="formData.enabled"
              active-text="启用"
              inactive-text="禁用"
              size="large"
            />
            <div class="form-item-tips">
              启用后，侧边栏将显示"自动化"菜单，用户可以访问 Ansible、SSH、脚本和工作流功能
            </div>
          </el-form-item>
        </el-card>

        <!-- Ansible 配置 -->
        <el-card shadow="hover" style="margin-bottom: 20px;">
          <template #header>
            <div class="card-header">
              <span>Ansible 配置</span>
              <el-tag :type="formData.ansible.enabled ? 'success' : 'info'" size="small">
                {{ formData.ansible.enabled ? '已启用' : '已禁用' }}
              </el-tag>
            </div>
          </template>

          <el-form-item label="启用 Ansible">
            <el-switch
              v-model="formData.ansible.enabled"
              :disabled="!formData.enabled"
            />
          </el-form-item>

          <el-form-item label="Ansible 可执行文件路径">
            <el-input
              v-model="formData.ansible.binary_path"
              placeholder="/usr/bin/ansible-playbook"
              :disabled="!formData.enabled || !formData.ansible.enabled"
            />
            <div class="form-item-tips">
              ansible-playbook 命令的完整路径
            </div>
          </el-form-item>

          <el-form-item label="临时目录">
            <el-input
              v-model="formData.ansible.temp_dir"
              placeholder="/tmp/ansible-runs"
              :disabled="!formData.enabled || !formData.ansible.enabled"
            />
            <div class="form-item-tips">
              用于存储 Playbook 执行的临时文件
            </div>
          </el-form-item>

          <el-form-item label="执行超时（秒）">
            <el-input-number
              v-model="formData.ansible.timeout"
              :min="60"
              :max="7200"
              :step="60"
              :disabled="!formData.enabled || !formData.ansible.enabled"
            />
            <div class="form-item-tips">
              Playbook 执行的最大超时时间
            </div>
          </el-form-item>
        </el-card>

        <!-- SSH 配置 -->
        <el-card shadow="hover" style="margin-bottom: 20px;">
          <template #header>
            <div class="card-header">
              <span>SSH 配置</span>
              <el-tag :type="formData.ssh.enabled ? 'success' : 'info'" size="small">
                {{ formData.ssh.enabled ? '已启用' : '已禁用' }}
              </el-tag>
            </div>
          </template>

          <el-form-item label="启用 SSH">
            <el-switch
              v-model="formData.ssh.enabled"
              :disabled="!formData.enabled"
            />
          </el-form-item>

          <el-form-item label="SSH 超时（秒）">
            <el-input-number
              v-model="formData.ssh.timeout"
              :min="5"
              :max="300"
              :step="5"
              :disabled="!formData.enabled || !formData.ssh.enabled"
            />
            <div class="form-item-tips">
              单个 SSH 命令的最大执行时间
            </div>
          </el-form-item>

          <el-form-item label="最大并发数">
            <el-input-number
              v-model="formData.ssh.max_concurrent"
              :min="1"
              :max="100"
              :step="5"
              :disabled="!formData.enabled || !formData.ssh.enabled"
            />
            <div class="form-item-tips">
              同时执行 SSH 命令的最大节点数
            </div>
          </el-form-item>

          <el-form-item label="连接池大小">
            <el-input-number
              v-model="formData.ssh.connection_pool_size"
              :min="5"
              :max="100"
              :step="5"
              :disabled="!formData.enabled || !formData.ssh.enabled"
            />
            <div class="form-item-tips">
              SSH 连接池的最大连接数
            </div>
          </el-form-item>
        </el-card>

        <!-- 脚本配置 -->
        <el-card shadow="hover" style="margin-bottom: 20px;">
          <template #header>
            <div class="card-header">
              <span>脚本配置</span>
              <el-tag :type="formData.scripts.enabled ? 'success' : 'info'" size="small">
                {{ formData.scripts.enabled ? '已启用' : '已禁用' }}
              </el-tag>
            </div>
          </template>

          <el-form-item label="启用脚本">
            <el-switch
              v-model="formData.scripts.enabled"
              :disabled="!formData.enabled"
            />
          </el-form-item>

          <el-form-item label="脚本超时（秒）">
            <el-input-number
              v-model="formData.scripts.timeout"
              :min="60"
              :max="3600"
              :step="60"
              :disabled="!formData.enabled || !formData.scripts.enabled"
            />
            <div class="form-item-tips">
              脚本执行的最大超时时间
            </div>
          </el-form-item>
        </el-card>

        <!-- 工作流配置 -->
        <el-card shadow="hover" style="margin-bottom: 20px;">
          <template #header>
            <div class="card-header">
              <span>工作流配置</span>
              <el-tag :type="formData.workflows.enabled ? 'success' : 'info'" size="small">
                {{ formData.workflows.enabled ? '已启用' : '已禁用' }}
              </el-tag>
            </div>
          </template>

          <el-form-item label="启用工作流">
            <el-switch
              v-model="formData.workflows.enabled"
              :disabled="!formData.enabled"
            />
          </el-form-item>

          <el-form-item label="最大步骤数">
            <el-input-number
              v-model="formData.workflows.max_steps"
              :min="5"
              :max="100"
              :step="5"
              :disabled="!formData.enabled || !formData.workflows.enabled"
            />
            <div class="form-item-tips">
              单个工作流允许的最大步骤数
            </div>
          </el-form-item>

          <el-form-item label="步骤超时（秒）">
            <el-input-number
              v-model="formData.workflows.step_timeout"
              :min="60"
              :max="7200"
              :step="60"
              :disabled="!formData.enabled || !formData.workflows.enabled"
            />
            <div class="form-item-tips">
              工作流单个步骤的最大执行时间
            </div>
          </el-form-item>
        </el-card>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Setting, Check } from '@element-plus/icons-vue'
import { useFeaturesStore } from '@/store/modules/features'

const featuresStore = useFeaturesStore()

const formRef = ref(null)
const saving = ref(false)

const formData = reactive({
  enabled: false,
  ansible: {
    enabled: true,
    binary_path: '/usr/bin/ansible-playbook',
    temp_dir: '/tmp/ansible-runs',
    timeout: 3600
  },
  ssh: {
    enabled: true,
    timeout: 30,
    max_concurrent: 50,
    connection_pool_size: 20
  },
  scripts: {
    enabled: true,
    timeout: 600
  },
  workflows: {
    enabled: true,
    max_steps: 50,
    step_timeout: 1800
  }
})

// 加载配置
const loadSettings = async () => {
  await featuresStore.fetchFeatures()
  const features = featuresStore.allFeatures.automation
  
  formData.enabled = features.enabled
  formData.ansible = { ...features.ansible }
  formData.ssh = { ...features.ssh }
  formData.scripts = { ...features.scripts }
  formData.workflows = { ...features.workflows }
}

// 保存配置
const saveSettings = async () => {
  saving.value = true
  try {
    // 保存主开关
    const mainResult = await featuresStore.updateAutomationEnabled(formData.enabled)
    if (!mainResult.success) {
      ElMessage.error(mainResult.message)
      return
    }

    // 保存 Ansible 配置
    const ansibleResult = await featuresStore.updateAnsibleConfig(formData.ansible)
    if (!ansibleResult.success) {
      ElMessage.error('保存 Ansible 配置失败: ' + ansibleResult.message)
      return
    }

    // 保存 SSH 配置
    const sshResult = await featuresStore.updateSSHConfig(formData.ssh)
    if (!sshResult.success) {
      ElMessage.error('保存 SSH 配置失败: ' + sshResult.message)
      return
    }

    ElMessage.success('配置保存成功')
    
    // 重新加载配置
    await loadSettings()
    
    // 如果启用状态改变，提示刷新页面
    if (formData.enabled !== featuresStore.allFeatures.automation.enabled) {
      ElMessage.info({
        message: '功能状态已更改，请刷新页面以查看最新菜单',
        duration: 3000
      })
    }
  } catch (error) {
    console.error('Save settings failed:', error)
    ElMessage.error('保存配置失败: ' + (error.response?.data?.message || error.message))
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.form-item-tips {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

:deep(.el-form-item) {
  margin-bottom: 24px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
}

:deep(.el-card__header) {
  padding: 16px 20px;
  background-color: #f5f7fa;
}

:deep(.el-card__body) {
  padding: 24px 20px;
}
</style>

