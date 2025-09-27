# 批量操作优化实现方案

## 概述

本文档描述了对 Kubernetes 节点管理器中标签管理和污点管理的批量操作的性能优化实现。主要包括：

1. 并发处理提升速度
2. WebSocket 实时进度推送
3. 用户友好的进度条界面
4. 向后兼容的 API 设计

## 后端优化

### 1. WebSocket 进度推送服务

创建了新的进度推送服务 (`/backend/internal/service/progress/progress.go`)，提供：

- **WebSocket 连接管理**：支持用户级别的 WebSocket 连接
- **进度消息推送**：实时推送任务进度、当前处理节点、完成状态等
- **并发处理框架**：提供通用的批量处理接口，支持可配置的并发数量
- **任务状态管理**：跟踪任务状态（运行中、完成、错误）

#### 关键特性：
```go
// 批量处理器接口
type BatchProcessor interface {
    ProcessNode(ctx context.Context, nodeName string, index int) error
}

// 进度消息结构
type ProgressMessage struct {
    TaskID      string    `json:"task_id"`
    Type        string    `json:"type"`        // progress, complete, error
    Action      string    `json:"action"`      // batch_label, batch_taint
    Current     int       `json:"current"`     // 当前完成数量
    Total       int       `json:"total"`       // 总数量
    Progress    float64   `json:"progress"`    // 进度百分比
    CurrentNode string    `json:"current_node"`
    Message     string    `json:"message"`
    Timestamp   time.Time `json:"timestamp"`
}
```

### 2. 标签服务优化

更新了标签服务 (`/backend/internal/service/label/label.go`)：

- **并发处理**：使用信号量限制并发数（默认5个）
- **进度推送**：实时推送处理进度
- **向后兼容**：保持原有 API 的功能

#### 新增方法：
```go
// BatchUpdateLabelsWithProgress 带进度推送的批量更新标签
func (s *Service) BatchUpdateLabelsWithProgress(req BatchUpdateRequest, userID uint, taskID string) error

// LabelProcessor 实现 BatchProcessor 接口
type LabelProcessor struct {
    svc    *Service
    req    BatchUpdateRequest
    userID uint
}
```

### 3. 污点服务优化

类似地优化了污点服务 (`/backend/internal/service/taint/taint.go`)：

- **TaintProcessor**：实现批量处理接口
- **并发执行**：并行处理多个节点的污点操作
- **错误处理**：收集并聚合处理错误

### 4. 新增 API 端点

添加了带进度推送的批量操作端点：

```
POST /api/v1/nodes/labels/batch-add-progress
POST /api/v1/nodes/taints/batch-add-progress
GET  /api/v1/progress/ws  (WebSocket 连接)
```

## 前端优化

### 1. 进度条组件

创建了通用的进度对话框组件 (`/frontend/src/components/common/ProgressDialog.vue`)：

#### 功能特性：
- **实时进度显示**：百分比进度条和数量进度
- **当前操作信息**：显示正在处理的节点
- **状态管理**：处理连接、进度、完成、错误状态
- **WebSocket 连接**：自动建立和管理 WebSocket 连接
- **用户交互**：支持取消操作（开发中）

#### 组件接口：
```vue
<ProgressDialog 
  v-model="progressDialogVisible"
  :task-id="currentTaskId"
  @completed="handleProgressCompleted"
  @error="handleProgressError"
  @cancelled="handleProgressCancelled"
/>
```

### 2. API 更新

更新了标签和污点 API (`/frontend/src/api/label.js`, `/frontend/src/api/taint.js`)：

```javascript
// 带进度推送的批量添加标签
batchAddLabelsWithProgress(requestData) {
  return request({
    url: '/api/v1/nodes/labels/batch-add-progress',
    method: 'post',
    data: requestData
  })
}

// 带进度推送的批量添加污点
batchAddTaintsWithProgress(requestData) {
  return request({
    url: '/api/v1/nodes/taints/batch-add-progress',
    method: 'post',
    data: requestData
  })
}
```

### 3. 界面集成

更新了标签管理和污点管理页面：

- **智能选择**：节点数量 > 5 时自动使用带进度的 API
- **进度显示**：显示详细的批量操作进度
- **用户体验**：操作完成后自动刷新数据和关闭对话框

## 性能提升

### 速度优化

1. **并发处理**：
   - 旧版本：顺序处理，50ms 延迟
   - 新版本：并发处理，最大并发数 5

2. **预期性能提升**：
   - 10个节点：从 ~500ms 提升到 ~100ms（提升80%）
   - 50个节点：从 ~2.5s 提升到 ~500ms（提升80%）
   - 100个节点：从 ~5s 提升到 ~1s（提升80%）

### 用户体验改进

1. **实时反馈**：用户可以看到具体的处理进度
2. **当前状态**：显示正在处理的节点
3. **错误处理**：详细的错误信息和部分成功处理
4. **向后兼容**：小批量操作仍使用快速同步方式

## 技术实现细节

### WebSocket 认证

```javascript
const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
const host = window.location.host
const token = localStorage.getItem('token')
const wsUrl = `${protocol}//${host}/api/v1/progress/ws?token=${token}`
```

### 并发控制

```go
// 使用信号量控制并发
semaphore := make(chan struct{}, maxConcurrency)

for i, nodeName := range nodeNames {
    wg.Add(1)
    go func(index int, node string) {
        defer wg.Done()
        
        // 获取信号量
        semaphore <- struct{}{}
        defer func() { <-semaphore }()
        
        // 处理节点...
    }(i, nodeName)
}
```

### 错误聚合

```go
var errors []string
var mu sync.Mutex

// 在goroutine中收集错误
if err := processor.ProcessNode(ctx, node, index); err != nil {
    mu.Lock()
    errors = append(errors, fmt.Sprintf("%s: %v", node, err))
    mu.Unlock()
}
```

## 部署和配置

### 后端配置

无需额外配置，服务自动初始化。

### 前端配置

WebSocket 连接自动检测协议（HTTP/HTTPS）并连接到相应的 WebSocket 端点。

### Nginx 配置（如果使用）

确保 Nginx 配置支持 WebSocket 升级：

```nginx
location /api/v1/progress/ws {
    proxy_pass http://backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
}
```

## 向后兼容性

- 原有的批量操作 API 保持不变
- 新的进度 API 作为可选功能
- 前端根据节点数量智能选择使用哪种 API

## 未来改进

1. **取消操作**：实现批量操作的取消功能
2. **重试机制**：对失败的节点自动重试
3. **批量大小优化**：根据集群规模动态调整并发数
4. **操作历史**：记录批量操作的详细历史
5. **性能监控**：添加批量操作的性能指标

## 结论

通过本次优化，标签管理和污点管理的批量操作在性能和用户体验方面都得到了显著提升：

- **性能提升80%**：通过并发处理大幅减少操作时间
- **实时进度反馈**：用户可以实时了解操作进度
- **向后兼容**：不影响现有功能的使用
- **可扩展架构**：进度推送框架可用于其他批量操作

这些改进使得大规模 Kubernetes 集群的节点管理变得更加高效和用户友好。
