# 批量操作优化：单副本与多副本环境分析

## 📋 概述

此文档深入分析批量操作优化系统在单副本和多副本环境下的设计、实现原理、潜在问题以及解决方案。

---

## 🏗️ 整体架构设计

### 1. 双模式架构

系统采用**双模式架构**，根据部署环境自动选择最优的进度追踪机制：

```
┌─────────────────────────────────────────────────────┐
│         Progress Service (进度服务)                   │
│                                                       │
│  ┌─────────────────────┐  ┌──────────────────────┐  │
│  │   Memory Mode       │  │   Database Mode      │  │
│  │   (内存模式)         │  │   (数据库模式)        │  │
│  │   单副本环境         │  │   多副本环境          │  │
│  └─────────────────────┘  └──────────────────────┘  │
│           ▲                         ▲                │
│           │                         │                │
│           └─────────┬───────────────┘                │
│                     │                                │
│            由配置决定（自动选择）                      │
└─────────────────────────────────────────────────────┘
```

### 2. 模式选择机制

**配置文件**（`config.yaml`）：
```yaml
progress:
  enable_database: false  # false = 单副本内存模式, true = 多副本数据库模式
```

**代码实现**（`internal/service/services.go:187-191`）：
```go
// 检查是否启用数据库模式（用于多副本环境）
if cfg.Progress.EnableDatabase {
    progressSvc.EnableDatabaseMode(db)
    logger.Infof("Progress service database mode enabled for multi-replica support")
}
```

**判断逻辑**（`internal/service/progress/progress.go:794-797`）：
```go
// 如果启用了数据库模式，使用数据库进度服务
if s.useDatabase && s.dbProgressService != nil {
    return s.dbProgressService.ProcessBatchWithProgress(ctx, taskID, action, nodeNames, userID, maxConcurrency, processor)
}
// 否则使用原有的内存模式
```

---

## 🔍 单副本环境（内存模式）

### 1. 工作原理

在单副本环境中，所有进度信息都存储在内存中，通过 WebSocket 实时推送给前端。

#### 数据流程

```
1. 用户触发批量操作
   ↓
2. 后端创建任务（存储在内存 map 中）
   ↓
3. 并发处理节点（Goroutine + 信号量控制）
   ↓
4. 每处理一个节点，更新内存中的任务状态
   ↓
5. 通过 WebSocket 实时推送进度给前端
   ↓
6. 任务完成，更新内存状态并推送完成消息
```

#### 核心数据结构

```go
type Service struct {
    // 存储用户连接 map[userID]map[*Connection]bool
    connections map[uint]map[*Connection]bool
    
    // 存储任务进度 map[taskID]*TaskProgress
    tasks map[string]*TaskProgress
    
    // 完成任务的消息队列，用于重连时恢复
    completedTasks map[uint][]ProgressMessage
    
    // 内存模式
    useDatabase bool  // false
}
```

#### 任务进度结构

```go
type TaskProgress struct {
    TaskID        string
    Action        string
    Current       int              // 当前完成数量
    Total         int              // 总数量
    IsRunning     bool
    Completed     bool
    SuccessNodes  []string         // 成功节点列表
    FailedNodes   []model.NodeError // 失败节点列表
    UserID        uint
}
```

### 2. 并发处理逻辑

```go
// 使用信号量控制并发
semaphore := make(chan struct{}, maxConcurrency)
var wg sync.WaitGroup
var mu sync.Mutex
var failedNodes []model.NodeError
var successNodes []string

for i, nodeName := range nodeNames {
    wg.Add(1)
    go func(index int, node string) {
        defer wg.Done()
        
        // 获取信号量（控制并发数）
        semaphore <- struct{}{}
        defer func() { <-semaphore }()
        
        // 原子性地更新当前处理索引
        mu.Lock()
        processed++
        currentIndex := processed
        mu.Unlock()
        
        // 发送进度消息
        s.UpdateProgress(taskID, currentIndex, node, userID)
        
        // 处理节点
        if err := processor.ProcessNode(ctx, node, index); err != nil {
            // 失败：记录失败节点
            mu.Lock()
            failedNodes = append(failedNodes, model.NodeError{
                NodeName: node,
                Error:    err.Error(),
            })
            // 实时更新任务的失败列表
            s.tasks[taskID].FailedNodes = failedNodes
            mu.Unlock()
        } else {
            // 成功：记录成功节点
            mu.Lock()
            successNodes = append(successNodes, node)
            s.tasks[taskID].SuccessNodes = successNodes
            mu.Unlock()
        }
    }(i, nodeName)
}

wg.Wait()  // 等待所有节点处理完成
```

### 3. WebSocket 推送机制

```go
func (s *Service) sendToUser(userID uint, message ProgressMessage) {
    // 获取用户的所有 WebSocket 连接
    s.connMutex.RLock()
    userConns := s.connections[userID]
    s.connMutex.RUnlock()
    
    // 推送消息到所有连接
    for conn := range userConns {
        select {
        case conn.send <- message:
            // 发送成功
        case <-time.After(3 * time.Second):
            // 超时：对于重要消息，保存到队列中
            if message.Type == "complete" || message.Type == "error" {
                s.queueCompletionMessage(userID, message)
            }
        }
    }
}
```

### 4. 优势

- ✅ **性能极佳**：无数据库读写，所有操作在内存中完成
- ✅ **实时性强**：直接通过 WebSocket 推送，延迟极低（< 100ms）
- ✅ **简单高效**：无需额外的轮询和同步机制
- ✅ **资源占用低**：只需要维护少量内存状态

### 5. 局限性

- ❌ **单点故障**：进程重启后所有进度信息丢失
- ❌ **不支持多副本**：多个副本之间无法共享进度状态
- ❌ **连接断开风险**：如果用户 WebSocket 断开且任务完成，可能丢失完成消息
  - *缓解措施*：`completedTasks` 队列会保存最近的完成消息，重连后恢复

---

## 🌐 多副本环境（数据库模式）

### 1. 工作原理

在多副本环境中，进度信息持久化到数据库（PostgreSQL），所有副本通过数据库共享状态，并通过轮询机制同步消息。

#### 数据流程

```
1. 用户触发批量操作（可能落到任意副本）
   ↓
2. 处理副本创建数据库任务记录
   ↓
3. 并发处理节点（Goroutine + 信号量控制）
   ↓
4. 每处理一个节点，更新数据库中的任务状态
   ↓
5. 将进度消息写入数据库消息表（processed = false）
   ↓
6. 所有副本定期轮询数据库，获取未处理的消息
   ↓
7. 副本将消息推送给连接到该副本的用户
   ↓
8. 推送成功后标记消息为已处理（processed = true）
```

#### 关键设计问题：多副本场景

**场景 A：用户连接到副本 A，任务在副本 B 执行**

```
用户 ──WebSocket──> 副本 A （监听进度）
                       ▲
                       │ (轮询消息)
                       │
                   数据库
                       ▲
                       │ (写入进度)
                       │
                    副本 B （执行任务）
```

**解决方案**：
- 副本 B 执行任务时，将每个进度消息写入数据库
- 副本 A 每 500ms 轮询一次数据库，获取新的进度消息
- 副本 A 推送消息给已连接的用户
- 推送成功后，副本 A 将消息标记为 `processed = true`

### 2. 数据库表结构

#### ProgressTask 表

```sql
CREATE TABLE progress_tasks (
    id SERIAL PRIMARY KEY,
    task_id VARCHAR(255) UNIQUE NOT NULL,
    user_id INTEGER NOT NULL,
    action VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,         -- running, completed, failed
    current INTEGER DEFAULT 0,
    total INTEGER NOT NULL,
    progress NUMERIC(5,2) DEFAULT 0,
    current_node VARCHAR(255),
    message TEXT,
    error_msg TEXT,
    success_nodes JSONB DEFAULT '[]',    -- 成功节点列表 (JSON)
    failed_nodes JSONB DEFAULT '[]',     -- 失败节点列表 (JSON)
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    deleted_at TIMESTAMP
);
```

#### ProgressMessage 表

```sql
CREATE TABLE progress_messages (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    task_id VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,           -- progress, complete, error
    action VARCHAR(50) NOT NULL,
    current INTEGER DEFAULT 0,
    total INTEGER NOT NULL,
    progress NUMERIC(5,2) DEFAULT 0,
    current_node VARCHAR(255),
    message TEXT,
    error_msg TEXT,
    success_nodes JSONB DEFAULT '[]',    -- 成功节点列表 (JSON)
    failed_nodes JSONB DEFAULT '[]',     -- 失败节点列表 (JSON)
    processed BOOLEAN DEFAULT FALSE,     -- 是否已推送给用户
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_progress_messages_unprocessed 
    ON progress_messages(user_id, processed, created_at) 
    WHERE processed = FALSE;
```

### 3. 核心服务结构

```go
type DatabaseProgressService struct {
    db                *gorm.DB
    logger            *logger.Logger
    wsService         *Service           // 原有的 WebSocket 服务
    stopPolling       chan struct{}
    pollingWg         sync.WaitGroup
    lastProcessedTime time.Time
    pollInterval      time.Duration      // 500ms
}
```

### 4. 关键机制

#### 4.1 任务创建

```go
func (dps *DatabaseProgressService) CreateTask(taskID, action string, total int, userID uint) error {
    task := &model.ProgressTask{
        TaskID:    taskID,
        UserID:    userID,
        Action:    action,
        Status:    model.TaskStatusRunning,
        Total:     total,
        Progress:  0,
        CreatedAt: time.Now(),
    }
    return dps.db.Create(task).Error
}
```

#### 4.2 进度更新

```go
func (dps *DatabaseProgressService) UpdateProgress(taskID string, current int, currentNode string, userID uint) error {
    // 1. 更新任务记录
    var task model.ProgressTask
    if err := dps.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
        return err
    }
    
    task.UpdateProgress(current, currentNode)
    task.Message = fmt.Sprintf("正在处理节点 %s (%d/%d)", currentNode, current, task.Total)
    
    if err := dps.db.Save(&task).Error; err != nil {
        return err
    }
    
    // 2. 创建进度消息（供其他副本读取）
    return dps.createProgressMessage(&task, "progress")
}
```

#### 4.3 消息轮询（核心）

```go
func (dps *DatabaseProgressService) startMessagePolling() {
    ticker := time.NewTicker(500 * time.Millisecond)  // 每 500ms 轮询一次
    defer ticker.Stop()
    
    for {
        select {
        case <-dps.stopPolling:
            return
        case <-ticker.C:
            dps.processUnsentMessages()
        }
    }
}
```

#### 4.4 未发送消息处理

```go
func (dps *DatabaseProgressService) processUnsentMessages() {
    var messages []model.ProgressMessage
    
    // 优先处理完成和错误消息，然后处理普通进度消息
    query := dps.db.Where("processed = ? AND created_at > ?", false, dps.lastProcessedTime).
        Order("CASE WHEN type IN ('complete', 'error') THEN 0 ELSE 1 END, created_at ASC").
        Limit(100)  // 批量处理，避免一次性读取过多
    
    if err := query.Find(&messages).Error; err != nil {
        return
    }
    
    for _, msg := range messages {
        // 解析成功和失败节点列表（JSON → Go 结构）
        var successNodes []string
        var failedNodes []model.NodeError
        json.Unmarshal([]byte(msg.SuccessNodes), &successNodes)
        json.Unmarshal([]byte(msg.FailedNodes), &failedNodes)
        
        // 转换为 WebSocket 消息格式
        wsMessage := ProgressMessage{
            TaskID:       msg.TaskID,
            Type:         msg.Type,
            Action:       msg.Action,
            Current:      msg.Current,
            Total:        msg.Total,
            Progress:     msg.Progress,
            CurrentNode:  msg.CurrentNode,
            SuccessNodes: successNodes,
            FailedNodes:  failedNodes,
            Message:      msg.Message,
            Error:        msg.ErrorMsg,
            Timestamp:    msg.CreatedAt,
        }
        
        // 检查是否有连接的用户
        hasConnection := dps.wsService.hasUserConnection(msg.UserID)
        
        if hasConnection {
            // 推送给用户
            dps.wsService.sendToUser(msg.UserID, wsMessage)
        } else if msg.Type == "complete" || msg.Type == "error" {
            // 重要消息：等待一下再重试
            time.Sleep(100 * time.Millisecond)
            dps.wsService.sendToUser(msg.UserID, wsMessage)
        }
        
        // 标记为已处理
        dps.db.Model(&msg).Update("processed", true)
    }
    
    // 更新最后处理时间（避免重复处理）
    if len(messages) > 0 {
        dps.lastProcessedTime = messages[len(messages)-1].CreatedAt
    }
}
```

#### 4.5 节点列表更新

```go
func (dps *DatabaseProgressService) UpdateNodeLists(taskID string, successNodes []string, failedNodes []model.NodeError) error {
    var task model.ProgressTask
    if err := dps.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
        return err
    }
    
    // 转换为 JSON
    if len(successNodes) > 0 {
        successJSON, _ := json.Marshal(successNodes)
        task.SuccessNodes = string(successJSON)
    }
    
    if len(failedNodes) > 0 {
        failedJSON, _ := json.Marshal(failedNodes)
        task.FailedNodes = string(failedJSON)
    }
    
    return dps.db.Save(&task).Error
}
```

### 5. 并发处理逻辑（数据库模式）

```go
func (dps *DatabaseProgressService) ProcessBatchWithProgress(...) error {
    // 创建数据库任务
    dps.CreateTask(taskID, action, total, userID)
    
    // 并发处理节点
    semaphore := make(chan struct{}, maxConcurrency)
    var wg sync.WaitGroup
    var mu sync.Mutex
    var failedNodes []model.NodeError
    var successNodes []string
    
    for i, nodeName := range nodeNames {
        wg.Add(1)
        go func(index int, node string) {
            defer func() {
                if r := recover(); r != nil {
                    // Panic 保护
                    mu.Lock()
                    failedNodes = append(failedNodes, model.NodeError{
                        NodeName: node,
                        Error:    fmt.Sprintf("panic: %v", r),
                    })
                    dps.UpdateNodeLists(taskID, successNodes, failedNodes)
                    mu.Unlock()
                }
                wg.Done()
            }()
            
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            mu.Lock()
            processed++
            currentIndex := processed
            mu.Unlock()
            
            // 更新数据库进度（会创建消息记录）
            dps.UpdateProgress(taskID, currentIndex, node, userID)
            
            // 处理节点
            if err := processor.ProcessNode(ctx, node, index); err != nil {
                mu.Lock()
                failedNodes = append(failedNodes, model.NodeError{
                    NodeName: node,
                    Error:    err.Error(),
                })
                dps.UpdateNodeLists(taskID, successNodes, failedNodes)
                mu.Unlock()
            } else {
                mu.Lock()
                successNodes = append(successNodes, node)
                dps.UpdateNodeLists(taskID, successNodes, failedNodes)
                mu.Unlock()
            }
        }(i, nodeName)
    }
    
    wg.Wait()
    
    // 标记任务完成
    if len(failedNodes) > 0 {
        dps.ErrorTask(taskID, fmt.Errorf("部分节点失败"), userID)
    } else {
        dps.CompleteTask(taskID, userID)
    }
    
    return nil
}
```

### 6. 优势

- ✅ **高可用性**：任意副本可以处理任务，单个副本故障不影响服务
- ✅ **数据持久化**：进度信息不会因进程重启而丢失
- ✅ **跨副本同步**：所有副本都能获取到任务进度
- ✅ **用户体验一致**：用户连接到任何副本都能看到相同的进度
- ✅ **负载均衡**：可以通过负载均衡器分散请求到多个副本

### 7. 潜在挑战与解决方案

#### 挑战 1：轮询延迟

**问题**：500ms 的轮询间隔导致进度更新有 0.5 秒的延迟。

**影响**：
- 对于快速完成的任务（< 1秒），可能出现进度跳跃
- 完成消息可能延迟到达

**解决方案**：
- ✅ **已实现**：完成和错误消息优先级更高（排序优先处理）
- ✅ **已实现**：限制批次大小（`Limit(100)`）避免单次查询过多
- ⚠️ **可优化**：缩短轮询间隔到 200ms（需权衡数据库压力）
- ⚠️ **可优化**：使用 PostgreSQL LISTEN/NOTIFY 实现真正的实时推送

#### 挑战 2：数据库写入压力

**问题**：每个进度更新都写入数据库，高并发时可能产生大量写入。

**场景**：
- 100 个节点的批量操作
- 每个节点处理前后各发送一次进度
- 总共 200+ 次数据库写入

**影响**：
- 数据库 I/O 压力增大
- 可能成为性能瓶颈

**解决方案**：
- ✅ **已实现**：使用 JSONB 字段存储节点列表，减少写入次数
- ✅ **已实现**：使用索引优化查询（`idx_progress_messages_unprocessed`）
- ⚠️ **可优化**：批量写入消息（攒一批再写入）
- ⚠️ **可优化**：使用 Redis 作为消息队列，降低数据库压力

#### 挑战 3：消息重复推送

**问题**：多个副本可能同时读取到同一条未处理的消息。

**场景**：
```
时刻 T0: 副本 A 和副本 B 同时轮询
时刻 T0: 都读取到消息 ID=100（processed=false）
时刻 T1: 副本 A 推送消息给用户
时刻 T2: 副本 B 也推送消息给用户（重复）
时刻 T3: 副本 A 标记消息为 processed=true
时刻 T4: 副本 B 也标记消息为 processed=true
```

**影响**：
- 用户可能收到重复的进度消息
- 前端需要去重处理

**解决方案**：
- ✅ **前端去重**：前端基于 `task_id` + `current` 去重
- ⚠️ **可优化**：使用数据库行级锁（`SELECT ... FOR UPDATE SKIP LOCKED`）
  ```sql
  SELECT * FROM progress_messages 
  WHERE processed = false 
  ORDER BY created_at 
  LIMIT 100
  FOR UPDATE SKIP LOCKED;  -- 跳过已被其他事务锁定的行
  ```

#### 挑战 4：用户断线重连

**问题**：用户在任务执行期间断开连接，重连时如何恢复进度？

**场景**：
```
1. 用户发起批量操作（100个节点）
2. WebSocket 连接，开始接收进度
3. 网络抖动，连接断开（处理了 50 个节点）
4. 用户刷新页面，重新连接
5. 如何显示已完成的 50 个节点？
```

**解决方案**：
- ✅ **已实现**：WebSocket 连接建立时，主动查询数据库中的运行任务
  ```go
  // 在 HandleWebSocket 中
  if s.useDatabase && s.dbProgressService != nil {
      go func() {
          time.Sleep(100 * time.Millisecond)
          s.dbProgressService.processUnsentMessages()  // 立即推送未处理的消息
      }()
  }
  
  // 发送当前任务状态
  s.sendCurrentTaskStatus(userID)
  ```

- ✅ **已实现**：查询数据库获取当前任务的最新状态
  ```go
  func (s *Service) sendCurrentTaskStatus(userID uint) {
      if s.useDatabase && s.dbProgressService != nil {
          tasks, _ := s.dbProgressService.GetUserTasks(userID, model.TaskStatusRunning)
          for _, task := range tasks {
              s.sendToUser(userID, task.ToProgressMessage())
          }
      }
  }
  ```

#### 挑战 5：任务清理

**问题**：完成的任务和消息会不断累积，占用数据库空间。

**影响**：
- 数据库表持续增长
- 查询性能下降

**解决方案**：
- ✅ **已实现**：使用软删除（`deleted_at`）
- ⚠️ **需补充**：定期清理策略
  - 保留最近 7 天的任务记录
  - 立即删除已处理的消息（`processed = true`）
  - 定时任务（每天凌晨执行）

**建议实现**：
```go
func (dps *DatabaseProgressService) StartCleanupScheduler() {
    ticker := time.NewTicker(24 * time.Hour)
    go func() {
        for range ticker.C {
            dps.cleanupOldTasks()
        }
    }()
}

func (dps *DatabaseProgressService) cleanupOldTasks() {
    // 删除 7 天前的已完成任务
    cutoff := time.Now().AddDate(0, 0, -7)
    dps.db.Where("completed_at < ? AND status IN (?)", 
        cutoff, 
        []string{"completed", "failed"},
    ).Delete(&model.ProgressTask{})
    
    // 删除已处理的消息
    dps.db.Where("processed = true").Delete(&model.ProgressMessage{})
}
```

#### 挑战 6：数据库连接池压力

**问题**：多副本环境下，每个副本都在轮询数据库。

**场景**：
- 4 个副本
- 每 500ms 轮询一次
- 每秒 8 次查询（仅轮询）
- 加上任务执行时的写入，连接数可能激增

**解决方案**：
- ✅ **已配置**：数据库连接池参数
  ```go
  MaxOpenConns: 25,  // 每个副本最多 25 个连接
  MaxIdleConns: 10,  // 空闲连接保持 10 个
  MaxLifetime:  3600 // 连接最长存活 1 小时
  ```

- ⚠️ **监控建议**：
  - 监控数据库连接数
  - 监控查询响应时间
  - 设置告警阈值

---

## 🔄 关键场景分析

### 场景 1：正常流程（单副本）

```
用户                  后端（内存模式）              前端
 │                        │                        │
 ├─ 批量禁止调度 100 节点─>│                        │
 │                        ├─ 创建内存任务           │
 │                        ├─ 启动 10 个 Goroutine  │
 │                        ├─ 处理节点 1             │
 │                        ├──WebSocket 推送────────>│ 显示: 1/100
 │                        ├─ 处理节点 2             │
 │                        ├──WebSocket 推送────────>│ 显示: 2/100
 │                        │   ... (并发处理)        │
 │                        ├─ 处理节点 100           │
 │                        ├──WebSocket 推送────────>│ 显示: 100/100
 │                        ├─ 标记任务完成           │
 │                        ├──完成消息推送──────────>│ 显示总结
 │<───────────────────────┴──返回结果─────────────┤
```

**时间线**：
- T0: 用户点击批量禁止调度
- T0 + 50ms: WebSocket 收到第一条进度消息
- T0 + 5s: 100 个节点处理完成（假设并发 10，每节点 0.5s）
- T0 + 5.1s: 前端收到完成消息

**特点**：
- ✅ 延迟极低（< 100ms）
- ✅ 实时性极好

---

### 场景 2：正常流程（多副本 - 用户和任务在同一副本）

```
用户                  副本 A                    数据库                前端
 │                        │                        │                    │
 ├─ 批量禁止调度 100 节点─>│                        │                    │
 │                        ├─ 创建数据库任务────────>│                    │
 │                        ├─ 启动 10 个 Goroutine  │                    │
 │                        ├─ 处理节点 1             │                    │
 │                        ├─ 写入进度消息──────────>│                    │
 │                        ├─ 轮询未处理消息────────>│                    │
 │                        ├<─ 返回消息 [ID=1]───────┤                    │
 │                        ├──WebSocket 推送────────────────────────────>│ 显示: 1/100
 │                        ├─ 标记消息已处理────────>│                    │
 │                        │   ... (并发处理)        │                    │
 │                        ├─ 写入完成消息──────────>│                    │
 │                        ├─ 轮询未处理消息────────>│                    │
 │                        ├<─ 返回完成消息──────────┤                    │
 │                        ├──WebSocket 推送────────────────────────────>│ 显示总结
 │<───────────────────────┴──返回结果──────────────┴────────────────────┤
```

**时间线**：
- T0: 用户点击批量禁止调度
- T0 + 50ms: 第一条进度消息写入数据库
- T0 + 550ms: 副本 A 轮询到第一条消息并推送
- T0 + 5s: 100 个节点处理完成
- T0 + 5s: 完成消息写入数据库
- T0 + 5.5s: 副本 A 轮询到完成消息并推送

**特点**：
- ⚠️ 有 500ms 的轮询延迟
- ✅ 数据持久化
- ✅ 可扩展性好

---

### 场景 3：跨副本（用户在副本 A，任务在副本 B）

```
用户                  副本 A                  数据库                副本 B
 │                        │                        │                    │
 ├─ 批量禁止调度 100 节点───────────────────────────────────────────────>│
 │  <WebSocket 连接>      │                        │                    ├─ 创建数据库任务
 │                        │                        │<─────────────────────┤
 │                        │                        │                    ├─ 启动 Goroutine
 │                        │                        │<─ 写入进度消息──────┤
 │                        ├─ 轮询未处理消息────────>│                    │
 │                        ├<─ 返回消息 [ID=1]───────┤                    │
 │                        ├──WebSocket 推送────────>│                    │
 │                        ├─ 标记消息已处理────────>│                    │
 │                        │   ... (持续轮询)        │<─ 持续写入进度─────┤
 │                        ├─ 轮询完成消息──────────>│                    │
 │                        ├<─ 返回完成消息──────────┤                    │
 │                        ├──WebSocket 推送────────>│                    │
 │<───────────────────────┴────────────────────────┴────────────────────┤
```

**关键点**：
- ✅ 副本 B 执行任务，副本 A 推送进度，用户无感知
- ✅ 数据库作为中介，实现跨副本通信
- ⚠️ 轮询延迟 500ms

---

### 场景 4：用户断线重连

```
用户                  后端（数据库模式）        数据库                前端
 │                        │                        │                    │
 ├─ 批量操作（100节点）───>│                        │                    │
 │                        ├─ 创建任务──────────────>│                    │
 │                        ├─ 处理中（50/100）       │                    │
 │                        ├─ 写入进度消息──────────>│                    │
 │  <WebSocket 推送>      ├──进度推送──────────────────────────────────>│
 │                        │                        │                    │
 ├─ 断开连接 ❌            │                        │                    │
 │                        ├─ 继续处理（51-100）     │                    │
 │                        ├─ 写入进度消息──────────>│                    │
 │                        │  (无人接收)             │                    │
 │                        ├─ 任务完成──────────────>│                    │
 │                        │                        │                    │
 ├─ 刷新页面 🔄            │                        │                    │
 ├─ 重新建立 WebSocket────>│                        │                    │
 │                        ├─ 查询用户任务──────────>│                    │
 │                        ├<─ 返回已完成任务────────┤                    │
 │                        ├─ 查询未处理消息────────>│                    │
 │                        ├<─ 返回完成消息──────────┤                    │
 │                        ├──推送任务状态──────────────────────────────>│ 显示: 100/100 ✅
 │                        ├──推送完成消息──────────────────────────────>│ 显示总结
```

**恢复机制**：
1. WebSocket 重连时触发 `HandleWebSocket`
2. 调用 `sendCurrentTaskStatus` 查询运行中的任务
3. 调用 `processUnsentMessages` 推送未发送的消息
4. 前端收到完整的任务状态

**时间线**：
- T0: 任务开始
- T5: 用户断开连接（50/100）
- T10: 任务完成（100/100）
- T15: 用户重连
- T15 + 100ms: 收到完成消息和任务状态

**特点**：
- ✅ 数据不丢失
- ✅ 用户体验连贯
- ⚠️ 中间进度无法恢复（只能看到最终状态）

---

### 场景 5：副本故障切换

```
用户                  副本 A                  数据库                副本 B（负载均衡）
 │                        │                        │                    │
 ├─ 批量操作 ────────────>│                        │                    │
 │                        ├─ 创建任务──────────────>│                    │
 │                        ├─ 处理中（30/100）       │                    │
 │                        ├─ 写入进度消息──────────>│                    │
 │  <WebSocket 推送>      ├──进度推送──────────────>│                    │
 │                        │                        │                    │
 │                        💥 副本 A 崩溃            │                    │
 │                        │                        │                    │
 ├─ 负载均衡器重连 ───────────────────────────────────────────────────>│
 │  <新的 WebSocket>      │                        │                    ├─ 接受连接
 │                        │                        │<─ 查询任务和消息────┤
 │                        │                        ├──返回任务状态─────>│
 │                        │                        │                    ├──推送状态────>│
 │                        │                        │                    │
 │                        │                      ✅ 任务在数据库中，继续处理
```

**关键点**：
- ✅ 任务不会因为副本崩溃而丢失
- ✅ 其他副本可以继续推送进度
- ⚠️ 但任务执行进程已终止，需要重新提交任务（当前未实现自动恢复）

**改进建议**：
- 实现任务状态检查：如果任务长时间（> 5 分钟）处于 `running` 但无更新，标记为 `failed`
- 实现任务重试机制

---

## 📊 性能对比

| 指标                   | 单副本（内存模式）      | 多副本（数据库模式）    |
|------------------------|-------------------------|-------------------------|
| **进度更新延迟**       | < 50ms                  | 500-1000ms              |
| **完成消息延迟**       | < 50ms                  | 500-1000ms              |
| **数据库写入**         | 无                      | 每个进度 2 次写入       |
| **数据库查询**         | 无                      | 每 500ms 一次           |
| **内存占用**           | 低（仅内存状态）        | 中（内存 + 数据库缓存） |
| **可扩展性**           | 单机                    | 水平扩展                |
| **高可用性**           | 无（单点故障）          | 高（多副本冗余）        |
| **数据持久化**         | 否（重启丢失）          | 是（持久化到数据库）    |
| **断线恢复能力**       | 有限（队列保存）        | 完整（数据库恢复）      |

---

## 🛠️ 优化建议

### 1. 短期优化（可立即实施）

#### 1.1 降低轮询间隔（200ms）

**修改**：`internal/service/progress/database.go:223`

```go
ticker := time.NewTicker(200 * time.Millisecond)  // 从 500ms 降低到 200ms
```

**影响**：
- ✅ 进度更新延迟降低到 200-400ms
- ⚠️ 数据库查询频率增加 2.5 倍

**建议**：在数据库负载可承受的情况下实施。

#### 1.2 批量写入进度消息

**当前**：每次进度更新都写入一条消息记录。

**优化**：攒一批（如 10 条）再批量写入。

```go
type DatabaseProgressService struct {
    // ... 现有字段
    pendingMessages []model.ProgressMessage
    pendingMutex    sync.Mutex
}

func (dps *DatabaseProgressService) batchInsertMessages() {
    dps.pendingMutex.Lock()
    if len(dps.pendingMessages) == 0 {
        dps.pendingMutex.Unlock()
        return
    }
    
    messages := dps.pendingMessages
    dps.pendingMessages = nil
    dps.pendingMutex.Unlock()
    
    // 批量写入
    dps.db.Create(&messages)
}

// 后台定时批量写入
func (dps *DatabaseProgressService) startBatchInserter() {
    ticker := time.NewTicker(100 * time.Millisecond)
    go func() {
        for range ticker.C {
            dps.batchInsertMessages()
        }
    }()
}
```

**效果**：
- ✅ 减少数据库写入次数 10 倍
- ⚠️ 进度更新延迟增加 100ms

#### 1.3 使用数据库行锁避免重复推送

**修改**：`internal/service/progress/database.go:242`

```go
query := dps.db.Raw(`
    SELECT * FROM progress_messages
    WHERE processed = false AND created_at > ?
    ORDER BY CASE WHEN type IN ('complete', 'error') THEN 0 ELSE 1 END, created_at ASC
    LIMIT 100
    FOR UPDATE SKIP LOCKED
`, dps.lastProcessedTime)
```

**效果**：
- ✅ 避免多副本重复推送消息
- ✅ 提高消息处理效率

### 2. 中期优化（需要架构调整）

#### 2.1 使用 Redis 作为消息队列

**架构**：

```
副本 B （执行任务）
   ↓ 写入进度
Redis Stream（消息队列）
   ↓ 推送消息
副本 A （订阅者）
   ↓ WebSocket
前端
```

**优势**：
- ✅ 实时性极高（< 10ms）
- ✅ 支持发布订阅模式
- ✅ 降低数据库压力

**实现**：

```go
// 写入进度到 Redis Stream
func (dps *DatabaseProgressService) publishProgress(msg ProgressMessage) {
    rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: fmt.Sprintf("progress:%d", msg.UserID),
        Values: map[string]interface{}{
            "task_id": msg.TaskID,
            "type":    msg.Type,
            "data":    json.Marshal(msg),
        },
    })
}

// 订阅用户的进度消息
func (dps *DatabaseProgressService) subscribeProgress(userID uint) {
    stream := fmt.Sprintf("progress:%d", userID)
    for {
        messages, _ := rdb.XRead(ctx, &redis.XReadArgs{
            Streams: []string{stream, "0"},
            Block:   0,  // 阻塞等待
        }).Result()
        
        for _, msg := range messages[0].Messages {
            // 推送给 WebSocket
            dps.wsService.sendToUser(userID, parseMessage(msg))
        }
    }
}
```

#### 2.2 PostgreSQL LISTEN/NOTIFY

**架构**：

```
副本 B （执行任务）
   ↓ NOTIFY progress_update
PostgreSQL（LISTEN 通道）
   ↓ 实时通知
副本 A（LISTEN progress_update）
   ↓ WebSocket
前端
```

**优势**：
- ✅ 真正的实时推送（< 10ms）
- ✅ 无需轮询
- ✅ 不引入额外组件

**实现**：

```go
// 监听数据库通知
func (dps *DatabaseProgressService) listenNotifications() {
    listener := pq.NewListener(databaseURL, 10*time.Second, time.Minute, nil)
    listener.Listen("progress_update")
    
    for notification := range listener.Notify {
        var msg ProgressMessage
        json.Unmarshal([]byte(notification.Extra), &msg)
        dps.wsService.sendToUser(msg.UserID, msg)
    }
}

// 发送通知
func (dps *DatabaseProgressService) notifyProgress(msg ProgressMessage) {
    payload, _ := json.Marshal(msg)
    dps.db.Exec("SELECT pg_notify('progress_update', ?)", string(payload))
}
```

### 3. 长期优化（需要重构）

#### 3.1 实现分布式任务队列

使用 Kafka / RabbitMQ / NATS 等专业消息队列。

#### 3.2 实现任务断点续传

当副本崩溃时，其他副本可以接管任务继续执行。

---

## 🧪 测试建议

### 1. 单副本测试

```bash
# 测试场景 1：正常批量操作
curl -X POST http://localhost:8080/api/v1/nodes/batch/cordon \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"cluster_name":"test","node_names":["node-1","node-2",...,"node-100"]}'

# 测试场景 2：断线重连
# 1. 启动批量操作
# 2. 中途关闭浏览器标签
# 3. 重新打开页面并连接 WebSocket
# 4. 验证是否收到完成消息
```

### 2. 多副本测试

```bash
# 启动 3 个副本
docker-compose up --scale app=3

# 测试场景 1：跨副本进度同步
# 1. 连接到副本 A 的 WebSocket
# 2. 向副本 B 发送批量操作请求
# 3. 验证副本 A 能否推送进度

# 测试场景 2：副本故障切换
# 1. 连接到副本 A 的 WebSocket
# 2. 发起批量操作
# 3. 中途杀死副本 A 进程
# 4. 负载均衡器重连到副本 B
# 5. 验证进度是否恢复

# 测试场景 3：高并发
# 使用 JMeter 或 Locust 模拟 100 个用户同时发起批量操作
```

### 3. 压力测试

```bash
# 测试数据库写入性能
# 批量操作 1000 个节点，观察数据库 CPU 和 I/O

# 测试轮询性能
# 启动 10 个副本，每个副本每 500ms 轮询一次，观察数据库连接数
```

---

## 📋 总结

### 单副本环境（内存模式）

**适用场景**：
- 小规模部署（单实例足够）
- 对实时性要求极高
- 数据丢失影响可接受

**特点**：
- ✅ 性能最佳
- ✅ 实现简单
- ❌ 无高可用性

### 多副本环境（数据库模式）

**适用场景**：
- 生产环境（需要高可用）
- 大规模集群（需要负载均衡）
- 数据持久化要求高

**特点**：
- ✅ 高可用性
- ✅ 可水平扩展
- ✅ 数据持久化
- ⚠️ 有轮询延迟
- ⚠️ 数据库压力较大

### 推荐配置

| 环境         | 副本数 | 模式选择       | 数据库类型  |
|--------------|--------|----------------|-------------|
| 开发环境     | 1      | 内存模式       | SQLite      |
| 测试环境     | 2      | 数据库模式     | PostgreSQL  |
| 生产环境     | 3-5    | 数据库模式     | PostgreSQL  |

### 关键指标监控

1. **进度更新延迟**：从任务执行到前端收到消息的时间
2. **完成消息到达率**：完成消息成功推送的比例
3. **数据库查询 QPS**：每秒查询次数
4. **数据库写入 QPS**：每秒写入次数
5. **WebSocket 连接数**：当前活跃的 WebSocket 连接
6. **任务成功率**：批量操作成功完成的比例

---

## 🔗 相关文档

- [实时通知系统设计文档](./realtime-notification-system.md)（推荐阅读）
- [多实例集群广播配置指南](./multi-instance-broadcast.md)
- [批量操作优化设计文档](./batch-operations-optimization.md)
- [微服务架构文档](./microservice-architecture.md)

---

## 🎉 最新更新

**2025-11-20**：新增实时通知系统，支持 PostgreSQL LISTEN/NOTIFY 和 Redis Pub/Sub，进度延迟从 500ms 降低到 < 10ms。详见 [实时通知系统文档](./realtime-notification-system.md)

