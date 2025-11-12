# 多实例集群广播配置指南

## 📋 功能概述

当在多实例部署环境中创建新的 Kubernetes 集群时，系统会自动将集群信息广播到所有实例，确保每个实例都能处理新集群的请求。

## 🔄 工作原理

### 1. 集群创建流程

```
用户创建集群 (通过任意实例)
       ↓
实例 A: 创建集群记录 + 创建 K8s Client
       ↓
实例 A: 广播到所有其他实例
       ↓
实例 B/C/D: 接收广播 → 重新加载集群 → 创建 K8s Client
       ↓
所有实例都能处理该集群的请求 ✅
```

### 2. 实例发现机制

系统支持三种实例发现方法（按优先级顺序）：

#### 方法 1：环境变量 `POD_IPS`（推荐用于 Kubernetes）

通过 Downward API 自动注入所有 Pod IP：

```yaml
env:
  - name: POD_IPS
    valueFrom:
      fieldRef:
        fieldPath: status.podIPs  # 自动获取所有 Pod IP
  - name: POD_PORT
    value: "8080"
```

格式：`POD_IPS=10.10.12.95,10.10.12.96,10.10.12.97,10.10.12.98`

#### 方法 2：环境变量 `INSTANCE_ADDRESSES`（手动配置）

手动指定所有实例的完整地址：

```yaml
env:
  - name: INSTANCE_ADDRESSES
    value: "10.10.12.95:8080,10.10.12.96:8080,10.10.12.97:8080,10.10.12.98:8080"
```

格式：`INSTANCE_ADDRESSES=host1:port1,host2:port2,...`

#### 方法 3：Kubernetes Service 发现（通过 DNS）

使用 Headless Service 进行服务发现：

```yaml
env:
  - name: SERVICE_NAME
    value: "kube-node-manager"
  - name: POD_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  - name: POD_PORT
    value: "8080"
```

系统会通过 DNS 解析 `<service-name>.<namespace>.svc.cluster.local` 获取所有实例 IP。

## 📦 Kubernetes 部署配置

### StatefulSet 配置示例

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: kube-node-manager
  namespace: kube-node-manager
spec:
  replicas: 4
  serviceName: kube-node-manager  # Headless Service 名称
  selector:
    matchLabels:
      app: kube-node-manager
  template:
    metadata:
      labels:
        app: kube-node-manager
    spec:
      serviceAccountName: kube-node-manager
      containers:
        - name: kube-node-manager
          image: your-registry/kube-node-manager:latest
          ports:
            - containerPort: 8080
              name: http
          env:
            # 方法 1: 使用 Downward API（推荐）
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: POD_PORT
              value: "8080"
            
            # 方法 3: Service 发现
            - name: SERVICE_NAME
              value: "kube-node-manager"
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            
            # 数据库配置
            - name: DB_TYPE
              value: "postgres"
            - name: DB_HOST
              value: "postgres-service"
            - name: DB_PORT
              value: "5432"
            - name: DB_DATABASE
              value: "kube_node_manager"
            - name: DB_USERNAME
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: username
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
          livenessProbe:
            httpGet:
              path: /health/live
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health/ready
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 5
```

### Headless Service 配置

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kube-node-manager
  namespace: kube-node-manager
spec:
  clusterIP: None  # Headless Service
  selector:
    app: kube-node-manager
  ports:
    - name: http
      port: 8080
      targetPort: 8080
```

### 负载均衡 Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kube-node-manager-lb
  namespace: kube-node-manager
spec:
  type: LoadBalancer
  selector:
    app: kube-node-manager
  ports:
    - name: http
      port: 80
      targetPort: 8080
```

## 🔒 安全配置

### NetworkPolicy 限制内部 API 访问

内部 API 端点 (`/api/v1/internal/*`) 仅应允许同 namespace 内的 Pod 访问：

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kube-node-manager-internal-api
  namespace: kube-node-manager
spec:
  podSelector:
    matchLabels:
      app: kube-node-manager
  policyTypes:
    - Ingress
  ingress:
    # 允许同 namespace 内的 Pod 访问内部 API
    - from:
        - podSelector:
            matchLabels:
              app: kube-node-manager
      ports:
        - protocol: TCP
          port: 8080
    # 允许外部访问公共 API（通过 Ingress）
    - from:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 8080
```

### RBAC 配置（如需通过 K8s API 发现实例）

如果使用 Kubernetes API 进行服务发现，需要以下 RBAC 权限：

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-node-manager
  namespace: kube-node-manager

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kube-node-manager
  namespace: kube-node-manager
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["services"]
    verbs: ["get"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kube-node-manager
  namespace: kube-node-manager
subjects:
  - kind: ServiceAccount
    name: kube-node-manager
    namespace: kube-node-manager
roleRef:
  kind: Role
  name: kube-node-manager
  apiGroup: rbac.authorization.k8s.io
```

## 🧪 验证配置

### 1. 检查实例发现

部署后查看日志，确认实例发现是否成功：

```bash
kubectl logs -n kube-node-manager kube-node-manager-0 | grep "Found.*instances"
```

预期输出：
```
Found 4 instances from POD_IPS environment variable
```

### 2. 测试集群创建广播

1. 创建一个新集群：

```bash
curl -X POST http://your-service/api/v1/clusters \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-cluster",
    "description": "Test cluster",
    "kube_config": "..."
  }'
```

2. 查看所有实例的日志，确认广播成功：

```bash
# 查看发起广播的实例
kubectl logs -n kube-node-manager <pod-name> | grep "Broadcasting cluster test-cluster"

# 查看接收广播的实例
kubectl logs -n kube-node-manager <other-pod-name> | grep "Received cluster reload request for: test-cluster"
```

预期输出（发起方）：
```
Broadcasting cluster test-cluster creation to 3 instances
Successfully broadcasted cluster test-cluster to instance 10.10.12.96:8080
Successfully broadcasted cluster test-cluster to instance 10.10.12.97:8080
Successfully broadcasted cluster test-cluster to instance 10.10.12.98:8080
Completed broadcasting cluster test-cluster creation
```

预期输出（接收方）：
```
Received cluster reload request for: test-cluster
Successfully reloaded cluster: test-cluster
```

### 3. 验证所有实例可处理新集群请求

向不同实例发送请求，验证都能正常处理：

```bash
# 向实例 1 请求
kubectl exec -it kube-node-manager-0 -- curl http://localhost:8080/api/v1/clusters

# 向实例 2 请求
kubectl exec -it kube-node-manager-1 -- curl http://localhost:8080/api/v1/clusters
```

## 🐛 故障排查

### 问题 1：广播失败 - "No other instances found"

**症状**：
```
No other instances found for broadcasting cluster creation
```

**原因**：实例发现配置未正确设置

**解决方案**：
1. 检查环境变量是否正确配置
2. 验证 Headless Service 是否创建
3. 检查 DNS 解析是否正常

```bash
# 检查环境变量
kubectl exec -it kube-node-manager-0 -- env | grep -E 'POD_IPS|INSTANCE_ADDRESSES|SERVICE_NAME'

# 检查 DNS 解析
kubectl exec -it kube-node-manager-0 -- nslookup kube-node-manager.kube-node-manager.svc.cluster.local
```

### 问题 2：广播超时 - "Failed to broadcast"

**症状**：
```
Failed to broadcast to 10.10.12.96:8080: context deadline exceeded
```

**原因**：网络不通或 Pod 未就绪

**解决方案**：
1. 检查网络策略是否阻止了 Pod 间通信
2. 验证目标 Pod 是否就绪

```bash
# 检查 Pod 状态
kubectl get pods -n kube-node-manager

# 测试网络连通性
kubectl exec -it kube-node-manager-0 -- curl -v http://10.10.12.96:8080/health
```

### 问题 3：集群创建后仍报 "client not found"

**症状**：
```
Kubernetes client not found for cluster: test-cluster
```

**原因**：
- 广播未成功执行
- 目标实例重启导致 client 丢失

**解决方案**：

1. 手动触发集群重载（临时方案）：

```bash
# 对每个实例手动调用重载 API
kubectl exec -it kube-node-manager-0 -- curl -X POST http://localhost:8080/api/v1/internal/clusters/test-cluster/reload
kubectl exec -it kube-node-manager-1 -- curl -X POST http://localhost:8080/api/v1/internal/clusters/test-cluster/reload
```

2. 重启所有实例（持久方案）：

```bash
kubectl rollout restart statefulset/kube-node-manager -n kube-node-manager
```

实例重启后会自动从数据库加载所有集群。

## 📈 性能优化

### 1. 调整广播超时

默认广播超时为 5 秒，如果网络延迟较高，可能需要调整：

修改 `backend/internal/service/cluster/cluster.go`：

```go
client := &http.Client{
    Timeout: 10 * time.Second,  // 增加到 10 秒
}
```

### 2. 限制并发广播数

默认使用 goroutine 并行广播，对于大量实例可能需要限制并发：

```go
// 使用信号量限制并发
semaphore := make(chan struct{}, 10)  // 最多 10 个并发请求

for _, instance := range instances {
    semaphore <- struct{}{}  // 获取信号量
    wg.Add(1)
    go func(addr string) {
        defer wg.Done()
        defer func() { <-semaphore }()  // 释放信号量
        // ... 广播逻辑
    }(instance)
}
```

### 3. 启用 HTTP/2

使用 HTTP/2 可以提高广播效率：

```go
import "golang.org/x/net/http2"

client := &http.Client{
    Timeout: 5 * time.Second,
    Transport: &http2.Transport{},
}
```

## 🔄 升级指南

### 从单实例升级到多实例

1. **更新部署配置**：
   - 将 Deployment 改为 StatefulSet
   - 添加环境变量配置
   - 创建 Headless Service

2. **滚动升级**：
   ```bash
   kubectl apply -f statefulset.yaml
   kubectl apply -f service-headless.yaml
   ```

3. **验证升级**：
   ```bash
   # 检查所有 Pod 是否就绪
   kubectl get pods -n kube-node-manager
   
   # 检查日志确认实例发现
   kubectl logs -n kube-node-manager kube-node-manager-0 | grep "Found.*instances"
   ```

4. **测试集群创建**：
   创建一个测试集群，确认所有实例都能处理请求。

## 📝 最佳实践

1. **使用 StatefulSet**：确保 Pod 有稳定的网络标识
2. **配置 Headless Service**：便于服务发现
3. **启用健康检查**：确保只有就绪的实例才接收流量
4. **配置 NetworkPolicy**：限制内部 API 仅内部访问
5. **监控日志**：定期检查广播成功率
6. **设置告警**：当广播失败率超过阈值时发送告警

## 🔗 相关文档

- [Kubernetes StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
- [Headless Services](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services)
- [Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [Downward API](https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/)

