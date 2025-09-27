<template>
  <div class="label-manage-simple">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">标签管理</h1>
        <p class="page-description">管理Kubernetes节点标签</p>
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
            <div class="stat-value">{{ labelStats.total }}</div>
            <div class="stat-label">总模板数</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 标签模板列表 -->
    <el-card class="table-card">
      <div v-loading="loading">
        <div v-if="labelTemplates.length === 0 && !loading" class="empty-state">
          <el-empty description="暂无标签模板数据">
            <el-button type="primary" @click="showCreateDialog">创建第一个模板</el-button>
          </el-empty>
        </div>
        
        <div v-else class="label-grid">
          <div
            v-for="template in labelTemplates"
            :key="template.id"
            class="label-card"
          >
            <div class="label-header">
              <div class="label-info">
                <div class="label-name">{{ template.name || '未命名模板' }}</div>
                <div class="label-count">{{ getLabelCount(template) }} 个标签</div>
              </div>
              <div class="label-actions">
                <el-button size="small" type="primary" @click="viewTemplate(template)">
                  查看
                </el-button>
              </div>
            </div>

            <div class="label-content">
              <div v-if="hasLabels(template)" class="label-tags">
                <el-tag
                  v-for="(value, key) in getLabels(template)"
                  :key="key"
                  size="small"
                  class="label-tag"
                >
                  {{ key }}={{ value }}
                </el-tag>
              </div>
              <div v-else class="no-labels">
                暂无标签内容
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 创建模板对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="创建标签模板"
      width="600px"
    >
      <p>创建标签模板功能正在开发中...</p>
      <template #footer>
        <el-button @click="createDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 查看模板对话框 -->
    <el-dialog
      v-model="viewDialogVisible"
      :title="selectedTemplate?.name || '标签模板'"
      width="600px"
    >
      <div v-if="selectedTemplate">
        <h4>标签详情：</h4>
        <div class="template-labels">
          <div
            v-for="(value, key) in getLabels(selectedTemplate)"
            :key="key"
            class="label-item"
          >
            <strong>{{ key }}</strong>: {{ value }}
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
import labelApi from '@/api/label'
import { Refresh } from '@element-plus/icons-vue'

// 响应式数据
const loading = ref(false)
const labelTemplates = ref([])
const createDialogVisible = ref(false)
const viewDialogVisible = ref(false)
const selectedTemplate = ref(null)

// 计算属性
const labelStats = computed(() => {
  return {
    total: labelTemplates.value.length
  }
})

// 方法
const refreshData = async () => {
  try {
    loading.value = true
    console.log('🚀 开始获取标签模板...')
    
    const response = await labelApi.getTemplateList({
      page: 1,
      page_size: 100
    })
    
    console.log('📡 API完整响应:', JSON.stringify(response, null, 2))
    console.log('📡 response.data类型:', typeof response.data)
    console.log('📡 response.data内容:', response.data)
    
    if (response && response.data) {
      console.log('✅ response.data 存在')
      
      // 处理多种可能的数据格式
      if (Array.isArray(response.data)) {
        console.log('🔍 数据格式: 直接数组')
        labelTemplates.value = response.data
      } else if (response.data.templates && Array.isArray(response.data.templates)) {
        console.log('🔍 数据格式: response.data.templates')
        console.log('📋 templates内容:', response.data.templates)
        labelTemplates.value = response.data.templates
      } else if (response.data.data && Array.isArray(response.data.data)) {
        console.log('🔍 数据格式: response.data.data')
        labelTemplates.value = response.data.data
      } else {
        console.warn('❌ 未识别的数据格式:')
        console.warn('response.data:', response.data)
        console.warn('response.data.templates存在?', !!response.data.templates)
        console.warn('response.data.templates是数组?', Array.isArray(response.data.templates))
        console.warn('response.data.data存在?', !!response.data.data)
        console.warn('response.data.data是数组?', Array.isArray(response.data.data))
        labelTemplates.value = []
      }
    } else {
      console.warn('❌ response.data 不存在')
      console.warn('response:', response)
      labelTemplates.value = []
    }
    
    console.log('🎯 最终模板数据:', labelTemplates.value)
    console.log('🎯 模板数量:', labelTemplates.value.length)
    
    if (labelTemplates.value.length > 0) {
      ElMessage.success(`成功加载 ${labelTemplates.value.length} 个标签模板`)
    } else {
      ElMessage.warning('没有找到标签模板数据')
    }
  } catch (error) {
    console.error('💥 获取标签模板失败:', error)
    console.error('💥 错误详情:', error.response || error.message || error)
    ElMessage.error('获取标签模板失败: ' + (error.message || '未知错误'))
    labelTemplates.value = []
  } finally {
    loading.value = false
  }
}

const getLabelCount = (template) => {
  if (!template) return 0
  if (template.labels && typeof template.labels === 'object') {
    return Object.keys(template.labels).length
  }
  return 0
}

const hasLabels = (template) => {
  return getLabelCount(template) > 0
}

const getLabels = (template) => {
  if (template && template.labels && typeof template.labels === 'object') {
    return template.labels
  }
  return {}
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
.label-manage-simple {
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

.label-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
  padding: 20px;
}

.label-card {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 16px;
  background: #fff;
  transition: all 0.2s;
}

.label-card:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.1);
}

.label-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.label-name {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.label-count {
  font-size: 12px;
  color: #999;
}

.label-content {
  margin-bottom: 12px;
}

.label-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.label-tag {
  font-family: monospace;
  font-size: 12px;
}

.no-labels {
  color: #999;
  font-style: italic;
  text-align: center;
  padding: 12px 0;
}

.template-labels {
  max-height: 300px;
  overflow-y: auto;
}

.label-item {
  padding: 8px;
  margin-bottom: 8px;
  background: #f5f5f5;
  border-radius: 4px;
  font-family: monospace;
}

.label-item strong {
  color: #1890ff;
}
</style>
