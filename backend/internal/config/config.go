package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	LDAP       LDAPConfig       `mapstructure:"ldap"`
	Progress   ProgressConfig   `mapstructure:"progress"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Type              string `mapstructure:"type"` // sqlite, postgres
	DSN               string `mapstructure:"dsn"`
	Host              string `mapstructure:"host"`
	Port              int    `mapstructure:"port"`
	Database          string `mapstructure:"database"`
	Username          string `mapstructure:"username"`
	Password          string `mapstructure:"password"`
	SSLMode           string `mapstructure:"ssl_mode"`
	MaxOpenConns      int    `mapstructure:"max_open_conns"`
	MaxIdleConns      int    `mapstructure:"max_idle_conns"`
	MaxLifetime       int    `mapstructure:"max_lifetime"`        // seconds
	AutoMigrate       bool   `mapstructure:"auto_migrate"`        // 启动时自动迁移，默认 true
	ValidateOnStartup bool   `mapstructure:"validate_on_startup"` // 启动时验证结构，默认 true
	RepairOnStartup   bool   `mapstructure:"repair_on_startup"`   // 启动时自动修复，默认 true
	MigrationTimeout  int    `mapstructure:"migration_timeout"`   // 迁移超时（秒），0 表示不限制，默认 300
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	ExpireTime int    `mapstructure:"expire_time"`
}

type LDAPConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	BaseDN     string `mapstructure:"base_dn"`
	UserFilter string `mapstructure:"user_filter"`
	AdminDN    string `mapstructure:"admin_dn"`
	AdminPass  string `mapstructure:"admin_pass"`
}

type ProgressConfig struct {
	EnableDatabase bool          `mapstructure:"enable_database"` // 启用数据库模式用于多副本支持
	NotifyType     string        `mapstructure:"notify_type"`     // 通知方式：polling, postgres, redis
	Redis          RedisConfig   `mapstructure:"redis"`           // Redis 配置
	PollInterval   int           `mapstructure:"poll_interval"`   // 轮询间隔（毫秒），仅 polling 模式使用
}

type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`  // 启用 Redis
	Addr     string `mapstructure:"addr"`     // Redis 地址 (host:port)
	Password string `mapstructure:"password"` // Redis 密码
	DB       int    `mapstructure:"db"`       // Redis 数据库编号
}

type MonitoringConfig struct {
	Enabled                bool          `mapstructure:"enabled"`                  // 启用节点异常监控
	Interval               int           `mapstructure:"interval"`                 // 监控周期（秒）
	ReportSchedulerEnabled bool          `mapstructure:"report_scheduler_enabled"` // 启用报告调度器
	Cache                  CacheConfig   `mapstructure:"cache"`                    // 缓存配置
	Cleanup                CleanupConfig `mapstructure:"cleanup"`                  // 清理配置
}

type CleanupConfig struct {
	Enabled       bool   `mapstructure:"enabled"`        // 是否启用自动清理
	RetentionDays int    `mapstructure:"retention_days"` // 保留天数
	CleanupTime   string `mapstructure:"cleanup_time"`   // 清理时间（HH:MM）
	BatchSize     int    `mapstructure:"batch_size"`     // 批量删除大小
}

type CacheConfig struct {
	Enabled  bool                `mapstructure:"enabled"`  // 启用缓存
	Type     string              `mapstructure:"type"`     // 缓存类型：postgres, memory, none
	Postgres PostgresCacheConfig `mapstructure:"postgres"` // PostgreSQL 缓存配置
	TTL      CacheTTLConfig      `mapstructure:"ttl"`      // 缓存 TTL 配置
}

type PostgresCacheConfig struct {
	TableName       string `mapstructure:"table_name"`       // 缓存表名
	CleanupInterval int    `mapstructure:"cleanup_interval"` // 清理周期（秒）
	UseUnlogged     bool   `mapstructure:"use_unlogged"`     // 使用 UNLOGGED 表
}

type CacheTTLConfig struct {
	Statistics int `mapstructure:"statistics"` // 统计数据缓存 TTL（秒）
	Active     int `mapstructure:"active"`     // 活跃异常缓存 TTL（秒）
	Clusters   int `mapstructure:"clusters"`   // 集群列表缓存 TTL（秒）
	TypeStats  int `mapstructure:"type_stats"` // 类型统计缓存 TTL（秒）
}

func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.type", "sqlite")
	// 不设置 database.dsn 的默认值，让它根据数据库类型动态确定
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "kube_node_manager")
	viper.SetDefault("database.username", "postgres")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_lifetime", 3600)
	viper.SetDefault("database.auto_migrate", true)
	viper.SetDefault("database.validate_on_startup", true)
	viper.SetDefault("database.repair_on_startup", true)
	viper.SetDefault("database.migration_timeout", 300)
	viper.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	viper.SetDefault("jwt.expire_time", 86400)
	viper.SetDefault("ldap.enabled", false)
	viper.SetDefault("ldap.port", 389)
	viper.SetDefault("progress.enable_database", false)
	viper.SetDefault("progress.notify_type", "polling") // polling, postgres, redis
	viper.SetDefault("progress.poll_interval", 500)     // 500ms
	viper.SetDefault("progress.redis.enabled", false)
	viper.SetDefault("progress.redis.addr", "localhost:6379")
	viper.SetDefault("progress.redis.password", "")
	viper.SetDefault("progress.redis.db", 0)
	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.interval", 60)
	viper.SetDefault("monitoring.report_scheduler_enabled", true)
	viper.SetDefault("monitoring.cache.enabled", true)
	viper.SetDefault("monitoring.cache.type", "postgres")
	viper.SetDefault("monitoring.cache.postgres.table_name", "cache_entries")
	viper.SetDefault("monitoring.cache.postgres.cleanup_interval", 300)
	viper.SetDefault("monitoring.cache.postgres.use_unlogged", true)
	viper.SetDefault("monitoring.cache.ttl.statistics", 300)
	viper.SetDefault("monitoring.cache.ttl.active", 30)
	viper.SetDefault("monitoring.cache.ttl.clusters", 600)
	viper.SetDefault("monitoring.cache.ttl.type_stats", 300)
	viper.SetDefault("monitoring.cleanup.enabled", true)
	viper.SetDefault("monitoring.cleanup.retention_days", 90)
	viper.SetDefault("monitoring.cleanup.cleanup_time", "02:00")
	viper.SetDefault("monitoring.cleanup.batch_size", 1000)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using defaults and environment variables")
		} else {
			log.Printf("Error reading config file: %v", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal("Unable to decode config:", err)
	}

	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}

	// 根据数据库类型设置合适的默认 DSN（如果 DSN 为空）
	if config.Database.DSN == "" {
		switch config.Database.Type {
		case "sqlite":
			config.Database.DSN = "./data/kube-node-manager.db"
		case "postgres", "postgresql":
			// PostgreSQL 不设置默认 DSN，让系统从单独的参数构建
			config.Database.DSN = ""
		}
	}

	return &config
}
