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

// 数据模型定义
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
	Name        string `gorm:"uniqueIndex;not null"`
	Description string
	KubeConfig  string `gorm:"type:text"`
	Status      string `gorm:"default:active"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

type LabelTemplate struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex;not null"`
	Description string
	Key         string `gorm:"not null"`
	Value       string
	CreatedBy   uint `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TaintTemplate struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex;not null"`
	Description string
	Key         string `gorm:"not null"`
	Value       string
	Effect      string `gorm:"not null"`
	CreatedBy   uint   `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AuditLog struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"index;not null"`
	ClusterID  *uint  `gorm:"index"`
	Action     string `gorm:"not null"`
	Resource   string
	ResourceID string
	Details    string `gorm:"type:text"`
	IPAddress  string
	UserAgent  string
	Status     string `gorm:"default:success"`
	ErrorMsg   string `gorm:"type:text"`
	CreatedAt  time.Time
}

type MigrationStats struct {
	TableName string
	Total     int64
	Success   int64
	Failed    int64
	Skipped   int64
}

func main() {
	if len(os.Args) < 3 || os.Args[1] != "--config" {
		fmt.Println("Usage: go run sqlite-to-postgres-v2.go --config <config-file>")
		os.Exit(1)
	}

	configFile := os.Args[2]
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 初始化数据库连接
	sqliteDB, err := initSQLite(config.SQLite.Path)
	if err != nil {
		log.Fatal("Failed to init SQLite:", err)
	}

	postgresDB, err := initPostgreSQL(config.PostgreSQL)
	if err != nil {
		log.Fatal("Failed to init PostgreSQL:", err)
	}

	// 执行迁移
	log.Println("开始数据迁移...")

	var allStats []MigrationStats

	// 按顺序迁移表（考虑外键依赖）
	allStats = append(allStats, migrateUsersV2(sqliteDB, postgresDB))
	allStats = append(allStats, migrateClustersV2(sqliteDB, postgresDB))
	allStats = append(allStats, migrateLabelTemplatesV2(sqliteDB, postgresDB))
	allStats = append(allStats, migrateTaintTemplatesV2(sqliteDB, postgresDB))
	allStats = append(allStats, migrateAuditLogsV2(sqliteDB, postgresDB))

	printMigrationSummary(allStats)
	log.Println("Migration completed!")
}

func loadConfig(configFile string) (Config, error) {
	var config Config

	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return config, err
	}

	if err := viper.Unmarshal(&config); err != nil {
		return config, err
	}

	return config, nil
}

func initSQLite(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	return db, err
}

func initPostgreSQL(config PostgreSQLConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s password=%s",
		config.Host, config.Port, config.Username, config.Database, config.SSLMode, config.Password)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		return nil, err
	}

	// 自动迁移表结构
	err = db.AutoMigrate(&User{}, &Cluster{}, &LabelTemplate{}, &TaintTemplate{}, &AuditLog{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %v", err)
	}

	return db, nil
}

// 改进的用户迁移函数
func migrateUsersV2(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating users...")
	stats := MigrationStats{TableName: "users"}

	src.Model(&User{}).Count(&stats.Total)

	var users []User
	result := src.Unscoped().Find(&users)
	if result.Error != nil {
		log.Printf("Error reading users from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	for _, user := range users {
		// 检查用户是否已存在
		var existing User
		if err := dst.Where("username = ? OR email = ?", user.Username, user.Email).First(&existing).Error; err == nil {
			log.Printf("User %s already exists (ID: %d), skipping", user.Username, existing.ID)
			stats.Skipped++
			continue
		}

		// 保存原始ID，重置为0让PostgreSQL自动生成
		originalID := user.ID
		user.ID = 0

		if err := dst.Create(&user).Error; err != nil {
			log.Printf("Error inserting user %s (original ID: %d): %v", user.Username, originalID, err)
			stats.Failed++
		} else {
			log.Printf("Successfully inserted user %s (original ID: %d, new ID: %d)", user.Username, originalID, user.ID)
			stats.Success++
		}
	}

	log.Printf("Users migration completed: %d success, %d failed, %d skipped", stats.Success, stats.Failed, stats.Skipped)
	return stats
}

// 改进的集群迁移函数
func migrateClustersV2(src, dst *gorm.DB) MigrationStats {
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
		// 检查集群是否已存在
		var existing Cluster
		if err := dst.Where("name = ?", cluster.Name).First(&existing).Error; err == nil {
			log.Printf("Cluster %s already exists (ID: %d), skipping", cluster.Name, existing.ID)
			stats.Skipped++
			continue
		}

		originalID := cluster.ID
		cluster.ID = 0

		if err := dst.Create(&cluster).Error; err != nil {
			log.Printf("Error inserting cluster %s (original ID: %d): %v", cluster.Name, originalID, err)
			stats.Failed++
		} else {
			log.Printf("Successfully inserted cluster %s (original ID: %d, new ID: %d)", cluster.Name, originalID, cluster.ID)
			stats.Success++
		}
	}

	log.Printf("Clusters migration completed: %d success, %d failed, %d skipped", stats.Success, stats.Failed, stats.Skipped)
	return stats
}

// 改进的标签模板迁移函数
func migrateLabelTemplatesV2(src, dst *gorm.DB) MigrationStats {
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
		var existing LabelTemplate
		if err := dst.Where("name = ?", template.Name).First(&existing).Error; err == nil {
			log.Printf("Label template %s already exists (ID: %d), skipping", template.Name, existing.ID)
			stats.Skipped++
			continue
		}

		originalID := template.ID
		template.ID = 0

		if err := dst.Create(&template).Error; err != nil {
			log.Printf("Error inserting label template %s (original ID: %d): %v", template.Name, originalID, err)
			stats.Failed++
		} else {
			log.Printf("Successfully inserted label template %s (original ID: %d, new ID: %d)", template.Name, originalID, template.ID)
			stats.Success++
		}
	}

	log.Printf("Label templates migration completed: %d success, %d failed, %d skipped", stats.Success, stats.Failed, stats.Skipped)
	return stats
}

// 改进的污点模板迁移函数
func migrateTaintTemplatesV2(src, dst *gorm.DB) MigrationStats {
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
		var existing TaintTemplate
		if err := dst.Where("name = ?", template.Name).First(&existing).Error; err == nil {
			log.Printf("Taint template %s already exists (ID: %d), skipping", template.Name, existing.ID)
			stats.Skipped++
			continue
		}

		originalID := template.ID
		template.ID = 0

		if err := dst.Create(&template).Error; err != nil {
			log.Printf("Error inserting taint template %s (original ID: %d): %v", template.Name, originalID, err)
			stats.Failed++
		} else {
			log.Printf("Successfully inserted taint template %s (original ID: %d, new ID: %d)", template.Name, originalID, template.ID)
			stats.Success++
		}
	}

	log.Printf("Taint templates migration completed: %d success, %d failed, %d skipped", stats.Success, stats.Failed, stats.Skipped)
	return stats
}

// 改进的审计日志迁移函数 - 只迁移有效的外键引用
func migrateAuditLogsV2(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating audit logs...")
	stats := MigrationStats{TableName: "audit_logs"}

	src.Model(&AuditLog{}).Count(&stats.Total)

	// 获取PostgreSQL中存在的用户和集群ID
	var userIDs []uint
	dst.Model(&User{}).Pluck("id", &userIDs)
	userIDMap := make(map[uint]bool)
	for _, id := range userIDs {
		userIDMap[id] = true
	}

	var clusterIDs []uint
	dst.Model(&Cluster{}).Pluck("id", &clusterIDs)
	clusterIDMap := make(map[uint]bool)
	for _, id := range clusterIDs {
		clusterIDMap[id] = true
	}

	log.Printf("Found %d users and %d clusters in PostgreSQL", len(userIDs), len(clusterIDs))

	var auditLogs []AuditLog
	result := src.Unscoped().Find(&auditLogs)
	if result.Error != nil {
		log.Printf("Error reading audit logs from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	batchSize := 100
	for i := 0; i < len(auditLogs); i += batchSize {
		end := i + batchSize
		if end > len(auditLogs) {
			end = len(auditLogs)
		}

		batch := auditLogs[i:end]
		for _, auditLog := range batch {
			// 检查外键约束
			if !userIDMap[auditLog.UserID] {
				log.Printf("Skipping audit log %d: user ID %d not found", i+1, auditLog.UserID)
				stats.Skipped++
				continue
			}

			if auditLog.ClusterID != nil && !clusterIDMap[*auditLog.ClusterID] {
				log.Printf("Skipping audit log %d: cluster ID %d not found", i+1, *auditLog.ClusterID)
				stats.Skipped++
				continue
			}

			auditLog.ID = 0
			if err := dst.Create(&auditLog).Error; err != nil {
				log.Printf("Error inserting audit log %d: %v", i+1, err)
				stats.Failed++
			} else {
				stats.Success++
			}
		}

		if i%1000 == 0 {
			log.Printf("Processed %d/%d audit logs...", i, len(auditLogs))
		}
	}

	log.Printf("Audit logs migration completed: %d success, %d failed, %d skipped", stats.Success, stats.Failed, stats.Skipped)
	return stats
}

func printMigrationSummary(allStats []MigrationStats) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("MIGRATION SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("%-20s %8s %8s %8s %8s\n", "TABLE", "TOTAL", "SUCCESS", "FAILED", "SKIPPED")
	fmt.Println(strings.Repeat("-", 60))

	var totalAll, successAll, failedAll, skippedAll int64

	for _, stat := range allStats {
		fmt.Printf("%-20s %8d %8d %8d %8d\n",
			stat.TableName, stat.Total, stat.Success, stat.Failed, stat.Skipped)
		totalAll += stat.Total
		successAll += stat.Success
		failedAll += stat.Failed
		skippedAll += stat.Skipped
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-20s %8d %8d %8d %8d\n",
		"TOTAL", totalAll, successAll, failedAll, skippedAll)
	fmt.Println(strings.Repeat("=", 60))

	if failedAll > 0 {
		fmt.Printf("⚠️  Migration completed with %d errors\n", failedAll)
	} else {
		fmt.Println("✅ Migration completed successfully!")
	}
	fmt.Println()
}
