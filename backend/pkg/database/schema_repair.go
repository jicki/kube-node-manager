package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SchemaRepairer 数据库结构修复器
type SchemaRepairer struct {
	db         *gorm.DB
	dbType     DatabaseType
	validator  *SchemaValidator
	dryRun     bool
	repairLogs []RepairLog
}

// RepairLog 修复日志
type RepairLog struct {
	Timestamp time.Time
	Action    string
	SQL       string
	Success   bool
	Error     string
}

// RepairResult 修复结果
type RepairResult struct {
	Success           bool
	TablesCreated     []string
	ColumnsAdded      []string
	IndexesCreated    []string
	TypesFixed        []string
	Errors            []string
	SQLStatements     []string
	RepairLogs        []RepairLog
	DryRun            bool
}

// NewSchemaRepairer 创建修复器
func NewSchemaRepairer(db *gorm.DB, dbType DatabaseType, dryRun bool) *SchemaRepairer {
	return &SchemaRepairer{
		db:         db,
		dbType:     dbType,
		validator:  NewSchemaValidator(db, dbType),
		dryRun:     dryRun,
		repairLogs: []RepairLog{},
	}
}

// Repair 执行修复
func (sr *SchemaRepairer) Repair(validationResult *ValidationResult) (*RepairResult, error) {
	log.Println("Starting database schema repair...")
	
	if sr.dryRun {
		log.Println("Running in DRY RUN mode - no changes will be applied")
	}

	result := &RepairResult{
		Success:        true,
		TablesCreated:  []string{},
		ColumnsAdded:   []string{},
		IndexesCreated: []string{},
		TypesFixed:     []string{},
		Errors:         []string{},
		SQLStatements:  []string{},
		DryRun:         sr.dryRun,
	}

	// 1. 创建缺失的表
	for _, tableName := range validationResult.MissingTables {
		if err := sr.createTable(tableName, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create table %s: %v", tableName, err))
			result.Success = false
		} else {
			result.TablesCreated = append(result.TablesCreated, tableName)
		}
	}

	// 2. 修复现有表的问题
	for _, tableResult := range validationResult.TableResults {
		if !tableResult.Exists {
			continue // 已经在步骤1处理
		}

		// 添加缺失的字段
		for _, columnName := range tableResult.MissingColumns {
			if err := sr.addColumn(tableResult.TableName, columnName, result); err != nil {
				result.Errors = append(result.Errors, 
					fmt.Sprintf("Failed to add column %s.%s: %v", tableResult.TableName, columnName, err))
				result.Success = false
			} else {
				result.ColumnsAdded = append(result.ColumnsAdded, 
					fmt.Sprintf("%s.%s", tableResult.TableName, columnName))
			}
		}

		// 创建缺失的索引
		for _, indexName := range tableResult.MissingIndexes {
			if err := sr.createIndex(tableResult.TableName, indexName, result); err != nil {
				result.Errors = append(result.Errors, 
					fmt.Sprintf("Failed to create index %s on %s: %v", indexName, tableResult.TableName, err))
				result.Success = false
			} else {
				result.IndexesCreated = append(result.IndexesCreated, 
					fmt.Sprintf("%s.%s", tableResult.TableName, indexName))
			}
		}

		// 修复类型不匹配（仅处理 critical 级别）
		for _, mismatch := range tableResult.TypeMismatches {
			if mismatch.Severity == "critical" {
				if err := sr.fixColumnType(tableResult.TableName, mismatch, result); err != nil {
					result.Errors = append(result.Errors, 
						fmt.Sprintf("Failed to fix column type %s.%s: %v", tableResult.TableName, mismatch.ColumnName, err))
					// 类型修复失败不设置 Success = false，因为这可能是兼容的
				} else {
					result.TypesFixed = append(result.TypesFixed, 
						fmt.Sprintf("%s.%s: %s -> %s", tableResult.TableName, mismatch.ColumnName, 
							mismatch.ActualType, mismatch.ExpectedType))
				}
			}
		}
	}

	// 将修复日志添加到结果
	result.RepairLogs = sr.repairLogs

	if sr.dryRun {
		log.Printf("DRY RUN completed. Generated %d SQL statements", len(result.SQLStatements))
	} else {
		log.Printf("Repair completed. Success: %v, Errors: %d", result.Success, len(result.Errors))
	}

	return result, nil
}

// createTable 创建表
func (sr *SchemaRepairer) createTable(tableName string, result *RepairResult) error {
	schema, err := GetTableSchema(tableName)
	if err != nil {
		return err
	}

	sql := sr.generateCreateTableSQL(*schema)
	result.SQLStatements = append(result.SQLStatements, sql)

	if !sr.dryRun {
		if err := sr.executeSQL(sql, "CREATE TABLE "+tableName); err != nil {
			return err
		}
	}

	log.Printf("Created table: %s (dry_run: %v)", tableName, sr.dryRun)
	return nil
}

// generateCreateTableSQL 生成创建表的 SQL
func (sr *SchemaRepairer) generateCreateTableSQL(schema TableSchema) string {
	var sql strings.Builder
	
	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", schema.Name))

	// 生成字段定义
	for i, col := range schema.Columns {
		if i > 0 {
			sql.WriteString(",\n")
		}
		sql.WriteString("  ")
		sql.WriteString(sr.generateColumnDefinition(col))
	}

	// 添加主键约束
	primaryKeys := []string{}
	for _, col := range schema.Columns {
		if col.PrimaryKey {
			primaryKeys = append(primaryKeys, col.Name)
		}
	}
	if len(primaryKeys) > 0 {
		sql.WriteString(",\n  PRIMARY KEY (")
		sql.WriteString(strings.Join(primaryKeys, ", "))
		sql.WriteString(")")
	}

	sql.WriteString("\n)")

	return sql.String()
}

// generateColumnDefinition 生成字段定义
func (sr *SchemaRepairer) generateColumnDefinition(col ColumnDefinition) string {
	var def strings.Builder

	def.WriteString(col.Name)
	def.WriteString(" ")
	def.WriteString(col.GetType(sr.dbType))

	if col.PrimaryKey && sr.dbType == DatabaseTypeSQLite {
		def.WriteString(" PRIMARY KEY")
		if col.AutoIncr {
			def.WriteString(" AUTOINCREMENT")
		}
	}

	if !col.Nullable && !col.PrimaryKey {
		def.WriteString(" NOT NULL")
	}

	if col.DefaultValue != nil {
		def.WriteString(" DEFAULT ")
		// 判断是否需要引号
		defaultVal := *col.DefaultValue
		if sr.needsQuotes(defaultVal, col.Type) {
			def.WriteString("'")
			def.WriteString(defaultVal)
			def.WriteString("'")
		} else {
			def.WriteString(defaultVal)
		}
	}

	if col.Unique && !col.PrimaryKey {
		def.WriteString(" UNIQUE")
	}

	return def.String()
}

// needsQuotes 判断默认值是否需要引号
func (sr *SchemaRepairer) needsQuotes(value, colType string) bool {
	// 数字类型不需要引号
	numericTypes := []string{"INTEGER", "INT", "BIGINT", "SMALLINT", "REAL", "DOUBLE", "NUMERIC", "DECIMAL"}
	for _, t := range numericTypes {
		if strings.Contains(strings.ToUpper(colType), t) {
			return false
		}
	}
	
	// 布尔值不需要引号
	if strings.ToLower(value) == "true" || strings.ToLower(value) == "false" {
		return false
	}
	
	// NULL 不需要引号
	if strings.ToUpper(value) == "NULL" {
		return false
	}
	
	// 函数调用不需要引号
	if strings.Contains(value, "(") {
		return false
	}
	
	return true
}

// addColumn 添加字段
func (sr *SchemaRepairer) addColumn(tableName, columnName string, result *RepairResult) error {
	schema, err := GetTableSchema(tableName)
	if err != nil {
		return err
	}

	// 找到字段定义
	var colDef *ColumnDefinition
	for _, col := range schema.Columns {
		if col.Name == columnName {
			colDef = &col
			break
		}
	}

	if colDef == nil {
		return fmt.Errorf("column definition not found: %s.%s", tableName, columnName)
	}

	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", 
		tableName, sr.generateColumnDefinition(*colDef))
	
	result.SQLStatements = append(result.SQLStatements, sql)

	if !sr.dryRun {
		if err := sr.executeSQL(sql, "ADD COLUMN "+tableName+"."+columnName); err != nil {
			return err
		}
	}

	log.Printf("Added column: %s.%s (dry_run: %v)", tableName, columnName, sr.dryRun)
	return nil
}

// createIndex 创建索引
func (sr *SchemaRepairer) createIndex(tableName, indexName string, result *RepairResult) error {
	schema, err := GetTableSchema(tableName)
	if err != nil {
		return err
	}

	// 找到索引定义
	var indexDef *IndexDefinition
	for _, idx := range schema.Indexes {
		if idx.Name == indexName {
			indexDef = &idx
			break
		}
	}

	if indexDef == nil {
		return fmt.Errorf("index definition not found: %s.%s", tableName, indexName)
	}

	sql := sr.generateCreateIndexSQL(tableName, *indexDef)
	result.SQLStatements = append(result.SQLStatements, sql)

	if !sr.dryRun {
		if err := sr.executeSQL(sql, "CREATE INDEX "+indexName); err != nil {
			return err
		}
	}

	log.Printf("Created index: %s.%s (dry_run: %v)", tableName, indexName, sr.dryRun)
	return nil
}

// generateCreateIndexSQL 生成创建索引的 SQL
func (sr *SchemaRepairer) generateCreateIndexSQL(tableName string, index IndexDefinition) string {
	var sql strings.Builder

	sql.WriteString("CREATE ")
	if index.Unique {
		sql.WriteString("UNIQUE ")
	}
	sql.WriteString("INDEX ")
	
	// PostgreSQL 使用 IF NOT EXISTS
	if sr.dbType == DatabaseTypePostgreSQL {
		sql.WriteString("IF NOT EXISTS ")
	}
	
	sql.WriteString(index.Name)
	sql.WriteString(" ON ")
	sql.WriteString(tableName)
	sql.WriteString(" (")
	sql.WriteString(strings.Join(index.Columns, ", "))
	sql.WriteString(")")

	return sql.String()
}

// fixColumnType 修复字段类型
func (sr *SchemaRepairer) fixColumnType(tableName string, mismatch ColumnTypeMismatch, result *RepairResult) error {
	// SQLite 不支持直接修改字段类型，需要重建表
	if sr.dbType == DatabaseTypeSQLite {
		log.Printf("Warning: SQLite does not support ALTER COLUMN TYPE. Skipping type fix for %s.%s", 
			tableName, mismatch.ColumnName)
		return nil
	}

	// PostgreSQL 支持 ALTER COLUMN TYPE
	sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", 
		tableName, mismatch.ColumnName, mismatch.ExpectedType)
	
	result.SQLStatements = append(result.SQLStatements, sql)

	if !sr.dryRun {
		if err := sr.executeSQL(sql, "ALTER COLUMN TYPE "+tableName+"."+mismatch.ColumnName); err != nil {
			return err
		}
	}

	log.Printf("Fixed column type: %s.%s (dry_run: %v)", tableName, mismatch.ColumnName, sr.dryRun)
	return nil
}

// executeSQL 执行 SQL 语句
func (sr *SchemaRepairer) executeSQL(sql, action string) error {
	startTime := time.Now()
	
	err := sr.db.Exec(sql).Error
	
	repairLog := RepairLog{
		Timestamp: startTime,
		Action:    action,
		SQL:       sql,
		Success:   err == nil,
	}
	
	if err != nil {
		repairLog.Error = err.Error()
	}
	
	sr.repairLogs = append(sr.repairLogs, repairLog)
	
	return err
}

// PrintRepairResult 打印修复结果
func (sr *SchemaRepairer) PrintRepairResult(result *RepairResult) {
	fmt.Println("\n=== Database Schema Repair Result ===")
	
	if result.DryRun {
		fmt.Println("🔍 DRY RUN MODE - No changes were applied")
	}
	
	if result.Success {
		fmt.Println("✅ Repair completed successfully")
	} else {
		fmt.Printf("⚠️  Repair completed with %d error(s)\n", len(result.Errors))
	}
	
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Tables Created:   %d\n", len(result.TablesCreated))
	fmt.Printf("  Columns Added:    %d\n", len(result.ColumnsAdded))
	fmt.Printf("  Indexes Created:  %d\n", len(result.IndexesCreated))
	fmt.Printf("  Types Fixed:      %d\n", len(result.TypesFixed))
	fmt.Printf("  Errors:           %d\n", len(result.Errors))
	fmt.Printf("  SQL Statements:   %d\n", len(result.SQLStatements))
	
	if len(result.TablesCreated) > 0 {
		fmt.Println("\nTables Created:")
		for _, table := range result.TablesCreated {
			fmt.Printf("  ✓ %s\n", table)
		}
	}
	
	if len(result.ColumnsAdded) > 0 {
		fmt.Println("\nColumns Added:")
		for _, col := range result.ColumnsAdded {
			fmt.Printf("  ✓ %s\n", col)
		}
	}
	
	if len(result.IndexesCreated) > 0 {
		fmt.Println("\nIndexes Created:")
		for _, idx := range result.IndexesCreated {
			fmt.Printf("  ✓ %s\n", idx)
		}
	}
	
	if len(result.TypesFixed) > 0 {
		fmt.Println("\nColumn Types Fixed:")
		for _, fix := range result.TypesFixed {
			fmt.Printf("  ✓ %s\n", fix)
		}
	}
	
	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, errMsg := range result.Errors {
			fmt.Printf("  ✗ %s\n", errMsg)
		}
	}
	
	if result.DryRun && len(result.SQLStatements) > 0 {
		fmt.Println("\nGenerated SQL Statements:")
		for i, sql := range result.SQLStatements {
			fmt.Printf("\n-- Statement %d:\n%s;\n", i+1, sql)
		}
	}
	
	fmt.Println()
}

// ValidateAndRepair 验证并修复数据库结构
func ValidateAndRepair(db *gorm.DB, dbType DatabaseType, dryRun bool) error {
	// 1. 创建验证器并验证
	validator := NewSchemaValidator(db, dbType)
	validationResult, err := validator.Validate()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// 打印验证结果
	validator.PrintValidationResult(validationResult)

	// 2. 如果验证通过，无需修复
	if validationResult.Valid {
		log.Println("Database schema is valid. No repair needed.")
		return nil
	}

	// 3. 创建修复器并修复
	repairer := NewSchemaRepairer(db, dbType, dryRun)
	repairResult, err := repairer.Repair(validationResult)
	if err != nil {
		return fmt.Errorf("repair failed: %w", err)
	}

	// 打印修复结果
	repairer.PrintRepairResult(repairResult)

	// 4. 如果不是 dry run 且修复成功，重新验证
	if !dryRun && repairResult.Success {
		log.Println("\nRe-validating database schema after repair...")
		newValidationResult, err := validator.Validate()
		if err != nil {
			return fmt.Errorf("re-validation failed: %w", err)
		}

		if newValidationResult.Valid {
			log.Println("✅ Database schema is now valid after repair")
		} else {
			log.Printf("⚠️  Some issues remain after repair (%d critical, %d warnings)", 
				newValidationResult.CriticalIssues, newValidationResult.WarningIssues)
		}
	}

	return nil
}

// GenerateRepairSQL 仅生成修复 SQL，不执行
func GenerateRepairSQL(db *gorm.DB, dbType DatabaseType) ([]string, error) {
	validator := NewSchemaValidator(db, dbType)
	validationResult, err := validator.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if validationResult.Valid {
		return []string{}, nil
	}

	repairer := NewSchemaRepairer(db, dbType, true) // dry run = true
	repairResult, err := repairer.Repair(validationResult)
	if err != nil {
		return nil, fmt.Errorf("repair generation failed: %w", err)
	}

	return repairResult.SQLStatements, nil
}

