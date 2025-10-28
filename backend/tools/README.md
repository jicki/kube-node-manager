# 数据库工具

本目录包含数据库管理和维护工具。

## 迁移工具

### migrate.go

手动执行数据库迁移的工具。

**使用方法：**

```bash
cd backend
go run tools/migrate.go
```

**功能：**
- 自动检测数据库类型（SQLite/PostgreSQL）
- 创建所有必需的表
- 显示当前数据库中的表列表

**适用场景：**
1. 首次部署后初始化数据库
2. 升级版本后需要手动同步数据库结构
3. 排查数据库结构问题

**注意：**
- 应用启动时会自动执行迁移，通常不需要手动运行
- 迁移是幂等的，多次运行不会造成问题
- 确保有足够的数据库权限

**示例输出：**

```
2025/10/28 13:36:17 Starting database migration...
2025/10/28 13:36:17 Database migration completed successfully!

Tables in database:
  - anomaly_report_configs
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
  - taint_templates
  - users
```

## 其他工具

将来可能添加的工具：

- `backup.go` - 数据库备份工具
- `restore.go` - 数据库恢复工具
- `cleanup.go` - 数据清理工具
- `export.go` - 数据导出工具

