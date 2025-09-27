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
          {{ node.os_image || node.osImage || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="容器运行时">
          {{ node.container_runtime || node.containerRuntime || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="内核版本">
          {{ node.kernel_version || node.kernelVersion || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">
          {{ formatTime(node.created_at || node.createdAt) }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 资源信息 -->
      <el-descriptions title="资源信息" :column="2" border style="margin-top: 20px;">
        <el-descriptions-item label="CPU">
          <div class="resource-detail">
            <div class="resource-line">
              <span class="resource-type">总量：</span>
              <span class="resource-value-total">{{ formatCPU(node.capacity?.cpu) || 'N/A' }}</span>
            </div>
            <div class="resource-line">
              <span class="resource-type">可分配：</span>
              <span class="resource-value-allocatable">{{ formatCPU(node.allocatable?.cpu) || 'N/A' }}</span>
            </div>
          </div>
        </el-descriptions-item>
        <el-descriptions-item label="内存">
          <div class="resource-detail">
            <div class="resource-line">
              <span class="resource-type">总量：</span>
              <span class="resource-value-total">{{ formatMemoryCorrect(node.capacity?.memory) || 'N/A' }}</span>
            </div>
            <div class="resource-line">
              <span class="resource-type">可分配：</span>
              <span class="resource-value-allocatable">{{ formatMemoryCorrect(node.allocatable?.memory) || 'N/A' }}</span>
            </div>
          </div>
        </el-descriptions-item>
        <el-descriptions-item label="存储">
          <div class="resource-detail">
            <div class="resource-line">
              <span class="resource-type">总量：</span>
              <span class="resource-value-total">{{ formatMemoryCorrect(node.capacity?.['ephemeral-storage']) || 'N/A' }}</span>
            </div>
            <div class="resource-line">
              <span class="resource-type">可分配：</span>
              <span class="resource-value-allocatable">{{ formatMemoryCorrect(node.allocatable?.['ephemeral-storage']) || 'N/A' }}</span>
            </div>
          </div>
        </el-descriptions-item>
        <el-descriptions-item label="Pod数量">
          <div class="resource-detail">
            <div class="resource-line">
              <span class="resource-type">总量：</span>
              <span class="resource-value-total">{{ node.capacity?.pods || '0' }}</span>
            </div>
            <div class="resource-line">
              <span class="resource-type">可分配：</span>
              <span class="resource-value-allocatable">{{ node.allocatable?.pods || '0' }}</span>
            </div>
          </div>
        </el-descriptions-item>
      </el-descriptions>

      <!-- 地址信息 -->
      <el-descriptions title="地址信息" :column="1" border style="margin-top: 20px;" v-if="node.addresses && node.addresses.length > 0">
        <el-descriptions-item 
          v-for="address in node.addresses" 
          :key="address.type" 
          :label="formatAddressType(address.type)"
        >
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

// 正确的内存格式化函数，处理Kubernetes内存格式
const formatMemoryCorrect = (value) => {
  if (!value) return 'N/A'
  
  // 解析Kubernetes内存格式（例如：3906252Ki, 4Gi等）
  const memStr = String(value).trim()
  
  // 匹配数字和单位
  const match = memStr.match(/^(\d+(?:\.\d+)?)(.*)?$/)
  if (!match) return value
  
  const [, numStr, unit = ''] = match
  const num = parseFloat(numStr)
  
  // 单位转换表（字节）
  const unitMap = {
    'Ki': 1024,
    'Mi': 1024 * 1024,  
    'Gi': 1024 * 1024 * 1024,
    'Ti': 1024 * 1024 * 1024 * 1024,
    'K': 1000,
    'M': 1000 * 1000,
    'G': 1000 * 1000 * 1000,
    'T': 1000 * 1000 * 1000 * 1000,
    '': 1 // 如果没有单位，假设为字节
  }
  
  const multiplier = unitMap[unit] || 1
  const bytes = num * multiplier
  
  // 转换为适合显示的单位
  if (bytes >= 1024 * 1024 * 1024 * 1024) {
    return (bytes / (1024 * 1024 * 1024 * 1024)).toFixed(1) + ' Ti'
  } else if (bytes >= 1024 * 1024 * 1024) {
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' Gi'
  } else if (bytes >= 1024 * 1024) {
    return (bytes / (1024 * 1024)).toFixed(1) + ' Mi'
  } else if (bytes >= 1024) {
    return (bytes / 1024).toFixed(1) + ' Ki'
  } else {
    return bytes + ' B'
  }
}

// 格式化地址类型
const formatAddressType = (type) => {
  const typeMap = {
    'InternalIP': '内网IP',
    'ExternalIP': '外网IP',
    'Hostname': '主机名',
    'InternalDNS': '内网DNS',
    'ExternalDNS': '外网DNS'
  }
  return typeMap[type] || type
}

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

/* 资源详情样式 */
.resource-detail {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-type {
  font-size: 12px;
  color: #666;
  min-width: 50px;
  font-weight: 500;
}

.resource-value-total {
  color: #1890ff;
  font-weight: 600;
  font-family: 'Monaco', 'Consolas', monospace;
  font-size: 13px;
}

.resource-value-allocatable {
  color: #52c41a;
  font-weight: 600;
  font-family: 'Monaco', 'Consolas', monospace;
  font-size: 13px;
}
</style>
