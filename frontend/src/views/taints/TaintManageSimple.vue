<template>
  <div class="taint-manage-simple">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">污点管理</h1>
        <p class="page-description">管理Kubernetes节点污点</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="24" class="stats-row">
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ taintStats.total }}</div>
            <div class="stat-label">总模板数</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 污点模板列表 -->
    <el-card class="table-card">
      <div v-loading="loading">
        <div v-if="taintTemplates.length === 0 && !loading" class="empty-state">
          <el-empty description="暂无污点模板数据">
            <el-button type="primary" @click="showCreateDialog">创建第一个模板</el-button>
          </el-empty>
        </div>
        
        <div v-else class="taint-grid">
          <div
            v-for="template in taintTemplates"
            :key="template.id"
            class="taint-card"
          >
            <div class="taint-header">
              <div class="taint-info">
                <div class="taint-name">{{ template.name || '未命名模板' }}</div>
                <div class="taint-count">{{ getTaintCount(template) }} 个污点</div>
              </div>
              <div class="taint-actions">
                <el-button size="small" type="primary" @click="viewTemplate(template)">
                  查看
                </el-button>
              </div>
            </div>

            <div class="taint-content">
              <div v-if="hasTaints(template)" class="taint-tags">
                <el-tag
                  v-for="(taintInfo, key) in getTaints(template)"
                  :key="key"
                  size="small"
                  :type="getTaintEffectType(taintInfo.effect)"
                  class="taint-tag"
                >
                  {{ key }}={{ taintInfo.value }}:{{ taintInfo.effect }}
                </el-tag>
              </div>
              <div v-else class="no-taints">
                暂无污点内容
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 创建模板对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="创建污点模板"
      width="600px"
    >
      <p>创建污点模板功能正在开发中...</p>
      <template #footer>
        <el-button @click="createDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 查看模板对话框 -->
    <el-dialog
      v-model="viewDialogVisible"
      :title="selectedTemplate?.name || '污点模板'"
      width="600px"
    >
      <div v-if="selectedTemplate">
        <h4>污点详情：</h4>
        <div class="template-taints">
          <div
            v-for="(taintInfo, key) in getTaints(selectedTemplate)"
            :key="key"
            class="taint-item"
          >
            <strong>{{ key }}</strong>: {{ taintInfo.value }} ({{ taintInfo.effect }})
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="viewDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import taintApi from '@/api/taint'
import { Refresh } from '@element-plus/icons-vue'

// 响应式数据
const loading = ref(false)
const taintTemplates = ref([])
const createDialogVisible = ref(false)
const viewDialogVisible = ref(false)
const selectedTemplate = ref(null)

// 计算属性
const taintStats = computed(() => {
  return {
    total: taintTemplates.value.length
  }
})

// 方法
const refreshData = async () => {
  try {
    loading.value = true
    console.log('正在获取污点模板...')
    
    const response = await taintApi.getTemplateList({
      page: 1,
      page_size: 100
    })
    
    console.log('API响应:', response)
    
    if (response && response.data) {
      // 处理多种可能的数据格式
      if (Array.isArray(response.data)) {
        // 数据直接是数组
        taintTemplates.value = response.data
      } else if (response.data.templates && Array.isArray(response.data.templates)) {
        // 数据在 templates 字段中（与标签API格式一致）
        taintTemplates.value = response.data.templates
      } else if (response.data.data && Array.isArray(response.data.data)) {
        // 数据在 data.data 中
        taintTemplates.value = response.data.data
      } else {
        console.warn('未识别的数据格式:', response.data)
        taintTemplates.value = []
      }
    } else {
      taintTemplates.value = []
    }
    
    console.log('解析后的模板数据:', taintTemplates.value)
    ElMessage.success('数据刷新成功')
  } catch (error) {
    console.error('获取污点模板失败:', error)
    ElMessage.error('获取污点模板失败: ' + (error.message || '未知错误'))
    taintTemplates.value = []
  } finally {
    loading.value = false
  }
}

const getTaintCount = (template) => {
  if (!template) return 0
  if (template.taints && typeof template.taints === 'object') {
    return Object.keys(template.taints).length
  }
  return 0
}

const hasTaints = (template) => {
  return getTaintCount(template) > 0
}

const getTaints = (template) => {
  if (template && template.taints && typeof template.taints === 'object') {
    return template.taints
  }
  return {}
}

const getTaintEffectType = (effect) => {
  switch (effect) {
    case 'NoSchedule':
      return 'danger'
    case 'PreferNoSchedule':
      return 'warning'
    case 'NoExecute':
      return 'info'
    default:
      return ''
  }
}

const showCreateDialog = () => {
  createDialogVisible.value = true
}

const viewTemplate = (template) => {
  selectedTemplate.value = template
  viewDialogVisible.value = true
}

// 组件挂载
onMounted(() => {
  refreshData()
})
</script>

<style scoped>
.taint-manage-simple {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
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

.stats-row {
  margin-bottom: 24px;
}

.stat-card {
  text-align: center;
}

.stat-content {
  padding: 20px;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: #1890ff;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #666;
}

.table-card {
  margin-bottom: 24px;
}

.empty-state {
  padding: 40px;
  text-align: center;
}

.taint-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
  padding: 20px;
}

.taint-card {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 16px;
  background: #fff;
  transition: all 0.2s;
}

.taint-card:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.1);
}

.taint-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.taint-name {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.taint-count {
  font-size: 12px;
  color: #999;
}

.taint-content {
  margin-bottom: 12px;
}

.taint-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.taint-tag {
  font-family: monospace;
  font-size: 12px;
}

.no-taints {
  color: #999;
  font-style: italic;
  text-align: center;
  padding: 12px 0;
}

.template-taints {
  max-height: 300px;
  overflow-y: auto;
}

.taint-item {
  padding: 8px;
  margin-bottom: 8px;
  background: #f5f5f5;
  border-radius: 4px;
  font-family: monospace;
}

.taint-item strong {
  color: #1890ff;
}
</style>
