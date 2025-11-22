-- Migration: 024_migrate_ansible_ssh_keys_to_system.sql
-- 目的：将 Ansible SSH 密钥迁移到系统级 SSH 密钥
-- 日期：2025
-- 
-- 此迁移将：
-- 1. 创建 system_ssh_keys 表（如果不存在）
-- 2. 将 ansible_ssh_keys 表中的数据迁移到 system_ssh_keys
-- 3. 更新 ansible_inventories 表中的 ssh_key_id 引用
-- 4. 保留 ansible_ssh_keys 表以便回滚（标记为已废弃）

-- ========================================
-- 1. 创建系统级 SSH 密钥表
-- ========================================
CREATE TABLE IF NOT EXISTS system_ssh_keys (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,           -- 'private_key' 或 'password'
    username VARCHAR(255) NOT NULL,      -- SSH 用户名
    private_key TEXT,                     -- 私钥内容（加密存储）
    passphrase TEXT,                      -- 私钥密码（加密存储）
    password TEXT,                        -- SSH 密码（加密存储）
    port INTEGER DEFAULT 22,             -- SSH 端口
    is_default BOOLEAN DEFAULT FALSE,    -- 是否为默认密钥
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

-- 创建唯一索引（支持软删除）
CREATE UNIQUE INDEX IF NOT EXISTS idx_system_ssh_keys_name_not_deleted 
    ON system_ssh_keys (name) 
    WHERE deleted_at IS NULL;

-- 创建其他索引
CREATE INDEX IF NOT EXISTS idx_system_ssh_keys_type ON system_ssh_keys (type);
CREATE INDEX IF NOT EXISTS idx_system_ssh_keys_deleted_at ON system_ssh_keys (deleted_at);
CREATE INDEX IF NOT EXISTS idx_system_ssh_keys_is_default ON system_ssh_keys (is_default);

-- ========================================
-- 2. 迁移数据
-- ========================================
-- 将 ansible_ssh_keys 中的数据复制到 system_ssh_keys
-- 保留原始 ID 映射关系以便更新引用
DO $$
DECLARE
    old_key RECORD;
    new_key_id INTEGER;
    mapping_exists BOOLEAN;
BEGIN
    -- 检查是否已经有迁移记录（避免重复迁移）
    -- 使用临时表存储ID映射关系
    CREATE TEMP TABLE IF NOT EXISTS ssh_key_id_mapping (
        old_id INTEGER PRIMARY KEY,
        new_id INTEGER NOT NULL
    );
    
    -- 遍历所有 ansible_ssh_keys 记录
    FOR old_key IN 
        SELECT id, name, description, type, username, 
               private_key, passphrase, password, port, is_default, 
               created_by, created_at, updated_at, deleted_at
        FROM ansible_ssh_keys
        WHERE id NOT IN (SELECT old_id FROM ssh_key_id_mapping)
    LOOP
        -- 检查是否已存在同名的系统SSH密钥
        SELECT id INTO new_key_id
        FROM system_ssh_keys
        WHERE name = old_key.name 
          AND (deleted_at IS NULL OR deleted_at = old_key.deleted_at);
        
        IF new_key_id IS NULL THEN
            -- 插入新记录
            INSERT INTO system_ssh_keys (
                name, description, type, username,
                private_key, passphrase, password, port, is_default,
                created_by, created_at, updated_at, deleted_at
            ) VALUES (
                old_key.name, old_key.description, old_key.type, old_key.username,
                old_key.private_key, old_key.passphrase, old_key.password, 
                old_key.port, old_key.is_default,
                old_key.created_by, old_key.created_at, old_key.updated_at, old_key.deleted_at
            ) RETURNING id INTO new_key_id;
            
            RAISE NOTICE 'Migrated SSH key: % (old ID: %, new ID: %)', old_key.name, old_key.id, new_key_id;
        ELSE
            RAISE NOTICE 'SSH key already exists: % (old ID: %, existing ID: %)', old_key.name, old_key.id, new_key_id;
        END IF;
        
        -- 记录ID映射关系
        INSERT INTO ssh_key_id_mapping (old_id, new_id) VALUES (old_key.id, new_key_id);
    END LOOP;
END $$;

-- ========================================
-- 3. 更新 Inventory 引用
-- ========================================
-- 更新 ansible_inventories 表中的 ssh_key_id 引用
UPDATE ansible_inventories
SET ssh_key_id = mapping.new_id
FROM (SELECT old_id, new_id FROM ssh_key_id_mapping) AS mapping
WHERE ansible_inventories.ssh_key_id = mapping.old_id;

-- ========================================
-- 4. 添加注释标记旧表已废弃
-- ========================================
COMMENT ON TABLE ansible_ssh_keys IS 'DEPRECATED: This table is deprecated. Use system_ssh_keys instead. Kept for backward compatibility and rollback purposes.';

-- ========================================
-- 5. 清理
-- ========================================
-- 删除临时映射表（如果不需要保留映射关系）
DROP TABLE IF EXISTS ssh_key_id_mapping;

-- ========================================
-- 验证迁移结果
-- ========================================
DO $$
DECLARE
    ansible_count INTEGER;
    system_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO ansible_count FROM ansible_ssh_keys WHERE deleted_at IS NULL;
    SELECT COUNT(*) INTO system_count FROM system_ssh_keys WHERE deleted_at IS NULL;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Migration Summary:';
    RAISE NOTICE 'Ansible SSH Keys (not deleted): %', ansible_count;
    RAISE NOTICE 'System SSH Keys (not deleted): %', system_count;
    RAISE NOTICE '========================================';
    
    IF system_count < ansible_count THEN
        RAISE WARNING 'System SSH keys count is less than Ansible SSH keys count. Please verify migration!';
    END IF;
END $$;

