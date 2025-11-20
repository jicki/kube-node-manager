# 数据库迁移管理系统

## 快速开始

### ⭐ 自动迁移（推荐）

从 **v2.34.0** 开始，数据库迁移在应用启动时自动执行，无需手动运行迁移工具。

**默认行为：**
- ✅ 启动时自动检测数据库版本
- ✅ 自动执行待迁移脚本
- ✅ 验证数据库结构
- ✅ 自动修复结构问题
- ✅ 迁移失败时退出程序

**配置项：**（在 `config.yaml` 中）

```yaml
database:
  auto_migrate: true            # 启动时自动迁移（默认: true）
  validate_on_startup: true     # 启动时验证结构（默认: true）
  repair_on_startup: true       # 启动时自动修复（默认: true）
  migration_timeout: 300        # 迁移超时（秒，默认: 300）
```

**健康检查接口：**

```bash
# 查看数据库健康状态和版本信息
curl http://localhost:8080/api/health/database

# 查看迁移状态和历史
curl http://localhost:8080/api/health/migration

# 验证数据库结构
curl http://localhost:8080/api/health/schema
```

### 手动迁移（可选）

如需手动管理迁移，可使用命令行工具：

```bash
# 查看版本信息
go run backend/tools/migrate.go -cmd version

# 运行迁移
go run backend/tools/migrate.go -cmd migrate

# 验证数据库结构
go run backend/tools/migrate.go -cmd validate

# 修复数据库结构（干运行）
go run backend/tools/migrate.go -cmd repair --dry-run

# 修复数据库结构（实际执行）
go run backend/tools/migrate.go -cmd repair
```

## 核心模块

| 模块 | 文件 | 功能 |
|------|------|------|
| **自动迁移启动** ⭐ | `auto_migrate.go` | **启动时自动迁移主流程** |
| 表结构定义 | `schema_definition.go` | 定义所有表的完整结构 |
| 版本管理器 | `version_manager.go` | 版本跟踪和迁移路径计算 |
| 结构验证器 | `schema_validator.go` | 验证数据库结构一致性 |
| 自动修复引擎 | `schema_repair.go` | 生成并执行修复 SQL |
| 迁移注册表 | `migration_registry.go` | 管理 23 个迁移脚本 |
| 迁移管理器 | `migration.go` | 执行 SQL 迁移文件 |
| 迁移服务层 | 在 `internal/service/migration_service.go` | 提供迁移状态查询接口 |
| 健康检查接口 | 在 `internal/handler/health/health.go` | HTTP 接口查看迁移状态 |
| 数据库初始化 | `database.go` | 初始化数据库连接 |

## 主要功能

### ⭐ 自动启动迁移（v2.34.0 新增）
- **启动时自动执行**：无需手动运行迁移工具
- **智能检测**：自动判断是否需要迁移
- **完整流程**：GORM AutoMigrate → SQL 迁移 → 结构验证 → 自动修复
- **失败保护**：迁移失败时退出程序，确保数据完整性
- **历史记录**：所有迁移操作记录到 `migration_histories` 表
- **超时控制**：可配置超时时间，防止迁移hang住
- **配置灵活**：可通过配置启用/禁用各项功能

### ✅ 健康检查接口（v2.34.0 新增）
- **版本信息**：`GET /api/health/database` - 查看应用和数据库版本
- **迁移状态**：`GET /api/health/migration` - 查看迁移状态和历史
- **结构验证**：`GET /api/health/schema` - 验证数据库结构

### ✅ 表结构定义和验证
- 28 个表的完整定义
- 支持 PostgreSQL 和 SQLite
- 运行时结构验证

### ✅ 版本管理
- 基于 VERSION 文件的版本跟踪
- 应用版本 → 数据库架构版本映射
- 自动检测待执行迁移

### ✅ 结构验证
- 表、字段、索引验证
- 类型匹配检查
- 分级问题报告（Critical/Warning/Info）

### ✅ 自动修复
- 创建缺失的表和字段
- 添加缺失的索引
- 支持干运行模式

### ✅ 迁移注册
- 23 个迁移脚本的元信息
- 依赖关系管理
- 按分类和版本查询

## 使用示例

### ⭐ 自动迁移（推荐）

在应用启动时自动执行：

```go
import "kube-node-manager/pkg/database"

func main() {
    // 1. 初始化数据库
    db, err := database.InitDatabase(dbConfig)
    if err != nil {
        log.Fatal("Failed to initialize database:", err)
    }
    
    // 2. 运行 GORM AutoMigrate
    if err := model.AutoMigrate(db); err != nil {
        log.Fatal("Failed to run GORM auto-migrations:", err)
    }
    
    // 3. 自动执行数据库迁移（包含验证和修复）
    autoMigrateConfig := database.AutoMigrateConfig{
        Enabled:           true,
        ValidateOnStartup: true,
        RepairOnStartup:   true,
        MigrationTimeout:  300, // 5分钟
    }
    
    if err := database.AutoMigrateOnStartup(db, autoMigrateConfig); err != nil {
        log.Fatal("Database migration failed:", err)
    }
    
    log.Println("✓ Database is ready and up-to-date")
    
    // 4. 启动应用服务...
}
```

### 查询迁移状态

通过 HTTP 接口：

```bash
# 查看数据库健康状态
curl http://localhost:8080/api/health/database

# 返回示例
{
  "status": "healthy",
  "timestamp": "2024-11-20T10:30:00Z",
  "details": {
    "connection": {
      "status": "healthy",
      "data": {...}
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

# 查看迁移历史
curl http://localhost:8080/api/health/migration

# 返回示例
{
  "status": "healthy",
  "timestamp": "2024-11-20T10:30:00Z",
  "app_version": "v2.34.1",
  "db_version": "023",
  "needs_migration": false,
  "migrations_applied": 23,
  "pending_migrations": 0,
  "recent_history": [
    {
      "id": 5,
      "app_version": "v2.34.1",
      "db_version": "023",
      "migration_type": "auto_startup",
      "status": "success",
      "duration_ms": 1234,
      "applied_at": "2024-11-20T10:00:00Z"
    }
  ]
}
```

### 手动验证和修复

```go
import "kube-node-manager/pkg/database"

func main() {
    // 初始化数据库
    db, _ := database.InitDatabase(dbConfig)
    
    // 执行验证和修复
    dbType := database.DatabaseTypePostgreSQL
    err := database.ValidateAndRepair(db, dbType, false)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 检查版本信息

```go
vm, _ := database.NewVersionManager(db, "./VERSION")
info := vm.GetVersionInfo()
fmt.Printf("App: %s, DB: %s\n", info.AppVersion, info.DBVersion)
```

### 生成修复 SQL

```go
sqlStatements, _ := database.GenerateRepairSQL(db, dbType)
for _, sql := range sqlStatements {
    fmt.Println(sql)
}
```

## 测试

运行集成测试：

```bash
cd backend/pkg/database
go test -v
```

测试覆盖：
- 表结构定义完整性
- 版本管理器功能
- 结构验证器
- 修复器（干运行模式）
- 迁移注册表
- 端到端迁移流程

## 版本映射

| 应用版本 | 数据库架构版本 | 迁移数量 |
|----------|----------------|----------|
| v2.34.1  | 023            | 23       |
| v2.34.0  | 022            | 22       |
| v2.33.0  | 021            | 21       |
| v2.32.0  | 019            | 19       |
| v2.31.0  | 017            | 17       |

完整版本映射见 `version_manager.go` 中的 `VersionMapping`。

## 文档

详细文档：[database-migration-system.md](../../../docs/database-migration-system.md)

包含：
- 详细功能说明
- 所有命令用法
- 使用场景和最佳实践
- 故障排查指南
- 性能优化建议
- 安全注意事项

## 架构设计

### 设计原则

1. **向后兼容**：不破坏现有的 MigrationManager
2. **渐进式采用**：可逐步从 GORM AutoMigrate 迁移
3. **安全优先**：所有操作支持 Dry Run
4. **数据库无关**：支持多种数据库类型
5. **版本追踪**：使用 VERSION 文件作为主版本号

### 工作流程

```
1. 读取 VERSION 文件 → 应用版本
2. 查询 schema_migrations → 数据库版本
3. 比较版本 → 确定待执行迁移
4. 验证数据库结构 → 生成差异报告
5. 执行修复操作 → 创建表/字段/索引
6. 记录迁移日志 → 更新版本信息
```

## 贡献

添加新迁移：

1. 创建 SQL 文件：`backend/migrations/024_description.sql`
2. 注册迁移：在 `migration_registry.go` 添加条目
3. 更新版本映射：在 `version_manager.go` 更新 `VersionMapping`
4. 运行测试：`go test -v`
5. 更新文档

## 许可证

与主项目相同

