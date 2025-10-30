# 实时同步功能部署和测试指南

## 概述

本文档提供实时同步功能（Informer + WebSocket + 智能缓存）的部署和测试指南。

## 系统架构

### 核心组件

1. **Informer Service** (`backend/internal/informer/`)
   - 监听 Kubernetes API 的节点变化事件
   - 支持多集群管理
   - 自动处理 Add/Update/Delete 事件

2. **Smart Cache** (`backend/internal/smartcache/`)
   - 由 Informer 实时更新的内存缓存
   - 存储完整的节点信息（corev1.Node）
   - 提供快速的节点查询接口

3. **WebSocket Hub** (`backend/internal/websocket/`)
   - 管理 WebSocket 客户端连接
   - 按集群分组广播消息
   - 支持心跳检测和自动重连

4. **Realtime Manager** (`backend/internal/realtime/`)
   - 统一管理所有实时同步组件
   - 协调 Informer、Smart Cache 和 WebSocket Hub
   - 提供集群注册和状态查询接口

## 部署步骤

### 1. 编译应用

```bash
cd backend
go build -o bin/kube-node-manager ./cmd/main.go
```

### 2. 配置检查

实时同步功能无需额外配置，会在应用启动时自动初始化。

### 3. 启动应用

```bash
./bin/kube-node-manager
```

### 4. 验证启动日志

查看日志中的关键启动信息：

```
INFO: Realtime Manager started successfully
INFO: WebSocket Hub started
```

### 5. 注册集群

当添加新集群或测试连接时，系统会自动注册集群到实时管理器：

```
INFO: Cluster registered: <cluster-name>
INFO: Informer for cluster <cluster-name> started and synced
```

## 功能测试

### 测试 1: Informer 实时监听

#### 测试目标
验证 Informer 能够正确监听 Kubernetes 集群的节点变化。

#### 测试步骤

1. 启动应用并添加一个 Kubernetes 集群

2. 在 Kubernetes 集群中执行节点操作：
   ```bash
   # 禁止调度
   kubectl cordon <node-name>
   ```

3. 查看应用日志：
   ```
   INFO: Node updated: cluster=<cluster-name>, node=<node-name>, changes=[Schedulable]
   INFO: SmartCache: Updated node <node-name> in cluster <cluster-name>, changes=[Schedulable]
   ```

4. 验证其他操作：
   ```bash
   # 恢复调度
   kubectl uncordon <node-name>
   
   # 添加标签
   kubectl label node <node-name> test-label=test-value
   
   # 添加污点
   kubectl taint node <node-name> test-taint=test-value:NoSchedule
   ```

#### 预期结果
- 每次 Kubernetes 资源变化都应该在日志中产生相应的 Informer 和 SmartCache 更新日志
- 变化应该在 1-2 秒内被检测到

### 测试 2: WebSocket 实时推送

#### 测试目标
验证 WebSocket 能够将节点变化实时推送给前端客户端。

#### 测试步骤

1. 使用 WebSocket 测试工具连接到服务：
   ```
   ws://localhost:8080/api/v1/nodes/ws?cluster=<cluster-name>
   ```

2. 在另一个终端执行节点操作：
   ```bash
   kubectl cordon <node-name>
   ```

3. 在 WebSocket 客户端观察接收到的消息：
   ```json
   {
     "type": "node_update",
     "cluster_name": "my-cluster",
     "node_name": "node-1",
     "data": { /* 完整的节点信息 */ },
     "changes": ["Schedulable"],
     "timestamp": "2024-01-01T12:00:00Z"
   }
   ```

#### 推荐的 WebSocket 测试工具

**浏览器开发者工具测试**：

```javascript
// 在浏览器控制台中执行
const ws = new WebSocket('ws://localhost:8080/api/v1/nodes/ws?cluster=my-cluster');

ws.onopen = () => {
  console.log('WebSocket 连接已建立');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('收到消息:', data);
};

ws.onerror = (error) => {
  console.error('WebSocket 错误:', error);
};

ws.onclose = () => {
  console.log('WebSocket 连接已关闭');
};
```

**命令行工具 (wscat)**：

```bash
# 安装 wscat
npm install -g wscat

# 连接并监听消息
wscat -c "ws://localhost:8080/api/v1/nodes/ws?cluster=my-cluster"
```

#### 预期结果
- WebSocket 连接成功建立
- 节点变化在 1-2 秒内推送到客户端
- 消息格式正确，包含完整的节点信息和变化字段

### 测试 3: 智能缓存性能

#### 测试目标
验证智能缓存能够提供快速的节点查询，避免频繁调用 Kubernetes API。

#### 测试步骤

1. 使用 API 查询节点列表：
   ```bash
   curl -X GET "http://localhost:8080/api/v1/nodes?cluster_name=my-cluster" \
     -H "Authorization: Bearer <token>"
   ```

2. 查看应用日志，验证是否使用智能缓存：
   ```
   INFO: Retrieved 10 nodes from smart cache for cluster my-cluster
   ```

3. 多次重复查询，观察响应时间：
   - 首次查询（冷启动）：可能需要从 K8s API 获取
   - 后续查询：应该从智能缓存获取，响应时间 < 50ms

4. 执行节点操作后立即查询：
   ```bash
   kubectl cordon node-1
   sleep 2
   curl -X GET "http://localhost:8080/api/v1/nodes/node-1?cluster_name=my-cluster"
   ```

#### 预期结果
- 智能缓存命中率高（日志显示 "Retrieved from smart cache"）
- 查询响应快速（< 50ms for list, < 10ms for single node）
- 节点变化后，缓存自动更新（无需手动刷新）

### 测试 4: 批量操作实时同步

#### 测试目标
验证批量操作后不再需要多次刷新，系统能自动同步所有变化。

#### 测试步骤

1. 通过 API 执行批量禁止调度操作：
   ```bash
   curl -X POST "http://localhost:8080/api/v1/nodes/batch-cordon" \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{
       "cluster_name": "my-cluster",
       "node_names": ["node-1", "node-2", "node-3"],
       "reason": "maintenance"
     }'
   ```

2. 监听 WebSocket 消息，观察是否收到所有节点的更新事件

3. 在前端刷新节点列表，验证所有节点状态正确

4. 查看日志，确认每个节点的变化都被 Informer 捕获：
   ```
   INFO: Node updated: cluster=my-cluster, node=node-1, changes=[Schedulable]
   INFO: Node updated: cluster=my-cluster, node=node-2, changes=[Schedulable]
   INFO: Node updated: cluster=my-cluster, node=node-3, changes=[Schedulable]
   ```

#### 预期结果
- 所有节点的变化都通过 Informer 自动同步
- WebSocket 推送所有节点的更新消息
- 前端显示实时更新，无需手动刷新
- 不再需要多次刷新即可看到所有变化

### 测试 5: 多集群支持

#### 测试目标
验证系统能够同时管理多个集群的实时同步。

#### 测试步骤

1. 添加多个 Kubernetes 集群（例如：cluster-1, cluster-2）

2. 分别为每个集群建立 WebSocket 连接：
   ```javascript
   const ws1 = new WebSocket('ws://localhost:8080/api/v1/nodes/ws?cluster=cluster-1');
   const ws2 = new WebSocket('ws://localhost:8080/api/v1/nodes/ws?cluster=cluster-2');
   ```

3. 在不同集群中分别执行节点操作

4. 验证 WebSocket 消息的隔离性：
   - ws1 应该只收到 cluster-1 的消息
   - ws2 应该只收到 cluster-2 的消息

#### 预期结果
- 每个集群的 Informer 独立运行
- WebSocket 消息正确路由到相应集群的订阅者
- 集群之间互不干扰

## 故障排查

### 问题 1: Informer 未启动

**症状**：日志中没有 "Informer started" 消息

**可能原因**：
1. 集群连接失败
2. RBAC 权限不足

**解决方案**：
```bash
# 检查集群连接
kubectl get nodes --kubeconfig=<kubeconfig-file>

# 检查 ServiceAccount 权限
kubectl auth can-i list nodes --as=system:serviceaccount:<namespace>:<serviceaccount>
kubectl auth can-i watch nodes --as=system:serviceaccount:<namespace>:<serviceaccount>
```

### 问题 2: WebSocket 连接失败

**症状**：前端无法建立 WebSocket 连接

**可能原因**：
1. 路由配置错误
2. 防火墙阻止 WebSocket
3. 反向代理不支持 WebSocket

**解决方案**：

**Nginx 配置示例**：
```nginx
location /api/v1/nodes/ws {
    proxy_pass http://backend:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_cache_bypass $http_upgrade;
}
```

**Docker Compose 端口映射**：
```yaml
services:
  backend:
    ports:
      - "8080:8080"
```

### 问题 3: 缓存数据不一致

**症状**：前端显示的数据与 Kubernetes 实际状态不一致

**可能原因**：
1. Informer 同步延迟
2. SmartCache 更新失败

**解决方案**：
```bash
# 查看 Informer 状态
curl -X GET "http://localhost:8080/api/v1/realtime/status" \
  -H "Authorization: Bearer <token>"

# 强制刷新缓存（如果实现了此接口）
curl -X POST "http://localhost:8080/api/v1/cache/refresh?cluster=my-cluster" \
  -H "Authorization: Bearer <token>"
```

## 监控和指标

### 关键日志

关注以下日志级别和消息：

- **INFO**: 正常操作事件（Informer 启动、节点更新等）
- **WARNING**: 潜在问题（Informer 同步延迟、WebSocket 连接断开等）
- **ERROR**: 严重错误（Informer 启动失败、缓存更新失败等）

### 性能指标

推荐监控的指标：

1. **Informer 延迟**：从 Kubernetes 事件发生到 SmartCache 更新的时间
2. **WebSocket 推送延迟**：从 SmartCache 更新到客户端接收消息的时间
3. **缓存命中率**：使用 SmartCache 的查询比例
4. **WebSocket 连接数**：当前活跃的 WebSocket 客户端数量
5. **内存使用**：SmartCache 占用的内存大小

## 性能优化建议

### 1. Informer 调优

- 调整 `resyncPeriod`（默认 30 秒）：
  ```go
  factory := informers.NewSharedInformerFactoryWithOptions(
      client,
      30*time.Second, // resyncPeriod
      informers.WithNamespace(""),
  )
  ```

### 2. WebSocket 优化

- 启用消息压缩（如果支持）
- 批量发送多个小消息而不是逐个发送
- 配置合理的心跳间隔（默认 30 秒）

### 3. 缓存优化

- 定期清理长时间未访问的缓存条目
- 考虑添加 LRU 缓存淘汰策略
- 监控内存使用，必要时限制缓存大小

## 前端集成示例

参考 `docs/frontend-websocket-integration.md` 获取完整的前端集成指南。

**基本示例**：

```vue
<script setup>
import { ref, onMounted, onUnmounted } from 'vue';

const nodes = ref([]);
let ws = null;

onMounted(() => {
  // 建立 WebSocket 连接
  const clusterName = 'my-cluster';
  ws = new WebSocket(`ws://localhost:8080/api/v1/nodes/ws?cluster=${clusterName}`);
  
  ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    
    if (message.type === 'node_update') {
      // 更新节点列表
      const index = nodes.value.findIndex(n => n.name === message.node_name);
      if (index >= 0) {
        nodes.value[index] = message.data;
      }
    }
  };
});

onUnmounted(() => {
  if (ws) {
    ws.close();
  }
});
</script>
```

## 安全考虑

### 1. WebSocket 认证

确保 WebSocket 连接也经过身份验证：

```go
// 在 WebSocket handler 中验证 token
func (h *Handler) HandleWebSocket(c *gin.Context) {
    token := c.Query("token") // 或从 header 获取
    // 验证 token...
    
    // 升级连接
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    // ...
}
```

### 2. 访问控制

限制用户只能访问有权限的集群：

```go
// 验证用户是否有权访问该集群
if !userHasClusterAccess(userID, clusterName) {
    c.JSON(403, gin.H{"error": "access denied"})
    return
}
```

### 3. 速率限制

防止 WebSocket 消息洪水：

```go
// 限制每个客户端的消息发送频率
type RateLimiter struct {
    rate  int           // 每秒最大消息数
    burst int           // 突发容量
    // ...
}
```

## 总结

实时同步功能通过 Informer、SmartCache 和 WebSocket 的紧密集成，提供了：

1. **实时性**：节点变化在 1-2 秒内同步到前端
2. **高性能**：智能缓存减少 API 调用，查询响应时间 < 50ms
3. **可扩展性**：支持多集群和多客户端
4. **用户体验**：无需手动刷新，自动更新界面

## 参考文档

- [实时同步实现指南](./realtime-sync-implementation-guide.md)
- [前端 WebSocket 集成](./frontend-websocket-integration.md)
- [实时同步状态](./realtime-sync-status.md)
- [Kubernetes Informer 官方文档](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes)

