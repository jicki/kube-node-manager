<template>
  <div class="workflow-editor">
    <div class="header">
      <h2>{{ isEdit ? '编辑工作流' : '创建工作流' }}</h2>
      <div class="actions">
        <el-button @click="handleCancel">取消</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          保存
        </el-button>
      </div>
    </div>

    <div class="content">
      <el-form :model="form" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="工作流名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入工作流名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="请输入工作流描述"
          />
        </el-form-item>
      </el-form>

      <div class="dag-section">
        <h3>工作流 DAG 设计</h3>
        <WorkflowDAGEditor
          v-model="form.dag"
          :inventories="inventories"
          @save="handleSave"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { createWorkflow, getWorkflow, updateWorkflow } from '@/api/workflow'
import { listInventories } from '@/api/ansible'
import WorkflowDAGEditor from '@/components/WorkflowDAGEditor.vue'

const route = useRoute()
const router = useRouter()

const isEdit = ref(false)
const workflowId = ref(null)
const saving = ref(false)
const formRef = ref(null)
const inventories = ref([])

const form = reactive({
  name: '',
  description: '',
  dag: {
    nodes: [],
    edges: []
  }
})

const rules = {
  name: [
    { required: true, message: '请输入工作流名称', trigger: 'blur' },
    { min: 2, max: 100, message: '长度在 2 到 100 个字符', trigger: 'blur' }
  ]
}

// 加载主机清单列表
const loadInventories = async () => {
  try {
    const response = await listInventories({ page: 1, page_size: 100 })
    inventories.value = response.data.data || []
  } catch (error) {
    console.error('Failed to load inventories:', error)
  }
}

// 加载工作流详情
const loadWorkflow = async () => {
  try {
    const response = await getWorkflow(workflowId.value)
    const workflow = response.data
    form.name = workflow.name
    form.description = workflow.description
    form.dag = workflow.dag || { nodes: [], edges: [] }
  } catch (error) {
    console.error('Failed to load workflow:', error)
    ElMessage.error(error.response?.data?.error || '加载工作流失败')
    router.back()
  }
}

// 保存工作流
const handleSave = async () => {
  try {
    await formRef.value.validate()

    // 验证 DAG
    if (!form.dag.nodes || form.dag.nodes.length < 2) {
      ElMessage.warning('工作流至少需要一个开始节点和一个结束节点')
      return
    }

    saving.value = true

    if (isEdit.value) {
      await updateWorkflow(workflowId.value, form)
      ElMessage.success('工作流更新成功')
    } else {
      await createWorkflow(form)
      ElMessage.success('工作流创建成功')
    }

    router.push('/ansible/workflows')
  } catch (error) {
    if (error.response) {
      console.error('Failed to save workflow:', error)
      ElMessage.error(error.response?.data?.error || '保存工作流失败')
    }
  } finally {
    saving.value = false
  }
}

// 取消
const handleCancel = () => {
  router.back()
}

// 初始化
onMounted(async () => {
  await loadInventories()

  // 检查是否为编辑模式
  if (route.params.id) {
    isEdit.value = true
    workflowId.value = parseInt(route.params.id)
    await loadWorkflow()
  }
})
</script>

<style scoped>
.workflow-editor {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 20px;
}

.workflow-editor .header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.workflow-editor .header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
}

.workflow-editor .header .actions {
  display: flex;
  gap: 12px;
}

.workflow-editor .content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.workflow-editor .content .el-form {
  background: white;
  padding: 20px;
  border-radius: 4px;
  margin-bottom: 20px;
}

.workflow-editor .content .dag-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: white;
  border-radius: 4px;
  overflow: hidden;
  min-height: 700px;
}

.workflow-editor .content .dag-section h3 {
  padding: 20px 20px 0;
  margin: 0 0 12px 0;
  font-size: 16px;
  font-weight: 600;
}
</style>

