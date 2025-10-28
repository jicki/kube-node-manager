<template>
  <div class="report-settings">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>
            <el-icon><DataAnalysis /></el-icon>
            异常报告配置
          </span>
          <el-button type="primary" @click="handleAdd">
            <el-icon><Plus /></el-icon>
            新增报告
          </el-button>
        </div>
      </template>

      <el-alert
        title="定时报告说明"
        type="info"
        :closable="false"
        style="margin-bottom: 20px"
      >
        <p>配置定时异常报告，系统将按照指定的时间自动生成并推送报告到飞书或邮箱。</p>
        <p>支持按日/周/月频率生成报告，可选择特定集群或全部集群。</p>
      </el-alert>

      <el-table :data="configs" v-loading="loading" border>
        <el-table-column prop="id" label="ID" width="60" />
        
        <el-table-column prop="report_name" label="报告名称" min-width="150">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
            <span style="margin-left: 8px">{{ row.report_name }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="frequency" label="频率" width="100">
          <template #default="{ row }">
            {{ formatFrequency(row.frequency) }}
          </template>
        </el-table-column>

        <el-table-column prop="schedule" label="Cron 表达式" width="150" />

        <el-table-column label="推送渠道" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.feishu_enabled" type="success" size="small" style="margin-right: 4px">
              飞书
            </el-tag>
            <el-tag v-if="row.email_enabled" type="warning" size="small">
              邮件
            </el-tag>
            <el-tag v-if="!row.feishu_enabled && !row.email_enabled" type="info" size="small">
              未配置
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="last_run_time" label="最后执行" width="160">
          <template #default="{ row }">
            {{ row.last_run_time ? formatTime(row.last_run_time) : '-' }}
          </template>
        </el-table-column>

        <el-table-column prop="next_run_time" label="下次执行" width="160">
          <template #default="{ row }">
            {{ row.next_run_time ? formatTime(row.next_run_time) : '-' }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              type="primary"
              link
              @click="handleTest(row)"
              :loading="testingId === row.id"
            >
              测试
            </el-button>
            <el-button
              size="small"
              type="success"
              link
              @click="handleRunNow(row)"
              :loading="runningId === row.id"
            >
              立即执行
            </el-button>
            <el-button
              size="small"
              link
              @click="handleEdit(row)"
            >
              编辑
            </el-button>
            <el-button
              size="small"
              type="danger"
              link
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑报告配置' : '新增报告配置'"
      width="700px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="报告名称" prop="report_name">
          <el-input
            v-model="form.report_name"
            placeholder="例如：每日异常报告"
            maxlength="100"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="启用状态" prop="enabled">
          <el-switch
            v-model="form.enabled"
            active-text="启用"
            inactive-text="禁用"
          />
          <el-text type="info" size="small" style="margin-left: 12px">
            禁用后将不会自动执行
          </el-text>
        </el-form-item>

        <el-form-item label="报告频率" prop="frequency">
          <el-radio-group v-model="form.frequency" @change="handleFrequencyChange">
            <el-radio label="daily">每日</el-radio>
            <el-radio label="weekly">每周</el-radio>
            <el-radio label="monthly">每月</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="执行时间" prop="schedule">
          <el-input
            v-model="form.schedule"
            placeholder="Cron 表达式，例如：0 9 * * *"
          >
            <template #append>
              <el-button @click="showCronHelp">说明</el-button>
            </template>
          </el-input>
          <el-text type="info" size="small">
            当前配置：{{ cronDescription }}
          </el-text>
        </el-form-item>

        <el-form-item label="目标集群" prop="cluster_ids">
          <el-select
            v-model="selectedClusters"
            multiple
            collapse-tags
            collapse-tags-tooltip
            placeholder="留空表示所有集群"
            style="width: 100%"
            clearable
          >
            <el-option
              v-for="cluster in clusters"
              :key="cluster.id"
              :label="cluster.name"
              :value="cluster.id"
            />
          </el-select>
        </el-form-item>

        <el-divider>飞书推送配置</el-divider>

        <el-form-item label="启用飞书推送" prop="feishu_enabled">
          <el-switch v-model="form.feishu_enabled" />
        </el-form-item>

        <el-form-item
          v-if="form.feishu_enabled"
          label="Webhook URL"
          prop="feishu_webhook"
        >
          <el-input
            v-model="form.feishu_webhook"
            type="textarea"
            :rows="3"
            placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..."
          />
          <el-text type="info" size="small">
            请在飞书群聊中创建自定义机器人，并将 Webhook URL 粘贴到此处
          </el-text>
        </el-form-item>

        <el-divider>邮件推送配置（预留）</el-divider>

        <el-form-item label="启用邮件推送" prop="email_enabled">
          <el-switch v-model="form.email_enabled" disabled />
          <el-text type="info" size="small" style="margin-left: 12px">
            功能开发中，敬请期待
          </el-text>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- Cron 表达式帮助对话框 -->
    <el-dialog v-model="cronHelpVisible" title="Cron 表达式说明" width="600px">
      <el-descriptions :column="1" border>
        <el-descriptions-item label="格式">
          分 时 日 月 周
        </el-descriptions-item>
        <el-descriptions-item label="每日 9:00">
          0 9 * * *
        </el-descriptions-item>
        <el-descriptions-item label="每周一 9:00">
          0 9 * * 1
        </el-descriptions-item>
        <el-descriptions-item label="每月1号 9:00">
          0 9 1 * *
        </el-descriptions-item>
        <el-descriptions-item label="每6小时">
          0 */6 * * *
        </el-descriptions-item>
      </el-descriptions>

      <el-alert
        title="在线工具"
        type="success"
        :closable="false"
        style="margin-top: 16px"
      >
        推荐使用在线工具生成 Cron 表达式：
        <el-link type="primary" href="https://crontab.guru" target="_blank">
          crontab.guru
        </el-link>
      </el-alert>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DataAnalysis, Plus } from '@element-plus/icons-vue'
import {
  getReportConfigs,
  createReportConfig,
  updateReportConfig,
  deleteReportConfig,
  testReportSend,
  runReportNow
} from '@/api/anomaly'
import clusterApi from '@/api/cluster'

// 数据
const configs = ref([])
const clusters = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const cronHelpVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const testingId = ref(null)
const runningId = ref(null)
const formRef = ref(null)
const selectedClusters = ref([])

// 表单数据
const form = reactive({
  id: null,
  report_name: '',
  enabled: true,
  frequency: 'daily',
  schedule: '0 9 * * *',
  cluster_ids: '',
  feishu_enabled: false,
  feishu_webhook: '',
  email_enabled: false,
  email_recipients: ''
})

// 表单验证规则
const rules = {
  report_name: [
    { required: true, message: '请输入报告名称', trigger: 'blur' },
    { min: 2, max: 100, message: '长度在 2 到 100 个字符', trigger: 'blur' }
  ],
  schedule: [
    { required: true, message: '请输入 Cron 表达式', trigger: 'blur' }
  ],
  feishu_webhook: [
    {
      validator: (rule, value, callback) => {
        if (form.feishu_enabled && !value) {
          callback(new Error('启用飞书推送时，Webhook URL 不能为空'))
        } else if (value && !value.startsWith('https://')) {
          callback(new Error('Webhook URL 必须以 https:// 开头'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

// 计算 Cron 表达式描述
const cronDescription = computed(() => {
  if (!form.schedule) return '无效的 Cron 表达式'
  
  const parts = form.schedule.split(' ')
  if (parts.length !== 5) return '无效的 Cron 表达式'
  
  const [minute, hour, day, month, weekday] = parts
  
  if (form.frequency === 'daily') {
    return `每天 ${hour}:${minute.padStart(2, '0')}`
  } else if (form.frequency === 'weekly') {
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
    const weekdayText = weekday === '*' ? '每天' : weekdays[parseInt(weekday)] || '周一'
    return `每周${weekdayText} ${hour}:${minute.padStart(2, '0')}`
  } else if (form.frequency === 'monthly') {
    return `每月 ${day} 号 ${hour}:${minute.padStart(2, '0')}`
  }
  
  return form.schedule
})

// 格式化频率
const formatFrequency = (freq) => {
  const map = {
    daily: '每日',
    weekly: '每周',
    monthly: '每月'
  }
  return map[freq] || freq
}

// 格式化时间
const formatTime = (time) => {
  if (!time) return '-'
  const date = new Date(time)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 加载配置列表
const loadConfigs = async () => {
  loading.value = true
  try {
    const res = await getReportConfigs()
    configs.value = res.data?.data || []
  } catch (error) {
    ElMessage.error('加载报告配置失败：' + error.message)
  } finally {
    loading.value = false
  }
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const res = await clusterApi.getClusters({ page: 1, page_size: 100 })
    clusters.value = res.data?.data?.clusters || []
  } catch (error) {
    console.error('加载集群列表失败：', error)
  }
}

// 新增报告
const handleAdd = () => {
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

// 编辑报告
const handleEdit = (row) => {
  isEdit.value = true
  Object.assign(form, row)
  
  // 解析 cluster_ids
  if (row.cluster_ids) {
    try {
      selectedClusters.value = JSON.parse(row.cluster_ids)
    } catch (e) {
      selectedClusters.value = []
    }
  } else {
    selectedClusters.value = []
  }
  
  dialogVisible.value = true
}

// 删除报告
const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除报告配置 "${row.report_name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await deleteReportConfig(row.id)
    ElMessage.success('删除成功')
    loadConfigs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败：' + error.message)
    }
  }
}

// 测试发送
const handleTest = async (row) => {
  testingId.value = row.id
  try {
    await testReportSend(row.id)
    ElMessage.success('测试报告已发送，请检查飞书群聊')
  } catch (error) {
    ElMessage.error('测试失败：' + error.message)
  } finally {
    testingId.value = null
  }
}

// 立即执行
const handleRunNow = async (row) => {
  runningId.value = row.id
  try {
    await runReportNow(row.id)
    ElMessage.success('报告已生成并发送')
    loadConfigs()
  } catch (error) {
    ElMessage.error('执行失败：' + error.message)
  } finally {
    runningId.value = null
  }
}

// 频率变化时自动更新 Cron 表达式
const handleFrequencyChange = (value) => {
  if (value === 'daily') {
    form.schedule = '0 9 * * *' // 每天 9:00
  } else if (value === 'weekly') {
    form.schedule = '0 9 * * 1' // 每周一 9:00
  } else if (value === 'monthly') {
    form.schedule = '0 9 1 * *' // 每月1号 9:00
  }
}

// 显示 Cron 帮助
const showCronHelp = () => {
  cronHelpVisible.value = true
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    submitting.value = true
    try {
      // 构建提交数据
      const data = {
        ...form,
        cluster_ids: selectedClusters.value.length > 0 
          ? JSON.stringify(selectedClusters.value) 
          : ''
      }
      
      if (isEdit.value) {
        await updateReportConfig(form.id, data)
        ElMessage.success('更新成功')
      } else {
        await createReportConfig(data)
        ElMessage.success('创建成功')
      }
      
      dialogVisible.value = false
      loadConfigs()
    } catch (error) {
      ElMessage.error('保存失败：' + error.message)
    } finally {
      submitting.value = false
    }
  })
}

// 重置表单
const resetForm = () => {
  Object.assign(form, {
    id: null,
    report_name: '',
    enabled: true,
    frequency: 'daily',
    schedule: '0 9 * * *',
    cluster_ids: '',
    feishu_enabled: false,
    feishu_webhook: '',
    email_enabled: false,
    email_recipients: ''
  })
  selectedClusters.value = []
  formRef.value?.resetFields()
}

// 初始化
onMounted(() => {
  loadConfigs()
  loadClusters()
})
</script>

<style scoped>
.report-settings {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header span {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
}
</style>

