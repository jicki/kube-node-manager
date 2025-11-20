# 基于代码的数据库迁移系统

> **版本**: v2.34.2+  
> **状态**: ✅ 生产就绪

## 概述

从 v2.34.2 开始，Kube Node Manager 采用全新的基于代码的数据库迁移系统：

- ✅ **表结构来自 GORM 模型**：不再维护重复的 schema 定义
- ✅ **版本号自动生成**：基于表结构 checksum，无需手动维护
- ✅ **迁移代码化**：使用 Go 函数而非 SQL 文件
- ✅ **版本存数据库**：在 `system_metadata` 表中，不依赖 VERSION 文件
- ✅ **自动过渡**：从旧的 SQL 迁移系统平滑过渡

## 快速开始

### 自动迁移（推荐）

应用启动时自动执行迁移，无需任何操作：

```bash
# 启动应用即可
./kube-node-manager
```

**启动日志示例：**

```
========================================
Starting Code-Based Database Migration
========================================
Initializing system metadata...
--- Running GORM AutoMigrate ---
✓ GORM AutoMigrate completed
--- Initializing Version Manager ---
Application Version:    v2.34.2
Current Schema (DB):    a1b2c3d4
Target Schema (Code):   a1b2c3d4
✓ Schema version is up-to-date
--- Executing Code Migrations ---
✓ No pending code migrations
--- Validating Database Schema ---
✓ Database schema validation passed
========================================
Database Migration Completed in 0.85s
✓ Database is ready and up-to-date
========================================
```

### 配置选项

在 `config.yaml` 中：

```yaml
database:
  type: postgres
  # ... 其他数据库配置 ...
  
  # 自动迁移配置
  auto_migrate: true            # 启动时自动迁移（默认: true）
  validate_on_startup: true     # 启动时验证结构（默认: true）
  repair_on_startup: true       # 启动时自动修复（默认: true）
  migration_timeout: 300        # 迁移超时（秒，默认: 300）
```

### 健康检查接口

```bash
# 数据库版本信息
curl http://localhost:8080/api/health/database

# 迁移状态和历史
curl http://localhost:8080/api/health/migration

# 数据库结构验证
curl http://localhost:8080/api/health/schema
```

## 核心概念

### 1. Schema Checksum

每次应用启动时，系统会：
1. 从 GORM 模型提取所有表结构
2. 计算表结构的 SHA256 checksum（取前 8 位 hex）
3. 与数据库中存储的版本号对比
4. 如不一致，执行必要的迁移

**示例 checksum**: `a1b2c3d4`

### 2. System Metadata 表

替代 VERSION 文件，所有版本信息存储在数据库中：

| Key | Value | 说明 |
|-----|-------|------|
| `schema_version` | `a1b2c3d4` | 当前 schema checksum |
| `app_version` | `v2.34.2` | 应用版本 |
| `migration_system` | `code_based` | 迁移系统类型 |
| `last_sql_migration` | `023` | 最后的 SQL 迁移（过渡用） |

### 3. 代码迁移

使用 Go 函数定义迁移，而非 SQL 文件：

```go
// 在 pkg/database/code_migrations.go 中
var CodeMigrations = []CodeMigration{
    {
        ID:          "M001",
        Description: "添加示例字段",
        DependsOn:   []string{},
        UpFunc: func(db *gorm.DB) error {
            return db.Exec("ALTER TABLE example ADD COLUMN new_field VARCHAR(255)").Error
        },
        DownFunc: func(db *gorm.DB) error {
            return db.Exec("ALTER TABLE example DROP COLUMN new_field").Error
        },
        CreatedAt: time.Date(2024, 11, 20, 0, 0, 0, 0, time.UTC),
    },
}
```

## 核心模块

| 模块 | 文件 | 功能 |
|------|------|------|
| **System Metadata** | `system_metadata.go` | 系统元数据表和辅助函数 |
| **Schema Extraction** | `schema_definition.go` | 从 GORM 模型提取表结构 |
| **Code Migrations** | `code_migrations.go` | 代码迁移注册和执行 |
| **Version Manager** | `version_manager.go` | 版本管理（基于 checksum） |
| **Auto Migration** | `auto_migrate.go` | 启动时自动迁移流程 |
| **Migration Service** | `internal/service/migration_service.go` | 迁移服务API |
| **Health Check** | `internal/handler/health/health.go` | HTTP健康检查接口 |

## 使用示例

### 添加新的代码迁移

1. 在 `pkg/database/code_migrations.go` 中注册：

```go
var CodeMigrations = []CodeMigration{
    {
        ID:          "M002",  // 唯一ID
        Description: "为用户表添加手机号字段",
        DependsOn:   []string{},  // 依赖的迁移ID
        UpFunc: func(db *gorm.DB) error {
            // 升级逻辑
            return db.Exec(`
                ALTER TABLE users 
                ADD COLUMN phone VARCHAR(20)
            `).Error
        },
        DownFunc: func(db *gorm.DB) error {
            // 回滚逻辑（可选）
            return db.Exec(`
                ALTER TABLE users 
                DROP COLUMN phone
            `).Error
        },
        CreatedAt: time.Now(),
    },
}
```

2. 重启应用，迁移自动执行

### 查询迁移状态

通过 HTTP 接口：

```bash
curl http://localhost:8080/api/health/migration
```

响应示例：

```json
{
  "status": "healthy",
  "app_version": "v2.34.2",
  "db_version": "a1b2c3d4",
  "latest_schema": "a1b2c3d4",
  "needs_migration": false,
  "migrations_applied": 2,
  "pending_migrations": 0,
  "recent_history": [
    {
      "id": 1,
      "migration_id": "M001",
      "description": "添加示例字段",
      "status": "success",
      "duration_ms": 123,
      "applied_at": "2024-11-20T10:00:00Z"
    }
  ]
}
```

### 修改表结构

1. 修改 `internal/model` 中的 GORM 模型：

```go
type User struct {
    ID       uint   `gorm:"primaryKey"`
    Username string `gorm:"uniqueIndex;not null"`
    Email    string `gorm:"uniqueIndex;not null"`
    Phone    string `gorm:"size:20"`  // 新增字段
    // ...
}
```

2. 重启应用
3. GORM AutoMigrate 自动添加字段
4. Schema checksum 自动更新

## 从旧系统迁移

### 自动过渡

系统会自动检测并执行过渡，无需任何操作。

首次启动时会：
1. 检测 `system_metadata` 表是否存在
2. 如不存在且 `schema_migrations` 存在，执行过渡
3. 从 `schema_migrations` 读取最后的 SQL 迁移版本
4. 保存到 `system_metadata` 表
5. 标记迁移系统为 `code_based`

### 手动过渡（如需要）

如果自动过渡失败，可以手动执行：

```bash
# PostgreSQL
psql -U username -d database_name -f backend/migrations/000_transition_to_code_based.sql

# SQLite
sqlite3 database.db < backend/migrations/000_transition_to_code_based.sql
```

## 与旧系统的区别

| 特性 | 旧系统（SQL 文件） | 新系统（基于代码） |
|------|-------------------|-------------------|
| 表结构定义 | 手动维护 schema_definition.go | 自动从 GORM 模型提取 |
| 版本号 | 手动维护 VERSION 文件 + VersionMapping | 自动计算 checksum |
| 迁移脚本 | SQL 文件 (001_xxx.sql) | Go 函数 (CodeMigration) |
| 版本存储 | schema_migrations 表 + VERSION 文件 | system_metadata 表 |
| 迁移注册 | migration_registry.go | code_migrations.go |
| 执行时机 | 启动时读取 SQL 文件 | 启动时执行 Go 函数 |

## 故障排查

### 迁移失败

**症状**: 应用启动失败，日志显示 "Database migration failed"

**排查步骤**:
1. 查看详细错误日志
2. 检查数据库连接
3. 检查数据库用户权限
4. 查看 `system_metadata` 表
5. 查看 `code_migration_records` 表

**解决方案**:

```sql
-- 查看当前版本
SELECT * FROM system_metadata WHERE key = 'schema_version';

-- 查看迁移记录
SELECT * FROM code_migration_records ORDER BY applied_at DESC;

-- 查看失败的迁移
SELECT * FROM code_migration_records WHERE status = 'failed';
```

### Schema 版本不匹配

**症状**: 日志显示 "Schema version mismatch"

**原因**: 代码中的表结构与数据库不一致

**解决方案**:
1. 让自动修复功能处理（`repair_on_startup: true`）
2. 或手动验证：

```bash
curl http://localhost:8080/api/health/schema
```

### 回滚到旧系统

**不推荐**，但如果必须：

```sql
-- 1. 备份数据库
-- 2. 删除新系统的表
DROP TABLE IF EXISTS system_metadata;
DROP TABLE IF EXISTS code_migration_records;

-- 3. 使用旧版本应用
```

## 最佳实践

1. **Always 备份**: 执行迁移前备份数据库
2. **测试环境先行**: 在测试环境验证后再部署生产
3. **监控迁移**: 使用健康检查接口监控迁移状态
4. **代码审查**: 所有迁移代码需要 code review
5. **回滚准备**: 提供 DownFunc 以支持回滚

## 相关文档

- [自动迁移文档](../../../docs/auto-migration.md) - 详细的自动迁移说明
- [SQL 迁移历史](../../migrations/README.md) - 旧的 SQL 迁移文件（已废弃）
- [健康检查 API](../../../docs/api/health-check.md) - 健康检查接口文档

## 贡献

添加新功能或修复 bug 时：

1. 修改 GORM 模型 (`internal/model`)
2. 如需数据迁移，在 `code_migrations.go` 中注册
3. 运行测试
4. 提交 PR

## 许可证

与主项目相同

---

**最后更新**: 2024-11-20  
**维护者**: Kube Node Manager Team
