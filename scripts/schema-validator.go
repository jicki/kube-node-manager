package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 数据库结构验证工具
func main() {
	fmt.Println("🛡️  数据库结构验证工具")
	fmt.Println("==========================================")

	// 连接PostgreSQL
	dsn := "host=pgm-wz9lq79tmh67w5y4.pg.rds.aliyuncs.com port=5432 user=kube_node_mgr dbname=kube_node_mgr sslmode=disable password=3OBs4fb9CiHvMU5j"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("无法连接PostgreSQL:", err)
	}

	// 定义预期的表结构
	expectedSchemas := getExpectedSchemas()

	allIssues := []ValidationIssue{}

	fmt.Println("\n🔍 验证表结构...")

	for tableName, expectedFields := range expectedSchemas {
		fmt.Printf("\n📊 验证表: %s\n", tableName)

		issues := validateTable(db, tableName, expectedFields)
		allIssues = append(allIssues, issues...)

		if len(issues) == 0 {
			fmt.Printf("   ✅ 表 %s 结构完全正确\n", tableName)
		} else {
			fmt.Printf("   ⚠️  表 %s 发现 %d 个问题\n", tableName, len(issues))
		}
	}

	// 验证数据完整性
	fmt.Println("\n🔍 验证数据完整性...")
	dataIssues := validateDataIntegrity(db)
	allIssues = append(allIssues, dataIssues...)

	// 生成报告
	fmt.Println("\n📋 验证报告")
	fmt.Println("==========================================")

	if len(allIssues) == 0 {
		fmt.Println("🎉 恭喜！所有验证都通过了")
		fmt.Println("   数据库结构和数据完整性都符合预期")
	} else {
		fmt.Printf("⚠️  发现 %d 个问题需要注意:\n\n", len(allIssues))

		// 按严重性分组
		critical := []ValidationIssue{}
		warnings := []ValidationIssue{}
		info := []ValidationIssue{}

		for _, issue := range allIssues {
			switch issue.Severity {
			case "CRITICAL":
				critical = append(critical, issue)
			case "WARNING":
				warnings = append(warnings, issue)
			default:
				info = append(info, issue)
			}
		}

		if len(critical) > 0 {
			fmt.Printf("🚨 严重问题 (%d):\n", len(critical))
			for i, issue := range critical {
				fmt.Printf("  %d. [%s] %s\n", i+1, issue.Type, issue.Message)
				if issue.Solution != "" {
					fmt.Printf("     💡 解决方案: %s\n", issue.Solution)
				}
			}
			fmt.Println()
		}

		if len(warnings) > 0 {
			fmt.Printf("⚠️  警告 (%d):\n", len(warnings))
			for i, issue := range warnings {
				fmt.Printf("  %d. [%s] %s\n", i+1, issue.Type, issue.Message)
				if issue.Solution != "" {
					fmt.Printf("     💡 建议: %s\n", issue.Solution)
				}
			}
			fmt.Println()
		}

		if len(info) > 0 {
			fmt.Printf("ℹ️  信息 (%d):\n", len(info))
			for i, issue := range info {
				fmt.Printf("  %d. [%s] %s\n", i+1, issue.Type, issue.Message)
			}
		}
	}

	// 生成修复脚本建议
	if len(allIssues) > 0 {
		fmt.Println("\n🔧 修复建议")
		fmt.Println("==========================================")
		generateFixSuggestions(allIssues)
	}
}

type ValidationIssue struct {
	Type     string
	Table    string
	Field    string
	Message  string
	Severity string // CRITICAL, WARNING, INFO
	Solution string
}

type FieldRequirement struct {
	Type       string
	NotNull    bool
	HasDefault bool
	ForeignKey string
	JsonFormat bool
}

func getExpectedSchemas() map[string]map[string]FieldRequirement {
	return map[string]map[string]FieldRequirement{
		"users": {
			"id":           {Type: "bigint", NotNull: true},
			"username":     {Type: "text", NotNull: true},
			"email":        {Type: "text", NotNull: true},
			"password":     {Type: "text", NotNull: true},
			"role":         {Type: "text", NotNull: false, HasDefault: true},
			"status":       {Type: "text", NotNull: false, HasDefault: true},
			"is_ldap_user": {Type: "boolean", NotNull: false, HasDefault: true},
			"last_login":   {Type: "timestamp", NotNull: false},
			"created_at":   {Type: "timestamp", NotNull: false},
			"updated_at":   {Type: "timestamp", NotNull: false},
			"deleted_at":   {Type: "timestamp", NotNull: false},
		},
		"clusters": {
			"id":          {Type: "bigint", NotNull: true},
			"name":        {Type: "text", NotNull: true},
			"description": {Type: "text", NotNull: false},
			"kube_config": {Type: "text", NotNull: true},
			"status":      {Type: "text", NotNull: false, HasDefault: true},
			"version":     {Type: "text", NotNull: false},
			"node_count":  {Type: "bigint", NotNull: false, HasDefault: true},
			"last_sync":   {Type: "timestamp", NotNull: false},
			"created_by":  {Type: "integer", NotNull: true, ForeignKey: "users(id)"},
			"created_at":  {Type: "timestamp", NotNull: false},
			"updated_at":  {Type: "timestamp", NotNull: false},
			"deleted_at":  {Type: "timestamp", NotNull: false},
		},
		"label_templates": {
			"id":          {Type: "bigint", NotNull: true},
			"name":        {Type: "text", NotNull: true},
			"description": {Type: "text", NotNull: false},
			"labels":      {Type: "text", NotNull: true, JsonFormat: true},
			"created_by":  {Type: "bigint", NotNull: false, ForeignKey: "users(id)"},
			"created_at":  {Type: "timestamp", NotNull: false},
			"updated_at":  {Type: "timestamp", NotNull: false},
			"deleted_at":  {Type: "timestamp", NotNull: false},
		},
		"taint_templates": {
			"id":          {Type: "bigint", NotNull: true},
			"name":        {Type: "text", NotNull: true},
			"description": {Type: "text", NotNull: false},
			"taints":      {Type: "text", NotNull: true, JsonFormat: true},
			"created_by":  {Type: "bigint", NotNull: false, ForeignKey: "users(id)"},
			"created_at":  {Type: "timestamp", NotNull: false},
			"updated_at":  {Type: "timestamp", NotNull: false},
			"deleted_at":  {Type: "timestamp", NotNull: false},
		},
		"audit_logs": {
			"id":            {Type: "bigint", NotNull: true},
			"user_id":       {Type: "bigint", NotNull: true, ForeignKey: "users(id)"},
			"cluster_id":    {Type: "bigint", NotNull: false, ForeignKey: "clusters(id)"},
			"node_name":     {Type: "text", NotNull: false},
			"action":        {Type: "text", NotNull: true},
			"resource_type": {Type: "text", NotNull: true},
			"details":       {Type: "text", NotNull: false},
			"reason":        {Type: "text", NotNull: false},
			"status":        {Type: "text", NotNull: false, HasDefault: true},
			"error_msg":     {Type: "text", NotNull: false},
			"ip_address":    {Type: "text", NotNull: false},
			"user_agent":    {Type: "text", NotNull: false},
			"created_at":    {Type: "timestamp", NotNull: false},
		},
	}
}

func validateTable(db *gorm.DB, tableName string, expectedFields map[string]FieldRequirement) []ValidationIssue {
	issues := []ValidationIssue{}

	// 检查表是否存在
	var count int64
	result := db.Table(tableName).Count(&count)
	if result.Error != nil {
		issues = append(issues, ValidationIssue{
			Type:     "MISSING_TABLE",
			Table:    tableName,
			Message:  fmt.Sprintf("表 %s 不存在", tableName),
			Severity: "CRITICAL",
			Solution: fmt.Sprintf("需要创建表 %s", tableName),
		})
		return issues
	}

	// 获取实际的表结构
	actualFields := getActualTableStructure(db, tableName)

	// 检查每个预期字段
	for fieldName, expectedField := range expectedFields {
		actualField, exists := actualFields[fieldName]

		if !exists {
			issues = append(issues, ValidationIssue{
				Type:     "MISSING_FIELD",
				Table:    tableName,
				Field:    fieldName,
				Message:  fmt.Sprintf("表 %s 缺少字段 %s", tableName, fieldName),
				Severity: "CRITICAL",
				Solution: fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, fieldName),
			})
			continue
		}

		// 检查NOT NULL约束
		if expectedField.NotNull && actualField.IsNullable {
			issues = append(issues, ValidationIssue{
				Type:     "MISSING_NOT_NULL",
				Table:    tableName,
				Field:    fieldName,
				Message:  fmt.Sprintf("字段 %s.%s 应该有NOT NULL约束", tableName, fieldName),
				Severity: "WARNING",
				Solution: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", tableName, fieldName),
			})
		}

		// 检查JSON格式字段的数据
		if expectedField.JsonFormat {
			jsonIssues := validateJsonField(db, tableName, fieldName)
			issues = append(issues, jsonIssues...)
		}
	}

	// 检查是否有多余的字段
	for fieldName := range actualFields {
		if _, expected := expectedFields[fieldName]; !expected {
			issues = append(issues, ValidationIssue{
				Type:     "UNEXPECTED_FIELD",
				Table:    tableName,
				Field:    fieldName,
				Message:  fmt.Sprintf("表 %s 有未预期的字段 %s", tableName, fieldName),
				Severity: "INFO",
			})
		}
	}

	return issues
}

type ActualField struct {
	Name          string
	DataType      string
	IsNullable    bool
	ColumnDefault string
}

func getActualTableStructure(db *gorm.DB, tableName string) map[string]ActualField {
	fields := make(map[string]ActualField)

	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = ?
		ORDER BY ordinal_position
	`

	rows, err := db.Raw(query, tableName).Rows()
	if err != nil {
		log.Printf("查询表结构失败: %v", err)
		return fields
	}
	defer rows.Close()

	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault *string

		rows.Scan(&columnName, &dataType, &isNullable, &columnDefault)

		defaultValue := ""
		if columnDefault != nil {
			defaultValue = *columnDefault
		}

		fields[columnName] = ActualField{
			Name:          columnName,
			DataType:      dataType,
			IsNullable:    isNullable == "YES",
			ColumnDefault: defaultValue,
		}
	}

	return fields
}

func validateJsonField(db *gorm.DB, tableName, fieldName string) []ValidationIssue {
	issues := []ValidationIssue{}

	// 检查JSON字段是否为空或无效
	var invalidCount int64
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s 
		WHERE %s IS NULL OR %s = '' OR %s = '{}'
	`, tableName, fieldName, fieldName, fieldName)

	result := db.Raw(query).Scan(&invalidCount)
	if result.Error == nil && invalidCount > 0 {
		issues = append(issues, ValidationIssue{
			Type:     "INVALID_JSON_DATA",
			Table:    tableName,
			Field:    fieldName,
			Message:  fmt.Sprintf("字段 %s.%s 有 %d 个无效的JSON值", tableName, fieldName, invalidCount),
			Severity: "WARNING",
			Solution: "检查并修复JSON数据格式",
		})
	}

	return issues
}

func validateDataIntegrity(db *gorm.DB) []ValidationIssue {
	issues := []ValidationIssue{}

	// 验证外键约束
	fkChecks := []struct {
		Name     string
		Query    string
		ErrorMsg string
	}{
		{
			"users_clusters_fk",
			"SELECT COUNT(*) FROM clusters c LEFT JOIN users u ON c.created_by = u.id WHERE u.id IS NULL AND c.created_by IS NOT NULL",
			"集群表中存在无效的created_by引用",
		},
		{
			"audit_logs_users_fk",
			"SELECT COUNT(*) FROM audit_logs a LEFT JOIN users u ON a.user_id = u.id WHERE u.id IS NULL",
			"审计日志表中存在无效的user_id引用",
		},
		{
			"audit_logs_clusters_fk",
			"SELECT COUNT(*) FROM audit_logs a LEFT JOIN clusters c ON a.cluster_id = c.id WHERE c.id IS NULL AND a.cluster_id IS NOT NULL AND a.cluster_id > 0",
			"审计日志表中存在无效的cluster_id引用",
		},
	}

	for _, check := range fkChecks {
		var count int64
		result := db.Raw(check.Query).Scan(&count)
		if result.Error == nil && count > 0 {
			issues = append(issues, ValidationIssue{
				Type:     "FK_VIOLATION",
				Message:  fmt.Sprintf("%s (%d 个记录)", check.ErrorMsg, count),
				Severity: "CRITICAL",
				Solution: "修复外键引用或添加缺失的关联记录",
			})
		}
	}

	return issues
}

func generateFixSuggestions(issues []ValidationIssue) {
	// 生成SQL修复脚本
	sqlCommands := []string{}

	for _, issue := range issues {
		switch issue.Type {
		case "MISSING_FIELD":
			// 根据字段类型生成ADD COLUMN语句
			sqlCommands = append(sqlCommands,
				fmt.Sprintf("-- 添加缺失字段: %s.%s", issue.Table, issue.Field))
			sqlCommands = append(sqlCommands, issue.Solution+";")

		case "MISSING_NOT_NULL":
			sqlCommands = append(sqlCommands,
				fmt.Sprintf("-- 添加NOT NULL约束: %s.%s", issue.Table, issue.Field))
			sqlCommands = append(sqlCommands, issue.Solution+";")
		}
	}

	if len(sqlCommands) > 0 {
		fmt.Println("生成的修复SQL脚本:")
		fmt.Println("```sql")
		for _, cmd := range sqlCommands {
			fmt.Println(cmd)
		}
		fmt.Println("```")
		fmt.Println()
	}

	// 生成Go代码修复建议
	fmt.Println("Go代码修复建议:")
	fmt.Println("1. 更新迁移脚本中的数据模型定义")
	fmt.Println("2. 确保AutoMigrate使用正确的结构体")
	fmt.Println("3. 在数据迁移时正确处理字段映射")
	fmt.Println("4. 添加数据转换逻辑处理JSON字段")
}
