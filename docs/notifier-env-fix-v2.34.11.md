# v2.34.11 - PostgreSQL Notifier 环境变量与诊断优化

**发布日期**: 2025-11-20

## 问题描述

### 1. 验证逻辑不准确
在 v2.34.10 及之前的版本中,当 PostgreSQL notifier 连接失败并回退到 polling 模式时,系统日志仍然显示:
```
✅ PostgreSQL notifier verified successfully
```

这给用户造成了误导,让用户以为 PostgreSQL LISTEN/NOTIFY 功能正常工作,但实际上系统已经降级到效率较低的 polling 模式。

### 2. 环境变量未被正确读取
Kubernetes 部署时通常使用以下环境变量配置数据库连接:
```yaml
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
```

但是,Viper 配置库的 `AutomaticEnv()` 默认期望环境变量名为 `DATABASE_HOST`, `DATABASE_PORT` 等(配置键 `database.host` 自动转换为 `DATABASE_HOST`)。

因此,即使在 Kubernetes 中设置了 `DB_HOST`,应用程序仍然使用配置文件中的 `host: "localhost"` 默认值,导致 PostgreSQL Listener 尝试连接到 `localhost:5432` 而不是实际的 PostgreSQL 服务地址。

### 3. 连接失败缺乏诊断信息
当 PostgreSQL Listener 连接失败时,日志仅显示:
```
ERROR: Failed to create PostgreSQL notifier, falling back to polling: 
failed to connect PostgreSQL listener: no connection (host=localhost port=5432)
```

缺少详细的诊断信息,用户难以快速定位问题根因。

---

## 修复内容

### 1. 改进验证逻辑 (`progress/database.go`)

**引入 `NotifierInfo` 结构**:
```go
type NotifierInfo struct {
    Type      string // postgres, redis, polling
    IsHealthy bool   // 是否健康
    IsFallback bool  // 是否是降级模式
}
```

**更新 `VerifyNotifier` 方法签名**:
```go
func (dps *DatabaseProgressService) VerifyNotifier() (*NotifierInfo, error)
```

**验证逻辑优化**:
- 对于 PostgreSQL/Redis notifier,尝试发送测试消息验证连接
- 对于 polling notifier:
  - 如果 `IsFallback: true`,警告用户这是降级模式
  - 如果 `IsFallback: false`,说明这是用户配置的模式

### 2. 修复环境变量绑定 (`config/config.go`)

**添加显式环境变量绑定**:
```go
// 支持 Kubernetes 中常用的 DB_ 前缀
viper.BindEnv("database.host", "DB_HOST", "DATABASE_HOST")
viper.BindEnv("database.port", "DB_PORT", "DATABASE_PORT")
viper.BindEnv("database.database", "DB_DATABASE", "DATABASE_DATABASE", "DB_NAME", "DATABASE_NAME")
viper.BindEnv("database.username", "DB_USERNAME", "DATABASE_USERNAME", "DB_USER", "DATABASE_USER")
viper.BindEnv("database.password", "DB_PASSWORD", "DATABASE_PASSWORD")
viper.BindEnv("database.ssl_mode", "DB_SSL_MODE", "DATABASE_SSL_MODE", "DB_SSLMODE", "DATABASE_SSLMODE")
viper.BindEnv("database.type", "DB_TYPE", "DATABASE_TYPE")
```

**效果**:
- 现在环境变量 `DB_HOST` 和 `DATABASE_HOST` 都可以被正确识别
- 优先级: 环境变量 > 配置文件 > 默认值

### 3. 增强连接诊断 (`progress/notifier.go`)

**初始化时验证主数据库连接**:
```go
// 首先验证 GORM 数据库连接是否可用
sqlDB, err := db.DB()
if err != nil {
    return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
}

if err := sqlDB.Ping(); err != nil {
    return nil, fmt.Errorf("main database connection is unhealthy, cannot create listener: %w (host=%s port=%d)", 
        err, dbConfig.Host, dbConfig.Port)
}
```

**连接失败时输出详细诊断信息**:
```
ERROR: PostgreSQL listener connection failed
ERROR:   Host: prod-cloudinfra-pdb.srv.deeproute.cn:5432
ERROR:   Database: kube_node_manager
ERROR:   Username: postgres
ERROR:   SSL Mode: disable
ERROR:   Password set: true
ERROR:   Error: dial tcp 127.0.0.1:5432: connect: connection refused
ERROR: Please check:
ERROR:   1. Network connectivity: can pods reach prod-cloudinfra-pdb.srv.deeproute.cn:5432?
ERROR:   2. Database credentials are correct
ERROR:   3. PostgreSQL server is running and accepting connections
ERROR:   4. Firewall rules allow connections from pods
```

### 4. 优化验证日志 (`services.go`)

**根据实际状态显示相应日志**:
```go
notifierInfo, err := progressSvc.VerifyNotifier()
if err != nil {
    logger.Errorf("⚠️  Notifier verification failed: %v", err)
    // 根据 notifierInfo.Type 给出针对性的故障排查建议
} else {
    if notifierInfo.IsFallback {
        logger.Warningf("⚠️  Using %s mode as fallback - real-time updates may be delayed in multi-replica environment", notifierInfo.Type)
        logger.Warningf("Consider configuring PostgreSQL or Redis for better performance")
    } else {
        logger.Infof("✅ %s notifier verified successfully - multi-replica progress updates ready", notifierInfo.Type)
    }
}
```

---

## 升级影响

### 受影响的部署场景
- ✅ Kubernetes 多副本环境
- ✅ 使用环境变量配置数据库连接的部署
- ✅ 使用 PostgreSQL LISTEN/NOTIFY 或 Redis Pub/Sub 的部署

### 不受影响的场景
- ✅ 单副本部署
- ✅ 使用配置文件直接配置数据库连接的部署
- ✅ 配置了 `notify_type: "polling"` 的部署

---

## 升级建议

### 1. 检查 Kubernetes 环境变量配置

确保 StatefulSet/Deployment 中正确设置了数据库环境变量:

```yaml
env:
- name: DB_HOST
  value: "postgres-service.default.svc.cluster.local"  # PostgreSQL Service 地址
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

### 2. 验证配置文件

如果使用 ConfigMap 挂载配置文件,确保 `progress.notify_type` 设置正确:

```yaml
# config.yaml
progress:
  enable_database: true
  notify_type: "postgres"  # 或 "redis" / "polling"
  poll_interval: 10000
```

### 3. 重新部署应用

```bash
kubectl rollout restart statefulset/kube-node-mgr -n kube-node-mgr
```

### 4. 检查日志验证修复效果

**成功案例**:
```
INFO: ✅ PostgreSQL listener connected successfully (verified via ping)
INFO: ✅ postgres notifier verified successfully - multi-replica progress updates ready
```

**降级案例**(配置了 PostgreSQL 但连接失败):
```
ERROR: PostgreSQL listener connection failed
ERROR:   Host: postgres-service.default.svc.cluster.local:5432
ERROR:   Database: kube_node_manager
ERROR:   ...
ERROR: Please check:
ERROR:   1. Network connectivity: can pods reach postgres-service.default.svc.cluster.local:5432?
ERROR:   ...
INFO: ⚠️  Using polling mode as fallback - real-time updates may be delayed in multi-replica environment
```

**配置为 Polling 模式**:
```
INFO: Using polling mode for progress updates
INFO: ✅ Polling mode verified (configured mode)
```

---

## 常见问题排查

### Q1: 为什么日志显示 "Using polling mode as fallback"?

**可能原因**:
1. **环境变量未设置**: 检查 K8s YAML 中是否设置了 `DB_HOST`, `DB_PORT` 等
2. **数据库服务不可达**: 检查 PostgreSQL Service 是否存在且可访问
3. **网络策略限制**: 检查 NetworkPolicy 是否允许 Pod 访问数据库
4. **数据库凭据错误**: 检查 Secret 中的数据库密码是否正确

**排查步骤**:
```bash
# 1. 检查环境变量
kubectl exec -it kube-node-mgr-0 -n kube-node-mgr -- env | grep DB_

# 2. 测试数据库连通性
kubectl exec -it kube-node-mgr-0 -n kube-node-mgr -- \
  nc -zv postgres-service.default.svc.cluster.local 5432

# 3. 检查数据库日志
kubectl logs -l app=postgres -n default --tail=50
```

### Q2: 如何确认 PostgreSQL LISTEN/NOTIFY 正常工作?

**验证方法**:
1. 查看应用启动日志,确认看到:
   ```
   ✅ PostgreSQL listener connected successfully (verified via ping)
   ✅ postgres notifier verified successfully
   ```

2. 执行批量节点操作,观察日志:
   ```
   INFO: Sending PostgreSQL notification: task=xxx type=progress user=1
   INFO: PostgreSQL notification sent successfully
   ```

3. 在 PostgreSQL 中查询活跃的监听器:
   ```sql
   SELECT * FROM pg_stat_activity 
   WHERE application_name LIKE '%listener%';
   ```

### Q3: 多副本环境下 Polling 模式有什么影响?

**影响**:
- ❌ 进度更新延迟: 默认 10 秒轮询间隔
- ❌ 数据库负载增加: 每个副本都定期查询数据库
- ❌ 用户体验下降: 进度条更新不够实时

**建议**:
- 强烈建议在多副本环境中使用 PostgreSQL LISTEN/NOTIFY 或 Redis Pub/Sub
- 如果必须使用 Polling,可以缩短 `poll_interval`,但会增加数据库负载

---

## 技术细节

### Viper 环境变量绑定机制

**默认行为** (`AutomaticEnv()`):
- 配置键: `database.host`
- 自动查找: `DATABASE_HOST`
- ❌ 不会自动查找: `DB_HOST`

**显式绑定**:
```go
viper.BindEnv("database.host", "DB_HOST", "DATABASE_HOST")
```
- 配置键: `database.host`
- 优先级: `DB_HOST` > `DATABASE_HOST` > 配置文件 > 默认值

### PostgreSQL Listener DSN 构建

```go
dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s password=%s",
    dbConfig.Host,     // 从环境变量 DB_HOST 或配置文件读取
    dbConfig.Port,     // 从环境变量 DB_PORT 或配置文件读取
    dbConfig.Username, // 从环境变量 DB_USERNAME 或配置文件读取
    dbConfig.Database, // 从环境变量 DB_DATABASE 或配置文件读取
    dbConfig.SSLMode,  // 从环境变量 DB_SSL_MODE 或配置文件读取
    dbConfig.Password, // 从环境变量 DB_PASSWORD 或 Secret 读取
)
```

---

## 总结

v2.34.11 主要解决了多副本 Kubernetes 环境中 PostgreSQL Notifier 的环境变量读取和诊断问题:

1. ✅ 环境变量 `DB_HOST` 等现在可以被正确识别
2. ✅ 验证逻辑准确反映实际的通知器状态
3. ✅ 连接失败时提供详细的诊断信息
4. ✅ 降级到 Polling 模式时给出明确警告

升级后,用户可以更容易地诊断和修复 PostgreSQL LISTEN/NOTIFY 的连接问题,提高多副本环境下的进度推送性能。

