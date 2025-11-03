# Ansible 任务队列优化

## 功能概述

任务队列优化功能为 Ansible 任务执行提供了智能调度和优先级管理能力，确保重要任务能够优先执行，同时实现公平调度，避免任务饥饿。

## 核心特性

### 1. 任务优先级

支持三种优先级级别：

- **高优先级（High）**: 紧急任务，优先执行
- **中优先级（Medium）**: 默认级别，正常调度
- **低优先级（Low）**: 非紧急任务，系统空闲时执行

### 2. 公平调度

- **用户级别的公平性**: 防止单个用户的大量任务占用所有执行资源
- **并发控制**: 每个用户的同时运行任务数有上限，避免资源独占
- **队列等待统计**: 自动跟踪任务的等待时长

### 3. 队列管理

- **优先级队列**: 使用堆数据结构实现高效的优先级调度
- **FIFO 原则**: 同优先级任务按照入队时间先进先出
- **实时统计**: 提供队列状态的实时统计信息

## 使用场景

### 场景 1: 紧急修复任务

当需要紧急修复生产环境问题时：

```json
{
  "name": "紧急修复 - 重启故障服务",
  "priority": "high",
  "template_id": 5,
  "inventory_id": 2
}
```

高优先级任务会立即排在队列前面，优先执行。

### 场景 2: 日常维护任务

定期的系统更新和维护：

```json
{
  "name": "每日系统更新",
  "priority": "medium",
  "template_id": 10,
  "inventory_id": 3
}
```

中优先级任务按正常顺序调度。

### 场景 3: 批量数据处理

不紧急的批量操作：

```json
{
  "name": "批量日志清理",
  "priority": "low",
  "template_id": 15,
  "inventory_id": 4
}
```

低优先级任务会在系统资源空闲时执行，不影响紧急任务。

## UI 功能

### 任务创建时选择优先级

在任务创建对话框中：

1. 找到"任务优先级"下拉选择框
2. 选择合适的优先级级别：
   - 高优先级：带红色图标，显示"优先执行"
   - 中优先级：带橙色图标，显示"默认"（默认选项）
   - 低优先级：带灰色图标，显示"空闲时执行"

### 任务列表显示优先级

在任务列表的"优先级"列中：

- 高优先级：红色标签，显示上箭头图标
- 中优先级：橙色标签，显示横线图标
- 低优先级：灰色标签，显示下箭头图标

### 队列统计信息

通过 API 获取实时队列统计：

```javascript
// 获取队列统计
const stats = await ansibleAPI.getQueueStats()

console.log('待执行任务数:', stats.total_pending)
console.log('正在运行任务数:', stats.total_running)
console.log('平均等待时间:', stats.avg_wait_seconds, '秒')
console.log('最长等待时间:', stats.max_wait_seconds, '秒')
console.log('按优先级统计:', stats.by_priority)
console.log('按用户统计:', stats.by_user)
```

## 技术实现

### 后端实现

#### 1. 数据模型扩展

为 `ansible_tasks` 表添加了以下字段：

- `priority`: 任务优先级（high/medium/low）
- `queued_at`: 任务入队时间
- `wait_duration`: 任务等待时长（秒）

#### 2. 优先级队列服务

`backend/internal/service/ansible/queue.go` 实现了：

- **TaskPriorityQueue**: 基于 Go 标准库 `container/heap` 的优先级队列
- **GetNextTask()**: 获取下一个应该执行的任务（考虑优先级和公平性）
- **GetQueueStats()**: 获取队列统计信息
- **UpdateWaitDuration()**: 更新任务的等待时长

#### 3. 调度逻辑

```go
// 调度算法伪代码
1. 查询所有待执行任务
2. 统计每个用户当前正在运行的任务数
3. 过滤掉已达到并发上限的用户的任务
4. 构建优先级队列：
   - 优先级高的任务排在前面
   - 同优先级按入队时间 FIFO
5. 从队列中取出最高优先级的任务
6. 返回该任务用于执行
```

#### 4. 等待时长追踪

在任务开始执行时，executor 会自动调用 `UpdateWaitDuration()`：

```go
// 任务开始执行时
task.MarkStarted()
queueSvc.UpdateWaitDuration(task.ID)
```

### 前端实现

#### 1. 优先级选择组件

在任务创建表单中添加了优先级选择器：

```vue
<el-form-item label="任务优先级">
  <el-select v-model="taskForm.priority">
    <el-option label="高优先级" value="high" />
    <el-option label="中优先级" value="medium" />
    <el-option label="低优先级" value="low" />
  </el-select>
</el-form-item>
```

#### 2. 优先级显示

在任务列表中以标签形式展示优先级：

```vue
<el-table-column label="优先级" width="100">
  <template #default="{ row }">
    <el-tag 
      :type="row.priority === 'high' ? 'danger' : row.priority === 'medium' ? 'warning' : 'info'"
    >
      {{ row.priority === 'high' ? '高' : row.priority === 'medium' ? '中' : '低' }}
    </el-tag>
  </template>
</el-table-column>
```

## API 接口

### 获取队列统计

```http
GET /api/v1/ansible/queue/stats
```

**响应示例:**

```json
{
  "code": 200,
  "data": {
    "total_pending": 15,
    "total_running": 3,
    "by_priority": {
      "high": 5,
      "medium": 8,
      "low": 2
    },
    "by_user": {
      "1": 10,
      "2": 5
    },
    "avg_wait_seconds": 120,
    "max_wait_seconds": 300,
    "max_wait_task_id": 1234
  }
}
```

## 配置参数

### 公平调度参数

在 `queue.go` 的 `GetNextTask()` 方法中可以配置：

```go
maxConcurrentPerUser := 2 // 每个用户最多同时运行2个任务
```

### 并发执行数

在 `executor.go` 中配置：

```go
maxConcurrent := 5 // 系统最多同时执行5个任务
```

## 性能优化

### 1. 数据库索引

为优化队列查询性能，创建了以下索引：

```sql
-- 优先级索引
CREATE INDEX idx_ansible_tasks_priority ON ansible_tasks (priority);

-- 队列复合索引（状态 + 优先级 + 入队时间）
CREATE INDEX idx_ansible_tasks_queue ON ansible_tasks (status, priority, queued_at) 
WHERE status = 'pending';
```

### 2. 查询优化

- 使用部分索引（WHERE status = 'pending'）减少索引大小
- 按优先级和入队时间排序查询，利用索引
- 只查询必要的字段，减少数据传输

## 监控与观测

### 关键指标

1. **队列长度**: `total_pending`
2. **平均等待时间**: `avg_wait_seconds`
3. **最长等待时间**: `max_wait_seconds`
4. **按优先级分布**: `by_priority`
5. **按用户分布**: `by_user`

### 告警建议

- 当 `avg_wait_seconds > 300` 时，考虑增加执行器数量
- 当 `max_wait_seconds > 600` 时，检查是否有任务阻塞
- 当某个用户的任务占比 > 70% 时，考虑调整公平调度参数

## 最佳实践

### 1. 优先级选择原则

- **高优先级**: 仅用于紧急故障修复、安全漏洞修补等
- **中优先级**: 日常运维任务的默认选择
- **低优先级**: 批量数据处理、定期清理等可延迟任务

### 2. 避免滥用高优先级

过多的高优先级任务会导致：
- 中低优先级任务长时间等待
- 失去优先级管理的意义
- 影响系统整体调度效率

建议：限制高优先级任务占比在 10% 以内。

### 3. 监控队列健康度

定期检查队列统计，及时发现并处理：
- 任务堆积（pending 数量过多）
- 等待时间过长
- 用户分布不均

## 版本信息

- **引入版本**: v2.30.0
- **最后更新**: 2025-11-03

## 相关文档

- [Ansible 模块功能概述](./ansible-overview.md)
- [Dry Run 模式](./ansible-dry-run-mode.md)
- [分阶段执行](./ansible-phased-execution.md)
- [批次执行控制](./ansible-batch-control.md)
- [前置检查](./ansible-preflight-checks.md)
- [超时控制](./ansible-timeout-control.md)
- [任务执行预估](./ansible-task-estimation.md)

