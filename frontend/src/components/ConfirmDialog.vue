<template>
  <el-dialog
    v-model="visible"
    :title="title"
    :width="width"
    @close="handleClose"
  >
    <el-alert
      :title="alertTitle"
      :type="alertType"
      :description="alertDescription"
      show-icon
      :closable="false"
      style="margin-bottom: 20px"
    />

    <div v-if="showDetails" class="details-section">
      <el-descriptions :column="1" border>
        <el-descriptions-item
          v-for="(value, key) in details"
          :key="key"
          :label="key"
        >
          {{ value }}
        </el-descriptions-item>
      </el-descriptions>
    </div>

    <div v-if="requireConfirmation" style="margin-top: 20px">
      <el-checkbox v-model="userConfirmed">
        {{ confirmationText }}
      </el-checkbox>
    </div>

    <template #footer>
      <el-button @click="handleCancel">取消</el-button>
      <el-button
        :type="confirmButtonType"
        @click="handleConfirm"
        :disabled="requireConfirmation && !userConfirmed"
        :loading="confirming"
      >
        {{ confirmButtonText }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default: '操作确认'
  },
  alertTitle: {
    type: String,
    required: true
  },
  alertDescription: {
    type: String,
    default: ''
  },
  alertType: {
    type: String,
    default: 'warning',
    validator: (value) => ['success', 'warning', 'info', 'error'].includes(value)
  },
  details: {
    type: Object,
    default: () => ({})
  },
  requireConfirmation: {
    type: Boolean,
    default: true
  },
  confirmationText: {
    type: String,
    default: '我已充分了解此操作的风险和影响'
  },
  confirmButtonText: {
    type: String,
    default: '确认执行'
  },
  confirmButtonType: {
    type: String,
    default: 'danger'
  },
  width: {
    type: String,
    default: '500px'
  }
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const userConfirmed = ref(false)
const confirming = ref(false)

const showDetails = computed(() => {
  return Object.keys(props.details).length > 0
})

const handleConfirm = () => {
  if (props.requireConfirmation && !userConfirmed.value) {
    return
  }
  emit('confirm')
}

const handleCancel = () => {
  emit('cancel')
  handleClose()
}

const handleClose = () => {
  userConfirmed.value = false
  visible.value = false
}

// 暴露方法供父组件使用
defineExpose({
  setConfirming: (value) => {
    confirming.value = value
  }
})
</script>

<style scoped>
.details-section {
  margin-top: 16px;
}
</style>

