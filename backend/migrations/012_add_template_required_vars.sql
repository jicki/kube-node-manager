-- 添加模板必需变量字段
-- Migration: 012_add_template_required_vars

-- +migrate Up
ALTER TABLE ansible_templates ADD COLUMN IF NOT EXISTS required_vars JSONB;

-- 添加注释
COMMENT ON COLUMN ansible_templates.required_vars IS '必需变量列表';

-- +migrate Down
ALTER TABLE ansible_templates DROP COLUMN required_vars;

