-- 修复 Ansible 相关表的外键约束，支持级联删除和 SET NULL
-- 解决删除时的外键约束冲突问题

-- ==========================================
-- 1. 修复 ansible_logs 表的外键约束
-- ==========================================

-- 删除旧的外键约束
ALTER TABLE IF EXISTS ansible_logs 
DROP CONSTRAINT IF EXISTS fk_ansible_logs_task;

-- 添加新的外键约束，删除任务时级联删除日志
ALTER TABLE ansible_logs
ADD CONSTRAINT fk_ansible_logs_task
FOREIGN KEY (task_id) 
REFERENCES ansible_tasks(id) 
ON DELETE CASCADE;

-- ==========================================
-- 2. 修复 ansible_tasks 表的外键约束
-- ==========================================

-- 2.1 Template 外键 - 删除模板时将任务的 template_id 设置为 NULL
ALTER TABLE IF EXISTS ansible_tasks
DROP CONSTRAINT IF EXISTS fk_ansible_tasks_template;

ALTER TABLE ansible_tasks
ADD CONSTRAINT fk_ansible_tasks_template
FOREIGN KEY (template_id) 
REFERENCES ansible_templates(id) 
ON DELETE SET NULL;

-- 2.2 Inventory 外键 - 删除清单时将任务的 inventory_id 设置为 NULL
ALTER TABLE IF EXISTS ansible_tasks
DROP CONSTRAINT IF EXISTS fk_ansible_tasks_inventory;

ALTER TABLE ansible_tasks
ADD CONSTRAINT fk_ansible_tasks_inventory
FOREIGN KEY (inventory_id) 
REFERENCES ansible_inventories(id) 
ON DELETE SET NULL;

-- 2.3 Cluster 外键 - 删除集群时将任务的 cluster_id 设置为 NULL
ALTER TABLE IF EXISTS ansible_tasks
DROP CONSTRAINT IF EXISTS fk_ansible_tasks_cluster;

ALTER TABLE ansible_tasks
ADD CONSTRAINT fk_ansible_tasks_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE SET NULL;

-- ==========================================
-- 3. 修复 ansible_inventories 表的外键约束
-- ==========================================

-- 3.1 Cluster 外键 - 删除集群时将清单的 cluster_id 设置为 NULL
ALTER TABLE IF EXISTS ansible_inventories
DROP CONSTRAINT IF EXISTS fk_ansible_inventories_cluster;

ALTER TABLE ansible_inventories
ADD CONSTRAINT fk_ansible_inventories_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE SET NULL;

-- 3.2 SSH Key 外键 - 删除 SSH 密钥时阻止删除（保护性删除）
ALTER TABLE IF EXISTS ansible_inventories
DROP CONSTRAINT IF EXISTS fk_ansible_inventories_ssh_key;

ALTER TABLE ansible_inventories
ADD CONSTRAINT fk_ansible_inventories_ssh_key
FOREIGN KEY (ssh_key_id) 
REFERENCES ansible_ssh_keys(id) 
ON DELETE RESTRICT;

-- ==========================================
-- 4. 为软删除字段添加复合唯一索引
-- ==========================================

-- 删除旧的唯一索引
DROP INDEX IF EXISTS idx_ansible_inventories_name;
DROP INDEX IF EXISTS idx_ansible_templates_name;
DROP INDEX IF EXISTS idx_ansible_ssh_keys_name;

-- 为 ansible_inventories 添加支持软删除的唯一索引
-- 只对未删除的记录生效
CREATE UNIQUE INDEX idx_ansible_inventories_name_active 
ON ansible_inventories(name) 
WHERE deleted_at IS NULL;

-- 为 ansible_templates 添加支持软删除的唯一索引
CREATE UNIQUE INDEX idx_ansible_templates_name_active 
ON ansible_templates(name) 
WHERE deleted_at IS NULL;

-- 为 ansible_ssh_keys 添加支持软删除的唯一索引
CREATE UNIQUE INDEX idx_ansible_ssh_keys_name_active 
ON ansible_ssh_keys(name) 
WHERE deleted_at IS NULL;

-- ==========================================
-- 5. 添加索引以提升查询性能
-- ==========================================

-- 为 deleted_at 添加索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_ansible_tasks_deleted_at ON ansible_tasks(deleted_at);
CREATE INDEX IF NOT EXISTS idx_ansible_templates_deleted_at ON ansible_templates(deleted_at);
CREATE INDEX IF NOT EXISTS idx_ansible_inventories_deleted_at ON ansible_inventories(deleted_at);
CREATE INDEX IF NOT EXISTS idx_ansible_ssh_keys_deleted_at ON ansible_ssh_keys(deleted_at);

-- 为状态字段添加索引
CREATE INDEX IF NOT EXISTS idx_ansible_tasks_status ON ansible_tasks(status);

-- 为日志的任务 ID 和创建时间添加复合索引
CREATE INDEX IF NOT EXISTS idx_ansible_logs_task_created 
ON ansible_logs(task_id, created_at DESC);

-- ==========================================
-- 说明
-- ==========================================

-- CASCADE: 删除父记录时，自动删除所有子记录
--   - ansible_tasks -> ansible_logs (删除任务时删除所有日志)

-- SET NULL: 删除父记录时，将子记录的外键设置为 NULL
--   - ansible_templates -> ansible_tasks (删除模板时保留任务，但清空模板引用)
--   - ansible_inventories -> ansible_tasks (删除清单时保留任务，但清空清单引用)
--   - clusters -> ansible_tasks (删除集群时保留任务，但清空集群引用)
--   - clusters -> ansible_inventories (删除集群时保留清单，但清空集群引用)

-- RESTRICT: 如果有子记录引用，则阻止删除父记录
--   - ansible_ssh_keys -> ansible_inventories (有清单使用时不能删除 SSH 密钥)

