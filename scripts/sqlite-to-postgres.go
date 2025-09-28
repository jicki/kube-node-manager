package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 配置结构
type Config struct {
	SQLite     SQLiteConfig     `mapstructure:"sqlite"`
	PostgreSQL PostgreSQLConfig `mapstructure:"postgresql"`
}

type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

type PostgreSQLConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// 数据模型定义（简化版，只包含必要字段）
type User struct {
	ID         uint   `gorm:"primaryKey"`
	Username   string `gorm:"uniqueIndex;not null"`
	Email      string `gorm:"uniqueIndex;not null"`
	Password   string `gorm:"not null"`
	Role       string `gorm:"default:user"`
	Status     string `gorm:"default:active"`
	IsLDAPUser bool   `gorm:"default:false"`
	LastLogin  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time `gorm:"index"`
}

type Cluster struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null;uniqueIndex"`
	Description string
	KubeConfig  string `gorm:"type:text;not null"`
	Status      string `gorm:"default:active"`
	Version     string
	NodeCount   int `gorm:"default:0"`
	LastSync    *time.Time
	CreatedBy   uint `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

type LabelTemplate struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string
	Labels      string `gorm:"type:text;not null"`
	CreatedBy   uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

type TaintTemplate struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string
	Taints      string `gorm:"type:text;not null"`
	CreatedBy   uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

type AuditLog struct {
	ID           uint `gorm:"primaryKey"`
	UserID       uint `gorm:"not null"`
	ClusterID    uint
	NodeName     string
	Action       string `gorm:"not null"`
	ResourceType string `gorm:"not null"`
	Details      string `gorm:"type:text"`
	Reason       string `gorm:"type:text"`
	Status       string `gorm:"default:success"`
	ErrorMsg     string
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
}

// 迁移统计信息
type MigrationStats struct {
	TableName string
	Total     int64
	Success   int64
	Failed    int64
	Skipped   int64
}

func main() {
	log.Println("Starting SQLite to PostgreSQL migration...")

	// 加载配置
	config, err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 连接数据库
	sqliteDB, err := connectSQLite(config.SQLite.Path)
	if err != nil {
		log.Fatal("Failed to connect to SQLite:", err)
	}

	postgresDB, err := connectPostgreSQL(config.PostgreSQL)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	// 创建表结构
	if err := createTables(postgresDB); err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	// 执行迁移
	stats := []MigrationStats{}

	// 迁移用户表
	userStats := migrateUsers(sqliteDB, postgresDB)
	stats = append(stats, userStats)

	// 迁移集群表
	clusterStats := migrateClusters(sqliteDB, postgresDB)
	stats = append(stats, clusterStats)

	// 迁移标签模板表
	labelTemplateStats := migrateLabelTemplates(sqliteDB, postgresDB)
	stats = append(stats, labelTemplateStats)

	// 迁移污点模板表
	taintTemplateStats := migrateTaintTemplates(sqliteDB, postgresDB)
	stats = append(stats, taintTemplateStats)

	// 迁移审计日志表
	auditLogStats := migrateAuditLogs(sqliteDB, postgresDB)
	stats = append(stats, auditLogStats)

	// 打印统计信息
	printMigrationSummary(stats)

	log.Println("Migration completed!")
}

// loadConfig 加载配置
func loadConfig() (*Config, error) {
	viper.SetConfigName("migration")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./scripts")

	// 设置默认值
	viper.SetDefault("sqlite.path", "./backend/data/kube-node-manager.db")
	viper.SetDefault("postgresql.host", "localhost")
	viper.SetDefault("postgresql.port", 5432)
	viper.SetDefault("postgresql.username", "postgres")
	viper.SetDefault("postgresql.password", "")
	viper.SetDefault("postgresql.database", "kube_node_manager")
	viper.SetDefault("postgresql.ssl_mode", "disable")

	// 支持环境变量
	viper.AutomaticEnv()
	viper.SetEnvPrefix("MIGRATION")

	// 尝试读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found, using defaults and environment variables")
		} else {
			return nil, fmt.Errorf("error reading config file: %v", err)
		}
	} else {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %v", err)
	}

	return &config, nil
}

// connectSQLite 连接到 SQLite 数据库
func connectSQLite(path string) (*gorm.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("SQLite database file does not exist: %s", path)
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to SQLite database: %s", path)
	return db, nil
}

// connectPostgreSQL 连接到 PostgreSQL 数据库
func connectPostgreSQL(config PostgreSQLConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Database, config.SSLMode)

	if config.Password != "" {
		dsn += fmt.Sprintf(" password=%s", config.Password)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %v", err)
	}

	log.Printf("Connected to PostgreSQL database: %s:%d/%s", config.Host, config.Port, config.Database)
	return db, nil
}

// createTables 在 PostgreSQL 中创建表结构
func createTables(db *gorm.DB) error {
	log.Println("Creating tables in PostgreSQL...")

	// 按顺序创建表（考虑外键依赖）
	tables := []interface{}{
		&User{},
		&Cluster{},
		&LabelTemplate{},
		&TaintTemplate{},
		&AuditLog{},
	}

	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			return fmt.Errorf("failed to migrate table %T: %v", table, err)
		}
	}

	log.Println("Tables created successfully")
	return nil
}

// migrateUsers 迁移用户表
func migrateUsers(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating users...")
	stats := MigrationStats{TableName: "users"}

	// 获取总数
	src.Model(&User{}).Count(&stats.Total)

	var users []User
	result := src.Unscoped().Find(&users)
	if result.Error != nil {
		log.Printf("Error reading users from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	// 批量插入
	batchSize := 100
	for i := 0; i < len(users); i += batchSize {
		end := i + batchSize
		if end > len(users) {
			end = len(users)
		}

		batch := users[i:end]
		for j := range batch {
			result := dst.Create(&batch[j])
			if result.Error != nil {
				log.Printf("Error inserting user %s: %v", batch[j].Username, result.Error)
				stats.Failed++
			} else {
				stats.Success++
			}
		}
	}

	log.Printf("Users migration completed: %d success, %d failed", stats.Success, stats.Failed)
	return stats
}

// migrateClusters 迁移集群表
func migrateClusters(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating clusters...")
	stats := MigrationStats{TableName: "clusters"}

	src.Model(&Cluster{}).Count(&stats.Total)

	var clusters []Cluster
	result := src.Unscoped().Find(&clusters)
	if result.Error != nil {
		log.Printf("Error reading clusters from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	for _, cluster := range clusters {
		result := dst.Create(&cluster)
		if result.Error != nil {
			log.Printf("Error inserting cluster %s: %v", cluster.Name, result.Error)
			stats.Failed++
		} else {
			stats.Success++
		}
	}

	log.Printf("Clusters migration completed: %d success, %d failed", stats.Success, stats.Failed)
	return stats
}

// migrateLabelTemplates 迁移标签模板表
func migrateLabelTemplates(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating label templates...")
	stats := MigrationStats{TableName: "label_templates"}

	src.Model(&LabelTemplate{}).Count(&stats.Total)

	var templates []LabelTemplate
	result := src.Unscoped().Find(&templates)
	if result.Error != nil {
		log.Printf("Error reading label templates from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	for _, template := range templates {
		result := dst.Create(&template)
		if result.Error != nil {
			log.Printf("Error inserting label template %s: %v", template.Name, result.Error)
			stats.Failed++
		} else {
			stats.Success++
		}
	}

	log.Printf("Label templates migration completed: %d success, %d failed", stats.Success, stats.Failed)
	return stats
}

// migrateTaintTemplates 迁移污点模板表
func migrateTaintTemplates(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating taint templates...")
	stats := MigrationStats{TableName: "taint_templates"}

	src.Model(&TaintTemplate{}).Count(&stats.Total)

	var templates []TaintTemplate
	result := src.Unscoped().Find(&templates)
	if result.Error != nil {
		log.Printf("Error reading taint templates from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	for _, template := range templates {
		result := dst.Create(&template)
		if result.Error != nil {
			log.Printf("Error inserting taint template %s: %v", template.Name, result.Error)
			stats.Failed++
		} else {
			stats.Success++
		}
	}

	log.Printf("Taint templates migration completed: %d success, %d failed", stats.Success, stats.Failed)
	return stats
}

// migrateAuditLogs 迁移审计日志表
func migrateAuditLogs(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating audit logs...")
	stats := MigrationStats{TableName: "audit_logs"}

	src.Model(&AuditLog{}).Count(&stats.Total)

	var logs []AuditLog
	result := src.Unscoped().Find(&logs)
	if result.Error != nil {
		log.Printf("Error reading audit logs from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	// 审计日志可能很多，使用批量插入
	batchSize := 500
	for i := 0; i < len(logs); i += batchSize {
		end := i + batchSize
		if end > len(logs) {
			end = len(logs)
		}

		batch := logs[i:end]
		for j := range batch {
			result := dst.Create(&batch[j])
			if result.Error != nil {
				log.Printf("Error inserting audit log %d: %v", batch[j].ID, result.Error)
				stats.Failed++
			} else {
				stats.Success++
			}
		}

		// 显示进度
		if i%1000 == 0 {
			log.Printf("Processed %d/%d audit logs...", i, len(logs))
		}
	}

	log.Printf("Audit logs migration completed: %d success, %d failed", stats.Success, stats.Failed)
	return stats
}

// printMigrationSummary 打印迁移摘要
func printMigrationSummary(stats []MigrationStats) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("MIGRATION SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("%-20s %10s %10s %10s %10s\n", "TABLE", "TOTAL", "SUCCESS", "FAILED", "SKIPPED")
	fmt.Println(strings.Repeat("-", 60))

	totalRecords := int64(0)
	totalSuccess := int64(0)
	totalFailed := int64(0)
	totalSkipped := int64(0)

	for _, stat := range stats {
		fmt.Printf("%-20s %10d %10d %10d %10d\n",
			stat.TableName, stat.Total, stat.Success, stat.Failed, stat.Skipped)
		totalRecords += stat.Total
		totalSuccess += stat.Success
		totalFailed += stat.Failed
		totalSkipped += stat.Skipped
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-20s %10d %10d %10d %10d\n",
		"TOTAL", totalRecords, totalSuccess, totalFailed, totalSkipped)
	fmt.Println(strings.Repeat("=", 60))

	if totalFailed > 0 {
		fmt.Printf("⚠️  Migration completed with %d errors\n", totalFailed)
	} else {
		fmt.Println("✅ Migration completed successfully!")
	}
	fmt.Println()
}
