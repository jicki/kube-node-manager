-- +migrate Up
-- 为 progress_tasks 和 progress_messages 表添加节点跟踪字段（PostgreSQL 版本）

-- 添加列前先检查是否存在，避免重复添加
DO $$ 
BEGIN
    -- 为 progress_tasks 添加 success_nodes 列
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_tasks' AND column_name='success_nodes'
    ) THEN
        ALTER TABLE progress_tasks ADD COLUMN success_nodes JSONB DEFAULT '[]'::JSONB;
    END IF;

    -- 为 progress_tasks 添加 failed_nodes 列
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_tasks' AND column_name='failed_nodes'
    ) THEN
        ALTER TABLE progress_tasks ADD COLUMN failed_nodes JSONB DEFAULT '[]'::JSONB;
    END IF;

    -- 为 progress_messages 添加 success_nodes 列
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_messages' AND column_name='success_nodes'
    ) THEN
        ALTER TABLE progress_messages ADD COLUMN success_nodes JSONB DEFAULT '[]'::JSONB;
    END IF;

    -- 为 progress_messages 添加 failed_nodes 列
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='progress_messages' AND column_name='failed_nodes'
    ) THEN
        ALTER TABLE progress_messages ADD COLUMN failed_nodes JSONB DEFAULT '[]'::JSONB;
    END IF;
END $$;

-- 添加注释
COMMENT ON COLUMN progress_tasks.success_nodes IS '成功处理的节点列表（JSON 数组）';
COMMENT ON COLUMN progress_tasks.failed_nodes IS '失败处理的节点列表（JSON 数组，包含节点名和错误信息）';
COMMENT ON COLUMN progress_messages.success_nodes IS '成功处理的节点列表（JSON 数组）';
COMMENT ON COLUMN progress_messages.failed_nodes IS '失败处理的节点列表（JSON 数组，包含节点名和错误信息）';

-- +migrate Down
-- 回滚：删除添加的列
ALTER TABLE progress_tasks DROP COLUMN IF EXISTS success_nodes;
ALTER TABLE progress_tasks DROP COLUMN IF EXISTS failed_nodes;
ALTER TABLE progress_messages DROP COLUMN IF EXISTS success_nodes;
ALTER TABLE progress_messages DROP COLUMN IF EXISTS failed_nodes;
