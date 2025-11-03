# UI 改进和新功能实施计划

## 创建时间
2025-01-13

## 问题分析和解决方案

---

## 1️⃣ 任务模板变量验证功能

### 📊 问题分析

**现状**: 
- ✅ 后端已完全实现 `required_vars` 字段
- ✅ 前端代码已实现变量输入UI（第476-504行）
- ❌ 用户反馈：创建任务时看不到模板变量输入框

**根本原因**:
```javascript
// 条件渲染：只有当模板有 required_vars 且不为空时才显示
<template v-if="selectedTemplate && selectedTemplate.required_vars && selectedTemplate.required_vars.length > 0">
```

可能的问题：
1. 数据库中现有模板的 `required_vars` 字段为 `null` 或空数组
2. 模板创建/编辑页面没有提供设置 `required_vars` 的 UI
3. API 响应中没有正确返回 `required_vars` 字段

### ✅ 解决方案

#### Step 1: 为模板管理页面添加必需变量配置UI

**文件**: `frontend/src/views/ansible/TaskTemplates.vue`（或相应的模板管理页面）

需要添加：
```vue
<el-form-item label="必需变量" prop="required_vars">
  <el-tag
    v-for="tag in templateForm.required_vars"
    :key="tag"
    closable
    @close="handleRemoveRequiredVar(tag)"
    style="margin-right: 8px"
  >
    {{ tag }}
  </el-tag>
  <el-input
    v-if="requiredVarInputVisible"
    ref="requiredVarInput"
    v-model="requiredVarInputValue"
    size="small"
    style="width: 120px"
    @keyup.enter="handleAddRequiredVar"
    @blur="handleAddRequiredVar"
  />
  <el-button
    v-else
    size="small"
    @click="showRequiredVarInput"
  >
    + 添加必需变量
  </el-button>
  <div style="margin-top: 8px; color: #909399; font-size: 12px">
    <el-icon><InfoFilled /></el-icon>
    必需变量会在创建任务时要求用户提供，用于参数化 Playbook
  </div>
</el-form-item>
```

#### Step 2: 验证 API 响应

确保模板列表 API 返回 `required_vars` 字段：
```javascript
// frontend/src/views/ansible/TaskCenter.vue
const loadTemplates = async () => {
  const response = await listTemplates()
  console.log('模板数据:', response.data) // 调试：检查是否有 required_vars
  templates.value = response.data.data || []
}
```

#### Step 3: 添加示例数据

为测试目的，在模板管理中添加一个带必需变量的示例模板：
```yaml
# Playbook 示例
- name: Deploy Application
  hosts: all
  vars:
    app_version: "{{ app_version }}"  # 必需变量
    deploy_env: "{{ deploy_env }}"    # 必需变量
  tasks:
    - name: Deploy
      debug:
        msg: "Deploying version {{ app_version }} to {{ deploy_env }}"
```

Required Vars: `["app_version", "deploy_env"]`

---

## 2️⃣ 任务执行前置检查按钮

### 📊 问题分析

**现状**:
- ✅ 后端 API 已实现
- ✅ 前端代码已实现（第301-308行）
- ❌ 用户反馈：看不到"执行检查"按钮

**根本原因**:
```javascript
v-if="row.status === 'pending'"
```

按钮只对状态为 `pending` 的任务显示。如果任务已经开始执行或完成，按钮会隐藏。

### ✅ 解决方案

#### Option 1: 扩展显示条件（推荐）

```vue
<!-- 修改为：pending 或 failed 状态都可以执行检查 -->
<el-button 
  size="small" 
  type="info" 
  @click="handlePreflightCheck(row)" 
  v-if="row.status === 'pending' || row.status === 'failed'"
>
  执行检查
</el-button>
```

#### Option 2: 在任务创建时自动执行前置检查

```javascript
const handleCreate = async () => {
  try {
    creating.value = true
    
    // 1. 创建任务
    const response = await createTask(taskForm)
    const task = response.data.data
    
    // 2. 自动执行前置检查（可选）
    if (autoPreflightCheck.value) {
      try {
        const checkResponse = await executePreflightChecks(task.id)
        if (checkResponse.data.status === 'fail') {
          // 显示检查结果，让用户决定是否继续
          preflightResult.value = checkResponse.data
          preflightDialogVisible.value = true
          return // 暂停，等待用户确认
        }
      } catch (error) {
        console.error('自动前置检查失败:', error)
      }
    }
    
    // 3. 提示成功
    ElMessage.success('任务创建成功')
    createDialogVisible.value = false
    refreshData()
  } catch (error) {
    ElMessage.error('任务创建失败: ' + error.message)
  } finally {
    creating.value = false
  }
}
```

#### Option 3: 添加独立的"检查"Tab

在任务中心添加一个"检查工具"Tab，允许用户在创建任务前进行检查：

```vue
<el-tabs v-model="activeTab">
  <el-tab-pane label="任务列表" name="tasks">
    <!-- 现有任务列表 -->
  </el-tab-pane>
  <el-tab-pane label="检查工具" name="checker">
    <el-card>
      <template #header>前置检查工具</template>
      <el-form>
        <el-form-item label="选择模板">
          <el-select v-model="checkerForm.template_id">
            <el-option
              v-for="tpl in templates"
              :key="tpl.id"
              :label="tpl.name"
              :value="tpl.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="选择清单">
          <el-select v-model="checkerForm.inventory_id">
            <el-option
              v-for="inv in inventories"
              :key="inv.id"
              :label="inv.name"
              :value="inv.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="runQuickCheck">
            快速检查
          </el-button>
        </el-form-item>
      </el-form>
      <!-- 检查结果显示 -->
    </el-card>
  </el-tab-pane>
</el-tabs>
```

---

## 3️⃣ 任务执行可视化前端 UI

### 📊 现状

- ✅ 后端 API 已完全实现
- ✅ 数据自动记录（8个执行阶段）
- ❌ 前端 UI 未实现

### ✅ 实施方案

#### Phase 1: 基础时间线展示（高优先级）

**文件**: `frontend/src/components/ansible/TaskTimelineVisualization.vue`

```vue
<template>
  <div class="task-timeline-visualization">
    <!-- 时间线头部 -->
    <el-card class="header-card">
      <el-row :gutter="20">
        <el-col :span="6">
          <el-statistic title="任务名称" :value="visualization.task_name" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="总耗时" :value="formatDuration(visualization.total_duration)" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="状态" :value="getStatusText(visualization.status)" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="阶段数" :value="visualization.timeline.length" />
        </el-col>
      </el-row>
    </el-card>

    <!-- 时间线 -->
    <el-timeline style="margin-top: 20px">
      <el-timeline-item
        v-for="(event, index) in visualization.timeline"
        :key="index"
        :timestamp="formatTimestamp(event.timestamp)"
        :type="getPhaseType(event.phase)"
        :icon="getPhaseIcon(event.phase)"
        placement="top"
      >
        <el-card shadow="hover">
          <div style="display: flex; justify-content: space-between; align-items: center">
            <div>
              <h4>{{ getPhaseLabel(event.phase) }}</h4>
              <p style="color: #909399; margin: 8px 0">{{ event.message }}</p>
            </div>
            <div style="text-align: right">
              <el-tag v-if="event.duration" type="info">
                耗时: {{ event.duration }}ms
              </el-tag>
              <div v-if="event.batch_number" style="margin-top: 4px">
                <el-tag size="small">批次 {{ event.batch_number }}</el-tag>
              </div>
            </div>
          </div>
          
          <!-- 批次详情 -->
          <div v-if="event.host_count" style="margin-top: 12px">
            <el-divider />
            <el-row :gutter="16">
              <el-col :span="8">
                <span>主机总数: {{ event.host_count }}</span>
              </el-col>
              <el-col :span="8">
                <span style="color: #67C23A">✓ 成功: {{ event.success_count }}</span>
              </el-col>
              <el-col :span="8">
                <span style="color: #F56C6C">✗ 失败: {{ event.fail_count }}</span>
              </el-col>
            </el-row>
          </div>
          
          <!-- 额外详情 -->
          <div v-if="event.details && Object.keys(event.details).length > 0" style="margin-top: 12px">
            <el-divider />
            <el-descriptions :column="2" size="small" border>
              <el-descriptions-item
                v-for="(value, key) in event.details"
                :key="key"
                :label="key"
              >
                {{ value }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-card>
      </el-timeline-item>
    </el-timeline>

    <!-- 阶段耗时分布饼图 -->
    <el-card style="margin-top: 20px">
      <template #header>
        <span>阶段耗时分布</span>
      </template>
      <div ref="chartRef" style="height: 400px"></div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import * as echarts from 'echarts'
import { getTaskVisualization } from '@/api/ansible'

const props = defineProps({
  taskId: {
    type: Number,
    required: true
  }
})

const visualization = ref({})
const chartRef = ref(null)
let chart = null

// 加载可视化数据
const loadVisualization = async () => {
  try {
    const response = await getTaskVisualization(props.taskId)
    visualization.value = response.data.data
    
    // 渲染图表
    renderChart()
  } catch (error) {
    console.error('加载可视化数据失败:', error)
  }
}

// 渲染饼图
const renderChart = () => {
  if (!chartRef.value) return
  
  if (!chart) {
    chart = echarts.init(chartRef.value)
  }
  
  const data = Object.entries(visualization.value.phase_distribution || {}).map(([name, value]) => ({
    name: getPhaseLabel(name),
    value: value
  }))
  
  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c}ms ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        name: '阶段耗时',
        type: 'pie',
        radius: '50%',
        data: data,
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }
  
  chart.setOption(option)
}

// 辅助方法
const getPhaseLabel = (phase) => {
  const labels = {
    'queued': '入队等待',
    'preflight_check': '前置检查',
    'executing': '执行中',
    'batch_paused': '批次暂停',
    'completed': '已完成',
    'failed': '执行失败',
    'cancelled': '已取消',
    'timeout': '执行超时'
  }
  return labels[phase] || phase
}

const getPhaseType = (phase) => {
  const types = {
    'queued': 'info',
    'preflight_check': 'warning',
    'executing': 'primary',
    'batch_paused': 'warning',
    'completed': 'success',
    'failed': 'danger',
    'cancelled': 'info',
    'timeout': 'danger'
  }
  return types[phase] || 'info'
}

const getPhaseIcon = (phase) => {
  // 返回 Element Plus 图标组件
  return null
}

const formatDuration = (ms) => {
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}秒`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}分${remainingSeconds}秒`
}

const formatTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString()
}

const getStatusText = (status) => {
  const texts = {
    'pending': '等待中',
    'running': '运行中',
    'success': '成功',
    'failed': '失败',
    'cancelled': '已取消'
  }
  return texts[status] || status
}

onMounted(() => {
  loadVisualization()
})

watch(() => props.taskId, () => {
  loadVisualization()
})
</script>

<style scoped>
.task-timeline-visualization {
  padding: 20px;
}
.header-card {
  margin-bottom: 20px;
}
</style>
```

#### Phase 2: 在任务详情中集成（中优先级）

**文件**: `frontend/src/views/ansible/TaskCenter.vue`

添加一个"可视化"Tab：

```vue
<!-- 在查看日志对话框中添加 Tab -->
<el-dialog 
  v-model="logDialogVisible" 
  title="任务详情" 
  width="90%"
  :close-on-click-modal="false"
>
  <el-tabs v-model="detailActiveTab">
    <el-tab-pane label="执行日志" name="logs">
      <LogViewer :task-id="currentTaskId" />
    </el-tab-pane>
    <el-tab-pane label="执行可视化" name="visualization">
      <TaskTimelineVisualization :task-id="currentTaskId" />
    </el-tab-pane>
  </el-tabs>
</el-dialog>
```

#### Phase 3: 高级可视化（低优先级）

使用更高级的可视化库（如 G6）：

1. **流程图展示**: 显示任务执行的流程图
2. **甘特图**: 显示各阶段的时间分布
3. **主机热力图**: 显示主机级别的执行状态

---

## 4️⃣ 任务依赖关系（DAG 工作流）开发

### 📊 需求分析

**用户场景**:
1. 部署流程：构建 → 测试 → 部署到测试环境 → 部署到生产环境
2. 数据处理流程：数据采集 → 数据清洗 → 数据分析 → 生成报告
3. 服务编排：启动数据库 → 启动缓存 → 启动应用服务

**核心需求**:
- 任务之间的依赖关系定义
- 自动按依赖顺序执行
- 支持并行执行（无依赖的任务）
- 失败处理策略
- 可视化工作流编辑器

### ✅ 实施方案

#### Phase 1: 数据模型设计（2天）

**数据库迁移**: `backend/migrations/019_add_workflow_dag.sql`

```sql
-- +migrate Up
-- 工作流定义表
CREATE TABLE IF NOT EXISTS ansible_workflows (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    dag JSONB NOT NULL,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    CONSTRAINT fk_ansible_workflows_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_ansible_workflows_user_id ON ansible_workflows(user_id);
CREATE INDEX idx_ansible_workflows_deleted_at ON ansible_workflows(deleted_at);

-- 工作流执行记录表
CREATE TABLE IF NOT EXISTS ansible_workflow_executions (
    id SERIAL PRIMARY KEY,
    workflow_id INTEGER NOT NULL,
    status VARCHAR(50) DEFAULT 'running',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP,
    error_message TEXT,
    user_id INTEGER NOT NULL,
    CONSTRAINT fk_ansible_workflow_executions_workflow FOREIGN KEY (workflow_id) REFERENCES ansible_workflows(id) ON DELETE CASCADE,
    CONSTRAINT fk_ansible_workflow_executions_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_ansible_workflow_executions_workflow_id ON ansible_workflow_executions(workflow_id);
CREATE INDEX idx_ansible_workflow_executions_user_id ON ansible_workflow_executions(user_id);
CREATE INDEX idx_ansible_workflow_executions_status ON ansible_workflow_executions(status);

-- 修改 ansible_tasks 表，添加工作流关联
ALTER TABLE ansible_tasks ADD COLUMN workflow_execution_id INTEGER;
ALTER TABLE ansible_tasks ADD COLUMN depends_on JSONB;
ALTER TABLE ansible_tasks ADD COLUMN node_id VARCHAR(50);

ALTER TABLE ansible_tasks ADD CONSTRAINT fk_ansible_tasks_workflow_execution 
    FOREIGN KEY (workflow_execution_id) REFERENCES ansible_workflow_executions(id) ON DELETE SET NULL;

CREATE INDEX idx_ansible_tasks_workflow_execution_id ON ansible_tasks(workflow_execution_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_ansible_tasks_workflow_execution_id;
ALTER TABLE ansible_tasks DROP CONSTRAINT IF EXISTS fk_ansible_tasks_workflow_execution;
ALTER TABLE ansible_tasks DROP COLUMN node_id;
ALTER TABLE ansible_tasks DROP COLUMN depends_on;
ALTER TABLE ansible_tasks DROP COLUMN workflow_execution_id;

DROP INDEX IF EXISTS idx_ansible_workflow_executions_status;
DROP INDEX IF EXISTS idx_ansible_workflow_executions_user_id;
DROP INDEX IF EXISTS idx_ansible_workflow_executions_workflow_id;
DROP TABLE IF EXISTS ansible_workflow_executions;

DROP INDEX IF EXISTS idx_ansible_workflows_deleted_at;
DROP INDEX IF EXISTS idx_ansible_workflows_user_id;
DROP TABLE IF EXISTS ansible_workflows;
```

**Go 模型**: `backend/internal/model/ansible.go`

```go
// 工作流定义
type AnsibleWorkflow struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null;size:255;comment:工作流名称"`
	Description string         `json:"description" gorm:"type:text;comment:工作流描述"`
	DAG         *WorkflowDAG   `json:"dag" gorm:"type:jsonb;comment:DAG定义"`
	UserID      uint           `json:"user_id" gorm:"not null;index;comment:创建用户ID"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (AnsibleWorkflow) TableName() string {
	return "ansible_workflows"
}

// DAG 定义
type WorkflowDAG struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

// 工作流节点
type WorkflowNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // task/start/end
	Label      string                 `json:"label"`
	TaskConfig *TaskCreateRequest     `json:"task_config,omitempty"`
	Position   Position               `json:"position"`
}

// 节点位置（用于UI展示）
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// 工作流边
type WorkflowEdge struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Condition string `json:"condition,omitempty"` // 条件表达式
}

// 工作流执行记录
type AnsibleWorkflowExecution struct {
	ID           uint           `json:"id" gorm:"primarykey"`
	WorkflowID   uint           `json:"workflow_id" gorm:"not null;index;comment:工作流ID"`
	Status       string         `json:"status" gorm:"size:50;default:'running';index;comment:执行状态"`
	StartedAt    time.Time      `json:"started_at" gorm:"comment:开始时间"`
	FinishedAt   *time.Time     `json:"finished_at" gorm:"comment:完成时间"`
	ErrorMessage string         `json:"error_message" gorm:"type:text;comment:错误信息"`
	UserID       uint           `json:"user_id" gorm:"not null;index;comment:执行用户ID"`
	
	// 关联
	Workflow *AnsibleWorkflow `json:"workflow,omitempty" gorm:"foreignKey:WorkflowID"`
	User     *User            `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tasks    []AnsibleTask    `json:"tasks,omitempty" gorm:"foreignKey:WorkflowExecutionID"`
}

// TableName 指定表名
func (AnsibleWorkflowExecution) TableName() string {
	return "ansible_workflow_executions"
}

// 扩展 AnsibleTask
type AnsibleTask struct {
	// ... 现有字段
	WorkflowExecutionID *uint    `json:"workflow_execution_id" gorm:"index;comment:工作流执行ID"`
	DependsOn           []string `json:"depends_on" gorm:"type:jsonb;comment:依赖的节点ID列表"`
	NodeID              string   `json:"node_id" gorm:"size:50;comment:工作流节点ID"`
	// ...
}
```

#### Phase 2: 后端服务实现（3-4天）

**文件**: `backend/internal/service/ansible/workflow.go`

```go
package ansible

import (
	"context"
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"sync"
	"time"

	"gorm.io/gorm"
)

// WorkflowService 工作流服务
type WorkflowService struct {
	db     *gorm.DB
	logger *logger.Logger
	svc    *Service // 引用主 Service 以执行任务
}

// NewWorkflowService 创建工作流服务实例
func NewWorkflowService(db *gorm.DB, logger *logger.Logger, svc *Service) *WorkflowService {
	return &WorkflowService{
		db:     db,
		logger: logger,
		svc:    svc,
	}
}

// CreateWorkflow 创建工作流
func (s *WorkflowService) CreateWorkflow(name, description string, dag *model.WorkflowDAG, userID uint) (*model.AnsibleWorkflow, error) {
	// 验证 DAG
	if err := s.validateDAG(dag); err != nil {
		return nil, fmt.Errorf("invalid DAG: %w", err)
	}
	
	workflow := &model.AnsibleWorkflow{
		Name:        name,
		Description: description,
		DAG:         dag,
		UserID:      userID,
	}
	
	if err := s.db.Create(workflow).Error; err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}
	
	s.logger.Infof("Created workflow %d: %s", workflow.ID, workflow.Name)
	return workflow, nil
}

// validateDAG 验证 DAG 有效性
func (s *WorkflowService) validateDAG(dag *model.WorkflowDAG) error {
	// 1. 检查是否有环
	if hasCycle(dag) {
		return fmt.Errorf("DAG contains cycle")
	}
	
	// 2. 检查节点引用
	nodeIDs := make(map[string]bool)
	for _, node := range dag.Nodes {
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIDs[node.ID] = true
	}
	
	for _, edge := range dag.Edges {
		if !nodeIDs[edge.Source] {
			return fmt.Errorf("edge source not found: %s", edge.Source)
		}
		if !nodeIDs[edge.Target] {
			return fmt.Errorf("edge target not found: %s", edge.Target)
		}
	}
	
	return nil
}

// hasCycle 检测 DAG 是否有环（DFS）
func hasCycle(dag *model.WorkflowDAG) bool {
	// 构建邻接表
	adj := make(map[string][]string)
	for _, edge := range dag.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
	}
	
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	var dfs func(string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		
		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}
		
		recStack[node] = false
		return false
	}
	
	for _, node := range dag.Nodes {
		if !visited[node.ID] {
			if dfs(node.ID) {
				return true
			}
		}
	}
	
	return false
}

// ExecuteWorkflow 执行工作流
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uint, userID uint) (*model.AnsibleWorkflowExecution, error) {
	// 1. 获取工作流定义
	var workflow model.AnsibleWorkflow
	if err := s.db.First(&workflow, workflowID).Error; err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}
	
	// 2. 创建执行记录
	execution := &model.AnsibleWorkflowExecution{
		WorkflowID: workflowID,
		Status:     "running",
		StartedAt:  time.Now(),
		UserID:     userID,
	}
	
	if err := s.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}
	
	// 3. 异步执行 DAG
	go s.executeDAG(ctx, execution, &workflow)
	
	return execution, nil
}

// executeDAG 执行 DAG
func (s *WorkflowService) executeDAG(ctx context.Context, execution *model.AnsibleWorkflowExecution, workflow *model.AnsibleWorkflow) {
	// 拓扑排序获取执行顺序
	levels := s.topologicalSort(workflow.DAG)
	
	s.logger.Infof("Starting workflow execution %d with %d levels", execution.ID, len(levels))
	
	// 按层级执行（每层可以并行执行）
	for levelIdx, level := range levels {
		s.logger.Infof("Executing level %d with %d nodes", levelIdx, len(level))
		
		// 并行执行同一层级的任务
		var wg sync.WaitGroup
		errCh := make(chan error, len(level))
		
		for _, nodeID := range level {
			node := s.getNodeByID(workflow.DAG, nodeID)
			if node == nil || node.Type != "task" {
				continue
			}
			
			wg.Add(1)
			go func(n *model.WorkflowNode) {
				defer wg.Done()
				
				// 创建并执行任务
				task, err := s.svc.CreateTask(*n.TaskConfig, execution.UserID)
				if err != nil {
					errCh <- fmt.Errorf("failed to create task for node %s: %w", n.ID, err)
					return
				}
				
				// 更新任务的工作流信息
				task.WorkflowExecutionID = &execution.ID
				task.NodeID = n.ID
				s.db.Save(task)
				
				// 等待任务完成
				if err := s.waitForTask(ctx, task.ID); err != nil {
					errCh <- fmt.Errorf("task %d failed: %w", task.ID, err)
					return
				}
				
				s.logger.Infof("Task %d for node %s completed successfully", task.ID, n.ID)
			}(node)
		}
		
		wg.Wait()
		close(errCh)
		
		// 检查是否有错误
		if len(errCh) > 0 {
			err := <-errCh
			s.logger.Errorf("Workflow execution %d failed: %v", execution.ID, err)
			
			// 更新执行状态为失败
			now := time.Now()
			execution.Status = "failed"
			execution.FinishedAt = &now
			execution.ErrorMessage = err.Error()
			s.db.Save(execution)
			return
		}
		
		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			s.logger.Warningf("Workflow execution %d cancelled", execution.ID)
			now := time.Now()
			execution.Status = "cancelled"
			execution.FinishedAt = &now
			s.db.Save(execution)
			return
		default:
		}
	}
	
	// 所有任务执行成功
	s.logger.Infof("Workflow execution %d completed successfully", execution.ID)
	now := time.Now()
	execution.Status = "completed"
	execution.FinishedAt = &now
	s.db.Save(execution)
}

// topologicalSort 拓扑排序（Kahn算法）
func (s *WorkflowService) topologicalSort(dag *model.WorkflowDAG) [][]string {
	// 构建邻接表和入度表
	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	
	for _, node := range dag.Nodes {
		if node.Type == "task" {
			inDegree[node.ID] = 0
		}
	}
	
	for _, edge := range dag.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
		if _, ok := inDegree[edge.Target]; ok {
			inDegree[edge.Target]++
		}
	}
	
	// 分层
	var levels [][]string
	queue := []string{}
	
	// 找到所有入度为0的节点
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}
	
	for len(queue) > 0 {
		// 当前层级
		level := queue
		queue = []string{}
		levels = append(levels, level)
		
		// 处理当前层级的所有节点
		for _, nodeID := range level {
			for _, neighbor := range adj[nodeID] {
				if _, ok := inDegree[neighbor]; ok {
					inDegree[neighbor]--
					if inDegree[neighbor] == 0 {
						queue = append(queue, neighbor)
					}
				}
			}
		}
	}
	
	return levels
}

// getNodeByID 根据ID获取节点
func (s *WorkflowService) getNodeByID(dag *model.WorkflowDAG, nodeID string) *model.WorkflowNode {
	for i := range dag.Nodes {
		if dag.Nodes[i].ID == nodeID {
			return &dag.Nodes[i]
		}
	}
	return nil
}

// waitForTask 等待任务完成
func (s *WorkflowService) waitForTask(ctx context.Context, taskID uint) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var task model.AnsibleTask
			if err := s.db.Select("id", "status").First(&task, taskID).Error; err != nil {
				return err
			}
			
			switch task.Status {
			case model.AnsibleTaskStatusSuccess:
				return nil
			case model.AnsibleTaskStatusFailed, model.AnsibleTaskStatusCancelled:
				return fmt.Errorf("task %d failed with status: %s", taskID, task.Status)
			}
		}
	}
}

// GetWorkflowExecution 获取工作流执行详情
func (s *WorkflowService) GetWorkflowExecution(executionID uint) (*model.AnsibleWorkflowExecution, error) {
	var execution model.AnsibleWorkflowExecution
	if err := s.db.Preload("Workflow").Preload("Tasks").First(&execution, executionID).Error; err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}
	return &execution, nil
}

// ListWorkflows 获取工作流列表
func (s *WorkflowService) ListWorkflows(userID uint) ([]model.AnsibleWorkflow, error) {
	var workflows []model.AnsibleWorkflow
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&workflows).Error; err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	return workflows, nil
}
```

#### Phase 3: API 端点（1天）

**文件**: `backend/cmd/main.go`

```go
// 工作流管理
workflow := ansible.Group("/workflows")
{
    workflow.POST("", handlers.AnsibleWorkflow.CreateWorkflow)
    workflow.GET("", handlers.AnsibleWorkflow.ListWorkflows)
    workflow.GET("/:id", handlers.AnsibleWorkflow.GetWorkflow)
    workflow.PUT("/:id", handlers.AnsibleWorkflow.UpdateWorkflow)
    workflow.DELETE("/:id", handlers.AnsibleWorkflow.DeleteWorkflow)
    workflow.POST("/:id/execute", handlers.AnsibleWorkflow.ExecuteWorkflow)
    workflow.GET("/:id/executions", handlers.AnsibleWorkflow.ListExecutions)
    workflow.GET("/executions/:id", handlers.AnsibleWorkflow.GetExecution)
}
```

#### Phase 4: 前端可视化编辑器（4-5天）

使用 **Vue Flow** 库实现拖拽式工作流编辑器。

**安装依赖**:
```bash
npm install @vue-flow/core @vue-flow/background @vue-flow/controls @vue-flow/minimap
```

**组件**: `frontend/src/components/ansible/WorkflowEditor.vue`

这个组件会比较复杂，需要实现：
1. 节点拖拽
2. 连线
3. 节点配置
4. 保存/加载
5. 执行和监控

---

## 📅 实施优先级和时间表

### 第一周（高优先级）

**Day 1-2**: 
- ✅ 修复模板变量显示问题
- ✅ 优化前置检查按钮显示逻辑
- ✅ 开发基础时间线可视化组件

**Day 3-5**:
- ✅ 集成时间线可视化到任务详情
- ✅ 添加饼图/柱状图展示
- ✅ 完善 UI 交互和样式

### 第二周（中优先级）

**Day 1-2**:
- DAG 工作流数据模型设计
- 数据库迁移

**Day 3-5**:
- 后端服务实现（WorkflowService）
- API 端点开发
- 单元测试

### 第三周（低优先级）

**Day 1-3**:
- 前端工作流编辑器开发
- Vue Flow 集成

**Day 4-5**:
- 工作流执行监控
- 测试和优化

---

## 📝 验收标准

### 模板变量验证
- ✅ 创建模板时可以设置必需变量
- ✅ 创建任务时自动显示变量输入框
- ✅ 缺少必需变量时无法提交

### 前置检查
- ✅ Pending 和 Failed 状态的任务可执行检查
- ✅ 检查结果清晰展示
- ✅ 可选：创建任务时自动执行检查

### 可视化
- ✅ 显示完整的执行时间线
- ✅ 阶段耗时可视化（饼图/柱状图）
- ✅ 批次执行详情展示
- ✅ 响应式设计

### DAG 工作流
- ✅ 可视化编辑工作流
- ✅ 自动检测环
- ✅ 按依赖顺序执行
- ✅ 支持并行执行
- ✅ 失败处理
- ✅ 执行状态监控

---

**文档创建**: 2025-01-13  
**预计完成**: 2025-02-03 (3周)  
**负责人**: 开发团队

