package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type         string
	DSN          string
	Host         string
	Port         int
	Database     string
	Username     string
	Password     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  int // seconds
}

// InitDatabase initializes database with support for multiple database types
func InitDatabase(config DatabaseConfig) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	dbType := strings.ToLower(config.Type)
	log.Printf("Initializing %s database", dbType)

	switch dbType {
	case "postgres", "postgresql":
		db, err = initPostgreSQL(config, gormConfig)
	case "sqlite":
		db, err = initSQLite(config, gormConfig)
	default:
		log.Printf("Unknown database type: %s, falling back to SQLite", config.Type)
		db, err = initSQLite(config, gormConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Configure connection pool
	if err := configureConnectionPool(db, config); err != nil {
		return nil, fmt.Errorf("failed to configure connection pool: %v", err)
	}

	log.Printf("Successfully initialized %s database", dbType)
	return db, nil
}

// initPostgreSQL initializes PostgreSQL database
func initPostgreSQL(config DatabaseConfig, gormConfig *gorm.Config) (*gorm.DB, error) {
	var dsn string

	// 检查 DSN 是否是 PostgreSQL 格式
	if config.DSN != "" && isPostgreSQLDSN(config.DSN) {
		dsn = config.DSN
		log.Println("Using provided PostgreSQL DSN")
	} else {
		if config.DSN != "" && !isPostgreSQLDSN(config.DSN) {
			log.Printf("Warning: DSN '%s' does not appear to be a PostgreSQL DSN, building DSN from individual components", config.DSN)
		}

		// Build DSN from individual components
		dsn = fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
			config.Host,
			config.Port,
			config.Username,
			config.Database,
			config.SSLMode,
		)

		if config.Password != "" {
			dsn += fmt.Sprintf(" password=%s", config.Password)
		}
		log.Println("Built PostgreSQL DSN from individual components")
	}

	log.Printf("Connecting to PostgreSQL database: %s", maskPasswordInDSN(dsn))

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB instance: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %v", err)
	}

	log.Println("Successfully connected to PostgreSQL database")
	return db, nil
}

// initSQLite initializes SQLite database
func initSQLite(config DatabaseConfig, gormConfig *gorm.Config) (*gorm.DB, error) {
	dsn := config.DSN
	if dsn == "" {
		dsn = "./data/kube-node-manager.db"
	}

	// Handle memory database
	if dsn == ":memory:" {
		log.Println("Using in-memory SQLite database (data will not persist)")
		return gorm.Open(sqlite.Open(dsn), gormConfig)
	}

	// Create directory for file database
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Failed to create database directory: %v, falling back to memory database", err)
		return gorm.Open(sqlite.Open(":memory:"), gormConfig)
	}

	// Check directory permissions
	if stat, err := os.Stat(dir); err != nil {
		log.Printf("Failed to check directory stats: %v, falling back to memory database", err)
		return gorm.Open(sqlite.Open(":memory:"), gormConfig)
	} else {
		log.Printf("Database directory permissions: %v", stat.Mode())
	}

	// Test write permissions
	testFile := filepath.Join(dir, "test.tmp")
	if f, err := os.Create(testFile); err != nil {
		log.Printf("Cannot write to database directory: %v, falling back to memory database", err)
		return gorm.Open(sqlite.Open(":memory:"), gormConfig)
	} else {
		f.Close()
		os.Remove(testFile)
	}

	// Open file database
	log.Printf("Opening SQLite database file: %s", dsn)
	db, err := gorm.Open(sqlite.Open(dsn), gormConfig)
	if err != nil {
		log.Printf("Failed to open SQLite file database: %v, falling back to memory database", err)
		return gorm.Open(sqlite.Open(":memory:"), gormConfig)
	}

	log.Printf("Successfully opened SQLite database: %s", dsn)
	return db, nil
}

// configureConnectionPool configures database connection pool
func configureConnectionPool(db *gorm.DB, config DatabaseConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Set connection pool parameters
	maxOpenConns := config.MaxOpenConns
	maxIdleConns := config.MaxIdleConns
	maxLifetime := config.MaxLifetime

	// Default values for SQLite (single connection)
	if strings.ToLower(config.Type) == "sqlite" {
		maxOpenConns = 1
		maxIdleConns = 1
	} else {
		// Default values for other databases
		if maxOpenConns <= 0 {
			maxOpenConns = 25
		}
		if maxIdleConns <= 0 {
			maxIdleConns = 10
		}
	}

	if maxLifetime <= 0 {
		maxLifetime = 3600 // 1 hour
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)

	log.Printf("Connection pool configured: MaxOpenConns=%d, MaxIdleConns=%d, MaxLifetime=%ds",
		maxOpenConns, maxIdleConns, maxLifetime)

	return nil
}

// isPostgreSQLDSN 检查 DSN 是否为 PostgreSQL 格式
func isPostgreSQLDSN(dsn string) bool {
	// PostgreSQL DSN 通常包含这些关键字之一
	pgKeywords := []string{"host=", "user=", "dbname=", "sslmode=", "port="}
	for _, keyword := range pgKeywords {
		if strings.Contains(dsn, keyword) {
			return true
		}
	}
	// 如果 DSN 看起来像文件路径，则不是 PostgreSQL DSN
	if strings.HasPrefix(dsn, "./") || strings.HasPrefix(dsn, "/") || dsn == ":memory:" {
		return false
	}
	return false
}

// maskPasswordInDSN masks password in DSN for logging
func maskPasswordInDSN(dsn string) string {
	if strings.Contains(dsn, "password=") {
		parts := strings.Split(dsn, " ")
		for i, part := range parts {
			if strings.HasPrefix(part, "password=") {
				parts[i] = "password=***"
			}
		}
		return strings.Join(parts, " ")
	}
	return dsn
}
