# SQLite to PostgreSQL 数据迁移工具

本工具用于将 Kube Node Manager 的 SQLite 数据库迁移到 PostgreSQL 数据库。

## 功能特性

- ✅ 支持完整的数据迁移（用户、集群、模板、审计日志）
- ✅ 支持配置文件和环境变量两种配置方式
- ✅ 自动创建目标表结构
- ✅ 批量数据处理，提高迁移效率
- ✅ 详细的迁移统计和错误报告
- ✅ 自动备份源数据库
- ✅ 支持预览模式
- ✅ 完整的错误处理和日志记录

## 迁移的数据表

1. **users** - 用户表
2. **clusters** - 集群表
3. **label_templates** - 标签模板表
4. **taint_templates** - 污点模板表
5. **audit_logs** - 审计日志表

## 使用方法

### 方法一：使用配置文件（推荐）

1. 复制配置文件模板：
```bash
cd scripts
cp migration.yaml.example migration.yaml
```

2. 编辑配置文件 `migration.yaml`：
```yaml
sqlite:
  path: "./backend/data/kube-node-manager.db"

postgresql:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "your-password"
  database: "kube_node_manager"
  ssl_mode: "disable"
```

3. 执行迁移：
```bash
./migrate.sh -c migration.yaml
```

### 方法二：使用命令行参数

```bash
./migrate.sh \
  -s ./backend/data/kube-node-manager.db \
  -h localhost \
  -p 5432 \
  -d kube_node_manager \
  -u postgres \
  -w your-password
```

### 方法三：使用环境变量

```bash
export MIGRATION_SQLITE_PATH="./backend/data/kube-node-manager.db"
export MIGRATION_POSTGRESQL_HOST="localhost"
export MIGRATION_POSTGRESQL_PORT="5432"
export MIGRATION_POSTGRESQL_USERNAME="postgres"
export MIGRATION_POSTGRESQL_PASSWORD="your-password"
export MIGRATION_POSTGRESQL_DATABASE="kube_node_manager"
export MIGRATION_POSTGRESQL_SSL_MODE="disable"

./migrate.sh
```

## 命令行选项

```
Usage: ./migrate.sh [options]

Options:
    -c, --config FILE       配置文件路径 (默认: ./migration.yaml)
    -s, --source PATH       SQLite 数据库路径
    -h, --host HOST         PostgreSQL 主机地址
    -p, --port PORT         PostgreSQL 端口
    -d, --database DB       PostgreSQL 数据库名
    -u, --username USER     PostgreSQL 用户名
    -w, --password PASS     PostgreSQL 密码
    --ssl-mode MODE         SSL 模式 (disable/require/verify-ca/verify-full)
    --dry-run              预览模式，不执行实际迁移
    --help                 显示帮助信息
```

## 环境变量

| 变量名 | 说明 | 默认值 |
|-------|------|--------|
| `MIGRATION_SQLITE_PATH` | SQLite 数据库路径 | `./backend/data/kube-node-manager.db` |
| `MIGRATION_POSTGRESQL_HOST` | PostgreSQL 主机地址 | `localhost` |
| `MIGRATION_POSTGRESQL_PORT` | PostgreSQL 端口 | `5432` |
| `MIGRATION_POSTGRESQL_USERNAME` | PostgreSQL 用户名 | `postgres` |
| `MIGRATION_POSTGRESQL_PASSWORD` | PostgreSQL 密码 | - |
| `MIGRATION_POSTGRESQL_DATABASE` | PostgreSQL 数据库名 | `kube_node_manager` |
| `MIGRATION_POSTGRESQL_SSL_MODE` | SSL 模式 | `disable` |

## 迁移前准备

### 1. 准备 PostgreSQL 数据库

确保 PostgreSQL 服务运行，并创建目标数据库：

```sql
-- 登录到 PostgreSQL
psql -h localhost -U postgres

-- 创建数据库
CREATE DATABASE kube_node_manager;

-- 创建用户（可选）
CREATE USER kube_user WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE kube_node_manager TO kube_user;
```

### 2. 备份现有数据

脚本会自动备份 SQLite 数据库，但建议手动备份：

```bash
# 备份 SQLite 数据库
cp backend/data/kube-node-manager.db backend/data/kube-node-manager.db.backup

# 如果有 PostgreSQL 数据，也建议备份
pg_dump -h localhost -U postgres kube_node_manager > postgres_backup.sql
```

## 迁移输出示例

```
[INFO] SQLite to PostgreSQL 迁移工具

[INFO] 检查依赖...
[SUCCESS] 依赖检查通过
[INFO] 构建迁移工具...
[SUCCESS] 迁移工具构建完成
[INFO] 备份 SQLite 数据库到 ./backend/data/kube-node-manager.db.backup.20241207_143022
[SUCCESS] SQLite 数据库备份完成
[SUCCESS] 准备连接 PostgreSQL
[INFO] 开始数据迁移...

Connected to SQLite database: ./backend/data/kube-node-manager.db
Connected to PostgreSQL database: localhost:5432/kube_node_manager
Creating tables in PostgreSQL...
Tables created successfully

Migrating users...
Users migration completed: 3 success, 0 failed
Migrating clusters...
Clusters migration completed: 2 success, 0 failed
Migrating label templates...
Label templates migration completed: 5 success, 0 failed
Migrating taint templates...
Taint templates migration completed: 3 success, 0 failed
Migrating audit logs...
Audit logs migration completed: 150 success, 0 failed

============================================================
MIGRATION SUMMARY
============================================================
TABLE                    TOTAL    SUCCESS     FAILED    SKIPPED
------------------------------------------------------------
users                        3          3          0          0
clusters                     2          2          0          0
label_templates              5          5          0          0
taint_templates              3          3          0          0
audit_logs                 150        150          0          0
------------------------------------------------------------
TOTAL                      163        163          0          0
============================================================
✅ Migration completed successfully!

[SUCCESS] 数据迁移完成!
[SUCCESS] 迁移流程完成!
```

## 错误处理

### 常见错误及解决方法

1. **SQLite 数据库不存在**
   ```
   Error: SQLite database file does not exist: ./backend/data/kube-node-manager.db
   ```
   - 检查 SQLite 数据库路径是否正确
   - 确保应用至少运行过一次以创建数据库

2. **PostgreSQL 连接失败**
   ```
   Error: failed to ping PostgreSQL: connection refused
   ```
   - 检查 PostgreSQL 服务是否运行
   - 检查主机地址、端口、用户名、密码是否正确
   - 检查防火墙设置

3. **权限错误**
   ```
   Error: permission denied for database
   ```
   - 确保 PostgreSQL 用户有创建表的权限
   - 使用具有足够权限的用户或创建专用用户

4. **数据冲突**
   ```
   Error: duplicate key value violates unique constraint
   ```
   - 目标数据库可能已有数据
   - 清空目标数据库或使用新的数据库

## 验证迁移结果

迁移完成后，可以通过以下方式验证：

### 1. 检查表结构
```sql
-- 连接到 PostgreSQL
psql -h localhost -U postgres kube_node_manager

-- 查看所有表
\dt

-- 查看表结构
\d users
\d clusters
\d label_templates
\d taint_templates
\d audit_logs
```

### 2. 检查数据数量
```sql
-- 检查各表记录数量
SELECT 'users' as table_name, COUNT(*) as count FROM users
UNION ALL
SELECT 'clusters', COUNT(*) FROM clusters
UNION ALL
SELECT 'label_templates', COUNT(*) FROM label_templates
UNION ALL
SELECT 'taint_templates', COUNT(*) FROM taint_templates
UNION ALL
SELECT 'audit_logs', COUNT(*) FROM audit_logs;
```

### 3. 测试应用连接

更新应用配置使用 PostgreSQL：
```yaml
# backend/configs/config.yaml
database:
  type: "postgres"
  host: "localhost"
  port: 5432
  database: "kube_node_manager"
  username: "postgres"
  password: "your-password"
  ssl_mode: "disable"
```

启动应用并验证功能正常。

## 注意事项

1. **数据一致性**：迁移过程中建议停止应用服务，避免数据不一致
2. **性能考虑**：大量数据迁移可能需要较长时间，建议在低峰期进行
3. **回滚计划**：保留 SQLite 备份，以便必要时回滚
4. **权限设置**：确保 PostgreSQL 用户权限足够但不过度授权
5. **SSL 配置**：生产环境建议启用 SSL 连接

## 故障排除

如果迁移失败，请检查：

1. **日志文件**：查看详细错误信息
2. **网络连接**：确保可以连接到 PostgreSQL
3. **磁盘空间**：确保有足够空间存储数据
4. **版本兼容**：确保 PostgreSQL 版本支持所需功能
5. **字符编码**：确保字符编码设置正确

## 性能优化

对于大量数据的迁移，可以考虑：

1. **调整批处理大小**：修改代码中的 `batchSize` 参数
2. **并行处理**：为不相关的表启用并行迁移
3. **索引优化**：迁移完成后再创建索引
4. **连接池设置**：调整数据库连接参数

## 联系支持

如果遇到问题，请提供：

1. 完整的错误日志
2. 数据库版本信息
3. 操作系统和 Go 版本
4. 数据量大小和硬件配置
