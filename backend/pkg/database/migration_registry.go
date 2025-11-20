package database

import (
	"fmt"
	"strings"
	"time"
)

// MigrationInfo 迁移信息
type MigrationInfo struct {
	Version     string    // 版本号（001, 002...023）
	Name        string    // 迁移名称
	FileName    string    // SQL 文件名
	Description string    // 描述
	AppVersion  string    // 对应的应用版本
	Category    string    // 分类（索引/功能/修复）
	CreatedAt   time.Time // 创建时间
	Dependencies []string  // 依赖的迁移版本
}

// MigrationRegistry 迁移注册表
var MigrationRegistry = []MigrationInfo{
	{
		Version:     "001",
		Name:        "Add Anomaly Indexes",
		FileName:    "001_add_anomaly_indexes.sql",
		Description: "为异常记录表添加性能优化索引，包括集群+时间+状态等复合索引",
		AppVersion:  "v2.24.0",
		Category:    "索引优化",
		CreatedAt:   parseTime("2024-01-15"),
	},
	{
		Version:     "002",
		Name:        "Add Anomaly Analytics",
		FileName:    "002_add_anomaly_analytics.sql",
		Description: "添加异常分析相关的数据库视图和聚合函数",
		AppVersion:  "v2.25.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-02-01"),
		Dependencies: []string{"001"},
	},
	{
		Version:     "003",
		Name:        "Performance Indexes",
		FileName:    "003_performance_indexes.sql",
		Description: "为核心表添加性能索引，优化查询速度",
		AppVersion:  "v2.25.0",
		Category:    "索引优化",
		CreatedAt:   parseTime("2024-02-10"),
	},
	{
		Version:     "004",
		Name:        "Fix Ansible Foreign Keys (Quick)",
		FileName:    "004_fix_ansible_foreign_keys_quick.sql",
		Description: "快速修复 Ansible 相关表的外键约束问题",
		AppVersion:  "v2.26.0",
		Category:    "问题修复",
		CreatedAt:   parseTime("2024-02-20"),
	},
	{
		Version:     "005",
		Name:        "Cleanup Old Unique Indexes",
		FileName:    "005_cleanup_old_unique_indexes.sql",
		Description: "清理旧的唯一索引，避免冲突",
		AppVersion:  "v2.26.0",
		Category:    "问题修复",
		CreatedAt:   parseTime("2024-02-25"),
		Dependencies: []string{"004"},
	},
	{
		Version:     "006",
		Name:        "Ensure Soft Delete Indexes",
		FileName:    "006_ensure_soft_delete_indexes.sql",
		Description: "确保软删除相关的索引正确创建",
		AppVersion:  "v2.26.0",
		Category:    "索引优化",
		CreatedAt:   parseTime("2024-03-01"),
	},
	{
		Version:     "007",
		Name:        "Add Ansible Schedules",
		FileName:    "007_add_ansible_schedules.sql",
		Description: "添加 Ansible 定时任务调度功能",
		AppVersion:  "v2.27.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-03-10"),
	},
	{
		Version:     "008",
		Name:        "Add Retry and Environment Fields",
		FileName:    "008_add_retry_and_environment_fields.sql",
		Description: "为 Ansible 任务添加重试策略和环境标签字段",
		AppVersion:  "v2.27.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-03-15"),
		Dependencies: []string{"007"},
	},
	{
		Version:     "009",
		Name:        "Add Dry Run Field",
		FileName:    "009_add_dry_run_field.sql",
		Description: "添加 Dry Run 模式支持，允许检查模式执行",
		AppVersion:  "v2.28.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-04-01"),
	},
	{
		Version:     "010",
		Name:        "Add Batch Execution Fields",
		FileName:    "010_add_batch_execution_fields.sql",
		Description: "添加分批执行相关字段，支持批量操作",
		AppVersion:  "v2.28.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-04-10"),
	},
	{
		Version:     "011",
		Name:        "Add Favorites and History",
		FileName:    "011_add_favorites_and_history.sql",
		Description: "添加收藏和历史记录功能",
		AppVersion:  "v2.29.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-05-01"),
	},
	{
		Version:     "012",
		Name:        "Add Template Required Vars",
		FileName:    "012_add_template_required_vars.sql",
		Description: "为模板添加必需变量列表字段",
		AppVersion:  "v2.29.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-05-10"),
	},
	{
		Version:     "013",
		Name:        "Add Preflight Checks",
		FileName:    "013_add_preflight_checks.sql",
		Description: "添加前置检查功能，在执行前验证环境",
		AppVersion:  "v2.30.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-06-01"),
	},
	{
		Version:     "014",
		Name:        "Add Task Timeout",
		FileName:    "014_add_task_timeout.sql",
		Description: "添加任务超时控制字段",
		AppVersion:  "v2.30.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-06-10"),
	},
	{
		Version:     "015",
		Name:        "Add Task Priority",
		FileName:    "015_add_task_priority.sql",
		Description: "添加任务优先级支持",
		AppVersion:  "v2.30.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-06-15"),
	},
	{
		Version:     "016",
		Name:        "Add Task Tags",
		FileName:    "016_add_task_tags.sql",
		Description: "添加任务标签功能，支持任务分类和筛选",
		AppVersion:  "v2.31.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-07-01"),
	},
	{
		Version:     "017",
		Name:        "Add Execution Timeline",
		FileName:    "017_add_execution_timeline.sql",
		Description: "添加执行时间线，记录任务执行的各个阶段",
		AppVersion:  "v2.31.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-07-10"),
	},
	{
		Version:     "018",
		Name:        "Fix Favorites Foreign Keys",
		FileName:    "018_fix_favorites_foreign_keys.sql",
		Description: "修复收藏表的外键约束问题",
		AppVersion:  "v2.32.0",
		Category:    "问题修复",
		CreatedAt:   parseTime("2024-08-01"),
		Dependencies: []string{"011"},
	},
	{
		Version:     "019",
		Name:        "Add Workflow DAG",
		FileName:    "019_add_workflow_dag.sql",
		Description: "添加工作流 DAG 支持，实现任务编排",
		AppVersion:  "v2.32.0",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-08-10"),
	},
	{
		Version:     "021",
		Name:        "Fix All Foreign Keys",
		FileName:    "021_fix_all_foreign_keys.sql",
		Description: "全面修复所有表的外键约束",
		AppVersion:  "v2.33.0",
		Category:    "问题修复",
		CreatedAt:   parseTime("2024-09-01"),
		Dependencies: []string{"018"},
	},
	{
		Version:     "022",
		Name:        "Add Template Unique Indexes with Soft Delete",
		FileName:    "022_add_template_unique_indexes_with_soft_delete.sql",
		Description: "为模板表添加支持软删除的唯一索引",
		AppVersion:  "v2.34.0",
		Category:    "索引优化",
		CreatedAt:   parseTime("2024-09-15"),
	},
	{
		Version:     "023",
		Name:        "Add Node Tracking to Progress",
		FileName:    "023_add_node_tracking_to_progress.sql",
		Description: "为进度表添加节点跟踪字段，记录成功和失败的节点",
		AppVersion:  "v2.34.1",
		Category:    "功能增强",
		CreatedAt:   parseTime("2024-10-01"),
	},
}

// GetMigrationByVersion 根据版本号获取迁移信息
func GetMigrationByVersion(version string) (*MigrationInfo, error) {
	for _, m := range MigrationRegistry {
		if m.Version == version {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("migration not found: %s", version)
}

// GetMigrationByFileName 根据文件名获取迁移信息
func GetMigrationByFileName(fileName string) (*MigrationInfo, error) {
	for _, m := range MigrationRegistry {
		if m.FileName == fileName {
			return &m, nil
		}
	}
	return nil, fmt.Errorf("migration not found: %s", fileName)
}

// GetMigrationsByAppVersion 获取指定应用版本的所有迁移
func GetMigrationsByAppVersion(appVersion string) []MigrationInfo {
	migrations := []MigrationInfo{}
	for _, m := range MigrationRegistry {
		if m.AppVersion == appVersion {
			migrations = append(migrations, m)
		}
	}
	return migrations
}

// GetMigrationsByCategory 根据分类获取迁移
func GetMigrationsByCategory(category string) []MigrationInfo {
	migrations := []MigrationInfo{}
	for _, m := range MigrationRegistry {
		if m.Category == category {
			migrations = append(migrations, m)
		}
	}
	return migrations
}

// GetMigrationsInRange 获取版本范围内的迁移
func GetMigrationsInRange(startVersion, endVersion string) []MigrationInfo {
	migrations := []MigrationInfo{}
	for _, m := range MigrationRegistry {
		if m.Version >= startVersion && m.Version <= endVersion {
			migrations = append(migrations, m)
		}
	}
	return migrations
}

// GetLatestMigration 获取最新的迁移
func GetLatestMigration() *MigrationInfo {
	if len(MigrationRegistry) == 0 {
		return nil
	}
	return &MigrationRegistry[len(MigrationRegistry)-1]
}

// GetMigrationCount 获取迁移总数
func GetMigrationCount() int {
	return len(MigrationRegistry)
}

// ValidateMigrationOrder 验证迁移顺序是否正确
func ValidateMigrationOrder() error {
	for i := 0; i < len(MigrationRegistry)-1; i++ {
		current := MigrationRegistry[i]
		next := MigrationRegistry[i+1]
		
		if current.Version >= next.Version {
			return fmt.Errorf("migration order invalid: %s should come before %s", 
				current.Version, next.Version)
		}
	}
	return nil
}

// ValidateMigrationDependencies 验证迁移依赖是否满足
func ValidateMigrationDependencies(version string) error {
	migration, err := GetMigrationByVersion(version)
	if err != nil {
		return err
	}
	
	if len(migration.Dependencies) == 0 {
		return nil
	}
	
	// 检查所有依赖是否在当前迁移之前
	for _, depVersion := range migration.Dependencies {
		depMigration, err := GetMigrationByVersion(depVersion)
		if err != nil {
			return fmt.Errorf("dependency not found: %s for migration %s", depVersion, version)
		}
		
		if depMigration.Version >= migration.Version {
			return fmt.Errorf("invalid dependency: %s depends on %s but it comes later", 
				version, depVersion)
		}
	}
	
	return nil
}

// GetMigrationStatistics 获取迁移统计信息
func GetMigrationStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// 总数
	stats["total"] = GetMigrationCount()
	
	// 按分类统计
	categoryStats := make(map[string]int)
	for _, m := range MigrationRegistry {
		categoryStats[m.Category]++
	}
	stats["by_category"] = categoryStats
	
	// 按应用版本统计
	versionStats := make(map[string]int)
	for _, m := range MigrationRegistry {
		versionStats[m.AppVersion]++
	}
	stats["by_app_version"] = versionStats
	
	// 最新迁移
	latest := GetLatestMigration()
	if latest != nil {
		stats["latest_version"] = latest.Version
		stats["latest_app_version"] = latest.AppVersion
	}
	
	return stats
}

// PrintMigrationList 打印迁移列表
func PrintMigrationList() {
	fmt.Println("\n=== Migration Registry ===")
	fmt.Printf("Total Migrations: %d\n\n", GetMigrationCount())
	
	fmt.Printf("%-8s %-10s %-50s %-15s\n", "Version", "App Ver", "Description", "Category")
	fmt.Println(strings.Repeat("-", 100))
	
	for _, m := range MigrationRegistry {
		desc := m.Description
		if len(desc) > 48 {
			desc = desc[:45] + "..."
		}
		fmt.Printf("%-8s %-10s %-50s %-15s\n", m.Version, m.AppVersion, desc, m.Category)
	}
	
	fmt.Println()
}

// PrintMigrationStatistics 打印迁移统计信息
func PrintMigrationStatistics() {
	stats := GetMigrationStatistics()
	
	fmt.Println("\n=== Migration Statistics ===")
	fmt.Printf("Total Migrations: %d\n", stats["total"])
	
	if latest, ok := stats["latest_version"].(string); ok {
		fmt.Printf("Latest Version:   %s\n", latest)
	}
	if latestApp, ok := stats["latest_app_version"].(string); ok {
		fmt.Printf("Latest App Ver:   %s\n", latestApp)
	}
	
	fmt.Println("\nBy Category:")
	if categoryStats, ok := stats["by_category"].(map[string]int); ok {
		for category, count := range categoryStats {
			fmt.Printf("  %-20s: %d\n", category, count)
		}
	}
	
	fmt.Println("\nBy Application Version:")
	if versionStats, ok := stats["by_app_version"].(map[string]int); ok {
		for version, count := range versionStats {
			fmt.Printf("  %-10s: %d migrations\n", version, count)
		}
	}
	
	fmt.Println()
}

// GetMigrationDetails 获取迁移的详细信息
func GetMigrationDetails(version string) (string, error) {
	migration, err := GetMigrationByVersion(version)
	if err != nil {
		return "", err
	}
	
	details := fmt.Sprintf(`
Migration Details:
------------------
Version:        %s
Name:           %s
File Name:      %s
Description:    %s
App Version:    %s
Category:       %s
Created At:     %s
`, 
		migration.Version,
		migration.Name,
		migration.FileName,
		migration.Description,
		migration.AppVersion,
		migration.Category,
		migration.CreatedAt.Format("2006-01-02"),
	)
	
	if len(migration.Dependencies) > 0 {
		details += fmt.Sprintf("Dependencies:   %v\n", migration.Dependencies)
	}
	
	return details, nil
}

// parseTime 辅助函数：解析时间字符串
func parseTime(dateStr string) time.Time {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}

