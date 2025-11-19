-- 添加支持软删除的模板名称唯一索引
-- 这些索引只对未删除的记录(deleted_at IS NULL)生效
-- 允许软删除后重新创建同名模板

-- 标签模板名称唯一索引
-- 如果索引已存在则先删除
DROP INDEX IF EXISTS idx_label_templates_name;

-- 创建部分唯一索引（只对未删除的记录）
CREATE UNIQUE INDEX IF NOT EXISTS idx_label_templates_name 
ON label_templates(name) 
WHERE deleted_at IS NULL;

-- 污点模板名称唯一索引
-- 如果索引已存在则先删除
DROP INDEX IF EXISTS idx_taint_templates_name;

-- 创建部分唯一索引（只对未删除的记录）
CREATE UNIQUE INDEX IF NOT EXISTS idx_taint_templates_name 
ON taint_templates(name) 
WHERE deleted_at IS NULL;

