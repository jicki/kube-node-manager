-- 修复迁移 004_fix_ansible_foreign_keys.sql 失败的问题
-- 
-- 原因：
-- SQL 解析器无法正确处理 PostgreSQL 的 DO $$ ... END $$; 块
-- 
-- 解决方案：
-- 1. 改进了 SQL 解析器，支持 dollar-quoted strings
-- 2. 从 schema_migrations 表删除失败的迁移记录
-- 3. 重新启动应用，让改进后的解析器重新执行迁移

BEGIN;

-- 删除失败的迁移记录
DELETE FROM schema_migrations WHERE version = '004_fix_ansible_foreign_keys.sql';

-- 验证删除
SELECT 
    COUNT(*) as remaining_count,
    'Migration 004 removed, ready to re-run with improved SQL parser' as status
FROM schema_migrations 
WHERE version = '004_fix_ansible_foreign_keys.sql';

COMMIT;

-- 执行后，重新启动应用，迁移会自动重新执行

