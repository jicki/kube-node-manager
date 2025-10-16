<template>
  <div class="page-container">
    <!-- 群组查询区域 -->
    <div class="card-container">
      <h2>群组查询</h2>
      <el-divider />
      
      <div class="query-section">
        <el-form :inline="true" :model="queryForm" class="query-form">
          <el-form-item label="Chat ID">
            <el-input
              v-model="queryForm.chatId"
              placeholder="例如: oc_1415d16708042a4c3822fabda26d6ae4"
              style="width: 350px"
              clearable
            />
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              @click="handleQuery"
              :loading="querying"
              :disabled="!queryForm.chatId"
            >
              查询
            </el-button>
          </el-form-item>
        </el-form>

        <!-- 查询结果 -->
        <div v-if="queryResult" class="query-result">
          <el-descriptions
            title="群组信息"
            :column="2"
            border
          >
            <el-descriptions-item label="群组 ID">
              {{ queryResult.chat_id }}
            </el-descriptions-item>
            <el-descriptions-item label="群组名称">
              {{ queryResult.name }}
            </el-descriptions-item>
            <el-descriptions-item label="描述" :span="2">
              {{ queryResult.description || '无' }}
            </el-descriptions-item>
            <el-descriptions-item label="所有者 ID">
              {{ queryResult.owner_id }}
            </el-descriptions-item>
            <el-descriptions-item label="是否外部群组">
              <el-tag :type="queryResult.external ? 'warning' : 'success'">
                {{ queryResult.external ? '是' : '否' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </div>
      </div>
    </div>

    <!-- 所有群组列表 -->
    <div class="card-container" style="margin-top: 20px">
      <h2>所有群组</h2>
      <el-divider />

      <div class="table-operations">
        <el-input
          v-model="searchText"
          placeholder="搜索群组名称或 ID"
          style="width: 300px"
          clearable
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button
          type="primary"
          @click="loadGroups"
          :loading="loading"
          :icon="Refresh"
        >
          刷新
        </el-button>
      </div>

      <el-table
        :data="filteredGroups"
        style="width: 100%; margin-top: 16px"
        v-loading="loading"
        border
        stripe
      >
        <el-table-column
          prop="chat_id"
          label="Chat ID"
          width="300"
        >
          <template #default="{ row }">
            <el-tooltip :content="row.chat_id" placement="top">
              <span class="chat-id">{{ row.chat_id }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column
          prop="name"
          label="群组名称"
          min-width="200"
        />
        <el-table-column
          prop="description"
          label="描述"
          min-width="250"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            {{ row.description || '-' }}
          </template>
        </el-table-column>
        <el-table-column
          prop="external"
          label="类型"
          width="100"
          align="center"
        >
          <template #default="{ row }">
            <el-tag :type="row.external ? 'warning' : 'success'" size="small">
              {{ row.external ? '外部' : '内部' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column
          label="操作"
          width="120"
          align="center"
        >
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              link
              @click="copyToClipboard(row.chat_id)"
            >
              复制 ID
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredGroups.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import { useFeishuStore } from '@/store/modules/feishu'

const feishuStore = useFeishuStore()

const queryForm = ref({
  chatId: ''
})

const queryResult = ref(null)
const querying = ref(false)
const loading = ref(false)
const searchText = ref('')
const currentPage = ref(1)
const pageSize = ref(20)

// Filtered groups based on search
const filteredGroups = computed(() => {
  if (!searchText.value) {
    return feishuStore.groups
  }
  const search = searchText.value.toLowerCase()
  return feishuStore.groups.filter(group => 
    group.name.toLowerCase().includes(search) ||
    group.chat_id.toLowerCase().includes(search) ||
    (group.description && group.description.toLowerCase().includes(search))
  )
})

// Paginated groups
const paginatedGroups = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredGroups.value.slice(start, end)
})

// Query specific group
const handleQuery = async () => {
  if (!queryForm.value.chatId) {
    ElMessage.warning('请输入 Chat ID')
    return
  }

  querying.value = true
  queryResult.value = null
  
  try {
    const result = await feishuStore.queryGroup(queryForm.value.chatId)
    queryResult.value = result
    ElMessage.success('查询成功')
  } catch (error) {
    ElMessage.error(feishuStore.error || '查询失败')
  } finally {
    querying.value = false
  }
}

// Load all groups
const loadGroups = async () => {
  loading.value = true
  try {
    await feishuStore.fetchGroups()
    ElMessage.success(`加载成功，共 ${feishuStore.groups.length} 个群组`)
  } catch (error) {
    ElMessage.error(feishuStore.error || '加载群组失败')
  } finally {
    loading.value = false
  }
}

// Search handler
const handleSearch = () => {
  currentPage.value = 1
}

// Pagination handlers
const handleSizeChange = () => {
  currentPage.value = 1
}

const handleCurrentChange = () => {
  // Page changed
}

// Copy to clipboard
const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    // Fallback for older browsers
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand('copy')
      ElMessage.success('已复制到剪贴板')
    } catch (err) {
      ElMessage.error('复制失败')
    }
    document.body.removeChild(textarea)
  }
}

onMounted(() => {
  // Check if Feishu is enabled
  feishuStore.fetchSettings().then(() => {
    if (!feishuStore.isEnabled) {
      ElMessage.warning('飞书未启用，请先在系统配置中启用')
      return
    }
    loadGroups()
  }).catch(() => {
    ElMessage.error('无法加载飞书配置')
  })
})
</script>

<style scoped>
.query-section {
  padding: 20px 0;
}

.query-form {
  margin-bottom: 20px;
}

.query-result {
  margin-top: 20px;
}

.table-operations {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chat-id {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #606266;
  cursor: pointer;
}

.chat-id:hover {
  color: #409eff;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>

