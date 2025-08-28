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

    <!-- 污点卡片列表 -->
    <div class="taint-grid">
      <div
        v-for="taint in taints"
        :key="`${taint.key}-${taint.effect}`"
        class="taint-card"
        :class="`taint-${taint.effect.toLowerCase().replace(/([a-z])([A-Z])/g, '$1-$2').toLowerCase()}`"
      >
        <div class="taint-header">
          <div class="taint-key-value">
            <div class="taint-key">{{ taint.key }}</div>
            <div class="taint-value">{{ taint.value || '(无值)' }}</div>
          </div>
          <div class="taint-effect">
            <el-tag
              :type="getTaintEffectType(taint.effect)"
              size="small"
            >
              {{ formatTaintEffect(taint.effect) }}
            </el-tag>
          </div>
        </div>

        <div class="taint-nodes">
          <div class="nodes-title">关联节点 ({{ taint.nodeCount || 0 }})</div>
          <div class="nodes-list">
            <el-tag
              v-for="node in (taint.nodes || []).slice(0, 3)"
              :key="node"
              size="small"
              class="node-tag"
            >
              {{ node }}
            </el-tag>
            <span v-if="(taint.nodes || []).length > 3" class="more-nodes">
              +{{ (taint.nodes || []).length - 3 }} 个
            </span>
          </div>
        </div>

        <div class="taint-actions">
          <el-button-group size="small">
            <el-button @click="editTaint(taint)">
              <el-icon><Edit /></el-icon>
              编辑
            </el-button>
            <el-button type="danger" @click="deleteTaint(taint)">
              <el-icon><Delete /></el-icon>
              删除
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
      :title="isEditing ? '编辑污点' : '添加污点'"
      width="600px"
    >
      <el-form
        ref="taintFormRef"
        :model="taintForm"
        :rules="taintRules"
        label-width="100px"
      >
        <el-form-item label="污点键" prop="key">
          <el-input
            v-model="taintForm.key"
            placeholder="例如: node-role, dedicated, special"
          />
        </el-form-item>

        <el-form-item label="污点值" prop="value">
          <el-input
            v-model="taintForm.value"
            placeholder="例如: master, gpu, database（可为空）"
          />
        </el-form-item>

        <el-form-item label="效果" prop="effect">
          <el-select v-model="taintForm.effect" placeholder="选择污点效果">
            <el-option label="NoSchedule - 禁止调度" value="NoSchedule" />
            <el-option label="PreferNoSchedule - 尽量不调度" value="PreferNoSchedule" />
            <el-option label="NoExecute - 禁止执行" value="NoExecute" />
          </el-select>
        </el-form-item>

        <el-form-item label="应用到节点">
          <el-select
            v-model="taintForm.selectedNodes"
            multiple
            filterable
            placeholder="选择要应用此污点的节点"
            style="width: 100%"
          >
            <el-option
              v-for="node in availableNodes"
              :key="node.name"
              :label="`${node.name} (${node.status})`"
              :value="node.name"
            />
          </el-select>
        </el-form-item>

        <el-alert
          title="污点效果说明"
          type="info"
          :closable="false"
          show-icon
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import taintApi from '@/api/taint'
import nodeApi from '@/api/node'
import { formatTaintEffect } from '@/utils/format'
import {
  Plus,
  Refresh,
  Edit,
  Delete
} from '@element-plus/icons-vue'

// 响应式数据
const loading = ref(false)
const saving = ref(false)
const taintDialogVisible = ref(false)
const isEditing = ref(false)
const taintFormRef = ref()

// 数据
const taints = ref([])
const availableNodes = ref([])

// 表单数据
const taintForm = reactive({
  key: '',
  value: '',
  effect: '',
  selectedNodes: []
})

// 表单验证规则
const taintRules = {
  key: [
    { required: true, message: '请输入污点键', trigger: 'blur' }
  ],
  effect: [
    { required: true, message: '请选择污点效果', trigger: 'change' }
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
    const response = await taintApi.getAllTaints()
    taints.value = response.data || []
  } catch (error) {
    ElMessage.error('获取污点数据失败')
  } finally {
    loading.value = false
  }
}

const fetchNodes = async () => {
  try {
    const response = await nodeApi.getNodes()
    availableNodes.value = response.data?.items || response.data || []
  } catch (error) {
    console.error('获取节点数据失败:', error)
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
    key: '',
    value: '',
    effect: '',
    selectedNodes: []
  })
}

// 编辑污点
const editTaint = (taint) => {
  isEditing.value = true
  Object.assign(taintForm, {
    key: taint.key,
    value: taint.value || '',
    effect: taint.effect,
    selectedNodes: taint.nodes || []
  })
  taintDialogVisible.value = true
}

// 删除污点
const deleteTaint = (taint) => {
  ElMessageBox.confirm(
    `确认删除污点 "${taint.key}:${taint.effect}" 吗？此操作将从所有关联节点中移除该污点。`,
    '删除污点',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    try {
      await taintApi.batchDeleteTaints(taint.nodes || [], [taint.key])
      ElMessage.success('污点已删除')
      refreshData()
    } catch (error) {
      ElMessage.error(`删除污点失败: ${error.message}`)
    }
  }).catch(() => {
    // 用户取消
  })
}

// 保存污点
const handleSaveTaint = async () => {
  try {
    await taintFormRef.value.validate()
    saving.value = true
    
    const taintData = {
      key: taintForm.key,
      value: taintForm.value,
      effect: taintForm.effect
    }
    
    if (taintForm.selectedNodes.length > 0) {
      await taintApi.batchAddTaints(taintForm.selectedNodes, [taintData])
    }
    
    ElMessage.success(isEditing.value ? '污点更新成功' : '污点创建成功')
    taintDialogVisible.value = false
    refreshData()
    
  } catch (error) {
    ElMessage.error(error.message || '保存污点失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  refreshData()
})
</script>

<style scoped>
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
</style>