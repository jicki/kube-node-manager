<template>
  <div class="ansible-ssh-keys">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>SSH 密钥管理</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            创建密钥
          </el-button>
        </div>
      </template>

      <!-- 密钥列表 -->
      <el-table :data="sshKeys" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="密钥名称" min-width="200" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="类型" width="120">
          <template #default="{ row }">
            <el-tag :type="row.type === 'private_key' ? 'success' : 'warning'">
              {{ row.type === 'private_key' ? '私钥' : '密码' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="username" label="用户名" width="150" />
        <el-table-column prop="port" label="端口" width="100" />
        <el-table-column label="默认" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.is_default" type="success" size="small">是</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleView(row)">查看</el-button>
            <el-button size="small" type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="success" @click="handleTest(row)">测试</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="queryParams.page"
        v-model:page-size="queryParams.page_size"
        :page-sizes="[10, 20, 50]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="loadSSHKeys"
        @current-change="loadSSHKeys"
        style="margin-top: 20px"
      />
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="dialogTitle" 
      width="60%"
      :close-on-click-modal="false"
    >
      <el-form :model="sshKeyForm" label-width="120px">
        <el-form-item label="密钥名称" :required="!isViewMode">
          <el-input v-model="sshKeyForm.name" placeholder="请输入密钥名称" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="sshKeyForm.description" type="textarea" :rows="3" placeholder="请输入描述" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item label="认证类型" :required="!isViewMode">
          <el-radio-group v-model="sshKeyForm.type" :disabled="isViewMode">
            <el-radio value="private_key">私钥</el-radio>
            <el-radio value="password">密码</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="用户名" :required="!isViewMode">
          <el-input v-model="sshKeyForm.username" placeholder="SSH 用户名，如 root" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item label="SSH 端口">
          <el-input-number v-model="sshKeyForm.port" :min="1" :max="65535" :disabled="isViewMode" />
        </el-form-item>
        <el-form-item v-if="sshKeyForm.type === 'private_key'" label="私钥内容" :required="!isViewMode">
          <el-input 
            v-model="sshKeyForm.private_key" 
            type="textarea" 
            :rows="10"
            placeholder="请粘贴 SSH 私钥内容&#10;&#10;示例：&#10;-----BEGIN RSA PRIVATE KEY-----&#10;MIIEpAIBAAKCAQEA...&#10;-----END RSA PRIVATE KEY-----" 
            style="font-family: 'Courier New', monospace; font-size: 13px;"
            :disabled="isViewMode"
            :show-password="!isViewMode"
          />
        </el-form-item>
        <el-form-item v-if="sshKeyForm.type === 'private_key'" label="私钥密码">
          <el-input 
            v-model="sshKeyForm.passphrase" 
            type="password" 
            placeholder="如果私钥有密码保护，请输入" 
            :disabled="isViewMode"
            show-password
          />
        </el-form-item>
        <el-form-item v-if="sshKeyForm.type === 'password'" label="SSH 密码" :required="!isViewMode">
          <el-input 
            v-model="sshKeyForm.password" 
            type="password" 
            placeholder="请输入 SSH 密码" 
            :disabled="isViewMode"
            show-password
          />
        </el-form-item>
        <el-form-item label="设为默认">
          <el-switch v-model="sshKeyForm.is_default" :disabled="isViewMode" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">{{ isViewMode ? '关闭' : '取消' }}</el-button>
          <el-button v-if="!isViewMode" type="primary" @click="handleSave" :loading="saving">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 测试连接对话框 -->
    <el-dialog v-model="testDialogVisible" title="测试 SSH 连接" width="500px">
      <el-form label-width="120px">
        <el-form-item label="目标主机">
          <el-input v-model="testHost" placeholder="请输入目标主机 IP 或域名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="testDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="performTest" :loading="testing">开始测试</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import * as ansibleAPI from '@/api/ansible'

const sshKeys = ref([])
const total = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const testDialogVisible = ref(false)
const dialogTitle = ref('')
const saving = ref(false)
const testing = ref(false)
const isViewMode = ref(false)
const testHost = ref('')
const currentTestKeyId = ref(null)

const queryParams = reactive({
  page: 1,
  page_size: 20,
  keyword: '',
  type: ''
})

const sshKeyForm = reactive({
  id: null,
  name: '',
  description: '',
  type: 'private_key',
  username: 'root',
  port: 22,
  private_key: '',
  passphrase: '',
  password: '',
  is_default: false
})

const loadSSHKeys = async () => {
  loading.value = true
  try {
    const res = await ansibleAPI.listSSHKeys(queryParams)
    console.log('SSH密钥列表响应:', res)
    sshKeys.value = res.data?.data || []
    total.value = res.data?.total || 0
    console.log('已加载SSH密钥:', sshKeys.value.length, '个')
  } catch (error) {
    console.error('加载SSH密钥失败:', error)
    ElMessage.error('加载SSH密钥失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  isViewMode.value = false
  dialogTitle.value = '创建 SSH 密钥'
  Object.assign(sshKeyForm, {
    id: null,
    name: '',
    description: '',
    type: 'private_key',
    username: 'root',
    port: 22,
    private_key: '',
    passphrase: '',
    password: '',
    is_default: false
  })
  dialogVisible.value = true
}

const handleView = async (row) => {
  isViewMode.value = true
  dialogTitle.value = '查看 SSH 密钥'
  try {
    const res = await ansibleAPI.getSSHKey(row.id)
    console.log('SSH密钥详情响应:', res)
    Object.assign(sshKeyForm, res.data?.data || {})
    // 安全起见，不显示敏感信息
    sshKeyForm.private_key = sshKeyForm.private_key ? '******' : ''
    sshKeyForm.passphrase = sshKeyForm.passphrase ? '******' : ''
    sshKeyForm.password = sshKeyForm.password ? '******' : ''
    dialogVisible.value = true
  } catch (error) {
    console.error('加载SSH密钥失败:', error)
    ElMessage.error('加载SSH密钥失败: ' + (error.message || '未知错误'))
  }
}

const handleEdit = async (row) => {
  isViewMode.value = false
  dialogTitle.value = '编辑 SSH 密钥'
  try {
    const res = await ansibleAPI.getSSHKey(row.id)
    console.log('编辑SSH密钥响应:', res)
    Object.assign(sshKeyForm, res.data?.data || {})
    // 清空敏感字段，让用户重新输入（如果需要修改）
    sshKeyForm.private_key = ''
    sshKeyForm.passphrase = ''
    sshKeyForm.password = ''
    dialogVisible.value = true
  } catch (error) {
    console.error('加载SSH密钥失败:', error)
    ElMessage.error('加载SSH密钥失败: ' + (error.message || '未知错误'))
  }
}

const handleSave = async () => {
  // 验证必填字段
  if (!sshKeyForm.name || !sshKeyForm.username) {
    ElMessage.warning('请填写必填项')
    return
  }

  if (sshKeyForm.type === 'private_key' && !sshKeyForm.private_key && !sshKeyForm.id) {
    ElMessage.warning('请输入私钥内容')
    return
  }

  if (sshKeyForm.type === 'password' && !sshKeyForm.password && !sshKeyForm.id) {
    ElMessage.warning('请输入 SSH 密码')
    return
  }

  saving.value = true
  try {
    const data = { ...sshKeyForm }
    
    if (sshKeyForm.id) {
      // 更新：如果敏感字段为空，不发送
      if (!data.private_key) delete data.private_key
      if (!data.passphrase) delete data.passphrase
      if (!data.password) delete data.password
      
      await ansibleAPI.updateSSHKey(sshKeyForm.id, data)
      ElMessage.success('SSH密钥已更新')
    } else {
      // 创建
      await ansibleAPI.createSSHKey(data)
      ElMessage.success('SSH密钥已创建')
    }
    
    dialogVisible.value = false
    loadSSHKeys()
  } catch (error) {
    console.error('保存SSH密钥失败:', error)
    const errorMsg = error.message || error.toString()
    if (errorMsg.includes('duplicate key') || errorMsg.includes('name already exists')) {
      ElMessage.error('保存失败：SSH密钥名称已存在，请使用其他名称')
    } else if (errorMsg.includes('invalid private key')) {
      ElMessage.error('保存失败：私钥格式不正确，请检查私钥内容')
    } else if (errorMsg.includes('encryption failed')) {
      ElMessage.error('保存失败：加密失败，请联系管理员')
    } else {
      ElMessage.error('保存失败: ' + (errorMsg || '未知错误'))
    }
  } finally {
    saving.value = false
  }
}

const handleTest = (row) => {
  currentTestKeyId.value = row.id
  testHost.value = ''
  testDialogVisible.value = true
}

const performTest = async () => {
  if (!testHost.value) {
    ElMessage.warning('请输入目标主机')
    return
  }

  testing.value = true
  try {
    await ansibleAPI.testSSHConnection(currentTestKeyId.value, {
      host: testHost.value
    })
    ElMessage.success('SSH 连接测试成功！')
    testDialogVisible.value = false
  } catch (error) {
    console.error('SSH连接测试失败:', error)
    ElMessage.error('连接测试失败: ' + (error.message || '未知错误'))
  } finally {
    testing.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定要删除 SSH 密钥 "${row.name}" 吗？`, '提示', {
      type: 'warning'
    })
    await ansibleAPI.deleteSSHKey(row.id)
    ElMessage.success('SSH密钥已删除')
    loadSSHKeys()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除SSH密钥失败:', error)
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadSSHKeys()
})
</script>

<style scoped>
.ansible-ssh-keys {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>

