<template>
  <div class="schedule-manage">
    <el-card class="header-card">
      <template #header>
        <div class="card-header">
          <span>定时任务管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            创建定时任务
          </el-button>
        </div>
      </template>
    </el-card>

    <!-- 筛选器 -->
    <el-card style="margin-top: 20px">
      <el-form :inline="true" :model="queryParams">
        <el-form-item label="状态">
          <el-select v-model="queryParams.enabled" placeholder="全部" clearable style="width: 120px">
            <el-option label="已启用" :value="true" />
            <el-option label="已禁用" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键字">
          <el-input v-model="queryParams.keyword" placeholder="搜索任务名称" clearable style="width: 200px" @keyup.enter="handleQuery" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleQuery">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
          <el-button @click="loadSchedules" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 定时任务列表 -->
    <el-card style="margin-top: 20px">
      <el-table :data="schedules" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="任务名称" min-width="200" />
        <el-table-column label="Cron 表达式" width="150">
          <template #default="{ row }">
            <el-tag type="info">{{ row.cron_expr }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="模板" width="150">
          <template #default="{ row }">
            {{ row.template?.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="清单" width="150">
          <template #default="{ row }">
            {{ row.inventory?.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">
              {{ row.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="执行次数" width="100">
          <template #default="{ row }">
            {{ row.run_count || 0 }}
          </template>
        </el-table-column>
        <el-table-column label="下次执行" width="180">
          <template #default="{ row }">
            {{ row.next_run_at ? formatDate(row.next_run_at) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="350" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button 
              size="small" 
              :type="row.enabled ? 'warning' : 'success'"
              @click="handleToggle(row)"
            >
              {{ row.enabled ? '禁用' : '启用' }}
            </el-button>
            <el-button 
              size="small" 
              type="primary" 
              @click="handleRunNow(row)"
              :disabled="!row.enabled"
            >
              立即执行
            </el-button>
            <el-button 
              size="small" 
              type="danger" 
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="queryParams.page"
        v-model:page-size="queryParams.page_size"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleQuery"
        @current-change="handleQuery"
        style="margin-top: 20px"
      />
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="dialogTitle" 
      width="700px"
      @close="resetForm"
    >
      <el-form :model="scheduleForm" label-width="120px" ref="formRef" :rules="formRules">
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="scheduleForm.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input 
            v-model="scheduleForm.description" 
            type="textarea" 
            :rows="2"
            placeholder="请输入任务描述"
          />
        </el-form-item>
        <el-form-item label="选择模板" prop="template_id">
          <el-select v-model="scheduleForm.template_id" placeholder="选择模板" style="width: 100%">
            <el-option 
              v-for="template in templates" 
              :key="template.id" 
              :label="template.name" 
              :value="template.id" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="主机清单" prop="inventory_id">
          <el-select v-model="scheduleForm.inventory_id" placeholder="选择主机清单" style="width: 100%">
            <el-option 
              v-for="inventory in inventories" 
              :key="inventory.id" 
              :label="inventory.name" 
              :value="inventory.id" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="集群">
          <el-select v-model="scheduleForm.cluster_id" placeholder="选择集群（可选）" clearable style="width: 100%">
            <el-option 
              v-for="cluster in clusters" 
              :key="cluster.id" 
              :label="cluster.name" 
              :value="cluster.id" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Cron 表达式" prop="cron_expr">
          <el-input v-model="scheduleForm.cron_expr" placeholder="例如: */5 * * * * (每5分钟) 或 0 */5 * * * * (每5分钟第0秒)">
            <template #append>
              <el-button @click="showCronHelp">帮助</el-button>
            </template>
          </el-input>
          <div class="cron-preview" v-if="cronPreview">
            <el-text type="info" size="small">{{ cronPreview }}</el-text>
          </div>
          <el-text type="info" size="small" style="display: block; margin-top: 8px;">
            支持标准 5 字段格式（分 时 日 月 周）和扩展 6 字段格式（秒 分 时 日 月 周）
          </el-text>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="scheduleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- Cron 帮助对话框 -->
    <el-dialog v-model="cronHelpVisible" title="Cron 表达式帮助" width="700px">
      <div class="cron-help">
        <el-alert 
          title="支持两种格式" 
          type="info" 
          description="标准 5 字段格式会自动转换为 6 字段格式，默认在第 0 秒执行" 
          :closable="false"
          style="margin-bottom: 20px"
        />
        
        <h4>标准 Cron 格式（5个字段）- 推荐</h4>
        <pre>分 时 日 月 星期

示例：
0 * * * *     - 每小时整点执行
0 0 * * *     - 每天午夜执行
0 0 * * 0     - 每周日午夜执行
0 0 1 * *     - 每月1号午夜执行
*/5 * * * *   - 每5分钟执行
*/1 * * * *   - 每1分钟执行
0 9-17 * * 1-5 - 工作日9点到17点每小时执行</pre>

        <h4 style="margin-top: 20px">扩展 Cron 格式（6个字段）- 支持秒级精度</h4>
        <pre>秒 分 时 日 月 星期

示例：
0 0 * * * *   - 每小时整点执行
0 */5 * * * * - 每5分钟执行
*/30 * * * * * - 每30秒执行
0 0 0 * * *   - 每天午夜执行</pre>

        <h4 style="margin-top: 20px">特殊表达式</h4>
        <pre>@hourly   - 每小时执行
@daily    - 每天执行
@weekly   - 每周执行
@monthly  - 每月执行
@yearly   - 每年执行</pre>

        <h4 style="margin-top: 20px">字段说明</h4>
        <el-table :data="cronFields" style="margin-top: 10px">
          <el-table-column prop="field" label="字段" width="100" />
          <el-table-column prop="range" label="取值范围" width="150" />
          <el-table-column prop="special" label="特殊字符" />
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import * as ansibleAPI from '@/api/ansible'
import clusterAPI from '@/api/cluster'

// 数据
const schedules = ref([])
const total = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const cronHelpVisible = ref(false)
const submitting = ref(false)
const formRef = ref(null)

const queryParams = reactive({
  page: 1,
  page_size: 20,
  enabled: null,
  keyword: ''
})

const scheduleForm = reactive({
  name: '',
  description: '',
  template_id: null,
  inventory_id: null,
  cluster_id: null,
  cron_expr: '',
  enabled: true
})

const templates = ref([])
const inventories = ref([])
const clusters = ref([])
const editingId = ref(null)

const formRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  template_id: [{ required: true, message: '请选择模板', trigger: 'change' }],
  inventory_id: [{ required: true, message: '请选择主机清单', trigger: 'change' }],
  cron_expr: [
    { required: true, message: '请输入 Cron 表达式', trigger: 'blur' },
    { 
      pattern: /^(@(hourly|daily|weekly|monthly|yearly))|(((\*|([0-9]|[1-5][0-9]))(\/[0-9]+)?|\*\/[0-9]+|[0-9](-[0-9]+)?(,[0-9](-[0-9]+)?)*)\s+){4,}((\*|([0-6]))(\/[0-9]+)?|\*\/[0-9]+|[0-6](-[0-6]+)?(,[0-6](-[0-6]+)?)*)$/,
      message: 'Cron 表达式格式不正确', 
      trigger: 'blur' 
    }
  ]
}

const cronFields = [
  { field: '分钟', range: '0-59', special: '* , - /' },
  { field: '小时', range: '0-23', special: '* , - /' },
  { field: '日期', range: '1-31', special: '* , - / ?' },
  { field: '月份', range: '1-12', special: '* , - /' },
  { field: '星期', range: '0-6', special: '* , - / ?' }
]

// 计算属性
const dialogTitle = computed(() => editingId.value ? '编辑定时任务' : '创建定时任务')

const cronPreview = computed(() => {
  if (!scheduleForm.cron_expr) return ''
  
  const expr = scheduleForm.cron_expr.trim()
  const specialCrons = {
    '@hourly': '每小时执行',
    '@daily': '每天执行',
    '@weekly': '每周执行',
    '@monthly': '每月执行',
    '@yearly': '每年执行'
  }
  
  if (specialCrons[expr]) {
    return specialCrons[expr]
  }
  
  // 简单的预览（可以使用库如 cronstrue 进行更好的解析）
  const parts = expr.split(/\s+/)
  if (parts.length >= 5) {
    return `表达式：${parts.slice(0, 5).join(' ')}`
  }
  
  return ''
})

// 方法
const loadSchedules = async () => {
  loading.value = true
  try {
    const res = await ansibleAPI.listSchedules(queryParams)
    schedules.value = res.data?.data || []
    total.value = res.data?.total || 0
  } catch (error) {
    console.error('加载定时任务失败:', error)
    ElMessage.error('加载定时任务失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const loadTemplates = async () => {
  try {
    const res = await ansibleAPI.listTemplates({ page_size: 100 })
    templates.value = res.data?.data || []
  } catch (error) {
    console.error('加载模板失败:', error)
  }
}

const loadInventories = async () => {
  try {
    const res = await ansibleAPI.listInventories({ page_size: 100 })
    inventories.value = res.data?.data || []
  } catch (error) {
    console.error('加载清单失败:', error)
  }
}

const loadClusters = async () => {
  try {
    const res = await clusterAPI.getClusters()
    clusters.value = res.data?.data?.clusters || []
  } catch (error) {
    console.error('加载集群失败:', error)
  }
}

const handleQuery = () => {
  queryParams.page = 1
  loadSchedules()
}

const handleReset = () => {
  queryParams.enabled = null
  queryParams.keyword = ''
  handleQuery()
}

const showCreateDialog = () => {
  editingId.value = null
  resetForm()
  dialogVisible.value = true
  loadTemplates()
  loadInventories()
  loadClusters()
}

const handleEdit = (row) => {
  editingId.value = row.id
  Object.assign(scheduleForm, {
    name: row.name,
    description: row.description,
    template_id: row.template_id,
    inventory_id: row.inventory_id,
    cluster_id: row.cluster_id,
    cron_expr: row.cron_expr,
    enabled: row.enabled
  })
  dialogVisible.value = true
  loadTemplates()
  loadInventories()
  loadClusters()
}

const handleSubmit = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    submitting.value = true
    try {
      if (editingId.value) {
        await ansibleAPI.updateSchedule(editingId.value, scheduleForm)
        ElMessage.success('定时任务已更新')
      } else {
        await ansibleAPI.createSchedule(scheduleForm)
        ElMessage.success('定时任务已创建')
      }
      dialogVisible.value = false
      loadSchedules()
    } catch (error) {
      ElMessage.error('操作失败: ' + (error.message || '未知错误'))
    } finally {
      submitting.value = false
    }
  })
}

const handleToggle = async (row) => {
  try {
    await ansibleAPI.toggleSchedule(row.id, !row.enabled)
    ElMessage.success(row.enabled ? '已禁用' : '已启用')
    loadSchedules()
  } catch (error) {
    ElMessage.error('操作失败: ' + (error.message || '未知错误'))
  }
}

const handleRunNow = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要立即执行定时任务 "${row.name}" 吗？`,
      '确认执行',
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      }
    )
    await ansibleAPI.runScheduleNow(row.id)
    ElMessage.success('任务已触发执行')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('执行失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除定时任务 "${row.name}" 吗？此操作不可恢复。`,
      '删除确认',
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      }
    )
    await ansibleAPI.deleteSchedule(row.id)
    ElMessage.success('删除成功')
    loadSchedules()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const resetForm = () => {
  Object.assign(scheduleForm, {
    name: '',
    description: '',
    template_id: null,
    inventory_id: null,
    cluster_id: null,
    cron_expr: '',
    enabled: true
  })
  if (formRef.value) {
    formRef.value.clearValidate()
  }
}

const showCronHelp = () => {
  cronHelpVisible.value = true
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 生命周期
onMounted(() => {
  loadSchedules()
})
</script>

<style scoped>
.schedule-manage {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cron-preview {
  margin-top: 8px;
  padding: 8px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.cron-help {
  line-height: 1.8;
}

.cron-help h4 {
  margin-bottom: 10px;
  color: #303133;
}

.cron-help pre {
  background-color: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
}
</style>

