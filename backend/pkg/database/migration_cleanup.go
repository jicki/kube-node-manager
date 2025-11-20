package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// CleanupLegacyConstraints 清理旧的自动生成的约束名
// 在从旧版本升级时，GORM 可能自动生成了约束名，与新的命名约束冲突
func CleanupLegacyConstraints(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		// 只在 PostgreSQL 上需要清理
		return nil
	}
	
	log.Println("Checking for legacy constraint conflicts...")
	
	// 定义可能冲突的约束
	legacyConstraints := []struct {
		table      string
		constraint string
	}{
		{"code_migration_records", "uni_code_migration_records_migration_id"},
		// 可以添加更多需要清理的约束
	}
	
	for _, c := range legacyConstraints {
		// 检查约束是否存在
		var count int64
		query := `
			SELECT COUNT(*) 
			FROM information_schema.table_constraints 
			WHERE table_name = ? AND constraint_name = ?
		`
		if err := db.Raw(query, c.table, c.constraint).Scan(&count).Error; err != nil {
			log.Printf("Warning: Failed to check constraint %s.%s: %v", c.table, c.constraint, err)
			continue
		}
		
		if count > 0 {
			// 约束存在，删除它
			sql := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s", c.table, c.constraint)
			if err := db.Exec(sql).Error; err != nil {
				log.Printf("Warning: Failed to drop constraint %s.%s: %v", c.table, c.constraint, err)
				// 继续处理其他约束
			} else {
				log.Printf("✓ Dropped legacy constraint: %s.%s", c.table, c.constraint)
			}
		}
	}
	
	log.Println("✓ Legacy constraint cleanup completed")
	return nil
}

// RecreateProblematicTables 重新创建有问题的表
func RecreateProblematicTables(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	
	log.Println("Checking for problematic tables...")
	
	// 对于 code_migration_records 表，如果存在就直接删除重建
	// 这是因为旧的表可能有不兼容的约束名
	if db.Migrator().HasTable("code_migration_records") {
		log.Println("Found existing code_migration_records table, dropping for clean recreation...")
		
		// 删除表（使用 CASCADE 确保删除所有依赖）
		if err := db.Exec("DROP TABLE IF EXISTS code_migration_records CASCADE").Error; err != nil {
			log.Printf("Warning: Failed to drop code_migration_records table: %v", err)
			// 尝试强制删除约束
			db.Exec("ALTER TABLE code_migration_records DROP CONSTRAINT IF EXISTS uni_code_migration_records_migration_id CASCADE")
			db.Exec("ALTER TABLE code_migration_records DROP CONSTRAINT IF EXISTS code_migration_records_pkey CASCADE")
			// 再次尝试删除表
			if err := db.Exec("DROP TABLE IF EXISTS code_migration_records CASCADE").Error; err != nil {
				return fmt.Errorf("failed to drop code_migration_records table: %w", err)
			}
		}
		
		log.Println("✓ Dropped code_migration_records table for clean recreation")
	}
	
	return nil
}

