-- +migrate Up
-- 添加任务优先级和队列相关字段
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS priority VARCHAR(20) DEFAULT 'medium';
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS queued_at TIMESTAMP;
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS wait_duration INTEGER DEFAULT 0;

-- 为优先级字段创建索引，优化队列查询
CREATE INDEX IF NOT EXISTS idx_ansible_tasks_priority ON ansible_tasks (priority);

-- 为组合查询创建复合索引（状态 + 优先级 + 入队时间）
CREATE INDEX IF NOT EXISTS idx_ansible_tasks_queue ON ansible_tasks (status, priority, queued_at) WHERE status = 'pending';

-- +migrate Down
-- 删除索引
DROP INDEX IF EXISTS idx_ansible_tasks_queue;
DROP INDEX IF EXISTS idx_ansible_tasks_priority;

-- 删除字段
ALTER TABLE ansible_tasks DROP COLUMN wait_duration;
ALTER TABLE ansible_tasks DROP COLUMN queued_at;
ALTER TABLE ansible_tasks DROP COLUMN priority;

