<template>
  <el-dialog
    v-model="visible"
    :title="config.title || '确认操作'"
    :width="config.width || '500px'"
    :close-on-click-modal="false"
    :close-on-press-escape="config.cancelable !== false"
    destroy-on-close
    class="enhanced-confirm-dialog"
    :class="config.type"
  >
    <div class="confirm-content">
      <!-- 图标区域 -->
      <div class="icon-section" v-if="!config.hideIcon">
        <div class="icon-container" :class="config.type">
          <el-icon :size="config.iconSize || 48">
            <component :is="getIcon()" />
          </el-icon>
        </div>
      </div>

      <!-- 内容区域 -->
      <div class="content-section">
        <!-- 主标题 -->
        <h3 v-if="config.title" class="confirm-title">{{ config.title }}</h3>
        
        <!-- 描述文本 -->
        <div class="confirm-message">
          <p v-if="typeof config.message === 'string'" v-html="config.message"></p>
          <div v-else-if="config.message">
            <component :is="config.message" v-bind="config.messageProps" />
          </div>
        </div>

        <!-- 详细信息 -->
        <div v-if="config.details" class="confirm-details">
          <el-collapse v-if="config.details.collapsible" v-model="detailsCollapsed">
            <el-collapse-item name="details" :title="config.details.title || '详细信息'">
              <div v-html="config.details.content"></div>
            </el-collapse-item>
          </el-collapse>
          <div v-else class="details-content" v-html="config.details.content"></div>
        </div>

        <!-- 输入区域 -->
        <div v-if="config.input" class="input-section">
          <el-form :model="inputData" :rules="inputRules" ref="inputFormRef">
            <el-form-item 
              v-for="(input, key) in config.input" 
              :key="key"
              :label="input.label"
              :prop="key"
              :required="input.required"
            >
              <el-input
                v-if="input.type === 'text' || !input.type"
                v-model="inputData[key]"
                :placeholder="input.placeholder"
                :maxlength="input.maxlength"
                :show-word-limit="input.showWordLimit"
                :clearable="input.clearable !== false"
                :disabled="confirming"
              />
              <el-input
                v-else-if="input.type === 'textarea'"
                v-model="inputData[key]"
                type="textarea"
                :rows="input.rows || 3"
                :placeholder="input.placeholder"
                :maxlength="input.maxlength"
                :show-word-limit="input.showWordLimit"
                :disabled="confirming"
              />
              <el-select
                v-else-if="input.type === 'select'"
                v-model="inputData[key]"
                :placeholder="input.placeholder"
                :clearable="input.clearable !== false"
                :disabled="confirming"
              >
                <el-option
                  v-for="option in input.options"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </el-form-item>
          </el-form>
        </div>

        <!-- 风险提示 -->
        <div v-if="config.risk" class="risk-warning">
          <el-alert
            :title="config.risk.title || '风险提示'"
            :description="config.risk.message"
            :type="config.risk.level || 'warning'"
            :closable="false"
            show-icon
          />
        </div>

        <!-- 影响范围 -->
        <div v-if="config.impact" class="impact-info">
          <div class="impact-title">
            <el-icon><InfoFilled /></el-icon>
            {{ config.impact.title || '影响范围' }}
          </div>
          <ul class="impact-list">
            <li v-for="(item, index) in config.impact.items" :key="index">
              {{ item }}
            </li>
          </ul>
        </div>

        <!-- 确认项目 -->
        <div v-if="config.checklist" class="checklist-section">
          <div class="checklist-title">请确认以下项目：</div>
          <el-checkbox-group v-model="checkedItems">
            <div v-for="(item, index) in config.checklist" :key="index" class="checklist-item">
              <el-checkbox :label="index" :disabled="confirming">
                {{ item }}
              </el-checkbox>
            </div>
          </el-checkbox-group>
        </div>

        <!-- 二次确认 -->
        <div v-if="config.doubleConfirm" class="double-confirm">
          <div class="double-confirm-text">
            请输入 <strong>{{ config.doubleConfirm.text }}</strong> 来确认操作：
          </div>
          <el-input
            v-model="doubleConfirmInput"
            :placeholder="`请输入 ${config.doubleConfirm.text}`"
            :disabled="confirming"
            class="double-confirm-input"
          />
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <!-- 左侧信息 -->
        <div v-if="config.footerInfo" class="footer-info">
          {{ config.footerInfo }}
        </div>
        
        <!-- 右侧按钮 -->
        <div class="footer-buttons">
          <el-button
            v-if="config.cancelable !== false"
            @click="handleCancel"
            :disabled="confirming"
          >
            {{ config.cancelText || '取消' }}
          </el-button>
          
          <el-button
            :type="getConfirmButtonType()"
            @click="handleConfirm"
            :loading="confirming"
            :disabled="!canConfirm"
          >
            <el-icon v-if="!confirming"><component :is="getConfirmIcon()" /></el-icon>
            {{ config.confirmText || '确认' }}
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, reactive, nextTick } from 'vue'
import {
  QuestionFilled,
  WarningFilled,
  InfoFilled,
  CircleCheckFilled,
  CircleCloseFilled,
  Check,
  Close,
  Warning,
  Delete
} from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  config: {
    type: Object,
    required: true,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

// 响应式数据
const visible = ref(false)
const confirming = ref(false)
const inputData = reactive({})
const inputFormRef = ref(null)
const checkedItems = ref([])
const doubleConfirmInput = ref('')
const detailsCollapsed = ref(['details'])

// 监听 props 变化
watch(() => props.modelValue, (value) => {
  visible.value = value
  if (value) {
    resetForm()
  }
})

watch(visible, (value) => {
  emit('update:modelValue', value)
})

// 计算属性
const inputRules = computed(() => {
  const rules = {}
  if (props.config.input) {
    Object.keys(props.config.input).forEach(key => {
      const input = props.config.input[key]
      if (input.required) {
        rules[key] = [
          { 
            required: true, 
            message: input.errorMessage || `请输入${input.label}`,
            trigger: 'blur'
          }
        ]
      }
      if (input.validator) {
        rules[key] = rules[key] || []
        rules[key].push({ validator: input.validator, trigger: 'blur' })
      }
    })
  }
  return rules
})

const canConfirm = computed(() => {
  // 检查必填项
  if (props.config.input) {
    for (const [key, input] of Object.entries(props.config.input)) {
      if (input.required && !inputData[key]) {
        return false
      }
    }
  }
  
  // 检查确认列表
  if (props.config.checklist) {
    if (checkedItems.value.length < props.config.checklist.length) {
      return false
    }
  }
  
  // 检查二次确认
  if (props.config.doubleConfirm) {
    if (doubleConfirmInput.value !== props.config.doubleConfirm.text) {
      return false
    }
  }
  
  return true
})

// 方法
const getIcon = () => {
  const icons = {
    'warning': WarningFilled,
    'error': CircleCloseFilled,
    'success': CircleCheckFilled,
    'info': InfoFilled,
    'question': QuestionFilled
  }
  return icons[props.config.type] || QuestionFilled
}

const getConfirmButtonType = () => {
  const types = {
    'warning': 'warning',
    'error': 'danger',
    'success': 'success',
    'info': 'primary',
    'question': 'primary'
  }
  return types[props.config.type] || 'primary'
}

const getConfirmIcon = () => {
  if (props.config.type === 'error' || props.config.type === 'warning') {
    return props.config.type === 'error' ? Delete : Warning
  }
  return Check
}

const resetForm = () => {
  // 重置输入数据
  Object.keys(inputData).forEach(key => {
    delete inputData[key]
  })
  
  if (props.config.input) {
    Object.keys(props.config.input).forEach(key => {
      inputData[key] = props.config.input[key].default || ''
    })
  }
  
  // 重置确认列表
  checkedItems.value = []
  
  // 重置二次确认
  doubleConfirmInput.value = ''
  
  // 重置表单验证
  nextTick(() => {
    if (inputFormRef.value) {
      inputFormRef.value.resetFields()
    }
  })
}

const handleConfirm = async () => {
  // 验证输入
  if (props.config.input && inputFormRef.value) {
    try {
      await inputFormRef.value.validate()
    } catch (error) {
      return
    }
  }
  
  confirming.value = true
  
  try {
    const result = {
      input: { ...inputData },
      checklist: [...checkedItems.value],
      doubleConfirm: doubleConfirmInput.value
    }
    
    await emit('confirm', result)
    visible.value = false
  } catch (error) {
    console.error('确认操作失败:', error)
  } finally {
    confirming.value = false
  }
}

const handleCancel = () => {
  emit('cancel')
  visible.value = false
}

// 暴露方法
defineExpose({
  resetForm,
  validate: async () => {
    if (inputFormRef.value) {
      return await inputFormRef.value.validate()
    }
    return true
  }
})
</script>

<style scoped>
.enhanced-confirm-dialog {
  --dialog-border-radius: 12px;
}

.enhanced-confirm-dialog :deep(.el-dialog) {
  border-radius: var(--dialog-border-radius);
  overflow: hidden;
}

.enhanced-confirm-dialog :deep(.el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.enhanced-confirm-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.enhanced-confirm-dialog :deep(.el-dialog__footer) {
  padding: 16px 24px 20px;
  border-top: 1px solid #f0f0f0;
  background: #fafafa;
}

.confirm-content {
  display: flex;
  gap: 20px;
  padding: 24px;
  min-height: 120px;
}

.icon-section {
  flex-shrink: 0;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 8px;
}

.icon-container {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.icon-container.warning {
  background: rgba(255, 193, 7, 0.1);
  color: #ffc107;
}

.icon-container.error {
  background: rgba(220, 53, 69, 0.1);
  color: #dc3545;
}

.icon-container.success {
  background: rgba(40, 167, 69, 0.1);
  color: #28a745;
}

.icon-container.info {
  background: rgba(23, 162, 184, 0.1);
  color: #17a2b8;
}

.icon-container.question {
  background: rgba(108, 117, 125, 0.1);
  color: #6c757d;
}

.content-section {
  flex: 1;
  min-width: 0;
}

.confirm-title {
  margin: 0 0 16px 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
  line-height: 1.4;
}

.confirm-message {
  margin-bottom: 20px;
  line-height: 1.6;
}

.confirm-message p {
  margin: 0;
  color: #666;
  font-size: 15px;
}

.confirm-details {
  margin-bottom: 20px;
}

.details-content {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
  border-left: 3px solid #dee2e6;
  font-size: 14px;
  color: #666;
  line-height: 1.5;
}

.input-section {
  margin-bottom: 20px;
}

.risk-warning {
  margin-bottom: 20px;
}

.impact-info {
  margin-bottom: 20px;
  padding: 16px;
  background: #e3f2fd;
  border-radius: 6px;
  border-left: 3px solid #2196f3;
}

.impact-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #1976d2;
  margin-bottom: 12px;
  font-size: 14px;
}

.impact-list {
  margin: 0;
  padding-left: 20px;
  color: #333;
}

.impact-list li {
  margin-bottom: 4px;
  font-size: 14px;
  line-height: 1.5;
}

.checklist-section {
  margin-bottom: 20px;
  padding: 16px;
  background: #fff3cd;
  border-radius: 6px;
  border-left: 3px solid #ffc107;
}

.checklist-title {
  font-weight: 600;
  color: #856404;
  margin-bottom: 12px;
  font-size: 14px;
}

.checklist-item {
  margin-bottom: 8px;
}

.checklist-item:last-child {
  margin-bottom: 0;
}

.double-confirm {
  margin-bottom: 20px;
  padding: 16px;
  background: #f8d7da;
  border-radius: 6px;
  border-left: 3px solid #dc3545;
}

.double-confirm-text {
  margin-bottom: 12px;
  color: #721c24;
  font-size: 14px;
  font-weight: 500;
}

.double-confirm-input {
  font-family: monospace;
}

.dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-info {
  font-size: 12px;
  color: #999;
}

.footer-buttons {
  display: flex;
  gap: 12px;
}

/* 不同类型的主题色 */
.enhanced-confirm-dialog.warning :deep(.el-dialog__header) {
  border-bottom-color: rgba(255, 193, 7, 0.2);
}

.enhanced-confirm-dialog.error :deep(.el-dialog__header) {
  border-bottom-color: rgba(220, 53, 69, 0.2);
}

.enhanced-confirm-dialog.success :deep(.el-dialog__header) {
  border-bottom-color: rgba(40, 167, 69, 0.2);
}

.enhanced-confirm-dialog.info :deep(.el-dialog__header) {
  border-bottom-color: rgba(23, 162, 184, 0.2);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .confirm-content {
    flex-direction: column;
    padding: 16px;
    gap: 16px;
  }
  
  .icon-section {
    align-self: center;
    padding-top: 0;
  }
  
  .icon-container {
    width: 48px;
    height: 48px;
  }
  
  .icon-container .el-icon {
    font-size: 32px !important;
  }
  
  .confirm-title {
    font-size: 16px;
    text-align: center;
  }
  
  .dialog-footer {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
  
  .footer-buttons {
    justify-content: center;
  }
}
</style>
