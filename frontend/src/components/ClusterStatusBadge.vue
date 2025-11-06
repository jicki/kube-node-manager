<template>
  <div class="cluster-status-badge">
    <!-- 健康状态 -->
    <el-tag
      v-if="health"
      :type="statusType"
      :effect="effect"
      size="small"
      class="status-tag"
    >
      <i :class="statusIcon"></i>
      <span>{{ statusText }}</span>
    </el-tag>

    <!-- 断路器状态 -->
    <el-tooltip
      v-if="health && health.circuit_open"
      content="断路器已打开，集群连接异常"
      placement="top"
    >
      <el-tag type="danger" size="small" class="circuit-tag">
        <i class="el-icon-warning-outline"></i>
        Circuit Open
      </el-tag>
    </el-tooltip>

    <!-- 失败次数 -->
    <el-tooltip
      v-if="health && health.failure_count > 0 && !health.circuit_open"
      :content="`连续失败 ${health.failure_count} 次`"
      placement="top"
    >
      <el-tag type="warning" size="small" class="failure-tag">
        <i class="el-icon-warning"></i>
        {{ health.failure_count }}
      </el-tag>
    </el-tooltip>

    <!-- 最后检查时间 -->
    <span v-if="health && health.last_check_time" class="last-check-time">
      <i class="el-icon-time"></i>
      {{ formatTime(health.last_check_time) }}
    </span>

    <!-- 重试按钮 -->
    <el-button
      v-if="health && health.circuit_open && showRetryButton"
      type="text"
      size="small"
      class="retry-button"
      @click="$emit('retry')"
    >
      <i class="el-icon-refresh"></i>
      重试
    </el-button>
  </div>
</template>

<script>
export default {
  name: 'ClusterStatusBadge',
  props: {
    health: {
      type: Object,
      default: null
    },
    showRetryButton: {
      type: Boolean,
      default: true
    }
  },
  computed: {
    statusType() {
      if (!this.health) return 'info'
      if (this.health.circuit_open) return 'danger'
      if (!this.health.is_healthy) return 'warning'
      return 'success'
    },
    effect() {
      return this.health && this.health.circuit_open ? 'dark' : 'light'
    },
    statusIcon() {
      if (!this.health) return 'el-icon-question'
      if (this.health.circuit_open) return 'el-icon-circle-close'
      if (!this.health.is_healthy) return 'el-icon-warning'
      return 'el-icon-circle-check'
    },
    statusText() {
      if (!this.health) return '未知'
      if (this.health.circuit_open) return '断路'
      if (!this.health.is_healthy) return '异常'
      return '正常'
    }
  },
  methods: {
    formatTime(timeStr) {
      if (!timeStr) return ''
      const time = new Date(timeStr)
      const now = new Date()
      const diff = now - time
      
      if (diff < 60000) return '刚刚'
      if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
      if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
      return `${Math.floor(diff / 86400000)}天前`
    }
  }
}
</script>

<style scoped lang="scss">
.cluster-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;

  .status-tag {
    i {
      margin-right: 4px;
    }
  }

  .circuit-tag,
  .failure-tag {
    i {
      margin-right: 4px;
    }
  }

  .last-check-time {
    font-size: 12px;
    color: #909399;
    
    i {
      margin-right: 4px;
    }
  }

  .retry-button {
    padding: 0 8px;
    
    i {
      margin-right: 4px;
    }
  }
}
</style>

