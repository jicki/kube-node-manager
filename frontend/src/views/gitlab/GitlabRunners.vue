<template>
  <div class="page-container">
    <div class="card-container">
      <!-- Statistics Cards -->
      <div class="stats-container">
        <div class="stat-card">
          <div class="stat-label">总数</div>
          <div class="stat-value">{{ runners.length }}</div>
        </div>
        <div class="stat-card stat-online">
          <div class="stat-label">在线</div>
          <div class="stat-value">
            {{ onlineCount }}
            <span class="stat-icon">●</span>
          </div>
        </div>
        <div class="stat-card stat-offline">
          <div class="stat-label">离线</div>
          <div class="stat-value">
            {{ offlineCount }}
            <span class="stat-icon">●</span>
          </div>
        </div>
      </div>

      <div class="toolbar">
        <div class="toolbar-left">
          <h2>GitLab Runners</h2>
          <div v-if="selectedRunners.length > 0" style="display: flex; align-items: center; gap: 8px; margin-left: 16px;">
            <el-button
              type="success"
              @click="handleBatchActivate"
            >
              批量激活 ({{ selectedRunners.length }})
            </el-button>
            <el-button
              type="warning"
              @click="handleBatchDeactivate"
            >
              批量停用 ({{ selectedRunners.length }})
            </el-button>
            <el-button
              type="danger"
              :disabled="!canBatchDelete"
              @click="handleBatchDelete"
            >
              批量删除 ({{ selectedOfflineCount }}/{{ selectedRunners.length }})
            </el-button>
            <span v-if="!canBatchDelete" style="color: #f56c6c; font-size: 12px">
              只能删除离线状态的 Runner
            </span>
          </div>
        </div>
        <div class="toolbar-right">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索标签或所有者"
            clearable
            style="width: 200px; margin-right: 8px"
            @input="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>

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

          <el-select
            v-model="filters.neverContacted"
            placeholder="联系状态"
            clearable
            style="width: 150px; margin-right: 8px"
            @change="handleFilterChange"
          >
            <el-option label="全部" value="" />
            <el-option label="从未联系" value="true" />
            <el-option label="已有联系" value="false" />
          </el-select>

          <el-select
            v-model="filters.active"
            placeholder="激活状态"
            clearable
            style="width: 120px; margin-right: 8px"
            @change="handleFilterChange"
          >
            <el-option label="全部" value="" />
            <el-option label="激活" value="true" />
            <el-option label="未激活" value="false" />
          </el-select>

          <el-button type="primary" @click="handleCreate">
            新建 Runner
          </el-button>

          <el-button
            v-if="createdRunner.token"
            type="success"
            @click="handleViewCreatedRunner"
          >
            查看最近创建的 Token
          </el-button>

          <el-button :icon="Refresh" @click="() => fetchRunners(true)" :loading="loading">
            刷新
          </el-button>
        </div>
      </div>

      <el-table
        ref="tableRef"
        :data="paginatedRunners"
        v-loading="loading"
        style="width: 100%"
        stripe
        :default-sort="{ prop: 'id', order: 'descending' }"
        @selection-change="handleSelectionChange"
        @sort-change="handleSortChange"
      >
        <el-table-column
          type="selection"
          width="55"
        />

        <el-table-column prop="id" label="ID" width="100" sortable align="center" />

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
            </div>
          </template>
        </el-table-column>

        <el-table-column label="标签" min-width="180">
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

        <el-table-column prop="runner_type" label="类型" width="110" align="center">
          <template #default="{ row }">
            <el-tag
              :type="getRunnerTypeColor(row.runner_type)"
              size="small"
            >
              {{ getRunnerTypeLabel(row.runner_type) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="所有者" min-width="220">
          <template #default="{ row }">
            <div v-if="getOwnerInfo(row)" class="owner-info">
              {{ getOwnerInfo(row) }}
            </div>
            <span v-else style="color: #909399">-</span>
          </template>
        </el-table-column>

        <el-table-column prop="online" label="在线状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag
              :type="row.online ? 'success' : 'danger'"
              size="small"
            >
              {{ row.online ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="active" label="激活状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag
              :type="row.active ? 'success' : 'info'"
              size="small"
            >
              {{ row.active ? '激活' : '未激活' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="paused" label="暂停状态" width="110" align="center">
          <template #default="{ row }">
            <el-tag
              :type="row.paused ? 'warning' : 'success'"
              size="small"
            >
              {{ row.paused ? '已暂停' : '运行中' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="配置" width="200">
          <template #default="{ row }">
            <div style="font-size: 12px; color: #606266; line-height: 1.8;">
              <div style="margin-bottom: 2px;">
                {{ getAccessLevelLabel(row.access_level) }}
              </div>
              <div v-if="row.tag_list && row.tag_list.length > 0">
                运行已打标签的作业
              </div>
              <div v-else>
                运行未打标签的作业
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="最后联系" width="170" sortable :sort-method="sortByContactedAt" align="center">
          <template #default="{ row }">
            {{ formatTime(row.contacted_at) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <el-dropdown @command="handleCommand($event, row)">
              <el-button type="primary" size="small">
                操作
                <el-icon class="el-icon--right"><arrow-down /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="details">
                    <el-icon><InfoFilled /></el-icon>
                    详情
                  </el-dropdown-item>
                  <el-dropdown-item command="edit">
                    <el-icon><Edit /></el-icon>
                    编辑
                  </el-dropdown-item>
                  <el-dropdown-item v-if="row.is_platform_created" command="viewToken" divided>
                    <el-icon><Key /></el-icon>
                    查看 Token
                  </el-dropdown-item>
                  <el-dropdown-item v-if="row.is_platform_created" command="resetToken">
                    <el-icon><RefreshRight /></el-icon>
                    重置 Token
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided>
                    <el-icon><Delete /></el-icon>
                    删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && filteredRunners.length === 0" class="empty-state">
        <el-empty :description="searchKeyword ? '没有找到匹配的 Runners' : '暂无 Runners'" />
      </div>

      <div v-if="sortedRunners.length > 0" class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.currentPage"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="sortedRunners.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
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
            <el-option label="不受保护" value="not_protected" />
            <el-option label="受保护" value="ref_protected" />
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

    <!-- 新建 Runner 对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="新建 Runner"
      width="700px"
    >
      <el-form
        ref="createFormRef"
        :model="createForm"
        :rules="createFormRules"
        label-width="auto"
      >
        <el-alert
          title="创建 Instance Runner"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 20px;"
        >
          <template #default>
            Instance Runner 可用于 GitLab 实例中的所有项目和组
          </template>
        </el-alert>

        <el-form-item label="描述" prop="description">
          <el-input
            v-model="createForm.description"
            placeholder="请输入 Runner 描述"
            maxlength="100"
            show-word-limit
          />
        </el-form-item>

        <el-divider content-position="left">
          <span style="font-size: 14px; color: #606266;">配置（可选）</span>
        </el-divider>

        <el-form-item label="标签">
          <el-select
            v-model="createForm.tag_list"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入标签后按回车添加，例如：docker, linux"
            style="width: 100%"
          >
            <el-option
              v-for="tag in createForm.tag_list"
              :key="tag"
              :label="tag"
              :value="tag"
            />
          </el-select>
          <div style="color: #909399; font-size: 12px; margin-top: 4px;">
            添加标签以指定 Runner 可执行的作业类型
          </div>
        </el-form-item>

        <el-form-item label="运行未打标签作业">
          <el-switch v-model="createForm.run_untagged" />
          <span style="margin-left: 10px; color: #909399; font-size: 12px;">
            {{ createForm.run_untagged ? '允许执行没有标签的作业' : '仅执行带标签的作业' }}
          </span>
        </el-form-item>

        <el-form-item label="Runner 描述">
          <el-input
            v-model="createForm.runner_description"
            type="textarea"
            :rows="2"
            placeholder="可选：添加关于此 Runner 的额外描述信息"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="受保护">
          <el-switch v-model="createForm.protected" />
          <span style="margin-left: 10px; color: #909399; font-size: 12px;">
            {{ createForm.protected ? '仅用于受保护的分支' : '可用于所有分支' }}
          </span>
        </el-form-item>

        <el-form-item label="锁定到当前项目">
          <el-switch v-model="createForm.locked" />
          <span style="margin-left: 10px; color: #909399; font-size: 12px;">
            {{ createForm.locked ? '已锁定' : '未锁定' }}
          </span>
        </el-form-item>

        <el-form-item label="最大作业超时">
          <el-input
            v-model.number="createForm.maximum_timeout"
            type="number"
            placeholder="留空使用默认值（最少 600 秒）"
            :min="600"
          >
            <template #append>秒</template>
          </el-input>
          <div style="color: #909399; font-size: 12px; margin-top: 4px;">
            Runner 在结束作业前可以运行的最大时间
          </div>
        </el-form-item>

        <el-form-item label="已暂停">
          <el-switch v-model="createForm.paused" />
          <span style="margin-left: 10px; color: #909399; font-size: 12px;">
            {{ createForm.paused ? 'Runner 已暂停，不接收新作业' : 'Runner 运行中' }}
          </span>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleCreateSubmit" :loading="submitting">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- Runner Token 对话框 -->
    <el-dialog
      v-model="tokenDialogVisible"
      :title="tokenDialogTitle"
      width="700px"
    >
      <el-alert
        v-if="tokenDialogMode === 'create'"
        title="重要提示"
        type="warning"
        :closable="false"
        show-icon
      >
        <p>此 Token 只会显示一次，请妥善保存！</p>
        <p style="margin-top: 8px; font-size: 12px;">
          💡 提示：在刷新页面前，您可以随时点击"查看最近创建的 Token"按钮重新查看。
        </p>
      </el-alert>

      <el-alert
        v-if="tokenDialogMode === 'view'"
        title="Token 信息"
        type="info"
        :closable="false"
        show-icon
      >
        <p>您可以复制此 Token 用于 Runner 注册。</p>
      </el-alert>

      <el-descriptions :column="1" border style="margin-top: 20px;">
        <el-descriptions-item label="Runner ID">{{ createdRunner.id }}</el-descriptions-item>
        <el-descriptions-item label="描述">{{ createdRunner.description }}</el-descriptions-item>
        <el-descriptions-item label="类型">
          {{ createdRunner.runner_type === 'instance_type' ? 'Instance Runner' : createdRunner.runner_type }}
        </el-descriptions-item>
        <el-descriptions-item v-if="tokenDialogMode === 'view' && createdRunner.created_by" label="创建者">
          {{ createdRunner.created_by }}
        </el-descriptions-item>
        <el-descriptions-item v-if="tokenDialogMode === 'view' && createdRunner.created_at" label="创建时间">
          {{ formatTime(createdRunner.created_at) }}
        </el-descriptions-item>
        <el-descriptions-item label="Token">
          <div style="display: flex; align-items: center; gap: 8px;">
            <el-input
              :model-value="createdRunner.token"
              readonly
              style="flex: 1;"
            >
              <template #append>
                <el-button @click="copyToken">
                  <el-icon><DocumentCopy /></el-icon>
                  复制
                </el-button>
              </template>
            </el-input>
          </div>
        </el-descriptions-item>
      </el-descriptions>

      <template #footer>
        <span class="dialog-footer">
          <el-button type="primary" @click="handleTokenDialogClose">
            {{ tokenDialogMode === 'create' ? '我已保存 Token' : '关闭' }}
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 详情 Runner 对话框 -->
    <el-dialog
      v-model="detailsDialogVisible"
      title="Runner 详情"
      width="900px"
      class="runner-details-dialog"
    >
      <el-descriptions :column="2" border size="default" v-if="selectedRunner">
        <el-descriptions-item label="ID">{{ selectedRunner.id }}</el-descriptions-item>
        <el-descriptions-item label="描述">{{ selectedRunner.description || '-' }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ selectedRunner.name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="IP地址">{{ selectedRunner.ip_address || '-' }}</el-descriptions-item>

        <el-descriptions-item label="在线状态">
          <el-tag :type="selectedRunner.online ? 'success' : 'danger'" size="small">
            {{ selectedRunner.online ? '在线' : '离线' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="激活状态">
          <el-tag :type="selectedRunner.active ? 'success' : 'info'" size="small">
            {{ selectedRunner.active ? '激活' : '未激活' }}
          </el-tag>
        </el-descriptions-item>

        <el-descriptions-item label="暂停状态">
          <el-tag :type="selectedRunner.paused ? 'warning' : 'success'" size="small">
            {{ selectedRunner.paused ? '已暂停' : '运行中' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="Runner类型">
          <el-tag :type="getRunnerTypeColor(selectedRunner.runner_type)" size="small">
            {{ getRunnerTypeLabel(selectedRunner.runner_type) }}
          </el-tag>
        </el-descriptions-item>

        <el-descriptions-item label="是否共享">
          {{ selectedRunner.is_shared ? '是' : '否' }}
        </el-descriptions-item>
        <el-descriptions-item label="访问级别">
          {{ getAccessLevelLabel(selectedRunner.access_level) }}
        </el-descriptions-item>

        <el-descriptions-item label="版本" :span="2">
          {{ selectedRunner.version || '-' }}
        </el-descriptions-item>

        <el-descriptions-item label="架构">
          {{ selectedRunner.architecture || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="平台">
          {{ selectedRunner.platform || '-' }}
        </el-descriptions-item>

        <el-descriptions-item label="创建时间" :span="2">
          {{ formatTime(selectedRunner.created_at) }}
        </el-descriptions-item>

        <el-descriptions-item label="最后联系" :span="2">
          {{ formatTime(selectedRunner.contacted_at) }}
        </el-descriptions-item>

        <el-descriptions-item label="标签" :span="2">
          <div v-if="selectedRunner.tag_list && selectedRunner.tag_list.length > 0">
            <el-tag
              v-for="tag in selectedRunner.tag_list"
              :key="tag"
              size="small"
              style="margin-right: 4px; margin-bottom: 4px"
            >
              {{ tag }}
            </el-tag>
          </div>
          <span v-else style="color: #909399">-</span>
        </el-descriptions-item>

        <el-descriptions-item label="所属项目" :span="2" v-if="selectedRunner.projects && selectedRunner.projects.length > 0">
          <div style="max-height: 150px; overflow-y: auto;">
            <div v-for="project in selectedRunner.projects" :key="project.id" style="margin: 4px 0;">
              {{ project.name_with_namespace || project.name }}
            </div>
          </div>
        </el-descriptions-item>

        <el-descriptions-item label="所属组" :span="2" v-if="selectedRunner.groups && selectedRunner.groups.length > 0">
          <div style="max-height: 150px; overflow-y: auto;">
            <div v-for="group in selectedRunner.groups" :key="group.id" style="margin: 4px 0;">
              {{ group.full_path || group.name }}
            </div>
          </div>
        </el-descriptions-item>
      </el-descriptions>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="detailsDialogVisible = false">关闭</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Refresh, 
  Location, 
  Search, 
  DocumentCopy, 
  ArrowDown, 
  InfoFilled, 
  Edit, 
  Key, 
  RefreshRight, 
  Delete 
} from '@element-plus/icons-vue'
import { useGitlabStore } from '@/store/modules/gitlab'
import * as gitlabApi from '@/api/gitlab'

const gitlabStore = useGitlabStore()

const loading = ref(false)
const runners = ref([])
const submitting = ref(false)
const selectedRunners = ref([])
const searchKeyword = ref('')
const tableRef = ref(null)
const currentSort = ref({
  prop: 'id',
  order: 'descending'
})

// Cache for runners data with timestamp
const runnersCache = ref({
  data: [],
  timestamp: 0,
  filters: {}
})
const CACHE_DURATION = 30 * 1000 // 30 seconds cache

const filters = ref({
  type: '',
  status: '',
  paused: null,
  neverContacted: '',
  active: ''
})

// Pagination
const pagination = ref({
  currentPage: 1,
  pageSize: 20
})

// Edit dialog
const editDialogVisible = ref(false)
const editFormRef = ref(null)
const editForm = ref({
  id: null,
  description: '',
  active: true,
  tag_list: [],
  access_level: ''
})

// Details dialog
const detailsDialogVisible = ref(false)
const selectedRunner = ref(null)

// Create dialog
const createDialogVisible = ref(false)
const createFormRef = ref(null)
const createForm = ref({
  description: '',
  tag_list: [],
  run_untagged: false,           // 运行未打标签作业 - 默认不开启
  runner_description: '',        // Runner 额外描述
  protected: false,              // 受保护 - 默认不开启
  locked: false,                 // 锁定到当前项目 - 默认不开启
  maximum_timeout: null,         // 最大作业超时（秒）
  paused: false                  // 已暂停 - 默认不开启
})

// Form validation rules
const createFormRules = {
  description: [
    { required: true, message: '请输入描述', trigger: 'blur' },
    { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' }
  ]
}

// Token dialog
const tokenDialogVisible = ref(false)
const tokenDialogTitle = ref('Runner Token')
const tokenDialogMode = ref('view') // 'create' or 'view'
const createdRunner = ref({
  id: null,
  token: '',
  description: '',
  runner_type: '',
  created_by: '',
  created_at: ''
})

// Fetch runners with caching
const fetchRunners = async (forceRefresh = false) => {
  const params = {}
  if (filters.value.type) params.type = filters.value.type
  if (filters.value.status) params.status = filters.value.status
  if (filters.value.paused !== null) params.paused = filters.value.paused

  // Create cache key from filters
  const cacheKey = JSON.stringify(params)
  const now = Date.now()

  // Check if we can use cached data
  if (
    !forceRefresh &&
    runnersCache.value.data.length > 0 &&
    runnersCache.value.filters === cacheKey &&
    (now - runnersCache.value.timestamp) < CACHE_DURATION
  ) {
    // Use cached data
    runners.value = runnersCache.value.data
    restoreSort()
    return
  }

  loading.value = true
  try {
    const data = await gitlabStore.fetchRunners(params)
    // Add is_platform_created flag to each runner
    runners.value = (data || []).map(runner => ({
      ...runner,
      is_platform_created: runner.is_platform_created || false
    }))

    // Debug: Log first runner's configuration data
    if (data && data.length > 0) {
      console.log('Sample runner data:', {
        id: data[0].id,
        version: data[0].version,
        architecture: data[0].architecture,
        platform: data[0].platform,
        is_platform_created: data[0].is_platform_created
      })
    }

    // Update cache
    runnersCache.value = {
      data: runners.value,
      timestamp: now,
      filters: cacheKey
    }

    // Restore sort after data is loaded
    restoreSort()
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

// Get access level label
const getAccessLevelLabel = (accessLevel) => {
  const labels = {
    not_protected: '不受保护',
    ref_protected: '受保护'
  }
  return labels[accessLevel] || accessLevel || '-'
}

// Get owner information
const getOwnerInfo = (row) => {
  // For shared/instance runners
  if (row.is_shared || row.runner_type === 'instance_type') {
    return '共享 Runner'
  }

  // For group runners
  if (row.runner_type === 'group_type' && row.groups && row.groups.length > 0) {
    return row.groups.map(g => g.full_path || g.name).join(', ')
  }

  // For project runners
  if (row.runner_type === 'project_type' && row.projects && row.projects.length > 0) {
    // Use name_with_namespace for better readability
    return row.projects.map(p => p.name_with_namespace || p.path_with_namespace || p.name).join(', ')
  }

  return null
}

// Filtered runners based on search keyword and filters
const filteredRunners = computed(() => {
  let result = runners.value

  // Filter by never contacted
  if (filters.value.neverContacted === 'true') {
    result = result.filter(runner => !runner.contacted_at)
  } else if (filters.value.neverContacted === 'false') {
    result = result.filter(runner => runner.contacted_at)
  }

  // Filter by active status
  if (filters.value.active === 'true') {
    result = result.filter(runner => runner.active)
  } else if (filters.value.active === 'false') {
    result = result.filter(runner => !runner.active)
  }

  // Filter by search keyword
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(runner => {
      // Search in tags
      if (runner.tag_list && runner.tag_list.some(tag => tag.toLowerCase().includes(keyword))) {
        return true
      }

      // Search in owner info
      const ownerInfo = getOwnerInfo(runner)
      if (ownerInfo && ownerInfo.toLowerCase().includes(keyword)) {
        return true
      }

      // Search in description
      if (runner.description && runner.description.toLowerCase().includes(keyword)) {
        return true
      }

      return false
    })
  }

  return result
})

// Handle search input
const handleSearch = () => {
  // The computed property will automatically update
}

// Handle filter change
const handleFilterChange = () => {
  // Reset to first page when filter changes
  pagination.value.currentPage = 1
}

// Sorted and filtered runners
const sortedRunners = computed(() => {
  let result = [...filteredRunners.value]

  if (currentSort.value.prop) {
    const { prop, order } = currentSort.value

    result.sort((a, b) => {
      let compareResult = 0

      // Use custom sort methods
      if (prop === 'contacted_at') {
        compareResult = sortByContactedAt(a, b)
      } else {
        // Default sorting for other props (like ID)
        const aVal = a[prop]
        const bVal = b[prop]

        if (typeof aVal === 'number' && typeof bVal === 'number') {
          compareResult = aVal - bVal
        } else {
          compareResult = String(aVal || '').localeCompare(String(bVal || ''))
        }
      }

      // Apply sort order (ascending or descending)
      return order === 'ascending' ? compareResult : -compareResult
    })
  }

  return result
})

// Paginated runners
const paginatedRunners = computed(() => {
  const start = (pagination.value.currentPage - 1) * pagination.value.pageSize
  const end = start + pagination.value.pageSize
  return sortedRunners.value.slice(start, end)
})

// Pagination handlers
const handleSizeChange = (newSize) => {
  pagination.value.pageSize = newSize
  pagination.value.currentPage = 1
}

const handleCurrentChange = (newPage) => {
  pagination.value.currentPage = newPage
}

// Sort by tag list
const sortByTagList = (a, b) => {
  const tagsA = a.tag_list && a.tag_list.length > 0 ? a.tag_list.join(',') : ''
  const tagsB = b.tag_list && b.tag_list.length > 0 ? b.tag_list.join(',') : ''
  return tagsA.localeCompare(tagsB)
}

// Sort by owner
const sortByOwner = (a, b) => {
  const ownerA = getOwnerInfo(a) || ''
  const ownerB = getOwnerInfo(b) || ''
  return ownerA.localeCompare(ownerB)
}

// Sort by contacted_at (handle null values)
const sortByContactedAt = (a, b) => {
  const timeA = a.contacted_at ? new Date(a.contacted_at).getTime() : 0
  const timeB = b.contacted_at ? new Date(b.contacted_at).getTime() : 0
  return timeA - timeB
}

// Sort by configuration (version primarily)
const sortByConfig = (a, b) => {
  const versionA = a.version || ''
  const versionB = b.version || ''
  return versionA.localeCompare(versionB)
}

// Handle sort change
const handleSortChange = ({ prop, order }) => {
  currentSort.value = { prop, order }
}

// Restore sort after data update
const restoreSort = () => {
  if (tableRef.value && currentSort.value.prop) {
    // Use nextTick to ensure DOM is updated
    nextTick(() => {
      // Set the table's visual sort indicator
      tableRef.value.sort(currentSort.value.prop, currentSort.value.order)
    })
  }
}

// Format time
const formatTime = (time) => {
  if (!time) return '-'

  const date = new Date(time)
  if (isNaN(date.getTime())) return '-'

  // Check if it's a valid date (not zero value)
  const year = date.getFullYear()
  if (year < 1900) return '-'

  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// Handle view details
const handleViewDetails = (runner) => {
  selectedRunner.value = runner
  detailsDialogVisible.value = true
}

// Handle edit
const handleEdit = (runner) => {
  editForm.value = {
    id: runner.id,
    description: runner.description || '',
    active: runner.active,
    tag_list: runner.tag_list || [],
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
      tag_list: editForm.value.tag_list
    }

    if (editForm.value.access_level) {
      updateData.access_level = editForm.value.access_level
    }

    await gitlabApi.updateGitlabRunner(editForm.value.id, updateData)
    ElMessage.success('Runner 更新成功')
    editDialogVisible.value = false
    fetchRunners(true)
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
    fetchRunners(true)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || 'Runner 删除失败')
      loading.value = false
    }
  }
}

// Handle selection change
const handleSelectionChange = (selection) => {
  selectedRunners.value = selection
}

// Computed: selected offline count
const selectedOfflineCount = computed(() => {
  return selectedRunners.value.filter(r => !r.online).length
})

// Computed: can batch delete (all selected are offline)
const canBatchDelete = computed(() => {
  return selectedRunners.value.length > 0 &&
         selectedRunners.value.every(r => !r.online)
})

// Computed: online count
const onlineCount = computed(() => {
  return runners.value.filter(r => r.online).length
})

// Computed: offline count
const offlineCount = computed(() => {
  return runners.value.filter(r => !r.online).length
})

// Handle batch delete
const handleBatchDelete = async () => {
  const offlineRunners = selectedRunners.value.filter(r => !r.online)

  if (offlineRunners.length === 0) {
    ElMessage.warning('请选择离线状态的 Runner')
    return
  }

  try {
    await ElMessageBox.confirm(
      '',
      '确认批量删除',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: true,
        customClass: 'batch-delete-dialog',
        message: `
          <div style="margin-bottom: 16px;">
            <p style="margin-bottom: 12px; font-size: 14px; color: #606266;">
              确定要删除以下 <strong style="color: #f56c6c;">${offlineRunners.length}</strong> 个离线 Runner 吗？此操作不可撤销。
            </p>
            <div style="background: #f5f7fa; padding: 12px; border-radius: 4px; max-height: 300px; overflow-y: auto;">
              <ul style="margin: 0; padding-left: 20px; list-style-type: disc;">
                ${offlineRunners.map(r => `<li style="margin: 6px 0; color: #606266; font-size: 13px;">${r.description || r.name || 'ID: ' + r.id}</li>`).join('')}
              </ul>
            </div>
          </div>
        `
      }
    )

    loading.value = true
    let successCount = 0
    let failCount = 0
    const errors = []

    for (const runner of offlineRunners) {
      try {
        await gitlabApi.deleteGitlabRunner(runner.id)
        successCount++
      } catch (error) {
        failCount++
        errors.push(`${runner.description || runner.id}: ${error.response?.data?.error || '删除失败'}`)
      }
    }

    if (successCount > 0) {
      ElMessage.success(`成功删除 ${successCount} 个 Runner${failCount > 0 ? `，失败 ${failCount} 个` : ''}`)
    }

    if (failCount > 0 && errors.length > 0) {
      console.error('批量删除错误：', errors)
    }

    selectedRunners.value = []
    fetchRunners(true)
  } catch (error) {
    if (error !== 'cancel') {
      loading.value = false
    }
  }
}

// Handle batch activate
const handleBatchActivate = async () => {
  if (selectedRunners.value.length === 0) {
    ElMessage.warning('请选择要激活的 Runner')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要激活选中的 ${selectedRunners.value.length} 个 Runner 吗？`,
      '确认批量激活',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info',
      }
    )

    loading.value = true
    let successCount = 0
    let failCount = 0

    for (const runner of selectedRunners.value) {
      try {
        await gitlabApi.updateGitlabRunner(runner.id, { active: true })
        successCount++
      } catch (error) {
        failCount++
      }
    }

    if (successCount > 0) {
      ElMessage.success(`成功激活 ${successCount} 个 Runner${failCount > 0 ? `，失败 ${failCount} 个` : ''}`)
    }

    selectedRunners.value = []
    fetchRunners(true)
  } catch (error) {
    if (error !== 'cancel') {
      loading.value = false
    }
  }
}

// Handle batch deactivate
const handleBatchDeactivate = async () => {
  if (selectedRunners.value.length === 0) {
    ElMessage.warning('请选择要停用的 Runner')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要停用选中的 ${selectedRunners.value.length} 个 Runner 吗？`,
      '确认批量停用',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    loading.value = true
    let successCount = 0
    let failCount = 0

    for (const runner of selectedRunners.value) {
      try {
        await gitlabApi.updateGitlabRunner(runner.id, { active: false })
        successCount++
      } catch (error) {
        failCount++
      }
    }

    if (successCount > 0) {
      ElMessage.success(`成功停用 ${successCount} 个 Runner${failCount > 0 ? `，失败 ${failCount} 个` : ''}`)
    }

    selectedRunners.value = []
    fetchRunners(true)
  } catch (error) {
    if (error !== 'cancel') {
      loading.value = false
    }
  }
}

// Handle create runner
const handleCreate = () => {
  // Reset form
  createForm.value = {
    description: '',
    tag_list: [],
    run_untagged: false,
    runner_description: '',
    protected: false,
    locked: false,
    maximum_timeout: null,
    paused: false
  }
  createDialogVisible.value = true
}

// Handle create submit
const handleCreateSubmit = async () => {
  if (!createFormRef.value) return

  try {
    await createFormRef.value.validate()
  } catch (error) {
    return
  }

  submitting.value = true
  try {
    // Prepare request data - only for Instance Runner
    const data = {
      runner_type: 'instance_type',
      description: createForm.value.description,
      tag_list: createForm.value.tag_list,
      run_untagged: createForm.value.run_untagged,
      locked: createForm.value.locked,
      paused: createForm.value.paused
    }

    // Add optional fields if provided
    if (createForm.value.runner_description) {
      data.description = createForm.value.description + ' - ' + createForm.value.runner_description
    }

    if (createForm.value.protected) {
      data.access_level = 'ref_protected'
    } else {
      data.access_level = 'not_protected'
    }

    if (createForm.value.maximum_timeout) {
      data.maximum_timeout = createForm.value.maximum_timeout
    }

    const response = await gitlabApi.createGitlabRunner(data)
    
    // API 返回的数据可能在 response 或 response.data 中
    const runnerData = response.data || response
    
    console.log('API Response:', runnerData) // 调试信息
    
      // Store created runner info
      createdRunner.value = {
        id: runnerData.id,
        token: runnerData.token,
        description: data.description, // 使用我们提交的描述
        runner_type: runnerData.runner_type || 'instance_type',
        created_by: '',
        created_at: ''
      }

      console.log('Created Runner:', createdRunner.value) // 调试信息

      ElMessage.success('Runner 创建成功')
      createDialogVisible.value = false
    
      // Show token dialog
      tokenDialogMode.value = 'create'
      tokenDialogTitle.value = 'Runner 创建成功'
      tokenDialogVisible.value = true

    // Refresh runners list
    fetchRunners(true)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || 'Runner 创建失败')
  } finally {
    submitting.value = false
  }
}

// Copy token to clipboard
const copyToken = async () => {
  try {
    if (!createdRunner.value.token) {
      ElMessage.warning('Token 为空，无法复制')
      return
    }
    await navigator.clipboard.writeText(createdRunner.value.token)
    ElMessage.success('Token 已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制')
  }
}

// Handle token dialog close
const handleTokenDialogClose = () => {
  tokenDialogVisible.value = false
  // 不清除 token，保留在内存中，方便用户重新查看
  // 注意：刷新页面后 token 会丢失，这是安全的
}

// Handle view created runner token
const handleViewCreatedRunner = () => {
  if (createdRunner.value.token) {
    tokenDialogMode.value = 'create'
    tokenDialogTitle.value = 'Runner 创建成功'
    tokenDialogVisible.value = true
  } else {
    ElMessage.warning('Token 已过期或不可用，请重新创建 Runner')
  }
}

// Handle view token
const handleViewToken = async (runner) => {
  try {
    const response = await gitlabApi.getGitlabRunnerToken(runner.id)
    createdRunner.value = {
      id: response.data.runner_id,
      token: response.data.token,
      description: response.data.description,
      runner_type: response.data.runner_type,
      created_by: response.data.created_by,
      created_at: response.data.created_at
    }
    tokenDialogMode.value = 'view'
    tokenDialogTitle.value = 'Runner Token'
    tokenDialogVisible.value = true
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '获取 Token 失败')
  }
}

// Handle reset token
const handleResetToken = async (runner) => {
  try {
    await ElMessageBox.confirm(
      '重置 Token 后，原有 Token 将立即失效，已注册的 Runner 需要重新注册。确定要继续吗？',
      '确认重置 Token',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const response = await gitlabApi.resetGitlabRunnerToken(runner.id)
    createdRunner.value = {
      id: response.data.id,
      token: response.data.token,
      description: response.data.description,
      runner_type: response.data.runner_type,
      created_by: '',
      created_at: ''
    }
    tokenDialogMode.value = 'create'
    tokenDialogTitle.value = 'Token 重置成功'
    tokenDialogVisible.value = true
    ElMessage.success('Token 重置成功')
    
    // Refresh list
    fetchRunners(true)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || 'Token 重置失败')
    }
  }
}

// Handle dropdown menu command
const handleCommand = (command, runner) => {
  switch (command) {
    case 'details':
      handleViewDetails(runner)
      break
    case 'edit':
      handleEdit(runner)
      break
    case 'viewToken':
      handleViewToken(runner)
      break
    case 'resetToken':
      handleResetToken(runner)
      break
    case 'delete':
      handleDelete(runner)
      break
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
.stats-container {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  flex: 1;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  transition: all 0.3s;
}

.stat-card:hover {
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  color: #303133;
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-icon {
  font-size: 16px;
}

.stat-online .stat-value {
  color: #67c23a;
}

.stat-online .stat-icon {
  color: #67c23a;
}

.stat-offline .stat-value {
  color: #f56c6c;
}

.stat-offline .stat-icon {
  color: #f56c6c;
}

.empty-state {
  padding: 40px 0;
  text-align: center;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0;
  margin-top: 16px;
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
  gap: 4px;
}

.owner-info {
  color: #606266;
  font-size: 14px;
}

/* Table header improvements */
:deep(.el-table th) {
  white-space: nowrap;
  padding: 12px 0;
}

:deep(.el-table th .cell) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  white-space: nowrap;
}

:deep(.el-table th.is-sortable .cell) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

:deep(.el-table .caret-wrapper) {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  height: 14px;
  width: 14px;
  margin-left: 4px;
}
</style>

<style>
/* Global style for batch delete dialog - not scoped */
.batch-delete-dialog {
  width: 520px !important;
  max-width: 90vw !important;
}

.batch-delete-dialog .el-message-box__content {
  padding: 20px 20px 0 !important;
}

.batch-delete-dialog .el-message-box__message {
  padding: 0 !important;
}

/* Runner details dialog - not scoped */
.runner-details-dialog {
  max-width: 95vw !important;
}

.runner-details-dialog .el-dialog__body {
  padding: 20px !important;
  max-height: 70vh;
  overflow-y: auto;
}

.runner-details-dialog .el-descriptions__label {
  width: 120px !important;
  min-width: 120px !important;
  white-space: nowrap;
  font-weight: 600;
  background-color: #fafafa;
}

.runner-details-dialog .el-descriptions__content {
  word-break: break-all;
}

.runner-details-dialog .el-descriptions {
  width: 100%;
}

.runner-details-dialog .el-descriptions__table {
  width: 100% !important;
  table-layout: fixed;
}

.runner-details-dialog .el-descriptions__cell {
  padding: 12px 16px !important;
}

/* Dropdown menu styles */
.el-dropdown-menu__item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.el-dropdown-menu__item .el-icon {
  font-size: 14px;
}
</style>
