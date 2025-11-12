# 数据库工具

本目录包含数据库管理和维护工具。

## 自动迁移功能（推荐）

从当前版本开始，`kube-node-manager` 已支持**自动数据库迁移**。

应用启动时会自动：
1. 运行 GORM 自动迁移（表结构）
2. 执行 SQL 迁移文件（`backend/migrations/*.sql`）
3. 初始化默认数据

**通常情况下，你无需手动运行迁移工具。**

> 📖 详细文档请参考：[自动迁移功能说明](../docs/auto-migration.md)

## 手动迁移工具

### migrate.go

手动执行数据库迁移的工具，用于调试和管理。

#### 1. 执行迁移

```bash
cd backend

# 方式 1：使用 migrate 命令
go run tools/migrate.go -cmd migrate

# 方式 2：使用 up 命令（别名）
go run tools/migrate.go -cmd up
```

**功能：**
- 自动检测数据库类型（SQLite/PostgreSQL）
- 运行 GORM 自动迁移（创建/更新表结构）
- 执行 SQL 迁移文件（`backend/migrations/*.sql`）
- 跟踪已执行的迁移，避免重复执行
- 显示当前数据库中的表列表

**示例输出：**

```
Starting database migration...
Starting database migration check...
Found 2 pending migration(s) to execute
Executing migration: 020_add_new_feature.sql
Successfully executed migration: 020_add_new_feature.sql
Executing migration: 021_fix_all_foreign_keys.sql
Successfully executed migration: 021_fix_all_foreign_keys.sql
All migrations executed successfully
Database migration completed successfully!

Tables in database:
  - anomaly_report_configs
  - ansible_inventories
  - ansible_logs
  - ansible_schedules
  - ansible_ssh_keys
  - ansible_tasks
  - ansible_templates
  - audit_logs
  - cache_entries
  - clusters
  - feishu_settings
  - feishu_user_mappings
  - feishu_user_sessions
  - gitlab_runners
  - gitlab_settings
  - label_templates
  - node_anomalies
  - progress_messages
  - progress_tasks
  - schema_migrations  ← 迁移跟踪表
  - taint_templates
  - users
```

#### 2. 查看迁移状态

```bash
cd backend
go run tools/migrate.go -cmd status
```

**功能：**
- 显示迁移文件总数
- 显示已执行的迁移数量
- 显示待执行的迁移列表

**示例输出（所有迁移已完成）：**

```
Checking migration status...

=== Migration Status ===
Total migrations:    21
Executed migrations: 21
Pending migrations:  0

All migrations are up to date!
```

**示例输出（有待执行的迁移）：**

```
Checking migration status...

=== Migration Status ===
Total migrations:    21
Executed migrations: 19
Pending migrations:  2

Pending migrations:
  - 020_add_new_feature.sql
  - 021_fix_all_foreign_keys.sql
```

## 支持的命令

| 命令 | 别名 | 说明 |
|------|------|------|
| `migrate` | `up` | 执行所有待执行的迁移 |
| `status` | - | 查看迁移状态 |

## 迁移文件管理

### 文件位置

```
backend/
  └── migrations/
      ├── 001_add_anomaly_indexes.sql
      ├── 002_add_anomaly_analytics.sql
      ├── 003_performance_indexes.sql
      ├── ...
      └── 021_fix_all_foreign_keys.sql
```

### 命名规范

```
<序号>_<描述性名称>.sql
```

- **序号**：三位数字，确保按顺序执行（001, 002, 003...）
- **描述**：使用下划线分隔的英文描述
- **扩展名**：必须是 `.sql`

### 创建新迁移

1. 在 `backend/migrations/` 目录下创建新文件
2. 使用下一个可用的序号（如 `022`）
3. 编写 SQL 语句
4. 启动应用或手动运行迁移工具

**示例迁移文件：**

```sql
-- 022_add_cluster_region.sql

-- 添加区域列
ALTER TABLE clusters ADD COLUMN IF NOT EXISTS region VARCHAR(50);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_clusters_region ON clusters(region);

-- 更新默认值
UPDATE clusters SET region = 'default' WHERE region IS NULL;
```

## 迁移跟踪机制

系统通过 `schema_migrations` 表跟踪已执行的迁移：

```sql
-- 查看已执行的迁移
SELECT * FROM schema_migrations ORDER BY version;
```

**输出示例：**

```
            version             |        applied_at
--------------------------------+----------------------------
 001_add_anomaly_indexes.sql   | 2024-11-01 10:00:00
 002_add_anomaly_analytics.sql | 2024-11-01 10:00:01
 021_fix_all_foreign_keys.sql  | 2024-11-12 14:30:00
```

## 适用场景

虽然应用已支持自动迁移，但以下场景仍建议使用手动工具：

1. **查看迁移状态**
   ```bash
   go run tools/migrate.go -cmd status
   ```

2. **在应用启动前预先执行迁移**（如 Kubernetes Init Container）
   ```bash
   go run tools/migrate.go -cmd migrate
   ```

3. **调试迁移问题**
   ```bash
   go run tools/migrate.go -cmd migrate
   ```

4. **验证新迁移文件**
   ```bash
   # 在测试环境验证
   export DATABASE_NAME=kube_node_manager_test
   go run tools/migrate.go -cmd migrate
   ```

## 故障排查

### 问题 1：迁移执行失败

**症状：**
```
Failed to run SQL migrations: failed to execute migration 021_fix_all_foreign_keys.sql: ...
```

**解决方案：**
1. 查看完整的错误信息
2. 检查迁移文件的 SQL 语法
3. 确认数据库连接正常
4. 检查依赖的表和数据是否存在

### 问题 2：迁移被重复执行

**原因：** `schema_migrations` 表被删除或损坏

**解决方案：**
```sql
-- 查看已执行的迁移
SELECT * FROM schema_migrations ORDER BY version;

-- 如果表为空但数据库已有结构，手动添加已执行的迁移记录
INSERT INTO schema_migrations (version, applied_at) 
VALUES ('001_add_anomaly_indexes.sql', NOW());
```

### 问题 3：跳过某个迁移

如果某个迁移不需要执行（如已手动执行）：

```sql
-- 手动标记为已执行
INSERT INTO schema_migrations (version, applied_at) 
VALUES ('021_fix_all_foreign_keys.sql', NOW());
```

### 问题 4：查看迁移详情

```sql
-- 统计迁移数量
SELECT COUNT(*) FROM schema_migrations;

-- 查看最近执行的迁移
SELECT * FROM schema_migrations 
ORDER BY applied_at DESC 
LIMIT 5;

-- 查看特定迁移是否已执行
SELECT * FROM schema_migrations 
WHERE version = '021_fix_all_foreign_keys.sql';
```

## 注意事项

1. **备份数据**：运行迁移前请备份重要数据
2. **测试环境验证**：在生产环境执行前，先在测试环境验证
3. **不要修改已执行的迁移**：已执行的迁移文件不应该再修改
4. **PostgreSQL 需要先创建数据库**：
   ```bash
   createdb kube_node_manager
   ```
5. **SQLite 会自动创建数据库文件**：默认位置 `./data/kube-node-manager.db`
6. **迁移是幂等的**：多次运行不会造成问题

## 最佳实践

1. **使用描述性的迁移文件名**
2. **每个迁移文件只做一件事**
3. **添加注释说明迁移目的**
4. **测试迁移的幂等性**（能否安全地重复执行）
5. **为关键迁移编写回滚脚本**
6. **迁移文件纳入版本控制**

## 相关文档

- [自动迁移功能详细说明](../docs/auto-migration.md)
- [数据库配置说明](../docs/implementation-summary.md)
- [外键约束修复指南](../../scripts/delete_cluster_safely.sh)

## 常见问题（FAQ）

**Q: 是否还需要手动运行迁移？**

A: 通常不需要。应用启动时会自动执行。手动工具主要用于调试和查看状态。

**Q: 如何查看当前迁移状态？**

A: 运行 `go run tools/migrate.go -cmd status`

**Q: 如何添加新的迁移？**

A: 在 `backend/migrations/` 目录下创建新的 `.sql` 文件，使用下一个序号。

**Q: 迁移失败怎么办？**

A: 检查错误信息，修复 SQL 语句，从 `schema_migrations` 表中删除失败的记录，重新执行。

**Q: 支持迁移回滚吗？**

A: 当前版本不支持自动回滚。需要手动编写反向迁移 SQL 或从备份恢复。

## 其他工具

将来可能添加的工具：

- `backup.go` - 数据库备份工具
- `restore.go` - 数据库恢复工具
- `cleanup.go` - 数据清理工具
- `export.go` - 数据导出工具

