<template>
  <div class="page-container">
    <div class="card-container">
      <div class="toolbar">
        <div class="toolbar-left">
          <h2>GitLab Runners</h2>
        </div>
        <div class="toolbar-right">
          <el-select
            v-model="filters.type"
            placeholder="Runner 类型"
            clearable
            style="width: 150px; margin-right: 8px"
            @change="fetchRunners"
          >
            <el-option label="全部" value="" />
            <el-option label="Instance" value="instance_type" />
            <el-option label="Group" value="group_type" />
            <el-option label="Project" value="project_type" />
          </el-select>

          <el-select
            v-model="filters.status"
            placeholder="状态"
            clearable
            style="width: 120px; margin-right: 8px"
            @change="fetchRunners"
          >
            <el-option label="全部" value="" />
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
            <el-option label="过时" value="stale" />
          </el-select>

          <el-button :icon="Refresh" @click="fetchRunners" :loading="loading">
            刷新
          </el-button>
        </div>
      </div>

      <el-table
        :data="runners"
        v-loading="loading"
        style="width: 100%"
        stripe
      >
        <el-table-column prop="id" label="ID" width="80" />

        <el-table-column prop="description" label="描述" min-width="200">
          <template #default="{ row }">
            <div>
              <div>{{ row.description || row.name || '-' }}</div>
              <div v-if="row.ip_address" style="color: #909399; font-size: 12px">
                {{ row.ip_address }}
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="类型" width="120">
          <template #default="{ row }">
            <el-tag
              :type="getRunnerTypeColor(row.runner_type)"
              size="small"
            >
              {{ getRunnerTypeLabel(row.runner_type) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.online ? 'success' : 'danger'"
              size="small"
            >
              {{ row.online ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="活动状态" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.active ? 'success' : 'info'"
              size="small"
            >
              {{ row.active ? '激活' : '暂停' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="共享" width="80">
          <template #default="{ row }">
            <el-tag
              v-if="row.is_shared"
              type="warning"
              size="small"
            >
              共享
            </el-tag>
            <span v-else style="color: #909399">-</span>
          </template>
        </el-table-column>

        <el-table-column prop="contacted_at" label="最后联系" width="180">
          <template #default="{ row }">
            {{ formatTime(row.contacted_at) }}
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && runners.length === 0" class="empty-state">
        <el-empty description="暂无 Runners" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { useGitlabStore } from '@/store/modules/gitlab'

const gitlabStore = useGitlabStore()

const loading = ref(false)
const runners = ref([])

const filters = ref({
  type: '',
  status: '',
  paused: null
})

// Fetch runners
const fetchRunners = async () => {
  loading.value = true
  try {
    const params = {}
    if (filters.value.type) params.type = filters.value.type
    if (filters.value.status) params.status = filters.value.status
    if (filters.value.paused !== null) params.paused = filters.value.paused

    const data = await gitlabStore.fetchRunners(params)
    runners.value = data || []
  } catch (error) {
    ElMessage.error(gitlabStore.error || '获取 Runners 失败')
    runners.value = []
  } finally {
    loading.value = false
  }
}

// Get runner type label
const getRunnerTypeLabel = (type) => {
  const labels = {
    instance_type: 'Instance',
    group_type: 'Group',
    project_type: 'Project'
  }
  return labels[type] || type
}

// Get runner type color
const getRunnerTypeColor = (type) => {
  const colors = {
    instance_type: 'danger',
    group_type: 'warning',
    project_type: 'primary'
  }
  return colors[type] || ''
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

onMounted(async () => {
  // Check if GitLab is enabled
  await gitlabStore.fetchSettings()
  if (!gitlabStore.isEnabled) {
    ElMessage.warning('GitLab 集成未启用，请先在设置中配置')
    return
  }

  fetchRunners()
})
</script>

<style scoped>
.empty-state {
  padding: 40px 0;
  text-align: center;
}
</style>
