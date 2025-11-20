# PostgreSQL Notifier 优化与日志降噪

## 问题分析

### 1. PostgreSQL Notifier 连接问题

**症状**：
```
PostgreSQL Listener verification failed: sql: database is closed
```

**根本原因**：
- PostgreSQL `pg_notify` 使用的 GORM DB 连接可能在验证时未正确初始化
- 缺少对底层 `sql.DB` 连接健康状态的检查
- 使用了错误的占位符语法（`?` 而非 PostgreSQL 的 `$1, $2`）

**解决方案**：

1. **添加连接健康检查**：
   - 在执行 `pg_notify` 前，先获取底层 `sql.DB` 并执行 `Ping()`
   - 确保连接处于活跃状态

2. **修正 SQL 占位符**：
   - 将 `SELECT pg_notify(?, ?)` 改为 `SELECT pg_notify($1, $2)`
   - 使用 `WithContext` 传递上下文以支持超时控制

3. **改进的 Notify 方法**：

```go
func (p *PostgresNotifier) Notify(ctx context.Context, message ProgressMessage) error {
    // 1. Marshal 消息
    payload, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }
    
    // 2. 检查 GORM DB 连接状态
    sqlDB, err := p.db.DB()
    if err != nil {
        return fmt.Errorf("failed to get database: %w", err)
    }
    
    // 3. 验证连接健康
    if err := sqlDB.Ping(); err != nil {
        return fmt.Errorf("database connection error: %w", err)
    }
    
    // 4. 使用正确的占位符执行 pg_notify
    result := p.db.WithContext(ctx).Exec("SELECT pg_notify($1, $2)", "progress_update", string(payload))
    
    if result.Error != nil {
        return fmt.Errorf("failed to notify: %w", result.Error)
    }
    
    return nil
}
```

### 2. 日志轰炸问题

**症状**：
- 批量操作时产生大量重复日志
- 每个进度更新都记录详细日志
- WebSocket 消息传递的每个步骤都记录日志

**影响**：
- 日志文件快速增长
- 日志可读性下降
- 影响系统性能（I/O 压力）

**优化策略**：

#### 2.1 进度更新日志降频

**优化前**：
- 每个节点处理都记录日志
- 每次进度更新都记录详细信息

**优化后**：
- 每10个节点记录一次进度
- 只记录关键节点（第1个、最后一个、每10的倍数）

```go
// 减少日志频率：每10个节点或最后一个节点记录一次
if currentIndex%10 == 0 || currentIndex == total {
    dps.logger.Infof("Progress: %d/%d nodes processed successfully", currentIndex, total)
}
```

#### 2.2 通知消息日志分级

**优化前**：
- 所有通知消息都记录 Info 或 Debug 日志
- Subscribe 接收到的每条消息都记录

**优化后**：
- 只记录重要消息（`complete`、`error` 类型）
- 普通进度消息不记录或降级为 Debug

```go
// 只记录重要消息（complete, error），避免日志轰炸
if message.Type == "complete" || message.Type == "error" {
    p.logger.Infof("Sending PostgreSQL notification: task=%s type=%s user=%d", 
        message.TaskID, message.Type, message.UserID)
}
```

#### 2.3 进度通知采样

**优化前**：
- 每次进度更新都发送 PostgreSQL 通知

**优化后**：
- 每10次更新发送一次通知
- 最后一次更新必定发送

```go
// 使用通知器发送实时通知（仅每10次发送一次，避免过多通知）
if !dps.usePolling && (task.Current%10 == 0 || task.Current == task.Total) {
    // ... 发送通知 ...
}
```

#### 2.4 轮询消息处理日志简化

**优化前**：
- 每次轮询都记录处理的消息数
- 每条消息的发送都记录

**优化后**：
- 只在有重要消息时记录
- 统计重要消息数量

```go
// 只记录有重要消息时的日志
importantCount := 0
for _, msg := range messages {
    if msg.Type == "complete" || msg.Type == "error" {
        importantCount++
    }
}
if importantCount > 0 {
    dps.logger.Infof("Processing %d unsent messages (%d important)", len(messages), importantCount)
}
```

#### 2.5 WebSocket 连接重试日志优化

**优化前**：
- 每次重试都记录日志

**优化后**：
- 只记录最后一次失败的尝试

```go
if hasConnection {
    dps.processUnsentMessages()
    dps.logger.Debugf("Force pushed completion message for task %s", taskID)
    break
} else {
    // 减少日志噪音，只记录最后一次尝试
    if i == 4 {
        dps.logger.Warningf("No WebSocket connection after 5 attempts for task %s", taskID)
    }
}
```

## 优化效果

### 日志量对比

**优化前**（处理 100 个节点）：
- 进度日志：~100 条（每个节点 1 条）
- 通知日志：~200 条（发送 + 接收）
- 轮询日志：~50 条
- **总计**：~350 条日志

**优化后**（处理 100 个节点）：
- 进度日志：~10 条（每 10 个节点 1 条）
- 通知日志：~4 条（只记录 complete/error）
- 轮询日志：~2 条（只在有重要消息时记录）
- **总计**：~16 条日志

**日志量减少**：**95%+**

### 关键日志保留

虽然大幅减少了日志量，但仍保留了所有关键信息：
- ✅ 任务开始/完成/失败
- ✅ 连接错误和异常
- ✅ 重要消息的发送状态
- ✅ 总体进度里程碑（10%, 20%, ..., 100%）

## 最佳实践

### 1. 日志分级建议

| 日志级别 | 使用场景 | 频率 |
|---------|---------|------|
| **Error** | 功能失败、连接错误、数据损坏 | 异常时 |
| **Warning** | 重试、降级、可恢复的问题 | 偶尔 |
| **Info** | 关键业务事件、任务完成、重要状态变更 | 低频 |
| **Debug** | 详细的执行流程、中间状态 | 开发/调试时启用 |

### 2. 多副本环境的日志策略

在多副本环境中：
- 避免每个副本都记录相同的日志
- 使用 Task ID 作为日志关联 ID
- 重要事件只记录一次（通过数据库去重）

### 3. 生产环境配置

**推荐日志级别**：
- 正常运行：`INFO`
- 调试问题：`DEBUG`
- 高负载：`WARNING`（只记录警告和错误）

**日志采样**：
```yaml
logging:
  level: info
  sampling:
    progress_updates: 10  # 每10次记录1次
    notifications: 1      # 重要通知每次记录
```

## 监控建议

虽然减少了日志，但建议增加以下监控指标：

1. **进度服务指标**：
   - 活跃任务数
   - 完成任务数（成功/失败）
   - 通知发送成功率

2. **PostgreSQL LISTEN/NOTIFY 指标**：
   - Listener 连接状态
   - 通知延迟
   - 通知丢失率

3. **WebSocket 指标**：
   - 活跃连接数
   - 消息队列长度
   - 消息发送失败率

## 故障排查

### 如何启用详细日志？

**临时启用 Debug 日志**：
```bash
# 设置环境变量
export LOG_LEVEL=debug

# 或在配置文件中
logging:
  level: debug
```

**针对特定模块**：
```go
// 在代码中临时修改
if taskID == "debug_task_id" {
    logger.SetLevel(logger.DebugLevel)
    defer logger.SetLevel(logger.InfoLevel)
}
```

### 常见问题诊断

1. **进度不更新**：
   - 检查 Error 级别日志
   - 验证 PostgreSQL Listener 连接状态
   - 确认 WebSocket 连接是否活跃

2. **通知延迟**：
   - 检查 PostgreSQL `pg_notify` 是否正常工作
   - 验证网络延迟
   - 检查消息队列是否堆积

3. **连接失败**：
   - 检查数据库配置
   - 验证网络连通性
   - 确认 PostgreSQL 是否支持 LISTEN/NOTIFY

## 相关文档

- [多副本进度通知系统设计](./realtime-notification-system.md)
- [PostgreSQL LISTEN/NOTIFY 配置指南](./multi-replica-postgresql-setup.md)
- [日志优化总结](./log-optimization-summary.md)

