-- 清理旧的唯一索引，只保留支持软删除的部分索引
-- 用于解决 "duplicate key value violates unique constraint" 错误

-- ==========================================
-- 1. 检查当前索引状态
-- ==========================================

-- 查看所有相关的索引
SELECT 
    schemaname as "Schema",
    tablename as "表名",
    indexname as "索引名",
    CASE 
        WHEN indexdef LIKE '%WHERE deleted_at IS NULL%' THEN '✓ 支持软删除'
        ELSE '✗ 旧索引（需要删除）'
    END as "类型"
FROM pg_indexes 
WHERE tablename IN ('ansible_inventories', 'ansible_templates', 'ansible_ssh_keys')
    AND indexname LIKE '%_name%'
ORDER BY tablename, indexname;

-- ==========================================
-- 2. 删除旧的唯一索引
-- ==========================================

-- ansible_inventories
DROP INDEX IF EXISTS idx_ansible_inventories_name CASCADE;

-- ansible_templates  
DROP INDEX IF EXISTS idx_ansible_templates_name CASCADE;

-- ansible_ssh_keys
DROP INDEX IF EXISTS idx_ansible_ssh_keys_name CASCADE;

-- ==========================================
-- 3. 确保新的部分索引存在
-- ==========================================

-- ansible_inventories
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_inventories_name_active 
ON ansible_inventories(name) 
WHERE deleted_at IS NULL;

-- ansible_templates
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_templates_name_active 
ON ansible_templates(name) 
WHERE deleted_at IS NULL;

-- ansible_ssh_keys
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_ssh_keys_name_active 
ON ansible_ssh_keys(name) 
WHERE deleted_at IS NULL;

-- ==========================================
-- 4. 验证最终结果
-- ==========================================

-- 应该只看到 *_name_active 索引
SELECT 
    tablename as "表名",
    indexname as "索引名",
    indexdef as "索引定义"
FROM pg_indexes 
WHERE tablename IN ('ansible_inventories', 'ansible_templates', 'ansible_ssh_keys')
    AND indexname LIKE '%_name%'
ORDER BY tablename;

-- ==========================================
-- 说明
-- ==========================================

-- 执行完成后，你应该看到：
-- ✓ idx_ansible_inventories_name_active (WHERE deleted_at IS NULL)
-- ✓ idx_ansible_templates_name_active (WHERE deleted_at IS NULL)
-- ✓ idx_ansible_ssh_keys_name_active (WHERE deleted_at IS NULL)

-- 不应该看到：
-- ✗ idx_ansible_inventories_name (没有 WHERE 条件)
-- ✗ idx_ansible_templates_name (没有 WHERE 条件)
-- ✗ idx_ansible_ssh_keys_name (没有 WHERE 条件)

