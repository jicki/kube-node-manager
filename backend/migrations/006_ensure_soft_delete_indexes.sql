-- 确保软删除兼容的唯一索引
-- 修复 "could not create unique index" 错误
-- 这个迁移必须在 GORM AutoMigrate 之后执行

-- ==========================================
-- 1. 删除所有旧的全局唯一索引
-- ==========================================

-- 删除可能由 GORM 创建的旧索引（不支持软删除）
DROP INDEX IF EXISTS idx_ansible_templates_name CASCADE;
DROP INDEX IF EXISTS idx_ansible_inventories_name CASCADE;
DROP INDEX IF EXISTS idx_ansible_ssh_keys_name CASCADE;

-- ==========================================
-- 2. 创建支持软删除的部分唯一索引
-- ==========================================

-- ansible_templates: 只对未删除的记录保证名称唯一
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_templates_name_active 
ON ansible_templates(name) 
WHERE deleted_at IS NULL;

-- ansible_inventories: 只对未删除的记录保证名称唯一
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_inventories_name_active 
ON ansible_inventories(name) 
WHERE deleted_at IS NULL;

-- ansible_ssh_keys: 只对未删除的记录保证名称唯一
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_ssh_keys_name_active 
ON ansible_ssh_keys(name) 
WHERE deleted_at IS NULL;

-- ==========================================
-- 说明
-- ==========================================

-- 部分唯一索引的优势：
-- 1. 支持软删除：已删除的记录不参与唯一性检查
-- 2. 允许重用名称：删除旧记录后可以创建同名新记录
-- 3. 性能优化：索引只包含活动记录，查询更快
--
-- 例如：
-- - 创建模板 "test" -> 成功
-- - 删除模板 "test" (软删除) -> 成功
-- - 再次创建模板 "test" -> 成功（因为旧记录的 deleted_at 不为 NULL）

