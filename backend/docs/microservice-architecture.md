# 微服务架构优化文档

## 概述

本文档描述了 Kube Node Manager 后端的微服务架构优化，包括多数据库支持、健康检查、监控指标、优雅关闭等功能。

## 新增功能

### 1. 多数据库支持

#### 支持的数据库类型
- **SQLite**: 适合开发环境和小规模部署
- **PostgreSQL**: 适合生产环境和高并发场景

#### 配置方式

```yaml
database:
  type: "postgres"  # sqlite 或 postgres
  host: "localhost"
  port: 5432
  database: "kube_node_manager"
  username: "postgres"
  password: "your-password"
  ssl_mode: "disable"
  max_open_conns: 25
  max_idle_conns: 10
  max_lifetime: 3600
```

#### 环境变量支持
```bash
export DATABASE_URL="postgres://user:pass@host:port/dbname?sslmode=disable"
```

### 2. 健康检查端点

#### 端点列表
- `GET /health/` - 基础健康检查
- `GET /health/live` - Kubernetes 存活探针
- `GET /health/ready` - Kubernetes 就绪探针  
- `GET /health/detailed` - 详细健康检查（包含数据库、系统资源等）

#### 响应格式

**基础健康检查**:
```json
{
  "status": "healthy",
  "service": "kube-node-manager",
  "version": "1.0.0",
  "timestamp": "2023-12-07T10:30:00Z",
  "uptime": "2h30m15s"
}
```

**详细健康检查**:
```json
{
  "status": "healthy",
  "service": "kube-node-manager",
  "version": "1.0.0",
  "timestamp": "2023-12-07T10:30:00Z",
  "uptime": "2h30m15s",
  "details": {
    "database": {
      "status": "healthy",
      "data": {
        "max_open_connections": 25,
        "open_connections": 3,
        "in_use": 1,
        "idle": 2
      }
    },
    "system": {
      "status": "healthy",
      "data": {
        "memory": {
          "alloc_mb": 45,
          "sys_mb": 78
        },
        "goroutines": 25
      }
    },
    "runtime": {
      "go_version": "go1.21.0",
      "hostname": "pod-abc123"
    }
  }
}
```

### 3. 监控指标端点

#### Prometheus 格式指标
- `GET /metrics` - Prometheus 格式的监控指标

#### 指标类型
- `kube_node_manager_up` - 应用状态（1=运行，0=故障）
- `kube_node_manager_start_time_seconds` - 应用启动时间戳
- `kube_node_manager_database_up` - 数据库连接状态
- `kube_node_manager_memory_usage_bytes` - 内存使用量
- `kube_node_manager_goroutines_total` - Goroutine 数量

### 4. 结构化日志

#### 日志格式支持
- **文本格式**: 适合开发环境，易读
- **JSON格式**: 适合生产环境，便于日志收集

#### 配置方式
```yaml
logging:
  format: "json"     # text 或 json
  level: "info"      # debug, info, warn, error
  structured: true   # 是否启用结构化日志
```

#### 环境变量
```bash
export LOG_FORMAT=json
export LOG_LEVEL=info
```

#### 日志输出示例

**JSON 格式**:
```json
{
  "timestamp": "2023-12-07T10:30:00.123456789Z",
  "level": "INFO",
  "service": "kube-node-manager",
  "version": "1.0.0",
  "hostname": "pod-abc123",
  "message": "Server starting on port 8080",
  "caller": "main.go:92 main.main",
  "fields": {
    "port": "8080",
    "mode": "release"
  }
}
```

### 5. 优雅关闭机制

#### 功能特性
- 监听 SIGINT 和 SIGTERM 信号
- 30秒超时等待正在处理的请求完成
- 自动关闭数据库连接
- 完整的关闭日志记录

#### 关闭流程
1. 接收到关闭信号
2. 停止接收新请求
3. 等待现有请求处理完成（最多30秒）
4. 关闭数据库连接
5. 记录关闭完成日志

### 6. 高性能连接池

#### PostgreSQL 连接池配置
```yaml
database:
  max_open_conns: 25    # 最大打开连接数
  max_idle_conns: 10    # 最大空闲连接数  
  max_lifetime: 3600    # 连接最大生存时间（秒）
```

#### SQLite 特殊处理
- 自动设置为单连接模式（max_open_conns=1, max_idle_conns=1）
- 避免SQLite的并发限制问题

## Kubernetes 部署优化

### 1. 探针配置

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: kube-node-manager
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3

        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
```

### 2. 优雅关闭配置

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      terminationGracePeriodSeconds: 60  # 给优雅关闭足够时间
      containers:
      - name: kube-node-manager
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sleep", "5"]  # 给负载均衡器时间更新
```

### 3. 资源限制

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi" 
    cpu: "500m"
```

## 监控集成

### 1. Prometheus 监控

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kube-node-manager
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/path: "/metrics"
    prometheus.io/port: "8080"
spec:
  selector:
    app: kube-node-manager
  ports:
  - port: 8080
    targetPort: 8080
```

### 2. ServiceMonitor 配置

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kube-node-manager
spec:
  selector:
    matchLabels:
      app: kube-node-manager
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

## 多副本部署支持

### 1. 无状态设计
- 所有状态存储在数据库中
- 支持水平扩展
- 无单点故障

### 2. 数据库连接优化
- 连接池配置适合多副本场景
- 自动重连机制
- 连接泄露检测

### 3. 负载均衡
```yaml
apiVersion: v1
kind: Service
metadata:
  name: kube-node-manager
spec:
  selector:
    app: kube-node-manager
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
  sessionAffinity: None  # 支持任意副本处理请求
```

## 环境变量覆盖

优先级：环境变量 > 配置文件 > 默认值

```bash
# 服务配置
export PORT=8080
export SERVER_MODE=release

# 数据库配置
export DATABASE_TYPE=postgres
export DATABASE_URL="postgres://user:pass@host:port/dbname"

# 日志配置  
export LOG_FORMAT=json
export LOG_LEVEL=info

# 服务标识
export SERVICE_NAME=kube-node-manager
export INSTANCE_ID=pod-abc123
```

## 性能优化建议

### 1. 生产环境配置
```yaml
server:
  mode: "release"
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 120

database:
  type: "postgres"
  max_open_conns: 25
  max_idle_conns: 10
  max_lifetime: 3600

logging:
  format: "json"
  level: "info"
  structured: true
```

### 2. 资源规划
- CPU: 100m-500m per pod
- Memory: 128Mi-512Mi per pod  
- Database: 根据节点数量规划连接数

### 3. 扩容策略
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: kube-node-manager-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: kube-node-manager
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## 故障排除

### 1. 健康检查失败
```bash
# 检查基础健康状态
curl http://localhost:8080/health/

# 检查详细健康状态
curl http://localhost:8080/health/detailed

# 检查就绪状态
curl http://localhost:8080/health/ready
```

### 2. 数据库连接问题
```bash
# 查看连接池状态（详细健康检查中包含）
curl http://localhost:8080/health/detailed | jq .details.database

# 检查环境变量
env | grep DATABASE
```

### 3. 监控指标异常
```bash
# 查看 Prometheus 指标
curl http://localhost:8080/metrics

# 检查应用状态指标
curl http://localhost:8080/metrics | grep kube_node_manager_up
```

### 4. 日志问题
```bash
# 切换到结构化日志
export LOG_FORMAT=json

# 调整日志级别
export LOG_LEVEL=debug
```

## 升级说明

### 1. 配置兼容性
- 保持与旧版本配置文件兼容
- 新增配置项有合理默认值
- 环境变量优先级最高

### 2. 滚动更新
```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 25%
      maxSurge: 25%
```

### 3. 数据迁移
- SQLite 到 PostgreSQL 迁移工具（待开发）
- 配置文件自动转换
- 向后兼容保证

## 总结

通过这些微服务架构优化，Kube Node Manager 现在支持：

1. ✅ **多数据库支持** - SQLite 和 PostgreSQL
2. ✅ **健康检查** - 完整的 K8s 探针支持
3. ✅ **监控指标** - Prometheus 格式指标
4. ✅ **结构化日志** - JSON 格式支持
5. ✅ **优雅关闭** - 信号处理和超时机制
6. ✅ **高性能连接池** - 针对多副本优化
7. ✅ **无状态设计** - 支持水平扩展
8. ✅ **配置灵活性** - 环境变量覆盖

这些优化使得应用更适合在 Kubernetes 环境中以微服务架构运行，支持多副本部署和高可用性。
