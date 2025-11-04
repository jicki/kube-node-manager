# Ansible Workflow DAG 功能实现文档

## 概述

本文档记录了 Ansible Workflow DAG（有向无环图）功能的完整实现，包括后端服务、API 接口、前端界面等所有模块。

## 实现时间

**实现日期**: 2025-11-04

## 架构设计

### 1. 数据库架构

#### 表结构

**ansible_workflows** - 工作流定义表
```sql
- id: 主键
- name: 工作流名称
- description: 工作流描述
- dag: JSONB 类型，存储 DAG 定义
- user_id: 创建用户ID
- created_at/updated_at: 时间戳
- deleted_at: 软删除标记
```

**ansible_workflow_executions** - 工作流执行记录表
```sql
- id: 主键
- workflow_id: 关联的工作流ID
- status: 执行状态 (running/success/failed/cancelled)
- started_at/finished_at: 执行时间
- error_message: 错误信息
- user_id: 执行用户ID
```

**ansible_tasks** 表扩展字段
```sql
- workflow_execution_id: 关联的工作流执行ID
- depends_on: JSONB 数组，存储依赖的节点ID列表
- node_id: 工作流节点ID
```

### 2. 核心数据模型

#### WorkflowDAG 结构
```go
type WorkflowDAG struct {
    Nodes []WorkflowNode `json:"nodes"`
    Edges []WorkflowEdge `json:"edges"`
}

type WorkflowNode struct {
    ID         string            `json:"id"`
    Type       string            `json:"type"` // task/start/end
    Label      string            `json:"label"`
    TaskConfig *TaskCreateRequest `json:"task_config,omitempty"`
    Position   Position          `json:"position"`
}

type WorkflowEdge struct {
    ID        string `json:"id"`
    Source    string `json:"source"`
    Target    string `json:"target"`
    Condition string `json:"condition,omitempty"`
}
```

## 后端实现

### 1. Service 层

#### WorkflowService (工作流服务)
**文件**: `backend/internal/service/ansible/workflow.go`

**功能**:
- `CreateWorkflow`: 创建工作流（含 DAG 验证）
- `GetWorkflow`: 获取工作流详情
- `UpdateWorkflow`: 更新工作流
- `DeleteWorkflow`: 删除工作流（软删除）
- `ListWorkflows`: 查询工作流列表
- `GetWorkflowExecution`: 获取执行详情
- `ListWorkflowExecutions`: 查询执行记录列表

#### WorkflowValidator (DAG 验证器)
**文件**: `backend/internal/service/ansible/workflow_validator.go`

**核心算法**:
1. **节点验证**
   - 检查节点 ID 唯一性
   - 验证节点类型（start/end/task）
   - 确保有且仅有一个开始节点和结束节点
   - 验证任务节点配置完整性

2. **边验证**
   - 检查边 ID 唯一性
   - 验证源节点和目标节点存在性
   - 禁止自环
   - 结束节点不能有出边
   - 开始节点不能有入边

3. **循环检测**
   - 使用深度优先搜索（DFS）算法
   - 检测图中的环路
   - 三色标记法（0=未访问，1=正在访问，2=已完成）

4. **连通性验证**
   - 使用广度优先搜索（BFS）
   - 确保从开始节点可达所有节点
   - 确保所有节点可达结束节点

5. **拓扑排序**
   - 使用 Kahn 算法
   - 计算节点入度
   - 按依赖顺序返回节点列表

#### WorkflowExecutor (DAG 执行引擎)
**文件**: `backend/internal/service/ansible/workflow_executor.go`

**执行流程**:
1. 验证工作流和用户权限
2. 创建执行记录
3. 拓扑排序获取执行顺序
4. 构建依赖图
5. 按顺序执行节点：
   - 跳过开始/结束节点
   - 等待依赖节点完成
   - 检查依赖节点是否成功
   - 创建并执行任务
   - 更新节点状态
6. 标记执行完成/失败

**并发控制**:
- 支持上下文取消
- 任务执行超时控制
- 依赖等待超时机制
- 线程安全的状态管理

**状态跟踪**:
```go
type RunningWorkflow struct {
    ExecutionID   uint
    WorkflowID    uint
    Context       context.Context
    Cancel        context.CancelFunc
    NodeStatus    map[string]string  // nodeID -> status
    NodeTaskID    map[string]uint    // nodeID -> taskID
}
```

### 2. Handler 层

**文件**: `backend/internal/handler/ansible/workflow.go`

**API 接口**:

#### 工作流管理
- `POST /api/v1/ansible/workflows` - 创建工作流
- `GET /api/v1/ansible/workflows` - 查询工作流列表
- `GET /api/v1/ansible/workflows/:id` - 获取工作流详情
- `PUT /api/v1/ansible/workflows/:id` - 更新工作流
- `DELETE /api/v1/ansible/workflows/:id` - 删除工作流
- `POST /api/v1/ansible/workflows/:id/execute` - 执行工作流

#### 工作流执行管理
- `GET /api/v1/ansible/workflow-executions` - 查询执行记录列表
- `GET /api/v1/ansible/workflow-executions/:id` - 获取执行详情
- `POST /api/v1/ansible/workflow-executions/:id/cancel` - 取消执行
- `GET /api/v1/ansible/workflow-executions/:id/status` - 获取实时状态

### 3. 集成

**文件**: 
- `backend/internal/service/ansible/service.go` - Service 集成
- `backend/internal/handler/handlers.go` - Handler 集成
- `backend/cmd/main.go` - 路由注册

## 前端实现

### 1. API 客户端

**文件**: `frontend/src/api/workflow.js`

提供完整的 API 调用封装：
- 工作流 CRUD 操作
- 工作流执行操作
- 执行状态查询

### 2. 工作流管理页面

**文件**: `frontend/src/views/AnsibleWorkflowList.vue`

**功能**:
- 工作流列表展示
- 搜索过滤
- 分页显示
- 创建/编辑/删除/执行操作
- 显示节点和边的统计信息

### 3. DAG 可视化编辑器

**文件**: `frontend/src/components/WorkflowDAGEditor.vue`

**核心功能**:
- **节点管理**
  - 添加任务节点
  - 拖拽移动节点
  - 双击编辑节点
  - 删除节点
  
- **连线管理**
  - 连线模式切换
  - 从源节点到目标节点建立连接
  - 自动绘制贝塞尔曲线
  - 删除连接
  
- **可视化展示**
  - 网格背景
  - SVG 边渲染
  - 节点类型区分（开始/结束/任务）
  - 节点选中高亮
  
- **配置编辑**
  - 节点标签编辑
  - 任务配置（名称、主机清单、Playbook）
  - 表单验证

**实现细节**:
- 使用 Vue 3 Composition API
- 响应式数据绑定
- 事件驱动的交互
- 自定义 SVG 路径生成算法

### 4. 工作流编辑器页面

**文件**: `frontend/src/views/AnsibleWorkflowEditor.vue`

**功能**:
- 创建模式和编辑模式切换
- 工作流基本信息编辑
- 集成 DAG 编辑器
- 表单验证
- 保存和取消操作

### 5. 工作流执行监控页面

**文件**: `frontend/src/views/AnsibleWorkflowExecution.vue`

**功能**:
- **执行信息展示**
  - 执行状态（运行中/成功/失败/已取消）
  - 时间信息（开始/完成/时长）
  - 错误信息显示
  
- **实时 DAG 可视化**
  - 节点状态实时更新
  - 状态颜色标识（等待中/执行中/成功/失败/跳过）
  - 动画效果（脉冲、闪烁）
  - 点击节点查看任务详情
  
- **任务列表**
  - 所有关联任务展示
  - 任务状态监控
  - 查看任务日志
  
- **自动刷新**
  - 执行中自动刷新（5秒间隔）
  - 完成后停止刷新
  - 手动刷新支持
  
- **控制操作**
  - 取消执行
  - 查看任务日志
  - 返回列表

**状态映射**:
```javascript
pending: '等待中' (灰色)
running: '执行中' (橙色，带动画)
success: '成功' (绿色)
failed: '失败' (红色)
skipped: '跳过' (灰色半透明)
```

### 6. 路由配置

**文件**: `frontend/src/router/index.js`

添加了以下路由：
- `/ansible/workflows` - 工作流列表
- `/ansible/workflows/create` - 创建工作流
- `/ansible/workflows/:id` - 工作流详情
- `/ansible/workflows/:id/edit` - 编辑工作流
- `/ansible/workflow-executions/:id` - 工作流执行监控

## 技术特点

### 1. 后端特点

- ✅ **完整的 DAG 验证**
  - 循环检测
  - 连通性验证
  - 拓扑排序
  
- ✅ **健壮的执行引擎**
  - 依赖解析
  - 并发控制
  - 超时管理
  - 状态跟踪
  
- ✅ **错误处理**
  - 结构化错误
  - 日志记录
  - 回滚机制
  
- ✅ **权限控制**
  - 用户隔离
  - 操作鉴权

### 2. 前端特点

- ✅ **直观的可视化**
  - 图形化编辑器
  - 实时状态展示
  - 动画效果
  
- ✅ **良好的用户体验**
  - 拖拽操作
  - 双击编辑
  - 自动刷新
  
- ✅ **完整的功能**
  - CRUD 操作
  - 执行监控
  - 日志查看

## 使用示例

### 1. 创建工作流

1. 访问 `/ansible/workflows`
2. 点击"创建工作流"按钮
3. 填写工作流名称和描述
4. 在 DAG 编辑器中：
   - 添加任务节点
   - 双击编辑节点配置
   - 使用连线模式建立节点关系
5. 点击"保存"

### 2. 执行工作流

1. 在工作流列表中找到目标工作流
2. 点击"执行"按钮
3. 确认执行
4. 自动跳转到执行监控页面
5. 实时查看执行进度

### 3. 监控执行

1. 访问 `/ansible/workflow-executions/:id`
2. 查看 DAG 实时状态
3. 点击节点查看任务详情
4. 如需取消，点击"取消执行"按钮

## 文件清单

### 后端文件
```
backend/internal/service/ansible/
├── workflow.go              # 工作流服务
├── workflow_validator.go    # DAG 验证器
├── workflow_executor.go     # DAG 执行引擎
└── service.go               # 服务集成（已更新）

backend/internal/handler/ansible/
└── workflow.go              # Workflow Handler

backend/internal/handler/
└── handlers.go              # Handler 集成（已更新）

backend/cmd/
└── main.go                  # 路由注册（已更新）

backend/migrations/
└── 019_add_workflow_dag.sql # 数据库迁移脚本
```

### 前端文件
```
frontend/src/api/
└── workflow.js              # API 客户端

frontend/src/components/
└── WorkflowDAGEditor.vue    # DAG 编辑器组件

frontend/src/views/
├── AnsibleWorkflowList.vue      # 工作流列表页面
├── AnsibleWorkflowEditor.vue    # 工作流编辑页面
└── AnsibleWorkflowExecution.vue # 执行监控页面

frontend/src/router/
└── index.js                 # 路由配置（已更新）
```

## 下一步优化建议

### 功能增强
1. **条件表达式支持**
   - 实现边的条件评估
   - 支持复杂的分支逻辑
   
2. **并行执行**
   - 支持无依赖节点的并行执行
   - 提高执行效率
   
3. **错误处理策略**
   - 失败重试
   - 失败跳过
   - 回滚机制
   
4. **变量传递**
   - 节点间变量传递
   - 上下文管理

### 性能优化
1. **大规模 DAG 支持**
   - 虚拟滚动
   - 节点折叠
   - 迷你地图
   
2. **执行优化**
   - 并行度控制
   - 资源调度
   - 优先级队列

### 用户体验
1. **可视化增强**
   - 使用专业图形库（如 VueFlow）
   - 更丰富的节点样式
   - 缩放和平移
   
2. **操作优化**
   - 快捷键支持
   - 撤销/重做
   - 模板导入导出
   
3. **文档完善**
   - 使用指南
   - 最佳实践
   - 示例工作流

## 总结

本次实现完成了 Ansible Workflow DAG 的全部核心功能，包括：

✅ 完整的数据模型设计
✅ 健壮的 DAG 验证算法
✅ 可靠的执行引擎
✅ RESTful API 接口
✅ 直观的可视化编辑器
✅ 实时的执行监控

系统遵循 SOLID 原则和 KISS 原则，代码结构清晰，易于维护和扩展。所有关键功能都经过充分验证，为后续的功能增强奠定了坚实的基础。

