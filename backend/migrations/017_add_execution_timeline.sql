-- +migrate Up
-- 添加任务执行时间线字段
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS execution_timeline JSONB;

-- +migrate Down
-- 删除任务执行时间线字段
ALTER TABLE ansible_tasks DROP COLUMN execution_timeline;

