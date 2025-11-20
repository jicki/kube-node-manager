-- 手动修复 023 迁移问题的脚本
-- 使用场景：当 023_add_node_tracking_to_progress_sqlite.sql 错误地在 PostgreSQL 上执行时

-- 1. 检查并删除 PostgreSQL 上错误添加的 TEXT 类型列（如果存在）
DO $$ 
BEGIN
    -- 检查 progress_tasks 表的 success_nodes 列类型
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_tasks' 
        AND column_name='success_nodes' 
        AND data_type='text'
    ) THEN
        -- 删除 TEXT 类型的列
        ALTER TABLE progress_tasks DROP COLUMN success_nodes;
        ALTER TABLE progress_tasks DROP COLUMN failed_nodes;
        RAISE NOTICE 'Dropped TEXT columns from progress_tasks';
    END IF;

    -- 检查 progress_messages 表的 success_nodes 列类型
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_messages' 
        AND column_name='success_nodes' 
        AND data_type='text'
    ) THEN
        -- 删除 TEXT 类型的列
        ALTER TABLE progress_messages DROP COLUMN success_nodes;
        ALTER TABLE progress_messages DROP COLUMN failed_nodes;
        RAISE NOTICE 'Dropped TEXT columns from progress_messages';
    END IF;
END $$;

-- 2. 添加正确的 JSONB 类型列
DO $$ 
BEGIN
    -- 为 progress_tasks 添加 success_nodes 列（JSONB 类型）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_tasks' AND column_name='success_nodes'
    ) THEN
        ALTER TABLE progress_tasks ADD COLUMN success_nodes JSONB DEFAULT '[]'::JSONB;
        RAISE NOTICE 'Added JSONB success_nodes to progress_tasks';
    END IF;

    -- 为 progress_tasks 添加 failed_nodes 列（JSONB 类型）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_tasks' AND column_name='failed_nodes'
    ) THEN
        ALTER TABLE progress_tasks ADD COLUMN failed_nodes JSONB DEFAULT '[]'::JSONB;
        RAISE NOTICE 'Added JSONB failed_nodes to progress_tasks';
    END IF;

    -- 为 progress_messages 添加 success_nodes 列（JSONB 类型）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_messages' AND column_name='success_nodes'
    ) THEN
        ALTER TABLE progress_messages ADD COLUMN success_nodes JSONB DEFAULT '[]'::JSONB;
        RAISE NOTICE 'Added JSONB success_nodes to progress_messages';
    END IF;

    -- 为 progress_messages 添加 failed_nodes 列（JSONB 类型）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_messages' AND column_name='failed_nodes'
    ) THEN
        ALTER TABLE progress_messages ADD COLUMN failed_nodes JSONB DEFAULT '[]'::JSONB;
        RAISE NOTICE 'Added JSONB failed_nodes to progress_messages';
    END IF;
END $$;

-- 3. 标记迁移为已完成（避免重复执行）
-- 注意：需要根据您的迁移表名调整
INSERT INTO schema_migrations (version, applied_at) 
VALUES ('023_add_node_tracking_to_progress', NOW())
ON CONFLICT (version) DO NOTHING;

-- 显示当前列信息
SELECT 
    table_name,
    column_name,
    data_type,
    column_default
FROM information_schema.columns
WHERE table_name IN ('progress_tasks', 'progress_messages')
  AND column_name IN ('success_nodes', 'failed_nodes')
ORDER BY table_name, column_name;

