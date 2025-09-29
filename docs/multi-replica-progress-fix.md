# 多副本环境下进度条卡住问题解决方案

## 问题背景

在多副本部署环境下，kube-node-manager 出现批量操作进度条卡住的问题。虽然配置文件显示单副本，但实际运行中检测到多个实例，导致进度同步失败。

## 问题根因分析

### 1. WebSocket连接路由问题
- **任务创建和WebSocket连接分离**：批量任务可能在实例A上创建，WebSocket连接路由到实例B
- **进度更新丢失**：任务在实例A上更新进度，但WebSocket连接在实例B上，进度消息无法到达客户端
- **完成消息路由失败**：任务在实例A上完成，但完成消息发送给实例A上不存在的WebSocket连接

### 2. 任务状态隔离
- **内存状态无法共享**：每个实例使用内存存储（`connections map`, `tasks map`, `completedTasks map`）
- **状态不同步**：任务状态无法在实例间传递
- **消息丢失**：WebSocket连接和任务状态在不同实例上

### 3. 负载均衡影响
- **请求分散**：负载均衡器将API请求和WebSocket连接分散到不同实例
- **无会话亲和性**：缺乏会话粘性配置，同一用户的请求可能路由到不同Pod

## 解决方案设计

### 方案1：基于数据库的状态共享（推荐）

#### 实现原理
- 使用数据库存储任务状态和进度消息
- 所有实例共享同一数据库，实现状态同步
- 轮询机制确保消息及时传递

#### 核心组件

**1. 数据库模型**
```go
// 任务状态表
type ProgressTask struct {
    TaskID      string     // 任务ID
    UserID      uint       // 用户ID
    Status      TaskStatus // 运行状态
    Current     int        // 当前进度
    Total       int        // 总数量
    Message     string     // 状态消息
    // ...
}

// 进度消息表
type ProgressMessage struct {
    UserID    uint   // 用户ID
    TaskID    string // 任务ID
    Type      string // 消息类型：progress/complete/error
    Processed bool   // 是否已处理
    // ...
}
```

**2. 数据库进度服务**
```go
type DatabaseProgressService struct {
    db        *gorm.DB
    wsService *Service // WebSocket服务
    // 消息轮询机制
}
```

**3. 配置启用**
```yaml
# config.yaml
progress:
  enable_database: true  # 启用数据库模式
```

#### 工作流程
1. **任务创建**：任务状态存储到数据库
2. **进度更新**：更新数据库中的任务状态，创建进度消息
3. **消息轮询**：所有实例轮询未处理的消息
4. **WebSocket推送**：有WebSocket连接的实例发送消息给客户端
5. **消息标记**：发送成功后标记消息为已处理

### 方案2：会话亲和性配置

#### Service配置
```yaml
apiVersion: v1
kind: Service
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 10800  # 3小时
```

#### Ingress配置
```yaml
annotations:
  nginx.ingress.kubernetes.io/affinity: "cookie"
  nginx.ingress.kubernetes.io/session-cookie-name: "kube-node-mgr-session"
  nginx.ingress.kubernetes.io/session-cookie-max-age: "10800"
```

## 部署指南

### 1. 使用多副本部署配置

```bash
# 部署多副本版本
kubectl apply -f deploy/k8s/k8s-multi-replica.yaml

# 或使用部署脚本
./deploy/scripts/deploy-multi-replica.sh
```

### 2. 环境变量配置

```yaml
env:
- name: PROGRESS_ENABLE_DATABASE
  value: "true"  # 启用数据库模式
```

### 3. 验证部署

```bash
# 检查Pod状态
kubectl get pods -n kube-node-mgr

# 检查数据库模式是否启用
kubectl logs -f statefulset/kube-node-mgr -n kube-node-mgr | grep "database mode"
```

## 配置对比

| 模式 | 单副本 | 多副本（内存） | 多副本（数据库） |
|------|--------|--------------|----------------|
| 副本数 | 1 | 2+ | 2+ |
| 状态存储 | 内存 | 各实例内存 | 共享数据库 |
| 进度同步 | ✅ | ❌ | ✅ |
| 会话亲和性 | 不需要 | 必需 | 推荐 |
| 资源消耗 | 低 | 中 | 中高 |
| 可用性 | 低 | 高 | 高 |

## 监控和故障排除

### 1. 检查数据库模式状态
```bash
kubectl logs -f statefulset/kube-node-mgr -n kube-node-mgr | grep -E "database mode|progress"
```

### 2. 监控任务状态
```bash
# 查看任务创建和完成
kubectl logs -f statefulset/kube-node-mgr -n kube-node-mgr | grep -E "Created task|completed successfully"
```

### 3. WebSocket连接监控
```bash
# 监控WebSocket连接
kubectl logs -f statefulset/kube-node-mgr -n kube-node-mgr | grep -E "WebSocket connected|disconnected"
```

### 4. 常见问题

**问题1：进度条仍然卡住**
- 检查 `PROGRESS_ENABLE_DATABASE` 是否为 `true`
- 确认数据库连接正常
- 查看是否有轮询错误日志

**问题2：WebSocket频繁断开**
- 检查会话亲和性配置
- 调整负载均衡器超时设置
- 确认Ingress WebSocket支持

**问题3：消息延迟**
- 调整轮询间隔（默认1秒）
- 检查数据库性能
- 优化消息批次大小

## 性能优化建议

### 1. 数据库优化
```go
// 增加数据库连接池
database:
  max_open_conns: 50
  max_idle_conns: 20
  max_lifetime: 3600
```

### 2. 轮询优化
```go
// 调整轮询间隔
pollInterval: 500 * time.Millisecond  // 500ms
```

### 3. 消息清理
- 自动清理已处理的消息（60秒后）
- 定期清理过期任务
- 设置合理的数据库索引

## 升级路径

### 从单副本升级
1. 备份现有数据
2. 更新配置启用数据库模式
3. 部署多副本版本
4. 验证功能正常

### 配置迁移
```bash
# 备份配置
kubectl get configmap kube-node-mgr-config -n kube-node-mgr -o yaml > backup-config.yaml

# 更新配置
kubectl patch configmap kube-node-mgr-config -n kube-node-mgr --patch '{"data":{"progress-enable-database":"true"}}'
```

## 总结

通过实施基于数据库的状态共享和会话亲和性配置，成功解决了多副本环境下的进度条卡住问题：

✅ **任务状态共享**：所有实例共享数据库中的任务状态
✅ **消息可靠传递**：轮询机制确保消息不丢失
✅ **WebSocket连接稳定**：会话亲和性减少连接跳转
✅ **向后兼容**：支持单副本和多副本模式切换
✅ **可扩展性**：支持水平扩展到更多副本

此解决方案确保了在多副本环境下，用户始终能够看到正确的批量操作进度，提升了系统的可用性和用户体验。