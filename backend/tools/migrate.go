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
	command := flag.String("cmd", "migrate", "Command to execute: migrate, status")
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

	// 创建迁移管理器
	// 智能检测迁移目录位置
	migrationsPath := detectMigrationsPath()
	migrationManager := database.NewMigrationManager(db, database.MigrationConfig{
		MigrationsPath: migrationsPath,
		UseEmbed:       false,
	})

	// 执行命令
	switch *command {
	case "migrate", "up":
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

	case "status":
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

	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: migrate, up, status")
		os.Exit(1)
	}
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
