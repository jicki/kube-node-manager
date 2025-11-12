-- 修复外键约束冲突
-- 用于删除被审计日志引用的集群

-- =====================================
-- 使用方法
-- =====================================
-- 1. 修改下面的 CLUSTER_ID (7) 为你要删除的集群 ID
-- 2. 执行: psql -h localhost -U postgres -d kube_node_manager -f fix_foreign_key_constraint.sql
-- =====================================

-- 查看要删除的集群信息
\echo '===== 集群信息 ====='
SELECT id, name, status, deleted_at, created_at 
FROM clusters 
WHERE id = 7;

-- 检查相关记录数量
\echo ''
\echo '===== 相关记录统计 ====='
SELECT 
    'audit_logs' as table_name,
    COUNT(*) as record_count
FROM audit_logs WHERE cluster_id = 7
UNION ALL
SELECT 
    'node_anomalies' as table_name,
    COUNT(*) as record_count
FROM node_anomalies WHERE cluster_id = 7
UNION ALL
SELECT 
    'ansible_inventories' as table_name,
    COUNT(*) as record_count
FROM ansible_inventories WHERE cluster_id = 7;

-- 开始删除
\echo ''
\echo '===== 开始删除操作 ====='

BEGIN;

-- 1. 更新审计日志（保留记录但解除关联）
UPDATE audit_logs SET cluster_id = NULL WHERE cluster_id = 7;
\echo '✓ 已更新审计日志关联'

-- 2. 删除节点异常记录
DELETE FROM node_anomalies WHERE cluster_id = 7;
\echo '✓ 已删除节点异常记录'

-- 3. 清除 Ansible 清单关联
UPDATE ansible_inventories SET cluster_id = NULL WHERE cluster_id = 7;
\echo '✓ 已清除 Ansible 清单关联'

-- 4. 删除集群记录
DELETE FROM clusters WHERE id = 7;
\echo '✓ 已删除集群记录'

COMMIT;

\echo ''
\echo '===== 删除完成 ====='

-- 验证删除
\echo ''
\echo '===== 验证删除结果 ====='
SELECT COUNT(*) as remaining_records FROM clusters WHERE id = 7;
\echo '(如果返回 0，说明删除成功)'

