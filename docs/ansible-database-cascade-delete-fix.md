# Ansible 数据库级联删除修复

## 问题描述

在删除 Ansible 相关资源时遇到以下问题：

### 1. 外键约束错误

```
ERROR: update or delete on table "ansible_tasks" violates foreign key constraint "fk_ansible_logs_task" on table "ansible_logs"
详细：Key (id)=(11) is still referenced from table "ansible_logs".

ERROR: update or delete on table "ansible_templates" violates foreign key constraint "fk_ansible_tasks_template" on table "ansible_tasks"
详细：Key (id)=(6) is still referenced from table "ansible_tasks".

ERROR: update or delete on table "ansible_inventories" violates foreign key constraint "fk_ansible_tasks_inventory" on table "ansible_tasks"
详细：Key (id)=(10) is still referenced from table "ansible_tasks".
```

### 2. 唯一索引冲突

```
ERROR: duplicate key value violates unique constraint "idx_ansible_inventories_name"
```

### 根本原因分析

1. **软删除与外键约束不兼容**
   - GORM 的软删除实际是执行 UPDATE 操作（设置 `deleted_at` 字段）
   - 数据库的外键约束（CASCADE/SET NULL）只对物理 DELETE 有效
   - 软删除不会触发外键约束的级联操作

2. **唯一索引与软删除冲突**
   - 使用 `uniqueIndex` 时，即使记录被软删除，索引仍然存在
   - 导致无法创建同名的新记录

3. **缺少级联删除逻辑**
   - 删除任务时，关联的日志没有被删除
   - 删除模板/清单时，关联的任务外键没有更新

## 解决方案

### 1. 数据库迁移脚本

创建 `004_fix_ansible_foreign_keys.sql` 迁移脚本：

#### 1.1 更新外键约束

```sql
-- ansible_logs -> ansible_tasks: CASCADE
ALTER TABLE ansible_logs
ADD CONSTRAINT fk_ansible_logs_task
FOREIGN KEY (task_id) 
REFERENCES ansible_tasks(id) 
ON DELETE CASCADE;

-- ansible_tasks -> ansible_templates: SET NULL
ALTER TABLE ansible_tasks
ADD CONSTRAINT fk_ansible_tasks_template
FOREIGN KEY (template_id) 
REFERENCES ansible_templates(id) 
ON DELETE SET NULL;

-- ansible_tasks -> ansible_inventories: SET NULL
ALTER TABLE ansible_tasks
ADD CONSTRAINT fk_ansible_tasks_inventory
FOREIGN KEY (inventory_id) 
REFERENCES ansible_inventories(id) 
ON DELETE SET NULL;
```

#### 1.2 修复唯一索引（支持软删除）

```sql
-- 删除旧的唯一索引
DROP INDEX IF EXISTS idx_ansible_inventories_name;
DROP INDEX IF EXISTS idx_ansible_templates_name;
DROP INDEX IF EXISTS idx_ansible_ssh_keys_name;

-- 创建部分索引（只对未删除的记录生效）
CREATE UNIQUE INDEX idx_ansible_inventories_name_active 
ON ansible_inventories(name) 
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_ansible_templates_name_active 
ON ansible_templates(name) 
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_ansible_ssh_keys_name_active 
ON ansible_ssh_keys(name) 
WHERE deleted_at IS NULL;
```

### 2. 代码修改

#### 2.1 模型层 (model/ansible.go)

添加级联约束注解（虽然对软删除无效，但保持数据库一致性）：

```go
// AnsibleLog
Task *AnsibleTask `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`

// AnsibleTask
Template  *AnsibleTemplate  `json:"template,omitempty" gorm:"foreignKey:TemplateID;constraint:OnDelete:SET NULL"`
Inventory *AnsibleInventory `json:"inventory,omitempty" gorm:"foreignKey:InventoryID;constraint:OnDelete:SET NULL"`
```

#### 2.2 删除逻辑修改

**删除模板 (template.go)**

```go
func (s *TemplateService) DeleteTemplate(id uint, userID uint) error {
    // ...省略查询代码...
    
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 1. 将使用此模板的任务的 template_id 设置为 NULL
        if err := tx.Model(&model.AnsibleTask{}).
            Where("template_id = ?", id).
            Update("template_id", nil).Error; err != nil {
            return fmt.Errorf("failed to update related tasks: %w", err)
        }
        
        // 2. 执行软删除
        if err := tx.Delete(&template).Error; err != nil {
            return fmt.Errorf("failed to delete template: %w", err)
        }
        
        return nil
    })
}
```

**删除清单 (inventory.go)**

```go
func (s *InventoryService) DeleteInventory(id uint, userID uint) error {
    // ...省略查询代码...
    
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 1. 将使用此清单的任务的 inventory_id 设置为 NULL
        if err := tx.Model(&model.AnsibleTask{}).
            Where("inventory_id = ?", id).
            Update("inventory_id", nil).Error; err != nil {
            return fmt.Errorf("failed to update related tasks: %w", err)
        }
        
        // 2. 执行软删除
        if err := tx.Delete(&inventory).Error; err != nil {
            return fmt.Errorf("failed to delete inventory: %w", err)
        }
        
        return nil
    })
}
```

**删除任务 (service.go)**

```go
func (s *Service) DeleteTask(taskID uint, userID uint, username string) error {
    // ...省略查询和状态检查...
    
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 1. 删除所有关联的日志
        if err := tx.Where("task_id = ?", taskID).
            Delete(&model.AnsibleLog{}).Error; err != nil {
            return fmt.Errorf("failed to delete task logs: %w", err)
        }
        
        // 2. 删除任务
        if err := tx.Delete(task).Error; err != nil {
            return fmt.Errorf("failed to delete task: %w", err)
        }
        
        return nil
    })
}
```

## 应用修复

### 方法 1: 自动迁移（开发环境）

如果后端支持自动迁移，重启应用即可：

```bash
cd backend
go run cmd/main.go
```

### 方法 2: 手动执行迁移脚本（生产环境推荐）

```bash
# PostgreSQL
psql -U postgres -d kube_node_manager -f backend/migrations/004_fix_ansible_foreign_keys.sql

# 或使用迁移工具
cd backend
go run tools/migrate.go -dir migrations -dsn "postgresql://user:pass@localhost/kube_node_manager"
```

### 方法 3: 容器环境

**Docker Compose:**

```bash
docker-compose exec postgres psql -U postgres -d kube_node_manager \
  -f /migrations/004_fix_ansible_foreign_keys.sql
```

**Kubernetes:**

```bash
# 将迁移脚本挂载到 Job 中执行
kubectl create -f deploy/k8s/migration-job.yaml
```

## 验证修复

### 1. 检查外键约束

```sql
-- 查看 ansible_logs 表的外键
SELECT 
    tc.constraint_name, 
    tc.table_name, 
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    rc.delete_rule
FROM information_schema.table_constraints AS tc 
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
JOIN information_schema.referential_constraints AS rc
    ON rc.constraint_name = tc.constraint_name
WHERE tc.table_name = 'ansible_logs'
    AND tc.constraint_type = 'FOREIGN KEY';
```

预期结果：
```
constraint_name         | table_name    | column_name | foreign_table_name | delete_rule
------------------------|---------------|-------------|--------------------|------------
fk_ansible_logs_task    | ansible_logs  | task_id     | ansible_tasks      | CASCADE
```

### 2. 检查唯一索引

```sql
-- 查看部分唯一索引
SELECT 
    indexname, 
    indexdef 
FROM pg_indexes 
WHERE tablename IN ('ansible_inventories', 'ansible_templates', 'ansible_ssh_keys')
    AND indexname LIKE '%_name_active';
```

预期结果：
```
indexname                                  | indexdef
-------------------------------------------|--------------------------------------------------
idx_ansible_inventories_name_active        | CREATE UNIQUE INDEX ... WHERE deleted_at IS NULL
idx_ansible_templates_name_active          | CREATE UNIQUE INDEX ... WHERE deleted_at IS NULL
idx_ansible_ssh_keys_name_active           | CREATE UNIQUE INDEX ... WHERE deleted_at IS NULL
```

### 3. 功能测试

#### 测试 1: 删除模板
```bash
# 删除已有任务引用的模板
curl -X DELETE http://localhost:8080/api/ansible/templates/1

# 验证：任务的 template_id 被设置为 NULL
SELECT id, name, template_id FROM ansible_tasks WHERE id IN (关联的任务ID);
```

#### 测试 2: 删除清单
```bash
# 删除已有任务引用的清单
curl -X DELETE http://localhost:8080/api/ansible/inventories/1

# 验证：任务的 inventory_id 被设置为 NULL
SELECT id, name, inventory_id FROM ansible_inventories WHERE id IN (关联的任务ID);
```

#### 测试 3: 删除任务
```bash
# 删除有日志的任务
curl -X DELETE http://localhost:8080/api/ansible/tasks/1

# 验证：所有关联日志被删除
SELECT COUNT(*) FROM ansible_logs WHERE task_id = 1;
-- 预期结果: 0
```

#### 测试 4: 创建同名资源
```bash
# 1. 创建一个清单
curl -X POST http://localhost:8080/api/ansible/inventories \
  -d '{"name":"test-inventory","source_type":"manual","content":"..."}'

# 2. 删除该清单
curl -X DELETE http://localhost:8080/api/ansible/inventories/1

# 3. 再次创建同名清单（应该成功）
curl -X POST http://localhost:8080/api/ansible/inventories \
  -d '{"name":"test-inventory","source_type":"manual","content":"..."}'
```

## 数据关系图

```
┌─────────────────────┐
│  ansible_templates  │
│  - id (PK)          │
│  - name (UNIQUE)    │  ON DELETE SET NULL
│  - deleted_at       │─────────────────┐
└─────────────────────┘                 │
                                        ▼
┌─────────────────────┐          ┌──────────────────┐
│ ansible_inventories │          │  ansible_tasks   │
│  - id (PK)          │          │  - id (PK)       │
│  - name (UNIQUE)    │  SET NULL│  - template_id   │
│  - ssh_key_id (FK)  │─────────▶│  - inventory_id  │
│  - deleted_at       │          │  - deleted_at    │
└─────────────────────┘          └──────────────────┘
                                        │
                                        │ ON DELETE CASCADE
                                        │
                                        ▼
                                 ┌──────────────────┐
                                 │  ansible_logs    │
                                 │  - id (PK)       │
                                 │  - task_id (FK)  │
                                 └──────────────────┘

┌─────────────────────┐
│ ansible_ssh_keys    │
│  - id (PK)          │  ON DELETE RESTRICT
│  - name (UNIQUE)    │  (阻止删除)
│  - deleted_at       │
└─────────────────────┘
```

## 级联规则说明

| 关系                               | 删除规则    | 说明                                        |
|-----------------------------------|------------|---------------------------------------------|
| ansible_tasks → ansible_logs       | CASCADE    | 删除任务时，自动删除所有日志                    |
| ansible_templates → ansible_tasks  | SET NULL   | 删除模板时，将任务的 template_id 设置为 NULL    |
| ansible_inventories → ansible_tasks| SET NULL   | 删除清单时，将任务的 inventory_id 设置为 NULL  |
| ansible_ssh_keys → ansible_inventories | RESTRICT | 有清单使用时，阻止删除 SSH 密钥              |

## 注意事项

1. **软删除特性保留**
   - 所有资源仍然使用软删除（保留 `deleted_at` 字段）
   - 已删除的记录可以通过 `Unscoped()` 查询恢复

2. **唯一索引适配**
   - 使用部分索引 `WHERE deleted_at IS NULL`
   - 允许同名记录在软删除后重新创建

3. **事务保证**
   - 所有删除操作使用数据库事务
   - 保证级联操作的原子性

4. **性能优化**
   - 添加了复合索引 `(task_id, created_at)` 优化日志查询
   - 为 `deleted_at` 字段添加索引

## 回滚方案

如果需要回滚修改：

```sql
-- 恢复原来的外键约束（不使用 CASCADE/SET NULL）
ALTER TABLE ansible_logs
DROP CONSTRAINT IF EXISTS fk_ansible_logs_task,
ADD CONSTRAINT fk_ansible_logs_task
FOREIGN KEY (task_id) REFERENCES ansible_tasks(id);

-- 恢复普通唯一索引
DROP INDEX IF EXISTS idx_ansible_inventories_name_active;
CREATE UNIQUE INDEX idx_ansible_inventories_name 
ON ansible_inventories(name);
```

## 参考文档

- [GORM 软删除](https://gorm.io/docs/delete.html#Soft-Delete)
- [PostgreSQL 外键约束](https://www.postgresql.org/docs/current/ddl-constraints.html#DDL-CONSTRAINTS-FK)
- [PostgreSQL 部分索引](https://www.postgresql.org/docs/current/indexes-partial.html)

## 版本历史

- **2025-10-31**: 初始版本 - 修复级联删除和唯一索引冲突问题

