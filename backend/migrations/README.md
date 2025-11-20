# SQL 迁移文件目录（已废弃）

> **⚠️ 重要提示**: 此目录中的 SQL 迁移文件已在 v2.34.2 后废弃，系统已切换到基于代码的迁移系统。

## 状态

- **当前状态**: 废弃（Deprecated）
- **废弃版本**: v2.34.2
- **废弃日期**: 2024-11-20
- **保留原因**: 历史记录和参考

## 新的迁移系统

从 v2.34.2 开始，Kube Node Manager 使用基于代码的迁移系统：

- **表结构定义**: 来自 `internal/model` 的 GORM 模型
- **版本管理**: 基于表结构 checksum 自动生成
- **迁移脚本**: 使用 Go 函数注册在 `pkg/database/code_migrations.go`
- **版本存储**: 存储在数据库的 `system_metadata` 表中
- **自动执行**: 应用启动时自动检测和执行

## 迁移历史

此目录包含以下历史迁移文件（v2.34.1 及之前）：

| 文件 | 版本 | 说明 | 日期 |
|------|------|------|------|
| `001_add_anomaly_indexes.sql` | v2.24.0 | 添加异常索引 | 2024-01-10 |
| `002_add_anomaly_analytics.sql` | v2.25.0 | 添加异常分析 | 2024-01-15 |
| `003_performance_indexes.sql` | v2.25.0 | 性能索引 | 2024-01-20 |
| `004_fix_ansible_foreign_keys_quick.sql` | v2.26.0 | 修复外键（快速版本） | 2024-02-01 |
| `005_cleanup_old_unique_indexes.sql` | v2.26.0 | 清理旧索引 | 2024-02-10 |
| `006_ensure_soft_delete_indexes.sql` | v2.26.0 | 软删除索引 | 2024-02-15 |
| `007_add_ansible_schedules.sql` | v2.27.0 | Ansible 调度 | 2024-03-10 |
| `008_add_retry_and_environment_fields.sql` | v2.27.0 | 重试和环境字段 | 2024-03-15 |
| `009_add_dry_run_field.sql` | v2.28.0 | Dry Run 模式 | 2024-04-01 |
| `010_add_batch_execution_fields.sql` | v2.28.0 | 批量执行字段 | 2024-04-10 |
| `011_add_favorites_and_history.sql` | v2.29.0 | 收藏和历史 | 2024-05-01 |
| `012_add_template_required_vars.sql` | v2.29.0 | 模板必需变量 | 2024-05-10 |
| `013_add_preflight_checks.sql` | v2.30.0 | 前置检查 | 2024-06-01 |
| `014_add_task_timeout.sql` | v2.30.0 | 任务超时 | 2024-06-10 |
| `015_add_task_priority.sql` | v2.30.0 | 任务优先级 | 2024-06-15 |
| `016_add_task_tags.sql` | v2.31.0 | 任务标签 | 2024-07-01 |
| `017_add_execution_timeline.sql` | v2.31.0 | 执行时间线 | 2024-07-10 |
| `018_fix_favorites_foreign_keys.sql` | v2.32.0 | 修复收藏外键 | 2024-08-01 |
| `019_add_workflow_dag.sql` | v2.32.0 | 工作流 DAG | 2024-08-10 |
| `021_fix_all_foreign_keys.sql` | v2.33.0 | 全面修复外键 | 2024-09-01 |
| `022_add_template_unique_indexes_with_soft_delete.sql` | v2.34.0 | 模板唯一索引 | 2024-09-15 |
| `023_add_node_tracking_to_progress.sql` | v2.34.1 | 进度节点跟踪 | 2024-10-01 |
| `000_transition_to_code_based.sql` | v2.34.2 | 过渡脚本 | 2024-11-20 |

## 过渡说明

### 自动过渡

系统会在首次启动时自动检测并执行过渡：

1. 检测 `system_metadata` 表是否存在
2. 如果不存在且 `schema_migrations` 存在，执行过渡
3. 从 `schema_migrations` 读取最后的 SQL 迁移版本
4. 保存到 `system_metadata` 表
5. 标记迁移系统为 `code_based`

### 手动过渡（如需要）

如果自动过渡失败，可以手动执行：

```bash
# PostgreSQL
psql -U username -d database_name -f 000_transition_to_code_based.sql

# SQLite
sqlite3 database.db < 000_transition_to_code_based.sql
```

## 新系统使用指南

### 添加新的迁移

在 `pkg/database/code_migrations.go` 中注册：

```go
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

### 查看迁移状态

```bash
# 通过 HTTP 接口
curl http://localhost:8080/api/health/migration

# 或查询数据库
SELECT * FROM system_metadata WHERE key = 'schema_version';
SELECT * FROM code_migration_records ORDER BY applied_at DESC;
```

## 相关文档

- [数据库迁移系统](../../docs/database-migration-system.md)
- [自动迁移文档](../../docs/auto-migration.md)
- [代码迁移 API](../pkg/database/README.md)

## 常见问题

### Q: 旧的 SQL 迁移文件会被删除吗？

A: 不会。这些文件会被保留作为历史记录和参考。

### Q: 新系统如何处理已有的数据库？

A: 系统会自动检测并过渡。首次启动时，会从 `schema_migrations` 表读取最后的迁移版本，并保存到 `system_metadata` 表中。

### Q: 如果我想回滚到旧系统怎么办？

A: 不建议回滚。如果必须回滚，请：
1. 备份数据库
2. 删除 `system_metadata` 表
3. 删除 `code_migration_records` 表
4. 使用旧版本的应用

### Q: 旧的迁移脚本还会执行吗？

A: 不会。新系统只执行代码迁移。所有表结构由 GORM 模型定义。

## 联系我们

如有问题，请：
- 提交 Issue
- 查看文档
- 联系开发团队

---

**最后更新**: 2024-11-20  
**维护者**: Kube Node Manager Team

