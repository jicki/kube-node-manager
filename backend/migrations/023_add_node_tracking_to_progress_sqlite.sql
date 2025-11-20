-- +migrate Up
-- 为 progress_tasks 和 progress_messages 表添加节点跟踪字段（SQLite 版本）

-- 注意：这个迁移文件仅用于 SQLite 数据库
-- 如果使用 PostgreSQL，请使用 023_add_node_tracking_to_progress.sql

ALTER TABLE progress_tasks ADD COLUMN success_nodes TEXT DEFAULT '[]';
ALTER TABLE progress_tasks ADD COLUMN failed_nodes TEXT DEFAULT '[]';
ALTER TABLE progress_messages ADD COLUMN success_nodes TEXT DEFAULT '[]';
ALTER TABLE progress_messages ADD COLUMN failed_nodes TEXT DEFAULT '[]';

-- +migrate Down
-- SQLite 不支持直接删除列，需要重建表
-- 为简化起见，在开发环境中可以保留这些列
-- 生产环境建议使用 PostgreSQL
