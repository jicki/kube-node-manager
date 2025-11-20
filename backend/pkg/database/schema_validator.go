package database

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

// SchemaValidator 数据库结构验证器
type SchemaValidator struct {
	db         *gorm.DB
	dbType     DatabaseType
	schemas    []TableSchema
	actualTables map[string]*ActualTableSchema
}

// ActualTableSchema 实际的表结构
type ActualTableSchema struct {
	Name    string
	Columns map[string]*ActualColumn
	Indexes map[string]*ActualIndex
}

// ActualColumn 实际的字段信息
type ActualColumn struct {
	Name         string
	Type         string
	Nullable     bool
	DefaultValue *string
	IsPrimaryKey bool
	IsUnique     bool
}

// ActualIndex 实际的索引信息
type ActualIndex struct {
	Name    string
	Columns []string
	Unique  bool
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid           bool
	TableResults    []TableValidationResult
	MissingTables   []string
	ExtraTables     []string
	TotalIssues     int
	CriticalIssues  int
	WarningIssues   int
}

// TableValidationResult 表验证结果
type TableValidationResult struct {
	TableName       string
	Exists          bool
	MissingColumns  []string
	ExtraColumns    []string
	TypeMismatches  []ColumnTypeMismatch
	MissingIndexes  []string
	ExtraIndexes    []string
	Issues          []ValidationIssue
}

// ColumnTypeMismatch 字段类型不匹配
type ColumnTypeMismatch struct {
	ColumnName   string
	ExpectedType string
	ActualType   string
	Severity     string // "critical" or "warning"
}

// ValidationIssue 验证问题
type ValidationIssue struct {
	Type     string // "missing_table", "missing_column", "type_mismatch", "missing_index", etc.
	Severity string // "critical", "warning", "info"
	Message  string
	Details  map[string]interface{}
}

// NewSchemaValidator 创建结构验证器
func NewSchemaValidator(db *gorm.DB, dbType DatabaseType) *SchemaValidator {
	return &SchemaValidator{
		db:           db,
		dbType:       dbType,
		schemas:      AllTableSchemas(),
		actualTables: make(map[string]*ActualTableSchema),
	}
}

// Validate 验证数据库结构
func (sv *SchemaValidator) Validate() (*ValidationResult, error) {
	log.Println("Starting database schema validation...")

	// 1. 加载实际的数据库结构
	if err := sv.loadActualSchema(); err != nil {
		return nil, fmt.Errorf("failed to load actual schema: %w", err)
	}

	result := &ValidationResult{
		Valid:        true,
		TableResults: []TableValidationResult{},
	}

	// 2. 验证每个表
	expectedTables := make(map[string]bool)
	for _, schema := range sv.schemas {
		expectedTables[schema.Name] = true
		
		tableResult := sv.validateTable(schema)
		result.TableResults = append(result.TableResults, tableResult)

		// 统计问题数量
		if !tableResult.Exists {
			result.MissingTables = append(result.MissingTables, schema.Name)
			result.CriticalIssues++
			result.Valid = false
		} else {
			for _, issue := range tableResult.Issues {
				result.TotalIssues++
				if issue.Severity == "critical" {
					result.CriticalIssues++
					result.Valid = false
				} else if issue.Severity == "warning" {
					result.WarningIssues++
				}
			}
		}
	}

	// 3. 检查额外的表（不在定义中的表）
	for tableName := range sv.actualTables {
		if !expectedTables[tableName] {
			// 忽略系统表和临时表
			if !sv.isSystemTable(tableName) {
				result.ExtraTables = append(result.ExtraTables, tableName)
			}
		}
	}

	log.Printf("Validation completed: %d issues found (%d critical, %d warnings)", 
		result.TotalIssues, result.CriticalIssues, result.WarningIssues)

	return result, nil
}

// loadActualSchema 加载实际的数据库结构
func (sv *SchemaValidator) loadActualSchema() error {
	// 获取所有表名
	tables, err := sv.getTableNames()
	if err != nil {
		return fmt.Errorf("failed to get table names: %w", err)
	}

	// 加载每个表的结构
	for _, tableName := range tables {
		actualTable := &ActualTableSchema{
			Name:    tableName,
			Columns: make(map[string]*ActualColumn),
			Indexes: make(map[string]*ActualIndex),
		}

		// 加载字段信息
		columns, err := sv.getTableColumns(tableName)
		if err != nil {
			log.Printf("Warning: Failed to get columns for table %s: %v", tableName, err)
			continue
		}
		for _, col := range columns {
			actualTable.Columns[col.Name] = col
		}

		// 加载索引信息
		indexes, err := sv.getTableIndexes(tableName)
		if err != nil {
			log.Printf("Warning: Failed to get indexes for table %s: %v", tableName, err)
		} else {
			for _, idx := range indexes {
				actualTable.Indexes[idx.Name] = idx
			}
		}

		sv.actualTables[tableName] = actualTable
	}

	log.Printf("Loaded actual schema: %d tables", len(sv.actualTables))
	return nil
}

// getTableNames 获取所有表名
func (sv *SchemaValidator) getTableNames() ([]string, error) {
	var tables []string

	if sv.dbType == DatabaseTypePostgreSQL {
		query := `
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_type = 'BASE TABLE'
			ORDER BY table_name
		`
		if err := sv.db.Raw(query).Scan(&tables).Error; err != nil {
			return nil, err
		}
	} else if sv.dbType == DatabaseTypeSQLite {
		query := `
			SELECT name 
			FROM sqlite_master 
			WHERE type='table' 
			AND name NOT LIKE 'sqlite_%'
			ORDER BY name
		`
		if err := sv.db.Raw(query).Scan(&tables).Error; err != nil {
			return nil, err
		}
	}

	return tables, nil
}

// getTableColumns 获取表的字段信息
func (sv *SchemaValidator) getTableColumns(tableName string) ([]*ActualColumn, error) {
	var columns []*ActualColumn

	if sv.dbType == DatabaseTypePostgreSQL {
		query := `
			SELECT 
				column_name,
				data_type,
				is_nullable,
				column_default
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = ?
			ORDER BY ordinal_position
		`
		
		type pgColumn struct {
			ColumnName    string
			DataType      string
			IsNullable    string
			ColumnDefault *string
		}
		
		var pgCols []pgColumn
		if err := sv.db.Raw(query, tableName).Scan(&pgCols).Error; err != nil {
			return nil, err
		}

		for _, col := range pgCols {
			columns = append(columns, &ActualColumn{
				Name:         col.ColumnName,
				Type:         col.DataType,
				Nullable:     col.IsNullable == "YES",
				DefaultValue: col.ColumnDefault,
			})
		}
	} else if sv.dbType == DatabaseTypeSQLite {
		// SQLite 使用 PRAGMA table_info
		query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
		
		type sqliteColumn struct {
			CID          int
			Name         string
			Type         string
			NotNull      int
			DefaultValue *string
			PK           int
		}
		
		var sqliteCols []sqliteColumn
		if err := sv.db.Raw(query).Scan(&sqliteCols).Error; err != nil {
			return nil, err
		}

		for _, col := range sqliteCols {
			columns = append(columns, &ActualColumn{
				Name:         col.Name,
				Type:         col.Type,
				Nullable:     col.NotNull == 0,
				DefaultValue: col.DefaultValue,
				IsPrimaryKey: col.PK > 0,
			})
		}
	}

	return columns, nil
}

// getTableIndexes 获取表的索引信息
func (sv *SchemaValidator) getTableIndexes(tableName string) ([]*ActualIndex, error) {
	var indexes []*ActualIndex

	if sv.dbType == DatabaseTypePostgreSQL {
		query := `
			SELECT 
				i.relname AS index_name,
				ix.indisunique AS is_unique,
				ARRAY_AGG(a.attname ORDER BY a.attnum) AS column_names
			FROM pg_class t
			JOIN pg_index ix ON t.oid = ix.indrelid
			JOIN pg_class i ON i.oid = ix.indexrelid
			JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
			WHERE t.relname = $1 AND t.relkind = 'r'
			GROUP BY i.relname, ix.indisunique
			ORDER BY i.relname
		`
		
		type pgIndex struct {
			IndexName   string
			IsUnique    bool
			ColumnNames string
		}
		
		var pgIdxs []pgIndex
		if err := sv.db.Raw(query, tableName).Scan(&pgIdxs).Error; err != nil {
			return nil, err
		}

		for _, idx := range pgIdxs {
			// Parse column names (PostgreSQL array format: {col1,col2})
			colNames := strings.Trim(idx.ColumnNames, "{}")
			columns := strings.Split(colNames, ",")
			
			indexes = append(indexes, &ActualIndex{
				Name:    idx.IndexName,
				Unique:  idx.IsUnique,
				Columns: columns,
			})
		}
	} else if sv.dbType == DatabaseTypeSQLite {
		// SQLite 使用 PRAGMA index_list
		query := fmt.Sprintf("PRAGMA index_list(%s)", tableName)
		
		type sqliteIndexList struct {
			Seq    int
			Name   string
			Unique int
			Origin string
			Partial int
		}
		
		var idxList []sqliteIndexList
		if err := sv.db.Raw(query).Scan(&idxList).Error; err != nil {
			return nil, err
		}

		for _, idx := range idxList {
			// 获取索引的列信息
			infoQuery := fmt.Sprintf("PRAGMA index_info(%s)", idx.Name)
			
			type sqliteIndexInfo struct {
				Seqno int
				CID   int
				Name  string
			}
			
			var infoList []sqliteIndexInfo
			if err := sv.db.Raw(infoQuery).Scan(&infoList).Error; err != nil {
				log.Printf("Warning: Failed to get index info for %s: %v", idx.Name, err)
				continue
			}

			columns := make([]string, len(infoList))
			for _, info := range infoList {
				columns[info.Seqno] = info.Name
			}

			indexes = append(indexes, &ActualIndex{
				Name:    idx.Name,
				Unique:  idx.Unique == 1,
				Columns: columns,
			})
		}
	}

	return indexes, nil
}

// validateTable 验证单个表
func (sv *SchemaValidator) validateTable(schema TableSchema) TableValidationResult {
	result := TableValidationResult{
		TableName:      schema.Name,
		Exists:         false,
		MissingColumns: []string{},
		ExtraColumns:   []string{},
		TypeMismatches: []ColumnTypeMismatch{},
		MissingIndexes: []string{},
		ExtraIndexes:   []string{},
		Issues:         []ValidationIssue{},
	}

	// 检查表是否存在
	actualTable, exists := sv.actualTables[schema.Name]
	if !exists {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:     "missing_table",
			Severity: "critical",
			Message:  fmt.Sprintf("Table '%s' does not exist", schema.Name),
		})
		return result
	}

	result.Exists = true

	// 验证字段
	expectedColumns := make(map[string]bool)
	for _, expectedCol := range schema.Columns {
		expectedColumns[expectedCol.Name] = true
		
		actualCol, colExists := actualTable.Columns[expectedCol.Name]
		if !colExists {
			result.MissingColumns = append(result.MissingColumns, expectedCol.Name)
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "missing_column",
				Severity: "critical",
				Message:  fmt.Sprintf("Column '%s.%s' is missing", schema.Name, expectedCol.Name),
				Details: map[string]interface{}{
					"column": expectedCol.Name,
					"expected_type": expectedCol.GetType(sv.dbType),
				},
			})
			continue
		}

		// 验证字段类型
		expectedType := sv.normalizeType(expectedCol.GetType(sv.dbType))
		actualType := sv.normalizeType(actualCol.Type)
		
		if !sv.typesMatch(expectedType, actualType) {
			severity := "warning"
			if expectedCol.PrimaryKey || !expectedCol.Nullable {
				severity = "critical"
			}
			
			result.TypeMismatches = append(result.TypeMismatches, ColumnTypeMismatch{
				ColumnName:   expectedCol.Name,
				ExpectedType: expectedType,
				ActualType:   actualType,
				Severity:     severity,
			})
			
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "type_mismatch",
				Severity: severity,
				Message:  fmt.Sprintf("Column '%s.%s' type mismatch: expected %s, got %s", 
					schema.Name, expectedCol.Name, expectedType, actualType),
				Details: map[string]interface{}{
					"column": expectedCol.Name,
					"expected": expectedType,
					"actual": actualType,
				},
			})
		}
	}

	// 检查额外的字段
	for colName := range actualTable.Columns {
		if !expectedColumns[colName] {
			result.ExtraColumns = append(result.ExtraColumns, colName)
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "extra_column",
				Severity: "info",
				Message:  fmt.Sprintf("Column '%s.%s' exists but not in schema definition", schema.Name, colName),
			})
		}
	}

	// 验证索引
	expectedIndexes := make(map[string]bool)
	for _, expectedIdx := range schema.Indexes {
		expectedIndexes[expectedIdx.Name] = true
		
		_, idxExists := actualTable.Indexes[expectedIdx.Name]
		if !idxExists {
			result.MissingIndexes = append(result.MissingIndexes, expectedIdx.Name)
			result.Issues = append(result.Issues, ValidationIssue{
				Type:     "missing_index",
				Severity: "warning",
				Message:  fmt.Sprintf("Index '%s' is missing on table '%s'", expectedIdx.Name, schema.Name),
				Details: map[string]interface{}{
					"index": expectedIdx.Name,
					"columns": expectedIdx.Columns,
				},
			})
		}
	}

	// 检查额外的索引
	for idxName := range actualTable.Indexes {
		if !expectedIndexes[idxName] && !strings.HasSuffix(idxName, "_pkey") {
			result.ExtraIndexes = append(result.ExtraIndexes, idxName)
		}
	}

	return result
}

// normalizeType 规范化类型名称（用于比较）
func (sv *SchemaValidator) normalizeType(typeName string) string {
	typeName = strings.ToUpper(strings.TrimSpace(typeName))
	
	// 移除长度和精度信息
	if idx := strings.Index(typeName, "("); idx > 0 {
		typeName = typeName[:idx]
	}
	
	// PostgreSQL 类型映射
	typeMap := map[string]string{
		"CHARACTER VARYING": "VARCHAR",
		"TIMESTAMP WITHOUT TIME ZONE": "TIMESTAMP",
		"TIMESTAMP WITH TIME ZONE": "TIMESTAMPTZ",
		"DOUBLE PRECISION": "DOUBLE",
		"SERIAL": "INTEGER",
		"BIGSERIAL": "BIGINT",
	}
	
	if mapped, exists := typeMap[typeName]; exists {
		return mapped
	}
	
	return typeName
}

// typesMatch 判断两个类型是否匹配
func (sv *SchemaValidator) typesMatch(expected, actual string) bool {
	// 精确匹配
	if expected == actual {
		return true
	}
	
	// 兼容的类型匹配
	compatibleTypes := map[string][]string{
		"INTEGER":   {"INT", "BIGINT", "SMALLINT", "SERIAL", "BIGSERIAL"},
		"INT":       {"INTEGER", "BIGINT", "SMALLINT"},
		"BIGINT":    {"INTEGER", "INT", "BIGSERIAL"},
		"VARCHAR":   {"TEXT", "CHARACTER VARYING", "STRING"},
		"TEXT":      {"VARCHAR", "STRING", "CLOB"},
		"TIMESTAMP": {"DATETIME", "TIMESTAMPTZ"},
		"DATETIME":  {"TIMESTAMP", "TIMESTAMPTZ"},
		"BOOLEAN":   {"BOOL"},
		"BOOL":      {"BOOLEAN"},
		"DOUBLE":    {"REAL", "FLOAT", "NUMERIC"},
		"REAL":      {"DOUBLE", "FLOAT"},
		"BYTEA":     {"BLOB", "BINARY"},
		"JSONB":     {"JSON", "TEXT"},
		"JSON":      {"JSONB", "TEXT"},
	}
	
	// 检查是否在兼容列表中
	if compatible, exists := compatibleTypes[expected]; exists {
		for _, t := range compatible {
			if actual == t {
				return true
			}
		}
	}
	
	// 反向检查
	if compatible, exists := compatibleTypes[actual]; exists {
		for _, t := range compatible {
			if expected == t {
				return true
			}
		}
	}
	
	return false
}

// isSystemTable 判断是否是系统表
func (sv *SchemaValidator) isSystemTable(tableName string) bool {
	systemTables := []string{
		"pg_", "sql_", "sqlite_", "information_schema",
	}
	
	for _, prefix := range systemTables {
		if strings.HasPrefix(tableName, prefix) {
			return true
		}
	}
	
	return false
}

// PrintValidationResult 打印验证结果
func (sv *SchemaValidator) PrintValidationResult(result *ValidationResult) {
	fmt.Println("\n=== Database Schema Validation Result ===")
	
	if result.Valid {
		fmt.Println("✅ Database schema is valid")
	} else {
		fmt.Printf("❌ Database schema validation failed\n")
		fmt.Printf("   Critical Issues: %d\n", result.CriticalIssues)
		fmt.Printf("   Warnings: %d\n", result.WarningIssues)
		fmt.Printf("   Total Issues: %d\n", result.TotalIssues)
	}
	
	if len(result.MissingTables) > 0 {
		fmt.Printf("\nMissing Tables (%d):\n", len(result.MissingTables))
		for _, table := range result.MissingTables {
			fmt.Printf("  - %s\n", table)
		}
	}
	
	if len(result.ExtraTables) > 0 {
		fmt.Printf("\nExtra Tables (not in schema definition) (%d):\n", len(result.ExtraTables))
		for _, table := range result.ExtraTables {
			fmt.Printf("  - %s\n", table)
		}
	}
	
	// 打印每个表的详细问题
	for _, tableResult := range result.TableResults {
		if len(tableResult.Issues) == 0 {
			continue
		}
		
		fmt.Printf("\nTable: %s\n", tableResult.TableName)
		
		if len(tableResult.MissingColumns) > 0 {
			fmt.Printf("  Missing Columns: %v\n", tableResult.MissingColumns)
		}
		
		if len(tableResult.TypeMismatches) > 0 {
			fmt.Println("  Type Mismatches:")
			for _, mismatch := range tableResult.TypeMismatches {
				fmt.Printf("    - %s: expected %s, got %s [%s]\n", 
					mismatch.ColumnName, mismatch.ExpectedType, mismatch.ActualType, mismatch.Severity)
			}
		}
		
		if len(tableResult.MissingIndexes) > 0 {
			fmt.Printf("  Missing Indexes: %v\n", tableResult.MissingIndexes)
		}
	}
	
	fmt.Println()
}

// GetRepairSuggestions 获取修复建议
func (sv *SchemaValidator) GetRepairSuggestions(result *ValidationResult) []string {
	suggestions := []string{}
	
	if len(result.MissingTables) > 0 {
		suggestions = append(suggestions, "Run 'migrate repair' to create missing tables")
	}
	
	if result.CriticalIssues > 0 {
		suggestions = append(suggestions, "Critical issues detected. Database repair is required.")
	}
	
	if result.WarningIssues > 0 {
		suggestions = append(suggestions, "Warning issues detected. Consider running 'migrate repair --warnings'")
	}
	
	return suggestions
}

