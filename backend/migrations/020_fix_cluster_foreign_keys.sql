-- +migrate Up
-- 修复集群外键约束，防止删除集群时出现外键冲突
-- 解决: ERROR: update or delete on table "clusters" violates foreign key constraint

-- ==========================================
-- 1. 修复 audit_logs 表的外键约束
-- ==========================================

-- 删除旧的外键约束（如果存在）
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

-- 添加新的外键约束，删除集群时将 cluster_id 设置为 NULL
-- 这样可以保留审计记录，但解除集群关联
ALTER TABLE audit_logs
ADD CONSTRAINT fk_audit_logs_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE SET NULL;

-- ==========================================
-- 2. 修复 node_anomalies 表的外键约束
-- ==========================================

-- 删除旧的外键约束（如果存在）
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

-- 添加新的外键约束，删除集群时级联删除节点异常记录
-- 节点异常记录依赖于集群，集群删除后这些记录无意义
ALTER TABLE node_anomalies
ADD CONSTRAINT fk_node_anomalies_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE CASCADE;

-- ==========================================
-- 3. 验证 ansible_task_history 表的外键约束
-- ==========================================

-- ansible_task_history 表可能引用 cluster_id
-- 删除旧的外键约束（如果存在）
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_task_history_cluster' 
        AND table_name = 'ansible_task_history'
    ) THEN
        ALTER TABLE ansible_task_history DROP CONSTRAINT fk_ansible_task_history_cluster;
    END IF;
END $$;

-- 如果 cluster_id 字段存在，添加外键约束
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'ansible_task_history' 
        AND column_name = 'cluster_id'
    ) THEN
        ALTER TABLE ansible_task_history
        ADD CONSTRAINT fk_ansible_task_history_cluster
        FOREIGN KEY (cluster_id) 
        REFERENCES clusters(id) 
        ON DELETE SET NULL;
    END IF;
END $$;

-- ==========================================
-- 4. 创建索引以提高删除性能
-- ==========================================

-- audit_logs 的 cluster_id 索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_audit_logs_cluster_id ON audit_logs(cluster_id);

-- node_anomalies 的 cluster_id 索引（已存在 idx_cluster_node，这里确保单独索引）
CREATE INDEX IF NOT EXISTS idx_node_anomalies_cluster_id ON node_anomalies(cluster_id);

-- ==========================================
-- 5. 添加注释说明
-- ==========================================

COMMENT ON CONSTRAINT fk_audit_logs_cluster ON audit_logs IS 
    '审计日志与集群的外键约束，删除集群时设置为NULL以保留审计历史';

COMMENT ON CONSTRAINT fk_node_anomalies_cluster ON node_anomalies IS 
    '节点异常与集群的外键约束，删除集群时级联删除异常记录';

-- +migrate Down
-- 回滚时恢复为 RESTRICT 约束（默认行为）

-- 恢复 audit_logs 外键约束
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_cluster;
ALTER TABLE audit_logs
ADD CONSTRAINT fk_audit_logs_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE RESTRICT;

-- 恢复 node_anomalies 外键约束
ALTER TABLE node_anomalies DROP CONSTRAINT IF EXISTS fk_node_anomalies_cluster;
ALTER TABLE node_anomalies
ADD CONSTRAINT fk_node_anomalies_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE RESTRICT;

-- 恢复 ansible_task_history 外键约束
ALTER TABLE ansible_task_history DROP CONSTRAINT IF EXISTS fk_ansible_task_history_cluster;
ALTER TABLE ansible_task_history
ADD CONSTRAINT fk_ansible_task_history_cluster
FOREIGN KEY (cluster_id) 
REFERENCES clusters(id) 
ON DELETE RESTRICT;

