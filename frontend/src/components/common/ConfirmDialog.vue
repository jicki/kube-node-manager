<template>
  <el-dialog
    v-model="visible"
    :title="title"
    :width="width"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    append-to-body
  >
    <div class="confirm-content">
      <div class="confirm-icon">
        <el-icon v-if="type === 'warning'" class="warning-icon">
          <WarningFilled />
        </el-icon>
        <el-icon v-else-if="type === 'error'" class="error-icon">
          <CircleCloseFilled />
        </el-icon>
        <el-icon v-else-if="type === 'info'" class="info-icon">
          <InfoFilled />
        </el-icon>
        <el-icon v-else class="warning-icon">
          <QuestionFilled />
        </el-icon>
      </div>
      
      <div class="confirm-text">
        <div class="confirm-message">{{ message }}</div>
        <div v-if="description" class="confirm-description">
          {{ description }}
        </div>
        
        <!-- 详细信息 -->
        <div v-if="details && details.length > 0" class="confirm-details">
          <div class="details-title">详细信息：</div>
          <ul class="details-list">
            <li v-for="(detail, index) in details" :key="index">
              {{ detail }}
            </li>
          </ul>
        </div>
        
        <!-- 输入框（用于确认输入） -->
        <div v-if="requireInput" class="confirm-input">
          <el-input
            v-model="inputValue"
            :placeholder="inputPlaceholder"
            size="default"
            @keyup.enter="handleConfirm"
          />
          <div v-if="inputError" class="input-error">
            {{ inputError }}
          </div>
        </div>
      </div>
    </div>
    
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancel">
          {{ cancelText }}
        </el-button>
        <el-button
          :type="confirmButtonType"
          :loading="loading"
          :disabled="confirmDisabled"
          @click="handleConfirm"
        >
          {{ confirmText }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import {
  WarningFilled,
  CircleCloseFilled,
  InfoFilled,
  QuestionFilled
} from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default: '确认操作'
  },
  message: {
    type: String,
    required: true
  },
  description: {
    type: String,
    default: ''
  },
  details: {
    type: Array,
    default: () => []
  },
  type: {
    type: String,
    default: 'warning',
    validator: (value) => ['warning', 'error', 'info', 'question'].includes(value)
  },
  confirmText: {
    type: String,
    default: '确定'
  },
  cancelText: {
    type: String,
    default: '取消'
  },
  width: {
    type: String,
    default: '420px'
  },
  loading: {
    type: Boolean,
    default: false
  },
  // 是否需要输入确认
  requireInput: {
    type: Boolean,
    default: false
  },
  inputPlaceholder: {
    type: String,
    default: '请输入以确认'
  },
  // 输入验证函数
  inputValidator: {
    type: Function,
    default: null
  },
  // 危险操作（红色按钮）
  dangerous: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const visible = ref(false)
const inputValue = ref('')
const inputError = ref('')

// 计算属性
const confirmButtonType = computed(() => {
  return props.dangerous ? 'danger' : 'primary'
})

const confirmDisabled = computed(() => {
  if (props.requireInput) {
    return !inputValue.value || !!inputError.value
  }
  return false
})

// 监听modelValue变化
watch(() => props.modelValue, (newValue) => {
  visible.value = newValue
  if (newValue && props.requireInput) {
    inputValue.value = ''
    inputError.value = ''
  }
})

watch(visible, (newValue) => {
  emit('update:modelValue', newValue)
})

// 监听输入值变化并验证
watch(inputValue, (newValue) => {
  if (props.requireInput && props.inputValidator) {
    const result = props.inputValidator(newValue)
    if (typeof result === 'string') {
      inputError.value = result
    } else {
      inputError.value = result ? '' : '输入不正确'
    }
  } else {
    inputError.value = ''
  }
})

// 处理确认
const handleConfirm = () => {
  if (confirmDisabled.value) return
  
  emit('confirm', {
    inputValue: inputValue.value
  })
  
  if (!props.loading) {
    visible.value = false
  }
}

// 处理取消
const handleCancel = () => {
  emit('cancel')
  visible.value = false
}

// 暴露方法供外部调用
defineExpose({
  show: () => {
    visible.value = true
  },
  hide: () => {
    visible.value = false
  }
})
</script>

<style scoped>
.confirm-content {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.confirm-icon {
  flex-shrink: 0;
  font-size: 24px;
  margin-top: 2px;
}

.warning-icon {
  color: #faad14;
}

.error-icon {
  color: #ff4d4f;
}

.info-icon {
  color: #1677ff;
}

.confirm-text {
  flex: 1;
}

.confirm-message {
  font-size: 16px;
  color: #333;
  margin-bottom: 8px;
  line-height: 1.5;
}

.confirm-description {
  font-size: 14px;
  color: #666;
  margin-bottom: 12px;
  line-height: 1.5;
}

.confirm-details {
  margin: 12px 0;
  padding: 12px;
  background: #f5f5f5;
  border-radius: 4px;
}

.details-title {
  font-size: 14px;
  font-weight: 500;
  color: #333;
  margin-bottom: 8px;
}

.details-list {
  margin: 0;
  padding-left: 16px;
  font-size: 13px;
  color: #666;
  line-height: 1.6;
}

.confirm-input {
  margin-top: 16px;
}

.input-error {
  margin-top: 8px;
  font-size: 12px;
  color: #ff4d4f;
}

.dialog-footer {
  text-align: right;
}

.dialog-footer .el-button + .el-button {
  margin-left: 12px;
}
</style>