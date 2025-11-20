package database

import (
	"fmt"
)

// DatabaseType 数据库类型
type DatabaseType string

const (
	DatabaseTypePostgreSQL DatabaseType = "postgres"
	DatabaseTypeSQLite     DatabaseType = "sqlite"
)

// ColumnDefinition 字段定义
type ColumnDefinition struct {
	Name         string
	Type         string  // 数据库类型
	Nullable     bool
	DefaultValue *string
	PrimaryKey   bool
	Unique       bool
	AutoIncr     bool
	Comment      string
	ForeignKey   *ForeignKeyDef
}

// ForeignKeyDef 外键定义
type ForeignKeyDef struct {
	Table      string
	Column     string
	OnDelete   string // CASCADE, SET NULL, RESTRICT
	OnUpdate   string
}

// IndexDefinition 索引定义
type IndexDefinition struct {
	Name    string
	Columns []string
	Unique  bool
	Type    string // btree, hash, gin, etc.
}

// ConstraintDefinition 约束定义
type ConstraintDefinition struct {
	Name       string
	Type       string // CHECK, UNIQUE, etc.
	Expression string
}

// TableSchema 表结构定义
type TableSchema struct {
	Name        string
	Columns     []ColumnDefinition
	Indexes     []IndexDefinition
	Constraints []ConstraintDefinition
	Comment     string
}

// GetType 根据数据库类型返回对应的字段类型
func (c *ColumnDefinition) GetType(dbType DatabaseType) string {
	if dbType == DatabaseTypeSQLite {
		// SQLite 类型映射
		switch c.Type {
		case "SERIAL", "BIGSERIAL":
			return "INTEGER"
		case "VARCHAR", "TEXT", "JSONB", "BYTEA":
			return "TEXT"
		case "INTEGER", "BIGINT", "INT", "SMALLINT":
			return "INTEGER"
		case "BOOLEAN":
			return "INTEGER"
		case "TIMESTAMP", "TIMESTAMPTZ":
			return "DATETIME"
		case "NUMERIC", "DECIMAL", "DOUBLE PRECISION", "REAL":
			return "REAL"
		default:
			return "TEXT"
		}
	}
	// PostgreSQL 使用原始类型
	return c.Type
}

// AllTableSchemas 返回所有表的结构定义
func AllTableSchemas() []TableSchema {
	return []TableSchema{
		usersTableSchema(),
		clustersTableSchema(),
		labelTemplatesTableSchema(),
		taintTemplatesTableSchema(),
		auditLogsTableSchema(),
		progressTasksTableSchema(),
		progressMessagesTableSchema(),
		gitlabSettingsTableSchema(),
		gitlabRunnersTableSchema(),
		feishuSettingsTableSchema(),
		feishuUserMappingsTableSchema(),
		feishuUserSessionsTableSchema(),
		nodeAnomaliesTableSchema(),
		anomalyReportConfigsTableSchema(),
		cacheEntriesTableSchema(),
		ansibleTasksTableSchema(),
		ansibleTemplatesTableSchema(),
		ansibleLogsTableSchema(),
		ansibleInventoriesTableSchema(),
		ansibleSSHKeysTableSchema(),
		ansibleSchedulesTableSchema(),
		ansibleFavoritesTableSchema(),
		ansibleTaskHistoryTableSchema(),
		ansibleTagsTableSchema(),
		ansibleTaskTagsTableSchema(),
		ansibleWorkflowsTableSchema(),
		ansibleWorkflowExecutionsTableSchema(),
		schemaMigrationsTableSchema(),
	}
}

// usersTableSchema users 表结构
func usersTableSchema() TableSchema {
	return TableSchema{
		Name: "users",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "username", Type: "VARCHAR(255)", Nullable: false, Unique: true},
			{Name: "email", Type: "VARCHAR(255)", Nullable: false, Unique: true},
			{Name: "password", Type: "VARCHAR(255)", Nullable: false},
			{Name: "role", Type: "VARCHAR(50)", Nullable: false, DefaultValue: strPtr("user")},
			{Name: "status", Type: "VARCHAR(50)", Nullable: false, DefaultValue: strPtr("active")},
			{Name: "is_ldap_user", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "last_login", Type: "TIMESTAMP", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_users_deleted_at", Columns: []string{"deleted_at"}},
			{Name: "idx_users_username", Columns: []string{"username"}, Unique: true},
			{Name: "idx_users_email", Columns: []string{"email"}, Unique: true},
		},
		Comment: "用户表",
	}
}

// clustersTableSchema clusters 表结构
func clustersTableSchema() TableSchema {
	return TableSchema{
		Name: "clusters",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false, Unique: true},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "kube_config", Type: "TEXT", Nullable: false},
			{Name: "status", Type: "VARCHAR(50)", Nullable: false, DefaultValue: strPtr("active")},
			{Name: "version", Type: "VARCHAR(100)", Nullable: true},
			{Name: "node_count", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "last_sync", Type: "TIMESTAMP", Nullable: true},
			{Name: "priority", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "created_by", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_clusters_deleted_at", Columns: []string{"deleted_at"}},
			{Name: "idx_clusters_name", Columns: []string{"name"}, Unique: true},
			{Name: "idx_clusters_created_by", Columns: []string{"created_by"}},
		},
		Comment: "集群表",
	}
}

// labelTemplatesTableSchema label_templates 表结构
func labelTemplatesTableSchema() TableSchema {
	return TableSchema{
		Name: "label_templates",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "labels", Type: "TEXT", Nullable: false, Comment: "JSON格式存储"},
			{Name: "created_by", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_label_templates_deleted_at", Columns: []string{"deleted_at"}},
			{Name: "idx_label_templates_created_by", Columns: []string{"created_by"}},
		},
		Comment: "标签模板表",
	}
}

// taintTemplatesTableSchema taint_templates 表结构
func taintTemplatesTableSchema() TableSchema {
	return TableSchema{
		Name: "taint_templates",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "taints", Type: "TEXT", Nullable: false, Comment: "JSON格式存储"},
			{Name: "created_by", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_taint_templates_deleted_at", Columns: []string{"deleted_at"}},
			{Name: "idx_taint_templates_created_by", Columns: []string{"created_by"}},
		},
		Comment: "污点模板表",
	}
}

// auditLogsTableSchema audit_logs 表结构
func auditLogsTableSchema() TableSchema {
	return TableSchema{
		Name: "audit_logs",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "cluster_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "clusters", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "node_name", Type: "VARCHAR(255)", Nullable: true},
			{Name: "action", Type: "VARCHAR(50)", Nullable: false},
			{Name: "resource_type", Type: "VARCHAR(50)", Nullable: false},
			{Name: "details", Type: "TEXT", Nullable: true},
			{Name: "reason", Type: "TEXT", Nullable: true},
			{Name: "status", Type: "VARCHAR(50)", Nullable: false, DefaultValue: strPtr("success")},
			{Name: "error_msg", Type: "TEXT", Nullable: true},
			{Name: "ip_address", Type: "VARCHAR(50)", Nullable: true},
			{Name: "user_agent", Type: "TEXT", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_audit_logs_user_id", Columns: []string{"user_id"}},
			{Name: "idx_audit_logs_cluster_id", Columns: []string{"cluster_id"}},
			{Name: "idx_audit_logs_created_at", Columns: []string{"created_at"}},
		},
		Comment: "审计日志表",
	}
}

// progressTasksTableSchema progress_tasks 表结构
func progressTasksTableSchema() TableSchema {
	return TableSchema{
		Name: "progress_tasks",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "task_id", Type: "VARCHAR(255)", Nullable: false, Unique: true},
			{Name: "user_id", Type: "INTEGER", Nullable: false},
			{Name: "action", Type: "VARCHAR(100)", Nullable: false},
			{Name: "status", Type: "VARCHAR(50)", Nullable: false},
			{Name: "current", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "total", Type: "INTEGER", Nullable: false},
			{Name: "progress", Type: "DOUBLE PRECISION", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "current_node", Type: "VARCHAR(255)", Nullable: true},
			{Name: "success_nodes", Type: "TEXT", Nullable: true, Comment: "JSON数组"},
			{Name: "failed_nodes", Type: "TEXT", Nullable: true, Comment: "JSON数组"},
			{Name: "message", Type: "TEXT", Nullable: true},
			{Name: "error_msg", Type: "TEXT", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "completed_at", Type: "TIMESTAMP", Nullable: true},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_progress_tasks_task_id", Columns: []string{"task_id"}, Unique: true},
			{Name: "idx_progress_tasks_user_id", Columns: []string{"user_id"}},
			{Name: "idx_progress_tasks_status", Columns: []string{"status"}},
			{Name: "idx_progress_tasks_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "进度任务表",
	}
}

// progressMessagesTableSchema progress_messages 表结构
func progressMessagesTableSchema() TableSchema {
	return TableSchema{
		Name: "progress_messages",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "user_id", Type: "INTEGER", Nullable: false},
			{Name: "task_id", Type: "VARCHAR(255)", Nullable: false},
			{Name: "type", Type: "VARCHAR(50)", Nullable: false},
			{Name: "action", Type: "VARCHAR(100)", Nullable: true},
			{Name: "current", Type: "INTEGER", Nullable: true},
			{Name: "total", Type: "INTEGER", Nullable: true},
			{Name: "progress", Type: "DOUBLE PRECISION", Nullable: true},
			{Name: "current_node", Type: "VARCHAR(255)", Nullable: true},
			{Name: "success_nodes", Type: "TEXT", Nullable: true, Comment: "JSON数组"},
			{Name: "failed_nodes", Type: "TEXT", Nullable: true, Comment: "JSON数组"},
			{Name: "message", Type: "TEXT", Nullable: true},
			{Name: "error_msg", Type: "TEXT", Nullable: true},
			{Name: "processed", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_progress_messages_user_id", Columns: []string{"user_id"}},
			{Name: "idx_progress_messages_task_id", Columns: []string{"task_id"}},
			{Name: "idx_progress_messages_processed", Columns: []string{"processed"}},
			{Name: "idx_progress_messages_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "进度消息表",
	}
}

// gitlabSettingsTableSchema gitlab_settings 表结构
func gitlabSettingsTableSchema() TableSchema {
	return TableSchema{
		Name: "gitlab_settings",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "enabled", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "domain", Type: "VARCHAR(255)", Nullable: true},
			{Name: "token", Type: "TEXT", Nullable: true, Comment: "加密存储"},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_gitlab_settings_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "GitLab配置表",
	}
}

// gitlabRunnersTableSchema gitlab_runners 表结构
func gitlabRunnersTableSchema() TableSchema {
	return TableSchema{
		Name: "gitlab_runners",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "runner_id", Type: "INTEGER", Nullable: false, Unique: true},
			{Name: "token", Type: "TEXT", Nullable: false, Comment: "加密存储"},
			{Name: "description", Type: "VARCHAR(255)", Nullable: true},
			{Name: "runner_type", Type: "VARCHAR(50)", Nullable: true},
			{Name: "created_by", Type: "VARCHAR(100)", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_gitlab_runners_runner_id", Columns: []string{"runner_id"}, Unique: true},
			{Name: "idx_gitlab_runners_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "GitLab Runner表",
	}
}

// feishuSettingsTableSchema feishu_settings 表结构
func feishuSettingsTableSchema() TableSchema {
	return TableSchema{
		Name: "feishu_settings",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "enabled", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "app_id", Type: "VARCHAR(255)", Nullable: true},
			{Name: "app_secret", Type: "TEXT", Nullable: true, Comment: "加密存储"},
			{Name: "bot_enabled", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_feishu_settings_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "飞书配置表",
	}
}

// feishuUserMappingsTableSchema feishu_user_mappings 表结构
func feishuUserMappingsTableSchema() TableSchema {
	return TableSchema{
		Name: "feishu_user_mappings",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "feishu_user_id", Type: "VARCHAR(255)", Nullable: false, Unique: true},
			{Name: "system_user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "CASCADE",
			}},
			{Name: "username", Type: "VARCHAR(100)", Nullable: true},
			{Name: "feishu_name", Type: "VARCHAR(255)", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_feishu_user_mappings_feishu_user_id", Columns: []string{"feishu_user_id"}, Unique: true},
			{Name: "idx_feishu_user_mappings_system_user_id", Columns: []string{"system_user_id"}},
			{Name: "idx_feishu_user_mappings_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "飞书用户映射表",
	}
}

// feishuUserSessionsTableSchema feishu_user_sessions 表结构
func feishuUserSessionsTableSchema() TableSchema {
	return TableSchema{
		Name: "feishu_user_sessions",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "feishu_user_id", Type: "VARCHAR(255)", Nullable: false, Unique: true},
			{Name: "current_cluster", Type: "VARCHAR(255)", Nullable: true},
			{Name: "last_command_time", Type: "TIMESTAMP", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_feishu_user_sessions_feishu_user_id", Columns: []string{"feishu_user_id"}, Unique: true},
			{Name: "idx_feishu_user_sessions_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "飞书用户会话表",
	}
}

// nodeAnomaliesTableSchema node_anomalies 表结构
func nodeAnomaliesTableSchema() TableSchema {
	return TableSchema{
		Name: "node_anomalies",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "cluster_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "clusters", Column: "id", OnDelete: "CASCADE",
			}},
			{Name: "cluster_name", Type: "VARCHAR(255)", Nullable: false},
			{Name: "node_name", Type: "VARCHAR(255)", Nullable: false},
			{Name: "anomaly_type", Type: "VARCHAR(50)", Nullable: false},
			{Name: "status", Type: "VARCHAR(50)", Nullable: false, DefaultValue: strPtr("Active")},
			{Name: "start_time", Type: "TIMESTAMP", Nullable: false},
			{Name: "end_time", Type: "TIMESTAMP", Nullable: true},
			{Name: "duration", Type: "BIGINT", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "reason", Type: "TEXT", Nullable: true},
			{Name: "message", Type: "TEXT", Nullable: true},
			{Name: "last_check", Type: "TIMESTAMP", Nullable: false},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_cluster_node", Columns: []string{"cluster_id", "node_name"}},
			{Name: "idx_anomaly_type", Columns: []string{"anomaly_type"}},
			{Name: "idx_status", Columns: []string{"status"}},
			{Name: "idx_start_time", Columns: []string{"start_time"}},
			{Name: "idx_node_anomalies_deleted_at", Columns: []string{"deleted_at"}},
			{Name: "idx_anomalies_cluster_time_status", Columns: []string{"cluster_id", "start_time", "status"}},
			{Name: "idx_anomalies_statistics", Columns: []string{"cluster_id", "anomaly_type", "status", "start_time"}},
			{Name: "idx_anomalies_node_time", Columns: []string{"node_name", "start_time"}},
		},
		Comment: "节点异常记录表",
	}
}

// anomalyReportConfigsTableSchema anomaly_report_configs 表结构
func anomalyReportConfigsTableSchema() TableSchema {
	return TableSchema{
		Name: "anomaly_report_configs",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "enabled", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "report_name", Type: "VARCHAR(100)", Nullable: false},
			{Name: "schedule", Type: "VARCHAR(50)", Nullable: true},
			{Name: "frequency", Type: "VARCHAR(20)", Nullable: true},
			{Name: "cluster_ids", Type: "TEXT", Nullable: true, Comment: "JSON数组"},
			{Name: "feishu_enabled", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "feishu_webhook", Type: "VARCHAR(500)", Nullable: true},
			{Name: "email_enabled", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "email_recipients", Type: "TEXT", Nullable: true, Comment: "JSON数组"},
			{Name: "last_run_time", Type: "TIMESTAMP", Nullable: true},
			{Name: "next_run_time", Type: "TIMESTAMP", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_anomaly_report_configs_enabled", Columns: []string{"enabled"}},
		},
		Comment: "异常报告配置表",
	}
}

// cacheEntriesTableSchema cache_entries 表结构
func cacheEntriesTableSchema() TableSchema {
	return TableSchema{
		Name: "cache_entries",
		Columns: []ColumnDefinition{
			{Name: "key", Type: "VARCHAR(255)", PrimaryKey: true, Nullable: false},
			{Name: "value", Type: "BYTEA", Nullable: false},
			{Name: "expires_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_cache_entries_expires_at", Columns: []string{"expires_at"}},
		},
		Comment: "缓存条目表",
	}
}

// ansibleTasksTableSchema ansible_tasks 表结构
func ansibleTasksTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_tasks",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false, Comment: "任务名称"},
			{Name: "template_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "ansible_templates", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "cluster_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "clusters", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "inventory_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "ansible_inventories", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "status", Type: "VARCHAR(50)", Nullable: false, Comment: "任务状态"},
			{Name: "user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "started_at", Type: "TIMESTAMP", Nullable: true, Comment: "开始时间"},
			{Name: "finished_at", Type: "TIMESTAMP", Nullable: true, Comment: "完成时间"},
			{Name: "duration", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0"), Comment: "执行时长(秒)"},
			{Name: "hosts_total", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "hosts_ok", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "hosts_failed", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "hosts_skipped", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "error_msg", Type: "TEXT", Nullable: true},
			{Name: "playbook_content", Type: "TEXT", Nullable: false, Comment: "Playbook内容"},
			{Name: "full_log", Type: "TEXT", Nullable: true, Comment: "完整日志"},
			{Name: "log_size", Type: "BIGINT", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "extra_vars", Type: "JSONB", Nullable: true, Comment: "额外变量"},
			{Name: "retry_policy", Type: "JSONB", Nullable: true, Comment: "重试策略"},
			{Name: "retry_count", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "max_retries", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "dry_run", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "batch_config", Type: "JSONB", Nullable: true, Comment: "分批执行配置"},
			{Name: "current_batch", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "total_batches", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "batch_status", Type: "VARCHAR(50)", Nullable: true},
			{Name: "preflight_checks", Type: "JSONB", Nullable: true, Comment: "前置检查结果"},
			{Name: "timeout_seconds", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "is_timed_out", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "priority", Type: "VARCHAR(20)", Nullable: false, DefaultValue: strPtr("medium")},
			{Name: "queued_at", Type: "TIMESTAMP", Nullable: true},
			{Name: "wait_duration", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "execution_timeline", Type: "JSONB", Nullable: true, Comment: "执行时间线"},
			{Name: "workflow_execution_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "ansible_workflow_executions", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "depends_on", Type: "JSONB", Nullable: true, Comment: "依赖的节点ID列表"},
			{Name: "node_id", Type: "VARCHAR(50)", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_tasks_template_id", Columns: []string{"template_id"}},
			{Name: "idx_ansible_tasks_cluster_id", Columns: []string{"cluster_id"}},
			{Name: "idx_ansible_tasks_inventory_id", Columns: []string{"inventory_id"}},
			{Name: "idx_ansible_tasks_status", Columns: []string{"status"}},
			{Name: "idx_ansible_tasks_user_id", Columns: []string{"user_id"}},
			{Name: "idx_ansible_tasks_priority", Columns: []string{"priority"}},
			{Name: "idx_ansible_tasks_workflow_execution_id", Columns: []string{"workflow_execution_id"}},
			{Name: "idx_ansible_tasks_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "Ansible任务表",
	}
}

// ansibleTemplatesTableSchema ansible_templates 表结构
func ansibleTemplatesTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_templates",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false, Comment: "模板名称"},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "playbook_content", Type: "TEXT", Nullable: false, Comment: "Playbook内容"},
			{Name: "variables", Type: "JSONB", Nullable: true, Comment: "变量定义"},
			{Name: "required_vars", Type: "JSONB", Nullable: true, Comment: "必需变量列表"},
			{Name: "tags", Type: "VARCHAR(255)", Nullable: true},
			{Name: "risk_level", Type: "VARCHAR(20)", Nullable: false, DefaultValue: strPtr("low")},
			{Name: "user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_templates_user_id", Columns: []string{"user_id"}},
			{Name: "idx_ansible_templates_deleted_at", Columns: []string{"deleted_at"}},
			{Name: "idx_ansible_templates_name_deleted_at", Columns: []string{"name", "deleted_at"}, Unique: true},
		},
		Comment: "Ansible模板表",
	}
}

// ansibleLogsTableSchema ansible_logs 表结构
func ansibleLogsTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_logs",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "task_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "ansible_tasks", Column: "id", OnDelete: "CASCADE",
			}},
			{Name: "log_type", Type: "VARCHAR(50)", Nullable: true},
			{Name: "content", Type: "TEXT", Nullable: false},
			{Name: "line_number", Type: "INTEGER", Nullable: true},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_logs_task_id", Columns: []string{"task_id"}},
			{Name: "idx_ansible_logs_created_at", Columns: []string{"created_at"}},
		},
		Comment: "Ansible日志表",
	}
}

// ansibleInventoriesTableSchema ansible_inventories 表结构
func ansibleInventoriesTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_inventories",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false, Comment: "清单名称"},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "source_type", Type: "VARCHAR(50)", Nullable: true},
			{Name: "cluster_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "clusters", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "ssh_key_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "ansible_ssh_keys", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "content", Type: "TEXT", Nullable: false, Comment: "清单内容(INI或YAML)"},
			{Name: "hosts_data", Type: "JSONB", Nullable: true, Comment: "结构化主机数据"},
			{Name: "environment", Type: "VARCHAR(20)", Nullable: false, DefaultValue: strPtr("dev")},
			{Name: "user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_inventories_cluster_id", Columns: []string{"cluster_id"}},
			{Name: "idx_ansible_inventories_ssh_key_id", Columns: []string{"ssh_key_id"}},
			{Name: "idx_ansible_inventories_user_id", Columns: []string{"user_id"}},
			{Name: "idx_ansible_inventories_deleted_at", Columns: []string{"deleted_at"}},
			{Name: "idx_ansible_inventories_name_deleted_at", Columns: []string{"name", "deleted_at"}, Unique: true},
		},
		Comment: "Ansible主机清单表",
	}
}

// ansibleSSHKeysTableSchema ansible_ssh_keys 表结构
func ansibleSSHKeysTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_ssh_keys",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "type", Type: "VARCHAR(50)", Nullable: false},
			{Name: "username", Type: "VARCHAR(255)", Nullable: false},
			{Name: "private_key", Type: "TEXT", Nullable: true, Comment: "加密存储"},
			{Name: "passphrase", Type: "TEXT", Nullable: true, Comment: "加密存储"},
			{Name: "password", Type: "TEXT", Nullable: true, Comment: "加密存储"},
			{Name: "port", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("22")},
			{Name: "is_default", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "created_by", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_ssh_keys_created_by", Columns: []string{"created_by"}},
			{Name: "idx_ansible_ssh_keys_deleted_at", Columns: []string{"deleted_at"}},
			{Name: "idx_ansible_ssh_keys_name_deleted_at", Columns: []string{"name", "deleted_at"}, Unique: true},
		},
		Comment: "Ansible SSH密钥表",
	}
}

// ansibleSchedulesTableSchema ansible_schedules 表结构
func ansibleSchedulesTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_schedules",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "template_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "ansible_templates", Column: "id", OnDelete: "CASCADE",
			}},
			{Name: "inventory_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "ansible_inventories", Column: "id", OnDelete: "CASCADE",
			}},
			{Name: "cluster_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "clusters", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "cron_expr", Type: "VARCHAR(100)", Nullable: false, Comment: "Cron表达式"},
			{Name: "extra_vars", Type: "JSONB", Nullable: true},
			{Name: "enabled", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("true")},
			{Name: "last_run_at", Type: "TIMESTAMP", Nullable: true},
			{Name: "next_run_at", Type: "TIMESTAMP", Nullable: true},
			{Name: "run_count", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("0")},
			{Name: "user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_schedules_template_id", Columns: []string{"template_id"}},
			{Name: "idx_ansible_schedules_inventory_id", Columns: []string{"inventory_id"}},
			{Name: "idx_ansible_schedules_cluster_id", Columns: []string{"cluster_id"}},
			{Name: "idx_ansible_schedules_enabled", Columns: []string{"enabled"}},
			{Name: "idx_ansible_schedules_next_run_at", Columns: []string{"next_run_at"}},
			{Name: "idx_ansible_schedules_user_id", Columns: []string{"user_id"}},
			{Name: "idx_ansible_schedules_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "Ansible定时任务表",
	}
}

// ansibleFavoritesTableSchema ansible_favorites 表结构
func ansibleFavoritesTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_favorites",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "user_id", Type: "INTEGER", Nullable: false},
			{Name: "target_type", Type: "VARCHAR(50)", Nullable: false, Comment: "task/template/inventory"},
			{Name: "target_id", Type: "INTEGER", Nullable: false},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_favorites_user_id", Columns: []string{"user_id"}},
			{Name: "idx_ansible_favorites_target", Columns: []string{"target_type", "target_id"}},
			{Name: "idx_ansible_favorites_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "Ansible收藏表",
	}
}

// ansibleTaskHistoryTableSchema ansible_task_history 表结构
func ansibleTaskHistoryTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_task_history",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "user_id", Type: "INTEGER", Nullable: false},
			{Name: "task_name", Type: "VARCHAR(255)", Nullable: true},
			{Name: "template_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "ansible_templates", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "inventory_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "ansible_inventories", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "cluster_id", Type: "INTEGER", Nullable: true, ForeignKey: &ForeignKeyDef{
				Table: "clusters", Column: "id", OnDelete: "SET NULL",
			}},
			{Name: "playbook_content", Type: "TEXT", Nullable: true},
			{Name: "extra_vars", Type: "JSONB", Nullable: true},
			{Name: "dry_run", Type: "BOOLEAN", Nullable: false, DefaultValue: strPtr("false")},
			{Name: "batch_config", Type: "JSONB", Nullable: true},
			{Name: "last_used_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "use_count", Type: "INTEGER", Nullable: false, DefaultValue: strPtr("1")},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_task_history_user_id", Columns: []string{"user_id"}},
			{Name: "idx_ansible_task_history_template_id", Columns: []string{"template_id"}},
			{Name: "idx_ansible_task_history_inventory_id", Columns: []string{"inventory_id"}},
			{Name: "idx_ansible_task_history_cluster_id", Columns: []string{"cluster_id"}},
			{Name: "idx_ansible_task_history_last_used_at", Columns: []string{"last_used_at"}},
		},
		Comment: "Ansible任务历史表",
	}
}

// ansibleTagsTableSchema ansible_tags 表结构
func ansibleTagsTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_tags",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(50)", Nullable: false, Unique: true},
			{Name: "color", Type: "VARCHAR(20)", Nullable: false, DefaultValue: strPtr("#409EFF")},
			{Name: "description", Type: "VARCHAR(255)", Nullable: true},
			{Name: "user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_tags_name", Columns: []string{"name"}, Unique: true},
			{Name: "idx_ansible_tags_user_id", Columns: []string{"user_id"}},
			{Name: "idx_ansible_tags_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "Ansible标签表",
	}
}

// ansibleTaskTagsTableSchema ansible_task_tags 表结构
func ansibleTaskTagsTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_task_tags",
		Columns: []ColumnDefinition{
			{Name: "task_id", Type: "INTEGER", PrimaryKey: true, Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "ansible_tasks", Column: "id", OnDelete: "CASCADE",
			}},
			{Name: "tag_id", Type: "INTEGER", PrimaryKey: true, Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "ansible_tags", Column: "id", OnDelete: "CASCADE",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_task_tags_task_id", Columns: []string{"task_id"}},
			{Name: "idx_ansible_task_tags_tag_id", Columns: []string{"tag_id"}},
		},
		Comment: "Ansible任务标签关联表",
	}
}

// ansibleWorkflowsTableSchema ansible_workflows 表结构
func ansibleWorkflowsTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_workflows",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "name", Type: "VARCHAR(255)", Nullable: false, Comment: "工作流名称"},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "dag", Type: "JSONB", Nullable: true, Comment: "DAG定义"},
			{Name: "user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "deleted_at", Type: "TIMESTAMP", Nullable: true},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_workflows_user_id", Columns: []string{"user_id"}},
			{Name: "idx_ansible_workflows_deleted_at", Columns: []string{"deleted_at"}},
		},
		Comment: "Ansible工作流表",
	}
}

// ansibleWorkflowExecutionsTableSchema ansible_workflow_executions 表结构
func ansibleWorkflowExecutionsTableSchema() TableSchema {
	return TableSchema{
		Name: "ansible_workflow_executions",
		Columns: []ColumnDefinition{
			{Name: "id", Type: "SERIAL", PrimaryKey: true, AutoIncr: true, Nullable: false},
			{Name: "workflow_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "ansible_workflows", Column: "id", OnDelete: "CASCADE",
			}},
			{Name: "status", Type: "VARCHAR(50)", Nullable: false, DefaultValue: strPtr("running")},
			{Name: "started_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "finished_at", Type: "TIMESTAMP", Nullable: true},
			{Name: "error_message", Type: "TEXT", Nullable: true},
			{Name: "user_id", Type: "INTEGER", Nullable: false, ForeignKey: &ForeignKeyDef{
				Table: "users", Column: "id", OnDelete: "RESTRICT",
			}},
			{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
			{Name: "updated_at", Type: "TIMESTAMP", Nullable: false},
		},
		Indexes: []IndexDefinition{
			{Name: "idx_ansible_workflow_executions_workflow_id", Columns: []string{"workflow_id"}},
			{Name: "idx_ansible_workflow_executions_status", Columns: []string{"status"}},
			{Name: "idx_ansible_workflow_executions_user_id", Columns: []string{"user_id"}},
		},
		Comment: "Ansible工作流执行记录表",
	}
}

// schemaMigrationsTableSchema schema_migrations 表结构
func schemaMigrationsTableSchema() TableSchema {
	return TableSchema{
		Name: "schema_migrations",
		Columns: []ColumnDefinition{
			{Name: "version", Type: "VARCHAR(255)", PrimaryKey: true, Nullable: false},
			{Name: "applied_at", Type: "TIMESTAMP", Nullable: false},
		},
		Comment: "数据库迁移版本记录表",
	}
}

// strPtr 返回字符串指针（用于默认值）
func strPtr(s string) *string {
	return &s
}

// GetTableSchema 根据表名获取表结构定义
func GetTableSchema(tableName string) (*TableSchema, error) {
	for _, schema := range AllTableSchemas() {
		if schema.Name == tableName {
			return &schema, nil
		}
	}
	return nil, fmt.Errorf("table schema not found: %s", tableName)
}

// GetTableNames 返回所有表名列表
func GetTableNames() []string {
	schemas := AllTableSchemas()
	names := make([]string, len(schemas))
	for i, schema := range schemas {
		names[i] = schema.Name
	}
	return names
}

