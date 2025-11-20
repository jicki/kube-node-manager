package main

import (
	"flag"
	"fmt"
	"kube-node-manager/internal/config"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/database"
	"log"
	"os"
)

func main() {
	// 定义命令行参数
	command := flag.String("cmd", "migrate", "Command to execute: migrate, status, validate, repair, version, compare, list")
	dryRun := flag.Bool("dry-run", false, "Dry run mode (for repair command)")
	flag.Parse()

	cfg := config.LoadConfig()

	// 初始化数据库
	dbConfig := database.DatabaseConfig{
		Type:         cfg.Database.Type,
		DSN:          cfg.Database.DSN,
		Host:         cfg.Database.Host,
		Port:         cfg.Database.Port,
		Database:     cfg.Database.Database,
		Username:     cfg.Database.Username,
		Password:     cfg.Database.Password,
		SSLMode:      cfg.Database.SSLMode,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		MaxLifetime:  cfg.Database.MaxLifetime,
	}
	db, err := database.InitDatabase(dbConfig)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 获取数据库类型
	dbType := getDBType(cfg.Database.Type)

	// 创建迁移管理器
	migrationsPath := detectMigrationsPath()
	migrationManager := database.NewMigrationManager(db, database.MigrationConfig{
		MigrationsPath: migrationsPath,
		UseEmbed:       false,
	})

	// 执行命令
	switch *command {
	case "migrate", "up":
		executeMigrate(db, cfg, migrationManager)

	case "status":
		executeStatus(migrationManager)

	case "validate":
		executeValidate(db, dbType)

	case "repair":
		executeRepair(db, dbType, *dryRun)

	case "version":
		executeVersion(db)

	case "compare":
		executeCompare(db, dbType)

	case "list":
		executeList()

	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("\nAvailable commands:")
		fmt.Println("  migrate/up    - Run database migrations")
		fmt.Println("  status        - Show migration status")
		fmt.Println("  validate      - Validate database schema")
		fmt.Println("  repair        - Repair database schema (use --dry-run for preview)")
		fmt.Println("  version       - Show version information")
		fmt.Println("  compare       - Compare current schema with expected schema")
		fmt.Println("  list          - List all migrations")
		os.Exit(1)
	}
}

// executeMigrate 执行迁移
func executeMigrate(db *database.DB, cfg *config.Config, migrationManager *database.MigrationManager) {
	log.Println("Starting database migration...")

	// 运行 GORM 自动迁移
	if err := model.AutoMigrate(db); err != nil {
		log.Fatal("Failed to run GORM auto-migrations:", err)
	}

	// 运行 SQL 迁移
	if err := migrationManager.AutoMigrate(); err != nil {
		log.Fatal("Failed to run SQL migrations:", err)
	}

	log.Println("Database migration completed successfully!")

	// 获取数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database connection:", err)
	}
	defer sqlDB.Close()

	// 根据数据库类型列出表
	var tables []string
	if cfg.Database.Type == "sqlite" {
		result := db.Raw("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name").Scan(&tables)
		if result.Error != nil {
			log.Fatal("Failed to list tables:", result.Error)
		}
	} else if cfg.Database.Type == "postgres" || cfg.Database.Type == "postgresql" {
		result := db.Raw(`
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			ORDER BY table_name
		`).Scan(&tables)
		if result.Error != nil {
			log.Fatal("Failed to list tables:", result.Error)
		}
	}

	log.Println("\nTables in database:")
	for _, table := range tables {
		log.Printf("  - %s", table)
	}
}

// executeStatus 执行状态检查
func executeStatus(migrationManager *database.MigrationManager) {
	log.Println("Checking migration status...")

	status, err := migrationManager.GetStatus()
	if err != nil {
		log.Fatal("Failed to get migration status:", err)
	}

	fmt.Println("\n=== Migration Status ===")
	fmt.Printf("Total migrations:    %d\n", status["total_migrations"])
	fmt.Printf("Executed migrations: %d\n", status["executed_migrations"])
	fmt.Printf("Pending migrations:  %d\n", status["pending_migrations"])

	pendingList := status["pending_list"].([]string)
	if len(pendingList) > 0 {
		fmt.Println("\nPending migrations:")
		for _, migration := range pendingList {
			fmt.Printf("  - %s\n", migration)
		}
	} else {
		fmt.Println("\nAll migrations are up to date!")
	}
}

// executeValidate 执行验证
func executeValidate(db *database.DB, dbType database.DatabaseType) {
	log.Println("Validating database schema...")

	validator := database.NewSchemaValidator(db, dbType)
	result, err := validator.Validate()
	if err != nil {
		log.Fatal("Validation failed:", err)
	}

	validator.PrintValidationResult(result)

	if !result.Valid {
		fmt.Println("\n💡 Suggestions:")
		suggestions := validator.GetRepairSuggestions(result)
		for _, suggestion := range suggestions {
			fmt.Printf("  - %s\n", suggestion)
		}
		os.Exit(1)
	}
}

// executeRepair 执行修复
func executeRepair(db *database.DB, dbType database.DatabaseType, dryRun bool) {
	if dryRun {
		log.Println("Running in DRY RUN mode...")
	} else {
		log.Println("Repairing database schema...")
	}

	if err := database.ValidateAndRepair(db, dbType, dryRun); err != nil {
		log.Fatal("Repair failed:", err)
	}
}

// executeVersion 执行版本查看
func executeVersion(db *database.DB) {
	versionPath := database.DetectVersionPath()
	versionManager, err := database.NewVersionManager(db, versionPath)
	if err != nil {
		log.Fatal("Failed to create version manager:", err)
	}

	versionManager.PrintVersionInfo()

	// 打印迁移统计
	database.PrintMigrationStatistics()
}

// executeCompare 执行比较
func executeCompare(db *database.DB, dbType database.DatabaseType) {
	log.Println("Comparing database schema...")

	validator := database.NewSchemaValidator(db, dbType)
	result, err := validator.Validate()
	if err != nil {
		log.Fatal("Comparison failed:", err)
	}

	// 打印详细的比较结果
	fmt.Println("\n=== Schema Comparison ===")
	
	if result.Valid {
		fmt.Println("✅ Database schema matches the expected schema")
		return
	}

	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  Missing Tables:   %d\n", len(result.MissingTables))
	fmt.Printf("  Extra Tables:     %d\n", len(result.ExtraTables))
	fmt.Printf("  Critical Issues:  %d\n", result.CriticalIssues)
	fmt.Printf("  Warnings:         %d\n", result.WarningIssues)
	fmt.Printf("  Total Issues:     %d\n", result.TotalIssues)

	// 打印详细差异
	for _, tableResult := range result.TableResults {
		if len(tableResult.Issues) == 0 {
			continue
		}

		fmt.Printf("\n📋 Table: %s\n", tableResult.TableName)
		
		if len(tableResult.MissingColumns) > 0 {
			fmt.Printf("  Missing Columns (%d):\n", len(tableResult.MissingColumns))
			for _, col := range tableResult.MissingColumns {
				fmt.Printf("    - %s\n", col)
			}
		}

		if len(tableResult.ExtraColumns) > 0 {
			fmt.Printf("  Extra Columns (%d):\n", len(tableResult.ExtraColumns))
			for _, col := range tableResult.ExtraColumns {
				fmt.Printf("    - %s\n", col)
			}
		}

		if len(tableResult.TypeMismatches) > 0 {
			fmt.Printf("  Type Mismatches (%d):\n", len(tableResult.TypeMismatches))
			for _, mismatch := range tableResult.TypeMismatches {
				fmt.Printf("    - %s: %s -> %s [%s]\n", 
					mismatch.ColumnName, mismatch.ActualType, mismatch.ExpectedType, mismatch.Severity)
			}
		}

		if len(tableResult.MissingIndexes) > 0 {
			fmt.Printf("  Missing Indexes (%d):\n", len(tableResult.MissingIndexes))
			for _, idx := range tableResult.MissingIndexes {
				fmt.Printf("    - %s\n", idx)
			}
		}
	}
	
	fmt.Println()
}

// executeList 列出所有迁移
func executeList() {
	database.PrintMigrationList()
}

// detectMigrationsPath 智能检测迁移文件目录位置
func detectMigrationsPath() string {
	// 尝试的路径列表（按优先级排序）
	possiblePaths := []string{
		"./migrations",                    // 当前目录下的 migrations
		"./backend/migrations",            // 项目根目录下的 backend/migrations
		"../migrations",                   // 父目录下的 migrations
		"/app/migrations",                 // 容器中的绝对路径
		"/app/backend/migrations",         // 容器中的另一个可能路径
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			log.Printf("Found migrations directory at: %s", path)
			return path
		}
	}

	// 如果都找不到，返回默认路径（让迁移管理器处理）
	log.Println("Warning: migrations directory not found, using default path: ./migrations")
	return "./migrations"
}

// getDBType 获取数据库类型
func getDBType(dbTypeStr string) database.DatabaseType {
	switch dbTypeStr {
	case "postgres", "postgresql":
		return database.DatabaseTypePostgreSQL
	case "sqlite":
		return database.DatabaseTypeSQLite
	default:
		return database.DatabaseTypeSQLite
	}
}
