-- 修复迁移 004_fix_ansible_foreign_keys.sql 失败的问题
-- 
-- 原因：
-- 1. SQL 解析器无法正确处理 PostgreSQL 的 DO $$ ... END $$; 块
-- 2. 部分索引已存在（之前部分执行成功）
-- 
-- 解决方案：
-- 1. 改进了 SQL 解析器，支持 dollar-quoted strings
-- 2. 为所有 CREATE INDEX 语句添加 IF NOT EXISTS
-- 3. 从 schema_migrations 表删除失败的迁移记录
-- 4. 重新启动应用，让修复后的迁移重新执行

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

