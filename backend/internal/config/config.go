package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	LDAP     LDAPConfig     `mapstructure:"ldap"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	DSN string `mapstructure:"dsn"`
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

func LoadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.dsn", "./data/kube-node-manager.db")
	viper.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	viper.SetDefault("jwt.expire_time", 86400)
	viper.SetDefault("ldap.enabled", false)
	viper.SetDefault("ldap.port", 389)
	
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
	
	return &config
}