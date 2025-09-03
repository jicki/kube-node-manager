<template>
  <div class="audit-log-page">
    <div class="page-header">
      <div class="header-title">
        <h2>审计日志</h2>
        <span class="sub-title">查看系统操作记录和用户活动日志</span>
      </div>
    </div>

    <div class="page-content">
      <!-- 搜索和筛选 -->
      <el-card class="filter-card" shadow="never">
        <el-form :model="searchForm" :inline="true" class="search-form">
          <el-form-item label="用户名">
            <el-input
              v-model="searchForm.username"
              placeholder="请输入用户名"
              clearable
              style="width: 200px"
            />
          </el-form-item>
          <el-form-item label="操作类型">
            <el-select 
              v-model="searchForm.action"
              placeholder="请选择操作类型"
              clearable
              style="width: 180px"
            >
              <el-option label="全部" value="" />
              <el-option label="CREATE" value="CREATE" />
              <el-option label="UPDATE" value="UPDATE" />
              <el-option label="DELETE" value="DELETE" />
              <el-option label="LIST" value="LIST" />
              <el-option label="LOGIN" value="LOGIN" />
            </el-select>
          </el-form-item>
          <el-form-item label="资源类型">
            <el-select 
              v-model="searchForm.resource_type"
              placeholder="请选择资源类型"
              clearable
              style="width: 180px"
            >
              <el-option label="全部" value="" />
              <el-option label="Node" value="Node" />
              <el-option label="Label" value="Label" />
              <el-option label="Taint" value="Taint" />
              <el-option label="User" value="User" />
              <el-option label="Cluster" value="Cluster" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select 
              v-model="searchForm.status"
              placeholder="请选择状态"
              clearable
              style="width: 120px"
            >
              <el-option label="全部" value="" />
              <el-option label="成功" value="success" />
              <el-option label="失败" value="failure" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch" :loading="loading">
              <el-icon><Search /></el-icon>
              搜索
            </el-button>
            <el-button @click="handleReset">
              <el-icon><Refresh /></el-icon>
              重置
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 审计日志表格 -->
      <el-card shadow="never">
        <el-table
          :data="auditLogs"
          v-loading="loading"
          style="width: 100%"
          stripe
          :default-sort="{ prop: 'created_at', order: 'descending' }"
        >
          <el-table-column prop="id" label="ID" width="80" />
          
          <el-table-column prop="username" label="用户" width="120">
            <template #default="{ row }">
              <div class="user-info">
                <el-avatar :size="24" class="user-avatar">
                  {{ (row.user?.username || 'U').charAt(0).toUpperCase() }}
                </el-avatar>
                <span>{{ row.user?.username || 'Unknown' }}</span>
              </div>
            </template>
          </el-table-column>
          
          <el-table-column prop="action" label="操作" width="100">
            <template #default="{ row }">
              <el-tag 
                :type="getActionTagType(row.action)" 
                size="small"
                :icon="getActionIcon(row.action)"
              >
                {{ row.action }}
              </el-tag>
            </template>
          </el-table-column>
          
          <el-table-column prop="resource_type" label="资源类型" width="100">
            <template #default="{ row }">
              <el-tag type="info" size="small">
                {{ row.resource_type }}
              </el-tag>
            </template>
          </el-table-column>
          
          <el-table-column prop="node_name" label="节点" width="200">
            <template #default="{ row }">
              <span v-if="row.node_name" class="node-name">
                {{ row.node_name }}
              </span>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>
          
          <el-table-column prop="details" label="详情" min-width="300">
            <template #default="{ row }">
              <div class="details-content">
                {{ row.details || '-' }}
              </div>
            </template>
          </el-table-column>
          
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag 
                :type="row.status === 'success' ? 'success' : 'danger'" 
                size="small"
              >
                {{ row.status === 'success' ? '成功' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          
          <el-table-column prop="ip_address" label="IP地址" width="140">
            <template #default="{ row }">
              <span class="ip-address">{{ row.ip_address || '-' }}</span>
            </template>
          </el-table-column>
          
          <el-table-column prop="created_at" label="时间" width="180">
            <template #default="{ row }">
              <div class="time-info">
                <div>{{ formatTime(row.created_at) }}</div>
                <div class="time-ago">{{ formatRelativeTime(row.created_at) }}</div>
              </div>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <div class="pagination-wrapper">
          <el-pagination
            v-model:current-page="pagination.current"
            v-model:page-size="pagination.size"
            :page-sizes="[20, 50, 100, 200]"
            :total="pagination.total"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
          />
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Refresh, Plus, Edit, Delete, User, List, Document } from '@element-plus/icons-vue'
import auditApi from '@/api/audit'
import { formatTime, formatRelativeTime } from '@/utils/format'

// 响应式数据
const loading = ref(false)
const auditLogs = ref([])

// 搜索表单
const searchForm = reactive({
  username: '',
  action: '',
  resource_type: '',
  status: ''
})

// 分页
const pagination = reactive({
  current: 1,
  size: 20,
  total: 0
})

// 获取审计日志
const fetchAuditLogs = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.current,
      page_size: pagination.size,
      ...searchForm
    }
    
    // 清除空值参数
    Object.keys(params).forEach(key => {
      if (params[key] === '' || params[key] === null || params[key] === undefined) {
        delete params[key]
      }
    })
    
    const response = await auditApi.getAuditLogs(params)
    if (response.data && response.data.data) {
      auditLogs.value = response.data.data.logs || []
      pagination.total = response.data.data.total || 0
    }
  } catch (error) {
    console.error('获取审计日志失败:', error)
    ElMessage.error('获取审计日志失败')
    auditLogs.value = []
    pagination.total = 0
  } finally {
    loading.value = false
  }
}

// 搜索处理
const handleSearch = () => {
  pagination.current = 1
  fetchAuditLogs()
}

// 重置搜索
const handleReset = () => {
  Object.keys(searchForm).forEach(key => {
    searchForm[key] = ''
  })
  pagination.current = 1
  fetchAuditLogs()
}

// 分页处理
const handleSizeChange = (size) => {
  pagination.size = size
  pagination.current = 1
  fetchAuditLogs()
}

const handleCurrentChange = (current) => {
  pagination.current = current
  fetchAuditLogs()
}

// 获取操作类型标签样式
const getActionTagType = (action) => {
  switch (action) {
    case 'CREATE': return 'success'
    case 'UPDATE': return 'warning'
    case 'DELETE': return 'danger'
    case 'LOGIN': return 'primary'
    case 'LIST': return 'info'
    default: return 'info'
  }
}

// 获取操作图标
const getActionIcon = (action) => {
  switch (action) {
    case 'CREATE': return Plus
    case 'UPDATE': return Edit
    case 'DELETE': return Delete
    case 'LOGIN': return User
    case 'LIST': return List
    default: return Document
  }
}

onMounted(() => {
  fetchAuditLogs()
})
</script>

<style scoped>
.audit-log-page {
  padding: 24px;
  background: #f5f5f5;
  min-height: 100vh;
}

.page-header {
  margin-bottom: 24px;
}

.header-title h2 {
  margin: 0 0 4px 0;
  color: #333;
  font-size: 24px;
  font-weight: 600;
}

.sub-title {
  color: #666;
  font-size: 14px;
}

.page-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-card {
  border: none;
}

.search-form {
  margin: 0;
}

.search-form .el-form-item {
  margin-bottom: 0;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  font-size: 12px;
  font-weight: 600;
}

.node-name {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #1890ff;
}

.details-content {
  line-height: 1.4;
  color: #333;
}

.ip-address {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #52c41a;
}

.time-info {
  line-height: 1.3;
}

.time-ago {
  font-size: 11px;
  color: #999;
}

.text-muted {
  color: #999;
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .audit-log-page {
    padding: 16px;
  }
  
  .search-form {
    flex-direction: column;
  }
  
  .search-form .el-form-item {
    margin-right: 0;
    margin-bottom: 16px;
  }
}
</style>
