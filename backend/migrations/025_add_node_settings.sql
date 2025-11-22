-- create node_settings table
CREATE TABLE IF NOT EXISTS node_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_name VARCHAR(255) NOT NULL,
    node_name VARCHAR(255) NOT NULL,
    ssh_port INTEGER DEFAULT 22,
    ssh_user VARCHAR(255),
    system_ssh_key_id INTEGER,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    FOREIGN KEY (system_ssh_key_id) REFERENCES system_ssh_keys(id)
);

-- create indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_cluster_node ON node_settings(cluster_name, node_name);
CREATE INDEX IF NOT EXISTS idx_node_settings_deleted_at ON node_settings(deleted_at);
