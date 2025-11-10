<template>
  <div class="page-container">
    <div class="card-container">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2>GitLab Jobs</h2>
        </div>
        <div class="toolbar-right">
          <el-input
            v-model="filters.tag"
            placeholder="Tag 过滤（支持模糊匹配）"
            clearable
            style="width: 200px; margin-right: 8px"
            @keyup.enter="applyFilters"
          >
            <template #prepend>
              <span>Tag</span>
            </template>
          </el-input>

          <el-select
            v-model="filters.status"
            placeholder="状态"
            clearable
            style="width: 120px; margin-right: 8px"
            @change="fetchJobs"
          >
            <el-option label="全部" value="" />
            <el-option label="创建" value="created" />
            <el-option label="待处理" value="pending" />
            <el-option label="运行中" value="running" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="canceled" />
            <el-option label="已跳过" value="skipped" />
            <el-option label="手动" value="manual" />
          </el-select>

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
        </div>
      </div>

      <el-table
        :data="filteredJobs"
        v-loading="loading"
        style="width: 100%"
        stripe
      >
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag
              :type="getJobStatusColor(row.status)"
              size="small"
            >
              {{ getJobStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="作业" min-width="180">
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

        <el-table-column label="项目" min-width="200">
          <template #default="{ row }">
            <div v-if="row.project">
              <span style="font-size: 12px">
                {{ row.project.name_with_namespace || row.project.name || '-' }}
              </span>
            </div>
            <div v-else>-</div>
          </template>
        </el-table-column>

        <el-table-column label="Runner" min-width="150">
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

        <el-table-column label="流水线" width="120">
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

        <el-table-column prop="stage" label="阶段" width="120" show-overflow-tooltip />

        <el-table-column label="Tag 列表" min-width="150">
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
              <span style="color: #909399">无</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="耗时" width="100" align="right">
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

      <div v-if="!loading && filteredJobs.length === 0" class="empty-state">
        <el-empty description="暂无 Jobs 数据">
          <el-button type="primary" @click="fetchJobs">
            查询 Jobs
          </el-button>
        </el-empty>
      </div>

      <!-- 分页组件 -->
      <div v-if="filteredJobs.length > 0" class="pagination-container">
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
          您可以按状态过滤，或输入 Tag 进行前端过滤。
        </p>
      </el-alert>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
import { listAllJobs } from '@/api/gitlab'
import { useGitlabStore } from '@/store/modules/gitlab'

const gitlabStore = useGitlabStore()

const loading = ref(false)
const jobs = ref([])

const filters = ref({
  status: '',
  tag: ''
})

const pagination = ref({
  currentPage: 1,
  pageSize: 20,
  total: 0
})

// 前端 Tag 过滤
const filteredJobs = computed(() => {
  if (!filters.value.tag) {
    return jobs.value
  }
  
  const tagFilter = filters.value.tag.toLowerCase()
  return jobs.value.filter(job => {
    if (!job.tag_list || job.tag_list.length === 0) {
      return false
    }
    return job.tag_list.some(tag => 
      tag.toLowerCase().includes(tagFilter)
    )
  })
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

    const response = await listAllJobs(params)
    jobs.value = response.data || []
    
    // 动态计算总数以支持分页
    if (jobs.value.length > 0) {
      if (jobs.value.length === pagination.value.pageSize) {
        // 当前页已满，假设可能有更多页
        pagination.value.total = pagination.value.currentPage * pagination.value.pageSize + pagination.value.pageSize
      } else {
        // 当前页未满，这是最后一页
        pagination.value.total = (pagination.value.currentPage - 1) * pagination.value.pageSize + jobs.value.length
      }
    } else {
      // 无数据返回
      pagination.value.total = (pagination.value.currentPage - 1) * pagination.value.pageSize
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '获取 Jobs 失败')
    jobs.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
}

// 应用过滤器（包括 tag 过滤）
const applyFilters = () => {
  // Tag 过滤在前端进行，只需重新获取数据
  pagination.value.currentPage = 1
  fetchJobs()
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
    created: '创建',
    pending: '待处理',
    running: '运行中',
    success: '成功',
    failed: '失败',
    canceled: '已取消',
    skipped: '已跳过',
    manual: '手动'
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
    manual: 'warning'
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

