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
	Type         string `mapstructure:"type"` // sqlite, postgres
	DSN          string `mapstructure:"dsn"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Database     string `mapstructure:"database"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  int    `mapstructure:"max_lifetime"` // seconds
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
	EnableDatabase bool `mapstructure:"enable_database"` // 启用数据库模式用于多副本支持
}

type MonitoringConfig struct {
	Enabled  bool `mapstructure:"enabled"`  // 启用节点异常监控
	Interval int  `mapstructure:"interval"` // 监控周期（秒）
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
	viper.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	viper.SetDefault("jwt.expire_time", 86400)
	viper.SetDefault("ldap.enabled", false)
	viper.SetDefault("ldap.port", 389)
	viper.SetDefault("progress.enable_database", false)
	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.interval", 60)

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
