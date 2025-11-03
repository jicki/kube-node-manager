-- +migrate Up
-- 创建标签表
CREATE TABLE ansible_tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    color VARCHAR(20) DEFAULT '#409EFF',
    description VARCHAR(255),
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- 为标签表创建索引
CREATE INDEX idx_ansible_tags_user_id ON ansible_tags (user_id);
CREATE INDEX idx_ansible_tags_deleted_at ON ansible_tags (deleted_at);

-- 创建任务标签关联表
CREATE TABLE ansible_task_tags (
    task_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (task_id, tag_id),
    FOREIGN KEY (task_id) REFERENCES ansible_tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES ansible_tags(id) ON DELETE CASCADE
);

-- 为关联表创建索引
CREATE INDEX idx_ansible_task_tags_task_id ON ansible_task_tags (task_id);
CREATE INDEX idx_ansible_task_tags_tag_id ON ansible_task_tags (tag_id);

-- +migrate Down
-- 删除关联表
DROP INDEX IF EXISTS idx_ansible_task_tags_tag_id;
DROP INDEX IF EXISTS idx_ansible_task_tags_task_id;
DROP TABLE IF EXISTS ansible_task_tags;

-- 删除标签表
DROP INDEX IF EXISTS idx_ansible_tags_deleted_at;
DROP INDEX IF EXISTS idx_ansible_tags_user_id;
DROP TABLE IF EXISTS ansible_tags;

