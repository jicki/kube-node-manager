# 数据库配置指南

本文档详细说明了如何正确配置 Kube Node Manager 的数据库连接。

## 支持的数据库类型

- **SQLite**: 适合开发环境和小规模部署
- **PostgreSQL**: 适合生产环境和高并发场景

## 配置方法

### SQLite 配置

```yaml
database:
  type: "sqlite"
  dsn: "./data/kube-node-manager.db"  # 或 ":memory:" 用于内存数据库
```

### PostgreSQL 配置

有两种配置方法：

#### 方法一：使用单独参数（推荐）

```yaml
database:
  type: "postgres"
  # 注意：dsn 字段留空或删除
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

#### 方法二：使用完整 DSN

```yaml
database:
  type: "postgres"
  dsn: "host=localhost port=5432 user=postgres dbname=kube_node_manager sslmode=disable password=your-password"
  # 如果使用 DSN，上面的单独参数将被忽略
```

## 常见错误及解决方案

### 错误：failed to parse as DSN (invalid dsn)

**错误信息示例**：
```
Failed to initialize database:failed to initialize database: failed to connect to PostgreSQL: cannot parse `./data/kube-node-manager.db`: failed to parse as DSN (invalid dsn)
```

**原因**：
- 数据库类型设置为 `postgres`
- 但 `dsn` 字段仍然是 SQLite 的文件路径格式

**解决方案**：
1. **清空 DSN 字段**（推荐）：
   ```yaml
   database:
     type: "postgres"
     dsn: ""  # 清空或删除这一行
     host: "localhost"
     port: 5432
     # ... 其他 PostgreSQL 配置
   ```

2. **使用正确的 PostgreSQL DSN**：
   ```yaml
   database:
     type: "postgres"
     dsn: "host=localhost port=5432 user=postgres dbname=kube_node_manager sslmode=disable password=your-password"
   ```

### 错误：connection refused

**解决方案**：
- 确保 PostgreSQL 服务正在运行
- 检查主机地址和端口是否正确
- 检查防火墙设置

### 错误：authentication failed

**解决方案**：
- 检查用户名和密码是否正确
- 确保用户有访问指定数据库的权限

## 环境变量覆盖

可以使用环境变量覆盖配置文件中的数据库设置：

```bash
# 数据库基本配置
export DATABASE_TYPE="postgres"
export DATABASE_HOST="localhost"
export DATABASE_PORT="5432"
export DATABASE_DATABASE="kube_node_manager"
export DATABASE_USERNAME="postgres"
export DATABASE_PASSWORD="your-password"
export DATABASE_SSL_MODE="disable"

# 连接池配置
export DATABASE_MAX_OPEN_CONNS="25"
export DATABASE_MAX_IDLE_CONNS="10"
export DATABASE_MAX_LIFETIME="3600"

# 完整 DSN 方式
export DATABASE_URL="host=localhost port=5432 user=postgres dbname=kube_node_manager sslmode=disable password=your-password"
```

## 生产环境最佳实践

### 1. 安全配置

```yaml
database:
  type: "postgres"
  host: "db.example.com"
  port: 5432
  database: "kube_node_manager"
  username: "knm_user"  # 使用专用用户，而不是 postgres 超级用户
  password: "secure-complex-password"
  ssl_mode: "require"   # 启用 SSL
```

### 2. 性能优化

```yaml
database:
  max_open_conns: 50    # 根据服务器性能调整
  max_idle_conns: 20    # 通常设为 max_open_conns 的 40%
  max_lifetime: 7200    # 2小时，避免长连接问题
```

### 3. 监控配置

启用数据库连接监控：
```yaml
logging:
  format: "json"
  level: "info"
  structured: true

monitoring:
  enabled: true
  path: "/metrics"
```

## Docker 部署配置

### docker-compose.yml 示例

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: kube_node_manager
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: docker123
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  kube-node-manager:
    image: kube-node-manager:latest
    environment:
      DATABASE_TYPE: postgres
      DATABASE_HOST: postgres  # 服务名
      DATABASE_PORT: 5432
      DATABASE_DATABASE: kube_node_manager
      DATABASE_USERNAME: postgres
      DATABASE_PASSWORD: docker123
      DATABASE_SSL_MODE: disable
    depends_on:
      - postgres
    ports:
      - "8080:8080"

volumes:
  postgres_data:
```

## Kubernetes 部署配置

### ConfigMap 示例

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: knm-config
data:
  config.yaml: |
    database:
      type: "postgres"
      host: "postgres-service"
      port: 5432
      database: "kube_node_manager"
      ssl_mode: "require"
      max_open_conns: 25
      max_idle_conns: 10
      max_lifetime: 3600
```

### Secret 示例

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: knm-db-secret
type: Opaque
stringData:
  username: postgres
  password: your-secure-password
```

## 数据迁移

如果需要从 SQLite 迁移到 PostgreSQL，请使用提供的迁移工具：

```bash
cd scripts
./migrate.sh -c migration.yaml
```

详细迁移说明请参考：[README_MIGRATION.md](../scripts/README_MIGRATION.md)

## 故障排除

### 1. 连接测试

```bash
# 测试 PostgreSQL 连接
psql -h localhost -p 5432 -U postgres -d kube_node_manager -c "SELECT version();"
```

### 2. 查看连接状态

```sql
-- 查看当前连接数
SELECT count(*) FROM pg_stat_activity WHERE datname = 'kube_node_manager';

-- 查看连接详情
SELECT pid, usename, datname, client_addr, state, query_start 
FROM pg_stat_activity 
WHERE datname = 'kube_node_manager';
```

### 3. 启用调试日志

临时启用调试模式查看详细连接信息：

```yaml
server:
  mode: "debug"  # 仅用于调试，生产环境请使用 "release"

logging:
  level: "debug"
```

### 4. 健康检查

应用启动后，可以通过健康检查端点验证数据库连接：

```bash
# 基础健康检查
curl http://localhost:8080/health/

# 详细健康检查（包含数据库连接信息）
curl http://localhost:8080/health/detailed
```

## 配置示例文件

项目提供了多个配置示例文件：

- `config.yaml.example` - 基础配置示例
- `config-postgres.yaml.example` - PostgreSQL 专用配置示例

选择合适的示例文件复制为 `config.yaml` 并根据实际环境修改。
