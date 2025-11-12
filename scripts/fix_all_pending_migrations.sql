-- 修复所有待执行迁移的问题
-- 
-- 问题原因：
-- 多个迁移文件部分执行成功，导致列/索引已存在但迁移记录未保存
-- 
-- 修复内容：
-- 1. 为所有 ALTER TABLE ADD COLUMN 添加 IF NOT EXISTS
-- 2. 为所有 CREATE INDEX 添加 IF NOT EXISTS
-- 3. 删除所有失败的迁移记录，让它们重新执行
--
-- 影响的迁移文件：
-- - 012_add_template_required_vars.sql
-- - 013_add_preflight_checks.sql
-- - 014_add_task_timeout.sql
-- - 015_add_task_priority.sql
-- - 016_add_task_tags.sql
-- - 017_add_execution_timeline.sql
-- - 019_add_workflow_dag.sql

BEGIN;

-- 删除所有可能失败的迁移记录
DELETE FROM schema_migrations WHERE version IN (
    '012_add_template_required_vars.sql',
    '013_add_preflight_checks.sql',
    '014_add_task_timeout.sql',
    '015_add_task_priority.sql',
    '016_add_task_tags.sql',
    '017_add_execution_timeline.sql',
    '018_fix_favorites_foreign_keys.sql',
    '019_add_workflow_dag.sql',
    '021_fix_all_foreign_keys.sql'
);

-- 验证删除
SELECT 
    COUNT(*) as removed_count,
    'All pending migrations cleared, ready to re-run' as status
FROM schema_migrations 
WHERE version IN (
    '012_add_template_required_vars.sql',
    '013_add_preflight_checks.sql',
    '014_add_task_timeout.sql',
    '015_add_task_priority.sql',
    '016_add_task_tags.sql',
    '017_add_execution_timeline.sql',
    '018_fix_favorites_foreign_keys.sql',
    '019_add_workflow_dag.sql',
    '021_fix_all_foreign_keys.sql'
);

COMMIT;

-- 执行后，重新启动应用，所有迁移会自动重新执行
-- 现在所有迁移文件都具有幂等性（IF NOT EXISTS）

