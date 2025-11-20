-- 023_add_node_tracking_to_progress.sql
-- 为进度追踪添加成功/失败节点列表字段

-- 为 ProgressTask 表添加节点追踪字段
ALTER TABLE progress_tasks ADD COLUMN IF NOT EXISTS success_nodes TEXT DEFAULT '';
ALTER TABLE progress_tasks ADD COLUMN IF NOT EXISTS failed_nodes TEXT DEFAULT '';

-- 为 ProgressMessage 表添加节点追踪字段
ALTER TABLE progress_messages ADD COLUMN IF NOT EXISTS current_node VARCHAR(255) DEFAULT '';
ALTER TABLE progress_messages ADD COLUMN IF NOT EXISTS success_nodes TEXT DEFAULT '';
ALTER TABLE progress_messages ADD COLUMN IF NOT EXISTS failed_nodes TEXT DEFAULT '';

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_progress_tasks_user_status ON progress_tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_progress_messages_user_processed ON progress_messages(user_id, processed);

-- 注释
COMMENT ON COLUMN progress_tasks.success_nodes IS '成功处理的节点列表（JSON格式）';
COMMENT ON COLUMN progress_tasks.failed_nodes IS '失败的节点列表（JSON格式，包含错误信息）';
COMMENT ON COLUMN progress_messages.current_node IS '当前正在处理的节点名称';
COMMENT ON COLUMN progress_messages.success_nodes IS '成功处理的节点列表（JSON格式）';
COMMENT ON COLUMN progress_messages.failed_nodes IS '失败的节点列表（JSON格式，包含错误信息）';

