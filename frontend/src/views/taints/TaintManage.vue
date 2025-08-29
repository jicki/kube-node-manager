<template>
  <div class="taint-manage">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">污点管理</h1>
        <p class="page-description">管理Kubernetes节点污点，控制Pod调度策略</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="showAddDialog">
          <el-icon><Plus /></el-icon>
          添加污点
        </el-button>
        <el-button @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="24" class="stats-row">
      <el-col :xs="8" :sm="6">
        <el-card class="stat-card stat-card-danger">
          <div class="stat-content">
            <div class="stat-value">{{ taintStats.noSchedule }}</div>
            <div class="stat-label">NoSchedule</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="8" :sm="6">
        <el-card class="stat-card stat-card-warning">
          <div class="stat-content">
            <div class="stat-value">{{ taintStats.preferNoSchedule }}</div>
            <div class="stat-label">PreferNoSchedule</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="8" :sm="6">
        <el-card class="stat-card stat-card-error">
          <div class="stat-content">
            <div class="stat-value">{{ taintStats.noExecute }}</div>
            <div class="stat-label">NoExecute</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="6">
        <el-card class="stat-card stat-card-info">
          <div class="stat-content">
            <div class="stat-value">{{ taintStats.total }}</div>
            <div class="stat-label">总污点数</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 污点模板列表 -->
    <div class="taint-grid">
      <div
        v-for="template in taints"
        :key="template.id"
        class="taint-card"
      >
        <div class="taint-header">
          <div class="taint-key-value">
            <div class="taint-key">{{ template.name }}</div>
            <div class="taint-value">{{ (template.taints || []).length }} 个污点</div>
          </div>
          <div class="taint-actions">
            <el-dropdown @command="(cmd) => handleTaintAction(cmd, template)">
              <el-button type="text" size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">
                    <el-icon><Edit /></el-icon>
                    编辑
                  </el-dropdown-item>
                  <el-dropdown-item command="copy">
                    <el-icon><CopyDocument /></el-icon>
                    复制
                  </el-dropdown-item>
                  <el-dropdown-item command="apply">
                    <el-icon><Plus /></el-icon>
                    应用到节点
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided>
                    <el-icon><Delete /></el-icon>
                    删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <div class="taint-meta">
          <el-tag type="primary" size="small">
            污点模板
          </el-tag>
          <span class="create-time">{{ template.created_at }}</span>
        </div>

        <div v-if="template.description" class="taint-description">
          {{ template.description }}
        </div>

        <div class="taint-content">
          <div class="taints-title">包含污点:</div>
          <div class="taints-list">
            <el-tag
              v-for="taint in template.taints || []"
              :key="`${taint.key}-${taint.effect}`"
              size="small"
              class="taint-item-tag"
              :type="getTaintEffectType(taint.effect)"
            >
              {{ taint.key }}{{ taint.value ? `=${taint.value}` : '' }}:{{ taint.effect }}
            </el-tag>
          </div>
        </div>

        <div class="taint-actions">
          <el-button-group size="small">
            <el-button @click="applyTemplateToNodes(template)">
              <el-icon><Plus /></el-icon>
              应用到节点
            </el-button>
            <el-button @click="editTemplate(template)">
              <el-icon><Edit /></el-icon>
              编辑模板
            </el-button>
          </el-button-group>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="taints.length === 0" class="empty-state">
        <el-empty description="暂无污点数据" :image-size="80">
          <el-button type="primary" @click="showAddDialog">
            <el-icon><Plus /></el-icon>
            添加第一个污点
          </el-button>
        </el-empty>
      </div>
    </div>

    <!-- 添加/编辑污点对话框 -->
    <el-dialog
      v-model="taintDialogVisible"
      :title="isEditing ? '编辑模板' : '创建模板'"
      width="900px"
      class="template-dialog"
    >
              <el-form
        ref="taintFormRef"
        :model="taintForm"
        :rules="taintRules"
        label-width="110px"
        style="margin-top: 20px;"
      >
        <el-form-item label="模板名称" prop="name">
          <el-input
            v-model="taintForm.name"
            placeholder="输入模板名称，如：Master节点污点、GPU节点污点"
          />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="taintForm.description"
            type="textarea"
            :rows="2"
            placeholder="模板用途描述"
          />
        </el-form-item>

        <el-divider content-position="left">污点配置</el-divider>

        <div class="taints-config">
          <div 
            v-for="(taint, index) in taintForm.taints" 
            :key="index"
            class="taint-config-item"
          >
            <el-row :gutter="12" align="middle" class="taint-row">
              <el-col :xs="24" :sm="8">
                <el-form-item 
                  :prop="`taints.${index}.key`" 
                  :rules="[{ required: true, message: '请输入污点键', trigger: 'blur' }]"
                  style="margin-bottom: 12px;"
                >
                  <el-input
                    v-model="taint.key"
                    placeholder="污点键，如：node-role、dedicated"
                    size="large"
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="7">
                <el-form-item style="margin-bottom: 12px;">
                  <el-input
                    v-model="taint.value"
                    placeholder="污点值，如：master、gpu（可为空）"
                    size="large"
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="7">
                <el-form-item 
                  :prop="`taints.${index}.effect`" 
                  :rules="[{ required: true, message: '请选择效果', trigger: 'change' }]"
                  style="margin-bottom: 12px;"
                >
                  <el-select 
                    v-model="taint.effect" 
                    placeholder="选择污点效果" 
                    style="width: 100%"
                    size="large"
                  >
                    <el-option label="NoSchedule - 禁止调度" value="NoSchedule" />
                    <el-option label="PreferNoSchedule - 尽量不调度" value="PreferNoSchedule" />
                    <el-option label="NoExecute - 禁止执行" value="NoExecute" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="2" class="delete-col">
                <el-button
                  type="danger"
                  size="large"
                  :icon="Delete"
                  circle
                  @click="removeTaint(index)"
                  :disabled="taintForm.taints.length === 1"
                />
              </el-col>
            </el-row>
          </div>
          
          <el-button
            type="dashed"
            block
            size="large"
            @click="addTaint"
            :icon="Plus"
            class="add-taint-btn"
          >
            添加污点
          </el-button>
        </div>

        <el-alert
          title="污点效果说明"
          type="info"
          :closable="false"
          show-icon
          style="margin-top: 20px;"
        >
          <ul style="margin: 0; padding-left: 20px;">
            <li>NoSchedule: 新的Pod不会调度到该节点，已存在的Pod不受影响</li>
            <li>PreferNoSchedule: 尽量避免调度Pod到该节点，但不是强制的</li>
            <li>NoExecute: 不仅不调度新Pod，还会驱逐已存在的Pod</li>
          </ul>
        </el-alert>
      </el-form>

      <template #footer>
        <el-button @click="taintDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="saving"
          @click="handleSaveTaint"
        >
          {{ isEditing ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 应用模板对话框 -->
    <el-dialog
      v-model="applyDialogVisible"
      :title="`应用模板: ${selectedTemplate?.name}`"
      width="600px"
    >
      <div class="template-info">
        <h4>模板包含的污点:</h4>
        <div class="template-taints">
          <el-tag
            v-for="taint in selectedTemplate?.taints || []"
            :key="`${taint.key}-${taint.effect}`"
            class="taint-tag"
            :type="getTaintEffectType(taint.effect)"
          >
            {{ taint.key }}{{ taint.value ? `=${taint.value}` : '' }}:{{ taint.effect }}
          </el-tag>
        </div>
      </div>

      <el-divider />

      <el-form label-width="100px">
        <el-form-item label="选择节点" required>
          <el-select
            v-model="selectedNodes"
            multiple
            filterable
            placeholder="选择要应用模板的节点"
            style="width: 100%"
          >
            <el-option
              v-for="node in availableNodes"
              :key="node.name"
              :label="`${node.name} (${node.status})`"
              :value="node.name"
            >
              <div class="node-option">
                <span class="node-name">{{ node.name }}</span>
                <el-tag 
                  :type="node.status === 'Ready' ? 'success' : 'danger'" 
                  size="small"
                >
                  {{ node.status }}
                </el-tag>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="applyDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="applying"
          @click="handleApplyTemplate"
        >
          应用到节点
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import taintApi from '@/api/taint'
import nodeApi from '@/api/node'
import { useClusterStore } from '@/store/modules/cluster'
import { formatTaintEffect } from '@/utils/format'
import {
  Plus,
  Refresh,
  Edit,
  Delete,
  MoreFilled,
  CopyDocument
} from '@element-plus/icons-vue'

// 响应式数据
const loading = ref(false)
const saving = ref(false)
const applying = ref(false)
const taintDialogVisible = ref(false)
const applyDialogVisible = ref(false)
const isEditing = ref(false)
const taintFormRef = ref()

// 数据
const taints = ref([])
const availableNodes = ref([])
const selectedTemplate = ref(null)
const selectedNodes = ref([])

// 表单数据
const taintForm = reactive({
  name: '',
  description: '',
  taints: [{ key: '', value: '', effect: '' }]
})

// 表单验证规则
const taintRules = {
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' }
  ]
}

// 计算属性
const taintStats = computed(() => {
  const total = taints.value.length
  const noSchedule = taints.value.filter(t => t.effect === 'NoSchedule').length
  const preferNoSchedule = taints.value.filter(t => t.effect === 'PreferNoSchedule').length
  const noExecute = taints.value.filter(t => t.effect === 'NoExecute').length
  
  return { total, noSchedule, preferNoSchedule, noExecute }
})

// 获取污点效果类型
const getTaintEffectType = (effect) => {
  const typeMap = {
    NoSchedule: 'danger',
    PreferNoSchedule: 'warning',
    NoExecute: 'error'
  }
  return typeMap[effect] || 'info'
}

// 方法
const fetchTaints = async () => {
  try {
    loading.value = true
    const response = await taintApi.getTemplateList()
    if (response.data && response.data.code === 200) {
      const data = response.data.data
      taints.value = data.templates || []
    } else {
      taints.value = []
    }
  } catch (error) {
    console.warn('获取污点模板失败:', error)
    taints.value = []
  } finally {
    loading.value = false
  }
}

const fetchNodes = async () => {
  try {
    const clusterStore = useClusterStore()
    const clusterName = clusterStore.currentClusterName
    
    // 如果没有集群，直接设置为空数组
    if (!clusterName) {
      availableNodes.value = []
      return
    }
    
    const response = await nodeApi.getNodes({
      cluster_name: clusterName
    })
    // 后端返回格式: { code, message, data: [...] } - data直接是节点数组
    availableNodes.value = response.data.data || []
  } catch (error) {
    console.error('获取节点数据失败:', error)
    availableNodes.value = []
  }
}

const refreshData = () => {
  fetchTaints()
  fetchNodes()
}

// 显示添加对话框
const showAddDialog = () => {
  isEditing.value = false
  resetTaintForm()
  taintDialogVisible.value = true
}

// 重置表单
const resetTaintForm = () => {
  Object.assign(taintForm, {
    name: '',
    description: '',
    taints: [{ key: '', value: '', effect: '' }]
  })
}

// 添加污点
const addTaint = () => {
  taintForm.taints.push({ key: '', value: '', effect: '' })
}

// 移除污点
const removeTaint = (index) => {
  if (taintForm.taints.length > 1) {
    taintForm.taints.splice(index, 1)
  }
}

// 处理污点操作
const handleTaintAction = (command, template) => {
  switch (command) {
    case 'edit':
      editTemplate(template)
      break
    case 'copy':
      copyTemplate(template)
      break
    case 'apply':
      applyTemplateToNodes(template)
      break
    case 'delete':
      deleteTemplate(template)
      break
  }
}

// 编辑模板
const editTemplate = (template) => {
  isEditing.value = true
  
  Object.assign(taintForm, {
    name: template.name,
    description: template.description || '',
    taints: template.taints && template.taints.length > 0 ? [...template.taints] : [{ key: '', value: '', effect: '' }]
  })
  
  // 保存当前编辑的模板ID
  taintForm.id = template.id
  
  taintDialogVisible.value = true
}

// 复制模板
const copyTemplate = (template) => {
  const taintsText = (template.taints || [])
    .map(taint => `${taint.key}${taint.value ? `=${taint.value}` : ''}:${taint.effect}`)
    .join(', ')
  const text = `${template.name}: ${taintsText}`
  
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('模板信息已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

// 删除模板
const deleteTemplate = (template) => {
  ElMessageBox.confirm(
    `确认删除模板 "${template.name}" 吗？此操作不可撤销。`,
    '删除模板',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await taintApi.deleteTemplate(template.id)
      ElMessage.success('模板已删除')
      refreshData()
    } catch (error) {
      ElMessage.error(`删除模板失败: ${error.message}`)
    }
  }).catch(() => {
    // 用户取消
  })
}

// 保存模板
const handleSaveTaint = async () => {
  try {
    await taintFormRef.value.validate()
    saving.value = true
    
    // 验证污点配置
    const validTaints = taintForm.taints.filter(taint => taint.key.trim() && taint.effect)
    if (validTaints.length === 0) {
      ElMessage.error('请至少添加一个有效的污点')
      return
    }
    
    const templateData = {
      name: taintForm.name,
      description: taintForm.description,
      taints: validTaints.map(taint => ({
        key: taint.key.trim(),
        value: taint.value.trim(),
        effect: taint.effect
      }))
    }
    
    if (isEditing.value) {
      // 更新模板
      await taintApi.updateTemplate(taintForm.id, templateData)
      ElMessage.success('模板更新成功')
    } else {
      // 创建新模板
      await taintApi.createTemplate(templateData)
      ElMessage.success('模板创建成功')
    }
    
    taintDialogVisible.value = false
    refreshData()
    
  } catch (error) {
    ElMessage.error(error.message || '保存模板失败')
  } finally {
    saving.value = false
  }
}

// 应用模板到节点
const applyTemplateToNodes = (template) => {
  selectedTemplate.value = template
  applyDialogVisible.value = true
  fetchNodes() // 获取节点列表
}

// 应用模板
const handleApplyTemplate = async () => {
  try {
    if (selectedNodes.value.length === 0) {
      ElMessage.error('请选择要应用的节点')
      return
    }
    
    const clusterStore = useClusterStore()
    const clusterName = clusterStore.currentClusterName
    
    if (!clusterName) {
      ElMessage.error('请先选择集群')
      return
    }
    
    applying.value = true
    
    const applyData = {
      cluster_name: clusterName,
      node_names: selectedNodes.value,
      template_id: selectedTemplate.value.id,
      operation: 'add'
    }
    
    await taintApi.applyTemplate(applyData)
    ElMessage.success('模板应用成功')
    applyDialogVisible.value = false
    
  } catch (error) {
    ElMessage.error(`应用模板失败: ${error.message}`)
  } finally {
    applying.value = false
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped>
.node-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  font-size: 14px;
}

.node-name {
  font-weight: 500;
  color: #333;
  letter-spacing: 0.3px;
}

:deep(.el-form-item) {
  margin-bottom: 22px;
}

:deep(.el-form-item__label) {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  letter-spacing: 0.2px;
  line-height: 1.6;
}

:deep(.el-input__wrapper) {
  font-size: 14px;
  padding: 12px 15px;
}

:deep(.el-select) {
  font-size: 14px;
}

:deep(.el-select__wrapper) {
  padding: 8px 15px;
  min-height: 44px;
}

.taint-manage {
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

.stat-card-danger {
  border-left: 4px solid #ff4d4f;
}

.stat-card-warning {
  border-left: 4px solid #faad14;
}

.stat-card-error {
  border-left: 4px solid #f50;
}

.stat-card-info {
  border-left: 4px solid #1890ff;
}

.stat-content {
  padding: 16px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 12px;
  color: #666;
}

.taint-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.taint-card {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 20px;
  background: #fff;
  transition: all 0.3s;
}

.taint-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.taint-noschedule {
  border-left: 4px solid #ff4d4f;
}

.taint-prefer-no-schedule {
  border-left: 4px solid #faad14;
}

.taint-no-execute {
  border-left: 4px solid #f50;
}

.taint-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.taint-key-value {
  flex: 1;
}

.taint-key {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.taint-value {
  font-size: 14px;
  color: #666;
  font-family: 'Monaco', 'Menlo', monospace;
}

.taint-nodes {
  margin-bottom: 16px;
}

.nodes-title {
  font-size: 13px;
  color: #666;
  font-weight: 500;
  margin-bottom: 8px;
}

.nodes-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.node-tag {
  font-size: 11px;
  height: 20px;
  line-height: 18px;
}

.more-nodes {
  font-size: 12px;
  color: #999;
}

.taint-actions {
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
}

.empty-state {
  grid-column: 1 / -1;
  padding: 40px;
  text-align: center;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }
  
  .taint-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .stats-row .el-col {
    margin-bottom: 16px;
  }
}

/* 新增样式 */
.taints-config {
  margin-top: 16px;
}

.taint-config-item {
  margin-bottom: 16px;
}

.create-time {
  color: #999;
  font-size: 12px;
}

.taint-content {
  margin-bottom: 16px;
}

.taints-title {
  font-size: 13px;
  color: #666;
  font-weight: 500;
  margin-bottom: 8px;
}

.taints-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.taint-item-tag {
  font-size: 11px;
  height: 20px;
  line-height: 18px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.template-info {
  margin-bottom: 16px;
}

.template-info h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #333;
}

.template-taints {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.taint-tag {
  font-family: 'Monaco', 'Menlo', monospace;
}

.taint-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 13px;
}

.taint-description {
  font-size: 13px;
  color: #666;
  margin-bottom: 16px;
  line-height: 1.5;
}

/* 表单优化样式 */
.template-dialog {
  --el-dialog-border-radius: 8px;
}

.template-dialog :deep(.el-dialog__body) {
  padding: 20px 30px 30px;
}

.taint-row {
  margin-bottom: 8px;
}

.delete-col {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding-top: 4px;
}

.add-taint-btn {
  margin-top: 12px;
  height: 44px;
  border-style: dashed;
  border-color: #d9d9d9;
  color: #666;
  font-size: 14px;
}

.add-taint-btn:hover {
  border-color: #1890ff;
  color: #1890ff;
}

.taints-config {
  background-color: #fafafa;
  border-radius: 6px;
  padding: 16px;
  border: 1px solid #f0f0f0;
}

.taint-config-item {
  background-color: white;
  border-radius: 4px;
  padding: 12px;
  margin-bottom: 12px;
  border: 1px solid #e8e8e8;
}

.taint-config-item:last-child {
  margin-bottom: 0;
}

/* 响应式优化 */
@media (max-width: 768px) {
  .template-dialog {
    --el-dialog-width: 95vw !important;
    --el-dialog-margin-top: 5vh !important;
  }
  
  .delete-col {
    justify-content: flex-start;
    padding-top: 0;
    margin-top: 8px;
  }
  
  .taint-row .el-col {
    margin-bottom: 8px;
  }
}

@media (max-width: 576px) {
  .template-dialog {
    --el-dialog-margin-top: 2vh !important;
  }
  
  .taints-config {
    padding: 12px;
  }
  
  .taint-config-item {
    padding: 8px;
  }
}
</style>