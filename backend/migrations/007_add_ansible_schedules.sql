-- 添加 Ansible 定时任务调度表
-- 支持通过 Cron 表达式定期执行 Ansible 任务

-- ==========================================
-- 1. 创建定时任务调度表
-- ==========================================

CREATE TABLE IF NOT EXISTS ansible_schedules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_id INTEGER NOT NULL,
    inventory_id INTEGER NOT NULL,
    cluster_id INTEGER,
    cron_expr VARCHAR(100) NOT NULL,
    extra_vars JSONB,
    enabled BOOLEAN DEFAULT TRUE,
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    run_count INTEGER DEFAULT 0,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- ==========================================
-- 2. 创建索引
-- ==========================================

-- 基本索引
CREATE INDEX IF NOT EXISTS idx_ansible_schedules_template_id ON ansible_schedules(template_id);
CREATE INDEX IF NOT EXISTS idx_ansible_schedules_inventory_id ON ansible_schedules(inventory_id);
CREATE INDEX IF NOT EXISTS idx_ansible_schedules_cluster_id ON ansible_schedules(cluster_id);
CREATE INDEX IF NOT EXISTS idx_ansible_schedules_user_id ON ansible_schedules(user_id);
CREATE INDEX IF NOT EXISTS idx_ansible_schedules_deleted_at ON ansible_schedules(deleted_at);

-- 功能索引
CREATE INDEX IF NOT EXISTS idx_ansible_schedules_enabled ON ansible_schedules(enabled) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_ansible_schedules_next_run_at ON ansible_schedules(next_run_at) WHERE enabled = TRUE AND deleted_at IS NULL;

-- 唯一索引（支持软删除）
CREATE UNIQUE INDEX IF NOT EXISTS idx_ansible_schedules_name_active 
ON ansible_schedules(name) 
WHERE deleted_at IS NULL;

-- ==========================================
-- 3. 添加注释
-- ==========================================

COMMENT ON TABLE ansible_schedules IS 'Ansible 定时任务调度表';
COMMENT ON COLUMN ansible_schedules.id IS '主键ID';
COMMENT ON COLUMN ansible_schedules.name IS '调度任务名称';
COMMENT ON COLUMN ansible_schedules.description IS '调度任务描述';
COMMENT ON COLUMN ansible_schedules.template_id IS '关联模板ID';
COMMENT ON COLUMN ansible_schedules.inventory_id IS '关联主机清单ID';
COMMENT ON COLUMN ansible_schedules.cluster_id IS '关联集群ID';
COMMENT ON COLUMN ansible_schedules.cron_expr IS 'Cron表达式';
COMMENT ON COLUMN ansible_schedules.extra_vars IS '额外变量(JSON)';
COMMENT ON COLUMN ansible_schedules.enabled IS '是否启用';
COMMENT ON COLUMN ansible_schedules.last_run_at IS '上次执行时间';
COMMENT ON COLUMN ansible_schedules.next_run_at IS '下次执行时间';
COMMENT ON COLUMN ansible_schedules.run_count IS '执行次数';
COMMENT ON COLUMN ansible_schedules.user_id IS '创建用户ID';
COMMENT ON COLUMN ansible_schedules.created_at IS '创建时间';
COMMENT ON COLUMN ansible_schedules.updated_at IS '更新时间';
COMMENT ON COLUMN ansible_schedules.deleted_at IS '删除时间（软删除）';

