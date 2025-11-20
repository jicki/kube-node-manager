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
	"unicode"

	"gorm.io/gorm"
)

// SchemaMigration 记录已执行的迁移（旧格式，保持兼容）
type SchemaMigration struct {
	Version   string    `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}

// MigrationHistory 迁移历史记录（新格式，包含更多信息）
type MigrationHistory struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Version       string    `gorm:"size:255" json:"version"`                          // 迁移版本号
	AppVersion    string    `gorm:"size:50" json:"app_version"`                       // 应用版本
	DBVersion     string    `gorm:"size:50" json:"db_version"`                        // 数据库架构版本
	MigrationType string    `gorm:"size:50;not null" json:"migration_type"`           // 迁移类型: sql/auto_repair/gorm/auto_startup
	Status        string    `gorm:"size:20;not null;default:'success'" json:"status"` // 状态: success/failed/pending
	DurationMs    int64     `gorm:"default:0" json:"duration_ms"`                     // 执行耗时（毫秒）
	ErrorMessage  string    `gorm:"type:text" json:"error_message,omitempty"`         // 错误信息
	AppliedAt     time.Time `gorm:"not null;index" json:"applied_at"`                 // 应用时间
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName 指定表名
func (MigrationHistory) TableName() string {
	return "migration_histories"
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

	// 智能分割SQL语句（支持 PostgreSQL 的 $$ 语法）
	statements := m.splitSQLStatements(sqlContent)

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

// splitSQLStatements 智能分割SQL语句，支持 PostgreSQL 的 dollar-quoted strings
func (m *MigrationManager) splitSQLStatements(sqlContent string) []string {
	var statements []string
	var currentStmt strings.Builder
	inDollarQuote := false
	dollarTag := ""

	runes := []rune(sqlContent)
	i := 0

	for i < len(runes) {
		// 检查是否是 dollar quote 标记
		if runes[i] == '$' {
			// 提取 dollar tag (包括前后的 $)
			tag := string(runes[i]) // 开始的 $
			j := i + 1
			
			// 读取 tag 内容（字母、数字、下划线）
			for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				tag += string(runes[j])
				j++
			}
			
			// 必须以 $ 结束才是有效的 dollar quote
			if j < len(runes) && runes[j] == '$' {
				tag += string(runes[j]) // 结束的 $
				
				if !inDollarQuote {
					// 进入 dollar quote
					inDollarQuote = true
					dollarTag = tag
					currentStmt.WriteString(tag)
					i = j + 1
					continue
				} else if tag == dollarTag {
					// 退出 dollar quote
					inDollarQuote = false
					currentStmt.WriteString(tag)
					dollarTag = ""
					i = j + 1
					continue
				}
			}
		}

		// 如果不在 dollar quote 中，检查分号
		if !inDollarQuote && runes[i] == ';' {
			stmt := strings.TrimSpace(currentStmt.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			currentStmt.Reset()
			i++
			continue
		}

		currentStmt.WriteRune(runes[i])
		i++
	}

	// 添加最后一个语句（如果有）
	stmt := strings.TrimSpace(currentStmt.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
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

