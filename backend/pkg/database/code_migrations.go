package database

import (
	"fmt"
	"log"
	"sort"
	"time"

	"gorm.io/gorm"
)

// MigrationFunc 迁移函数类型
type MigrationFunc func(db *gorm.DB) error

// CodeMigration 代码迁移定义
type CodeMigration struct {
	ID          string        // 唯一标识如 "M001"
	Description string        // 迁移描述
	DependsOn   []string      // 依赖的迁移 ID
	UpFunc      MigrationFunc // 升级函数
	DownFunc    MigrationFunc // 回滚函数（可选）
	CreatedAt   time.Time     // 创建时间
}

// CodeMigrationRecord 代码迁移执行记录
type CodeMigrationRecord struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	MigrationID string    `gorm:"uniqueIndex:idx_code_migration_id;size:255;not null" json:"migration_id"`
	Description string    `gorm:"type:text" json:"description"`
	Status      string    `gorm:"size:20;not null;default:'success'" json:"status"` // success, failed, pending
	DurationMs  int64     `gorm:"default:0" json:"duration_ms"`
	ErrorMsg    string    `gorm:"type:text" json:"error_msg,omitempty"`
	AppliedAt   time.Time `gorm:"not null;index:idx_code_migration_applied" json:"applied_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (CodeMigrationRecord) TableName() string {
	return "code_migration_records"
}

// CodeMigrations 代码迁移注册表
// 未来的代码迁移在这里注册
var CodeMigrations = []CodeMigration{
	// 示例：
	// {
	//     ID:          "M001",
	//     Description: "添加示例字段",
	//     DependsOn:   []string{},
	//     UpFunc: func(db *gorm.DB) error {
	//         return db.Exec("ALTER TABLE example ADD COLUMN new_field VARCHAR(255)").Error
	//     },
	//     DownFunc: func(db *gorm.DB) error {
	//         return db.Exec("ALTER TABLE example DROP COLUMN new_field").Error
	//     },
	//     CreatedAt: time.Date(2024, 11, 20, 0, 0, 0, 0, time.UTC),
	// },
}

// CodeMigrationExecutor 代码迁移执行器
type CodeMigrationExecutor struct {
	db         *gorm.DB
	migrations []CodeMigration
}

// NewCodeMigrationExecutor 创建代码迁移执行器
func NewCodeMigrationExecutor(db *gorm.DB) (*CodeMigrationExecutor, error) {
	executor := &CodeMigrationExecutor{
		db:         db,
		migrations: CodeMigrations,
	}
	
	// 确保迁移记录表存在
	// 注意：约束冲突已在 CleanupLegacyConstraints 中处理
	if err := db.AutoMigrate(&CodeMigrationRecord{}); err != nil {
		return nil, fmt.Errorf("failed to create code_migration_records table: %w", err)
	}
	
	return executor, nil
}

// GetPendingMigrations 获取待执行的迁移
func (e *CodeMigrationExecutor) GetPendingMigrations() ([]CodeMigration, error) {
	// 获取已执行的迁移 ID
	var executed []CodeMigrationRecord
	if err := e.db.Where("status = ?", "success").Find(&executed).Error; err != nil {
		return nil, fmt.Errorf("failed to get executed migrations: %w", err)
	}
	
	executedMap := make(map[string]bool)
	for _, record := range executed {
		executedMap[record.MigrationID] = true
	}
	
	// 筛选未执行的迁移
	var pending []CodeMigration
	for _, migration := range e.migrations {
		if !executedMap[migration.ID] {
			pending = append(pending, migration)
		}
	}
	
	// 按依赖关系排序
	sorted, err := e.topologicalSort(pending)
	if err != nil {
		return nil, fmt.Errorf("failed to sort migrations: %w", err)
	}
	
	return sorted, nil
}

// ExecuteCodeMigrations 执行代码迁移
func (e *CodeMigrationExecutor) ExecuteCodeMigrations() error {
	pending, err := e.GetPendingMigrations()
	if err != nil {
		return err
	}
	
	if len(pending) == 0 {
		log.Println("✓ No pending code migrations")
		return nil
	}
	
	log.Printf("Found %d pending code migrations", len(pending))
	
	for _, migration := range pending {
		if err := e.executeMigration(migration); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.ID, err)
		}
	}
	
	log.Printf("✓ All code migrations completed successfully")
	return nil
}

// executeMigration 执行单个迁移
func (e *CodeMigrationExecutor) executeMigration(migration CodeMigration) error {
	log.Printf("Executing code migration: %s - %s", migration.ID, migration.Description)
	
	startTime := time.Now()
	
	// 在事务中执行迁移
	err := e.db.Transaction(func(tx *gorm.DB) error {
		// 执行迁移函数
		if err := migration.UpFunc(tx); err != nil {
			return err
		}
		
		// 记录迁移执行
		duration := time.Since(startTime).Milliseconds()
		record := CodeMigrationRecord{
			MigrationID: migration.ID,
			Description: migration.Description,
			Status:      "success",
			DurationMs:  duration,
			AppliedAt:   time.Now(),
		}
		
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}
		
		return nil
	})
	
	if err != nil {
		// 记录失败
		duration := time.Since(startTime).Milliseconds()
		record := CodeMigrationRecord{
			MigrationID: migration.ID,
			Description: migration.Description,
			Status:      "failed",
			DurationMs:  duration,
			ErrorMsg:    err.Error(),
			AppliedAt:   time.Now(),
		}
		
		// 尽力记录失败，即使失败也不影响错误返回
		_ = e.db.Create(&record).Error
		
		return err
	}
	
	duration := time.Since(startTime).Milliseconds()
	log.Printf("✓ Migration %s completed in %dms", migration.ID, duration)
	
	return nil
}

// topologicalSort 对迁移进行拓扑排序（处理依赖关系）
func (e *CodeMigrationExecutor) topologicalSort(migrations []CodeMigration) ([]CodeMigration, error) {
	// 构建依赖图
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	migrationMap := make(map[string]CodeMigration)
	
	for _, m := range migrations {
		migrationMap[m.ID] = m
		inDegree[m.ID] = 0
		graph[m.ID] = []string{}
	}
	
	for _, m := range migrations {
		for _, dep := range m.DependsOn {
			graph[dep] = append(graph[dep], m.ID)
			inDegree[m.ID]++
		}
	}
	
	// 拓扑排序
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	
	// 按字母顺序排序队列，确保稳定的执行顺序
	sort.Strings(queue)
	
	var sorted []CodeMigration
	
	for len(queue) > 0 {
		// 取出队首
		current := queue[0]
		queue = queue[1:]
		
		sorted = append(sorted, migrationMap[current])
		
		// 处理依赖当前节点的节点
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue) // 保持排序
			}
		}
	}
	
	// 检测循环依赖
	if len(sorted) != len(migrations) {
		return nil, fmt.Errorf("circular dependency detected in migrations")
	}
	
	return sorted, nil
}

// GetExecutedMigrations 获取已执行的迁移
func (e *CodeMigrationExecutor) GetExecutedMigrations() ([]CodeMigrationRecord, error) {
	var records []CodeMigrationRecord
	
	if err := e.db.Where("status = ?", "success").Order("applied_at ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get executed migrations: %w", err)
	}
	
	return records, nil
}

// GetMigrationStatus 获取迁移状态
func (e *CodeMigrationExecutor) GetMigrationStatus() (map[string]interface{}, error) {
	executed, err := e.GetExecutedMigrations()
	if err != nil {
		return nil, err
	}
	
	pending, err := e.GetPendingMigrations()
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"total_migrations":    len(e.migrations),
		"executed_migrations": len(executed),
		"pending_migrations":  len(pending),
		"last_migration":      getLastMigration(executed),
	}, nil
}

// getLastMigration 获取最后执行的迁移
func getLastMigration(records []CodeMigrationRecord) *string {
	if len(records) == 0 {
		return nil
	}
	
	last := records[len(records)-1]
	result := last.MigrationID
	return &result
}

// RollbackMigration 回滚迁移（如果提供了 DownFunc）
func (e *CodeMigrationExecutor) RollbackMigration(migrationID string) error {
	// 查找迁移
	var migration *CodeMigration
	for _, m := range e.migrations {
		if m.ID == migrationID {
			migration = &m
			break
		}
	}
	
	if migration == nil {
		return fmt.Errorf("migration not found: %s", migrationID)
	}
	
	if migration.DownFunc == nil {
		return fmt.Errorf("migration %s does not support rollback", migrationID)
	}
	
	log.Printf("Rolling back migration: %s", migrationID)
	
	// 在事务中执行回滚
	err := e.db.Transaction(func(tx *gorm.DB) error {
		if err := migration.DownFunc(tx); err != nil {
			return err
		}
		
		// 删除执行记录
		if err := tx.Where("migration_id = ?", migrationID).Delete(&CodeMigrationRecord{}).Error; err != nil {
			return fmt.Errorf("failed to delete migration record: %w", err)
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	
	log.Printf("✓ Migration %s rolled back successfully", migrationID)
	return nil
}

