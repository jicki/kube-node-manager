<template>
  <div class="workflows-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>工作流管理</span>
          <el-button v-if="hasPermission('admin')" type="primary">
            <el-icon><Plus /></el-icon>
            创建工作流
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="workflows" style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="200" />
        <el-table-column prop="description" label="描述" min-width="250" show-overflow-tooltip />
        <el-table-column prop="category" label="分类" width="120">
          <template #default="{ row }">
            <el-tag>{{ row.category }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="version" label="版本" width="80" />
        <el-table-column prop="is_builtin" label="类型" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.is_builtin" type="info">内置</el-tag>
            <el-tag v-else type="success">自定义</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small">查看</el-button>
            <el-button size="small" type="primary">执行</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.size"
        :total="pagination.total"
        layout="total, sizes, prev, pager, next"
        @size-change="fetchWorkflows"
        @current-change="fetchWorkflows"
        style="margin-top: 20px; justify-content: flex-end"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { listWorkflows } from '@/api/workflow'
import { useAuthStore } from '@/store/modules/auth'

const authStore = useAuthStore()
const hasPermission = (permission) => authStore.hasPermission(permission)

const loading = ref(false)
const workflows = ref([])
const pagination = reactive({ page: 1, size: 20, total: 0 })

const fetchWorkflows = async () => {
  loading.value = true
  try {
    const res = await listWorkflows({ page: pagination.page, size: pagination.size })
    workflows.value = res.data || []
    pagination.total = res.total || 0
  } catch (error) {
    ElMessage.error('获取工作流列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchWorkflows()
})
</script>

<style scoped>
.workflows-container {
  padding: 20px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

