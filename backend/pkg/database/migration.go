package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SchemaMigration 记录已执行的迁移
type SchemaMigration struct {
	Version   string    `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}

// MigrationConfig 迁移配置
type MigrationConfig struct {
	// MigrationsPath SQL迁移文件所在目录（仅用于文件系统模式）
	MigrationsPath string
	// EmbedMigrations 嵌入的迁移文件系统（用于打包后的二进制文件）
	EmbedMigrations embed.FS
	// UseEmbed 是否使用嵌入的文件系统
	UseEmbed bool
	// TableName 迁移跟踪表名
	TableName string
}

// MigrationManager 迁移管理器
type MigrationManager struct {
	db     *gorm.DB
	config MigrationConfig
}

// NewMigrationManager 创建迁移管理器
func NewMigrationManager(db *gorm.DB, config MigrationConfig) *MigrationManager {
	if config.TableName == "" {
		config.TableName = "schema_migrations"
	}
	if config.MigrationsPath == "" && !config.UseEmbed {
		config.MigrationsPath = "./migrations"
	}
	return &MigrationManager{
		db:     db,
		config: config,
	}
}

// AutoMigrate 自动运行所有待执行的迁移
func (m *MigrationManager) AutoMigrate() error {
	log.Println("Starting database migration check...")

	// 1. 确保迁移跟踪表存在
	if err := m.ensureMigrationTable(); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	// 2. 获取所有迁移文件
	migrations, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	if len(migrations) == 0 {
		log.Println("No migration files found, skipping migration")
		return nil
	}

	// 3. 获取已执行的迁移
	executedMigrations, err := m.getExecutedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	// 4. 找出待执行的迁移
	pendingMigrations := m.getPendingMigrations(migrations, executedMigrations)

	if len(pendingMigrations) == 0 {
		log.Println("All migrations are up to date, skipping migration")
		return nil
	}

	log.Printf("Found %d pending migration(s) to execute", len(pendingMigrations))

	// 5. 执行待执行的迁移
	for _, migration := range pendingMigrations {
		if err := m.executeMigration(migration); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration, err)
		}
	}

	log.Println("All migrations executed successfully")
	return nil
}

// ensureMigrationTable 确保迁移跟踪表存在
func (m *MigrationManager) ensureMigrationTable() error {
	return m.db.AutoMigrate(&SchemaMigration{})
}

// getMigrationFiles 获取所有迁移文件列表（已排序）
func (m *MigrationManager) getMigrationFiles() ([]string, error) {
	var files []string

	if m.config.UseEmbed {
		// 使用嵌入的文件系统
		entries, err := fs.ReadDir(m.config.EmbedMigrations, "migrations")
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded migrations: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".sql") {
				files = append(files, name)
			}
		}
	} else {
		// 使用文件系统
		// 检查目录是否存在
		if _, err := os.Stat(m.config.MigrationsPath); os.IsNotExist(err) {
			log.Printf("Migration directory %s does not exist, skipping migration", m.config.MigrationsPath)
			return files, nil
		}

		entries, err := os.ReadDir(m.config.MigrationsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migrations directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".sql") {
				files = append(files, name)
			}
		}
	}

	// 按文件名排序（确保迁移按顺序执行）
	sort.Strings(files)

	return files, nil
}

// getExecutedMigrations 获取已执行的迁移列表
func (m *MigrationManager) getExecutedMigrations() (map[string]bool, error) {
	var migrations []SchemaMigration
	if err := m.db.Find(&migrations).Error; err != nil {
		return nil, err
	}

	executed := make(map[string]bool)
	for _, migration := range migrations {
		executed[migration.Version] = true
	}

	return executed, nil
}

// getPendingMigrations 获取待执行的迁移列表
func (m *MigrationManager) getPendingMigrations(all []string, executed map[string]bool) []string {
	var pending []string
	for _, migration := range all {
		if !executed[migration] {
			pending = append(pending, migration)
		}
	}
	return pending
}

// executeMigration 执行单个迁移
func (m *MigrationManager) executeMigration(filename string) error {
	log.Printf("Executing migration: %s", filename)

	// 1. 读取迁移文件内容
	var content []byte
	var err error

	if m.config.UseEmbed {
		content, err = fs.ReadFile(m.config.EmbedMigrations, filepath.Join("migrations", filename))
	} else {
		content, err = os.ReadFile(filepath.Join(m.config.MigrationsPath, filename))
	}

	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// 2. 在事务中执行迁移
	err = m.db.Transaction(func(tx *gorm.DB) error {
		// 执行SQL语句
		sqlDB, err := tx.DB()
		if err != nil {
			return fmt.Errorf("failed to get database connection: %w", err)
		}

		if err := m.executeSQLStatements(sqlDB, string(content)); err != nil {
			return fmt.Errorf("failed to execute SQL: %w", err)
		}

		// 记录迁移
		migration := SchemaMigration{
			Version:   filename,
			AppliedAt: time.Now(),
		}

		if err := tx.Create(&migration).Error; err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("Successfully executed migration: %s", filename)
	return nil
}

// executeSQLStatements 执行SQL语句（支持多条语句）
func (m *MigrationManager) executeSQLStatements(db *sql.DB, sqlContent string) error {
	// 移除注释和空行
	lines := strings.Split(sqlContent, "\n")
	var cleanedLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}

	sqlContent = strings.Join(cleanedLines, "\n")

	// 按分号分割SQL语句（简单处理）
	statements := strings.Split(sqlContent, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// 执行单条SQL语句
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %s\nError: %w", stmt, err)
		}
	}

	return nil
}

// GetStatus 获取迁移状态
func (m *MigrationManager) GetStatus() (map[string]interface{}, error) {
	allMigrations, err := m.getMigrationFiles()
	if err != nil {
		return nil, err
	}

	executedMigrations, err := m.getExecutedMigrations()
	if err != nil {
		return nil, err
	}

	pendingMigrations := m.getPendingMigrations(allMigrations, executedMigrations)

	status := map[string]interface{}{
		"total_migrations":    len(allMigrations),
		"executed_migrations": len(executedMigrations),
		"pending_migrations":  len(pendingMigrations),
		"pending_list":        pendingMigrations,
	}

	return status, nil
}

