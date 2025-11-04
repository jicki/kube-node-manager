<template>
  <div class="workflow-list">
    <div class="header">
      <h2>Ansible 工作流管理</h2>
      <div class="actions">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索工作流名称或描述"
          class="search-input"
          prefix-icon="el-icon-search"
          clearable
          @change="loadWorkflows"
        />
        <el-button type="primary" icon="el-icon-plus" @click="handleCreate">
          创建工作流
        </el-button>
      </div>
    </div>

    <el-table :data="workflows" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="工作流名称" min-width="200" />
      <el-table-column prop="description" label="描述" min-width="250" show-overflow-tooltip />
      <el-table-column label="节点数" width="100">
        <template #default="{ row }">
          <el-tag size="small">{{ row.dag?.nodes?.length || 0 }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="边数" width="100">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ row.dag?.edges?.length || 0 }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column prop="updated_at" label="更新时间" width="180">
        <template #default="{ row }">
          {{ formatTime(row.updated_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="success" icon="el-icon-video-play" @click="handleExecute(row)">
            执行
          </el-button>
          <el-button size="small" icon="el-icon-edit" @click="handleEdit(row)">
            编辑
          </el-button>
          <el-button size="small" type="info" icon="el-icon-view" @click="handleView(row)">
            查看
          </el-button>
          <el-button size="small" type="danger" icon="el-icon-delete" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadWorkflows"
        @current-change="loadWorkflows"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listWorkflows, deleteWorkflow, executeWorkflow } from '@/api/workflow'
import { formatTime } from '@/utils/time'

const router = useRouter()
const workflows = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 加载工作流列表
const loadWorkflows = async () => {
  loading.value = true
  try {
    const response = await listWorkflows({
      page: currentPage.value,
      page_size: pageSize.value,
      keyword: searchKeyword.value
    })
    workflows.value = response.workflows || []
    total.value = response.total || 0
  } catch (error) {
    console.error('Failed to load workflows:', error)
    ElMessage.error(error.response?.data?.error || '加载工作流列表失败')
  } finally {
    loading.value = false
  }
}

// 创建工作流
const handleCreate = () => {
  router.push('/ansible/workflows/create')
}

// 编辑工作流
const handleEdit = (workflow) => {
  router.push(`/ansible/workflows/${workflow.id}/edit`)
}

// 查看工作流
const handleView = (workflow) => {
  router.push(`/ansible/workflows/${workflow.id}`)
}

// 删除工作流
const handleDelete = async (workflow) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除工作流"${workflow.name}"吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteWorkflow(workflow.id)
    ElMessage.success('工作流删除成功')
    loadWorkflows()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete workflow:', error)
      ElMessage.error(error.response?.data?.error || '删除工作流失败')
    }
  }
}

// 执行工作流
const handleExecute = async (workflow) => {
  try {
    await ElMessageBox.confirm(
      `确定要执行工作流"${workflow.name}"吗？`,
      '确认执行',
      {
        confirmButtonText: '执行',
        cancelButtonText: '取消',
        type: 'info'
      }
    )

    const response = await executeWorkflow(workflow.id)
    ElMessage.success(`工作流开始执行，执行 ID: ${response.id}`)
    router.push(`/ansible/workflow-executions/${response.id}`)
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to execute workflow:', error)
      ElMessage.error(error.response?.data?.error || '执行工作流失败')
    }
  }
}

onMounted(() => {
  loadWorkflows()
})
</script>

<style scoped lang="scss">
.workflow-list {
  padding: 20px;

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;

    h2 {
      margin: 0;
      font-size: 24px;
      font-weight: 600;
    }

    .actions {
      display: flex;
      gap: 12px;

      .search-input {
        width: 300px;
      }
    }
  }

  .pagination {
    margin-top: 20px;
    display: flex;
    justify-content: flex-end;
  }
}
</style>

