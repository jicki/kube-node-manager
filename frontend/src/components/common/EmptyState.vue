<template>
  <div class="empty-state">
    <el-icon :size="iconSize" class="empty-icon" :color="iconColor">
      <component :is="icon" />
    </el-icon>
    <p class="empty-title">{{ title }}</p>
    <p class="empty-description">{{ description }}</p>
    <el-button 
      v-if="action" 
      :type="action.type || 'primary'" 
      @click="action.handler"
    >
      {{ action.text }}
    </el-button>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { 
  CircleCheck, 
  Warning, 
  InfoFilled, 
  Search 
} from '@element-plus/icons-vue'

const props = defineProps({
  // 空状态类型：empty（无数据）, error（错误）, success（成功）, noResults（无结果）
  type: {
    type: String,
    default: 'empty',
    validator: (value) => ['empty', 'error', 'success', 'noResults'].includes(value)
  },
  // 标题
  title: {
    type: String,
    default: ''
  },
  // 描述
  description: {
    type: String,
    default: ''
  },
  // 图标（可以传入自定义图标组件）
  customIcon: {
    type: Object,
    default: null
  },
  // 操作按钮
  action: {
    type: Object,
    default: null
    // 格式：{ text: '按钮文字', handler: () => {}, type: 'primary' }
  },
  // 图标大小
  iconSize: {
    type: Number,
    default: 64
  }
})

// 根据类型选择默认图标和颜色
const icon = computed(() => {
  if (props.customIcon) {
    return props.customIcon
  }

  const iconMap = {
    empty: CircleCheck,
    error: Warning,
    success: CircleCheck,
    noResults: Search
  }
  return iconMap[props.type] || CircleCheck
})

const iconColor = computed(() => {
  const colorMap = {
    empty: '#909399',
    error: '#F56C6C',
    success: '#67C23A',
    noResults: '#909399'
  }
  return colorMap[props.type] || '#909399'
})
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.empty-icon {
  margin-bottom: 20px;
  opacity: 0.6;
}

.empty-title {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
  margin: 0 0 12px;
  line-height: 1.5;
}

.empty-description {
  font-size: 14px;
  color: #909399;
  margin: 0 0 24px;
  line-height: 1.6;
  max-width: 400px;
}

.empty-state .el-button {
  margin-top: 4px;
}
</style>

