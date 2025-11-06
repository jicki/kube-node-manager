<template>
  <div class="cluster-wrapper">
    <!-- 集群头部 -->
    <div class="cluster-header">
      <div class="cluster-title">
        <h3>{{ clusterName }}</h3>
        <cluster-status-badge
          :health="clusterHealth"
          :show-retry-button="true"
          @retry="handleRetry"
        />
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="cluster-content">
      <!-- 加载中 -->
      <div v-if="loading" class="loading-state">
        <el-icon class="is-loading">
          <Loading />
        </el-icon>
        <p>加载中...</p>
      </div>

      <!-- 错误状态 -->
      <div v-else-if="error" class="error-state">
        <el-result
          icon="error"
          :title="`集群 ${clusterName} 连接失败`"
          :sub-title="errorMessage"
        >
          <template #extra>
            <el-button type="primary" @click="handleRetry">
              <i class="el-icon-refresh"></i>
              重试
            </el-button>
            <el-button
              v-if="isAdmin"
              type="warning"
              @click="handleResetCircuitBreaker"
            >
              <i class="el-icon-setting"></i>
              重置断路器
            </el-button>
          </template>
        </el-result>

        <!-- 错误详情（可折叠） -->
        <el-collapse v-if="error && errorDetails" class="error-details">
          <el-collapse-item title="错误详情" name="1">
            <pre>{{ errorDetails }}</pre>
          </el-collapse-item>
        </el-collapse>
      </div>

      <!-- 成功状态 - 渲染slot内容 -->
      <div v-else-if="success" class="success-state">
        <slot :data="data"></slot>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <el-empty description="暂无数据"></el-empty>
      </div>
    </div>
  </div>
</template>

<script>
import { Loading } from '@element-plus/icons-vue'
import ClusterStatusBadge from './ClusterStatusBadge.vue'
import clusterApi from '@/api/cluster'
import { ElMessage } from 'element-plus'

export default {
  name: 'ClusterWrapper',
  components: {
    ClusterStatusBadge,
    Loading
  },
  props: {
    clusterName: {
      type: String,
      required: true
    },
    loading: {
      type: Boolean,
      default: false
    },
    error: {
      type: Boolean,
      default: false
    },
    errorMessage: {
      type: String,
      default: '集群连接超时或不可达'
    },
    errorDetails: {
      type: String,
      default: ''
    },
    success: {
      type: Boolean,
      default: false
    },
    data: {
      type: [Object, Array],
      default: null
    }
  },
  data() {
    return {
      clusterHealth: null,
      healthCheckTimer: null
    }
  },
  computed: {
    isAdmin() {
      // 检查当前用户是否是管理员
      return this.$store.getters.role === 'admin'
    }
  },
  mounted() {
    // 初始加载健康状态
    this.loadClusterHealth()

    // 定期刷新健康状态（30秒）
    this.healthCheckTimer = setInterval(() => {
      this.loadClusterHealth()
    }, 30000)
  },
  beforeUnmount() {
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer)
    }
  },
  methods: {
    async loadClusterHealth() {
      try {
        const res = await clusterApi.getClusterHealth(this.clusterName)
        this.clusterHealth = res.data
      } catch (err) {
        console.warn(`Failed to load health for cluster ${this.clusterName}:`, err)
      }
    },

    handleRetry() {
      this.$emit('retry')
    },

    async handleResetCircuitBreaker() {
      try {
        await this.$confirm(
          '确定要重置断路器吗？这将允许系统重新尝试连接该集群。',
          '确认重置',
          {
            type: 'warning'
          }
        )

        await clusterApi.resetCircuitBreaker(this.clusterName)
        
        ElMessage.success('断路器已重置')
        
        // 刷新健康状态
        await this.loadClusterHealth()
        
        // 触发重试
        this.$emit('retry')
      } catch (err) {
        if (err !== 'cancel') {
          ElMessage.error('重置断路器失败: ' + (err.message || err))
        }
      }
    }
  }
}
</script>

<style scoped lang="scss">
.cluster-wrapper {
  margin-bottom: 24px;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  background-color: #fff;

  .cluster-header {
    padding: 16px 20px;
    border-bottom: 1px solid #ebeef5;
    background-color: #f5f7fa;

    .cluster-title {
      display: flex;
      align-items: center;
      justify-content: space-between;

      h3 {
        margin: 0;
        font-size: 16px;
        font-weight: 500;
      }
    }
  }

  .cluster-content {
    padding: 20px;
    min-height: 200px;

    .loading-state,
    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 200px;
      color: #909399;

      i {
        font-size: 48px;
        margin-bottom: 16px;
      }

      p {
        margin: 0;
        font-size: 14px;
      }
    }

    .error-state {
      .error-details {
        margin-top: 16px;

        pre {
          margin: 0;
          padding: 12px;
          background-color: #f5f7fa;
          border-radius: 4px;
          font-size: 12px;
          overflow-x: auto;
        }
      }
    }
  }
}
</style>

