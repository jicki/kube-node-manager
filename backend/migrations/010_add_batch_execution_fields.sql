-- 为 ansible_tasks 表添加分批执行相关字段
-- Migration: 010_add_batch_execution_fields

-- 添加批量执行配置字段
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS batch_config JSONB;
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS current_batch INTEGER DEFAULT 0;
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS total_batches INTEGER DEFAULT 0;
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS batch_status VARCHAR(50);

-- 添加注释
COMMENT ON COLUMN ansible_tasks.batch_config IS '分批执行配置';
COMMENT ON COLUMN ansible_tasks.current_batch IS '当前执行批次';
COMMENT ON COLUMN ansible_tasks.total_batches IS '总批次数';
COMMENT ON COLUMN ansible_tasks.batch_status IS '批次状态(running/paused/completed)';

-- 为现有记录设置默认值
UPDATE ansible_tasks SET current_batch = 0 WHERE current_batch IS NULL;
UPDATE ansible_tasks SET total_batches = 0 WHERE total_batches IS NULL;

