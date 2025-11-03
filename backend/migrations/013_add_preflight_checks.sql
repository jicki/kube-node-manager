-- 添加前置检查字段
-- Migration: 013_add_preflight_checks

-- +migrate Up
ALTER TABLE ansible_tasks ADD COLUMN preflight_checks JSONB;

-- 添加注释
COMMENT ON COLUMN ansible_tasks.preflight_checks IS '前置检查结果';

-- +migrate Down
ALTER TABLE ansible_tasks DROP COLUMN preflight_checks;

