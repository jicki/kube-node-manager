-- 修复 ansible_favorites 表的外键约束问题
-- 执行方法：
-- PostgreSQL: psql -U username -d database_name -f fix_favorites_constraints.sql
-- SQLite: sqlite3 database.db < fix_favorites_constraints.sql

-- 删除错误的外键约束（如果存在）
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_task;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_template;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_inventory;

-- PostgreSQL 特有的约束名（GORM 自动生成的）
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_task_id;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_template_id;
ALTER TABLE ansible_favorites DROP CONSTRAINT IF EXISTS fk_ansible_favorites_inventory_id;

-- 创建优化索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_ansible_favorites_user_type_target ON ansible_favorites(user_id, target_type, target_id);

-- 完成
SELECT 'Favorites constraints fixed successfully!' as status;

