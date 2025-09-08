<template>
  <div class="label-manage">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">标签管理</h1>
        <p class="page-description">管理Kubernetes节点标签，支持批量操作</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="showAddDialog">
          <el-icon><Plus /></el-icon>
          添加标签
        </el-button>
        <el-button @click="refreshData">
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
            <div class="stat-label">总标签数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.active }}</div>
            <div class="stat-label">模板数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.single }}</div>
            <div class="stat-label">单个标签</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <div class="stat-value">{{ labelStats.multiple }}</div>
            <div class="stat-label">多个标签组</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <SearchBox
        v-model="searchKeyword"
        placeholder="搜索标签键值..."
        :advanced-search="true"
        :filters="searchFilters"
        :realtime="true"
        @search="handleSearch"
        @clear="handleSearchClear"
      />
    </el-card>

    <!-- 标签模板列表 -->
    <div class="label-grid">
      <div
        v-for="template in filteredLabels"
        :key="template.id"
        class="label-card"
      >
        <div class="label-header">
          <div class="label-key-value">
            <div class="label-key">{{ template.name }}</div>
            <div class="label-value">{{ Object.keys(template.labels || {}).length }} 个标签</div>
          </div>
          <div class="label-actions">
            <el-dropdown @command="(cmd) => handleLabelAction(cmd, template)">
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

        <div class="label-meta">
          <el-tag type="primary" size="small">
            标签模板
          </el-tag>
          <span class="create-time">{{ template.created_at }}</span>
        </div>

        <div v-if="template.description" class="label-description">
          {{ template.description }}
        </div>

        <div class="label-content">
          <div class="labels-title">包含标签:</div>
          <div class="labels-list">
            <div
              v-for="(value, key) in template.labels || {}"
              :key="key"
              class="label-item-wrapper"
            >
              <el-tag
                size="small"
                class="label-item-tag"
              >
                <div class="label-tag-content">
                  <span class="label-key">{{ key }}</span>
                  <span v-if="typeof value === 'string' && value.includes('|MULTI_VALUE|')" class="label-values">
                    =[{{ value.split('|MULTI_VALUE|').filter(v => v !== '').join('|') || '空' }}]
                  </span>
                  <span v-else-if="Array.isArray(value) && value.length > 1" class="label-values">
                    =[{{ value.filter(v => v !== '').join('|') || '空' }}]
                  </span>
                  <span v-else-if="Array.isArray(value) ? value[0] : value" class="label-value">
                    ={{ Array.isArray(value) ? value[0] : value }}
                  </span>
                </div>
              </el-tag>
            </div>
          </div>
        </div>

        <div class="label-footer">
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
      <div v-if="filteredLabels.length === 0" class="empty-state">
        <el-empty description="暂无标签数据" :image-size="80">
          <el-button type="primary" @click="showAddDialog">
            <el-icon><Plus /></el-icon>
            添加第一个标签
          </el-button>
        </el-empty>
      </div>
    </div>

    <!-- 分页 -->
    <div class="pagination-container">
      <el-pagination
        v-model:current-page="pagination.current"
        v-model:page-size="pagination.size"
        :page-sizes="[12, 24, 48, 96]"
        :total="pagination.total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <!-- 添加/编辑标签对话框 -->
    <el-dialog
      v-model="labelDialogVisible"
      :title="isEditing ? '编辑模板' : '创建模板'"
      width="1000px"
      class="template-dialog"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
              <el-form
        ref="labelFormRef"
        :model="labelForm"
        :rules="labelRules"
        label-width="110px"
        style="margin-top: 20px;"
      >
        <el-form-item label="模板名称" prop="name">
          <el-input
            v-model="labelForm.name"
            placeholder="输入模板名称，如：Web应用标签、生产环境标签"
          />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="labelForm.description"
            type="textarea"
            :rows="2"
            placeholder="模板用途描述"
          />
        </el-form-item>

        <el-divider content-position="left">标签配置</el-divider>

        <div class="labels-config">
          <div 
            v-for="(label, index) in labelForm.labels" 
            :key="index"
            class="label-config-item"
          >
            <el-row :gutter="12" align="middle" class="label-row">
              <el-col :xs="24" :sm="11">
                <el-form-item 
                  :prop="`labels.${index}.key`" 
                  :rules="[{ required: true, message: '请输入标签键', trigger: 'blur' }]"
                  style="margin-bottom: 12px;"
                >
                  <el-input
                    v-model="label.key"
                    placeholder="标签键，如：app、version、environment"
                    size="large"
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="11">
                <el-form-item style="margin-bottom: 12px;">
                  <div class="value-input-group">
                    <el-tag
                      v-for="(value, valueIndex) in label.values"
                      :key="`label-value-${valueIndex}`"
                      closable
                      size="small"
                      class="value-tag"
                      @close="removeLabelValue(index, valueIndex)"
                      :disable-transitions="false"
                    >
                      {{ value || '(空值)' }}
                    </el-tag>
                    <el-input
                      v-if="label.inputVisible"
                      ref="labelValueInputRef"
                      v-model="label.inputValue"
                      size="small"
                      class="value-input"
                      @keyup.enter="confirmLabelValue(index)"
                      @blur="confirmLabelValue(index)"
                      placeholder="输入值"
                    />
                    <el-button 
                      v-else 
                      size="small" 
                      @click="showLabelValueInput(index)"
                      class="add-value-btn"
                    >
                      + 添加值
                    </el-button>
                  </div>
                  <div class="value-help-text">
                    可添加多个值，应用时可选择使用哪个值
                  </div>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="2" class="delete-col">
                <el-button
                  type="danger"
                  size="large"
                  :icon="Delete"
                  circle
                  @click="removeLabel(index)"
                  :disabled="labelForm.labels.length === 1"
                />
              </el-col>
            </el-row>
          </div>
          
          <el-button
            type="dashed"
            block
            size="large"
            @click="addLabel"
            :icon="Plus"
            class="add-label-btn"
          >
            添加标签
          </el-button>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="labelDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="saving"
          @click="handleSaveLabel"
        >
          {{ isEditing ? '更新' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 节点列表对话框 -->
    <el-dialog
      v-model="nodesDialogVisible"
      :title="`标签 ${selectedLabel?.key} 关联的节点`"
      width="800px"
    >
      <el-table :data="selectedLabelNodes" style="width: 100%">
        <el-table-column prop="name" label="节点名称" />
        <el-table-column prop="status" label="状态">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Ready' ? 'success' : 'danger'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="roles" label="角色">
          <template #default="{ row }">
            <el-tag
              v-for="role in row.roles"
              :key="role"
              :type="role === 'master' ? 'danger' : 'primary'"
              size="small"
            >
              {{ role }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="{ row }">
            <el-button
              type="text"
              size="small"
              @click="removeLabelFromNode(row)"
            >
              移除标签
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 应用模板对话框 -->
    <el-dialog
      v-model="applyDialogVisible"
      :title="`应用模板: ${selectedTemplate?.name}`"
      width="1200px"
      class="apply-dialog"
    >
      <div class="template-info">
        <h4>模板包含的标签:</h4>
        <div class="template-labels-config">
          <div 
            v-for="(value, key, index) in selectedTemplate?.labels || {}" 
            :key="`${key}-${index}`"
            class="apply-label-item"
          >
            <div class="label-info">
              <el-tag
                class="label-tag"
                type="primary"
                size="small"
              >
                {{ key }}
              </el-tag>
            </div>
            
            <div v-if="getValueArray(value).length > 1" class="value-selector">
              <el-form-item :label="`选择 ${key} 的值:`" style="margin-bottom: 12px;">
                <el-select 
                  v-model="selectedTemplate.selectedValues[key]" 
                  placeholder="选择要应用的值"
                  style="width: 200px;"
                  size="small"
                >
                  <el-option 
                    v-for="(val, valueIndex) in getValueArray(value)"
                    :key="valueIndex"
                    :label="val || '(空值)'"
                    :value="val"
                  />
                </el-select>
              </el-form-item>
            </div>
            
            <div v-else class="single-value">
              <span class="value-text">
                值: {{ getSingleValue(value) || '(空值)' }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <el-divider />

      <el-form label-width="100px">
        <el-form-item label="选择节点" required>
          <NodeSelector
            v-model="selectedNodes"
            :nodes="availableNodes"
            :loading="loading"
            :show-labels="true"
            :max-label-display="2"
          />
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
import { ref, computed, onMounted, reactive, nextTick } from 'vue'
import labelApi from '@/api/label'
import nodeApi from '@/api/node'
import { useClusterStore } from '@/store/modules/cluster'
import { useNodeStore } from '@/store/modules/node'
import SearchBox from '@/components/common/SearchBox.vue'
import NodeSelector from '@/components/common/NodeSelector.vue'
import {
  Plus,
  Refresh,
  MoreFilled,
  Edit,
  Delete,
  CopyDocument,
  Minus,
  Collection,
  Monitor,
  Select,
  Check
} from '@element-plus/icons-vue'

// 响应式数据
const loading = ref(false)
const saving = ref(false)
const applying = ref(false)
const searchKeyword = ref('')
const labelDialogVisible = ref(false)
const nodesDialogVisible = ref(false)
const applyDialogVisible = ref(false)
const isEditing = ref(false)
const labelFormRef = ref()

// 数据
const labels = ref([])
const availableNodes = ref([])
const selectedLabel = ref(null)
const selectedLabelNodes = ref([])
const selectedTemplate = ref(null)
const selectedNodes = ref([])

// 分页
const pagination = reactive({
  current: 1,
  size: 24,
  total: 0
})

// 表单数据
const labelForm = reactive({
  name: '',
  description: '',
  labels: [{ key: '', values: [''], selectedValue: '', inputVisible: false, inputValue: '' }]
})

// 搜索筛选
const searchFilters = ref([
  {
    key: 'type',
    label: '类型',
    type: 'select',
    placeholder: '选择标签类型',
    options: [
      { label: '全部', value: '' },
      { label: '单个标签', value: 'single' },
      { label: '多个标签组', value: 'multiple' }
    ]
  }
])

// 表单验证规则
const labelRules = {
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' }
  ]
}

// 计算属性
const labelStats = computed(() => {
  // 统计模板数量和标签数量
  const totalTemplates = labels.value.length
  let totalLabels = 0
  let singleLabelTemplates = 0  // 单个标签模板数
  let multipleLabelTemplates = 0  // 多个标签组模板数
  
  labels.value.forEach(template => {
    if (template.labels && typeof template.labels === 'object') {
      const labelCount = Object.keys(template.labels).length
      totalLabels += labelCount
      
      // 根据标签数量分类
      if (labelCount === 1) {
        singleLabelTemplates++
      } else if (labelCount > 1) {
        multipleLabelTemplates++
      }
    }
  })
  
  // 返回新的分类统计
  return { 
    total: totalLabels, // 总标签数
    active: totalTemplates, // 模板数
    single: singleLabelTemplates, // 单个标签模板数
    multiple: multipleLabelTemplates // 多个标签组模板数
  }
})

// 应用高级搜索筛选的最终结果
const filteredAndSortedLabels = ref([])
const isSearchActive = ref(false)

// 计算应用筛选和排序后的结果
const applyFiltersAndSort = (keyword, filters) => {
  isSearchActive.value = !!(keyword || filters.type)
  let result = [...labels.value]

  // 文本搜索
  if (keyword) {
    const query = keyword.toLowerCase()
    result = result.filter(template => {
      // 搜索模板名称
      if (template.name && template.name.toLowerCase().includes(query)) {
        return true
      }
      
      // 搜索描述
      if (template.description && template.description.toLowerCase().includes(query)) {
        return true
      }
      
      // 搜索标签Key
      if (template.labels && typeof template.labels === 'object') {
        return Object.keys(template.labels).some(key => 
          key.toLowerCase().includes(query)
        )
      }
      
      return false
    })
  }

  // 按类型筛选
  if (filters.type) {
    if (filters.type === 'single') {
      // 筛选只有一个标签的模板
      result = result.filter(template => {
        if (template.labels && typeof template.labels === 'object') {
          return Object.keys(template.labels).length === 1
        }
        return false
      })
    } else if (filters.type === 'multiple') {
      // 筛选有多个标签的模板（标签组）
      result = result.filter(template => {
        if (template.labels && typeof template.labels === 'object') {
          return Object.keys(template.labels).length > 1
        }
        return false
      })
    }
  }

  filteredAndSortedLabels.value = result
}

const filteredLabels = computed(() => {
  // 如果有搜索条件，使用筛选后的结果，否则使用原始数据
  const result = isSearchActive.value ? filteredAndSortedLabels.value : labels.value
  
  return result.slice(
    (pagination.current - 1) * pagination.size,
    pagination.current * pagination.size
  )
})

// 方法
const fetchLabels = async () => {
  try {
    loading.value = true
    const response = await labelApi.getTemplateList({
      page: pagination.current,
      page_size: pagination.size
    })
    if (response.data && response.data.code === 200) {
      const data = response.data.data
      // 清理从后端获取的模板数据
      const cleanedTemplates = (data.templates || []).map(template => {
        if (template.labels) {
          const cleanedLabels = {}
          Object.entries(template.labels).forEach(([key, value]) => {
            if (typeof value === 'string' && value.includes('|MULTI_VALUE|')) {
              // 分离并清理多值
              const values = value.split('|MULTI_VALUE|').filter(v => v.trim() !== '')
              const cleanedValues = values.map(v => {
                if (!isValidLabelValue(v)) {
                  console.warn(`从后端获取的标签值不符合格式，将被清理: ${v}`)
                  return sanitizeLabelValue(v)
                }
                return v
              }).filter(v => v !== '')
              
              // 对于多值，存储为数组；单值直接存储
              cleanedLabels[key] = cleanedValues.length > 1 
                ? cleanedValues
                : (cleanedValues[0] || '')
            } else {
              // 验证单值
              if (!isValidLabelValue(value)) {
                console.warn(`从后端获取的标签值不符合格式，将被清理: ${value}`)
                cleanedLabels[key] = sanitizeLabelValue(value)
              } else {
                cleanedLabels[key] = value
              }
            }
          })
          template.labels = cleanedLabels
        }
        return template
      })
      
      labels.value = cleanedTemplates
      pagination.total = data.total || 0
      // 初始化筛选结果
      if (!isSearchActive.value) {
        filteredAndSortedLabels.value = labels.value
      }
    } else {
      labels.value = []
      pagination.total = 0
      filteredAndSortedLabels.value = []
      isSearchActive.value = false
    }
  } catch (error) {
    console.warn('获取标签模板失败:', error)
    labels.value = []
    pagination.total = 0
    filteredAndSortedLabels.value = []
    isSearchActive.value = false
  } finally {
    loading.value = false
  }
}

const fetchNodes = async (forceRefresh = false) => {
  try {
    const clusterStore = useClusterStore()
    const nodeStore = useNodeStore()
    const clusterName = clusterStore.currentClusterName
    
    console.log('fetchNodes 开始执行')
    console.log('当前集群名称:', clusterName)
    console.log('forceRefresh:', forceRefresh)
    console.log('nodeStore.nodes 长度:', nodeStore.nodes.length)
    console.log('nodeStore.currentClusterName:', nodeStore.currentClusterName)
    
    // 如果没有集群，直接设置为空数组
    if (!clusterName) {
      console.log('没有集群名称，设置空数组')
      availableNodes.value = []
      return
    }
    
    // 优化：如果nodeStore中已有当前集群的节点数据且不强制刷新，直接使用
    if (!forceRefresh && nodeStore.nodes.length > 0 && nodeStore.currentClusterName === clusterName) {
      console.log('使用缓存的节点数据，避免重复请求')
      availableNodes.value = nodeStore.nodes
      console.log('缓存数据设置完成，availableNodes.value 长度:', availableNodes.value.length)
      return
    }
    
    console.log('发起API请求获取节点数据')
    // 显示加载状态
    loading.value = true
    
    const response = await nodeApi.getNodes({
      cluster_name: clusterName
    })
    console.log('API响应:', response)
    
    // 后端返回格式: { code, message, data: [...] } - data直接是节点数组
    const nodes = response.data.data || []
    console.log('解析的节点数据:', nodes)
    console.log('节点数量:', nodes.length)
    
    availableNodes.value = nodes
    
    // 更新nodeStore缓存
    if (nodes.length > 0) {
      nodeStore.setNodes(nodes)
      nodeStore.currentClusterName = clusterName
      console.log('已更新nodeStore缓存')
    } else {
      console.warn('获取到的节点数据为空')
    }
    
  } catch (error) {
    console.error('获取节点数据失败:', error)
    console.error('错误详情:', error.response?.data || error.message)
    availableNodes.value = []
  } finally {
    loading.value = false
    console.log('fetchNodes 执行完成，最终 availableNodes.value 长度:', availableNodes.value.length)
  }
}

const refreshData = (includeNodes = false) => {
  fetchLabels()
  if (includeNodes) {
    fetchNodes(true) // 强制刷新节点数据
  }
}

const handleSearch = (params) => {
  applyFiltersAndSort(params.keyword, params.filters)
}

const handleSearchClear = () => {
  isSearchActive.value = false
  filteredAndSortedLabels.value = labels.value
}

const handleSizeChange = (size) => {
  pagination.size = size
  pagination.current = 1
}

const handleCurrentChange = (current) => {
  pagination.current = current
}

// 显示添加对话框
const showAddDialog = () => {
  isEditing.value = false
  resetLabelForm()
  labelDialogVisible.value = true
}

// 重置表单
const resetLabelForm = () => {
  Object.assign(labelForm, {
    name: '',
    description: '',
    labels: [{ 
      key: '', 
      values: [''], 
      selectedValue: '',
      inputVisible: false,
      inputValue: ''
    }]
  })
}

// 添加标签
const addLabel = () => {
  labelForm.labels.push({ 
    key: '', 
    values: [''], 
    selectedValue: '',
    inputVisible: false,
    inputValue: ''
  })
}

// 移除标签
const removeLabel = (index) => {
  if (labelForm.labels.length > 1) {
    labelForm.labels.splice(index, 1)
  }
}

// 显示标签值输入框
const showLabelValueInput = (labelIndex) => {
  labelForm.labels[labelIndex].inputVisible = true
  labelForm.labels[labelIndex].inputValue = ''
  nextTick(() => {
    // 聚焦到输入框
    const inputRefs = document.querySelectorAll('.value-input input')
    if (inputRefs[labelIndex]) {
      inputRefs[labelIndex].focus()
    }
  })
}

// 确认添加标签值
const confirmLabelValue = (labelIndex) => {
  const label = labelForm.labels[labelIndex]
  const inputValue = label.inputValue?.trim()
  
  if (inputValue && !label.values.includes(inputValue)) {
    // 如果第一个值是空的，替换它
    if (label.values.length === 1 && label.values[0] === '') {
      label.values[0] = inputValue
    } else {
      label.values.push(inputValue)
    }
  }
  
  label.inputVisible = false
  label.inputValue = ''
}

// 移除标签值
const removeLabelValue = (labelIndex, valueIndex) => {
  const label = labelForm.labels[labelIndex]
  if (label.values.length > 1) {
    label.values.splice(valueIndex, 1)
  } else {
    // 保留至少一个空值
    label.values = ['']
  }
}

// 处理标签操作
const handleLabelAction = (command, template) => {
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
  
  // 转换标签对象为数组，支持多值
  const labelsArray = Object.entries(template.labels || {}).map(([key, value]) => {
    // 处理可能的多值格式
    let values = []
    if (Array.isArray(value)) {
      values = value
    } else if (typeof value === 'string') {
      // 检查是否是用分隔符连接的多值
      if (value.includes('|MULTI_VALUE|')) {
        values = value.split('|MULTI_VALUE|').filter(v => v.trim() !== '')
      } else {
        values = [value]
      }
    } else {
      values = ['']
    }
    
    return { 
      key, 
      values,
      selectedValue: '',
      inputVisible: false,
      inputValue: ''
    }
  })
  
  Object.assign(labelForm, {
    name: template.name,
    description: template.description || '',
    labels: labelsArray.length > 0 ? labelsArray : [{ 
      key: '', 
      values: [''], 
      selectedValue: '',
      inputVisible: false,
      inputValue: ''
    }]
  })
  
  // 保存当前编辑的模板ID
  labelForm.id = template.id
  
  labelDialogVisible.value = true
}

// 复制模板
const copyTemplate = (template) => {
  const labelsText = Object.entries(template.labels || {})
    .map(([key, value]) => value ? `${key}=${value}` : key)
    .join(', ')
  const text = `${template.name}: ${labelsText}`
  
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
      await labelApi.deleteTemplate(template.id)
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
const handleSaveLabel = async () => {
  try {
    await labelFormRef.value.validate()
    saving.value = true
    
    // 验证标签配置
    const validLabels = labelForm.labels.filter(label => label.key.trim())
    if (validLabels.length === 0) {
      ElMessage.error('请至少添加一个有效的标签')
      return
    }
    
    // 转换标签数组为对象，后端只支持string类型，所以多值用分隔符连接
    const labelsObj = {}
    validLabels.forEach(label => {
      const key = label.key.trim()
      const cleanValues = label.values.filter(v => v !== undefined && v !== null && v !== '').map(v => v.toString().trim())
      
      // 验证和清理每个值，确保符合Kubernetes格式
      const validCleanValues = cleanValues.map(value => {
        if (!isValidLabelValue(value)) {
          console.warn(`标签值不符合Kubernetes格式，将被清理: ${value}`)
          return sanitizeLabelValue(value)
        }
        return value
      }).filter(v => v !== '') // 移除清理后的空值
      
      // 对于多值，存储为数组；单值直接存储
      if (validCleanValues.length > 1) {
        labelsObj[key] = validCleanValues
      } else {
        labelsObj[key] = validCleanValues[0] || ''
      }
    })
    
    const templateData = {
      name: labelForm.name,
      description: labelForm.description,
      labels: labelsObj
    }
    
    if (isEditing.value) {
      // 更新模板
      await labelApi.updateTemplate(labelForm.id, templateData)
      ElMessage.success('模板更新成功')
    } else {
      // 创建新模板
      await labelApi.createTemplate(templateData)
      ElMessage.success('模板创建成功')
    }
    
    labelDialogVisible.value = false
    refreshData()
    
  } catch (error) {
    ElMessage.error(error.message || '保存模板失败')
  } finally {
    saving.value = false
  }
}

// 显示标签关联的节点
const showLabelNodes = async (label) => {
  selectedLabel.value = label
  try {
    // 这里应该获取真实的节点数据
    selectedLabelNodes.value = (label.nodes || []).map(nodeName => ({
      name: nodeName,
      status: 'Ready', // 模拟数据
      roles: ['worker'] // 模拟数据
    }))
    nodesDialogVisible.value = true
  } catch (error) {
    ElMessage.error('获取节点信息失败')
  }
}

// 获取值数组的辅助方法
const getValueArray = (value) => {
  if (Array.isArray(value)) {
    return value
  } else if (typeof value === 'string' && value.includes('|MULTI_VALUE|')) {
    return value.split('|MULTI_VALUE|').filter(v => v.trim() !== '')
  }
  return [value]
}

// 获取单个值的辅助方法
const getSingleValue = (value) => {
  if (Array.isArray(value)) {
    return value[0]
  } else if (typeof value === 'string' && value.includes('|MULTI_VALUE|')) {
    const values = value.split('|MULTI_VALUE|').filter(v => v.trim() !== '')
    return values[0]
  }
  return value
}

// 验证标签值是否符合Kubernetes格式
const isValidLabelValue = (value) => {
  if (!value || typeof value !== 'string') return true // 空值是合法的
  
  // Kubernetes标签值的正则表达式
  // 必须是空字符串或包含字母数字字符、'-'、'_' 或 '.'，并且必须以字母数字字符开始和结束
  const labelValueRegex = /^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$/
  
  return labelValueRegex.test(value) && value.length <= 63 // Kubernetes限制标签值最大长度为63字符
}

// 清理标签值，移除不合法字符
const sanitizeLabelValue = (value) => {
  if (!value || typeof value !== 'string') return ''
  
  // 移除 |MULTI_VALUE| 分隔符和其他不合法字符
  let cleaned = value.replace(/\|MULTI_VALUE\|/g, '').trim()
  
  // 只保留字母数字字符、'-'、'_' 和 '.'
  cleaned = cleaned.replace(/[^A-Za-z0-9\-_.]/g, '')
  
  // 确保以字母数字字符开始和结束
  cleaned = cleaned.replace(/^[^A-Za-z0-9]+/, '').replace(/[^A-Za-z0-9]+$/, '')
  
  // 限制长度
  if (cleaned.length > 63) {
    cleaned = cleaned.substring(0, 63)
  }
  
  return cleaned
}

// 应用模板到节点
const applyTemplateToNodes = (template) => {
  // 显示节点选择对话框
  showApplyDialog(template)
}

// 显示应用对话框
const showApplyDialog = async (template) => {
  console.log('showApplyDialog 接收到的 template:', template)
  console.log('template.labels:', template.labels)
  
  // 懒加载节点数据
  await fetchNodes()
  
  // 调试节点数据
  console.log('fetchNodes 完成后的 availableNodes:', availableNodes.value)
  console.log('availableNodes 长度:', availableNodes.value.length)
  console.log('前3个节点数据:', availableNodes.value.slice(0, 3))
  
  // 深拷贝模板以避免修改原始数据
  const templateCopy = JSON.parse(JSON.stringify(template))
  
  // 初始化选中的值，并预先清理所有值
  templateCopy.selectedValues = {}
  if (templateCopy.labels) {
    // 预处理模板标签，清理不符合格式的值
    const cleanedLabels = {}
    Object.entries(templateCopy.labels).forEach(([key, value]) => {
      let values = []
      
      if (Array.isArray(value)) {
        values = value
      } else if (typeof value === 'string' && value.includes('|MULTI_VALUE|')) {
        values = value.split('|MULTI_VALUE|').filter(v => v.trim() !== '')
      } else {
        values = [value]
      }
      
      // 清理每个值，确保符合Kubernetes格式
      const cleanedValues = values.map(v => {
        const cleanValue = String(v || '').trim()
        if (!isValidLabelValue(cleanValue)) {
          console.warn(`模板中的标签值不符合格式，将被清理: ${cleanValue}`)
          return sanitizeLabelValue(cleanValue)
        }
        return cleanValue
      }).filter(v => v !== '') // 移除清理后的空值
      
      if (cleanedValues.length > 0) {
        // 对于多值标签，存储数组以便UI选择，但不再使用MULTI_VALUE分隔符
        if (cleanedValues.length > 1) {
          cleanedLabels[key] = cleanedValues  // 存储为数组
          templateCopy.selectedValues[key] = cleanedValues[0] // 默认选择第一个值
        } else {
          cleanedLabels[key] = cleanedValues[0] // 单值直接存储
        }
      }
    })
    
    // 使用清理后的标签
    templateCopy.labels = cleanedLabels
  }
  
  selectedTemplate.value = templateCopy
  applyDialogVisible.value = true
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
    
    // 构造应用数据，使用选中的值，并确保值符合Kubernetes标签格式
    const labelsToApply = {}
    if (selectedTemplate.value.labels) {
      Object.entries(selectedTemplate.value.labels).forEach(([key, value]) => {
        const valueArray = getValueArray(value)
        let finalValue = ''
        
        console.log(`处理标签 ${key}:`, { value, valueArray, selectedValue: selectedTemplate.value.selectedValues[key] })
        
        if (valueArray.length > 1) {
          // 使用选中的值，但要确保它是单一值
          const selectedValue = selectedTemplate.value.selectedValues[key] || valueArray[0]
          console.log(`多值标签 ${key}: selectedValue=${selectedValue}, valueArray=${JSON.stringify(valueArray)}`)
          finalValue = getSingleValue(selectedValue) // 确保选中的值也经过处理
        } else {
          // 使用默认值
          finalValue = getSingleValue(value)
        }
        
        // 强制清理所有可能包含MULTI_VALUE分隔符的值
        if (typeof finalValue === 'string' && finalValue.includes('|MULTI_VALUE|')) {
          console.warn(`发现包含MULTI_VALUE的值: ${finalValue}，将强制清理`)
          const cleanValues = finalValue.split('|MULTI_VALUE|').filter(v => v.trim() !== '')
          finalValue = cleanValues[0] || ''
          console.log(`清理后的值: ${finalValue}`)
        }
        
        console.log(`最终标签值: ${key} = ${finalValue}`)
        
        // 验证最终值符合Kubernetes标签格式
        if (finalValue && isValidLabelValue(finalValue)) {
          labelsToApply[key] = finalValue
        } else if (finalValue) {
          // 如果值不符合格式，记录警告并使用清理后的值
          console.warn(`标签值不符合Kubernetes格式，将被清理: ${finalValue}`)
          const sanitizedValue = sanitizeLabelValue(finalValue)
          if (sanitizedValue && sanitizedValue !== '') {
            labelsToApply[key] = sanitizedValue
          }
        }
      })
    }
    
    const applyData = {
      cluster_name: clusterName,
      node_names: selectedNodes.value,
      template_id: selectedTemplate.value.id,
      operation: 'add',
      labels: labelsToApply // 包含选定的标签值
    }
    
    console.log('即将发送给后端的数据:', applyData)
    console.log('标签数据详情:', JSON.stringify(labelsToApply, null, 2))
    
    await labelApi.applyTemplate(applyData)
    ElMessage.success('模板应用成功')
    applyDialogVisible.value = false
    
  } catch (error) {
    console.error('应用标签模板失败:', error)
    console.error('发送到后端的数据:', applyData)
    ElMessage.error(`应用模板失败: ${error.message}`)
  } finally {
    applying.value = false
  }
}

// 从单个节点移除标签
const removeLabelFromNode = async (node) => {
  try {
    // 获取当前集群名称
    const clusterStore = useClusterStore()
    const clusterName = clusterStore.currentClusterName
    
    if (!clusterName) {
      ElMessage.error('请先选择集群')
      return
    }
    
    // 使用URL参数传递cluster_name
    const response = await labelApi.deleteNodeLabel(node.name, selectedLabel.value.key, { cluster_name: clusterName })
    ElMessage.success(`已从节点 ${node.name} 移除标签`)
    // 刷新数据
    showLabelNodes(selectedLabel.value)
    refreshData()
  } catch (error) {
    ElMessage.error(`移除标签失败: ${error.message}`)
  }
}

// 移除选中的节点
const removeSelectedNode = (nodeName) => {
  selectedNodes.value = selectedNodes.value.filter(name => name !== nodeName)
}

// 重置搜索状态
const resetSearchState = () => {
  searchKeyword.value = ''
  isSearchActive.value = false
  filteredAndSortedLabels.value = []
}

onMounted(() => {
  // 页面进入时重置搜索状态，避免从其他页面切换回来时保留搜索条件
  resetSearchState()
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

:deep(.el-textarea__inner) {
  font-size: 14px;
  padding: 12px 15px;
  line-height: 1.6;
}

:deep(.el-select) {
  font-size: 14px;
}

:deep(.el-select__wrapper) {
  padding: 8px 15px;
  min-height: 44px;
}

.label-manage {
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

.stat-content {
  padding: 16px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1890ff;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #666;
}

.search-card {
  margin-bottom: 24px;
}

.label-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}

.label-card {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 24px;
  background: #fff;
  transition: all 0.3s;
  position: relative;
  min-height: 300px;
  display: flex;
  flex-direction: column;
}

.label-card:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.label-card.system-label {
  border-left: 4px solid #722ed1;
}

.label-card:not(.system-label) {
  border-left: 4px solid #1890ff;
}

.label-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.label-key-value {
  flex: 1;
}

.label-key {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.label-value {
  font-size: 14px;
  color: #666;
  font-family: 'Monaco', 'Menlo', monospace;
}

.label-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 13px;
}

.node-count {
  color: #666;
}

.label-description {
  font-size: 13px;
  color: #666;
  margin-bottom: 16px;
  line-height: 1.5;
}

.label-nodes {
  margin-bottom: 16px;
}

.nodes-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.nodes-title {
  font-size: 13px;
  color: #666;
  font-weight: 500;
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

.label-footer {
  border-top: 1px solid #f0f0f0;
  padding-top: 12px;
  margin-top: auto;
  flex-shrink: 0;
}

.empty-state {
  grid-column: 1 / -1;
  padding: 40px;
  text-align: center;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

.node-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.node-name {
  flex: 1;
  text-align: left;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }
  
  .label-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .stats-row .el-col {
    margin-bottom: 16px;
  }
}

@media (max-width: 480px) {
  .label-card {
    padding: 16px;
  }
  
  .label-header {
    flex-direction: column;
    gap: 8px;
  }
}

/* 新增样式 */
.labels-config {
  margin-top: 16px;
}

.label-config-item {
  margin-bottom: 16px;
}

.create-time {
  color: #999;
  font-size: 12px;
}

.label-content {
  margin-bottom: 16px;
  flex: 1;
  overflow: hidden;
}

.labels-title {
  font-size: 13px;
  color: #666;
  font-weight: 500;
  margin-bottom: 8px;
}

.labels-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 100%;
}

.label-item-wrapper {
  width: 100%;
}

.label-item-tag {
  font-size: 12px;
  min-height: 28px;
  height: auto;
  line-height: 1.4;
  font-family: 'Monaco', 'Menlo', monospace;
  width: 100%;
  max-width: 100%;
  padding: 6px 10px;
  display: block;
  white-space: normal;
  word-break: break-all;
}

.label-tag-content {
  display: block;
  width: 100%;
  line-height: 1.4;
}

.template-info {
  margin-bottom: 16px;
}

.template-info h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #333;
}

.template-labels {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.label-tag {
  font-family: 'Monaco', 'Menlo', monospace;
}

/* 表单优化样式 */
.template-dialog {
  --el-dialog-border-radius: 8px;
  --el-dialog-padding-primary: 0;
}

.template-dialog :deep(.el-dialog) {
  max-height: 90vh;
  display: flex;
  flex-direction: column;
}

.template-dialog :deep(.el-dialog__body) {
  padding: 20px 40px 30px;
  flex: 1;
  overflow-y: auto;
  max-height: calc(90vh - 160px);
}

.template-dialog :deep(.el-dialog__header) {
  padding: 20px 40px 10px;
  border-bottom: 1px solid #f0f0f0;
  flex-shrink: 0;
}

.template-dialog :deep(.el-dialog__footer) {
  padding: 15px 40px 20px;
  border-top: 1px solid #f0f0f0;
  flex-shrink: 0;
}

.label-row {
  margin-bottom: 8px;
}

.delete-col {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding-top: 4px;
}

.add-label-btn {
  margin-top: 12px;
  height: 44px;
  border-style: dashed;
  border-color: #d9d9d9;
  color: #666;
  font-size: 14px;
}

.add-label-btn:hover {
  border-color: #1890ff;
  color: #1890ff;
}

.labels-config {
  background-color: #fafafa;
  border-radius: 6px;
  padding: 16px;
  border: 1px solid #f0f0f0;
}

.label-config-item {
  background-color: white;
  border-radius: 4px;
  padding: 12px;
  margin-bottom: 12px;
  border: 1px solid #e8e8e8;
}

.label-config-item:last-child {
  margin-bottom: 0;
}

/* 响应式优化 */
@media (max-width: 1024px) {
  .template-dialog {
    --el-dialog-width: 95vw !important;
    --el-dialog-margin-top: 3vh !important;
  }
  
  .template-dialog :deep(.el-dialog__body) {
    padding: 15px 25px 20px;
  }
  
  .template-dialog :deep(.el-dialog__header) {
    padding: 15px 25px 10px;
  }
  
  .template-dialog :deep(.el-dialog__footer) {
    padding: 12px 25px 15px;
  }
}

@media (max-width: 768px) {
  .template-dialog {
    --el-dialog-width: 98vw !important;
    --el-dialog-margin-top: 2vh !important;
  }
  
  .template-dialog :deep(.el-dialog__body) {
    padding: 15px 20px 20px;
    max-height: calc(90vh - 140px);
  }
  
  .template-dialog :deep(.el-dialog__header) {
    padding: 15px 20px 10px;
  }
  
  .template-dialog :deep(.el-dialog__footer) {
    padding: 12px 20px 15px;
  }
  
  .delete-col {
    justify-content: flex-start;
    padding-top: 0;
    margin-top: 8px;
  }
  
  .label-row .el-col {
    margin-bottom: 8px;
  }
  
  .value-input-group {
    min-height: 36px;
    padding: 6px 10px;
  }
  
  .value-tag {
    height: 24px;
    line-height: 22px;
  }
  
  .add-value-btn {
    height: 24px;
    line-height: 22px;
    padding: 0 8px;
  }
  
  .label-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .label-card {
    padding: 20px;
    min-height: 240px;
  }
}

/* 多值输入组件样式 */
.value-input-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-start;
  min-height: 40px;
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  padding: 8px 12px;
  background: #fff;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.value-input-group:hover {
  border-color: #c0c4cc;
}

.value-input-group:focus-within {
  border-color: #409EFF;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
}

.value-tag {
  margin: 2px 0;
  font-size: 12px;
  height: 26px;
  line-height: 24px;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
}

.value-input {
  flex: 1;
  min-width: 100px;
  margin: 2px 0;
  border: none;
  outline: none;
}

.value-input :deep(.el-input__wrapper) {
  box-shadow: none;
  background: transparent;
  padding: 4px 8px;
}

.add-value-btn {
  height: 26px;
  line-height: 24px;
  font-size: 12px;
  padding: 0 12px;
  margin: 2px 0;
  border-style: dashed;
  border-color: #d9d9d9;
  flex-shrink: 0;
}

.add-value-btn:hover {
  border-color: #409EFF;
  color: #409EFF;
}

.value-help-text {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

/* 应用模板对话框样式 */
.template-labels-config {
  max-height: 300px;
  overflow-y: auto;
}

.apply-label-item {
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 12px;
  background: #fafafa;
}

.label-info {
  margin-bottom: 8px;
}

.value-selector {
  margin-top: 8px;
}

.value-selector .el-form-item__label {
  font-size: 13px;
  color: #606266;
}

.single-value {
  margin-top: 8px;
}

.value-text {
  font-size: 13px;
  color: #909399;
  font-family: 'Monaco', 'Menlo', monospace;
}

/* 标签标签内部样式 */
.label-key {
  font-weight: 600;
}

.label-values {
  color: #E6A23C;
  font-weight: 500;
  word-break: break-all;
  max-width: 100%;
  display: inline-block;
}

.label-value {
  color: #409EFF;
  word-break: break-word;
  max-width: 100%;
  display: inline-block;
}

  /* 应用对话框样式 */
  .apply-dialog {
    max-width: 95vw;
    --el-dialog-padding-primary: 0;
  }
  
  .apply-dialog :deep(.el-dialog__header) {
    padding: 20px 24px 16px;
    border-bottom: 1px solid #f0f0f0;
    margin: 0;
  }
  
  .apply-dialog :deep(.el-dialog__body) {
    padding: 0;
    max-height: 70vh;
    overflow-y: auto;
  }
  
  .apply-dialog :deep(.el-dialog__footer) {
    padding: 16px 24px 20px;
    border-top: 1px solid #f0f0f0;
    margin: 0;
  }
  
  .apply-dialog-content {
    display: flex;
    flex-direction: column;
    gap: 24px;
    padding: 20px 24px;
  }
  
  .template-info-section {
    background: #fafbfc;
    padding: 20px;
    border-radius: 8px;
    border: 1px solid #e8e8e8;
  }
  
  .node-selection-section {
    flex: 1;
  }
  
  .section-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
    font-weight: 600;
    color: #2c3e50;
    margin: 0 0 16px 0;
    padding-bottom: 8px;
    border-bottom: 2px solid #e8f4fd;
  }
  
  .template-labels {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 12px;
  }
  
  .template-label-tag {
    font-family: 'Monaco', 'Menlo', monospace;
    font-size: 13px;
    height: 32px;
    line-height: 30px;
    padding: 0 12px;
    border-radius: 6px;
    background: #f8f9fa;
    border: 1px solid #dee2e6;
    color: #495057;
    transition: all 0.3s;
  }
  
  .template-label-tag:hover {
    background: #e9ecef;
    border-color: #adb5bd;
    transform: translateY(-1px);
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  }
  
  .template-label-tag .label-key {
    font-weight: 600;
    color: #007bff;
  }
  
  .template-label-tag .label-separator {
    margin: 0 4px;
    color: #6c757d;
  }
  
  .template-label-tag .label-value {
    font-weight: 500;
    color: #28a745;
  }
  
  .template-description {
    margin-top: 12px;
    padding: 12px;
    background: white;
    border-radius: 6px;
    border-left: 4px solid #007bff;
  }
  
  .template-description p {
    margin: 0;
    color: #6c757d;
    font-size: 14px;
    line-height: 1.5;
  }
  
  .node-selector-wrapper {
    background: white;
    border-radius: 8px;
    border: 1px solid #e8e8e8;
    padding: 16px;
  }
  
  .selected-summary {
    background: #f0f9ff;
    border: 1px solid #bae6fd;
    border-radius: 8px;
    padding: 16px;
  }
  
  .summary-card {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  
  .summary-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    color: #0284c7;
    font-size: 14px;
  }
  
  .summary-nodes {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  
  .selected-node-tag {
    background: #dbeafe;
    border-color: #93c5fd;
    color: #1e40af;
    font-size: 12px;
  }
  
  .more-nodes-tag {
    background: #f3f4f6;
    border-color: #d1d5db;
    color: #6b7280;
    font-size: 12px;
  }
  
  .dialog-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .footer-info {
    flex: 1;
  }
  
  .selection-count {
    color: #6b7280;
    font-size: 14px;
  }
  
  .selection-count strong {
    color: #1f2937;
  }
  
  .footer-actions {
    display: flex;
    gap: 12px;
  }
  
  /* 响应式设计 */
  @media (max-width: 1024px) {
    .apply-dialog {
      width: 95vw !important;
    }
    
    .apply-dialog-content {
      gap: 20px;
      padding: 16px 20px;
    }
    
    .template-info-section {
      padding: 16px;
    }
    
    .node-selector-wrapper {
      padding: 12px;
    }
  }
  
  @media (max-width: 768px) {
    .apply-dialog {
      width: 98vw !important;
      margin: 10px auto;
    }
    
    .apply-dialog-content {
      gap: 16px;
      padding: 12px 16px;
    }
    
    .section-title {
      font-size: 15px;
    }
    
    .template-label-tag {
      font-size: 12px;
      height: 28px;
      line-height: 26px;
      padding: 0 10px;
    }
    
    .dialog-footer {
      flex-direction: column;
      gap: 12px;
      align-items: stretch;
    }
    
    .footer-actions {
      justify-content: center;
    }
  }
</style>