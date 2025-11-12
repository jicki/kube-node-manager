-- Performance optimization indexes
-- 用于优化查询性能的复合索引

-- ====================================================
-- 1. 异常监控相关索引优化
-- ====================================================

-- 复合索引：集群ID + 状态 + 开始时间（降序）
-- 优化场景：按集群和状态筛选异常，按时间排序
CREATE INDEX IF NOT EXISTS idx_anomaly_cluster_status_time 
ON node_anomalies(cluster_id, status, start_time DESC);

-- 复合索引：节点名称 + 集群ID + 状态
-- 优化场景：查询特定节点的异常记录
CREATE INDEX IF NOT EXISTS idx_anomaly_node_cluster_status 
ON node_anomalies(node_name, cluster_id, status);

-- 复合索引：异常类型 + 集群ID + 开始时间
-- 优化场景：按异常类型统计和查询
CREATE INDEX IF NOT EXISTS idx_anomaly_type_cluster_time 
ON node_anomalies(anomaly_type, cluster_id, start_time DESC);

-- 单列索引：持续时间
-- 优化场景：按持续时间排序和筛选（已解决的异常）
CREATE INDEX IF NOT EXISTS idx_anomaly_duration 
ON node_anomalies(duration) 
WHERE status = 'Resolved';

-- 复合索引：状态 + 更新时间
-- 优化场景：查询活跃异常和最近更新的记录
CREATE INDEX IF NOT EXISTS idx_anomaly_status_updated 
ON node_anomalies(status, updated_at DESC);

-- ====================================================
-- 2. 审计日志相关索引优化
-- ====================================================

-- 复合索引：用户ID + 创建时间（降序）
-- 优化场景：查询特定用户的操作历史
CREATE INDEX IF NOT EXISTS idx_audit_user_time 
ON audit_logs(user_id, created_at DESC);

-- 复合索引：集群ID + 操作类型 + 创建时间
-- 优化场景：按集群和操作类型筛选审计日志
CREATE INDEX IF NOT EXISTS idx_audit_cluster_action_time 
ON audit_logs(cluster_id, action, created_at DESC);

-- 复合索引：资源类型 + 节点名称 + 创建时间
-- 优化场景：查询特定资源的操作历史
CREATE INDEX IF NOT EXISTS idx_audit_resource_time 
ON audit_logs(resource_type, node_name, created_at DESC);

-- 部分索引：重要操作（仅索引更新类操作）
-- 优化场景：快速查询修改类操作
CREATE INDEX IF NOT EXISTS idx_audit_update_actions 
ON audit_logs(created_at DESC) 
WHERE action IN ('node_update', 'label_update', 'taint_update', 'node_cordon', 'node_uncordon', 'node_drain');

-- ====================================================
-- 3. 集群相关索引优化
-- ====================================================

-- 单列索引：集群名称（如果还没有）
CREATE INDEX IF NOT EXISTS idx_clusters_name 
ON clusters(name);

-- 单列索引：集群状态
CREATE INDEX IF NOT EXISTS idx_clusters_status 
ON clusters(status);

-- ====================================================
-- 4. 用户相关索引优化
-- ====================================================

-- 单列索引：用户名（如果还没有）
CREATE INDEX IF NOT EXISTS idx_users_username 
ON users(username);

-- 单列索引：邮箱
CREATE INDEX IF NOT EXISTS idx_users_email 
ON users(email);

-- ====================================================
-- 5. Feishu 相关索引优化
-- ====================================================

-- 复合索引：Feishu用户会话 - 用户ID + 更新时间
CREATE INDEX IF NOT EXISTS idx_feishu_user_sessions_user_time 
ON feishu_user_sessions(feishu_user_id, updated_at DESC);

-- ====================================================
-- 6. 异常报告配置索引优化
-- ====================================================

-- 单列索引：启用状态
CREATE INDEX IF NOT EXISTS idx_anomaly_report_enabled 
ON anomaly_report_configs(enabled);

-- 复合索引：启用状态 + 下次运行时间
CREATE INDEX IF NOT EXISTS idx_anomaly_report_next_run 
ON anomaly_report_configs(enabled, next_run_time) 
WHERE enabled = true;

-- ====================================================
-- 分析与统计信息
-- ====================================================

-- PostgreSQL: 更新表统计信息
-- ANALYZE node_anomalies;
-- ANALYZE audit_logs;
-- ANALYZE clusters;
-- ANALYZE users;

-- SQLite: VACUUM and ANALYZE
-- VACUUM;
-- ANALYZE;

-- ====================================================
-- 索引使用说明
-- ====================================================

-- 1. idx_anomaly_cluster_status_time
--    查询示例: SELECT * FROM node_anomalies WHERE cluster_id = 1 AND status = 'Active' ORDER BY start_time DESC
--    预期提升: 60-80%

-- 2. idx_audit_user_time
--    查询示例: SELECT * FROM audit_logs WHERE user_id = 123 ORDER BY created_at DESC LIMIT 100
--    预期提升: 50-70%

-- 3. idx_audit_update_actions
--    查询示例: SELECT * FROM audit_logs WHERE action = 'node_update' ORDER BY created_at DESC
--    预期提升: 40-60%

-- 4. idx_anomaly_duration
--    查询示例: SELECT AVG(duration) FROM node_anomalies WHERE status = 'Resolved'
--    预期提升: 30-50%

-- ====================================================
-- 回滚脚本
-- ====================================================

-- 如需回滚，执行以下语句删除索引：
-- DROP INDEX IF EXISTS idx_anomaly_cluster_status_time;
-- DROP INDEX IF EXISTS idx_anomaly_node_cluster_status;
-- DROP INDEX IF EXISTS idx_anomaly_type_cluster_time;
-- DROP INDEX IF EXISTS idx_anomaly_duration;
-- DROP INDEX IF EXISTS idx_anomaly_status_updated;
-- DROP INDEX IF EXISTS idx_audit_user_time;
-- DROP INDEX IF EXISTS idx_audit_cluster_action_time;
-- DROP INDEX IF EXISTS idx_audit_resource_time;
-- DROP INDEX IF EXISTS idx_audit_update_actions;
-- DROP INDEX IF EXISTS idx_clusters_name;
-- DROP INDEX IF EXISTS idx_clusters_status;
-- DROP INDEX IF EXISTS idx_users_username;
-- DROP INDEX IF EXISTS idx_users_email;
-- DROP INDEX IF EXISTS idx_feishu_user_sessions_user_time;
-- DROP INDEX IF EXISTS idx_anomaly_report_enabled;
-- DROP INDEX IF EXISTS idx_anomaly_report_next_run;

