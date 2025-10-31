-- 快速修复唯一索引问题 - 支持软删除
-- 只需要修复唯一索引部分

BEGIN;

-- ==========================================
-- 修复 ansible_inventories 唯一索引
-- ==========================================

-- 删除旧的唯一索引
DROP INDEX IF EXISTS idx_ansible_inventories_name;

-- 创建部分唯一索引（只对未删除的记录生效）
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_inventories_name_active 
ON ansible_inventories(name) 
WHERE deleted_at IS NULL;

COMMIT;

-- ==========================================
-- 修复 ansible_templates 唯一索引
-- ==========================================

BEGIN;

-- 删除旧的唯一索引
DROP INDEX IF EXISTS idx_ansible_templates_name;

-- 创建部分唯一索引（只对未删除的记录生效）
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_templates_name_active 
ON ansible_templates(name) 
WHERE deleted_at IS NULL;

COMMIT;

-- ==========================================
-- 修复 ansible_ssh_keys 唯一索引
-- ==========================================

BEGIN;

-- 删除旧的唯一索引
DROP INDEX IF EXISTS idx_ansible_ssh_keys_name;

-- 创建部分唯一索引（只对未删除的记录生效）
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_ssh_keys_name_active 
ON ansible_ssh_keys(name) 
WHERE deleted_at IS NULL;

COMMIT;

-- ==========================================
-- 验证结果
-- ==========================================

-- 查看新创建的索引
SELECT 
    tablename as "表名",
    indexname as "索引名"
FROM pg_indexes 
WHERE tablename IN ('ansible_inventories', 'ansible_templates', 'ansible_ssh_keys')
    AND indexname LIKE '%_name%'
ORDER BY tablename;

