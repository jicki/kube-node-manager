-- ========================================
-- 从 SQL 迁移系统过渡到基于代码的迁移系统
-- ========================================
--
-- 用途：
--   此脚本用于将现有的基于 SQL 文件的迁移系统过渡到新的基于代码的迁移系统。
--   通常情况下，应用会自动检测并执行过渡，但如果需要手动执行，可以运行此脚本。
--
-- 使用场景：
--   1. 自动过渡失败时的手动修复
--   2. 在升级前预先执行过渡
--   3. 在测试环境中验证过渡流程
--
-- 执行方式：
--   PostgreSQL: psql -U username -d database_name -f 000_transition_to_code_based.sql
--   SQLite:     sqlite3 database.db < 000_transition_to_code_based.sql
--
-- 注意事项：
--   1. 此脚本是幂等的，可以安全地多次执行
--   2. 执行前请确保已备份数据库
--   3. 此脚本不会删除或修改现有数据
--
-- 版本: 1.0
-- 日期: 2024-11-20
-- ========================================

BEGIN;

-- ========================================
-- 1. 创建 system_metadata 表
-- ========================================

CREATE TABLE IF NOT EXISTS system_metadata (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 为 PostgreSQL 添加注释
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'system_metadata'
    ) THEN
        COMMENT ON TABLE system_metadata IS '系统元数据表，存储系统级别的配置和状态信息';
        COMMENT ON COLUMN system_metadata.key IS '元数据键';
        COMMENT ON COLUMN system_metadata.value IS '元数据值';
        COMMENT ON COLUMN system_metadata.updated_at IS '更新时间';
    END IF;
EXCEPTION
    WHEN others THEN
        -- SQLite 不支持 DO 块和注释，忽略错误
        NULL;
END $$;

-- ========================================
-- 2. 从 schema_migrations 迁移版本信息
-- ========================================

-- 保存最后的 SQL 迁移版本
INSERT INTO system_metadata (key, value, updated_at)
SELECT 
    'last_sql_migration' as key,
    version as value,
    applied_at as updated_at
FROM schema_migrations
ORDER BY applied_at DESC
LIMIT 1
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = EXCLUDED.updated_at;

-- ========================================
-- 3. 标记迁移系统类型
-- ========================================

INSERT INTO system_metadata (key, value, updated_at)
VALUES ('migration_system', 'code_based', CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE SET
    value = 'code_based',
    updated_at = CURRENT_TIMESTAMP;

-- ========================================
-- 4. 创建代码迁移记录表
-- ========================================

CREATE TABLE IF NOT EXISTS code_migration_records (
    id SERIAL PRIMARY KEY,
    migration_id VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    duration_ms BIGINT DEFAULT 0,
    error_msg TEXT,
    applied_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_code_migration_records_applied_at 
    ON code_migration_records(applied_at);
CREATE INDEX IF NOT EXISTS idx_code_migration_records_status 
    ON code_migration_records(status);

-- ========================================
-- 5. 验证过渡结果
-- ========================================

-- 检查 system_metadata 表是否创建成功
DO $$ 
DECLARE
    metadata_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO metadata_count 
    FROM system_metadata;
    
    RAISE NOTICE 'System metadata table created with % records', metadata_count;
    
    -- 显示过渡信息
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Transition Summary:';
    RAISE NOTICE '========================================';
    
    -- 显示最后的 SQL 迁移
    DECLARE
        last_sql_migration VARCHAR(255);
    BEGIN
        SELECT value INTO last_sql_migration 
        FROM system_metadata 
        WHERE key = 'last_sql_migration';
        
        IF last_sql_migration IS NOT NULL THEN
            RAISE NOTICE 'Last SQL Migration: %', last_sql_migration;
        ELSE
            RAISE NOTICE 'No SQL migrations found';
        END IF;
    END;
    
    -- 显示迁移系统类型
    DECLARE
        migration_system VARCHAR(255);
    BEGIN
        SELECT value INTO migration_system 
        FROM system_metadata 
        WHERE key = 'migration_system';
        
        RAISE NOTICE 'Migration System: %', migration_system;
    END;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Transition completed successfully!';
    RAISE NOTICE '========================================';
    
EXCEPTION
    WHEN others THEN
        -- SQLite 不支持 DO 块，忽略错误
        NULL;
END $$;

COMMIT;

-- ========================================
-- 验证查询（手动执行以查看结果）
-- ========================================

-- 查看所有系统元数据
-- SELECT * FROM system_metadata ORDER BY key;

-- 查看最后的 SQL 迁移版本
-- SELECT * FROM system_metadata WHERE key = 'last_sql_migration';

-- 查看迁移系统类型
-- SELECT * FROM system_metadata WHERE key = 'migration_system';

-- 统计 SQL 迁移记录数量
-- SELECT COUNT(*) as sql_migration_count FROM schema_migrations;

-- 统计代码迁移记录数量
-- SELECT COUNT(*) as code_migration_count FROM code_migration_records;

