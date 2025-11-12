-- +migrate Up
-- 工作流定义表
CREATE TABLE IF NOT EXISTS ansible_workflows (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    dag JSONB NOT NULL,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_ansible_workflows_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ansible_workflows_user_id ON ansible_workflows(user_id);
CREATE INDEX IF NOT EXISTS idx_ansible_workflows_deleted_at ON ansible_workflows(deleted_at);
CREATE INDEX IF NOT EXISTS idx_ansible_workflows_name ON ansible_workflows(name);

-- 工作流执行记录表
CREATE TABLE IF NOT EXISTS ansible_workflow_executions (
    id SERIAL PRIMARY KEY,
    workflow_id INTEGER NOT NULL,
    status VARCHAR(50) DEFAULT 'running',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_ansible_workflow_executions_workflow FOREIGN KEY (workflow_id) REFERENCES ansible_workflows(id) ON DELETE CASCADE,
    CONSTRAINT fk_ansible_workflow_executions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ansible_workflow_executions_workflow_id ON ansible_workflow_executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_ansible_workflow_executions_user_id ON ansible_workflow_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_ansible_workflow_executions_status ON ansible_workflow_executions(status);

-- 修改 ansible_tasks 表，添加工作流关联字段
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS workflow_execution_id INTEGER;
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS depends_on JSONB;
ALTER TABLE ansible_tasks ADD COLUMN IF NOT EXISTS node_id VARCHAR(50);

DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_ansible_tasks_workflow_execution' 
        AND table_name = 'ansible_tasks'
    ) THEN
        ALTER TABLE ansible_tasks ADD CONSTRAINT fk_ansible_tasks_workflow_execution 
            FOREIGN KEY (workflow_execution_id) REFERENCES ansible_workflow_executions(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_ansible_tasks_workflow_execution_id ON ansible_tasks(workflow_execution_id);
CREATE INDEX IF NOT EXISTS idx_ansible_tasks_node_id ON ansible_tasks(node_id);

-- +migrate Down
-- 移除 ansible_tasks 表的工作流关联字段
DROP INDEX IF EXISTS idx_ansible_tasks_node_id;
DROP INDEX IF EXISTS idx_ansible_tasks_workflow_execution_id;
ALTER TABLE ansible_tasks DROP CONSTRAINT IF EXISTS fk_ansible_tasks_workflow_execution;
ALTER TABLE ansible_tasks DROP COLUMN IF EXISTS node_id;
ALTER TABLE ansible_tasks DROP COLUMN IF EXISTS depends_on;
ALTER TABLE ansible_tasks DROP COLUMN IF EXISTS workflow_execution_id;

-- 删除工作流执行记录表
DROP INDEX IF EXISTS idx_ansible_workflow_executions_status;
DROP INDEX IF EXISTS idx_ansible_workflow_executions_user_id;
DROP INDEX IF EXISTS idx_ansible_workflow_executions_workflow_id;
DROP TABLE IF EXISTS ansible_workflow_executions;

-- 删除工作流定义表
DROP INDEX IF EXISTS idx_ansible_workflows_name;
DROP INDEX IF EXISTS idx_ansible_workflows_deleted_at;
DROP INDEX IF EXISTS idx_ansible_workflows_user_id;
DROP TABLE IF EXISTS ansible_workflows;

