-- 为异常记录表添加性能优化索引
-- 这些索引可以显著提升查询性能

-- 复合索引：集群+时间+状态（用于统计查询）
CREATE INDEX IF NOT EXISTS idx_anomalies_cluster_time_status 
ON node_anomalies(cluster_id, start_time, status);

-- 复合索引：集群+异常类型+状态+时间（用于类型统计）
CREATE INDEX IF NOT EXISTS idx_anomalies_statistics 
ON node_anomalies(cluster_id, anomaly_type, status, start_time);

-- 索引：节点名+时间（用于节点历史查询）
CREATE INDEX IF NOT EXISTS idx_anomalies_node_time 
ON node_anomalies(node_name, start_time DESC);

-- 索引：状态（用于快速查找活跃异常）
CREATE INDEX IF NOT EXISTS idx_anomalies_status
ON node_anomalies(status);

-- 说明：
-- 1. 这些索引已经在 GORM 模型中通过 `gorm:"index"` 标签定义
-- 2. 此文件仅用于手动创建或确保索引存在
-- 3. PostgreSQL 会自动使用这些索引优化查询
-- 4. 使用 IF NOT EXISTS 避免重复创建

-- 查看索引使用情况（仅供参考，不会执行）:
-- SELECT schemaname, tablename, indexname, idx_scan 
-- FROM pg_stat_user_indexes 
-- WHERE tablename = 'node_anomalies' 
-- ORDER BY idx_scan DESC;

