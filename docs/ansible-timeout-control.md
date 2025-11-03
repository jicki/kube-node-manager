# Ansible 任务执行超时控制功能使用指南

## 功能概述

任务执行超时控制功能用于防止任务无限期运行，提高系统稳定性，避免资源浪费。当任务执行时间超过设定的超时时间，系统会自动取消任务执行。

## 实施日期

2025-11-03

## 功能特性

### 核心能力

- ✅ **灵活的超时配置**：支持从 60 秒到 24 小时的超时设置
- ✅ **自动取消机制**：超时后自动终止任务执行
- ✅ **超时状态标记**：明确标识任务是否因超时而失败
- ✅ **快速设置选项**：提供常用超时时间快捷按钮（5分钟、10分钟、30分钟、1小时、2小时）
- ✅ **友好的时长显示**：自动转换为可读的时长格式
- ✅ **任务列表展示**：在任务列表中显示超时限制和超时状态

### 使用场景

#### 1. 防止任务失控
**问题**：某些任务可能由于网络问题或主机响应缓慢而长时间挂起  
**解决**：设置合理的超时时间，自动取消超时任务

#### 2. 资源管理
**问题**：长时间运行的任务占用系统资源  
**解决**：限制任务执行时间，释放系统资源

#### 3. 生产环境保护
**问题**：生产环境的任务不应该无限期运行  
**解决**：强制设置超时时间，确保任务在合理时间内完成

## 使用方法

### 1. 创建任务时设置超时

在创建任务对话框中：

1. 找到 **"执行超时"** 配置项
2. 启用超时控制开关
3. 设置超时时间（秒）
4. 或使用快速设置按钮选择常用时长

**示例**：
```
执行超时: [启用]
超时时间: 1800 秒 (30分钟)

快速设置:
[5分钟] [10分钟] [30分钟] [1小时] [2小时]
```

### 2. 超时时间范围

- **最小值**: 60 秒（1 分钟）
- **最大值**: 86400 秒（24 小时）
- **步进值**: 60 秒
- **默认值**: 1800 秒（30 分钟）

### 3. 常用超时设置建议

| 任务类型 | 建议超时时间 | 说明 |
|---------|------------|------|
| 快速配置更新 | 5-10 分钟 | 简单的配置文件更新、服务重启等 |
| 常规部署 | 30 分钟 | 应用部署、配置同步等 |
| 大规模变更 | 1-2 小时 | 大量主机的变更操作 |
| 数据迁移/备份 | 2-4 小时 | 耗时较长的数据操作 |

### 4. 任务列表中的超时信息

任务列表中会显示：

**状态列**：
- 正常状态标签（pending/running/success/failed）
- 如果任务因超时而失败，会额外显示 **"超时"** 标签（黄色）

**耗时列**：
- 实际执行时长（秒）
- 超时限制时长（如果设置了超时）

**示例**：
```
状态: [失败] [超时]
耗时: 1835秒
      限制: 30分钟
```

## 技术实现

### 后端实现

#### 1. 数据模型

**backend/internal/model/ansible.go**:

```go
type AnsibleTask struct {
    // ... 其他字段
    TimeoutSeconds   int    `json:"timeout_seconds" gorm:"default:0;comment:超时时间(秒),0表示不限制"`
    IsTimedOut       bool   `json:"is_timed_out" gorm:"default:false;comment:是否超时"`
}

type TaskCreateRequest struct {
    // ... 其他字段
    TimeoutSeconds  int  `json:"timeout_seconds"` // 超时时间（秒），0表示不限制
}
```

#### 2. 任务执行器超时控制

**backend/internal/service/ansible/executor.go**:

核心实现：

```go
// 创建上下文（带超时控制）
var ctx context.Context
var cancel context.CancelFunc

if task.TimeoutSeconds > 0 {
    // 设置了超时时间
    ctx, cancel = context.WithTimeout(context.Background(), time.Duration(task.TimeoutSeconds)*time.Second)
    e.logger.Infof("Task %d: timeout set to %d seconds", taskID, task.TimeoutSeconds)
} else {
    // 没有设置超时时间
    ctx, cancel = context.WithCancel(context.Background())
}

// 执行命令
cmd := exec.CommandContext(ctx, "ansible-playbook", args...)

// 等待命令完成
err = cmd.Wait()

// 检查是否超时
isTimedOut := false
if ctx.Err() == context.DeadlineExceeded {
    isTimedOut = true
    e.logger.Warnf("Task %d exceeded timeout limit (%d seconds)", task.ID, task.TimeoutSeconds)
}

// 标记是否超时
task.IsTimedOut = isTimedOut

// 设置错误信息
if isTimedOut {
    errorMsg = fmt.Sprintf("任务执行超时（超过 %d 秒）", task.TimeoutSeconds)
}
```

#### 3. Context 超时机制

使用 Go 标准库的 `context.WithTimeout`：
- 自动在超时时间到达时取消上下文
- 通过 `context.DeadlineExceeded` 错误判断是否超时
- 传递给 `exec.CommandContext`，自动终止子进程

#### 4. 数据库设计

**迁移文件 (014_add_task_timeout.sql)**:

```sql
ALTER TABLE ansible_tasks ADD COLUMN timeout_seconds INTEGER DEFAULT 0;
ALTER TABLE ansible_tasks ADD COLUMN is_timed_out BOOLEAN DEFAULT false;

COMMENT ON COLUMN ansible_tasks.timeout_seconds IS '超时时间(秒),0表示不限制';
COMMENT ON COLUMN ansible_tasks.is_timed_out IS '是否超时';

-- 添加索引，方便查询超时任务
CREATE INDEX idx_ansible_tasks_is_timed_out ON ansible_tasks (is_timed_out) WHERE is_timed_out = true;
```

### 前端实现

#### 1. 任务表单

**frontend/src/views/ansible/TaskCenter.vue**:

数据结构：
```javascript
const taskForm = reactive({
  // ... 其他字段
  timeout_seconds: 1800  // 默认 30 分钟
})

const timeoutEnabled = ref(false)
```

超时配置界面：
```html
<el-form-item label="执行超时">
  <el-switch 
    v-model="timeoutEnabled" 
    active-text="启用超时控制"
    inactive-text="不限制"
  />
</el-form-item>

<el-form-item v-if="timeoutEnabled" label="超时时间">
  <el-input-number 
    v-model="taskForm.timeout_seconds" 
    :min="60" 
    :max="86400"
    :step="60"
  />
  <span>秒（{{ formatDuration(taskForm.timeout_seconds) }}）</span>
</el-form-item>

<!-- 快速设置按钮 -->
<el-form-item v-if="timeoutEnabled" label="快速设置">
  <el-button-group>
    <el-button size="small" @click="taskForm.timeout_seconds = 300">5分钟</el-button>
    <el-button size="small" @click="taskForm.timeout_seconds = 600">10分钟</el-button>
    <el-button size="small" @click="taskForm.timeout_seconds = 1800">30分钟</el-button>
    <el-button size="small" @click="taskForm.timeout_seconds = 3600">1小时</el-button>
    <el-button size="small" @click="taskForm.timeout_seconds = 7200">2小时</el-button>
  </el-button-group>
</el-form-item>
```

#### 2. 时长格式化

```javascript
const formatDuration = (seconds) => {
  if (!seconds) return '不限制'
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return minutes > 0 ? `${hours}小时${minutes}分钟` : `${hours}小时`
}
```

#### 3. 任务列表展示

状态列显示超时标签：
```html
<el-table-column label="状态" width="150">
  <template #default="{ row }">
    <el-tag :type="getStatusType(row.status)">
      {{ getStatusText(row.status) }}
    </el-tag>
    <el-tag v-if="row.is_timed_out" type="warning" size="small">
      超时
    </el-tag>
  </template>
</el-table-column>
```

耗时列显示超时限制：
```html
<el-table-column label="耗时" width="150">
  <template #default="{ row }">
    <span>{{ row.duration ? `${row.duration}秒` : '-' }}</span>
    <div v-if="row.timeout_seconds > 0" style="font-size: 12px; color: #909399">
      限制: {{ formatDuration(row.timeout_seconds) }}
    </div>
  </template>
</el-table-column>
```

## 使用示例

### 示例 1：创建带超时的任务

**请求**：
```json
{
  "name": "更新 Nginx 配置",
  "template_id": 1,
  "inventory_id": 2,
  "extra_vars": {
    "nginx_port": "8080"
  },
  "timeout_seconds": 600
}
```

**响应**：
```json
{
  "code": 200,
  "data": {
    "id": 123,
    "name": "更新 Nginx 配置",
    "status": "pending",
    "timeout_seconds": 600,
    "is_timed_out": false
  }
}
```

### 示例 2：任务超时场景

**任务执行过程**：
```
1. 任务开始执行 (started_at: 2025-11-03 10:00:00)
2. 设置的超时时间: 600 秒 (10分钟)
3. 实际执行时间: 625 秒
4. 在 10:10:00 时，context 超时触发
5. 任务被自动取消
```

**任务最终状态**：
```json
{
  "id": 123,
  "status": "failed",
  "is_timed_out": true,
  "error_msg": "任务执行超时（超过 600 秒）",
  "duration": 625,
  "timeout_seconds": 600
}
```

### 示例 3：不设置超时

如果 `timeout_seconds` 为 0 或不提供，任务将不受超时限制：

**请求**：
```json
{
  "name": "长时间数据迁移",
  "template_id": 5,
  "inventory_id": 3,
  "timeout_seconds": 0
}
```

任务可以运行任意时长，直到完成或手动取消。

## 最佳实践

### 1. 什么时候应该设置超时

**推荐设置**：
- ✅ 所有生产环境任务
- ✅ 网络依赖较多的任务
- ✅ 涉及外部服务调用的任务
- ✅ 批量操作任务

**可以不设置**：
- 开发/测试环境的任务
- 已经非常了解执行时长的任务
- 明确需要长时间运行的任务（如大规模数据迁移）

### 2. 如何设置合理的超时时间

**计算公式**：
```
超时时间 = 预估执行时间 × 2 + 缓冲时间

缓冲时间建议:
- 小任务(<5分钟): 额外 2-5 分钟
- 中型任务(5-30分钟): 额外 5-15 分钟
- 大型任务(>30分钟): 额外 20-30 分钟
```

**示例**：
- 预估 5 分钟的任务 → 设置 12-15 分钟超时
- 预估 15 分钟的任务 → 设置 35-40 分钟超时
- 预估 1 小时的任务 → 设置 2.5 小时超时

### 3. 超时后如何处理

**步骤**：
1. 查看任务日志，分析超时原因
2. 评估是否需要调整超时时间
3. 检查目标主机状态（网络、资源等）
4. 修复问题后重新执行任务

**常见超时原因**：
- 网络延迟或不稳定
- 目标主机资源不足（CPU、内存、磁盘）
- Playbook 逻辑问题（死循环、等待超时）
- 依赖的外部服务响应缓慢

### 4. 与其他功能配合使用

**与 Dry Run 配合**：
- Dry Run 任务也应该设置超时
- 通常 Dry Run 执行更快，可以设置更短的超时

**与分批执行配合**：
- 超时时间应该考虑所有批次的总执行时间
- 计算公式：`每批执行时间 × 批次数量 × 1.5`

**与重试策略配合**：
- 超时导致的失败会触发重试（如果启用）
- 重试时超时时间保持不变
- 注意：多次重试可能导致总时长过长

## 注意事项

### 1. 超时机制的局限性

**不能保证的情况**：
- 任务已经在目标主机上执行的操作不会回滚
- 超时只是终止 Ansible 进程，目标主机上可能还有残留进程
- 超时后立即重新执行可能会遇到资源冲突

**建议**：
- 使用幂等的 Playbook，确保重复执行不会产生副作用
- 超时后检查目标主机状态，确认清理完成后再重新执行
- 对于非幂等操作，使用 Dry Run 先验证

### 2. 性能考虑

**超时检测开销**：
- 非常小，使用 Go 的 context 机制，几乎无性能开销
- 不会影响正常任务的执行效率

**数据库索引**：
- 已为 `is_timed_out` 字段创建部分索引
- 只索引超时任务，减小索引大小
- 查询超时任务性能良好

### 3. 日志记录

**超时日志示例**：
```
[WARN] Task 123 exceeded timeout limit (600 seconds)
[INFO] Task 123 completed, status: failed, error: 任务执行超时（超过 600 秒）
```

**日志保留**：
- 超时任务的日志会完整保留
- 可以查看超时前的执行情况
- 有助于诊断超时原因

### 4. 与手动取消的区别

| 操作 | 触发方式 | is_timed_out | error_msg |
|-----|---------|--------------|-----------|
| 超时自动取消 | 系统自动 | true | "任务执行超时（超过 X 秒）" |
| 手动取消 | 用户点击 | false | "任务已被用户取消" |

## 监控和统计

### 查询超时任务

**SQL 查询**：
```sql
SELECT id, name, status, duration, timeout_seconds, error_msg
FROM ansible_tasks
WHERE is_timed_out = true
ORDER BY created_at DESC
LIMIT 10;
```

### 统计超时率

**按时间段统计**：
```sql
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total_tasks,
    SUM(CASE WHEN is_timed_out THEN 1 ELSE 0 END) as timed_out_tasks,
    ROUND(SUM(CASE WHEN is_timed_out THEN 1 ELSE 0 END)::numeric / COUNT(*) * 100, 2) as timeout_rate
FROM ansible_tasks
WHERE created_at >= NOW() - INTERVAL '7 days'
GROUP BY DATE(created_at)
ORDER BY date DESC;
```

## 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Ansible Dry Run 模式](./ansible-dry-run-mode.md)
- [Ansible 分阶段执行](./ansible-phased-execution.md)
- [Ansible 任务执行前置检查](./ansible-preflight-checks.md)

## 更新日志

### v2.27.0 (2025-11-03)

**后端实现**（已完成）：
- ✅ 数据模型：添加 `TimeoutSeconds` 和 `IsTimedOut` 字段
- ✅ Context 超时控制：使用 `context.WithTimeout`
- ✅ 超时检测：通过 `context.DeadlineExceeded` 判断
- ✅ 超时标记：自动设置 `is_timed_out` 字段
- ✅ 错误信息：生成友好的超时错误消息
- ✅ 数据库迁移：添加字段和索引

**前端实现**（已完成）：
- ✅ 超时配置界面：开关和输入框
- ✅ 快速设置按钮：常用时长选择
- ✅ 时长格式化：智能显示时、分、秒
- ✅ 任务列表展示：状态标签和超时限制
- ✅ 超时状态标识：黄色 "超时" 标签

**核心功能**：
1. **灵活超时配置**: 60秒 - 24小时
2. **自动取消机制**: Context 超时自动终止
3. **超时状态追踪**: is_timed_out 字段
4. **友好的 UI**: 快速设置 + 可读时长显示
5. **完整的监控**: 索引支持超时任务查询

