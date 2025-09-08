<template>
  <el-dialog
    v-model="visible"
    title="节点详情"
    width="800px"
    :before-close="handleClose"
  >
    <div v-if="node" class="node-detail">
      <!-- 基本信息 -->
      <el-descriptions title="基本信息" :column="2" border>
        <el-descriptions-item label="节点名称">
          {{ node.name }}
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusType(node.status)">
            {{ formatNodeStatus(node.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="角色">
          {{ formatNodeRoles(node.roles) }}
        </el-descriptions-item>
        <el-descriptions-item label="版本">
          {{ node.version }}
        </el-descriptions-item>
        <el-descriptions-item label="操作系统">
          {{ node.osImage }}
        </el-descriptions-item>
        <el-descriptions-item label="容器运行时">
          {{ node.containerRuntime }}
        </el-descriptions-item>
        <el-descriptions-item label="内核版本">
          {{ node.kernelVersion }}
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">
          {{ formatTime(node.createdAt) }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 资源信息 -->
      <el-descriptions title="资源信息" :column="2" border style="margin-top: 20px;">
        <el-descriptions-item label="CPU">
          {{ formatCPU(node.cpu) }}
        </el-descriptions-item>
        <el-descriptions-item label="内存">
          {{ formatMemory(node.memory) }}
        </el-descriptions-item>
        <el-descriptions-item label="存储">
          {{ formatMemory(node.storage) }}
        </el-descriptions-item>
        <el-descriptions-item label="Pod数量">
          {{ node.pods }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 地址信息 -->
      <el-descriptions title="地址信息" :column="1" border style="margin-top: 20px;" v-if="node.addresses">
        <el-descriptions-item v-for="address in node.addresses" :key="address.type" :label="address.type">
          {{ address.address }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 条件信息 -->
      <div style="margin-top: 20px;">
        <h4>节点条件</h4>
        <el-table :data="node.conditions" style="width: 100%" v-if="node.conditions">
          <el-table-column prop="type" label="类型" width="150" />
          <el-table-column prop="status" label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.status === 'True' ? 'success' : 'danger'">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" width="120" />
          <el-table-column prop="message" label="信息" />
          <el-table-column prop="lastTransitionTime" label="最后变更时间" width="160">
            <template #default="{ row }">
              {{ formatTime(row.lastTransitionTime) }}
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 标签信息 -->
      <div class="labels-section" v-if="node.labels">
        <h4>标签</h4>
        <div class="labels-container">
          <el-tooltip
            v-for="(value, key) in node.labels"
            :key="key"
            :content="`${key}=${value}`"
            placement="top"
            :disabled="isLabelShort(key, value)"
          >
            <el-tag 
              class="label-tag"
              size="small"
            >
              {{ key }}: {{ value }}
            </el-tag>
          </el-tooltip>
        </div>
      </div>

      <!-- 污点信息 -->
      <div style="margin-top: 20px;" v-if="node.taints && node.taints.length > 0">
        <h4>污点</h4>
        <el-table :data="node.taints" style="width: 100%">
          <el-table-column prop="key" label="键" width="200" />
          <el-table-column prop="value" label="值" width="150" />
          <el-table-column prop="effect" label="效果" width="120" />
        </el-table>
      </div>
    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button type="primary" @click="handleRefresh">刷新</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed } from 'vue'
import { formatTime, formatNodeStatus, formatNodeRoles, formatCPU, formatMemory } from '@/utils/format'

// 判断标签是否较短，不需要tooltip
const isLabelShort = (key, value) => {
  const fullText = `${key}=${value}`
  return fullText.length <= 50 // 50个字符以内认为是短标签
}

// Props
const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  node: {
    type: Object,
    default: null
  }
})

// Emits
const emit = defineEmits(['update:modelValue', 'refresh'])

// 计算属性
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 方法
const getStatusType = (status) => {
  switch (status) {
    case 'Ready':
      return 'success'
    case 'NotReady':
      return 'danger'
    case 'SchedulingDisabled':
      return 'warning'
    default:
      return 'info'
  }
}

const handleClose = () => {
  visible.value = false
}

const handleRefresh = () => {
  emit('refresh')
  visible.value = false
}
</script>

<style scoped>
.node-detail {
  padding: 0 10px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

h4 {
  margin: 10px 0;
  color: #606266;
  font-weight: 600;
}

.labels-section {
  margin-top: 20px;
}

.labels-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
  line-height: 1.6;
}

.label-tag {
  margin: 0;
  font-size: 12px;
  min-height: 24px;
  line-height: 22px;
  padding: 0 10px;
  border-radius: 12px;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  background: #f0f9ff;
  color: #0958d9;
  border: 1px solid #91d5ff;
  max-width: 100%;
  word-break: break-all;
  white-space: normal;
  text-align: left;
}

/* 当标签内容较长时，允许换行显示 */
.label-tag {
  overflow-wrap: break-word;
  hyphens: auto;
}
</style>
