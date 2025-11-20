-- 023_add_node_tracking_to_progress_sqlite.sql
-- 为进度追踪添加成功/失败节点列表字段 (SQLite版本)

-- SQLite 不支持 IF NOT EXISTS 在 ALTER TABLE ADD COLUMN 中，需要检查后添加
-- 为 progress_tasks 表添加节点追踪字段
ALTER TABLE progress_tasks ADD COLUMN success_nodes TEXT DEFAULT '';
ALTER TABLE progress_tasks ADD COLUMN failed_nodes TEXT DEFAULT '';

-- 为 progress_messages 表添加节点追踪字段
ALTER TABLE progress_messages ADD COLUMN current_node VARCHAR(255) DEFAULT '';
ALTER TABLE progress_messages ADD COLUMN success_nodes TEXT DEFAULT '';
ALTER TABLE progress_messages ADD COLUMN failed_nodes TEXT DEFAULT '';

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_progress_tasks_user_status ON progress_tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_progress_messages_user_processed ON progress_messages(user_id, processed);

