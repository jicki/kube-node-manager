<template>
  <div class="scripts-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>脚本管理</span>
          <el-button v-if="hasPermission('admin')" type="primary">
            <el-icon><Plus /></el-icon>
            创建脚本
          </el-button>
        </div>
      </template>

      <el-table v-loading="loading" :data="scripts" style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="200" />
        <el-table-column prop="description" label="描述" min-width="250" show-overflow-tooltip />
        <el-table-column prop="language" label="语言" width="100">
          <template #default="{ row }">
            <el-tag>{{ row.language }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="category" label="分类" width="120" />
        <el-table-column prop="version" label="版本" width="80" />
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
        @size-change="fetchScripts"
        @current-change="fetchScripts"
        style="margin-top: 20px; justify-content: flex-end"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { listScripts } from '@/api/script'
import { useAuthStore } from '@/store/modules/auth'

const authStore = useAuthStore()
const hasPermission = (permission) => authStore.hasPermission(permission)

const loading = ref(false)
const scripts = ref([])
const pagination = reactive({ page: 1, size: 20, total: 0 })

const fetchScripts = async () => {
  loading.value = true
  try {
    const res = await listScripts({ page: pagination.page, size: pagination.size })
    scripts.value = res.data || []
    pagination.total = res.total || 0
  } catch (error) {
    ElMessage.error('获取脚本列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchScripts()
})
</script>

<style scoped>
.scripts-container {
  padding: 20px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

