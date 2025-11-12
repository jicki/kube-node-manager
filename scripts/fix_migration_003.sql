-- 修复迁移 003_performance_indexes.sql 失败的问题
-- 
-- 原因：
-- 1. 使用了错误的列名 cluster_name（应该是 cluster_id）
-- 2. 使用了错误的列名 resource_name（应该是 node_name）
-- 3. 引用了不存在的表 feishu_messages 和 feishu_sessions
-- 
-- 解决方案：
-- 1. 从 schema_migrations 表删除失败的迁移记录
-- 2. 重新启动应用，让修复后的迁移文件重新执行

BEGIN;

-- 删除失败的迁移记录
DELETE FROM schema_migrations WHERE version = '003_performance_indexes.sql';

-- 验证删除
SELECT 
    COUNT(*) as remaining_count,
    'Migration 003 removed, ready to re-run' as status
FROM schema_migrations 
WHERE version = '003_performance_indexes.sql';

COMMIT;

-- 执行后，重新启动应用，迁移会自动重新执行

