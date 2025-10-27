-- 002_add_anomaly_analytics.sql
-- 新增异常分析功能的数据库优化
-- 创建时间: 2025-10-27

-- 为节点异常表添加额外的复合索引以优化新的统计查询
-- 注意: 这些索引会在 AUTO MIGRATE 时由 GORM 自动创建，此文件仅作为文档记录

-- 优化按时间范围 + 集群查询（用于统计分析）
-- CREATE INDEX IF NOT EXISTS idx_anomalies_time_cluster ON node_anomalies(start_time, cluster_id);

-- 优化按节点名称 + 时间范围查询（用于单节点趋势分析）
-- CREATE INDEX IF NOT EXISTS idx_anomalies_node_time ON node_anomalies(node_name, start_time);

-- 优化按状态 + 时间范围查询（用于 MTTR 计算）
-- CREATE INDEX IF NOT EXISTS idx_anomalies_status_time ON node_anomalies(status, start_time);

-- 异常报告配置表（由 GORM AUTO MIGRATE 创建）
-- CREATE TABLE IF NOT EXISTS anomaly_report_configs (
--     id                SERIAL PRIMARY KEY,
--     enabled           BOOLEAN DEFAULT FALSE,
--     report_name       VARCHAR(100) NOT NULL,
--     schedule          VARCHAR(50),
--     frequency         VARCHAR(20),
--     cluster_ids       TEXT,
--     feishu_enabled    BOOLEAN DEFAULT FALSE,
--     feishu_webhook    VARCHAR(500),
--     email_enabled     BOOLEAN DEFAULT FALSE,
--     email_recipients  TEXT,
--     last_run_time     TIMESTAMP,
--     next_run_time     TIMESTAMP,
--     created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

-- 为报告配置表添加索引
-- CREATE INDEX IF NOT EXISTS idx_report_configs_enabled ON anomaly_report_configs(enabled);

-- 说明:
-- 1. 本文件的 SQL 语句已注释，因为所有表结构和索引都由 GORM 的 AutoMigrate 自动管理
-- 2. 此文件仅作为文档，记录数据库schema的变更历史
-- 3. 如需手动执行迁移，可以取消注释相应的 SQL 语句
-- 4. 建议使用 GORM AutoMigrate 而不是手动执行SQL，以确保代码和数据库schema一致

-- 数据库兼容性说明:
-- - 支持 SQLite 和 PostgreSQL
-- - SQLite 使用 INTEGER PRIMARY KEY AUTOINCREMENT
-- - PostgreSQL 使用 SERIAL PRIMARY KEY
-- - GORM 会根据数据库类型自动选择合适的语法

