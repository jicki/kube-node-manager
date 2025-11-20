package database

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSchemaDefinitionCompleteness 测试表结构定义的完整性
func TestSchemaDefinitionCompleteness(t *testing.T) {
	schemas := AllTableSchemas()
	
	if len(schemas) == 0 {
		t.Fatal("No table schemas defined")
	}
	
	t.Logf("Found %d table schemas", len(schemas))
	
	for _, schema := range schemas {
		if schema.Name == "" {
			t.Errorf("Table schema has empty name")
		}
		
		if len(schema.Columns) == 0 {
			t.Errorf("Table %s has no columns defined", schema.Name)
		}
		
		// 验证至少有一个主键
		hasPrimaryKey := false
		for _, col := range schema.Columns {
			if col.PrimaryKey {
				hasPrimaryKey = true
				break
			}
		}
		
		if !hasPrimaryKey {
			t.Errorf("Table %s has no primary key defined", schema.Name)
		}
		
		t.Logf("✓ Table %s: %d columns, %d indexes", 
			schema.Name, len(schema.Columns), len(schema.Indexes))
	}
}

// TestGetTableSchema 测试根据名称获取表结构
func TestGetTableSchema(t *testing.T) {
	testCases := []string{
		"users",
		"clusters",
		"ansible_tasks",
		"node_anomalies",
	}
	
	for _, tableName := range testCases {
		schema, err := GetTableSchema(tableName)
		if err != nil {
			t.Errorf("Failed to get schema for table %s: %v", tableName, err)
			continue
		}
		
		if schema.Name != tableName {
			t.Errorf("Expected table name %s, got %s", tableName, schema.Name)
		}
		
		t.Logf("✓ Successfully retrieved schema for %s", tableName)
	}
	
	// 测试不存在的表
	_, err := GetTableSchema("nonexistent_table")
	if err == nil {
		t.Error("Expected error for nonexistent table, got nil")
	}
}

// TestColumnTypeMapping 测试字段类型映射
func TestColumnTypeMapping(t *testing.T) {
	testCases := []struct {
		originalType string
		sqliteType   string
		postgresType string
	}{
		{"SERIAL", "INTEGER", "SERIAL"},
		{"VARCHAR(255)", "TEXT", "VARCHAR(255)"},
		{"BOOLEAN", "INTEGER", "BOOLEAN"},
		{"TIMESTAMP", "DATETIME", "TIMESTAMP"},
		{"JSONB", "TEXT", "JSONB"},
	}
	
	for _, tc := range testCases {
		col := ColumnDefinition{Type: tc.originalType}
		
		sqliteType := col.GetType(DatabaseTypeSQLite)
		if sqliteType != tc.sqliteType {
			t.Errorf("SQLite type mapping failed for %s: expected %s, got %s", 
				tc.originalType, tc.sqliteType, sqliteType)
		}
		
		postgresType := col.GetType(DatabaseTypePostgreSQL)
		if postgresType != tc.postgresType {
			t.Errorf("PostgreSQL type mapping failed for %s: expected %s, got %s", 
				tc.originalType, tc.postgresType, postgresType)
		}
	}
}

// TestVersionManager 测试版本管理器
func TestVersionManager(t *testing.T) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal("Failed to create test database:", err)
	}
	
	// 创建版本管理器（使用空路径，会使用默认值）
	vm, err := NewVersionManager(db, "")
	if err != nil {
		t.Log("Note: VERSION file may not exist in test environment")
	}
	
	// 测试版本信息
	info := vm.GetVersionInfo()
	if info == nil {
		t.Fatal("Version info is nil")
	}
	
	t.Logf("App Version: %s", info.AppVersion)
	t.Logf("DB Version: %s", info.DBVersion)
	t.Logf("Latest Schema Version: %s", info.LatestSchemaVersion)
	
	// 测试版本比较
	compareTests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"001", "002", -1},
		{"002", "001", 1},
		{"005", "005", 0},
	}
	
	for _, tc := range compareTests {
		result := vm.CompareVersions(tc.v1, tc.v2)
		if result != tc.expected {
			t.Errorf("CompareVersions(%s, %s) = %d, expected %d", 
				tc.v1, tc.v2, result, tc.expected)
		}
	}
}

// TestSchemaValidator 测试结构验证器
func TestSchemaValidator(t *testing.T) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal("Failed to create test database:", err)
	}
	
	// 创建验证器
	validator := NewSchemaValidator(db, DatabaseTypeSQLite)
	if validator == nil {
		t.Fatal("Failed to create validator")
	}
	
	// 在空数据库上运行验证（应该发现所有表都缺失）
	result, err := validator.Validate()
	if err != nil {
		t.Fatal("Validation failed:", err)
	}
	
	if result.Valid {
		t.Error("Expected validation to fail on empty database")
	}
	
	if len(result.MissingTables) == 0 {
		t.Error("Expected missing tables on empty database")
	}
	
	t.Logf("Validation found %d missing tables (expected)", len(result.MissingTables))
	t.Logf("Critical issues: %d, Warnings: %d", result.CriticalIssues, result.WarningIssues)
}

// TestSchemaRepairer 测试修复器（干运行模式）
func TestSchemaRepairer(t *testing.T) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal("Failed to create test database:", err)
	}
	
	// 先验证
	validator := NewSchemaValidator(db, DatabaseTypeSQLite)
	validationResult, err := validator.Validate()
	if err != nil {
		t.Fatal("Validation failed:", err)
	}
	
	// 创建修复器（干运行模式）
	repairer := NewSchemaRepairer(db, DatabaseTypeSQLite, true)
	if repairer == nil {
		t.Fatal("Failed to create repairer")
	}
	
	// 执行修复（干运行）
	repairResult, err := repairer.Repair(validationResult)
	if err != nil {
		t.Fatal("Repair failed:", err)
	}
	
	if !repairResult.DryRun {
		t.Error("Expected dry run mode")
	}
	
	if len(repairResult.SQLStatements) == 0 {
		t.Error("Expected SQL statements to be generated")
	}
	
	t.Logf("Generated %d SQL statements in dry run mode", len(repairResult.SQLStatements))
	t.Logf("Tables to create: %d", len(repairResult.TablesCreated))
}

// TestMigrationRegistry 测试迁移注册表
func TestMigrationRegistry(t *testing.T) {
	// 测试注册表不为空
	if len(MigrationRegistry) == 0 {
		t.Fatal("Migration registry is empty")
	}
	
	t.Logf("Found %d migrations in registry", len(MigrationRegistry))
	
	// 测试迁移顺序
	if err := ValidateMigrationOrder(); err != nil {
		t.Error("Migration order validation failed:", err)
	}
	
	// 测试获取迁移
	testVersions := []string{"001", "010", "023"}
	for _, version := range testVersions {
		migration, err := GetMigrationByVersion(version)
		if err != nil {
			t.Errorf("Failed to get migration %s: %v", version, err)
			continue
		}
		
		if migration.Version != version {
			t.Errorf("Expected version %s, got %s", version, migration.Version)
		}
		
		t.Logf("✓ Migration %s: %s", migration.Version, migration.Name)
	}
	
	// 测试获取最新迁移
	latest := GetLatestMigration()
	if latest == nil {
		t.Fatal("Failed to get latest migration")
	}
	
	if latest.Version != "023" {
		t.Errorf("Expected latest version to be 023, got %s", latest.Version)
	}
	
	t.Logf("Latest migration: %s (%s)", latest.Version, latest.AppVersion)
	
	// 测试统计
	stats := GetMigrationStatistics()
	if stats["total"].(int) != len(MigrationRegistry) {
		t.Error("Statistics total does not match registry length")
	}
	
	t.Logf("Migration statistics: %v", stats)
}

// TestMigrationDependencies 测试迁移依赖
func TestMigrationDependencies(t *testing.T) {
	for _, migration := range MigrationRegistry {
		if len(migration.Dependencies) == 0 {
			continue
		}
		
		if err := ValidateMigrationDependencies(migration.Version); err != nil {
			t.Errorf("Dependency validation failed for %s: %v", migration.Version, err)
		} else {
			t.Logf("✓ Dependencies valid for %s: %v", migration.Version, migration.Dependencies)
		}
	}
}

// TestEndToEndMigration 端到端测试（创建表并验证）
func TestEndToEndMigration(t *testing.T) {
	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal("Failed to create test database:", err)
	}
	
	// 1. 初始验证（应该失败）
	validator := NewSchemaValidator(db, DatabaseTypeSQLite)
	initialResult, err := validator.Validate()
	if err != nil {
		t.Fatal("Initial validation failed:", err)
	}
	
	if initialResult.Valid {
		t.Error("Expected initial validation to fail")
	}
	
	t.Logf("Initial state: %d missing tables", len(initialResult.MissingTables))
	
	// 2. 执行修复（非干运行）
	repairer := NewSchemaRepairer(db, DatabaseTypeSQLite, false)
	repairResult, err := repairer.Repair(initialResult)
	if err != nil {
		t.Fatal("Repair failed:", err)
	}
	
	t.Logf("Repair created %d tables", len(repairResult.TablesCreated))
	
	// 3. 重新验证（应该通过）
	finalResult, err := validator.Validate()
	if err != nil {
		t.Fatal("Final validation failed:", err)
	}
	
	if !finalResult.Valid {
		t.Errorf("Expected final validation to pass, but got %d critical issues", 
			finalResult.CriticalIssues)
		
		// 打印详细错误
		for _, tableResult := range finalResult.TableResults {
			if len(tableResult.Issues) > 0 {
				t.Logf("Issues in table %s:", tableResult.TableName)
				for _, issue := range tableResult.Issues {
					t.Logf("  - %s: %s", issue.Type, issue.Message)
				}
			}
		}
	} else {
		t.Log("✅ End-to-end test passed: database created and validated successfully")
	}
}

