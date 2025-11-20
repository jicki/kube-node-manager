package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// SystemMetadata 系统元数据表，用于存储系统级别的配置和状态信息
type SystemMetadata struct {
	Key       string    `gorm:"primaryKey;size:255" json:"key"`
	Value     string    `gorm:"not null;type:text" json:"value"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (SystemMetadata) TableName() string {
	return "system_metadata"
}

// 预定义的元数据键
const (
	MetaKeySchemaVersion    = "schema_version"     // 当前 schema checksum
	MetaKeyAppVersion       = "app_version"        // 应用版本
	MetaKeyLastMigration    = "last_migration"     // 最后执行的迁移
	MetaKeyMigrationSystem  = "migration_system"   // 迁移系统类型：sql_based / code_based
	MetaKeyLastSQLMigration = "last_sql_migration" // 最后的 SQL 迁移版本（过渡用）
)

// GetMetadata 获取元数据值
func GetMetadata(db *gorm.DB, key string) (string, error) {
	var metadata SystemMetadata
	
	if err := db.Where("key = ?", key).First(&metadata).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("metadata key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get metadata: %w", err)
	}
	
	return metadata.Value, nil
}

// SetMetadata 设置元数据值
func SetMetadata(db *gorm.DB, key, value string) error {
	metadata := SystemMetadata{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	
	// 使用 UPSERT 语义
	result := db.Where("key = ?", key).Assign(map[string]interface{}{
		"value":      value,
		"updated_at": time.Now(),
	}).FirstOrCreate(&metadata)
	
	if result.Error != nil {
		return fmt.Errorf("failed to set metadata: %w", result.Error)
	}
	
	return nil
}

// GetSchemaVersion 获取当前 schema 版本
func GetSchemaVersion(db *gorm.DB) (string, error) {
	version, err := GetMetadata(db, MetaKeySchemaVersion)
	if err != nil {
		// 如果不存在，返回空字符串而不是错误（初次启动情况）
		return "", nil
	}
	return version, nil
}

// SetSchemaVersion 设置 schema 版本
func SetSchemaVersion(db *gorm.DB, version string) error {
	return SetMetadata(db, MetaKeySchemaVersion, version)
}

// GetAppVersion 获取应用版本
func GetAppVersion(db *gorm.DB) (string, error) {
	version, err := GetMetadata(db, MetaKeyAppVersion)
	if err != nil {
		return "", nil
	}
	return version, nil
}

// SetAppVersion 设置应用版本
func SetAppVersion(db *gorm.DB, version string) error {
	return SetMetadata(db, MetaKeyAppVersion, version)
}

// GetMigrationSystem 获取迁移系统类型
func GetMigrationSystem(db *gorm.DB) (string, error) {
	system, err := GetMetadata(db, MetaKeyMigrationSystem)
	if err != nil {
		return "", nil
	}
	return system, nil
}

// SetMigrationSystem 设置迁移系统类型
func SetMigrationSystem(db *gorm.DB, system string) error {
	return SetMetadata(db, MetaKeyMigrationSystem, system)
}

// InitSystemMetadata 初始化系统元数据表
func InitSystemMetadata(db *gorm.DB) error {
	// 创建表
	if err := db.AutoMigrate(&SystemMetadata{}); err != nil {
		return fmt.Errorf("failed to create system_metadata table: %w", err)
	}
	
	return nil
}

// IsTransitionNeeded 检查是否需要从旧的 SQL 迁移系统过渡
func IsTransitionNeeded(db *gorm.DB) (bool, error) {
	// 检查 system_metadata 表是否存在
	if !db.Migrator().HasTable(&SystemMetadata{}) {
		// 检查 schema_migrations 表是否存在
		if db.Migrator().HasTable("schema_migrations") {
			return true, nil
		}
	}
	
	// 检查迁移系统类型
	system, err := GetMigrationSystem(db)
	if err != nil || system == "" {
		// 如果没有设置迁移系统类型，且存在 schema_migrations，则需要过渡
		if db.Migrator().HasTable("schema_migrations") {
			return true, nil
		}
	}
	
	return false, nil
}

// GetAllMetadata 获取所有元数据
func GetAllMetadata(db *gorm.DB) (map[string]string, error) {
	var metadata []SystemMetadata
	
	if err := db.Find(&metadata).Error; err != nil {
		return nil, fmt.Errorf("failed to get all metadata: %w", err)
	}
	
	result := make(map[string]string)
	for _, m := range metadata {
		result[m.Key] = m.Value
	}
	
	return result, nil
}

