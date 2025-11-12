-- +migrate Up
-- 全面修复所有表的外键约束，防止删除操作时出现外键冲突
-- 确保新部署和现有系统都不会出现外键约束问题
-- 创建时间: 2024-11-12

-- ==========================================
-- 策略说明：
-- 1. 对于审计/日志类数据：ON DELETE SET NULL（保留记录，解除关联）
-- 2. 对于依赖主体的数据：ON DELETE CASCADE（级联删除）
-- 3. 对于可选引用的数据：ON DELETE SET NULL（保留记录）
-- 4. 对于必需引用的数据：ON DELETE RESTRICT（阻止删除，需手动处理）
-- ==========================================

-- ==========================================
-- 一、修复 clusters 表相关的外键约束
-- ==========================================

-- 1.1 audit_logs.cluster_id → clusters.id (ON DELETE SET NULL)
-- 审计日志应该保留，即使集群被删除
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_audit_logs_cluster' 
        AND table_name = 'audit_logs'
    ) THEN
        ALTER TABLE audit_logs DROP CONSTRAINT fk_audit_logs_cluster;
    END IF;
END $$;

ALTER TABLE audit_logs
ADD CONSTRAINT fk_audit_logs_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE SET NULL
ON UPDATE CASCADE;

COMMENT ON CONSTRAINT fk_audit_logs_cluster ON audit_logs IS 
    '审计日志与集群的外键约束，删除集群时设置为NULL以保留审计历史';

-- 1.2 node_anomalies.cluster_id → clusters.id (ON DELETE CASCADE)
-- 节点异常依赖于集群，集群删除后这些记录无意义
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_node_anomalies_cluster' 
        AND table_name = 'node_anomalies'
    ) THEN
        ALTER TABLE node_anomalies DROP CONSTRAINT fk_node_anomalies_cluster;
    END IF;
END $$;

ALTER TABLE node_anomalies
ADD CONSTRAINT fk_node_anomalies_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE CASCADE
ON UPDATE CASCADE;

COMMENT ON CONSTRAINT fk_node_anomalies_cluster ON node_anomalies IS 
    '节点异常与集群的外键约束，删除集群时级联删除异常记录';

-- 1.3 ansible_task_history.cluster_id → clusters.id (ON DELETE SET NULL)
-- 任务历史应该保留，即使集群被删除
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_task_history' 
        AND column_name = 'cluster_id'
    ) THEN
        IF EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_ansible_task_history_cluster' 
            AND table_name = 'ansible_task_history'
        ) THEN
            ALTER TABLE ansible_task_history DROP CONSTRAINT fk_ansible_task_history_cluster;
        END IF;
        
        ALTER TABLE ansible_task_history
        ADD CONSTRAINT fk_ansible_task_history_cluster
        FOREIGN KEY (cluster_id) 
        REFERENCES clusters(id) 
        ON DELETE SET NULL
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 1.4 ansible_schedules.cluster_id → clusters.id (ON DELETE SET NULL)
-- 调度任务保留，但解除集群关联
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_schedules_cluster' 
        AND table_name = 'ansible_schedules'
    ) THEN
        ALTER TABLE ansible_schedules DROP CONSTRAINT fk_ansible_schedules_cluster;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_schedules' 
        AND column_name = 'cluster_id'
    ) THEN
        ALTER TABLE ansible_schedules
        ADD CONSTRAINT fk_ansible_schedules_cluster
        FOREIGN KEY (cluster_id) 
        REFERENCES clusters(id) 
        ON DELETE SET NULL
        ON UPDATE CASCADE;
    END IF;
END $$;

-- ==========================================
-- 二、修复 users 表相关的外键约束
-- ==========================================

-- 2.1 audit_logs.user_id → users.id (ON DELETE RESTRICT)
-- 用户有审计日志时不允许删除
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_audit_logs_user' 
        AND table_name = 'audit_logs'
    ) THEN
        ALTER TABLE audit_logs DROP CONSTRAINT fk_audit_logs_user;
    END IF;
END $$;

ALTER TABLE audit_logs
ADD CONSTRAINT fk_audit_logs_user
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE RESTRICT
ON UPDATE CASCADE;

-- 2.2 clusters.created_by → users.id (ON DELETE RESTRICT)
-- 用户创建的集群存在时不允许删除用户
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_clusters_creator' 
        AND table_name = 'clusters'
    ) THEN
        ALTER TABLE clusters DROP CONSTRAINT fk_clusters_creator;
    END IF;
END $$;

ALTER TABLE clusters
ADD CONSTRAINT fk_clusters_creator
FOREIGN KEY (created_by) 
REFERENCES users(id) 
ON DELETE RESTRICT
ON UPDATE CASCADE;

-- 2.3 ansible_tasks.user_id → users.id (ON DELETE RESTRICT)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_tasks_user' 
        AND table_name = 'ansible_tasks'
    ) THEN
        ALTER TABLE ansible_tasks DROP CONSTRAINT fk_ansible_tasks_user;
    END IF;
END $$;

ALTER TABLE ansible_tasks
ADD CONSTRAINT fk_ansible_tasks_user
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE RESTRICT
ON UPDATE CASCADE;

-- 2.4 ansible_templates.user_id → users.id (ON DELETE RESTRICT)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_templates_user' 
        AND table_name = 'ansible_templates'
    ) THEN
        ALTER TABLE ansible_templates DROP CONSTRAINT fk_ansible_templates_user;
    END IF;
END $$;

ALTER TABLE ansible_templates
ADD CONSTRAINT fk_ansible_templates_user
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE RESTRICT
ON UPDATE CASCADE;

-- 2.5 ansible_inventories.user_id → users.id (ON DELETE RESTRICT)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_inventories_user' 
        AND table_name = 'ansible_inventories'
    ) THEN
        ALTER TABLE ansible_inventories DROP CONSTRAINT fk_ansible_inventories_user;
    END IF;
END $$;

ALTER TABLE ansible_inventories
ADD CONSTRAINT fk_ansible_inventories_user
FOREIGN KEY (user_id) 
REFERENCES users(id) 
ON DELETE RESTRICT
ON UPDATE CASCADE;

-- 2.6 ansible_ssh_keys.created_by → users.id (ON DELETE RESTRICT)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_ssh_keys_creator' 
        AND table_name = 'ansible_ssh_keys'
    ) THEN
        ALTER TABLE ansible_ssh_keys DROP CONSTRAINT fk_ansible_ssh_keys_creator;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_ssh_keys' 
        AND column_name = 'created_by'
    ) THEN
        ALTER TABLE ansible_ssh_keys
        ADD CONSTRAINT fk_ansible_ssh_keys_creator
        FOREIGN KEY (created_by) 
        REFERENCES users(id) 
        ON DELETE RESTRICT
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 2.7 ansible_schedules.user_id → users.id (ON DELETE RESTRICT)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_schedules_user' 
        AND table_name = 'ansible_schedules'
    ) THEN
        ALTER TABLE ansible_schedules DROP CONSTRAINT fk_ansible_schedules_user;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_schedules' 
        AND column_name = 'user_id'
    ) THEN
        ALTER TABLE ansible_schedules
        ADD CONSTRAINT fk_ansible_schedules_user
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE RESTRICT
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 2.8 ansible_favorites.user_id → users.id (ON DELETE CASCADE)
-- 用户删除时，其收藏也应删除
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_favorites_user' 
        AND table_name = 'ansible_favorites'
    ) THEN
        ALTER TABLE ansible_favorites DROP CONSTRAINT fk_ansible_favorites_user;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_favorites' 
        AND column_name = 'user_id'
    ) THEN
        ALTER TABLE ansible_favorites
        ADD CONSTRAINT fk_ansible_favorites_user
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 2.9 ansible_task_history.user_id → users.id (ON DELETE CASCADE)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_task_history_user' 
        AND table_name = 'ansible_task_history'
    ) THEN
        ALTER TABLE ansible_task_history DROP CONSTRAINT fk_ansible_task_history_user;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_task_history' 
        AND column_name = 'user_id'
    ) THEN
        ALTER TABLE ansible_task_history
        ADD CONSTRAINT fk_ansible_task_history_user
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 2.10 ansible_tags.user_id → users.id (ON DELETE RESTRICT)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_tags_user' 
        AND table_name = 'ansible_tags'
    ) THEN
        ALTER TABLE ansible_tags DROP CONSTRAINT fk_ansible_tags_user;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_tags' 
        AND column_name = 'user_id'
    ) THEN
        ALTER TABLE ansible_tags
        ADD CONSTRAINT fk_ansible_tags_user
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE RESTRICT
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 2.11 feishu_user_mappings.system_user_id → users.id (ON DELETE CASCADE)
-- 用户删除时，飞书映射也应删除
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_feishu_mappings_user' 
        AND table_name = 'feishu_user_mappings'
    ) THEN
        ALTER TABLE feishu_user_mappings DROP CONSTRAINT fk_feishu_mappings_user;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'feishu_user_mappings' 
        AND column_name = 'system_user_id'
    ) THEN
        ALTER TABLE feishu_user_mappings
        ADD CONSTRAINT fk_feishu_mappings_user
        FOREIGN KEY (system_user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 2.12 progress_tasks.user_id → users.id (ON DELETE CASCADE)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_progress_tasks_user' 
        AND table_name = 'progress_tasks'
    ) THEN
        ALTER TABLE progress_tasks DROP CONSTRAINT fk_progress_tasks_user;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'progress_tasks' 
        AND column_name = 'user_id'
    ) THEN
        ALTER TABLE progress_tasks
        ADD CONSTRAINT fk_progress_tasks_user
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 2.13 progress_messages.user_id → users.id (ON DELETE CASCADE)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_progress_messages_user' 
        AND table_name = 'progress_messages'
    ) THEN
        ALTER TABLE progress_messages DROP CONSTRAINT fk_progress_messages_user;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'progress_messages' 
        AND column_name = 'user_id'
    ) THEN
        ALTER TABLE progress_messages
        ADD CONSTRAINT fk_progress_messages_user
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- ==========================================
-- 三、修复 ansible_templates 表相关的外键约束
-- ==========================================

-- 3.1 ansible_schedules.template_id → ansible_templates.id (ON DELETE RESTRICT)
-- 模板被调度使用时不允许删除
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_schedules_template' 
        AND table_name = 'ansible_schedules'
    ) THEN
        ALTER TABLE ansible_schedules DROP CONSTRAINT fk_ansible_schedules_template;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_schedules' 
        AND column_name = 'template_id'
    ) THEN
        ALTER TABLE ansible_schedules
        ADD CONSTRAINT fk_ansible_schedules_template
        FOREIGN KEY (template_id) 
        REFERENCES ansible_templates(id) 
        ON DELETE RESTRICT
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 3.2 ansible_task_history.template_id → ansible_templates.id (ON DELETE SET NULL)
-- 历史记录保留，但解除模板关联
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_task_history_template' 
        AND table_name = 'ansible_task_history'
    ) THEN
        ALTER TABLE ansible_task_history DROP CONSTRAINT fk_ansible_task_history_template;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_task_history' 
        AND column_name = 'template_id'
    ) THEN
        ALTER TABLE ansible_task_history
        ADD CONSTRAINT fk_ansible_task_history_template
        FOREIGN KEY (template_id) 
        REFERENCES ansible_templates(id) 
        ON DELETE SET NULL
        ON UPDATE CASCADE;
    END IF;
END $$;

-- ==========================================
-- 四、修复 ansible_inventories 表相关的外键约束
-- ==========================================

-- 4.1 ansible_schedules.inventory_id → ansible_inventories.id (ON DELETE RESTRICT)
-- 清单被调度使用时不允许删除
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_schedules_inventory' 
        AND table_name = 'ansible_schedules'
    ) THEN
        ALTER TABLE ansible_schedules DROP CONSTRAINT fk_ansible_schedules_inventory;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_schedules' 
        AND column_name = 'inventory_id'
    ) THEN
        ALTER TABLE ansible_schedules
        ADD CONSTRAINT fk_ansible_schedules_inventory
        FOREIGN KEY (inventory_id) 
        REFERENCES ansible_inventories(id) 
        ON DELETE RESTRICT
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 4.2 ansible_task_history.inventory_id → ansible_inventories.id (ON DELETE SET NULL)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_task_history_inventory' 
        AND table_name = 'ansible_task_history'
    ) THEN
        ALTER TABLE ansible_task_history DROP CONSTRAINT fk_ansible_task_history_inventory;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_task_history' 
        AND column_name = 'inventory_id'
    ) THEN
        ALTER TABLE ansible_task_history
        ADD CONSTRAINT fk_ansible_task_history_inventory
        FOREIGN KEY (inventory_id) 
        REFERENCES ansible_inventories(id) 
        ON DELETE SET NULL
        ON UPDATE CASCADE;
    END IF;
END $$;

-- ==========================================
-- 五、修复 ansible_ssh_keys 表相关的外键约束
-- ==========================================

-- 5.1 验证 ansible_inventories.ssh_key_id 约束 (已在 004 迁移中处理)
-- 这里确认配置正确
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_inventories_ssh_key' 
        AND table_name = 'ansible_inventories'
    ) THEN
        -- 检查是否已经是 RESTRICT
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.referential_constraints
            WHERE constraint_name = 'fk_ansible_inventories_ssh_key'
            AND delete_rule = 'RESTRICT'
        ) THEN
            ALTER TABLE ansible_inventories DROP CONSTRAINT fk_ansible_inventories_ssh_key;
            ALTER TABLE ansible_inventories
            ADD CONSTRAINT fk_ansible_inventories_ssh_key
            FOREIGN KEY (ssh_key_id) 
            REFERENCES ansible_ssh_keys(id) 
            ON DELETE RESTRICT
            ON UPDATE CASCADE;
        END IF;
    END IF;
END $$;

-- ==========================================
-- 六、修复多对多关联表的外键约束
-- ==========================================

-- 6.1 ansible_task_tags.task_id → ansible_tasks.id (ON DELETE CASCADE)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_task_tags_task' 
        AND table_name = 'ansible_task_tags'
    ) THEN
        ALTER TABLE ansible_task_tags DROP CONSTRAINT fk_ansible_task_tags_task;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_task_tags' 
        AND column_name = 'task_id'
    ) THEN
        ALTER TABLE ansible_task_tags
        ADD CONSTRAINT fk_ansible_task_tags_task
        FOREIGN KEY (task_id) 
        REFERENCES ansible_tasks(id) 
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- 6.2 ansible_task_tags.tag_id → ansible_tags.id (ON DELETE CASCADE)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_task_tags_tag' 
        AND table_name = 'ansible_task_tags'
    ) THEN
        ALTER TABLE ansible_task_tags DROP CONSTRAINT fk_ansible_task_tags_tag;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_task_tags' 
        AND column_name = 'tag_id'
    ) THEN
        ALTER TABLE ansible_task_tags
        ADD CONSTRAINT fk_ansible_task_tags_tag
        FOREIGN KEY (tag_id) 
        REFERENCES ansible_tags(id) 
        ON DELETE CASCADE
        ON UPDATE CASCADE;
    END IF;
END $$;

-- ==========================================
-- 七、创建缺失的索引以提高外键查询性能
-- ==========================================

-- 为经常作为外键条件查询的字段创建索引
CREATE INDEX IF NOT EXISTS idx_audit_logs_cluster_id ON audit_logs(cluster_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_node_anomalies_cluster_id ON node_anomalies(cluster_id);
CREATE INDEX IF NOT EXISTS idx_clusters_created_by ON clusters(created_by);
CREATE INDEX IF NOT EXISTS idx_ansible_tasks_user_id ON ansible_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_ansible_templates_user_id ON ansible_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_ansible_inventories_user_id ON ansible_inventories(user_id);

-- ==========================================
-- 八、添加注释文档
-- ==========================================

COMMENT ON TABLE audit_logs IS '审计日志表 - 删除用户被限制，删除集群时cluster_id设为NULL';
COMMENT ON TABLE node_anomalies IS '节点异常表 - 删除集群时级联删除';
COMMENT ON TABLE clusters IS '集群表 - 删除创建用户被限制';
COMMENT ON TABLE ansible_tasks IS 'Ansible任务表 - 删除用户/模板/清单/集群时相应字段设为NULL';
COMMENT ON TABLE ansible_schedules IS 'Ansible调度表 - 删除模板/清单被限制，删除集群时cluster_id设为NULL';
COMMENT ON TABLE ansible_favorites IS '收藏表 - 删除用户时级联删除';
COMMENT ON TABLE ansible_task_history IS '任务历史表 - 删除用户时级联删除，删除其他实体时字段设为NULL';
COMMENT ON TABLE ansible_task_tags IS '任务标签关联表 - 删除任务或标签时级联删除';
COMMENT ON TABLE feishu_user_mappings IS '飞书用户映射表 - 删除用户时级联删除';
COMMENT ON TABLE progress_tasks IS '进度任务表 - 删除用户时级联删除';
COMMENT ON TABLE progress_messages IS '进度消息表 - 删除用户时级联删除';

-- ==========================================
-- 九、验证所有外键约束
-- ==========================================

-- 查询所有外键约束以验证配置
DO $$
DECLARE
    fk_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO fk_count
    FROM information_schema.table_constraints
    WHERE constraint_type = 'FOREIGN KEY';
    
    RAISE NOTICE '已配置 % 个外键约束', fk_count;
END $$;

-- +migrate Down
-- 回滚时恢复为默认的 RESTRICT 约束（不推荐回滚，因为会导致删除问题）
-- 如果确实需要回滚，请手动执行

-- 注意：回滚可能导致删除操作失败
-- 建议保留此迁移以确保系统稳定性

