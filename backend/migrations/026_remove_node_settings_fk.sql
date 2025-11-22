-- 删除 node_settings 表的外键约束
-- 这样可以让 system_ssh_key_id 引用 ansible_ssh_keys 或 system_ssh_keys 表

-- PostgreSQL
DO $$
BEGIN
    -- 删除外键约束
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_node_settings_system_ssh_key' 
        AND table_name = 'node_settings'
    ) THEN
        ALTER TABLE node_settings DROP CONSTRAINT fk_node_settings_system_ssh_key;
    END IF;
    
    -- 如果约束名不同，尝试其他可能的名称
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name LIKE '%node_settings%system_ssh_key%' 
        AND table_name = 'node_settings'
        AND constraint_type = 'FOREIGN KEY'
    ) THEN
        EXECUTE (
            SELECT 'ALTER TABLE node_settings DROP CONSTRAINT ' || constraint_name
            FROM information_schema.table_constraints
            WHERE constraint_name LIKE '%node_settings%system_ssh_key%'
            AND table_name = 'node_settings'
            AND constraint_type = 'FOREIGN KEY'
            LIMIT 1
        );
    END IF;
END $$;

-- SQLite (SQLite 不直接支持删除外键，需要重建表)
-- 如果是 SQLite，需要通过重建表来删除外键
-- 这部分在代码迁移中处理会更安全

