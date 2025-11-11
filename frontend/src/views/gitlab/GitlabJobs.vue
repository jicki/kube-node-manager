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
            style="width: 240px; margin-right: 8px"
            @change="applyFilters"
          >
            <el-option label="全部（活跃状态）" value="">
              <span style="color: #67C23A">✓</span> 全部（活跃状态）
            </el-option>
            <el-option-group label="🟢 活跃状态（后端过滤）">
              <el-option label="已创建" value="created">
                <span style="color: #67C23A">✓</span> 已创建
              </el-option>
              <el-option label="等待中" value="pending">
                <span style="color: #67C23A">✓</span> 等待中
              </el-option>
              <el-option label="正在运行" value="running">
                <span style="color: #67C23A">✓</span> 正在运行
              </el-option>
              <el-option label="正在准备" value="preparing">
                <span style="color: #67C23A">✓</span> 正在准备
              </el-option>
              <el-option label="已计划" value="scheduled">
                <span style="color: #67C23A">✓</span> 已计划
              </el-option>
              <el-option label="等待资源" value="waiting_for_resource">
                <span style="color: #67C23A">✓</span> 等待资源
              </el-option>
              <el-option label="手动触发" value="manual">
                <span style="color: #67C23A">✓</span> 手动触发
              </el-option>
            </el-option-group>
          </el-select>
          
          <!-- 状态说明提示 -->
          <el-tooltip placement="bottom" effect="light">
            <template #content>
              <div style="max-width: 380px; padding: 4px;">
                <p style="margin: 0 0 8px 0; font-weight: 600; color: #303133;">
                  📊 状态筛选说明
                </p>
                
                <div style="margin-bottom: 10px;">
                  <p style="margin: 0 0 4px 0; font-weight: 500; color: #67C23A;">
                    🟢 活跃状态（后端过滤）：
                  </p>
                  <p style="margin: 0 0 0 16px; font-size: 13px; color: #606266;">
                    • 已创建、等待中、正在运行<br/>
                    • 正在准备、已计划、等待资源<br/>
                    • 手动触发
                  </p>
                  <p style="margin: 4px 0 0 16px; font-size: 12px; color: #909399;">
                    响应速度：8-12秒｜查询范围：3-7 天｜数据实时
                  </p>
                </div>
                
                <div style="margin-bottom: 10px;">
                  <p style="margin: 0 0 4px 0; font-weight: 500; color: #909399;">
                    📋 已完成状态（表格筛选）：
                  </p>
                  <p style="margin: 0 0 0 16px; font-size: 13px; color: #606266;">
                    • 成功、失败、已取消、已跳过
                  </p>
                  <p style="margin: 4px 0 0 16px; font-size: 12px; color: #909399;">
                    使用表格"状态"列的筛选按钮进行过滤
                  </p>
                </div>
                
                <div style="padding: 8px; background: #E8F4FD; border-left: 3px solid #409EFF; border-radius: 4px;">
                  <p style="margin: 0 0 4px 0; font-weight: 500; color: #409EFF;">
                    💡 提示：
                  </p>
                  <p style="margin: 0; font-size: 12px; color: #606266;">
                    某些状态（如"正在准备"、"已计划"）可能在当前 GitLab 中较少出现，查询结果为空属正常现象
                  </p>
                </div>
              </div>
            </template>
            <el-icon style="margin-left: 4px; margin-right: 8px; color: #409EFF; cursor: help; font-size: 16px">
              <InfoFilled />
            </el-icon>
          </el-tooltip>

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
        <el-table-column 
          label="状态" 
          align="center"
          :filters="[
            { text: '已创建', value: 'created' },
            { text: '等待中', value: 'pending' },
            { text: '正在运行', value: 'running' },
            { text: '成功', value: 'success' },
            { text: '失败', value: 'failed' },
            { text: '已取消', value: 'canceled' },
            { text: '已跳过', value: 'skipped' },
            { text: '手动', value: 'manual' },
            { text: '已计划', value: 'scheduled' },
            { text: '等待资源', value: 'waiting_for_resource' },
            { text: '正在准备', value: 'preparing' }
          ]"
          :filter-method="filterStatus"
        >
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
        title="📋 使用说明"
        type="info"
        :closable="false"
      >
        <div style="line-height: 1.8;">
          <div style="margin-bottom: 12px;">
            <strong style="color: #67C23A;">🟢 活跃状态（后端过滤）：</strong>
            <div style="padding-left: 20px; margin-top: 4px;">
              • <strong>已创建、等待中、正在运行</strong>
              <br/>
              • <strong>正在准备、已计划、等待资源</strong>
              <br/>
              • <strong>手动触发</strong>
              <br/>
              <br/>
              • 响应速度：<span style="color: #67C23A;">8-12 秒</span>
              <br/>
              • 查询范围：最近 3-7 天
              <br/>
              • 数据量：活跃状态的 jobs（通常 500-1000+ 个）
            </div>
          </div>

          <div style="margin-bottom: 12px;">
            <strong style="color: #909399;">📊 已完成状态（表格筛选）：</strong>
            <div style="padding-left: 20px; margin-top: 4px;">
              • <strong>成功、失败、已取消、已跳过</strong>
              <br/>
              • 使用方法：点击表格"状态"列的 
              <el-icon style="vertical-align: middle; margin: 0 2px;"><Filter /></el-icon> 
              筛选按钮
              <br/>
              • 原因：后端查询耗时 <span style="color: #F56C6C;">16+ 秒</span>，已优化为前端筛选
            </div>
          </div>

          <div style="padding: 12px; background: #E8F4FD; border-left: 3px solid #409EFF; border-radius: 4px;">
            <strong style="color: #409EFF;">💡 温馨提示：</strong>
            <div style="padding-left: 20px; margin-top: 4px; color: #606266;">
              • 某些状态（如<strong>正在准备、已计划</strong>）在 GitLab 中可能较少出现
              <br/>
              • 查询结果为空时属正常现象，不代表功能不可用
              <br/>
              • 建议优先查询"全部"或"已创建"状态查看整体情况
            </div>
          </div>
        </div>
      </el-alert>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search, InfoFilled, Filter } from '@element-plus/icons-vue'
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

// Filter status in table
const filterStatus = (value, row) => {
  return row.status === value
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

