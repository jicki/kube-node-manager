<template>
  <div class="ansible-templates">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>任务模板管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            创建模板
          </el-button>
        </div>
      </template>

      <!-- 模板列表 -->
      <el-table :data="templates" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="模板名称" min-width="200" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="tags" label="标签" width="150" />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleView(row)">查看</el-button>
            <el-button size="small" type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="queryParams.page"
        v-model:page-size="queryParams.page_size"
        :page-sizes="[10, 20, 50]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadTemplates"
        @current-change="loadTemplates"
        style="margin-top: 20px"
      />
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="dialogTitle" 
      width="60%"
      :close-on-click-modal="false"
    >
      <el-form :model="templateForm" label-width="120px">
        <el-form-item label="模板名称" :required="!isViewMode">
          <el-input v-model="templateForm.name" placeholder="请输入模板名称" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="templateForm.description" type="textarea" :rows="3" placeholder="请输入描述" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="templateForm.tags" placeholder="多个标签用逗号分隔" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item label="Playbook 内容" :required="!isViewMode">
          <el-input 
            v-model="templateForm.playbook_content" 
            type="textarea" 
            :rows="15"
            placeholder="请输入 Ansible Playbook YAML 内容&#10;&#10;示例：&#10;---&#10;- name: 示例 Playbook&#10;  hosts: all&#10;  tasks:&#10;    - name: Ping 测试&#10;      ping:" 
            style="font-family: 'Courier New', monospace; font-size: 13px;"
            :disabled="isViewMode"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">{{ isViewMode ? '关闭' : '取消' }}</el-button>
          <el-button v-if="!isViewMode" type="primary" @click="handleSave" :loading="saving">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import * as ansibleAPI from '@/api/ansible'

const templates = ref([])
const total = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const saving = ref(false)
const isEdit = ref(false)
const isViewMode = ref(false) // 是否为查看模式（只读）

const queryParams = reactive({
  page: 1,
  page_size: 20
})

const templateForm = reactive({
  id: null,
  name: '',
  description: '',
  tags: '',
  playbook_content: ''
})

const loadTemplates = async () => {
  loading.value = true
  try {
    const res = await ansibleAPI.listTemplates(queryParams)
    console.log('模板列表响应:', res)
    console.log('模板数据:', res.data)
    // axios拦截器返回完整response，所以路径是: res.data.data 和 res.data.total
    templates.value = res.data?.data || []
    total.value = res.data?.total || 0
    console.log('已加载模板:', templates.value.length, '个')
  } catch (error) {
    console.error('加载模板失败:', error)
    ElMessage.error('加载模板失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  isEdit.value = false
  isViewMode.value = false // 创建模式，可编辑
  dialogTitle.value = '创建模板'
  Object.assign(templateForm, {
    id: null,
    name: '',
    description: '',
    tags: '',
    playbook_content: ''
  })
  dialogVisible.value = true
}

const handleView = (row) => {
  isViewMode.value = true // 查看模式，只读
  dialogTitle.value = '查看模板'
  Object.assign(templateForm, row)
  dialogVisible.value = true
}

const handleEdit = (row) => {
  isEdit.value = true
  isViewMode.value = false // 编辑模式，可编辑
  dialogTitle.value = '编辑模板'
  Object.assign(templateForm, row)
  dialogVisible.value = true
}

const handleSave = async () => {
  if (!templateForm.name || !templateForm.playbook_content) {
    ElMessage.warning('请填写必填项')
    return
  }

  saving.value = true
  try {
    if (isEdit.value) {
      await ansibleAPI.updateTemplate(templateForm.id, templateForm)
      ElMessage.success('模板已更新')
    } else {
      await ansibleAPI.createTemplate(templateForm)
      ElMessage.success('模板已创建')
    }
    dialogVisible.value = false
    loadTemplates()
  } catch (error) {
    ElMessage.error('保存失败: ' + error.message)
  } finally {
    saving.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除此模板吗？删除后无法恢复。', 
      '删除确认', 
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await ansibleAPI.deleteTemplate(row.id)
    ElMessage.success('模板已删除')
    loadTemplates()
  } catch (error) {
    if (error !== 'cancel') {
      const errorMsg = error.message || error.toString()
      if (errorMsg.includes('tasks are using this template')) {
        // 提取任务数量
        const match = errorMsg.match(/(\d+) tasks/)
        const taskCount = match ? match[1] : '若干'
        ElMessage.error({
          message: `无法删除：有 ${taskCount} 个任务正在使用此模板。请先删除这些任务后再试。`,
          duration: 5000
        })
      } else {
        ElMessage.error('删除失败: ' + errorMsg)
      }
    }
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadTemplates()
})
</script>

<style scoped>
.ansible-templates {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>

