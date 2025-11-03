-- +migrate Up
-- 删除 ansible_favorites 表的错误外键约束
-- 因为 target_id 是动态引用，不应该有固定的外键约束

-- 删除可能存在的外键约束
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_task;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_template;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_inventory;

-- 创建复合索引以优化查询性能
CREATE INDEX IF NOT EXISTS idx_ansible_favorites_user_type_target ON ansible_favorites(user_id, target_type, target_id);

-- +migrate Down
-- 回滚时删除索引
DROP INDEX IF EXISTS idx_ansible_favorites_user_type_target;

