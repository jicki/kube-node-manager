# Ansible 任务执行可视化

## 功能概述

任务执行可视化功能为 Ansible 任务提供了直观的执行流程展示，帮助用户快速了解任务执行的各个阶段、主机状态和性能指标。

## 核心特性

### 1. 执行时间线

自动记录任务执行的每个关键阶段：

- **queued**: 任务入队等待
- **preflight_check**: 前置检查
- **executing**: 任务执行中
- **batch_paused**: 批次暂停（如果启用分批执行）
- **completed**: 执行完成
- **failed**: 执行失败
- **cancelled**: 任务取消
- **timeout**: 执行超时

每个阶段记录：
- 阶段名称
- 事件消息
- 时间戳
- 耗时（毫秒）
- 详细信息（如主机数、成功/失败数等）

### 2. 主机执行状态

展示每个主机的执行情况：

- 主机名
- 执行状态（ok/failed/skipped/unreachable）
- 开始/结束时间
- 执行时长
- 成功/失败/跳过的任务数
- 是否有变更
- 错误信息

### 3. 阶段耗时分布

以可视化图表展示各个阶段的耗时占比，帮助识别性能瓶颈。

### 4. 自动数据收集

执行器在任务执行过程中自动收集并记录：
- 任务开始时添加"executing"事件
- 任务完成时添加"completed/failed/timeout"事件
- 从 Ansible 日志中提取主机执行信息

## 使用场景

### 场景 1: 任务性能分析

当任务执行缓慢时，通过可视化查看：
- 哪个阶段耗时最多
- 哪些主机执行较慢
- 是否有主机失败

### 场景 2: 故障排查

当任务失败时，快速定位：
- 在哪个阶段失败
- 哪些主机失败
- 具体的错误信息

### 场景 3: 执行效率优化

通过时间线数据：
- 识别可以并行执行的部分
- 优化分批执行策略
- 调整超时配置

## API 接口

### 获取任务执行可视化数据

```http
GET /api/v1/ansible/tasks/{id}/visualization
```

**响应示例:**

```json
{
  "code": 200,
  "data": {
    "task_id": 123,
    "task_name": "部署应用",
    "status": "success",
    "timeline": [
      {
        "phase": "queued",
        "message": "任务已入队",
        "timestamp": "2025-11-03T10:00:00Z",
        "duration": 5000,
        "details": {}
      },
      {
        "phase": "preflight_check",
        "message": "前置检查: pass",
        "timestamp": "2025-11-03T10:00:05Z",
        "duration": 2000,
        "details": {}
      },
      {
        "phase": "executing",
        "message": "任务开始执行",
        "timestamp": "2025-11-03T10:00:07Z",
        "duration": 45000,
        "host_count": 10,
        "details": {
          "dry_run": false,
          "batch_enabled": true
        }
      },
      {
        "phase": "completed",
        "message": "任务执行成功",
        "timestamp": "2025-11-03T10:00:52Z",
        "duration": 0,
        "host_count": 10,
        "success_count": 10,
        "fail_count": 0,
        "details": {
          "hosts_total": 10,
          "hosts_ok": 10,
          "hosts_failed": 0,
          "hosts_skipped": 0
        }
      }
    ],
    "host_statuses": [
      {
        "host_name": "web-01",
        "status": "ok",
        "start_time": "2025-11-03T10:00:07Z",
        "end_time": "2025-11-03T10:00:25Z",
        "duration": 18000,
        "tasks_ok": 5,
        "tasks_failed": 0,
        "tasks_skipped": 0,
        "changed": true
      },
      {
        "host_name": "web-02",
        "status": "ok",
        "start_time": "2025-11-03T10:00:07Z",
        "end_time": "2025-11-03T10:00:30Z",
        "duration": 23000,
        "tasks_ok": 5,
        "tasks_failed": 0,
        "tasks_skipped": 0,
        "changed": true
      }
    ],
    "total_duration": 52000,
    "phase_distribution": {
      "queued": 5000,
      "preflight_check": 2000,
      "executing": 45000
    }
  }
}
```

### 获取任务时间线摘要

```http
GET /api/v1/ansible/tasks/{id}/timeline-summary
```

**响应示例:**

```json
{
  "code": 200,
  "data": {
    "task_id": 123,
    "task_name": "部署应用",
    "status": "success",
    "total_duration_ms": 52000,
    "total_duration_readable": "52s",
    "phase_count": 4,
    "host_count": 10,
    "phase_stats": {
      "queued": 1,
      "preflight_check": 1,
      "executing": 1,
      "completed": 1
    },
    "host_status_stats": {
      "ok": 10
    }
  }
}
```

## 技术实现

### 后端实现

#### 1. 数据模型

```go
// 任务模型添加执行时间线字段
type AnsibleTask struct {
    // ... 其他字段
    ExecutionTimeline *TaskExecutionTimeline `json:"execution_timeline" gorm:"type:jsonb"`
}

// 执行事件
type TaskExecutionEvent struct {
    Phase       ExecutionPhase `json:"phase"`
    Message     string         `json:"message"`
    Timestamp   time.Time      `json:"timestamp"`
    Duration    int            `json:"duration"`
    HostCount   int            `json:"host_count"`
    SuccessCount int           `json:"success_count"`
    FailCount   int            `json:"fail_count"`
    Details     map[string]interface{} `json:"details"`
}

// 主机执行状态
type HostExecutionStatus struct {
    HostName     string    `json:"host_name"`
    Status       string    `json:"status"`
    StartTime    time.Time `json:"start_time"`
    EndTime      time.Time `json:"end_time"`
    Duration     int       `json:"duration"`
    TasksOk      int       `json:"tasks_ok"`
    TasksFailed  int       `json:"tasks_failed"`
    TasksSkipped int       `json:"tasks_skipped"`
    Changed      bool      `json:"changed"`
}
```

#### 2. 事件记录

在任务执行器中的关键位置记录事件：

```go
// 任务开始执行
task.AddExecutionEvent(model.PhaseExecuting, "任务开始执行", map[string]interface{}{
    "dry_run": task.DryRun,
    "batch_enabled": task.IsBatchEnabled(),
})

// 任务完成
task.AddExecutionEvent(phase, message, map[string]interface{}{
    "hosts_total":   task.HostsTotal,
    "hosts_ok":      task.HostsOk,
    "hosts_failed":  task.HostsFailed,
    "hosts_skipped": task.HostsSkipped,
})
```

#### 3. 可视化服务

`VisualizationService` 提供：
- `GetTaskVisualization()`: 获取完整的可视化数据
- `GetTaskTimelineSummary()`: 获取摘要信息
- `generateBasicTimeline()`: 为旧任务生成基本时间线
- `extractHostStatuses()`: 从日志中提取主机状态

#### 4. 日志解析

从 Ansible 输出日志中解析主机状态：

```go
// 识别 Ansible 标准输出格式
"ok: [hostname]"       -> 主机成功
"failed: [hostname]"   -> 主机失败
"skipped: [hostname]"  -> 主机跳过
"changed: [hostname]"  -> 主机有变更
```

### 前端实现

#### 1. API 调用

```javascript
import * as ansibleAPI from '@/api/ansible'

// 获取可视化数据
const viz = await ansibleAPI.getTaskVisualization(taskId)

// 获取摘要
const summary = await ansibleAPI.getTaskTimelineSummary(taskId)
```

#### 2. 可视化展示

推荐使用以下图表：

**时间线图表：**
- 使用 Element Plus 的 Timeline 组件
- 显示每个阶段的事件

**饼图/环形图：**
- 展示各阶段耗时分布
- 使用 ECharts 或 Chart.js

**主机状态表格：**
- 使用 Element Plus 的 Table 组件
- 支持排序和筛选

**进度条：**
- 显示主机执行进度
- 使用不同颜色表示状态（成功/失败）

#### 3. UI 设计建议

```vue
<template>
  <el-dialog title="任务执行可视化" v-model="vizDialogVisible" width="80%">
    <!-- 摘要卡片 -->
    <el-card>
      <el-statistic title="总耗时" :value="totalDuration" suffix="ms" />
      <el-statistic title="主机数量" :value="hostCount" />
      <el-statistic title="成功主机" :value="successHostCount" />
    </el-card>

    <!-- 时间线 -->
    <el-timeline>
      <el-timeline-item 
        v-for="event in timeline" 
        :key="event.timestamp"
        :timestamp="formatTimestamp(event.timestamp)"
      >
        <h4>{{ event.phase }} - {{ event.message }}</h4>
        <p>耗时: {{ event.duration }}ms</p>
      </el-timeline-item>
    </el-timeline>

    <!-- 阶段分布图 -->
    <div ref="phaseChart" style="width: 100%; height: 300px"></div>

    <!-- 主机状态表格 -->
    <el-table :data="hostStatuses">
      <el-table-column prop="host_name" label="主机名" />
      <el-table-column prop="status" label="状态">
        <template #default="{ row }">
          <el-tag :type="getStatusType(row.status)">
            {{ row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="duration" label="耗时(ms)" sortable />
    </el-table>
  </el-dialog>
</template>
```

## 性能优化

### 1. 数据库优化

- 执行时间线存储为 JSONB 类型，支持高效查询
- 只在关键阶段记录事件，避免过多数据

### 2. 日志解析优化

- 异步解析主机状态，不阻塞任务完成
- 缓存解析结果，避免重复解析

### 3. 前端优化

- 懒加载可视化数据，只在用户点击时加载
- 使用虚拟滚动处理大量主机数据

## 扩展功能

### 1. 实时可视化

通过 WebSocket 实时推送执行事件，实现动态更新的可视化界面。

### 2. 性能对比

对比同一模板的多次执行，分析性能趋势。

### 3. 导出报告

将可视化数据导出为 PDF 或图片格式，用于文档和报告。

### 4. 告警阈值

设置各阶段耗时阈值，自动识别异常慢的执行。

## 版本信息

- **引入版本**: v2.32.0
- **最后更新**: 2025-11-03

## 相关文档

- [Ansible 模块功能概述](./ansible-overview.md)
- [任务队列优化](./ansible-task-queue-optimization.md)
- [前置检查](./ansible-preflight-checks.md)
- [超时控制](./ansible-timeout-control.md)
- [任务执行预估](./ansible-task-estimation.md)

