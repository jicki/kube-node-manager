# 自动数据库迁移系统

> **版本**: v2.34.0+  
> **状态**: ✅ 生产就绪

## 概述

从 v2.34.0 开始，Kube Node Manager 支持在应用启动时自动执行数据库迁移，无需手动运行迁移工具。

### 特性

- ✅ **零配置启动**：默认启用，开箱即用
- ✅ **智能检测**：自动判断是否需要迁移
- ✅ **完整流程**：GORM AutoMigrate → SQL 迁移 → 结构验证 → 自动修复
- ✅ **失败保护**：迁移失败时退出程序，确保数据完整性
- ✅ **历史记录**：所有迁移操作记录到数据库
- ✅ **健康检查**：提供 HTTP 接口查看迁移状态
- ✅ **超时控制**：可配置超时时间，防止迁移hang住
- ✅ **多实例安全**：使用数据库锁避免并发迁移

## 工作流程

```
应用启动
   ↓
初始化数据库连接
   ↓
运行 GORM AutoMigrate (创建/更新基础表结构)
   ↓
┌─────────────────────────────────────────┐
│     自动迁移系统 (AutoMigrateOnStartup)  │
├─────────────────────────────────────────┤
│ 1. 检查配置 (是否启用自动迁移)          │
│ 2. 初始化版本管理器                     │
│ 3. 读取应用版本 (VERSION 文件)         │
│ 4. 查询数据库版本 (schema_migrations)  │
│ 5. 计算待执行迁移                      │
│ 6. 执行 SQL 迁移脚本                   │
│ 7. 验证数据库结构                      │
│ 8. 自动修复结构问题 (如启用)          │
│ 9. 记录迁移历史到数据库                │
└─────────────────────────────────────────┘
   ↓
迁移成功 ✓
   ↓
启动应用服务
```

## 配置

### 配置项

在 `config.yaml` 中添加以下配置：

```yaml
database:
  # ... 其他数据库配置 ...
  
  # 自动迁移配置（v2.34.0+）
  auto_migrate: true            # 启动时自动迁移（默认: true）
  validate_on_startup: true     # 启动时验证结构（默认: true）
  repair_on_startup: true       # 启动时自动修复（默认: true）
  migration_timeout: 300        # 迁移超时（秒，默认: 300，0 表示不限制）
```

### 配置说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `auto_migrate` | bool | `true` | 是否启用自动迁移 |
| `validate_on_startup` | bool | `true` | 是否验证数据库结构 |
| `repair_on_startup` | bool | `true` | 是否自动修复结构问题 |
| `migration_timeout` | int | `300` | 迁移超时时间（秒），0 表示不限制 |

### 禁用自动迁移

如果需要手动控制迁移，可以禁用自动迁移：

```yaml
database:
  auto_migrate: false
```

然后使用命令行工具手动执行迁移：

```bash
go run backend/tools/migrate.go -cmd migrate
```

## 启动日志

### 成功示例

```
========================================
Starting Database Migration on Startup
========================================
Initializing version manager...
Application Version: v2.34.1
Database Version:    023
Latest Schema:       023
✓ Database is up-to-date, no migration needed
Database Type: postgres

--- Validating Database Schema ---
✓ Database schema validation passed

========================================
Database Migration Completed in 0.85s
✓ Database is ready and up-to-date
========================================
```

### 需要迁移示例

```
========================================
Starting Database Migration on Startup
========================================
Initializing version manager...
Application Version: v2.34.1
Database Version:    022
Latest Schema:       023
⚠️  Database migration needed: 1 pending migrations
Pending migrations: [023_add_node_tracking_to_progress.sql]
Database Type: postgres

--- Running SQL Migrations ---
Executing migration: 023_add_node_tracking_to_progress.sql
✓ SQL migrations completed successfully

--- Validating Database Schema ---
✓ Database schema validation passed
✓ Migration history recorded (ID: 15, Duration: 1234ms)

========================================
Database Migration Completed in 2.34s
✓ Database is ready and up-to-date
========================================
```

### 修复问题示例

```
========================================
Starting Database Migration on Startup
========================================
...

--- Validating Database Schema ---
⚠️  Schema validation found issues:
   Critical: 2, Warnings: 1, Total: 3

--- Repairing Database Schema ---
Creating missing table: ansible_workflows
Adding missing column: progress_messages.node_name
Creating missing index: idx_progress_messages_node
✓ Database schema repaired successfully
   Tables Created: 1
   Columns Added: 1
   Indexes Created: 1

========================================
Database Migration Completed in 3.45s
✓ Database is ready and up-to-date
========================================
```

## 健康检查接口

### 1. 数据库健康状态

查看数据库连接状态和版本信息：

```bash
curl http://localhost:8080/api/health/database
```

响应示例：

```json
{
  "status": "healthy",
  "timestamp": "2024-11-20T10:30:00Z",
  "details": {
    "connection": {
      "status": "healthy",
      "data": {
        "max_open_connections": 50,
        "open_connections": 5,
        "in_use": 2,
        "idle": 3
      }
    },
    "version": {
      "app_version": "v2.34.1",
      "db_version": "023",
      "latest_schema": "023",
      "needs_migration": false,
      "migrations_applied": 23,
      "last_migration": "023_add_node_tracking_to_progress.sql",
      "last_migration_time": "2024-11-20T10:00:00Z"
    }
  }
}
```

### 2. 迁移状态

查看迁移状态和历史：

```bash
curl http://localhost:8080/api/health/migration
```

响应示例：

```json
{
  "status": "healthy",
  "timestamp": "2024-11-20T10:30:00Z",
  "app_version": "v2.34.1",
  "db_version": "023",
  "latest_schema": "023",
  "needs_migration": false,
  "migrations_applied": 23,
  "pending_migrations": 0,
  "pending_list": [],
  "last_migration": "023_add_node_tracking_to_progress.sql",
  "last_migration_time": "2024-11-20T10:00:00Z",
  "recent_history": [
    {
      "id": 15,
      "app_version": "v2.34.1",
      "db_version": "023",
      "migration_type": "auto_startup",
      "status": "success",
      "duration_ms": 1234,
      "applied_at": "2024-11-20T10:00:00Z"
    }
  ],
  "statistics": {
    "total_migrations": 23,
    "by_category": {
      "indexes": 3,
      "foreign_keys": 4,
      "features": 9,
      "fixes": 4,
      "performance": 3
    }
  }
}
```

### 3. 结构验证

验证当前数据库结构：

```bash
curl http://localhost:8080/api/health/schema
```

响应示例：

```json
{
  "status": "valid",
  "valid": true,
  "critical_issues": 0,
  "warnings": 0,
  "total_issues": 0,
  "missing_tables": [],
  "extra_tables": [],
  "timestamp": "2024-11-20T10:30:00Z"
}
```

## 迁移历史记录

所有迁移操作都会记录到 `migration_histories` 表中：

### 表结构

```sql
CREATE TABLE migration_histories (
    id SERIAL PRIMARY KEY,
    version VARCHAR(255),
    app_version VARCHAR(50),
    db_version VARCHAR(50),
    migration_type VARCHAR(50) NOT NULL,  -- sql, auto_repair, gorm, auto_startup
    status VARCHAR(20) NOT NULL DEFAULT 'success',  -- success, failed, pending
    duration_ms BIGINT DEFAULT 0,
    error_message TEXT,
    applied_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### 查询示例

```sql
-- 查看最近 10 次迁移
SELECT * FROM migration_histories ORDER BY applied_at DESC LIMIT 10;

-- 查看失败的迁移
SELECT * FROM migration_histories WHERE status = 'failed';

-- 查看特定版本的迁移
SELECT * FROM migration_histories WHERE app_version = 'v2.34.1';
```

## 多实例部署

### 并发迁移保护

系统使用数据库行锁来防止多个实例同时执行迁移：

1. **第一个实例**：获取锁，执行迁移
2. **其他实例**：等待第一个实例完成
3. **超时机制**：防止死锁，默认 5 分钟超时

### 部署步骤

1. 确保所有实例使用相同的配置
2. 启动第一个实例，等待迁移完成
3. 启动其他实例，会自动检测迁移已完成

## 故障排查

### 迁移失败

**症状**：应用启动失败，日志显示 "Database migration failed"

**排查步骤**：

1. 查看详细错误日志
2. 检查数据库连接是否正常
3. 检查数据库用户权限（需要 DDL 权限）
4. 查看 `migration_histories` 表中的错误信息
5. 尝试手动执行失败的迁移脚本

**解决方案**：

```bash
# 1. 查看迁移状态
go run backend/tools/migrate.go -cmd status

# 2. 验证数据库结构
go run backend/tools/migrate.go -cmd validate

# 3. 手动修复（dry-run）
go run backend/tools/migrate.go -cmd repair --dry-run

# 4. 手动修复（实际执行）
go run backend/tools/migrate.go -cmd repair
```

### 迁移超时

**症状**：应用启动超过 5 分钟，日志显示 "migration timeout"

**原因**：
- 迁移脚本执行时间过长
- 数据库性能问题
- 网络延迟

**解决方案**：

增加超时时间：

```yaml
database:
  migration_timeout: 600  # 10 分钟
```

或者禁用超时：

```yaml
database:
  migration_timeout: 0  # 不限制
```

### 结构验证失败

**症状**：验证发现结构问题，但自动修复失败

**排查步骤**：

1. 查看具体的验证错误
2. 检查是否是无法自动修复的问题（如字段类型不匹配）
3. 手动执行修复 SQL

**解决方案**：

```bash
# 1. 查看详细验证报告
go run backend/tools/migrate.go -cmd validate

# 2. 生成修复 SQL（dry-run）
go run backend/tools/migrate.go -cmd repair --dry-run

# 3. 检查生成的 SQL 是否正确
# 4. 手动执行修复
go run backend/tools/migrate.go -cmd repair
```

## 性能影响

### 启动延迟

| 场景 | 延迟时间 | 说明 |
|------|----------|------|
| 无需迁移 | < 1 秒 | 只进行版本检查和验证 |
| 小型迁移 (1-2 个脚本) | 1-5 秒 | 执行少量 SQL 语句 |
| 大型迁移 (5+ 个脚本) | 5-30 秒 | 执行多个迁移脚本 |
| 首次部署 | 10-60 秒 | 创建所有表和索引 |

### 资源占用

- **内存**：增加 < 10MB（缓存结构定义）
- **CPU**：启动时短暂高峰，后续可忽略
- **数据库连接**：使用应用的数据库连接池

### 优化建议

1. **合并小型迁移**：将多个小改动合并为一个迁移脚本
2. **异步索引创建**：对大表创建索引时使用 `CREATE INDEX CONCURRENTLY`（PostgreSQL）
3. **批量操作**：大量数据变更时使用批量 SQL
4. **监控超时**：根据实际情况调整 `migration_timeout`

## 最佳实践

### 1. 开发环境

```yaml
database:
  auto_migrate: true
  validate_on_startup: true
  repair_on_startup: true
  migration_timeout: 60
```

### 2. 测试环境

```yaml
database:
  auto_migrate: true
  validate_on_startup: true
  repair_on_startup: true
  migration_timeout: 120
```

### 3. 生产环境

```yaml
database:
  auto_migrate: true
  validate_on_startup: true
  repair_on_startup: false  # 生产环境建议先验证，手动修复
  migration_timeout: 300
```

### 4. 多副本生产环境

```yaml
database:
  type: postgres  # 必须使用 PostgreSQL
  auto_migrate: true
  validate_on_startup: true
  repair_on_startup: false
  migration_timeout: 600  # 增加超时，应对多实例等待
```

## 向后兼容

### 保留手动迁移工具

`migrate` 工具仍然可用，适用于：

- 手动控制迁移时机
- 生产环境预先测试迁移
- 故障排查和修复
- CI/CD 流程中的迁移

### 迁移到自动迁移

从手动迁移切换到自动迁移：

1. 确保所有待迁移脚本已执行
2. 更新配置文件，启用 `auto_migrate`
3. 重启应用，观察启动日志
4. 通过健康检查接口验证

## 安全考虑

### 1. 权限检查

确保数据库用户有足够权限：

```sql
-- PostgreSQL
GRANT CREATE, ALTER, DROP ON DATABASE kube_node_manager TO kube_node_mgr;

-- 或授予 SCHEMA 级别权限
GRANT CREATE, USAGE ON SCHEMA public TO kube_node_mgr;
```

### 2. 备份

在执行迁移前，建议备份数据库：

```bash
# PostgreSQL
pg_dump -h localhost -U postgres kube_node_manager > backup_$(date +%Y%m%d_%H%M%S).sql

# SQLite
cp data/kube-node-manager.db data/kube-node-manager.db.backup_$(date +%Y%m%d_%H%M%S)
```

### 3. 审计

所有迁移操作都会记录到 `migration_histories` 表，包括：
- 执行时间
- 应用版本
- 数据库版本
- 执行耗时
- 错误信息（如有）

## 相关文档

- [数据库迁移系统概览](../backend/pkg/database/README.md)
- [迁移系统详细文档](database-migration-system.md)
- [多副本部署配置](../configs/config-multi-replica.yaml)
- [变更日志](CHANGELOG.md)

## 常见问题

### Q: 自动迁移会影响启动时间吗？

A: 会有轻微影响，但可接受：
- 无需迁移时：< 1 秒
- 有待执行迁移时：1-30 秒（取决于迁移脚本数量和复杂度）

### Q: 可以禁用自动迁移吗？

A: 可以，设置 `database.auto_migrate: false` 即可。

### Q: 多实例部署时如何避免并发迁移？

A: 系统使用数据库行锁自动处理，第一个实例执行迁移，其他实例等待。

### Q: 迁移失败了怎么办？

A: 
1. 查看详细错误日志
2. 使用 `migrate` 工具手动排查和修复
3. 查看 `migration_histories` 表中的错误信息
4. 必要时回滚到备份

### Q: 如何验证迁移是否成功？

A: 
1. 查看启动日志
2. 访问 `/api/health/migration` 接口
3. 查询 `migration_histories` 表

### Q: 生产环境建议的配置是什么？

A: 建议启用自动迁移和验证，但禁用自动修复，先手动验证修复 SQL。

## 更新日志

### v2.34.0 (2024-11-20)

- ✨ 新增：自动迁移系统
- ✨ 新增：健康检查接口（/api/health/database, /api/health/migration）
- ✨ 新增：迁移历史记录表（migration_histories）
- ✨ 新增：迁移服务层（MigrationService）
- 📝 文档：完整的自动迁移文档


