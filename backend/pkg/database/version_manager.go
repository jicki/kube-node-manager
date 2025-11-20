package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// VersionManager 版本管理器（基于代码的新版本）
type VersionManager struct {
	db                  *gorm.DB
	currentSchemaVersion string      // 当前数据库 schema 版本（checksum）
	targetSchemaVersion  string      // 目标 schema 版本（从代码计算）
	appVersion           string      // 应用版本（可选）
	migrationHistory     []Migration // 迁移历史
}

// Migration 迁移记录
type Migration struct {
	Version    string    // 迁移版本号
	AppVersion string    // 对应的应用版本
	AppliedAt  time.Time // 应用时间
}

// VersionInfo 版本信息
type VersionInfo struct {
	AppVersion          string     `json:"app_version"`           // 应用版本
	DBVersion           string     `json:"db_version"`            // 数据库 schema 版本（checksum）
	LatestSchemaVersion string     `json:"latest_schema_version"` // 最新 schema 版本（checksum）
	NeedsMigration      bool       `json:"needs_migration"`       // 是否需要迁移
	MigrationCount      int        `json:"migration_count"`       // 已执行迁移数量
	LastMigration       *string    `json:"last_migration"`        // 最后一次迁移
	LastMigrationTime   *time.Time `json:"last_migration_time"`   // 最后迁移时间
}

// NewVersionManager 创建版本管理器（新版本，不再需要 versionPath）
func NewVersionManager(db *gorm.DB, models []interface{}) (*VersionManager, error) {
	vm := &VersionManager{
		db: db,
	}

	// 确保 system_metadata 表存在
	if err := InitSystemMetadata(db); err != nil {
		return nil, fmt.Errorf("failed to init system_metadata: %w", err)
	}

	// 从数据库读取当前 schema 版本
	currentVersion, err := GetSchemaVersion(db)
	if err != nil {
		log.Printf("Warning: Failed to get schema version from DB: %v", err)
		currentVersion = ""
	}
	vm.currentSchemaVersion = currentVersion

	// 从数据库读取应用版本（可选）
	appVersion, err := GetAppVersion(db)
	if err != nil {
		log.Printf("Warning: Failed to get app version from DB: %v", err)
		appVersion = "unknown"
	}
	vm.appVersion = appVersion

	// 计算目标 schema 版本（从 GORM 模型）
	if len(models) > 0 {
		schemas, err := ExtractSchemaFromModels(db, models)
		if err != nil {
			return nil, fmt.Errorf("failed to extract schema from models: %w", err)
		}
		vm.targetSchemaVersion = CalculateSchemaChecksum(schemas)
	} else {
		// 如果没有提供模型，使用当前版本作为目标版本
		vm.targetSchemaVersion = currentVersion
	}

	log.Printf("Version Manager initialized: current=%s, target=%s", 
		vm.currentSchemaVersion, vm.targetSchemaVersion)

	return vm, nil
}

// GetVersionInfo 获取版本信息
func (vm *VersionManager) GetVersionInfo() *VersionInfo {
	info := &VersionInfo{
		AppVersion:          vm.appVersion,
		DBVersion:           vm.currentSchemaVersion,
		LatestSchemaVersion: vm.targetSchemaVersion,
		NeedsMigration:      vm.NeedsMigration(),
	}

	// 获取代码迁移统计
	executor, err := NewCodeMigrationExecutor(vm.db)
	if err == nil {
		executed, err := executor.GetExecutedMigrations()
		if err == nil {
			info.MigrationCount = len(executed)
			if len(executed) > 0 {
				last := executed[len(executed)-1]
				info.LastMigration = &last.MigrationID
				info.LastMigrationTime = &last.AppliedAt
			}
		}
	}

	return info
}

// NeedsMigration 判断是否需要迁移
func (vm *VersionManager) NeedsMigration() bool {
	// 如果当前版本为空（首次启动），不需要迁移（会自动创建）
	if vm.currentSchemaVersion == "" {
		return false
	}

	// 如果版本不一致，需要迁移
	if vm.currentSchemaVersion != vm.targetSchemaVersion {
		return true
	}

	// 检查是否有待执行的代码迁移
	executor, err := NewCodeMigrationExecutor(vm.db)
	if err != nil {
		return false
	}

	pending, err := executor.GetPendingMigrations()
	if err != nil {
		return false
	}

	return len(pending) > 0
}

// GetCurrentSchemaVersion 获取当前 schema 版本
func (vm *VersionManager) GetCurrentSchemaVersion() string {
	return vm.currentSchemaVersion
}

// GetTargetSchemaVersion 获取目标 schema 版本
func (vm *VersionManager) GetTargetSchemaVersion() string {
	return vm.targetSchemaVersion
}

// UpdateSchemaVersion 更新数据库中的 schema 版本
func (vm *VersionManager) UpdateSchemaVersion(version string) error {
	if err := SetSchemaVersion(vm.db, version); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	vm.currentSchemaVersion = version
	log.Printf("Schema version updated to: %s", version)
	return nil
}

// UpdateAppVersion 更新应用版本
func (vm *VersionManager) UpdateAppVersion(version string) error {
	if err := SetAppVersion(vm.db, version); err != nil {
		return fmt.Errorf("failed to update app version: %w", err)
	}

	vm.appVersion = version
	log.Printf("App version updated to: %s", version)
	return nil
}

// GetPendingMigrations 获取待执行的代码迁移
func (vm *VersionManager) GetPendingMigrations() []string {
	executor, err := NewCodeMigrationExecutor(vm.db)
	if err != nil {
		return []string{}
	}

	pending, err := executor.GetPendingMigrations()
	if err != nil {
		return []string{}
	}

	result := make([]string, len(pending))
	for i, m := range pending {
		result[i] = m.ID
	}

	return result
}

// PrintVersionInfo 打印版本信息（用于调试）
func (vm *VersionManager) PrintVersionInfo() {
	info := vm.GetVersionInfo()

	log.Println("========================================")
	log.Println("Database Version Information")
	log.Println("========================================")
	log.Printf("Application Version:    %s", info.AppVersion)
	log.Printf("Database Schema:        %s", info.DBVersion)
	log.Printf("Target Schema:          %s", info.LatestSchemaVersion)
	log.Printf("Needs Migration:        %v", info.NeedsMigration)
	log.Printf("Migrations Applied:     %d", info.MigrationCount)
	if info.LastMigration != nil {
		log.Printf("Last Migration:         %s", *info.LastMigration)
		if info.LastMigrationTime != nil {
			log.Printf("Last Migration Time:    %s", info.LastMigrationTime.Format(time.RFC3339))
		}
	}
	log.Println("========================================")
}

// RecordMigration 记录迁移（保持兼容性）
func (vm *VersionManager) RecordMigration(version string) error {
	// 这个方法在新系统中不再使用，代码迁移会自动记录
	// 保留此方法以保持向后兼容
	log.Printf("RecordMigration called with version: %s (deprecated in code-based system)", version)
	return nil
}

// ValidateVersion 验证版本格式
func (vm *VersionManager) ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Schema checksum 应该是 8 个十六进制字符
	if len(version) != 8 {
		return fmt.Errorf("invalid schema version format: %s (expected 8 hex chars)", version)
	}

	return nil
}

// GetMigrationHistory 获取迁移历史
func (vm *VersionManager) GetMigrationHistory() []Migration {
	return vm.migrationHistory
}

// GetCurrentAppVersion 获取当前应用版本
func (vm *VersionManager) GetCurrentAppVersion() string {
	return vm.appVersion
}

// GetCurrentDBVersion 获取当前数据库版本
func (vm *VersionManager) GetCurrentDBVersion() string {
	return vm.currentSchemaVersion
}

// GetLatestSchemaVersion 获取最新架构版本
func (vm *VersionManager) GetLatestSchemaVersion() string {
	return vm.targetSchemaVersion
}

// CompareVersions 比较两个版本（checksum 比较）
func (vm *VersionManager) CompareVersions(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	}
	return 1
}

// GetExpectedSchemaVersion 获取期望的 schema 版本（别名，保持兼容性）
func (vm *VersionManager) GetExpectedSchemaVersion() string {
	return vm.targetSchemaVersion
}
