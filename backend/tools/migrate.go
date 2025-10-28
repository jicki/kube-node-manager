package main

import (
	"kube-node-manager/internal/config"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/database"
	"log"
)

func main() {
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

	log.Println("Starting database migration...")

	// 运行迁移
	if err := model.AutoMigrate(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
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
