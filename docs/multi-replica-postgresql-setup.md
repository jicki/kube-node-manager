# 多副本环境 PostgreSQL LISTEN/NOTIFY 配置指南

## 概述

本文档说明如何在 Kubernetes 多副本环境中配置 kube-node-manager，使用 PostgreSQL LISTEN/NOTIFY 实现跨副本实时消息传递。

## 为什么选择 PostgreSQL LISTEN/NOTIFY？

### 优势

- ✅ **无需额外组件**：不需要部署 Redis
- ✅ **低延迟**：消息延迟 < 100ms
- ✅ **高可靠性**：利用 PostgreSQL 的成熟机制
- ✅ **简化运维**：减少组件数量和维护成本
- ✅ **配置简单**：只需正确配置数据库连接参数

### 与其他方案对比

| 方案 | 延迟 | 可靠性 | 额外组件 | 运维复杂度 |
|------|------|--------|----------|------------|
| PostgreSQL LISTEN/NOTIFY | < 100ms | 高 | 无 | 低 |
| Redis Pub/Sub | < 50ms | 高 | Redis | 中 |
| 轮询模式 | 500ms+ | 中 | 无 | 低 |

## 问题诊断：常见错误

### 错误现象

```
ERROR: PostgreSQL listener problem: dial tcp 127.0.0.1:5432: connect: connection refused
```

### 错误原因

PostgreSQL Listener 尝试连接 `localhost:5432`，但在 Kubernetes Pod 中：
- Pod 内没有本地 PostgreSQL 服务
- 数据库是外部 Service 或独立的 StatefulSet
- 环境变量未正确设置，使用了默认值

### 根本原因

代码中 PostgreSQL Listener 创建独立连接时，必须使用正确的数据库地址。在 v2.34.9 之前的版本中，如果环境变量未设置，会错误地使用 `localhost`。

**v2.34.9 修复：**
- PostgreSQL Listener 现在从配置文件直接读取数据库参数
- 确保 Listener 与主应用使用相同的数据库连接配置
- 添加启动时连接验证，及早发现配置问题

## 配置步骤

### 1. 数据库配置

#### 配置文件方式（推荐）

编辑 `configs/config.yaml` 或 `configs/config-multi-replica.yaml`：

```yaml
database:
  type: "postgres"
  # ⚠️  关键：使用 K8s Service 名称或可访问的外部地址
  host: "postgres-service.default.svc.cluster.local"
  port: 5432
  database: "kube_node_manager"
  username: "postgres"
  password: "${DB_PASSWORD}"  # 从环境变量读取
  ssl_mode: "disable"
  max_open_conns: 25
  max_idle_conns: 10
  max_lifetime: 3600

progress:
  enable_database: true  # 启用多副本支持
  notify_type: "postgres"  # 使用 PostgreSQL LISTEN/NOTIFY
  poll_interval: 10000  # 降级轮询间隔（毫秒）
```

#### 环境变量方式（备选）

在 K8s Deployment/StatefulSet 中设置：

```yaml
env:
  - name: DB_HOST
    value: "postgres-service.default.svc.cluster.local"
  - name: DB_PORT
    value: "5432"
  - name: DB_USERNAME
    value: "postgres"
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: postgres-secret
        key: password
  - name: DB_DATABASE
    value: "kube_node_manager"
  - name: DB_SSL_MODE
    value: "disable"
```

**注意：** v2.34.9+ 优先使用配置文件，环境变量作为备份。

### 2. Kubernetes 部署配置

#### Secret 配置

创建 PostgreSQL 密码 Secret：

```bash
kubectl create secret generic postgres-secret \
  --from-literal=password='your-secure-password' \
  -n kube-node-mgr
```

#### StatefulSet 配置示例

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: kube-node-mgr
  namespace: kube-node-mgr
spec:
  replicas: 3  # 多副本部署
  serviceName: kube-node-mgr-headless
  template:
    spec:
      containers:
      - name: kube-node-mgr
        image: kube-node-mgr:v2.34.9  # 使用修复版本
        env:
          # 数据库连接参数（可选，配置文件已设置则无需重复）
          - name: DB_HOST
            value: "postgres-service.default.svc.cluster.local"
          - name: DB_PORT
            value: "5432"
          - name: DB_USERNAME
            value: "postgres"
          - name: DB_PASSWORD
            valueFrom:
              secretKeyRef:
                name: postgres-secret
                key: password
          - name: DB_DATABASE
            value: "kube_node_manager"
```

### 3. PostgreSQL Service 配置

确保 PostgreSQL Service 可从所有 Pod 访问：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: default
spec:
  ports:
  - port: 5432
    targetPort: 5432
  selector:
    app: postgresql
```

**关键点：**
- Service 名称必须与配置中的 `host` 一致
- 如果 PostgreSQL 在不同 namespace，使用完整域名：`postgres-service.namespace.svc.cluster.local`

## 验证部署

### 1. 检查启动日志

正常启动日志应包含：

```
INFO: Initializing postgres database
INFO: Successfully initialized postgres database
INFO: Progress service enabled database mode for multi-replica support with postgres notification
INFO: Initializing PostgreSQL listener with host=postgres-service.default.svc.cluster.local port=5432 dbname=kube_node_manager
INFO: PostgreSQL listener connected successfully
INFO: PostgreSQL LISTEN/NOTIFY notifier initialized successfully
INFO: ✅ PostgreSQL notifier verified successfully - multi-replica progress updates ready
```

### 2. 检查常见错误

#### 连接被拒绝

```
ERROR: PostgreSQL listener problem: dial tcp 127.0.0.1:5432: connect: connection refused
```

**解决方案：**
- 检查 `DB_HOST` 环境变量或配置文件中的 `database.host`
- 确保使用正确的 Service 地址，而不是 `localhost`
- 验证 PostgreSQL Service 是否存在：`kubectl get svc postgres-service`

#### 超时

```
ERROR: failed to connect PostgreSQL listener within 10s timeout
```

**解决方案：**
- 检查网络策略是否允许 Pod 访问 PostgreSQL
- 验证 PostgreSQL 是否正常运行：`kubectl get pods -l app=postgresql`
- 检查 PostgreSQL 日志：`kubectl logs <postgresql-pod>`

#### 认证失败

```
ERROR: PostgreSQL listener problem: FATAL: password authentication failed
```

**解决方案：**
- 检查 `DB_USERNAME` 和 `DB_PASSWORD` 是否正确
- 验证 Secret 是否存在：`kubectl get secret postgres-secret`
- 检查 PostgreSQL 用户权限

### 3. 功能测试

测试批量标签操作：

```bash
# 1. 登录前端界面
# 2. 选择多个节点（建议 5-10 个）
# 3. 批量添加标签
# 4. 观察进度对话框

预期结果：
✅ 进度条实时更新
✅ 处理中/成功/失败节点数量正确显示
✅ 完成后显示准确的成功和失败计数
✅ 所有节点列表可见
```

### 4. 多副本验证

```bash
# 检查所有副本状态
kubectl get pods -l app=kube-node-mgr -o wide

# 查看每个副本的日志
for pod in $(kubectl get pods -l app=kube-node-mgr -o name); do
  echo "=== $pod ==="
  kubectl logs $pod | grep -E "(PostgreSQL|notifier|progress)"
done

# 验证跨副本消息传递
# 1. 连接到副本 A 的前端
# 2. 提交批量操作
# 3. 操作可能在副本 B 执行
# 4. 副本 A 应该实时收到进度更新
```

## 性能优化

### 连接池配置

对于高并发环境，调整连接池参数：

```yaml
database:
  max_open_conns: 50    # 增加最大连接数
  max_idle_conns: 20    # 增加空闲连接数
  max_lifetime: 7200    # 连接最大生存时间（秒）
```

### PostgreSQL 调优

```sql
-- 调整 PostgreSQL 参数以优化 LISTEN/NOTIFY
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '256MB';

-- 重载配置
SELECT pg_reload_conf();
```

## 故障排查

### 查看详细日志

```bash
# 查看应用日志
kubectl logs -l app=kube-node-mgr -f --tail=100

# 过滤 PostgreSQL 相关日志
kubectl logs -l app=kube-node-mgr | grep -i postgres

# 过滤通知器相关日志
kubectl logs -l app=kube-node-mgr | grep -i "notifier\|notification"
```

### 常见问题

#### 问题：进度条不更新

**可能原因：**
1. PostgreSQL Listener 连接失败
2. WebSocket 连接断开
3. 前端轮询降级失败

**排查步骤：**
```bash
# 1. 检查 PostgreSQL Listener 状态
kubectl logs <pod-name> | grep "PostgreSQL listener"

# 2. 检查 WebSocket 连接
# 打开浏览器开发者工具 -> Network -> WS

# 3. 检查数据库任务记录
# 连接到 PostgreSQL，查询 progress_tasks 表
```

#### 问题：某些副本收不到消息

**可能原因：**
- 副本的 PostgreSQL Listener 未正确订阅
- 网络问题导致消息丢失

**解决方案：**
```bash
# 重启问题副本
kubectl delete pod <pod-name>

# 查看新 Pod 的启动日志
kubectl logs <new-pod-name> -f
```

## 监控建议

### 关键指标

1. **PostgreSQL 连接数**
   ```sql
   SELECT count(*) FROM pg_stat_activity WHERE datname = 'kube_node_manager';
   ```

2. **通知延迟**
   - 监控日志中的 `Published to PostgreSQL` 和 `Forwarded notification` 时间差

3. **失败率**
   ```sql
   SELECT status, COUNT(*) 
   FROM progress_tasks 
   WHERE created_at > NOW() - INTERVAL '1 hour'
   GROUP BY status;
   ```

### 告警规则

- PostgreSQL Listener 连接失败超过 3 次
- 消息队列积压超过 100 条
- 任务完成率 < 95%

## 升级到 v2.34.9

如果从旧版本升级：

```bash
# 1. 更新镜像
kubectl set image statefulset/kube-node-mgr \
  kube-node-mgr=kube-node-mgr:v2.34.9 \
  -n kube-node-mgr

# 2. 滚动更新
kubectl rollout status statefulset/kube-node-mgr -n kube-node-mgr

# 3. 验证所有副本启动成功
kubectl get pods -l app=kube-node-mgr

# 4. 检查日志确认 PostgreSQL Listener 正常
kubectl logs -l app=kube-node-mgr | grep "PostgreSQL listener connected"
```

## 总结

通过正确配置 PostgreSQL LISTEN/NOTIFY：
- ✅ 无需部署额外的 Redis
- ✅ 实现 < 100ms 的实时消息传递
- ✅ 支持多副本环境的高可用部署
- ✅ 简化运维和故障排查

**关键要点：**
1. 确保 `database.host` 使用可访问的 Service 地址
2. 验证启动日志确认连接成功
3. 测试批量操作验证功能正常
4. 监控 PostgreSQL 连接和性能指标

