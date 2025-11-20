package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// AutoMigrateConfig 自动迁移配置
type AutoMigrateConfig struct {
	Enabled           bool   // 是否启用自动迁移
	ValidateOnStartup bool   // 启动时验证
	RepairOnStartup   bool   // 启动时修复
	MigrationTimeout  int    // 超时时间（秒），0表示不限制
}

// DefaultAutoMigrateConfig 返回默认配置
func DefaultAutoMigrateConfig() AutoMigrateConfig {
	return AutoMigrateConfig{
		Enabled:           true,
		ValidateOnStartup: true,
		RepairOnStartup:   true,
		MigrationTimeout:  300, // 5分钟
	}
}

// AutoMigrateOnStartup 应用启动时自动执行数据库迁移（新版本 - 基于代码）
func AutoMigrateOnStartup(db *gorm.DB, models []interface{}, config AutoMigrateConfig) error {
	startTime := time.Now()
	
	log.Println("========================================")
	log.Println("Starting Code-Based Database Migration")
	log.Println("========================================")
	
	// 1. 检查配置是否启用自动迁移
	if !config.Enabled {
		log.Println("Auto-migration is disabled in configuration")
		return nil
	}
	
	// 2. 初始化 system_metadata 表
	log.Println("Initializing system metadata...")
	if err := InitSystemMetadata(db); err != nil {
		return fmt.Errorf("failed to initialize system_metadata: %w", err)
	}
	
	// 3. 检查是否需要从旧系统过渡
	needsTransition, err := IsTransitionNeeded(db)
	if err != nil {
		return fmt.Errorf("failed to check transition need: %w", err)
	}
	
	if needsTransition {
		log.Println("\n--- Transitioning from SQL-based to Code-based Migration System ---")
		if err := performTransition(db); err != nil {
			return fmt.Errorf("transition failed: %w", err)
		}
		log.Println("✓ Transition completed successfully")
	}
	
	// 4. 执行 GORM AutoMigrate（所有模型）
	log.Println("\n--- Running GORM AutoMigrate ---")
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("GORM AutoMigrate failed: %w", err)
	}
	log.Println("✓ GORM AutoMigrate completed")
	
	// 5. 初始化版本管理器
	log.Println("\n--- Initializing Version Manager ---")
	versionManager, err := NewVersionManager(db, models)
	if err != nil {
		return fmt.Errorf("failed to initialize version manager: %w", err)
	}
	
	// 打印版本信息
	info := versionManager.GetVersionInfo()
	log.Printf("Application Version:    %s", info.AppVersion)
	log.Printf("Current Schema (DB):    %s", info.DBVersion)
	log.Printf("Target Schema (Code):   %s", info.LatestSchemaVersion)
	
	// 6. 计算并检查 schema 版本
	currentSchema := versionManager.GetCurrentSchemaVersion()
	targetSchema := versionManager.GetTargetSchemaVersion()
	
	if currentSchema == "" {
		log.Println("ℹ️  First time initialization detected")
	} else if currentSchema != targetSchema {
		log.Printf("⚠️  Schema version mismatch: %s -> %s", currentSchema, targetSchema)
	} else {
		log.Println("✓ Schema version is up-to-date")
	}
	
	// 7. 执行代码迁移（如果有）
	log.Println("\n--- Executing Code Migrations ---")
	executor, err := NewCodeMigrationExecutor(db)
	if err != nil {
		return fmt.Errorf("failed to create migration executor: %w", err)
	}
	
	if err := executor.ExecuteCodeMigrations(); err != nil {
		return fmt.Errorf("code migrations failed: %w", err)
	}
	
	// 8. 获取数据库类型
	dbType := detectDatabaseType(db)
	
	// 9. 验证数据库结构（如果启用）
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
			
			// 10. 如有问题且启用自动修复，则执行修复
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
	
	// 11. 更新 system_metadata 中的 schema 版本
	log.Println("\n--- Updating Schema Version ---")
	if err := versionManager.UpdateSchemaVersion(targetSchema); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}
	log.Printf("✓ Schema version set to: %s", targetSchema)
	
	// 12. 记录迁移历史
	if err := recordCodeMigrationHistory(db, info, startTime); err != nil {
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

// performTransition 执行从旧的 SQL 迁移系统到新的代码迁移系统的过渡
func performTransition(db *gorm.DB) error {
	log.Println("Performing transition from SQL-based to code-based migration system...")
	
	// 1. 从 schema_migrations 读取最后的 SQL 迁移版本
	var lastMigration SchemaMigration
	err := db.Order("applied_at DESC").First(&lastMigration).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to get last SQL migration: %w", err)
	}
	
	if err == nil {
		log.Printf("Last SQL migration: %s (applied at %s)", 
			lastMigration.Version, lastMigration.AppliedAt.Format(time.RFC3339))
		
		// 保存最后的 SQL 迁移版本
		if err := SetMetadata(db, MetaKeyLastSQLMigration, lastMigration.Version); err != nil {
			return fmt.Errorf("failed to save last SQL migration version: %w", err)
		}
	} else {
		log.Println("No SQL migrations found in old system")
	}
	
	// 2. 标记已完成旧系统迁移
	if err := SetMigrationSystem(db, "code_based"); err != nil {
		return fmt.Errorf("failed to set migration system type: %w", err)
	}
	
	log.Println("✓ Migration system transitioned to code-based")
	return nil
}

// recordCodeMigrationHistory 记录迁移历史到数据库
func recordCodeMigrationHistory(db *gorm.DB, info *VersionInfo, startTime time.Time) error {
	// 确保 migration_histories 表存在
	if err := db.AutoMigrate(&MigrationHistory{}); err != nil {
		return fmt.Errorf("failed to ensure migration_histories table: %w", err)
	}
	
	duration := time.Since(startTime).Milliseconds()
	
	history := MigrationHistory{
		AppVersion:    info.AppVersion,
		DBVersion:     info.DBVersion,
		MigrationType: "code_based_startup",
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

// MigrateWithTimeout 带超时的迁移执行
func MigrateWithTimeout(db *gorm.DB, models []interface{}, config AutoMigrateConfig) error {
	if config.MigrationTimeout <= 0 {
		// 无超时限制
		return AutoMigrateOnStartup(db, models, config)
	}
	
	// 使用通道实现超时
	done := make(chan error, 1)
	
	go func() {
		done <- AutoMigrateOnStartup(db, models, config)
	}()
	
	select {
	case err := <-done:
		return err
	case <-time.After(time.Duration(config.MigrationTimeout) * time.Second):
		return fmt.Errorf("migration timeout after %d seconds", config.MigrationTimeout)
	}
}
