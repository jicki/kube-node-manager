<template>
  <div class="page-container">
    <div class="card-container">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2>GitLab Jobs</h2>
        </div>
        <div class="toolbar-right">
          <el-select
            v-model="filters.status"
            placeholder="状态"
            clearable
            style="width: 160px; margin-right: 8px"
            @change="applyFilters"
          >
            <el-option label="全部" value="" />
            <el-option label="已创建" value="created" />
            <el-option label="等待中" value="pending" />
            <el-option label="正在运行" value="running" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="canceled" />
            <el-option label="已跳过" value="skipped" />
            <el-option label="手动" value="manual" />
            <el-option label="已计划" value="scheduled" />
            <el-option label="等待资源" value="waiting_for_resource" />
            <el-option label="正在准备" value="preparing" />
          </el-select>

          <el-input
            v-model="filters.tag"
            placeholder="支持模糊筛选"
            clearable
            style="width: 240px; margin-right: 8px"
            @keyup.enter="applyFilters"
          >
            <template #prepend>
              <span>标签</span>
            </template>
          </el-input>

          <el-button
            type="primary"
            :icon="Search"
            @click="applyFilters"
            :loading="loading"
          >
            查询
          </el-button>

          <el-button :icon="Refresh" @click="fetchJobs" :loading="loading">
            刷新
          </el-button>

          <!-- 数量显示 -->
          <div v-if="getCountDisplay()" style="margin-left: 16px; color: #606266; font-size: 14px; white-space: nowrap">
            {{ getCountDisplay() }}
          </div>
        </div>
      </div>

      <el-table
        :data="jobs"
        v-loading="loading"
        style="width: 100%"
        stripe
      >
        <el-table-column label="状态" align="center">
          <template #default="{ row }">
            <el-tag
              :type="getJobStatusColor(row.status)"
              size="small"
            >
              {{ getJobStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="作业" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">
            <div>
              <el-link
                :href="row.web_url"
                target="_blank"
                type="primary"
                style="font-weight: 600"
              >
                #{{ row.id }}: {{ row.name }}
              </el-link>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Runner" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <div v-if="row.runner">
              <el-tag size="small" type="info">
                {{ row.runner.description || row.runner.name || `#${row.runner.id}` }}
              </el-tag>
            </div>
            <div v-else>
              <span style="color: #909399">无</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="流水线" width="120" align="center">
          <template #default="{ row }">
            <div v-if="row.pipeline">
              <el-link
                v-if="row.pipeline.web_url"
                :href="row.pipeline.web_url"
                target="_blank"
                type="primary"
                size="small"
              >
                #{{ row.pipeline.id }}
              </el-link>
              <span v-else>#{{ row.pipeline.id }}</span>
            </div>
            <div v-else>-</div>
          </template>
        </el-table-column>

        <el-table-column label="阶段" width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.stage }}
          </template>
        </el-table-column>

        <el-table-column label="创建人" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <div v-if="row.user && row.user.name">
              <span style="font-size: 13px">{{ row.user.name }}</span>
              <span v-if="row.user.username" style="font-size: 12px; color: #909399; margin-left: 4px">
                @{{ row.user.username }}
              </span>
            </div>
            <div v-else style="color: #909399">-</div>
          </template>
        </el-table-column>

        <el-table-column label="标签" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <div v-if="row.tag_list && row.tag_list.length > 0">
              <el-tag
                v-for="tag in row.tag_list"
                :key="tag"
                size="small"
                style="margin-right: 4px; margin-bottom: 4px"
              >
                {{ tag }}
              </el-tag>
            </div>
            <div v-else>
              <span style="color: #909399">-</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="耗时" width="90" align="right">
          <template #default="{ row }">
            {{ formatDuration(row.duration) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              link
              type="primary"
              size="small"
              @click="openJobUrl(row.web_url)"
            >
              查看日志
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && jobs.length === 0" class="empty-state">
        <el-empty :description="getEmptyDescription()">
          <el-button type="primary" @click="fetchJobs" v-if="!filters.tag && !filters.status">
            查询 Jobs
          </el-button>
          <div v-else>
            <el-button type="primary" @click="clearFilters">
              清除过滤条件
            </el-button>
          </div>
        </el-empty>
      </div>

      <!-- 分页组件 -->
      <div v-if="jobs.length > 0" class="pagination-container">
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
    <div v-if="jobs.length === 0 && !loading" class="card-container" style="margin-top: 20px">
      <el-alert
        title="使用提示"
        type="info"
        :closable="false"
      >
        <p>此页面显示所有可见的 GitLab Jobs。</p>
        <p style="margin-top: 8px">
          您可以按状态和标签进行过滤，标签支持模糊匹配。
        </p>
      </el-alert>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
import { listAllJobs } from '@/api/gitlab'
import { useGitlabStore } from '@/store/modules/gitlab'

const gitlabStore = useGitlabStore()

const loading = ref(false)
const jobs = ref([])
const totalCount = ref(0) // 总数量
const filteredCount = ref(0) // 过滤后的数量

const filters = ref({
  status: '',
  tag: ''
})

const pagination = ref({
  currentPage: 1,
  pageSize: 20,
  total: 0
})

// Fetch jobs
const fetchJobs = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.value.currentPage,
      per_page: pagination.value.pageSize
    }
    if (filters.value.status) {
      params.status = filters.value.status
    }
    if (filters.value.tag) {
      params.tag = filters.value.tag
    }

    const response = await listAllJobs(params)
    
    // 处理新的响应格式
    if (response.data.jobs) {
      jobs.value = response.data.jobs || []
      totalCount.value = response.data.total || 0
      filteredCount.value = response.data.filtered_count || 0
      pagination.value.total = filteredCount.value
    } else {
      // 向后兼容旧格式
      jobs.value = response.data || []
      
      // 动态计算总数以支持分页
      if (jobs.value.length > 0) {
        if (jobs.value.length === pagination.value.pageSize) {
          pagination.value.total = pagination.value.currentPage * pagination.value.pageSize + pagination.value.pageSize
        } else {
          pagination.value.total = (pagination.value.currentPage - 1) * pagination.value.pageSize + jobs.value.length
        }
      } else {
        pagination.value.total = (pagination.value.currentPage - 1) * pagination.value.pageSize
      }
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '获取 Jobs 失败')
    jobs.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
}

// 应用过滤器
const applyFilters = () => {
  pagination.value.currentPage = 1
  fetchJobs()
}

// 清除过滤条件
const clearFilters = () => {
  filters.value.status = ''
  filters.value.tag = ''
  pagination.value.currentPage = 1
  fetchJobs()
}

// 获取空状态描述
const getEmptyDescription = () => {
  if (filters.value.tag && filters.value.status) {
    return `没有找到状态为"${getJobStatusLabel(filters.value.status)}"且标签包含"${filters.value.tag}"的 Jobs`
  } else if (filters.value.tag) {
    return `没有找到标签包含"${filters.value.tag}"的 Jobs（注：只有在 .gitlab-ci.yml 中配置了 tags 的 Job 才可被标签过滤）`
  } else if (filters.value.status) {
    return `没有找到状态为"${getJobStatusLabel(filters.value.status)}"的 Jobs`
  }
  return '暂无 Jobs 数据'
}

// 获取数量显示
const getCountDisplay = () => {
  const hasFilter = filters.value.status || filters.value.tag
  
  if (totalCount.value > 1000) {
    if (hasFilter) {
      return `共 1000+ 条，过滤后 ${filteredCount.value} 条`
    }
    return '共 1000+ 条'
  } else if (totalCount.value > 0) {
    if (hasFilter) {
      return `共 ${totalCount.value} 条，过滤后 ${filteredCount.value} 条`
    }
    return `共 ${totalCount.value} 条`
  }
  return ''
}

// Handle page size change
const handleSizeChange = () => {
  pagination.value.currentPage = 1
  fetchJobs()
}

// Handle page change
const handlePageChange = () => {
  fetchJobs()
}

// Get job status label
const getJobStatusLabel = (status) => {
  const labels = {
    created: '已创建',
    pending: '等待中',
    running: '正在运行',
    success: '成功',
    failed: '失败',
    canceled: '已取消',
    skipped: '已跳过',
    manual: '手动',
    scheduled: '已计划',
    waiting_for_resource: '等待资源',
    preparing: '正在准备'
  }
  return labels[status] || status
}

// Get job status color
const getJobStatusColor = (status) => {
  const colors = {
    created: 'info',
    pending: 'warning',
    running: 'primary',
    success: 'success',
    failed: 'danger',
    canceled: 'info',
    skipped: 'info',
    manual: 'warning',
    scheduled: 'info',
    waiting_for_resource: 'warning',
    preparing: 'info'
  }
  return colors[status] || ''
}

// Format duration (seconds to readable format)
const formatDuration = (seconds) => {
  if (seconds === null || seconds === undefined || seconds === 0) return '-'
  
  const duration = Number(seconds)
  if (isNaN(duration) || duration < 0) return '-'

  // Round for display
  const roundedDuration = Math.round(duration * 100) / 100

  const hours = Math.floor(roundedDuration / 3600)
  const minutes = Math.floor((roundedDuration % 3600) / 60)
  const secs = Math.round(roundedDuration % 60)

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`
  } else if (minutes > 0) {
    return `${minutes}m ${secs}s`
  } else if (roundedDuration >= 1) {
    return `${secs}s`
  } else {
    return `${roundedDuration.toFixed(2)}s`
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

// Open job URL in new tab
const openJobUrl = (url) => {
  if (url) {
    window.open(url, '_blank')
  }
}

onMounted(async () => {
  // Check if GitLab is enabled
  await gitlabStore.fetchSettings()
  if (!gitlabStore.isEnabled) {
    ElMessage.warning('GitLab 集成未启用，请先在设置中配置')
    return
  }
  
  // Fetch jobs on mount
  fetchJobs()
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

