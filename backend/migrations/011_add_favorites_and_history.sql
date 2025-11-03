-- +migrate Up
-- 添加收藏和历史记录表

-- 创建收藏表
CREATE TABLE IF NOT EXISTS ansible_favorites (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- 添加注释
COMMENT ON TABLE ansible_favorites IS 'Ansible 收藏记录';
COMMENT ON COLUMN ansible_favorites.user_id IS '用户ID';
COMMENT ON COLUMN ansible_favorites.target_type IS '目标类型(task/template/inventory)';
COMMENT ON COLUMN ansible_favorites.target_id IS '目标ID';

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_ansible_favorites_user_id ON ansible_favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_ansible_favorites_deleted_at ON ansible_favorites(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_favorites_unique ON ansible_favorites(user_id, target_type, target_id) WHERE deleted_at IS NULL;

-- 创建任务历史表
CREATE TABLE IF NOT EXISTS ansible_task_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    task_name VARCHAR(255),
    template_id INTEGER,
    inventory_id INTEGER,
    cluster_id INTEGER,
    playbook_content TEXT,
    extra_vars JSONB,
    dry_run BOOLEAN DEFAULT FALSE,
    batch_config JSONB,
    last_used_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    use_count INTEGER DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 添加注释
COMMENT ON TABLE ansible_task_history IS 'Ansible 任务执行历史';
COMMENT ON COLUMN ansible_task_history.user_id IS '用户ID';
COMMENT ON COLUMN ansible_task_history.task_name IS '任务名称';
COMMENT ON COLUMN ansible_task_history.template_id IS '模板ID';
COMMENT ON COLUMN ansible_task_history.inventory_id IS '清单ID';
COMMENT ON COLUMN ansible_task_history.cluster_id IS '集群ID';
COMMENT ON COLUMN ansible_task_history.playbook_content IS 'Playbook内容';
COMMENT ON COLUMN ansible_task_history.extra_vars IS '额外变量';
COMMENT ON COLUMN ansible_task_history.dry_run IS '是否Dry Run';
COMMENT ON COLUMN ansible_task_history.batch_config IS '分批配置';
COMMENT ON COLUMN ansible_task_history.last_used_at IS '最后使用时间';
COMMENT ON COLUMN ansible_task_history.use_count IS '使用次数';

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_ansible_task_history_user_id ON ansible_task_history(user_id);
CREATE INDEX IF NOT EXISTS idx_ansible_task_history_last_used_at ON ansible_task_history(last_used_at);

-- +migrate Down
-- 删除任务历史表
DROP INDEX IF EXISTS idx_ansible_task_history_last_used_at;
DROP INDEX IF EXISTS idx_ansible_task_history_user_id;
DROP TABLE IF EXISTS ansible_task_history;

-- 删除收藏表
DROP INDEX IF EXISTS idx_ansible_favorites_unique;
DROP INDEX IF EXISTS idx_ansible_favorites_deleted_at;
DROP INDEX IF EXISTS idx_ansible_favorites_user_id;
DROP TABLE IF EXISTS ansible_favorites;
