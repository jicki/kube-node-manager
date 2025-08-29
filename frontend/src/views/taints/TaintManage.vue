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

    <!-- 搜索和过滤 -->
    <el-card class="search-card">
      <SearchBox
        v-model="searchKeyword"
        placeholder="搜索模板名称、描述或污点Key..."
        :advanced-search="true"
        :filters="searchFilters"
        :realtime="true"
        @search="handleSearch"
        @clear="handleSearchClear"
      />
    </el-card>

    <!-- 污点模板列表 -->
    <div class="taint-grid">
      <div
        v-for="template in filteredAndSortedTaints"
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
            <div
              v-for="taint in template.taints || []"
              :key="`${taint.key}-${taint.effect}`"
              class="taint-item-wrapper"
            >
              <el-tag
                size="small"
                class="taint-item-tag"
                :type="getTaintEffectType(taint.effect)"
              >
                <div class="taint-tag-content">
                  <span class="taint-key">{{ taint.key }}</span>
                  <span v-if="typeof taint.value === 'string' && taint.value.includes('|MULTI_VALUE|')" class="taint-values">
                    =[{{ taint.value.split('|MULTI_VALUE|').filter(v => v !== '').join('|') || '空' }}]
                  </span>
                  <span v-else-if="taint.values && taint.values.length > 1" class="taint-values">
                    =[{{ taint.values.filter(v => v !== '').join('|') || '空' }}]
                  </span>
                  <span v-else-if="taint.value || (taint.values && taint.values[0])" class="taint-value">
                    ={{ taint.value || taint.values[0] }}
                  </span>
                  <span class="taint-effect">:{{ taint.effect }}</span>
                </div>
              </el-tag>
            </div>
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
      <div v-if="filteredAndSortedTaints.length === 0 && !searchKeyword" class="empty-state">
        <el-empty description="暂无污点数据" :image-size="80">
          <el-button type="primary" @click="showAddDialog">
            <el-icon><Plus /></el-icon>
            添加第一个污点
          </el-button>
        </el-empty>
      </div>

      <!-- 搜索无结果状态 -->
      <div v-if="filteredAndSortedTaints.length === 0 && searchKeyword" class="empty-search">
        <el-empty description="没有找到匹配的污点模板" :image-size="60">
          <el-button @click="searchKeyword = ''">清空搜索条件</el-button>
        </el-empty>
      </div>
    </div>

    <!-- 添加/编辑污点对话框 -->
    <el-dialog
      v-model="taintDialogVisible"
      :title="isEditing ? '编辑模板' : '创建模板'"
      width="1100px"
      class="template-dialog"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
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
            <el-row :gutter="16" align="middle" class="taint-row">
              <el-col :xs="24" :sm="24" :md="8" class="taint-key-col">
                <el-form-item 
                  :prop="`taints.${index}.key`" 
                  :rules="[{ required: true, message: '请输入污点键', trigger: 'blur' }]"
                  label="污点键"
                  style="margin-bottom: 16px;"
                >
                  <el-input
                    v-model="taint.key"
                    placeholder="污点键，如：node-role、dedicated"
                    size="large"
                    clearable
                  />
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="24" :md="7" class="taint-value-col">
                <el-form-item 
                  label="污点值" 
                  style="margin-bottom: 16px;"
                >
                  <div class="value-input-group">
                    <el-tag
                      v-for="(value, valueIndex) in taint.values"
                      :key="`value-${valueIndex}`"
                      closable
                      size="small"
                      class="value-tag"
                      @close="removeValue(index, valueIndex)"
                      :disable-transitions="false"
                    >
                      {{ value || '(空值)' }}
                    </el-tag>
                    <el-input
                      v-if="taint.inputVisible"
                      ref="valueInputRef"
                      v-model="taint.inputValue"
                      size="small"
                      class="value-input"
                      @keyup.enter="confirmValue(index)"
                      @blur="confirmValue(index)"
                      placeholder="输入值"
                    />
                    <el-button 
                      v-else 
                      size="small" 
                      @click="showValueInput(index)"
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
              <el-col :xs="24" :sm="16" :md="6" class="taint-effect-col">
                <el-form-item 
                  :prop="`taints.${index}.effect`" 
                  :rules="[{ required: true, message: '请选择效果', trigger: 'change' }]"
                  label="效果"
                  style="margin-bottom: 16px;"
                >
                  <el-select 
                    v-model="taint.effect" 
                    placeholder="选择效果" 
                    style="width: 100%; min-width: 180px;"
                    size="large"
                    clearable
                  >
                    <el-option label="NoSchedule" value="NoSchedule">
                      <div class="effect-option">
                        <span class="effect-name">NoSchedule</span>
                        <span class="effect-desc">禁止调度</span>
                      </div>
                    </el-option>
                    <el-option label="PreferNoSchedule" value="PreferNoSchedule">
                      <div class="effect-option">
                        <span class="effect-name">PreferNoSchedule</span>
                        <span class="effect-desc">尽量不调度</span>
                      </div>
                    </el-option>
                    <el-option label="NoExecute" value="NoExecute">
                      <div class="effect-option">
                        <span class="effect-name">NoExecute</span>
                        <span class="effect-desc">禁止执行</span>
                      </div>
                    </el-option>
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :xs="24" :sm="8" :md="3" class="delete-col">
                <el-form-item style="margin-bottom: 16px;" label=" ">
                  <el-button
                    type="danger"
                    size="large"
                    :icon="Delete"
                    circle
                    @click="removeTaint(index)"
                    :disabled="taintForm.taints.length === 1"
                    :title="taintForm.taints.length === 1 ? '至少需要保留一个污点' : '删除此污点'"
                  />
                </el-form-item>
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
      width="1200px"
      class="apply-dialog"
    >
      <div class="template-info">
        <h4>模板包含的污点:</h4>
        <div class="template-taints-config">
          <div 
            v-for="(taint, index) in selectedTemplate?.taints || []" 
            :key="`${taint.key}-${taint.effect}-${index}`"
            class="apply-taint-item"
          >
            <div class="taint-info">
              <el-tag
                class="taint-tag"
                :type="getTaintEffectType(taint.effect)"
                size="small"
              >
                {{ taint.key }}:{{ taint.effect }}
              </el-tag>
            </div>
            
            <div v-if="getTaintValueArray(taint).length > 1" class="value-selector">
              <el-form-item :label="`选择 ${taint.key} 的值:`" style="margin-bottom: 12px;">
                <el-select 
                  v-model="taint.selectedValue" 
                  placeholder="选择要应用的值"
                  style="width: 200px;"
                  size="small"
                >
                  <el-option 
                    v-for="(value, valueIndex) in getTaintValueArray(taint)"
                    :key="valueIndex"
                    :label="value || '(空值)'"
                    :value="value"
                  />
                </el-select>
              </el-form-item>
            </div>
            
            <div v-else class="single-value">
              <span class="value-text">
                值: {{ getTaintSingleValue(taint) || '(空值)' }}
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
import taintApi from '@/api/taint'
import nodeApi from '@/api/node'
import { useClusterStore } from '@/store/modules/cluster'
import { formatTaintEffect } from '@/utils/format'
import SearchBox from '@/components/common/SearchBox.vue'
import NodeSelector from '@/components/common/NodeSelector.vue'
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

// 搜索和过滤相关
const searchKeyword = ref('')

// 搜索筛选配置
const searchFilters = ref([
  {
    key: 'effect',
    label: '污点效果',
    type: 'select',
    placeholder: '选择污点效果',
    options: [
      { label: '全部效果', value: '' },
      { label: 'NoSchedule', value: 'NoSchedule' },
      { label: 'PreferNoSchedule', value: 'PreferNoSchedule' },
      { label: 'NoExecute', value: 'NoExecute' }
    ]
  },
  {
    key: 'sort',
    label: '排序方式',
    type: 'select',
    placeholder: '选择排序方式',
    options: [
      { label: '按创建时间', value: 'created_at' },
      { label: '按名称', value: 'name' },
      { label: '按污点数量', value: 'taint_count' }
    ]
  }
])

// 表单数据
const taintForm = reactive({
  name: '',
  description: '',
  taints: [{ key: '', values: [''], effect: '', selectedValue: '' }]
})

// 表单验证规则
const taintRules = {
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' }
  ]
}

// 计算属性
const taintStats = computed(() => {
  // 统计所有模板中的污点数量
  let total = 0
  let noSchedule = 0
  let preferNoSchedule = 0
  let noExecute = 0
  
  taints.value.forEach(template => {
    if (template.taints && Array.isArray(template.taints)) {
      template.taints.forEach(taint => {
        total++
        if (taint.effect === 'NoSchedule') {
          noSchedule++
        } else if (taint.effect === 'PreferNoSchedule') {
          preferNoSchedule++
        } else if (taint.effect === 'NoExecute') {
          noExecute++
        }
      })
    }
  })
  
  return { total, noSchedule, preferNoSchedule, noExecute }
})

// 过滤和搜索的计算属性
const filteredTaints = computed(() => {
  let result = [...taints.value]

  // 文本搜索
  if (searchKeyword.value) {
    const query = searchKeyword.value.toLowerCase()
    result = result.filter(template => {
      // 搜索模板名称
      if (template.name && template.name.toLowerCase().includes(query)) {
        return true
      }
      
      // 搜索描述
      if (template.description && template.description.toLowerCase().includes(query)) {
        return true
      }
      
      // 搜索污点Key
      if (template.taints && Array.isArray(template.taints)) {
        return template.taints.some(taint => 
          taint.key && taint.key.toLowerCase().includes(query)
        )
      }
      
      return false
    })
  }

  return result
})

// 应用高级搜索筛选的最终结果
const filteredAndSortedTaints = ref([])

// 计算应用筛选和排序后的结果
const applyFiltersAndSort = (keyword, filters) => {
  let result = [...taints.value]

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
      
      // 搜索污点Key
      if (template.taints && Array.isArray(template.taints)) {
        return template.taints.some(taint => 
          taint.key && taint.key.toLowerCase().includes(query)
        )
      }
      
      return false
    })
  }

  // 按效果筛选
  if (filters.effect) {
    result = result.filter(template => {
      if (!template.taints || !Array.isArray(template.taints)) return false
      return template.taints.some(taint => taint.effect === filters.effect)
    })
  }

  // 排序
  const sortBy = filters.sort || 'created_at'
  result.sort((a, b) => {
    switch (sortBy) {
      case 'name':
        return (a.name || '').localeCompare(b.name || '')
      case 'taint_count':
        const countA = (a.taints || []).length
        const countB = (b.taints || []).length
        return countB - countA
      case 'created_at':
      default:
        return new Date(b.created_at) - new Date(a.created_at)
    }
  })

  filteredAndSortedTaints.value = result
}

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
      // 更新筛选结果
      filteredAndSortedTaints.value = taints.value
    } else {
      taints.value = []
      filteredAndSortedTaints.value = []
    }
  } catch (error) {
    console.warn('获取污点模板失败:', error)
    taints.value = []
    filteredAndSortedTaints.value = []
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

// 搜索处理函数
const handleSearch = (params) => {
  applyFiltersAndSort(params.keyword, params.filters)
}

const handleSearchClear = () => {
  // 清空搜索时恢复原始数据
  filteredAndSortedTaints.value = taints.value
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
    taints: [{ 
      key: '', 
      values: [''], 
      effect: '', 
      selectedValue: '',
      inputVisible: false,
      inputValue: ''
    }]
  })
}

// 添加污点
const addTaint = () => {
  taintForm.taints.push({ 
    key: '', 
    values: [''], 
    effect: '', 
    selectedValue: '',
    inputVisible: false,
    inputValue: ''
  })
}

// 移除污点
const removeTaint = (index) => {
  if (taintForm.taints.length > 1) {
    taintForm.taints.splice(index, 1)
  }
}

// 显示值输入框
const showValueInput = (taintIndex) => {
  taintForm.taints[taintIndex].inputVisible = true
  taintForm.taints[taintIndex].inputValue = ''
  nextTick(() => {
    // 聚焦到输入框
    const inputRefs = document.querySelectorAll('.value-input input')
    if (inputRefs[taintIndex]) {
      inputRefs[taintIndex].focus()
    }
  })
}

// 确认添加值
const confirmValue = (taintIndex) => {
  const taint = taintForm.taints[taintIndex]
  const inputValue = taint.inputValue?.trim()
  
  if (inputValue && !taint.values.includes(inputValue)) {
    // 如果第一个值是空的，替换它
    if (taint.values.length === 1 && taint.values[0] === '') {
      taint.values[0] = inputValue
    } else {
      taint.values.push(inputValue)
    }
  }
  
  taint.inputVisible = false
  taint.inputValue = ''
}

// 移除值
const removeValue = (taintIndex, valueIndex) => {
  const taint = taintForm.taints[taintIndex]
  if (taint.values.length > 1) {
    taint.values.splice(valueIndex, 1)
  } else {
    // 保留至少一个空值
    taint.values = ['']
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
  
  // 转换现有污点数据格式以支持多值
  const convertedTaints = template.taints && template.taints.length > 0 
    ? template.taints.map(taint => {
        let values = []
        if (taint.values && Array.isArray(taint.values)) {
          values = taint.values
        } else if (taint.value && typeof taint.value === 'string') {
          // 检查是否是用分隔符连接的多值
          if (taint.value.includes('|MULTI_VALUE|')) {
            values = taint.value.split('|MULTI_VALUE|').filter(v => v.trim() !== '')
          } else {
            values = [taint.value]
          }
        } else {
          values = ['']
        }
        
        return {
          key: taint.key,
          values,
          effect: taint.effect,
          selectedValue: '',
          inputVisible: false,
          inputValue: ''
        }
      })
    : [{ 
        key: '', 
        values: [''], 
        effect: '', 
        selectedValue: '',
        inputVisible: false,
        inputValue: ''
      }]
  
  Object.assign(taintForm, {
    name: template.name,
    description: template.description || '',
    taints: convertedTaints
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
      taints: validTaints.map(taint => {
        const cleanValues = taint.values.filter(v => v !== undefined && v !== null && v !== '').map(v => v.toString().trim())
        
        return {
          key: taint.key.trim(),
          effect: taint.effect,
          // 如果有多个值，用分隔符连接；否则使用单个值
          value: cleanValues.length > 1 ? cleanValues.join('|MULTI_VALUE|') : (cleanValues[0] || '')
        }
      })
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

// 获取污点值数组的辅助方法
const getTaintValueArray = (taint) => {
  if (taint.values && Array.isArray(taint.values)) {
    return taint.values
  } else if (taint.value && typeof taint.value === 'string' && taint.value.includes('|MULTI_VALUE|')) {
    return taint.value.split('|MULTI_VALUE|').filter(v => v.trim() !== '')
  }
  return [taint.value || '']
}

// 获取污点单个值的辅助方法
const getTaintSingleValue = (taint) => {
  const values = getTaintValueArray(taint)
  return values[0] || ''
}

// 应用模板到节点
const applyTemplateToNodes = (template) => {
  // 深拷贝模板以避免修改原始数据
  const templateCopy = JSON.parse(JSON.stringify(template))
  
  // 初始化选中的值
  if (templateCopy.taints) {
    templateCopy.taints.forEach(taint => {
      const valueArray = getTaintValueArray(taint)
      if (valueArray.length > 1) {
        // 默认选择第一个非空值
        taint.selectedValue = valueArray.find(v => v && v.trim()) || valueArray[0] || ''
      }
    })
  }
  
  selectedTemplate.value = templateCopy
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
    
    // 构造应用数据，使用选中的值
    const taintsToApply = selectedTemplate.value.taints.map(taint => {
      const valueArray = getTaintValueArray(taint)
      const valueToUse = valueArray.length > 1 
        ? taint.selectedValue 
        : getTaintSingleValue(taint)
      
      return {
        key: taint.key,
        value: valueToUse,
        effect: taint.effect
      }
    })
    
    const applyData = {
      cluster_name: clusterName,
      node_names: selectedNodes.value,
      template_id: selectedTemplate.value.id,
      operation: 'add',
      taints: taintsToApply // 包含选定的污点值
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
  // 初始化筛选结果
  filteredAndSortedTaints.value = taints.value
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

.search-card {
  margin-bottom: 24px;
}

.empty-search {
  text-align: center;
  padding: 40px 20px;
}

.apply-dialog {
  max-width: 90vw;
}

.apply-dialog .el-dialog__body {
  padding: 20px 24px;
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
  grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}

.taint-card {
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 24px;
  background: #fff;
  transition: all 0.3s;
  min-height: 300px;
  display: flex;
  flex-direction: column;
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
  margin-top: auto;
  flex-shrink: 0;
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
  flex: 1;
  overflow: hidden;
}

.taints-title {
  font-size: 13px;
  color: #666;
  font-weight: 500;
  margin-bottom: 8px;
}

.taints-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 100%;
}

.taint-item-wrapper {
  width: 100%;
}

.taint-item-tag {
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

.taint-tag-content {
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

/* 污点效果选项样式 */
.effect-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.effect-name {
  font-weight: 600;
  color: #333;
}

.effect-desc {
  font-size: 12px;
  color: #666;
  font-weight: normal;
}

/* 改进的列样式 */
.taint-key-col .el-form-item__label,
.taint-value-col .el-form-item__label,
.taint-effect-col .el-form-item__label {
  font-size: 13px;
  font-weight: 600;
  color: #555;
}

/* 确保效果列有足够空间 */
.taint-effect-col {
  min-width: 180px;
}

.taint-effect-col .el-select {
  min-width: 180px;
}

.taint-config-item {
  background-color: white;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 16px;
  border: 1px solid #e8e8e8;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.taint-config-item:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.1);
}

/* 响应式优化 */
@media (max-width: 1200px) {
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
  
  .taint-config-item {
    padding: 16px;
  }
  
  .delete-col {
    text-align: center;
  }
  
  .delete-col .el-form-item__label {
    display: none;
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
  
  .taint-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .taint-card {
    padding: 20px;
    min-height: 240px;
  }
}

@media (min-width: 769px) and (max-width: 1024px) {
  .taint-key-col {
    flex: 0 0 35%;
    max-width: 35%;
  }
  
  .taint-value-col {
    flex: 0 0 30%;
    max-width: 30%;
  }
  
  .taint-effect-col {
    flex: 0 0 27%;
    max-width: 27%;
    min-width: 180px;
  }
  
  .delete-col {
    flex: 0 0 8%;
    max-width: 8%;
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
.template-taints-config {
  max-height: 300px;
  overflow-y: auto;
}

.apply-taint-item {
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 12px;
  background: #fafafa;
}

.taint-info {
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

/* 污点标签内部样式 */
.taint-key {
  font-weight: 600;
}

.taint-values {
  color: #E6A23C;
  font-weight: 500;
  word-break: break-all;
  max-width: 100%;
  display: inline-block;
}

.taint-value {
  color: #409EFF;
  word-break: break-word;
  max-width: 100%;
  display: inline-block;
}

.taint-effect {
  font-weight: 500;
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