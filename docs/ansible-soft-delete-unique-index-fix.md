# Ansible 软删除唯一索引修复

## 修改日期
2025-10-31

## 问题描述

在数据库迁移时遇到以下错误：

```
Failed to run migrations:ERROR: could not create unique index "idx_ansible_templates_name" (SQLSTATE 23505)
```

### 根本原因

1. **GORM 模型定义问题**：
   - 模型中使用了 `uniqueIndex` 标签
   - GORM 的 AutoMigrate 会自动创建全局唯一索引
   - 全局唯一索引不考虑软删除（`deleted_at`）字段

2. **软删除冲突**：
   - 当记录被软删除后（`deleted_at` 不为 NULL），仍然占用唯一约束
   - 无法创建同名的新记录
   - 如果数据库中存在已删除的重复记录，创建索引会失败

3. **迁移顺序问题**：
   - GORM AutoMigrate 先于迁移文件执行
   - AutoMigrate 尝试创建错误的全局唯一索引
   - 后续迁移文件无法修复已经失败的索引创建

### 影响的模型

- `AnsibleTemplate` - 任务模板
- `AnsibleInventory` - 主机清单  
- `AnsibleSSHKey` - SSH 密钥

---

## 解决方案

### 1. 修改模型定义

**位置**：`backend/internal/model/ansible.go`

**修改**：移除所有 `uniqueIndex` 标签，让迁移文件负责创建索引

#### AnsibleTemplate

```go
// 修改前
Name string `json:"name" gorm:"not null;size:255;uniqueIndex;comment:模板名称"`

// 修改后
Name string `json:"name" gorm:"not null;size:255;comment:模板名称"` // 唯一索引由迁移文件创建
```

#### AnsibleInventory

```go
// 修改前
Name string `json:"name" gorm:"not null;size:255;uniqueIndex;comment:清单名称"`

// 修改后
Name string `json:"name" gorm:"not null;size:255;comment:清单名称"` // 唯一索引由迁移文件创建
```

#### AnsibleSSHKey

```go
// 修改前
Name string `json:"name" gorm:"uniqueIndex;not null"`

// 修改后
Name string `json:"name" gorm:"not null"` // 唯一索引由迁移文件创建
```

### 2. 创建新的迁移文件

**文件**：`backend/migrations/006_ensure_soft_delete_indexes.sql`

**内容**：

```sql
-- 1. 删除所有旧的全局唯一索引
DROP INDEX IF EXISTS idx_ansible_templates_name CASCADE;
DROP INDEX IF EXISTS idx_ansible_inventories_name CASCADE;
DROP INDEX IF EXISTS idx_ansible_ssh_keys_name CASCADE;

-- 2. 创建支持软删除的部分唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_templates_name_active 
ON ansible_templates(name) 
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_inventories_name_active 
ON ansible_inventories(name) 
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_ssh_keys_name_active 
ON ansible_ssh_keys(name) 
WHERE deleted_at IS NULL;
```

---

## 技术细节

### 部分唯一索引（Partial Unique Index）

部分唯一索引只对满足特定条件的行强制唯一性：

```sql
CREATE UNIQUE INDEX idx_name_active 
ON table_name(column_name) 
WHERE deleted_at IS NULL;
```

**优势**：

1. **支持软删除**：只有未删除的记录（`deleted_at IS NULL`）参与唯一性检查
2. **允许重用名称**：删除旧记录后可以创建同名新记录
3. **性能优化**：索引更小，查询更快
4. **符合业务逻辑**：用户只关心当前活动的记录是否唯一

### 示例场景

```sql
-- 1. 创建模板 "test"
INSERT INTO ansible_templates (name, ...) VALUES ('test', ...);
-- 成功，name='test' 且 deleted_at IS NULL

-- 2. 软删除模板 "test"
UPDATE ansible_templates SET deleted_at = NOW() WHERE name = 'test';
-- 成功，deleted_at 不再为 NULL

-- 3. 再次创建模板 "test"
INSERT INTO ansible_templates (name, ...) VALUES ('test', ...);
-- 成功！因为旧记录的 deleted_at 不为 NULL，不参与唯一性检查

-- 4. 尝试创建重复的活动记录
INSERT INTO ansible_templates (name, ...) VALUES ('test', ...);
-- 失败！违反唯一约束，因为已经存在 name='test' 且 deleted_at IS NULL 的记录
```

---

## 数据库状态检查

### 检查当前索引

```sql
SELECT 
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes 
WHERE tablename IN ('ansible_inventories', 'ansible_templates', 'ansible_ssh_keys')
    AND indexname LIKE '%_name%'
ORDER BY tablename, indexname;
```

### 正确的索引状态

应该看到以下索引（支持软删除）：

```
idx_ansible_inventories_name_active  
idx_ansible_templates_name_active    
idx_ansible_ssh_keys_name_active     
```

**不应该**看到以下索引（不支持软删除）：

```
idx_ansible_inventories_name  ❌
idx_ansible_templates_name    ❌
idx_ansible_ssh_keys_name     ❌
```

### 检查重复数据

如果迁移失败，可能是因为存在重复数据。检查方法：

```sql
-- 检查 ansible_templates 中的重复名称
SELECT name, COUNT(*), 
       COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_count
FROM ansible_templates
GROUP BY name
HAVING COUNT(*) FILTER (WHERE deleted_at IS NULL) > 1;

-- 检查 ansible_inventories 中的重复名称
SELECT name, COUNT(*), 
       COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_count
FROM ansible_inventories
GROUP BY name
HAVING COUNT(*) FILTER (WHERE deleted_at IS NULL) > 1;

-- 检查 ansible_ssh_keys 中的重复名称
SELECT name, COUNT(*), 
       COUNT(*) FILTER (WHERE deleted_at IS NULL) as active_count
FROM ansible_ssh_keys
GROUP BY name
HAVING COUNT(*) FILTER (WHERE deleted_at IS NULL) > 1;
```

### 清理重复数据（如果需要）

```sql
-- 仅保留每个名称的最新活动记录
WITH duplicates AS (
    SELECT id, name,
           ROW_NUMBER() OVER (PARTITION BY name ORDER BY created_at DESC) as rn
    FROM ansible_templates
    WHERE deleted_at IS NULL
)
UPDATE ansible_templates 
SET deleted_at = NOW()
WHERE id IN (SELECT id FROM duplicates WHERE rn > 1);
```

---

## 迁移顺序

系统的迁移执行流程：

1. **GORM AutoMigrate**：创建表结构（不创建唯一索引，因为已从模型中移除）
2. **迁移文件 001-005**：执行其他迁移
3. **迁移文件 006**：删除旧索引，创建新的部分唯一索引

---

## 测试步骤

### 1. 清理并重新部署

```bash
# 如果是开发环境，可以删除并重建数据库
# 生产环境请谨慎操作

# 重新构建并启动
make docker-build
make deploy-dev
```

### 2. 验证索引创建

```sql
-- 连接到数据库
psql -h <host> -U <user> -d <database>

-- 检查索引
\di *ansible*name*

-- 应该看到：
-- idx_ansible_templates_name_active
-- idx_ansible_inventories_name_active  
-- idx_ansible_ssh_keys_name_active
```

### 3. 测试软删除场景

```bash
# 测试创建、删除、重新创建
curl -X POST /api/ansible/templates \
  -d '{"name":"test","playbook_content":"..."}'
# -> 成功

curl -X DELETE /api/ansible/templates/1
# -> 成功（软删除）

curl -X POST /api/ansible/templates \
  -d '{"name":"test","playbook_content":"..."}'
# -> 成功（允许重用名称）

curl -X POST /api/ansible/templates \
  -d '{"name":"test","playbook_content":"..."}'
# -> 失败（名称已存在）
```

---

## 向后兼容性

### 对现有数据的影响

- ✅ **无数据丢失**：只修改索引，不修改数据
- ✅ **兼容旧记录**：包括软删除的记录
- ✅ **API 行为不变**：唯一性约束逻辑不变
- ✅ **功能增强**：现在可以重用已删除记录的名称

### 对应用程序的影响

- ✅ **代码无需修改**：业务逻辑不变
- ✅ **API 响应不变**：错误码和消息一致
- ✅ **性能提升**：索引更小，查询更快

---

## 常见问题

### Q1: 为什么不在模型中定义索引？

**A**: GORM 的 `uniqueIndex` 标签不支持 `WHERE` 条件（部分索引），无法正确处理软删除。迁移文件提供了更精确的控制。

### Q2: 如果已经有重复数据怎么办？

**A**: 使用上面的 SQL 查询检查并清理重复数据。确保每个名称只有一条活动记录（`deleted_at IS NULL`）。

### Q3: 这个修复会影响性能吗？

**A**: 实际上会**提升**性能，因为部分索引更小（只包含活动记录），查询和维护都更快。

### Q4: SQLite 支持部分索引吗？

**A**: 支持！SQLite 3.8.0+ 支持部分索引，语法相同：

```sql
CREATE UNIQUE INDEX idx_name_active 
ON table_name(name) 
WHERE deleted_at IS NULL;
```

### Q5: 如果我想恢复到旧的行为怎么办？

**A**: 不建议恢复，因为会导致软删除冲突。如果必须恢复：

```go
// 在模型中添加回 uniqueIndex
Name string `json:"name" gorm:"not null;uniqueIndex"`
```

然后删除部分索引，让 GORM 创建全局索引。但这会禁用名称重用功能。

---

## 相关文件

### 修改的文件
- `backend/internal/model/ansible.go` - 移除 uniqueIndex 标签
- `backend/migrations/006_ensure_soft_delete_indexes.sql` - 新建迁移文件

### 相关迁移文件
- `backend/migrations/004_fix_ansible_foreign_keys.sql` - 首次尝试修复
- `backend/migrations/004_fix_ansible_foreign_keys_quick.sql` - 快速修复版本
- `backend/migrations/005_cleanup_old_unique_indexes.sql` - 清理旧索引

---

## 最佳实践

### 1. 软删除 + 唯一索引

对于使用软删除的模型，**始终**使用部分唯一索引：

```sql
-- ✅ 正确
CREATE UNIQUE INDEX idx_name_active 
ON table_name(name) 
WHERE deleted_at IS NULL;

-- ❌ 错误
CREATE UNIQUE INDEX idx_name 
ON table_name(name);
```

### 2. 迁移文件管理索引

对于复杂索引（部分索引、复合索引等），使用迁移文件而不是模型标签：

```go
// ✅ 推荐：在模型中添加注释
Name string `json:"name" gorm:"not null"` // 唯一索引由迁移文件创建

// ❌ 避免：在模型中定义复杂索引
Name string `json:"name" gorm:"uniqueIndex;not null"`
```

### 3. 测试软删除场景

在集成测试中包含软删除场景：

```go
func TestSoftDeleteUniqueness(t *testing.T) {
    // 创建记录
    template := createTemplate("test")
    
    // 软删除
    deleteTemplate(template.ID)
    
    // 应该允许创建同名记录
    template2 := createTemplate("test")
    assert.NotEqual(t, template.ID, template2.ID)
    
    // 不应该允许创建重复的活动记录
    _, err := createTemplate("test")
    assert.Error(t, err)
}
```

---

## 总结

这次修复通过两个关键改变解决了软删除与唯一索引的冲突：

1. **移除模型中的 uniqueIndex 标签**
   - 防止 GORM 创建不支持软删除的全局唯一索引
   - 保持模型定义简洁

2. **使用迁移文件创建部分唯一索引**
   - 只对未删除的记录强制唯一性
   - 允许重用已删除记录的名称
   - 提供更好的性能和灵活性

这是一个常见的数据库设计模式，适用于所有使用软删除的场景。

---

## 参考资料

- [PostgreSQL Partial Indexes](https://www.postgresql.org/docs/current/indexes-partial.html)
- [SQLite Partial Indexes](https://www.sqlite.org/partialindex.html)
- [GORM Indexes](https://gorm.io/docs/indexes.html)
- [Soft Delete Best Practices](https://www.cockroachlabs.com/blog/soft-delete/)

