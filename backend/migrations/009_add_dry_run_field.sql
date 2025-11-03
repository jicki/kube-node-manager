-- 为 ansible_tasks 表添加 dry_run 字段
-- Migration: 009_add_dry_run_field

-- 添加 dry_run 字段
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS dry_run BOOLEAN DEFAULT FALSE;

-- 添加注释
COMMENT ON COLUMN ansible_tasks.dry_run IS '是否为检查模式(Dry Run)';

-- 为现有记录设置默认值
UPDATE ansible_tasks SET dry_run = FALSE WHERE dry_run IS NULL;

