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
              <div class="runner-description">
                {{ row.description || row.name || '-' }}
              </div>
              <div v-if="row.ip_address" class="runner-meta">
                <el-icon><Location /></el-icon>
                {{ row.ip_address }}
              </div>
              <div v-if="row.version" class="runner-meta">
                <el-icon><InfoFilled /></el-icon>
                v{{ row.version }}
                <span v-if="row.platform"> · {{ row.platform }}</span>
                <span v-if="row.architecture"> · {{ row.architecture }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="标签" min-width="150">
          <template #default="{ row }">
            <div v-if="row.tag_list && row.tag_list.length > 0" class="tag-list">
              <el-tag
                v-for="tag in row.tag_list"
                :key="tag"
                size="small"
                style="margin-right: 4px; margin-bottom: 4px"
              >
                {{ tag }}
              </el-tag>
            </div>
            <span v-else style="color: #909399">-</span>
          </template>
        </el-table-column>

        <el-table-column label="类型" width="100">
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

        <el-table-column label="激活" width="80" align="center">
          <template #default="{ row }">
            <el-tag
              :type="row.active ? 'success' : 'info'"
              size="small"
            >
              {{ row.active ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="锁定" width="80" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.locked" style="color: #f56c6c"><Lock /></el-icon>
            <span v-else style="color: #909399">-</span>
          </template>
        </el-table-column>

        <el-table-column prop="contacted_at" label="最后联系" width="160">
          <template #default="{ row }">
            {{ formatTime(row.contacted_at) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              type="primary"
              size="small"
              @click="handleEdit(row)"
            >
              编辑
            </el-button>
            <el-button
              link
              type="danger"
              size="small"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && runners.length === 0" class="empty-state">
        <el-empty description="暂无 Runners" />
      </div>
    </div>

    <!-- 编辑 Runner 对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑 Runner"
      width="600px"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        label-width="100px"
      >
        <el-form-item label="描述">
          <el-input v-model="editForm.description" placeholder="请输入 Runner 描述" />
        </el-form-item>

        <el-form-item label="激活状态">
          <el-switch v-model="editForm.active" />
          <span style="margin-left: 10px; color: #909399">
            {{ editForm.active ? '激活' : '暂停' }}
          </span>
        </el-form-item>

        <el-form-item label="锁定">
          <el-switch v-model="editForm.locked" />
          <span style="margin-left: 10px; color: #909399">
            {{ editForm.locked ? '已锁定' : '未锁定' }}
          </span>
        </el-form-item>

        <el-form-item label="标签">
          <el-select
            v-model="editForm.tag_list"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入标签后按回车添加"
            style="width: 100%"
          >
            <el-option
              v-for="tag in editForm.tag_list"
              :key="tag"
              :label="tag"
              :value="tag"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="访问级别">
          <el-select v-model="editForm.access_level" placeholder="选择访问级别">
            <el-option label="不受保护 (not_protected)" value="not_protected" />
            <el-option label="受保护的 (ref_protected)" value="ref_protected" />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleEditSubmit" :loading="submitting">
            保存
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Location, InfoFilled, Lock } from '@element-plus/icons-vue'
import { useGitlabStore } from '@/store/modules/gitlab'
import * as gitlabApi from '@/api/gitlab'

const gitlabStore = useGitlabStore()

const loading = ref(false)
const runners = ref([])
const submitting = ref(false)

const filters = ref({
  type: '',
  status: '',
  paused: null
})

// Edit dialog
const editDialogVisible = ref(false)
const editFormRef = ref(null)
const editForm = ref({
  id: null,
  description: '',
  active: true,
  locked: false,
  tag_list: [],
  access_level: ''
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

// Handle edit
const handleEdit = async (runner) => {
  editForm.value = {
    id: runner.id,
    description: runner.description || '',
    active: runner.active,
    locked: runner.locked || false,
    tag_list: runner.tag_list ? [...runner.tag_list] : [],
    access_level: runner.access_level || ''
  }
  editDialogVisible.value = true
}

// Handle edit submit
const handleEditSubmit = async () => {
  submitting.value = true
  try {
    const updateData = {
      description: editForm.value.description,
      active: editForm.value.active,
      locked: editForm.value.locked,
      tag_list: editForm.value.tag_list
    }

    if (editForm.value.access_level) {
      updateData.access_level = editForm.value.access_level
    }

    await gitlabApi.updateGitlabRunner(editForm.value.id, updateData)
    ElMessage.success('Runner 更新成功')
    editDialogVisible.value = false
    fetchRunners()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || 'Runner 更新失败')
  } finally {
    submitting.value = false
  }
}

// Handle delete
const handleDelete = async (runner) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 Runner "${runner.description || runner.name || runner.id}" 吗？此操作不可撤销。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    loading.value = true
    await gitlabApi.deleteGitlabRunner(runner.id)
    ElMessage.success('Runner 删除成功')
    fetchRunners()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || 'Runner 删除失败')
      loading.value = false
    }
  }
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

.runner-description {
  font-weight: 500;
  margin-bottom: 4px;
}

.runner-meta {
  display: flex;
  align-items: center;
  color: #909399;
  font-size: 12px;
  margin-top: 4px;
}

.runner-meta .el-icon {
  margin-right: 4px;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
}
</style>
