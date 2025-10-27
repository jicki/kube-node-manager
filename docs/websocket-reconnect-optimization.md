# WebSocket 重连优化文档

## 问题描述

在批量操作（如批量 taint 复制）过程中，WebSocket 连接出现频繁断开和重连的问题，导致：

1. **日志过多**：每秒产生大量的连接/断开日志，影响日志可读性
2. **重连循环**：前端在任务完成后仍然不断重连
3. **资源浪费**：频繁的连接建立和销毁消耗系统资源

### 根本原因

- 前端在 `onclose` 事件中，如果进度 ≥100% 但未收到 `complete` 消息，会自动重连
- 每次重连时，后端会关闭旧连接，触发前端 `onclose` 事件
- 形成连接循环：**重连 → 后端关闭旧连接 → 触发 onclose → 再重连**

## 解决方案

### 1. 前端优化（`frontend/src/components/common/ProgressDialog.vue`）

#### 1.1 添加重连限制机制

```javascript
const reconnectCount = ref(0)
const maxReconnectAttempts = 5 // 最大重连次数
```

- 设置最大重连次数为 5 次
- 防止无限重连循环
- 超过限制后提示用户刷新页面

#### 1.2 改进重连延迟策略

```javascript
const reconnectDelay = Math.min(1000 * reconnectCount.value, 3000) // 递增延迟，最大3秒
```

- 使用递增延迟策略：第 1 次 1 秒，第 2 次 2 秒，第 3 次及以后 3 秒
- 避免频繁重连造成的资源浪费

#### 1.3 完成时立即关闭连接

```javascript
case 'complete':
  progressData.value = { ...data }
  isCompleted.value = true
  
  // 任务完成后立即关闭 WebSocket 连接，避免重连
  console.log('任务完成，关闭 WebSocket 连接')
  closeWebSocket()
  
  ElMessage.success(data.message || '批量操作完成')
  emit('completed', data)
  break
```

- 收到完成消息后立即关闭 WebSocket
- 防止后续的重连尝试

#### 1.4 改进连接状态检查

```javascript
websocket.value.onclose = (event) => {
  // 如果任务已完成或出错，不再重连
  if (isCompleted.value || isError.value) {
    console.log('任务已完成或出错，不再重连')
    return
  }
  
  // 检查重连次数限制
  if ((shouldReconnect || isNearCompletion) && reconnectCount.value < maxReconnectAttempts) {
    reconnectCount.value++
    // ... 重连逻辑
  }
}
```

#### 1.5 连接成功时重置计数器

```javascript
websocket.value.onopen = () => {
  console.log('WebSocket连接已建立')
  reconnectCount.value = 0 // 连接成功后重置重连计数
}
```

### 2. 后端优化（`backend/internal/service/progress/progress.go`）

#### 2.1 减少连接日志输出

**优化前**：
```go
s.logger.Infof("WebSocket connection attempt from %s", c.ClientIP())
s.logger.Infof("WebSocket token received (length: %d)", len(token))
s.logger.Infof("Validating WebSocket token...")
s.logger.Infof("WebSocket authentication successful for user %d", userID)
s.logger.Infof("WebSocket connected for user %d", userID)
```

**优化后**：
- 移除非必要的连接尝试日志
- 只在认证失败等异常情况下记录日志

#### 2.2 减少断开日志输出

**优化前**：
```go
s.logger.Infof("WebSocket connection closed normally for user %d: %v", conn.userID, err)
s.logger.Infof("WebSocket connection closed without status for user %d (browser tab switch/refresh)", conn.userID)
s.logger.Infof("WebSocket disconnected for user %d", userID)
s.logger.Infof("WritePump closed for user %d", conn.userID)
s.logger.Infof("ReadPump closed for user %d", conn.userID)
```

**优化后**：
```go
// readPump 中只记录异常关闭错误
if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
    s.logger.Errorf("WebSocket unexpected error for user %d: %v", conn.userID, err)
}
// 其他正常关闭不记录日志
```

#### 2.3 静默关闭旧连接

**优化前**：
```go
if existingConn, exists := s.connections[userID]; exists {
    s.logger.Infof("Closing existing WebSocket connection for user %d", userID)
    close(existingConn.send)
    existingConn.ws.Close()
}
```

**优化后**：
```go
if existingConn, exists := s.connections[userID]; exists {
    // 静默关闭旧连接，避免日志过多
    close(existingConn.send)
    existingConn.ws.Close()
}
```

## 优化效果

### 1. 日志输出减少

**优化前**：每次连接/断开会产生 8-10 条 INFO 日志
```
INFO: WebSocket connection attempt from 10.10.12.98
INFO: WebSocket token received (length: 217)
INFO: Validating WebSocket token...
INFO: WebSocket authentication successful for user 3
INFO: WebSocket connected for user 3
INFO: No pending tasks found for user 3 on reconnection
INFO: WebSocket connection closed without status for user 3
INFO: WebSocket disconnected for user 3
INFO: ReadPump closed for user 3
INFO: WritePump closed for user 3
```

**优化后**：正常连接/断开不产生日志，只在异常情况下记录错误日志

### 2. 重连次数限制

- 最多重连 5 次，避免无限循环
- 使用递增延迟策略，减少服务器压力
- 超过限制后提示用户，提供明确的操作指引

### 3. 任务完成后的行为改进

- 收到完成消息后立即关闭 WebSocket
- 不再尝试重连
- 避免资源浪费

## 测试建议

1. **正常流程测试**
   - 执行批量 taint 复制操作
   - 观察 WebSocket 连接行为
   - 确认任务完成后不再重连

2. **异常流程测试**
   - 网络不稳定情况下的重连行为
   - 达到最大重连次数的提示
   - 任务执行中断时的处理

3. **日志验证**
   - 确认正常操作不产生大量日志
   - 异常情况仍能正确记录错误日志

## 后续改进建议

1. **添加心跳机制**
   - 前端定期发送 ping 消息
   - 后端响应 pong 消息
   - 保持连接活跃，减少意外断开

2. **状态同步优化**
   - 重连时同步任务状态
   - 避免状态不一致导致的重连

3. **用户体验优化**
   - 显示连接状态指示器
   - 提供手动重连按钮
   - 改进错误提示信息

## 相关文件

- `frontend/src/components/common/ProgressDialog.vue` - 前端进度对话框组件
- `backend/internal/service/progress/progress.go` - 后端 WebSocket 进度服务
- `backend/internal/service/progress/database.go` - 数据库进度持久化服务

## 版本信息

- 优化日期：2025-10-27
- 影响版本：v2.11.0+
- 优化作者：AI Assistant

## 参考资料

- [WebSocket API 文档](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)
- [Gorilla WebSocket 库文档](https://pkg.go.dev/github.com/gorilla/websocket)
- [Element Plus 对话框文档](https://element-plus.org/zh-CN/component/dialog.html)

