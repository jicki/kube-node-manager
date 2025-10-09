<template>
  <div class="page-container">
    <div class="card-container">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2>GitLab Pipelines</h2>
        </div>
        <div class="toolbar-right">
          <el-input
            v-model="filters.projectId"
            placeholder="Project ID"
            style="width: 150px; margin-right: 8px"
            @keyup.enter="fetchPipelines"
          >
            <template #prepend>
              <span>项目</span>
            </template>
          </el-input>

          <el-input
            v-model="filters.ref"
            placeholder="分支/标签"
            clearable
            style="width: 150px; margin-right: 8px"
            @keyup.enter="fetchPipelines"
          />

          <el-select
            v-model="filters.status"
            placeholder="状态"
            clearable
            style="width: 120px; margin-right: 8px"
            @change="fetchPipelines"
          >
            <el-option label="全部" value="" />
            <el-option label="运行中" value="running" />
            <el-option label="待处理" value="pending" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="canceled" />
            <el-option label="已跳过" value="skipped" />
          </el-select>

          <el-button
            type="primary"
            :icon="Search"
            @click="fetchPipelines"
            :loading="loading"
          >
            查询
          </el-button>

          <el-button :icon="Refresh" @click="fetchPipelines" :loading="loading">
            刷新
          </el-button>
        </div>
      </div>

      <el-table
        :data="pipelines"
        v-loading="loading"
        style="width: 100%"
        stripe
      >
        <el-table-column prop="id" label="Pipeline ID" width="120" />

        <el-table-column prop="project_id" label="Project ID" width="100" />

        <el-table-column prop="ref" label="分支/标签" min-width="150">
          <template #default="{ row }">
            <el-tag size="small">{{ row.ref }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="getPipelineStatusColor(row.status)"
              size="small"
            >
              {{ getPipelineStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="sha" label="Commit SHA" width="120">
          <template #default="{ row }">
            <code style="font-size: 12px">{{ row.sha ? row.sha.substring(0, 8) : '-' }}</code>
          </template>
        </el-table-column>

        <el-table-column prop="duration" label="耗时" width="100">
          <template #default="{ row }">
            {{ formatDuration(row.duration) }}
          </template>
        </el-table-column>

        <el-table-column prop="queued_duration" label="排队时间" width="100">
          <template #default="{ row }">
            {{ formatDuration(row.queued_duration) }}
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.web_url"
              link
              type="primary"
              size="small"
              @click="openPipelineUrl(row.web_url)"
            >
              查看详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && pipelines.length === 0" class="empty-state">
        <el-empty description="暂无 Pipelines 数据">
          <el-button type="primary" @click="fetchPipelines">
            查询 Pipelines
          </el-button>
        </el-empty>
      </div>

      <!-- 分页组件 -->
      <div v-if="pipelines.length > 0" class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.currentPage"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          :small="false"
          background
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <!-- 使用提示 -->
    <div v-if="!filters.projectId" class="card-container" style="margin-top: 20px">
      <el-alert
        title="使用提示"
        type="info"
        :closable="false"
      >
        <p>请输入 GitLab Project ID 来查询 Pipelines。</p>
        <p style="margin-top: 8px">
          您可以在 GitLab 项目页面的设置中找到 Project ID。
        </p>
      </el-alert>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
import { useGitlabStore } from '@/store/modules/gitlab'

const gitlabStore = useGitlabStore()

const loading = ref(false)
const pipelines = ref([])

const filters = ref({
  projectId: '',
  ref: '',
  status: ''
})

const pagination = ref({
  currentPage: 1,
  pageSize: 20,
  total: 0
})

// Fetch pipelines
const fetchPipelines = async () => {
  if (!filters.value.projectId) {
    ElMessage.warning('请输入 Project ID')
    return
  }

  loading.value = true
  try {
    const params = {
      project_id: filters.value.projectId,
      page: pagination.value.currentPage,
      per_page: pagination.value.pageSize
    }
    if (filters.value.ref) params.ref = filters.value.ref
    if (filters.value.status) params.status = filters.value.status

    const data = await gitlabStore.fetchPipelines(params)
    pipelines.value = data || []
    
    // Note: GitLab API doesn't return total count in basic response
    // We dynamically calculate total to allow unlimited pagination
    if (data && data.length > 0) {
      if (data.length === pagination.value.pageSize) {
        // Current page is full, assume there might be more pages
        // Set total to allow at least one more page
        pagination.value.total = pagination.value.currentPage * pagination.value.pageSize + pagination.value.pageSize
      } else {
        // Current page is not full, this is the last page
        pagination.value.total = (pagination.value.currentPage - 1) * pagination.value.pageSize + data.length
      }
    } else {
      // No data returned
      pagination.value.total = (pagination.value.currentPage - 1) * pagination.value.pageSize
    }
  } catch (error) {
    ElMessage.error(gitlabStore.error || '获取 Pipelines 失败')
    pipelines.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
}

// Handle page size change
const handleSizeChange = () => {
  pagination.value.currentPage = 1
  fetchPipelines()
}

// Handle page change
const handlePageChange = () => {
  fetchPipelines()
}

// Get pipeline status label
const getPipelineStatusLabel = (status) => {
  const labels = {
    running: '运行中',
    pending: '待处理',
    success: '成功',
    failed: '失败',
    canceled: '已取消',
    skipped: '已跳过',
    manual: '手动',
    created: '已创建'
  }
  return labels[status] || status
}

// Get pipeline status color
const getPipelineStatusColor = (status) => {
  const colors = {
    running: 'primary',
    pending: 'warning',
    success: 'success',
    failed: 'danger',
    canceled: 'info',
    skipped: 'info',
    manual: 'warning',
    created: 'info'
  }
  return colors[status] || ''
}

// Format duration (seconds to readable format)
const formatDuration = (seconds) => {
  // Check for null or undefined (but allow 0, which means < 1 second)
  if (seconds === null || seconds === undefined) return '-'
  
  // Ensure it's a number
  const duration = Number(seconds)
  if (isNaN(duration) || duration < 0) return '-'

  const hours = Math.floor(duration / 3600)
  const minutes = Math.floor((duration % 3600) / 60)
  const secs = duration % 60

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`
  } else if (minutes > 0) {
    return `${minutes}m ${secs}s`
  } else {
    return `${secs}s`
  }
}

// Format time
const formatTime = (time) => {
  if (!time) return '-'
  const date = new Date(time)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// Open pipeline URL in new tab
const openPipelineUrl = (url) => {
  window.open(url, '_blank')
}

onMounted(async () => {
  // Check if GitLab is enabled
  await gitlabStore.fetchSettings()
  if (!gitlabStore.isEnabled) {
    ElMessage.warning('GitLab 集成未启用，请先在设置中配置')
    return
  }
})
</script>

<style scoped>
.empty-state {
  padding: 40px 0;
  text-align: center;
}

.pagination-container {
  padding: 20px 0;
  display: flex;
  justify-content: center;
}
</style>
