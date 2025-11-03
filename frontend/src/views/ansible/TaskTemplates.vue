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

      <!-- 快速筛选 -->
      <div style="margin-bottom: 16px">
        <el-space wrap>
          <el-text>快速筛选：</el-text>
          <el-tag 
            :type="queryParams.risk_level === '' ? 'primary' : ''" 
            style="cursor: pointer"
            @click="filterByRisk('')"
          >
            全部 ({{ total }})
          </el-tag>
          <el-tag 
            type="success" 
            :effect="queryParams.risk_level === 'low' ? 'dark' : 'plain'"
            style="cursor: pointer"
            @click="filterByRisk('low')"
          >
            低风险
          </el-tag>
          <el-tag 
            type="warning" 
            :effect="queryParams.risk_level === 'medium' ? 'dark' : 'plain'"
            style="cursor: pointer"
            @click="filterByRisk('medium')"
          >
            中风险
          </el-tag>
          <el-tag 
            type="danger" 
            :effect="queryParams.risk_level === 'high' ? 'dark' : 'plain'"
            style="cursor: pointer"
            @click="filterByRisk('high')"
          >
            高风险
          </el-tag>
        </el-space>
      </div>

      <!-- 模板列表 -->
      <el-table 
        :data="templates" 
        v-loading="loading" 
        style="width: 100%"
        :default-sort="{ prop: 'id', order: 'descending' }"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="name" label="模板名称" min-width="150" show-overflow-tooltip />
        <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
        <el-table-column prop="tags" label="标签" min-width="100" show-overflow-tooltip />
        <el-table-column label="风险等级" min-width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getRiskLevelType(row.risk_level)">
              {{ getRiskLevelText(row.risk_level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="160">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="320" fixed="right" align="center">
          <template #default="{ row }">
            <el-button 
              size="small" 
              :type="row.is_favorite ? 'warning' : ''" 
              :icon="row.is_favorite ? Star : StarFilled"
              @click="toggleFavorite(row)"
            >
              {{ row.is_favorite ? '已收藏' : '收藏' }}
            </el-button>
            <el-button size="small" @click="handleView(row)">查看</el-button>
            <el-button size="small" type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="success" @click="handleClone(row)">克隆</el-button>
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
      width="80%"
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
        <el-form-item label="风险等级">
          <el-select v-model="templateForm.risk_level" placeholder="选择风险等级" :disabled="isViewMode" style="width: 100%">
            <el-option label="低风险" value="low">
              <span>低风险 - 读取/查询操作</span>
            </el-option>
            <el-option label="中风险" value="medium">
              <span>中风险 - 配置变更/重启服务</span>
            </el-option>
            <el-option label="高风险" value="high">
              <span>高风险 - 删除/格式化/破坏性操作</span>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="Playbook 内容" :required="!isViewMode">
          <div style="margin-bottom: 8px;">
            <el-text type="info" size="small">
              Playbook 必须以 <code>- name:</code> 开头的数组格式，请参考以下示例
            </el-text>
          </div>
          <el-input 
            v-model="templateForm.playbook_content" 
            type="textarea" 
            :rows="20"
            placeholder="请输入 Ansible Playbook 内容（YAML 格式）&#10;&#10;示例：&#10;- name: 安装并启动 nginx&#10;  hosts: all&#10;  become: yes&#10;  tasks:&#10;    - name: 安装 nginx&#10;      apt:&#10;        name: nginx&#10;        state: present&#10;    - name: 启动 nginx&#10;      service:&#10;        name: nginx&#10;        state: started" 
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
import { Plus, Star, StarFilled } from '@element-plus/icons-vue'
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
  page_size: 20,
  risk_level: ''
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
    
    // 加载收藏状态
    await loadFavoriteStatus()
  } catch (error) {
    console.error('加载模板失败:', error)
    ElMessage.error('加载模板失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 加载收藏状态
const loadFavoriteStatus = async () => {
  try {
    const res = await ansibleAPI.listFavorites('template')
    const favoriteIds = new Set((res.data?.data || []).map(f => f.target_id))
    templates.value.forEach(template => {
      template.is_favorite = favoriteIds.has(template.id)
    })
  } catch (error) {
    console.error('加载收藏状态失败:', error)
  }
}

// 切换收藏状态
const toggleFavorite = async (template) => {
  try {
    if (template.is_favorite) {
      await ansibleAPI.removeFavorite('template', template.id)
      template.is_favorite = false
      ElMessage.success('已取消收藏')
    } else {
      await ansibleAPI.addFavorite({
        target_type: 'template',
        target_id: template.id
      })
      template.is_favorite = true
      ElMessage.success('收藏成功')
    }
  } catch (error) {
    console.error('操作收藏失败:', error)
    ElMessage.error('操作失败: ' + (error.message || '未知错误'))
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
    risk_level: 'low',
    playbook_content: ''
  })
  dialogVisible.value = true
}

const handleView = (row) => {
  isViewMode.value = true // 查看模式，只读
  dialogTitle.value = '查看模板'
  Object.assign(templateForm, {
    id: row.id,
    name: row.name,
    description: row.description,
    tags: row.tags,
    risk_level: row.risk_level || 'low',
    playbook_content: row.playbook_content || ''
  })
  dialogVisible.value = true
}

const handleEdit = (row) => {
  isEdit.value = true
  isViewMode.value = false // 编辑模式，可编辑
  dialogTitle.value = '编辑模板'
  Object.assign(templateForm, {
    id: row.id,
    name: row.name,
    description: row.description,
    tags: row.tags,
    risk_level: row.risk_level || 'low',
    playbook_content: row.playbook_content || ''
  })
  dialogVisible.value = true
}

const handleClone = (row) => {
  isEdit.value = false
  isViewMode.value = false
  dialogTitle.value = '克隆模板'
  Object.assign(templateForm, {
    id: null,
    name: `${row.name} - 副本`,
    description: row.description,
    tags: row.tags,
    risk_level: row.risk_level || 'low',
    playbook_content: row.playbook_content || ''
  })
  dialogVisible.value = true
  ElMessage.info('请修改模板名称后保存')
}

const filterByRisk = (level) => {
  queryParams.risk_level = level
  queryParams.page = 1
  loadTemplates()
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
    const errorMsg = error.message || error.toString()
    if (errorMsg.includes('duplicate key') || errorMsg.includes('name already exists')) {
      ElMessage.error('保存失败：模板名称已存在，请使用其他名称')
    } else if (errorMsg.includes('playbook must be an array')) {
      ElMessage.error({
        message: 'Playbook 格式错误：必须以 "- name:" 开头的数组格式。请参考示例。',
        duration: 5000
      })
    } else if (errorMsg.includes('invalid playbook')) {
      ElMessage.error({
        message: 'Playbook YAML 格式错误：' + errorMsg,
        duration: 5000
      })
    } else {
      ElMessage.error('保存失败: ' + errorMsg)
    }
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

const getRiskLevelType = (riskLevel) => {
  const typeMap = {
    low: 'success',
    medium: 'warning',
    high: 'danger'
  }
  return typeMap[riskLevel] || 'info'
}

const getRiskLevelText = (riskLevel) => {
  const textMap = {
    low: '低风险',
    medium: '中风险',
    high: '高风险'
  }
  return textMap[riskLevel] || riskLevel || '未知'
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

