package service

import (
	"fmt"
	"kube-node-manager/pkg/database"

	"gorm.io/gorm"
)

// MigrationService 迁移服务
type MigrationService struct {
	db             *gorm.DB
	versionManager *database.VersionManager
}

// NewMigrationService 创建迁移服务
func NewMigrationService(db *gorm.DB) (*MigrationService, error) {
	versionPath := database.DetectVersionPath()
	versionManager, err := database.NewVersionManager(db, versionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create version manager: %w", err)
	}

	return &MigrationService{
		db:             db,
		versionManager: versionManager,
	}, nil
}

// GetVersionInfo 获取版本信息
func (s *MigrationService) GetVersionInfo() *database.VersionInfo {
	return s.versionManager.GetVersionInfo()
}

// GetMigrationStatus 获取迁移状态
func (s *MigrationService) GetMigrationStatus() (map[string]interface{}, error) {
	info := s.versionManager.GetVersionInfo()
	
	pending := s.versionManager.GetPendingMigrations()
	
	return map[string]interface{}{
		"app_version":          info.AppVersion,
		"db_version":           info.DBVersion,
		"latest_schema":        info.LatestSchemaVersion,
		"needs_migration":      info.NeedsMigration,
		"migrations_applied":   info.MigrationCount,
		"pending_migrations":   len(pending),
		"pending_list":         pending,
		"last_migration":       info.LastMigration,
		"last_migration_time":  info.LastMigrationTime,
	}, nil
}

// GetMigrationHistory 获取迁移历史记录
func (s *MigrationService) GetMigrationHistory(limit int) ([]database.MigrationHistory, error) {
	var histories []database.MigrationHistory
	
	query := s.db.Order("applied_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	if err := query.Find(&histories).Error; err != nil {
		return nil, fmt.Errorf("failed to get migration history: %w", err)
	}
	
	return histories, nil
}

// ValidateSchema 验证数据库结构
func (s *MigrationService) ValidateSchema() (*database.ValidationResult, error) {
	dbType := detectDatabaseType(s.db)
	validator := database.NewSchemaValidator(s.db, dbType)
	
	result, err := validator.Validate()
	if err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}
	
	return result, nil
}

// GetMigrationRegistry 获取迁移注册表信息
func (s *MigrationService) GetMigrationRegistry() []database.MigrationInfo {
	return database.MigrationRegistry
}

// GetMigrationStatistics 获取迁移统计
func (s *MigrationService) GetMigrationStatistics() map[string]interface{} {
	return database.GetMigrationStatistics()
}

// detectDatabaseType 检测数据库类型
func detectDatabaseType(db *gorm.DB) database.DatabaseType {
	dialector := db.Dialector.Name()
	
	switch dialector {
	case "postgres":
		return database.DatabaseTypePostgreSQL
	case "sqlite":
		return database.DatabaseTypeSQLite
	default:
		return database.DatabaseTypeSQLite
	}
}

