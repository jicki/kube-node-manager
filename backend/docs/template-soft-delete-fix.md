# 模板软删除唯一约束修复

## 问题描述

在使用标签模板和污点模板功能时，如果删除一个模板后再次创建同名模板，会遇到以下错误：

```
Failed to create label template: failed to create template: ERROR: duplicate key value violates unique constraint "idx_label_templates_name" (SQLSTATE 23505)
```

## 问题原因

1. **软删除机制**：模板使用 GORM 的软删除功能（`DeletedAt` 字段），删除时只是标记为已删除，而不是物理删除
2. **唯一约束冲突**：数据库中存在 `name` 字段的唯一索引，但该索引没有考虑软删除的情况
3. **PostgreSQL 特性**：在 PostgreSQL 中，唯一索引会对所有记录生效，包括已软删除的记录

## 解决方案

### 1. 服务层修复

在 `CreateTemplate` 方法中添加对已软删除记录的清理逻辑：

**标签模板**（`backend/internal/service/label/label.go`）：
```go
// 检查是否存在同名但已软删除的记录，如果存在则硬删除
if err := s.db.Unscoped().Where("name = ? AND deleted_at IS NOT NULL", req.Name).Delete(&model.LabelTemplate{}).Error; err != nil {
    s.logger.Warnf("Failed to clean up soft-deleted template with name %s: %v", req.Name, err)
    // 不返回错误，继续创建
}
```

**污点模板**（`backend/internal/service/taint/taint.go`）：
```go
// 检查是否存在同名但已软删除的记录，如果存在则硬删除
if err := s.db.Unscoped().Where("name = ? AND deleted_at IS NOT NULL", req.Name).Delete(&model.TaintTemplate{}).Error; err != nil {
    s.logger.Warnf("Failed to clean up soft-deleted template with name %s: %v", req.Name, err)
    // 不返回错误，继续创建
}
```

### 2. 数据库迁移

创建迁移文件 `022_add_template_unique_indexes_with_soft_delete.sql`，添加**部分唯一索引**（Partial Index）：

```sql
-- 标签模板名称唯一索引（只对未删除的记录生效）
DROP INDEX IF EXISTS idx_label_templates_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_label_templates_name 
ON label_templates(name) 
WHERE deleted_at IS NULL;

-- 污点模板名称唯一索引（只对未删除的记录生效）
DROP INDEX IF EXISTS idx_taint_templates_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_taint_templates_name 
ON taint_templates(name) 
WHERE deleted_at IS NULL;
```

部分索引的优势：
- 只对 `deleted_at IS NULL` 的记录强制唯一性
- 允许多个已软删除的同名记录存在
- 同时支持 PostgreSQL 和 SQLite

## 测试验证

### 测试场景 1：创建和删除模板

```bash
# 1. 创建一个标签模板
curl -X POST http://localhost:8080/api/v1/labels/templates \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-template",
    "description": "测试模板",
    "labels": {
      "env": "test"
    }
  }'

# 2. 删除该模板
curl -X DELETE http://localhost:8080/api/v1/labels/templates/1 \
  -H "Authorization: Bearer YOUR_TOKEN"

# 3. 再次创建同名模板（应该成功）
curl -X POST http://localhost:8080/api/v1/labels/templates \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-template",
    "description": "测试模板2",
    "labels": {
      "env": "prod"
    }
  }'
```

### 测试场景 2：验证唯一约束仍然有效

```bash
# 1. 创建一个模板
curl -X POST http://localhost:8080/api/v1/labels/templates \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "unique-test",
    "description": "唯一性测试",
    "labels": {
      "env": "test"
    }
  }'

# 2. 尝试创建同名模板（应该失败）
curl -X POST http://localhost:8080/api/v1/labels/templates \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "unique-test",
    "description": "重复名称",
    "labels": {
      "env": "prod"
    }
  }'
# 预期错误：template name already exists: unique-test
```

## 验证索引

### PostgreSQL

```sql
-- 查看索引定义
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename IN ('label_templates', 'taint_templates') 
  AND indexname LIKE '%_name';
```

### SQLite

```sql
-- 查看索引定义
SELECT name, sql 
FROM sqlite_master 
WHERE type='index' 
  AND (name = 'idx_label_templates_name' OR name = 'idx_taint_templates_name');
```

## 部署说明

### 1. 更新代码

```bash
git pull origin main
```

### 2. 运行迁移

迁移会在应用启动时自动执行，或者可以手动执行：

**使用迁移工具**：
```bash
cd backend
go run tools/migrate.go
```

**手动执行 SQL**（PostgreSQL）：
```bash
psql -h localhost -U your_user -d kube_node_manager -f migrations/022_add_template_unique_indexes_with_soft_delete.sql
```

**手动执行 SQL**（SQLite）：
```bash
sqlite3 backend/data/kube-node-manager.db < backend/migrations/022_add_template_unique_indexes_with_soft_delete.sql
```

### 3. 重启应用

```bash
# Docker Compose
docker-compose restart backend

# Kubernetes
kubectl rollout restart deployment/kube-node-manager

# 本地开发
# 重启 Go 应用
```

## 影响范围

- ✅ 标签模板创建、删除、重建
- ✅ 污点模板创建、删除、重建
- ✅ 唯一性约束仍然有效（对未删除的记录）
- ✅ 同时支持 PostgreSQL 和 SQLite
- ✅ 向后兼容，不影响现有功能

## 相关文件

- `backend/internal/service/label/label.go` - 标签模板服务
- `backend/internal/service/taint/taint.go` - 污点模板服务
- `backend/migrations/022_add_template_unique_indexes_with_soft_delete.sql` - 数据库迁移
- `backend/internal/model/template.go` - 模板数据模型

## 最佳实践

1. **软删除清理**：在创建新记录前，自动清理已软删除的同名记录
2. **部分索引**：使用部分索引只对活跃记录强制唯一性
3. **日志记录**：清理操作添加警告日志，便于追踪
4. **不阻塞创建**：即使清理失败，也继续创建操作（数据库约束会保证最终一致性）

## 注意事项

1. **数据备份**：执行迁移前建议备份数据库
2. **索引重建**：迁移会先删除旧索引再创建新索引，可能需要短暂的锁表时间
3. **PostgreSQL 版本**：部分索引需要 PostgreSQL 7.2+ 或 SQLite 3.8.0+
4. **性能影响**：部分索引可能略微影响插入性能，但对查询性能无影响

## 故障排查

### 问题：迁移执行失败

**错误**：`ERROR: index "idx_label_templates_name" already exists`

**解决方案**：
```sql
-- 手动删除旧索引
DROP INDEX IF EXISTS idx_label_templates_name;
DROP INDEX IF EXISTS idx_taint_templates_name;

-- 重新运行迁移
```

### 问题：仍然提示重复键错误

**可能原因**：
1. 迁移未成功执行
2. 数据库中存在未删除的同名记录

**解决方案**：
```sql
-- 检查是否有重复的未删除记录
SELECT name, deleted_at, COUNT(*) 
FROM label_templates 
WHERE deleted_at IS NULL 
GROUP BY name, deleted_at 
HAVING COUNT(*) > 1;

-- 查看索引定义
\d+ label_templates  -- PostgreSQL
.schema label_templates  -- SQLite
```

## 版本历史

- **v1.0.0** (2025-11-19): 初始版本，修复软删除唯一约束问题

