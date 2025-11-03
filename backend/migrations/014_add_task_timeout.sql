-- 添加任务超时控制字段
-- Migration: 014_add_task_timeout

-- +migrate Up
ALTER TABLE ansible_tasks ADD COLUMN timeout_seconds INTEGER DEFAULT 0;
ALTER TABLE ansible_tasks ADD COLUMN is_timed_out BOOLEAN DEFAULT false;

-- 添加注释
COMMENT ON COLUMN ansible_tasks.timeout_seconds IS '超时时间(秒),0表示不限制';
COMMENT ON COLUMN ansible_tasks.is_timed_out IS '是否超时';

-- 添加索引，方便查询超时任务
CREATE INDEX idx_ansible_tasks_is_timed_out ON ansible_tasks (is_timed_out) WHERE is_timed_out = true;

-- +migrate Down
DROP INDEX IF EXISTS idx_ansible_tasks_is_timed_out;
ALTER TABLE ansible_tasks DROP COLUMN is_timed_out;
ALTER TABLE ansible_tasks DROP COLUMN timeout_seconds;

