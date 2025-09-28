package main

import (
	"encoding/json"
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

// ===== 正确的数据模型定义 (与后端一致) =====

type User struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	Username   string         `json:"username" gorm:"uniqueIndex;not null"`
	Email      string         `json:"email" gorm:"uniqueIndex;not null"`
	Password   string         `json:"-" gorm:"not null"`
	Role       string         `json:"role" gorm:"default:user"`
	Status     string         `json:"status" gorm:"default:active"`
	IsLDAPUser bool           `json:"is_ldap_user" gorm:"default:false"`
	LastLogin  *time.Time     `json:"last_login"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

type Cluster struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null;uniqueIndex"`
	Description string         `json:"description"`
	KubeConfig  string         `json:"kube_config" gorm:"type:text;not null"`
	Status      string         `json:"status" gorm:"default:active"`
	Version     string         `json:"version"`
	NodeCount   int            `json:"node_count" gorm:"default:0"`
	LastSync    *time.Time     `json:"last_sync"`
	CreatedBy   uint           `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// 正确的标签模板结构 - 使用JSON字段
type LabelTemplate struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	Labels      string         `json:"labels" gorm:"type:text;not null"` // JSON格式存储键值对
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// 正确的污点模板结构 - 使用JSON数组字段
type TaintTemplate struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	Taints      string         `json:"taints" gorm:"type:text;not null"` // JSON格式存储污点信息
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type AuditLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"not null"`
	ClusterID    uint      `json:"cluster_id"`
	NodeName     string    `json:"node_name"`
	Action       string    `json:"action" gorm:"not null"`
	ResourceType string    `json:"resource_type" gorm:"not null"`
	Details      string    `json:"details" gorm:"type:text"`
	Reason       string    `json:"reason" gorm:"type:text"`
	Status       string    `json:"status" gorm:"default:success"`
	ErrorMsg     string    `json:"error_msg"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
}

// ===== SQLite旧版本模型定义 (用于读取) =====

// SQLite中的旧版标签模板结构
type SQLiteLabelTemplate struct {
	ID          uint       `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	Description string     `gorm:"column:description"`
	Key         string     `gorm:"column:key"`    // SQLite中可能是分离的key字段
	Value       string     `gorm:"column:value"`  // SQLite中可能是分离的value字段
	Labels      string     `gorm:"column:labels"` // SQLite中可能已有JSON字段
	CreatedBy   uint       `gorm:"column:created_by"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

// SQLite中的旧版污点模板结构
type SQLiteTaintTemplate struct {
	ID          uint       `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	Description string     `gorm:"column:description"`
	Key         string     `gorm:"column:key"`    // SQLite中可能是分离的key字段
	Value       string     `gorm:"column:value"`  // SQLite中可能是分离的value字段
	Effect      string     `gorm:"column:effect"` // SQLite中可能是分离的effect字段
	Taints      string     `gorm:"column:taints"` // SQLite中可能已有JSON数组字段
	CreatedBy   uint       `gorm:"column:created_by"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

// SQLite中的旧版审计日志结构
type SQLiteAuditLog struct {
	ID         uint      `gorm:"column:id"`
	UserID     uint      `gorm:"column:user_id"`
	ClusterID  *uint     `gorm:"column:cluster_id"`
	Action     string    `gorm:"column:action"`
	Resource   string    `gorm:"column:resource"`    // SQLite中可能是旧版resource字段
	ResourceID string    `gorm:"column:resource_id"` // SQLite中可能是旧版resource_id字段
	Details    string    `gorm:"column:details"`
	IPAddress  string    `gorm:"column:ip_address"`
	UserAgent  string    `gorm:"column:user_agent"`
	Status     string    `gorm:"column:status"`
	ErrorMsg   string    `gorm:"column:error_msg"`
	CreatedAt  time.Time `gorm:"column:created_at"`
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
		fmt.Println("Usage: go run sqlite-to-postgres-v3.go --config <config-file>")
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
	log.Println("开始数据迁移 (v3 - 修复版本)...")

	var allStats []MigrationStats

	// 按顺序迁移表（考虑外键依赖）
	allStats = append(allStats, migrateUsers(sqliteDB, postgresDB))
	allStats = append(allStats, migrateClusters(sqliteDB, postgresDB))
	allStats = append(allStats, migrateLabelTemplatesV3(sqliteDB, postgresDB))
	allStats = append(allStats, migrateTaintTemplatesV3(sqliteDB, postgresDB))
	allStats = append(allStats, migrateAuditLogsV3(sqliteDB, postgresDB))

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

	// 使用正确的模型自动迁移表结构
	err = db.AutoMigrate(&User{}, &Cluster{}, &LabelTemplate{}, &TaintTemplate{}, &AuditLog{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %v", err)
	}

	return db, nil
}

func migrateUsers(src, dst *gorm.DB) MigrationStats {
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
		var existing User
		if err := dst.Where("username = ? OR email = ?", user.Username, user.Email).First(&existing).Error; err == nil {
			log.Printf("User %s already exists (ID: %d), skipping", user.Username, existing.ID)
			stats.Skipped++
			continue
		}

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

	// 获取默认的admin用户ID用于created_by字段
	var adminUser User
	if err := dst.Where("role = ?", "admin").First(&adminUser).Error; err != nil {
		log.Printf("Warning: No admin user found, using ID 1 as default")
		adminUser.ID = 1
	}

	for _, cluster := range clusters {
		var existing Cluster
		if err := dst.Where("name = ?", cluster.Name).First(&existing).Error; err == nil {
			log.Printf("Cluster %s already exists (ID: %d), skipping", cluster.Name, existing.ID)
			stats.Skipped++
			continue
		}

		originalID := cluster.ID
		cluster.ID = 0

		// 确保created_by字段有值
		if cluster.CreatedBy == 0 {
			cluster.CreatedBy = adminUser.ID
		}

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

// 修复版本的标签模板迁移函数
func migrateLabelTemplatesV3(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating label templates (v3 - 修复版本)...")
	stats := MigrationStats{TableName: "label_templates"}

	src.Model(&SQLiteLabelTemplate{}).Count(&stats.Total)

	var sqliteTemplates []SQLiteLabelTemplate
	result := src.Unscoped().Table("label_templates").Find(&sqliteTemplates)
	if result.Error != nil {
		log.Printf("Error reading label templates from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	for _, sqliteTemplate := range sqliteTemplates {
		var existing LabelTemplate
		if err := dst.Where("name = ?", sqliteTemplate.Name).First(&existing).Error; err == nil {
			log.Printf("Label template %s already exists (ID: %d), skipping", sqliteTemplate.Name, existing.ID)
			stats.Skipped++
			continue
		}

		// 创建新的标签模板
		template := LabelTemplate{
			Name:        sqliteTemplate.Name,
			Description: sqliteTemplate.Description,
			CreatedBy:   sqliteTemplate.CreatedBy,
			CreatedAt:   sqliteTemplate.CreatedAt,
			UpdatedAt:   sqliteTemplate.UpdatedAt,
		}

		// 智能处理Labels字段
		if sqliteTemplate.Labels != "" {
			// SQLite中已有JSON格式的Labels字段，直接使用
			template.Labels = sqliteTemplate.Labels
			log.Printf("Using existing JSON labels for template %s: %s", template.Name, template.Labels)
		} else if sqliteTemplate.Key != "" {
			// SQLite中只有分离的Key/Value字段，转换为JSON
			labelsMap := map[string]string{
				sqliteTemplate.Key: sqliteTemplate.Value,
			}
			labelsJSON, err := json.Marshal(labelsMap)
			if err != nil {
				log.Printf("Error marshaling labels for template %s: %v", template.Name, err)
				template.Labels = "{}"
			} else {
				template.Labels = string(labelsJSON)
			}
			log.Printf("Converted key/value to JSON for template %s: %s=%s -> %s",
				template.Name, sqliteTemplate.Key, sqliteTemplate.Value, template.Labels)
		} else {
			// 没有任何标签数据，使用空JSON
			template.Labels = "{}"
			log.Printf("No label data found for template %s, using empty JSON", template.Name)
		}

		if err := dst.Create(&template).Error; err != nil {
			log.Printf("Error inserting label template %s: %v", template.Name, err)
			stats.Failed++
		} else {
			log.Printf("Successfully inserted label template %s (new ID: %d)", template.Name, template.ID)
			stats.Success++
		}
	}

	log.Printf("Label templates migration completed: %d success, %d failed, %d skipped", stats.Success, stats.Failed, stats.Skipped)
	return stats
}

// 修复版本的污点模板迁移函数
func migrateTaintTemplatesV3(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating taint templates (v3 - 修复版本)...")
	stats := MigrationStats{TableName: "taint_templates"}

	src.Model(&SQLiteTaintTemplate{}).Count(&stats.Total)

	var sqliteTemplates []SQLiteTaintTemplate
	result := src.Unscoped().Table("taint_templates").Find(&sqliteTemplates)
	if result.Error != nil {
		log.Printf("Error reading taint templates from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	for _, sqliteTemplate := range sqliteTemplates {
		var existing TaintTemplate
		if err := dst.Where("name = ?", sqliteTemplate.Name).First(&existing).Error; err == nil {
			log.Printf("Taint template %s already exists (ID: %d), skipping", sqliteTemplate.Name, existing.ID)
			stats.Skipped++
			continue
		}

		// 创建新的污点模板
		template := TaintTemplate{
			Name:        sqliteTemplate.Name,
			Description: sqliteTemplate.Description,
			CreatedBy:   sqliteTemplate.CreatedBy,
			CreatedAt:   sqliteTemplate.CreatedAt,
			UpdatedAt:   sqliteTemplate.UpdatedAt,
		}

		// 智能处理Taints字段
		if sqliteTemplate.Taints != "" {
			// SQLite中已有JSON数组格式的Taints字段，直接使用
			template.Taints = sqliteTemplate.Taints
			log.Printf("Using existing JSON taints for template %s: %s", template.Name, template.Taints)
		} else if sqliteTemplate.Key != "" {
			// SQLite中只有分离的Key/Value/Effect字段，转换为JSON数组
			taint := map[string]string{
				"key":    sqliteTemplate.Key,
				"value":  sqliteTemplate.Value,
				"effect": sqliteTemplate.Effect,
			}
			taintsArray := []map[string]string{taint}
			taintsJSON, err := json.Marshal(taintsArray)
			if err != nil {
				log.Printf("Error marshaling taints for template %s: %v", template.Name, err)
				template.Taints = "[]"
			} else {
				template.Taints = string(taintsJSON)
			}
			log.Printf("Converted key/value/effect to JSON for template %s: %s=%s:%s -> %s",
				template.Name, sqliteTemplate.Key, sqliteTemplate.Value, sqliteTemplate.Effect, template.Taints)
		} else {
			// 没有任何污点数据，使用空JSON数组
			template.Taints = "[]"
			log.Printf("No taint data found for template %s, using empty JSON array", template.Name)
		}

		if err := dst.Create(&template).Error; err != nil {
			log.Printf("Error inserting taint template %s: %v", template.Name, err)
			stats.Failed++
		} else {
			log.Printf("Successfully inserted taint template %s (new ID: %d)", template.Name, template.ID)
			stats.Success++
		}
	}

	log.Printf("Taint templates migration completed: %d success, %d failed, %d skipped", stats.Success, stats.Failed, stats.Skipped)
	return stats
}

// 修复版本的审计日志迁移函数
func migrateAuditLogsV3(src, dst *gorm.DB) MigrationStats {
	log.Println("Migrating audit logs (v3 - 修复版本)...")
	stats := MigrationStats{TableName: "audit_logs"}

	src.Model(&SQLiteAuditLog{}).Count(&stats.Total)

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

	// 获取默认的admin用户和集群ID
	var adminUser User
	if err := dst.Where("role = ?", "admin").First(&adminUser).Error; err != nil {
		adminUser.ID = 1
	}

	var defaultCluster Cluster
	if err := dst.First(&defaultCluster).Error; err != nil {
		defaultCluster.ID = 1
	}

	log.Printf("Found %d users and %d clusters in PostgreSQL", len(userIDs), len(clusterIDs))

	var sqliteAuditLogs []SQLiteAuditLog
	result := src.Unscoped().Table("audit_logs").Find(&sqliteAuditLogs)
	if result.Error != nil {
		log.Printf("Error reading audit logs from SQLite: %v", result.Error)
		stats.Failed = stats.Total
		return stats
	}

	batchSize := 100
	for i := 0; i < len(sqliteAuditLogs); i += batchSize {
		end := i + batchSize
		if end > len(sqliteAuditLogs) {
			end = len(sqliteAuditLogs)
		}

		batch := sqliteAuditLogs[i:end]
		for j, sqliteLog := range batch {
			// 创建新的审计日志
			auditLog := AuditLog{
				UserID:    sqliteLog.UserID,
				Action:    sqliteLog.Action,
				Details:   sqliteLog.Details,
				IPAddress: sqliteLog.IPAddress,
				UserAgent: sqliteLog.UserAgent,
				Status:    sqliteLog.Status,
				ErrorMsg:  sqliteLog.ErrorMsg,
				CreatedAt: sqliteLog.CreatedAt,
			}

			// 处理UserID
			if auditLog.UserID == 0 || !userIDMap[auditLog.UserID] {
				auditLog.UserID = adminUser.ID
			}

			// 处理ClusterID
			if sqliteLog.ClusterID != nil && clusterIDMap[*sqliteLog.ClusterID] {
				auditLog.ClusterID = *sqliteLog.ClusterID
			} else {
				auditLog.ClusterID = defaultCluster.ID
			}

			// 智能推断ResourceType
			if sqliteLog.Resource != "" {
				auditLog.ResourceType = sqliteLog.Resource
			} else {
				// 根据Action推断ResourceType
				auditLog.ResourceType = inferResourceType(auditLog.Action, auditLog.Details)
			}

			if err := dst.Create(&auditLog).Error; err != nil {
				log.Printf("Error inserting audit log %d: %v", i+j+1, err)
				stats.Failed++
			} else {
				stats.Success++
			}
		}

		if i%1000 == 0 {
			log.Printf("Processed %d/%d audit logs...", i, len(sqliteAuditLogs))
		}
	}

	log.Printf("Audit logs migration completed: %d success, %d failed, %d skipped", stats.Success, stats.Failed, stats.Skipped)
	return stats
}

// 智能推断ResourceType
func inferResourceType(action, details string) string {
	action = strings.ToLower(action)
	details = strings.ToLower(details)

	if strings.Contains(details, "node") || strings.Contains(details, "cordon") || strings.Contains(details, "drain") {
		return "node"
	}
	if strings.Contains(details, "label") {
		return "label"
	}
	if strings.Contains(details, "taint") {
		return "taint"
	}
	if strings.Contains(details, "cluster") {
		return "cluster"
	}
	if strings.Contains(details, "user") || strings.Contains(action, "login") {
		return "user"
	}

	// 默认值
	return "unknown"
}

func printMigrationSummary(allStats []MigrationStats) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("MIGRATION SUMMARY (v3 - 修复版本)")
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
