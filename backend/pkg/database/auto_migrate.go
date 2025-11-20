package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/gorm"
)

// AutoMigrateConfig 自动迁移配置
type AutoMigrateConfig struct {
	Enabled           bool   // 是否启用自动迁移
	ValidateOnStartup bool   // 启动时验证
	RepairOnStartup   bool   // 启动时修复
	MigrationTimeout  int    // 超时时间（秒），0表示不限制
	MigrationsPath    string // 迁移文件路径
	VersionPath       string // VERSION 文件路径
}

// DefaultAutoMigrateConfig 返回默认配置
func DefaultAutoMigrateConfig() AutoMigrateConfig {
	return AutoMigrateConfig{
		Enabled:           true,
		ValidateOnStartup: true,
		RepairOnStartup:   true,
		MigrationTimeout:  300, // 5分钟
		MigrationsPath:    "",  // 自动检测
		VersionPath:       "",  // 自动检测
	}
}

// AutoMigrateOnStartup 应用启动时自动执行数据库迁移
func AutoMigrateOnStartup(db *gorm.DB, config AutoMigrateConfig) error {
	startTime := time.Now()
	
	log.Println("========================================")
	log.Println("Starting Database Migration on Startup")
	log.Println("========================================")
	
	// 1. 检查配置是否启用自动迁移
	if !config.Enabled {
		log.Println("Auto-migration is disabled in configuration")
		return nil
	}
	
	// 2. 检测路径
	if config.MigrationsPath == "" {
		config.MigrationsPath = detectMigrationsPath()
	}
	if config.VersionPath == "" {
		config.VersionPath = DetectVersionPath()
	}
	
	// 3. 初始化版本管理器
	log.Println("Initializing version manager...")
	versionManager, err := NewVersionManager(db, config.VersionPath)
	if err != nil {
		return fmt.Errorf("failed to initialize version manager: %w", err)
	}
	
	// 打印版本信息
	info := versionManager.GetVersionInfo()
	log.Printf("Application Version: %s", info.AppVersion)
	log.Printf("Database Version:    %s", info.DBVersion)
	log.Printf("Latest Schema:       %s", info.LatestSchemaVersion)
	
	// 4. 检查是否需要迁移
	needsMigration := versionManager.NeedsMigration()
	if needsMigration {
		pending := versionManager.GetPendingMigrations()
		log.Printf("⚠️  Database migration needed: %d pending migrations", len(pending))
		log.Printf("Pending migrations: %v", pending)
	} else {
		log.Println("✓ Database is up-to-date, no migration needed")
	}
	
	// 5. 获取数据库类型
	dbType := detectDatabaseType(db)
	log.Printf("Database Type: %s", dbType)
	
	// 6. 运行 SQL 迁移（如果需要）
	if needsMigration {
		log.Println("\n--- Running SQL Migrations ---")
		migrationManager := NewMigrationManager(db, MigrationConfig{
			MigrationsPath: config.MigrationsPath,
			UseEmbed:       false,
		})
		
		if err := migrationManager.AutoMigrate(); err != nil {
			return fmt.Errorf("SQL migration failed: %w", err)
		}
		log.Println("✓ SQL migrations completed successfully")
	}
	
	// 7. 验证数据库结构（如果启用）
	if config.ValidateOnStartup {
		log.Println("\n--- Validating Database Schema ---")
		validator := NewSchemaValidator(db, dbType)
		validationResult, err := validator.Validate()
		if err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}
		
		if validationResult.Valid {
			log.Println("✓ Database schema validation passed")
		} else {
			log.Printf("⚠️  Schema validation found issues:")
			log.Printf("   Critical: %d, Warnings: %d, Total: %d",
				validationResult.CriticalIssues,
				validationResult.WarningIssues,
				validationResult.TotalIssues)
			
			// 8. 如有问题且启用自动修复，则执行修复
			if config.RepairOnStartup && (validationResult.CriticalIssues > 0 || validationResult.WarningIssues > 0) {
				log.Println("\n--- Repairing Database Schema ---")
				repairer := NewSchemaRepairer(db, dbType, false)
				repairResult, err := repairer.Repair(validationResult)
				if err != nil {
					return fmt.Errorf("schema repair failed: %w", err)
				}
				
				if repairResult.Success {
					log.Println("✓ Database schema repaired successfully")
					log.Printf("   Tables Created: %d", len(repairResult.TablesCreated))
					log.Printf("   Columns Added: %d", len(repairResult.ColumnsAdded))
					log.Printf("   Indexes Created: %d", len(repairResult.IndexesCreated))
				} else {
					return fmt.Errorf("schema repair completed with errors: %d", len(repairResult.Errors))
				}
			} else if validationResult.CriticalIssues > 0 {
				return fmt.Errorf("critical schema issues found but auto-repair is disabled")
			}
		}
	}
	
	// 9. 记录迁移历史到数据库
	if err := recordMigrationHistory(db, versionManager, startTime); err != nil {
		log.Printf("Warning: Failed to record migration history: %v", err)
		// 不阻止启动
	}
	
	// 完成
	duration := time.Since(startTime)
	log.Println("\n========================================")
	log.Printf("Database Migration Completed in %.2fs", duration.Seconds())
	log.Println("✓ Database is ready and up-to-date")
	log.Println("========================================\n")
	
	return nil
}

// recordMigrationHistory 记录迁移历史
func recordMigrationHistory(db *gorm.DB, vm *VersionManager, startTime time.Time) error {
	// 确保 migration_histories 表存在
	if err := db.AutoMigrate(&MigrationHistory{}); err != nil {
		return fmt.Errorf("failed to ensure migration_histories table: %w", err)
	}
	
	info := vm.GetVersionInfo()
	duration := time.Since(startTime).Milliseconds()
	
	history := MigrationHistory{
		AppVersion:    info.AppVersion,
		DBVersion:     info.DBVersion,
		MigrationType: "auto_startup",
		Status:        "success",
		DurationMs:    duration,
		AppliedAt:     time.Now(),
	}
	
	if err := db.Create(&history).Error; err != nil {
		return fmt.Errorf("failed to create migration history: %w", err)
	}
	
	log.Printf("✓ Migration history recorded (ID: %d, Duration: %dms)", history.ID, history.DurationMs)
	return nil
}

// detectDatabaseType 检测数据库类型
func detectDatabaseType(db *gorm.DB) DatabaseType {
	dialector := db.Dialector.Name()
	
	switch dialector {
	case "postgres":
		return DatabaseTypePostgreSQL
	case "sqlite":
		return DatabaseTypeSQLite
	default:
		log.Printf("Warning: Unknown database type: %s, defaulting to SQLite", dialector)
		return DatabaseTypeSQLite
	}
}

// detectMigrationsPath 检测迁移文件路径（复用工具中的逻辑）
func detectMigrationsPath() string {
	possiblePaths := []string{
		"./migrations",
		"./backend/migrations",
		"../migrations",
		"/app/migrations",
		"/app/backend/migrations",
	}
	
	for _, path := range possiblePaths {
		if fileExists(path) {
			log.Printf("Found migrations directory at: %s", path)
			return path
		}
	}
	
	log.Println("Warning: migrations directory not found, using default: ./migrations")
	return "./migrations"
}

// fileExists 检查文件或目录是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MigrateWithTimeout 带超时的迁移执行
func MigrateWithTimeout(db *gorm.DB, config AutoMigrateConfig) error {
	if config.MigrationTimeout <= 0 {
		// 无超时限制
		return AutoMigrateOnStartup(db, config)
	}
	
	// 使用通道实现超时
	done := make(chan error, 1)
	
	go func() {
		done <- AutoMigrateOnStartup(db, config)
	}()
	
	select {
	case err := <-done:
		return err
	case <-time.After(time.Duration(config.MigrationTimeout) * time.Second):
		return fmt.Errorf("migration timeout after %d seconds", config.MigrationTimeout)
	}
}

