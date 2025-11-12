# 数据库自动迁移功能

## 概述

从当前版本开始，`kube-node-manager` 已实现**自动数据库迁移**功能。应用启动时会自动检测并执行所有待执行的SQL迁移文件，无需手动运行迁移命令。

## 工作原理

### 1. 迁移跟踪机制

系统通过 `schema_migrations` 表跟踪已执行的迁移：

```sql
CREATE TABLE schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL
);
```

每次成功执行迁移后，迁移文件名会被记录到此表中，确保不会重复执行。

### 2. 启动流程

应用启动时按以下顺序执行：

1. **数据库连接初始化** - 建立数据库连接
2. **GORM 自动迁移** - 创建/更新表结构（基于 Go 模型定义）
3. **SQL 迁移文件执行** - 自动检测并执行 `backend/migrations/` 目录下的待执行迁移
4. **初始化默认数据** - 创建默认管理员用户等

### 3. 迁移检测逻辑

```
启动应用
    ↓
检查 migrations/ 目录
    ↓
读取所有 .sql 文件
    ↓
查询 schema_migrations 表
    ↓
对比找出待执行的迁移
    ↓
    ├─ 无待执行迁移 → 跳过（日志：All migrations are up to date）
    └─ 有待执行迁移 → 按文件名顺序执行
                          ↓
                    在事务中执行SQL
                          ↓
                    记录到 schema_migrations
                          ↓
                    继续下一个迁移
```

## 优势

### ✅ 自动化
- 应用启动时自动运行迁移，无需人工干预
- 适合容器化部署和 Kubernetes 环境

### ✅ 幂等性
- 已执行的迁移不会重复运行
- 可以多次启动应用而不会出错

### ✅ 顺序保证
- 按文件名字母顺序执行迁移
- 确保迁移的依赖关系正确

### ✅ 事务安全
- 每个迁移在独立事务中执行
- 失败时自动回滚，不会污染数据库

### ✅ 多实例友好
- 即使多个实例同时启动，迁移也能正确执行
- 数据库级别的锁机制防止并发冲突

## 迁移文件命名规范

迁移文件必须放置在 `backend/migrations/` 目录下，并遵循以下命名规范：

```
<序号>_<描述性名称>.sql
```

### 示例

```
001_add_anomaly_indexes.sql
002_add_anomaly_analytics.sql
003_performance_indexes.sql
021_fix_all_foreign_keys.sql
```

### 命名要求

- **序号**：三位数字，确保按顺序执行（001, 002, 003...）
- **描述**：使用下划线分隔的英文描述
- **扩展名**：必须是 `.sql`

## 使用示例

### 场景 1：全新部署

```bash
# 直接启动应用
./bin/kube-node-manager

# 输出示例：
# Initializing postgres database
# Successfully initialized postgres database
# Starting database migration check...
# Found 21 pending migration(s) to execute
# Executing migration: 001_add_anomaly_indexes.sql
# Successfully executed migration: 001_add_anomaly_indexes.sql
# ...
# Executing migration: 021_fix_all_foreign_keys.sql
# Successfully executed migration: 021_fix_all_foreign_keys.sql
# All migrations executed successfully
```

### 场景 2：已有数据库，添加新迁移

假设数据库已执行到 `019_add_workflow_dag.sql`，现在添加了 `021_fix_all_foreign_keys.sql`：

```bash
# 启动应用
./bin/kube-node-manager

# 输出示例：
# Initializing postgres database
# Successfully initialized postgres database
# Starting database migration check...
# Found 1 pending migration(s) to execute
# Executing migration: 021_fix_all_foreign_keys.sql
# Successfully executed migration: 021_fix_all_foreign_keys.sql
# All migrations executed successfully
```

### 场景 3：所有迁移已完成

```bash
# 启动应用
./bin/kube-node-manager

# 输出示例：
# Initializing postgres database
# Successfully initialized postgres database
# Starting database migration check...
# All migrations are up to date, skipping migration
```

## 手动迁移工具（可选）

虽然应用已支持自动迁移，但仍保留了手动迁移工具用于调试和管理：

### 执行迁移

```bash
cd backend
go run tools/migrate.go -cmd migrate
```

或使用简写：

```bash
go run tools/migrate.go -cmd up
```

### 查看迁移状态

```bash
go run tools/migrate.go -cmd status
```

输出示例：

```
=== Migration Status ===
Total migrations:    21
Executed migrations: 21
Pending migrations:  0

All migrations are up to date!
```

或者如果有待执行的迁移：

```
=== Migration Status ===
Total migrations:    21
Executed migrations: 19
Pending migrations:  2

Pending migrations:
  - 020_add_new_feature.sql
  - 021_fix_all_foreign_keys.sql
```

## 配置说明

迁移管理器在 `cmd/main.go` 中初始化：

```go
migrationManager := database.NewMigrationManager(db, database.MigrationConfig{
    MigrationsPath: "./migrations",  // 迁移文件目录
    UseEmbed:       false,           // 是否使用嵌入文件系统
})

if err := migrationManager.AutoMigrate(); err != nil {
    log.Fatal("Failed to run SQL migrations:", err)
}
```

### 配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `MigrationsPath` | string | `./migrations` | 迁移文件所在目录 |
| `UseEmbed` | bool | `false` | 是否使用 Go embed 嵌入的文件系统 |
| `TableName` | string | `schema_migrations` | 迁移跟踪表名 |

## 故障排查

### 问题 1：迁移执行失败

**症状**：
```
Failed to run SQL migrations: failed to execute migration 021_fix_all_foreign_keys.sql: ...
```

**解决方案**：
1. 检查迁移文件的SQL语法是否正确
2. 检查数据库连接是否正常
3. 查看完整的错误信息，确定具体失败原因
4. 如果是外键约束问题，检查依赖的表和数据是否存在

### 问题 2：迁移目录找不到

**症状**：
```
Migration directory ./migrations does not exist, skipping migration
```

**解决方案**：
1. 确保在正确的工作目录下启动应用（通常是 `backend/` 目录）
2. 检查 `migrations/` 目录是否存在
3. 如果使用容器部署，确保迁移文件已正确复制到镜像中

### 问题 3：重复执行迁移

**症状**：
某个迁移被执行了多次

**原因**：
- `schema_migrations` 表被删除或损坏
- 迁移文件名被修改

**解决方案**：
1. 检查 `schema_migrations` 表的内容：
   ```sql
   SELECT * FROM schema_migrations ORDER BY version;
   ```
2. 如果表为空但数据库已有结构，手动添加已执行的迁移记录：
   ```sql
   INSERT INTO schema_migrations (version, applied_at) 
   VALUES ('001_add_anomaly_indexes.sql', NOW());
   ```

### 问题 4：多实例并发执行

**症状**：
多个应用实例同时启动时，迁移可能冲突

**解决方案**：
- PostgreSQL：数据库级别的锁机制会自动处理
- SQLite：建议单实例部署，或使用 init container 预先执行迁移
- 推荐做法：使用 Kubernetes Job 或 init container 在应用启动前执行迁移

## Docker 部署

### 镜像说明

从当前版本开始，Docker 镜像已包含 `migrations/` 目录，无需额外挂载迁移文件。

**镜像结构**：
```
/app/
  ├── main                    # 主程序二进制文件
  ├── VERSION                 # 版本信息
  ├── migrations/             # 数据库迁移文件（已内置）
  │   ├── 001_xxx.sql
  │   └── 021_xxx.sql
  └── data/                   # 数据目录
```

### 运行容器

```bash
# 使用 PostgreSQL
docker run -d \
  --name kube-node-manager \
  -p 8080:8080 \
  -e DATABASE_TYPE=postgres \
  -e DATABASE_HOST=postgres \
  -e DATABASE_PORT=5432 \
  -e DATABASE_NAME=kube_node_manager \
  -e DATABASE_USERNAME=postgres \
  -e DATABASE_PASSWORD=password \
  kube-node-manager:latest
```

**启动日志**：
```
Found migrations directory at: ./migrations  ← 自动检测到迁移文件
Starting database migration check...
Found 21 pending migration(s) to execute
...
All migrations executed successfully
Server starting on port 8080
```

> 📖 详细的 Docker 部署说明请参考：[Docker 镜像自动迁移支持](../../../docs/docker-migration-support.md)

## Kubernetes 部署建议

### 方案 1：应用启动时自动迁移（推荐）

直接启动应用，迁移会自动执行（镜像中已包含迁移文件）：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-node-manager
spec:
  replicas: 1  # 建议先启动一个实例，确保迁移完成
  template:
    spec:
      containers:
      - name: app
        image: kube-node-manager:latest
        # 应用会在启动时自动执行迁移
        # 迁移文件已内置在镜像中的 /app/migrations
```

### 方案 2：使用 Init Container

如果需要确保迁移在应用启动前完成：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-node-manager
spec:
  template:
    spec:
      initContainers:
      - name: migrate
        image: kube-node-manager:latest
        command: ["/app/migrate-and-exit"]  # 需要创建一个只运行迁移的脚本
        env:
        - name: DATABASE_TYPE
          value: "postgres"
        # ... 其他数据库配置
      
      containers:
      - name: app
        image: kube-node-manager:latest
        # 主应用容器
```

### 方案 3：使用 Kubernetes Job

创建一个独立的迁移 Job：

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: kube-node-manager-migration
spec:
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: migrate
        image: kube-node-manager:latest
        command: ["go", "run", "tools/migrate.go", "-cmd", "migrate"]
        env:
        - name: DATABASE_TYPE
          value: "postgres"
        # ... 其他数据库配置
```

然后等 Job 完成后再部署应用。

## 最佳实践

### 1. 迁移文件编写规范

```sql
-- 021_fix_all_foreign_keys.sql

-- 1. 添加描述性注释
-- 修复所有外键约束，确保集群删除时正确处理依赖

-- 2. 使用事务包装（如果数据库支持）
BEGIN;

-- 3. 先删除旧约束
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_cluster;

-- 4. 添加新约束
ALTER TABLE audit_logs 
ADD CONSTRAINT fk_audit_logs_cluster 
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE SET NULL;

-- 5. 提交事务
COMMIT;
```

### 2. 测试迁移

在生产环境执行前，务必在测试环境验证：

```bash
# 1. 备份数据库
pg_dump -h localhost -U postgres kube_node_manager > backup.sql

# 2. 创建测试环境
createdb kube_node_manager_test
psql kube_node_manager_test < backup.sql

# 3. 修改配置指向测试数据库
export DATABASE_NAME=kube_node_manager_test

# 4. 运行迁移
./bin/kube-node-manager

# 5. 验证迁移结果
go run tools/migrate.go -cmd status
```

### 3. 回滚策略

如果迁移出现问题，需要回滚：

```sql
-- 方法 1：从备份恢复
psql kube_node_manager < backup.sql

-- 方法 2：手动反向操作（需要提前编写回滚SQL）
-- 例如，如果迁移是添加列，回滚就是删除列
ALTER TABLE audit_logs DROP COLUMN IF EXISTS new_column;

-- 方法 3：从 schema_migrations 删除失败的迁移记录，修复SQL后重新执行
DELETE FROM schema_migrations WHERE version = '021_fix_all_foreign_keys.sql';
```

### 4. 版本控制

- 迁移文件必须纳入 Git 版本控制
- 迁移文件一旦提交到主分支，**不应该再修改**
- 如需修复，应创建新的迁移文件

### 5. 团队协作

- 多人开发时，协调迁移文件编号，避免冲突
- 可以使用时间戳作为前缀：`20241112_fix_foreign_keys.sql`
- 定期同步迁移文件到所有开发环境

## 技术细节

### 迁移管理器实现

核心代码位于 `backend/pkg/database/migration.go`：

```go
type MigrationManager struct {
    db     *gorm.DB
    config MigrationConfig
}

func (m *MigrationManager) AutoMigrate() error {
    // 1. 确保迁移跟踪表存在
    // 2. 获取所有迁移文件
    // 3. 获取已执行的迁移
    // 4. 找出待执行的迁移
    // 5. 按顺序执行待执行的迁移
}
```

### 事务处理

每个迁移文件在独立事务中执行：

```go
err := m.db.Transaction(func(tx *gorm.DB) error {
    // 执行SQL语句
    if err := m.executeSQLStatements(sqlDB, sqlContent); err != nil {
        return err  // 自动回滚
    }

    // 记录迁移
    migration := SchemaMigration{
        Version:   filename,
        AppliedAt: time.Now(),
    }
    return tx.Create(&migration).Error
})
```

### SQL 语句解析

迁移管理器会：
1. 移除注释行（以 `--` 开头）
2. 移除空行
3. 按分号 `;` 分割多条语句
4. 逐条执行 SQL 语句

## 常见问题（FAQ）

**Q: 是否还需要手动运行 `go run tools/migrate.go up`？**

A: 不需要。应用启动时会自动执行所有待执行的迁移。但手动工具仍然保留，可用于调试或查看迁移状态。

**Q: 如果我删除了 `schema_migrations` 表会怎样？**

A: 应用启动时会重新创建该表，但会认为所有迁移都未执行。如果数据库已有结构，可能会导致迁移执行失败。建议从备份恢复或手动重建 `schema_migrations` 表。

**Q: 可以修改已执行的迁移文件吗？**

A: 不建议。已执行的迁移文件不应该修改，因为其记录已在 `schema_migrations` 表中。如需修复，应创建新的迁移文件。

**Q: 如何跳过某个迁移？**

A: 手动在 `schema_migrations` 表中插入该迁移的记录：
```sql
INSERT INTO schema_migrations (version, applied_at) 
VALUES ('021_fix_all_foreign_keys.sql', NOW());
```

**Q: 支持迁移回滚吗？**

A: 当前版本不支持自动回滚。需要手动编写反向迁移SQL或从备份恢复。

**Q: 多个应用实例同时启动会有问题吗？**

A: PostgreSQL 有数据库级别的锁机制，通常不会有问题。但建议使用滚动更新策略，确保新实例启动前旧实例已停止。

## 升级说明

### 从旧版本升级

如果你的项目之前手动运行迁移，升级到支持自动迁移的版本后：

1. **首次启动时**，系统会创建 `schema_migrations` 表
2. 由于表为空，系统会认为所有迁移都未执行
3. 如果数据库已有结构，某些迁移可能会失败（如创建已存在的索引）

**解决方案**：

手动初始化 `schema_migrations` 表，标记已执行的迁移：

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL
);

-- 标记已执行的迁移（根据实际情况调整）
INSERT INTO schema_migrations (version, applied_at) VALUES
('001_add_anomaly_indexes.sql', NOW()),
('002_add_anomaly_analytics.sql', NOW()),
-- ... 添加所有已执行的迁移
('019_add_workflow_dag.sql', NOW());
```

然后再启动应用，只有新的迁移会被执行。

## 总结

通过自动迁移功能：

- ✅ **简化部署**：无需手动执行迁移命令
- ✅ **提高可靠性**：幂等性保证不会重复执行
- ✅ **适合容器化**：完美适配 Docker 和 Kubernetes
- ✅ **降低出错率**：减少人工操作，避免遗漏迁移

现在你只需要：
1. 编写迁移文件并放入 `backend/migrations/` 目录
2. 启动应用
3. 坐等迁移自动完成 🎉

