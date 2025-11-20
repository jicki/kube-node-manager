package database

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// VersionManager 版本管理器
type VersionManager struct {
	db                  *gorm.DB
	currentAppVersion   string      // 应用当前版本（从 VERSION 文件）
	currentDBVersion    string      // 数据库当前版本（最后执行的迁移版本）
	latestSchemaVersion string      // 最新架构版本
	migrationHistory    []Migration // 迁移历史
	versionPath         string      // VERSION 文件路径
}

// Migration 迁移记录
type Migration struct {
	Version    string    // 迁移版本号（如 "001", "002"）
	AppVersion string    // 对应的应用版本
	AppliedAt  time.Time // 应用时间
}

// VersionInfo 版本信息
type VersionInfo struct {
	AppVersion          string    `json:"app_version"`           // 应用版本
	DBVersion           string    `json:"db_version"`            // 数据库版本
	LatestSchemaVersion string    `json:"latest_schema_version"` // 最新架构版本
	NeedsMigration      bool      `json:"needs_migration"`       // 是否需要迁移
	MigrationCount      int       `json:"migration_count"`       // 已执行迁移数量
	LastMigration       *string   `json:"last_migration"`        // 最后一次迁移
	LastMigrationTime   *time.Time `json:"last_migration_time"`  // 最后迁移时间
}

// VersionMapping 版本映射表（应用版本 -> 数据库架构版本）
var VersionMapping = map[string]string{
	"v2.34.1": "023", // 当前最新迁移为 023
	"v2.34.0": "022",
	"v2.33.0": "021",
	"v2.32.0": "019",
	"v2.31.0": "017",
	"v2.30.0": "015",
	"v2.29.0": "012",
	"v2.28.0": "010",
	"v2.27.0": "008",
	"v2.26.0": "006",
	"v2.25.0": "003",
	"v2.24.0": "001",
}

// NewVersionManager 创建版本管理器
func NewVersionManager(db *gorm.DB, versionPath string) (*VersionManager, error) {
	vm := &VersionManager{
		db:          db,
		versionPath: versionPath,
	}

	// 读取应用版本
	if err := vm.loadAppVersion(); err != nil {
		log.Printf("Warning: Failed to load app version: %v", err)
		vm.currentAppVersion = "unknown"
	}

	// 获取数据库当前版本
	if err := vm.loadDBVersion(); err != nil {
		log.Printf("Warning: Failed to load DB version: %v", err)
		vm.currentDBVersion = "000"
	}

	// 设置最新架构版本
	vm.latestSchemaVersion = vm.getLatestSchemaVersion()

	return vm, nil
}

// loadAppVersion 从 VERSION 文件读取应用版本
func (vm *VersionManager) loadAppVersion() error {
	// 尝试多个可能的路径
	possiblePaths := []string{
		vm.versionPath,
		"./VERSION",
		"../VERSION",
		"../../VERSION",
		"/app/VERSION",
	}

	var content []byte
	var err error
	var foundPath string

	for _, path := range possiblePaths {
		if path == "" {
			continue
		}
		content, err = ioutil.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if err != nil {
		return fmt.Errorf("failed to read VERSION file from any location: %w", err)
	}

	version := strings.TrimSpace(string(content))
	if version == "" {
		return fmt.Errorf("VERSION file is empty")
	}

	// 确保版本号以 v 开头
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	vm.currentAppVersion = version
	log.Printf("Loaded app version: %s from %s", version, foundPath)
	return nil
}

// loadDBVersion 从数据库加载当前版本
func (vm *VersionManager) loadDBVersion() error {
	var migrations []SchemaMigration

	// 确保 schema_migrations 表存在
	if err := vm.db.AutoMigrate(&SchemaMigration{}); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	// 查询所有迁移记录
	if err := vm.db.Order("version DESC").Find(&migrations).Error; err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}

	if len(migrations) == 0 {
		vm.currentDBVersion = "000"
		log.Println("No migrations found in database, DB version: 000")
		return nil
	}

	// 构建迁移历史
	vm.migrationHistory = make([]Migration, len(migrations))
	for i, m := range migrations {
		vm.migrationHistory[i] = Migration{
			Version:   m.Version,
			AppliedAt: m.AppliedAt,
		}
	}

	// 获取最后一个迁移的版本号（提取数字部分）
	lastMigration := migrations[0].Version
	vm.currentDBVersion = vm.extractVersionNumber(lastMigration)
	
	log.Printf("Loaded DB version: %s (last migration: %s)", vm.currentDBVersion, lastMigration)
	return nil
}

// extractVersionNumber 从迁移文件名提取版本号
// 例如："001_add_anomaly_indexes.sql" -> "001"
func (vm *VersionManager) extractVersionNumber(filename string) string {
	// 移除 .sql 后缀
	name := strings.TrimSuffix(filename, ".sql")
	
	// 提取前缀数字
	parts := strings.SplitN(name, "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	
	return name
}

// getLatestSchemaVersion 获取最新架构版本
func (vm *VersionManager) getLatestSchemaVersion() string {
	// 从版本映射中找到最新的架构版本
	latestSchema := "000"
	for _, schemaVersion := range VersionMapping {
		if schemaVersion > latestSchema {
			latestSchema = schemaVersion
		}
	}
	return latestSchema
}

// GetVersionInfo 获取版本信息
func (vm *VersionManager) GetVersionInfo() *VersionInfo {
	info := &VersionInfo{
		AppVersion:          vm.currentAppVersion,
		DBVersion:           vm.currentDBVersion,
		LatestSchemaVersion: vm.latestSchemaVersion,
		MigrationCount:      len(vm.migrationHistory),
	}

	// 判断是否需要迁移
	info.NeedsMigration = vm.NeedsMigration()

	// 设置最后迁移信息
	if len(vm.migrationHistory) > 0 {
		lastMig := vm.migrationHistory[0]
		info.LastMigration = &lastMig.Version
		info.LastMigrationTime = &lastMig.AppliedAt
	}

	return info
}

// NeedsMigration 判断是否需要迁移
func (vm *VersionManager) NeedsMigration() bool {
	// 比较当前数据库版本和最新架构版本
	return vm.currentDBVersion < vm.latestSchemaVersion
}

// GetExpectedSchemaVersion 获取当前应用版本期望的架构版本
func (vm *VersionManager) GetExpectedSchemaVersion() string {
	// 从映射表中查找当前应用版本对应的架构版本
	if schemaVersion, exists := VersionMapping[vm.currentAppVersion]; exists {
		return schemaVersion
	}
	
	// 如果找不到精确匹配，使用最新架构版本
	log.Printf("No exact schema version mapping found for app version %s, using latest: %s", 
		vm.currentAppVersion, vm.latestSchemaVersion)
	return vm.latestSchemaVersion
}

// GetPendingMigrations 获取待执行的迁移列表
func (vm *VersionManager) GetPendingMigrations() []string {
	pendingMigrations := []string{}
	
	// 获取期望的架构版本
	expectedVersion := vm.GetExpectedSchemaVersion()
	
	// 找出所有小于等于期望版本但大于当前版本的迁移
	currentVersionInt := vm.versionToInt(vm.currentDBVersion)
	expectedVersionInt := vm.versionToInt(expectedVersion)
	
	for i := currentVersionInt + 1; i <= expectedVersionInt; i++ {
		versionStr := fmt.Sprintf("%03d", i)
		pendingMigrations = append(pendingMigrations, versionStr)
	}
	
	return pendingMigrations
}

// versionToInt 将版本字符串转换为整数（用于比较）
func (vm *VersionManager) versionToInt(version string) int {
	var versionInt int
	fmt.Sscanf(version, "%d", &versionInt)
	return versionInt
}

// CompareVersions 比较两个版本号
// 返回值: -1 表示 v1 < v2, 0 表示 v1 == v2, 1 表示 v1 > v2
func (vm *VersionManager) CompareVersions(v1, v2 string) int {
	v1Int := vm.versionToInt(v1)
	v2Int := vm.versionToInt(v2)
	
	if v1Int < v2Int {
		return -1
	} else if v1Int > v2Int {
		return 1
	}
	return 0
}

// GetMigrationHistory 获取迁移历史
func (vm *VersionManager) GetMigrationHistory() []Migration {
	return vm.migrationHistory
}

// GetCurrentAppVersion 获取当前应用版本
func (vm *VersionManager) GetCurrentAppVersion() string {
	return vm.currentAppVersion
}

// GetCurrentDBVersion 获取当前数据库版本
func (vm *VersionManager) GetCurrentDBVersion() string {
	return vm.currentDBVersion
}

// GetLatestSchemaVersion 获取最新架构版本
func (vm *VersionManager) GetLatestSchemaVersion() string {
	return vm.latestSchemaVersion
}

// ValidateVersion 验证版本格式是否正确
func (vm *VersionManager) ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	
	// 验证版本号格式（应该是 v开头后跟数字）
	if !strings.HasPrefix(version, "v") {
		return fmt.Errorf("version should start with 'v': %s", version)
	}
	
	versionPart := strings.TrimPrefix(version, "v")
	parts := strings.Split(versionPart, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid version format: %s (expected format: vX.Y.Z)", version)
	}
	
	return nil
}

// RecordMigration 记录迁移执行
func (vm *VersionManager) RecordMigration(version string) error {
	migration := SchemaMigration{
		Version:   version,
		AppliedAt: time.Now(),
	}
	
	if err := vm.db.Create(&migration).Error; err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	// 更新内存中的版本信息
	vm.currentDBVersion = vm.extractVersionNumber(version)
	vm.migrationHistory = append([]Migration{{
		Version:   version,
		AppliedAt: migration.AppliedAt,
	}}, vm.migrationHistory...)
	
	log.Printf("Recorded migration: %s", version)
	return nil
}

// GetUpgradePath 获取从当前版本到目标版本的升级路径
func (vm *VersionManager) GetUpgradePath(targetVersion string) ([]string, error) {
	currentInt := vm.versionToInt(vm.currentDBVersion)
	targetInt := vm.versionToInt(targetVersion)
	
	if currentInt >= targetInt {
		return []string{}, fmt.Errorf("target version %s is not newer than current version %s", 
			targetVersion, vm.currentDBVersion)
	}
	
	path := []string{}
	for i := currentInt + 1; i <= targetInt; i++ {
		path = append(path, fmt.Sprintf("%03d", i))
	}
	
	return path, nil
}

// DetectVersionPath 智能检测 VERSION 文件位置
func DetectVersionPath() string {
	possiblePaths := []string{
		"./VERSION",
		"../VERSION",
		"../../VERSION",
		"/app/VERSION",
	}
	
	// 尝试从当前工作目录获取
	if cwd, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths, 
			filepath.Join(cwd, "VERSION"),
			filepath.Join(filepath.Dir(cwd), "VERSION"),
		)
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			log.Printf("Found VERSION file at: %s", path)
			return path
		}
	}
	
	log.Println("Warning: VERSION file not found, using default: ./VERSION")
	return "./VERSION"
}

// PrintVersionInfo 打印版本信息（用于命令行输出）
func (vm *VersionManager) PrintVersionInfo() {
	info := vm.GetVersionInfo()
	
	fmt.Println("\n=== Version Information ===")
	fmt.Printf("Application Version:    %s\n", info.AppVersion)
	fmt.Printf("Database Version:       %s\n", info.DBVersion)
	fmt.Printf("Latest Schema Version:  %s\n", info.LatestSchemaVersion)
	fmt.Printf("Migrations Applied:     %d\n", info.MigrationCount)
	
	if info.LastMigration != nil {
		fmt.Printf("Last Migration:         %s\n", *info.LastMigration)
		if info.LastMigrationTime != nil {
			fmt.Printf("Last Migration Time:    %s\n", info.LastMigrationTime.Format(time.RFC3339))
		}
	}
	
	if info.NeedsMigration {
		fmt.Println("\n⚠️  Database migration needed!")
		pending := vm.GetPendingMigrations()
		if len(pending) > 0 {
			fmt.Printf("Pending migrations: %v\n", pending)
		}
	} else {
		fmt.Println("\n✅ Database is up to date")
	}
}

