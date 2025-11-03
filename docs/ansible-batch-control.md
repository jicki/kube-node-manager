# Ansible 批次执行控制功能使用指南

## 功能概述

批次执行控制功能为分批执行（金丝雀/灰度发布）提供了完整的运行时控制能力，允许运维人员在任务执行过程中进行人工干预，确保变更的安全性和可控性。

## 实施日期

2025-11-03

## 功能特性

### 核心能力

- ✅ **暂停批次执行**：在当前批次完成后暂停，等待人工确认
- ✅ **继续批次执行**：人工确认后继续执行下一批次
- ✅ **停止批次执行**：停止所有剩余批次，已完成的不回滚
- ✅ **实时状态显示**：显示当前批次状态（running/paused/stopped）
- ✅ **批次进度跟踪**：显示当前批次和总批次数
- ✅ **智能按钮显示**：根据任务和批次状态动态显示控制按钮

### 使用场景

#### 1. 金丝雀发布验证
**场景**：首批主机执行完成后，需要验证变更效果  
**操作流程**：
```
1. 配置分批执行（例如：20% + 每批后暂停）
2. 启动任务，首批20%主机开始执行
3. 首批完成后，任务自动暂停（batch_status = paused）
4. 运维人员验证首批主机的变更效果
   - 检查服务状态
   - 查看监控指标
   - 验证业务功能
5. 确认无问题后，点击"继续批次"
6. 重复步骤3-5，直到所有批次完成
```

#### 2. 紧急情况处理
**场景**：执行过程中发现问题，需要立即停止  
**操作流程**：
```
1. 任务正在执行第2批次（共5批次）
2. 监控发现异常（例如：服务响应时间增加）
3. 立即点击"停止批次"
4. 系统停止所有剩余批次（第3、4、5批次不再执行）
5. 第1、2批次已完成的主机保持变更状态
6. 排查问题，修复后重新执行或回滚
```

#### 3. 灰度发布策略
**场景**：大规模变更，需要逐步扩大影响范围  
**操作流程**：
```
1. 配置多批次策略：5% → 10% → 20% → 30% → 35%
2. 每批执行后自动暂停
3. 观察一段时间（例如：30分钟）
4. 确认无问题后继续下一批次
5. 发现问题立即停止，限制影响范围
```

#### 4. 分时段执行
**场景**：避免业务高峰期执行变更  
**操作流程**：
```
1. 启动任务，执行第一批次
2. 接近业务高峰期，点击"暂停批次"
3. 等待业务高峰期结束
4. 业务低峰期，点击"继续批次"
5. 重复步骤2-4，完成所有批次
```

## 技术实现

### 后端实现

#### 1. 数据模型（已存在）

```go
type AnsibleTask struct {
    // ...
    BatchConfig      *BatchExecutionConfig `json:"batch_config"`
    CurrentBatch     int                   `json:"current_batch"`
    TotalBatches     int                   `json:"total_batches"`
    BatchStatus      string                `json:"batch_status"` // running/paused/stopped/completed
    // ...
}
```

**批次状态说明**：
- `running` - 批次正在执行
- `paused` - 已暂停，等待继续
- `stopped` - 已停止，不会继续执行
- `completed` - 所有批次已完成

#### 2. 服务层方法

**backend/internal/service/ansible/service.go**:

```go
// PauseBatchExecution 暂停批次执行
func (s *Service) PauseBatchExecution(taskID uint) error {
    // 验证任务状态和批次配置
    // 更新 batch_status = 'paused'
}

// ContinueBatchExecution 继续批次执行
func (s *Service) ContinueBatchExecution(taskID uint) error {
    // 验证任务状态
    // 更新 batch_status = 'running'
    // current_batch++
    // 触发执行器继续执行
}

// StopBatchExecution 停止批次执行
func (s *Service) StopBatchExecution(taskID uint) error {
    // 验证任务状态
    // 更新 batch_status = 'stopped'
    // 更新 task_status = 'cancelled'
    // 取消执行器
}
```

#### 3. 执行器方法

**backend/internal/service/ansible/executor.go**:

```go
// ContinueBatchExecution 继续批次执行
func (e *TaskExecutor) ContinueBatchExecution(taskID uint) error {
    // 获取任务
    // 检查批次配置
    // 检查是否还有剩余批次
    // 如果所有批次已完成，标记任务完成
    // 否则，记录日志，等待下一批次执行
}
```

#### 4. API 端点

**backend/cmd/main.go**:

```go
ansible.POST("/tasks/:id/pause-batch", handlers.Ansible.PauseBatch)
ansible.POST("/tasks/:id/continue-batch", handlers.Ansible.ContinueBatch)
ansible.POST("/tasks/:id/stop-batch", handlers.Ansible.StopBatch)
```

### 前端实现

#### 1. API 封装

**frontend/src/api/ansible.js**:

```javascript
// 暂停批次执行
export function pauseBatch(id)

// 继续批次执行
export function continueBatch(id)

// 停止批次执行
export function stopBatch(id)
```

#### 2. UI 组件

**frontend/src/views/ansible/TaskCenter.vue**:

**批次控制按钮**（根据状态动态显示）:

```html
<!-- 当任务运行中且启用了分批执行 -->
<template v-if="row.status === 'running' && row.batch_config?.enabled">
  <!-- 批次运行中，显示"暂停批次"按钮 -->
  <el-button v-if="row.batch_status === 'running'" 
             type="warning" 
             @click="handlePauseBatch(row)">
    暂停批次
  </el-button>
  
  <!-- 批次已暂停，显示"继续批次"按钮 -->
  <el-button v-if="row.batch_status === 'paused'" 
             type="success" 
             @click="handleContinueBatch(row)">
    继续批次
  </el-button>
  
  <!-- 始终显示"停止批次"按钮 -->
  <el-button type="danger" 
             @click="handleStopBatch(row)">
    停止批次
  </el-button>
</template>
```

**批次状态显示**:

```html
<div v-if="row.batch_config?.enabled">
  批次: {{ row.current_batch }}/{{ row.total_batches }}
  <el-tag v-if="row.batch_status === 'paused'" 
          type="warning" 
          size="small">
    已暂停
  </el-tag>
</div>
```

#### 3. 处理方法

```javascript
// 暂停批次
const handlePauseBatch = async (row) => {
  await ElMessageBox.confirm(
    `确定要暂停任务 "${row.name}" 的批次执行吗？当前批次完成后将暂停。`,
    '暂停批次',
    { type: 'warning' }
  )
  await ansibleAPI.pauseBatch(row.id)
  ElMessage.success('批次执行已暂停')
  loadTasks() // 刷新列表
}

// 继续批次
const handleContinueBatch = async (row) => {
  await ElMessageBox.confirm(
    `确定要继续任务 "${row.name}" 的批次执行吗？将继续执行下一批次。`,
    '继续批次',
    { type: 'success' }
  )
  await ansibleAPI.continueBatch(row.id)
  ElMessage.success('批次执行已继续')
  loadTasks() // 刷新列表
}

// 停止批次
const handleStopBatch = async (row) => {
  await ElMessageBox.confirm(
    `确定要停止任务 "${row.name}" 的所有剩余批次吗？已完成的批次不会回滚，但剩余批次将不再执行。`,
    '停止批次',
    { type: 'error' }
  )
  await ansibleAPI.stopBatch(row.id)
  ElMessage.success('批次执行已停止')
  loadTasks()
  loadStatistics() // 刷新统计信息
}
```

## 使用示例

### 示例 1：标准金丝雀发布流程

#### 1. 创建分批执行任务

```json
{
  "name": "应用版本升级 v2.0",
  "template_id": 10,
  "inventory_id": 5,
  "batch_config": {
    "enabled": true,
    "batch_percent": 20,
    "pause_after_batch": true,
    "failure_threshold": 2,
    "max_batch_fail_rate": 10
  }
}
```

#### 2. 任务执行时间线

```
T0: 启动任务
    - 状态: running
    - 批次状态: running
    - 当前批次: 1/5
    - 执行: 20% 主机 (4/20)

T1: 第1批次完成
    - 状态: running
    - 批次状态: paused （自动暂停）
    - 当前批次: 1/5
    
T2: 运维人员验证 (15分钟)
    - 检查应用日志
    - 验证监控指标
    - 测试核心功能
    
T3: 确认无问题，点击"继续批次"
    - 状态: running
    - 批次状态: running
    - 当前批次: 2/5
    - 执行: 下一个 20% 主机 (4/20)

T4: 第2批次完成
    - 状态: running
    - 批次状态: paused
    - 当前批次: 2/5
    
T5: 继续... 直到所有批次完成

T6: 所有批次完成
    - 状态: success
    - 批次状态: completed
    - 当前批次: 5/5
```

### 示例 2：紧急停止场景

#### 情况：执行过程中发现严重问题

```
当前状态:
- 任务正在执行第3批次
- 已完成: 第1批次 (20%)、第2批次 (20%)
- 当前: 第3批次 (20%) 正在执行
- 剩余: 第4批次 (20%)、第5批次 (20%)

问题发现:
- 监控发现CPU使用率异常飙升
- 部分服务响应超时
- 需要立即停止变更

操作:
1. 点击"停止批次"按钮
2. 确认停止操作

结果:
- 第3批次被中断（如果 Ansible 任务可以被杀死）
- 第4、5批次不会执行
- 任务状态: cancelled
- 批次状态: stopped
- 当前批次: 3/5

后续处理:
- 40% 的主机 (第1、2批次) 已升级到 v2.0
- 60% 的主机仍然是旧版本
- 需要决定：
  a. 回滚已升级的40%主机
  b. 修复问题后继续升级剩余60%
```

### 示例 3：API 调用示例

```bash
# 1. 暂停批次执行
curl -X POST http://your-server/api/v1/ansible/tasks/123/pause-batch \
  -H "Authorization: Bearer YOUR_TOKEN"

# 响应
{
  "code": 200,
  "message": "Batch execution paused"
}

# 2. 继续批次执行
curl -X POST http://your-server/api/v1/ansible/tasks/123/continue-batch \
  -H "Authorization: Bearer YOUR_TOKEN"

# 响应
{
  "code": 200,
  "message": "Batch execution continued"
}

# 3. 停止批次执行
curl -X POST http://your-server/api/v1/ansible/tasks/123/stop-batch \
  -H "Authorization: Bearer YOUR_TOKEN"

# 响应
{
  "code": 200,
  "message": "Batch execution stopped"
}

# 4. 查询任务状态
curl -X GET http://your-server/api/v1/ansible/tasks/123 \
  -H "Authorization: Bearer YOUR_TOKEN"

# 响应
{
  "code": 200,
  "data": {
    "id": 123,
    "name": "应用版本升级 v2.0",
    "status": "running",
    "batch_config": {
      "enabled": true,
      "batch_percent": 20,
      "pause_after_batch": true
    },
    "current_batch": 2,
    "total_batches": 5,
    "batch_status": "paused",
    "hosts_total": 20,
    "hosts_ok": 8,
    "hosts_failed": 0
  }
}
```

## 安全机制

### 1. 状态验证

所有批次控制操作都会验证：
- 任务必须处于 `running` 状态
- 任务必须启用了分批执行
- 暂停操作：批次状态必须是 `running`
- 继续操作：批次状态必须是 `paused`

### 2. 权限控制

- 所有批次控制操作都需要管理员权限
- 通过 `checkAdminPermission` 中间件验证

### 3. 操作确认

前端所有控制操作都需要用户二次确认：
- 暂停：警告类型（warning）
- 继续：成功类型（success）
- 停止：错误类型（error），最严格

### 4. 日志记录

所有批次控制操作都会记录日志：
```
INFO: Batch execution paused for task 123
INFO: Batch execution continued for task 123, moving to batch 3/5
INFO: Batch execution stopped for task 123 at batch 2/5
```

## 最佳实践

### 1. 批次策略设计

**推荐策略**：
```
小规模（< 20台）: 不使用分批，或 50% + 50%
中规模（20-100台）: 20% → 30% → 50%
大规模（> 100台）: 5% → 10% → 20% → 30% → 35%
```

**关键参数**：
- `pause_after_batch: true` - 每批后手动确认
- `failure_threshold` - 根据主机数量设置（建议 < 5%）
- `max_batch_fail_rate: 10-20%` - 单批失败率阈值

### 2. 验证检查清单

每批执行后，建议检查：
- ✅ 服务状态（systemctl status）
- ✅ 应用日志（无ERROR）
- ✅ 监控指标（CPU、内存、磁盘、网络）
- ✅ 业务功能（核心接口测试）
- ✅ 依赖服务（数据库、缓存、消息队列连接）

### 3. 紧急响应流程

**发现问题时**：
1. 立即点击"停止批次"
2. 记录当前批次和已完成批次
3. 保留现场（日志、监控数据）
4. 通知相关人员
5. 制定回滚或修复方案

**问题分类处理**：
- **P0（严重）**：立即停止 + 回滚所有已变更主机
- **P1（重要）**：停止 + 评估影响 + 修复后继续
- **P2（一般）**：暂停 + 修复问题 + 继续执行
- **P3（轻微）**：记录问题 + 继续执行 + 后续修复

### 4. 文档记录

每次分批执行建议记录：
- 开始时间和结束时间
- 每批次的执行时间
- 暂停和继续的时间点
- 验证结果
- 发现的问题（如果有）
- 最终状态

## 注意事项

### 1. 暂停的实际意义

**重要**：当前实现中，"暂停"是在**批次之间**暂停，而不是在批次执行**过程中**暂停。

- ✅ 批次完成后暂停：支持
- ❌ 批次执行中途暂停：不支持

**原因**：Ansible 的 `serial` 参数控制批次大小，但无法在单个批次执行中途暂停。

### 2. 停止的影响范围

**停止批次**操作：
- ✅ 已完成的批次：保持变更状态（不回滚）
- ⚠️ 当前批次：尝试中断（取决于 Ansible 任务是否可以被 kill）
- ✅ 剩余批次：不会执行

### 3. 状态同步

- 前端每5秒自动刷新任务列表
- 批次状态变更后，刷新列表即可看到最新状态
- 如果需要实时性更高，可以通过 WebSocket 推送

### 4. 并发控制

- 同一个任务，不能同时执行多个批次控制操作
- 如果已经处于暂停状态，再次暂停会返回错误
- 如果已经处于运行状态，再次继续会返回错误

## 常见问题

### Q1：暂停后，当前批次会立即停止吗？

**A**：不会。"暂停"是在当前批次**完成后**暂停，不会中断正在执行的批次。如果需要立即停止，请使用"停止批次"功能。

### Q2：停止批次后，可以重新启动吗？

**A**：不可以。停止批次后，任务状态变为 `cancelled`，无法恢复。如果需要继续执行剩余主机，需要创建新任务（可以使用"重新执行"功能）。

### Q3：如何实现自动继续（无需手动确认）？

**A**：在创建任务时，设置 `pause_after_batch: false`。这样每批执行完成后会自动继续下一批次。

### Q4：批次执行失败率超过阈值会怎样？

**A**：如果单批失败率超过 `max_batch_fail_rate`，或总失败主机数超过 `failure_threshold`，任务会自动停止，不会继续执行后续批次。

### Q5：批次控制和普通取消有什么区别？

**A**：
- **普通取消**：适用于未启用分批执行的任务，取消整个任务
- **停止批次**：适用于分批执行的任务，停止剩余批次，已完成批次保持
- **暂停批次**：临时暂停，可以继续；停止批次是永久停止，不可继续

## 相关文档

- [Ansible 分阶段执行（金丝雀/灰度发布）](./ansible-batch-execution.md)
- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Ansible Dry Run 模式](./ansible-dry-run-mode.md)

## 更新日志

### v2.24.0 (2025-11-03)

**后端实现**：
- ✅ `PauseBatchExecution` - 暂停批次执行
- ✅ `ContinueBatchExecution` - 继续批次执行
- ✅ `StopBatchExecution` - 停止批次执行
- ✅ 批次状态管理和验证
- ✅ API 端点注册
- ✅ 执行器批次控制逻辑

**前端实现**：
- ✅ 批次控制 API 封装
- ✅ 智能按钮显示（根据状态动态显示）
- ✅ 操作确认对话框（warning/success/error 类型）
- ✅ 批次状态实时显示
- ✅ 批次进度显示（当前批次/总批次）

**核心功能**：
1. **暂停批次**：
   - 状态验证（must be running）
   - 批次状态检查（must not be paused）
   - 更新 `batch_status` 为 `paused`
   - 前端黄色按钮 + 警告确认

2. **继续批次**：
   - 状态验证（must be running & paused）
   - 批次号递增（current_batch++）
   - 更新 `batch_status` 为 `running`
   - 触发执行器继续
   - 前端绿色按钮 + 成功确认

3. **停止批次**：
   - 状态验证
   - 更新 `batch_status` 为 `stopped`
   - 更新 `task_status` 为 `cancelled`
   - 取消执行器
   - 前端红色按钮 + 错误确认

