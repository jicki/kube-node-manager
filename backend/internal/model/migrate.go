package model

import (
	"gorm.io/gorm"
)

// GetAllModels 返回所有 GORM 模型实例（用于新的代码迁移系统）
func GetAllModels() []interface{} {
	return []interface{}{
		&User{},
		&Cluster{},
		&LabelTemplate{},
		&TaintTemplate{},
		&AuditLog{},
		&ProgressTask{},
		&ProgressMessage{},
		&GitlabSettings{},
		&GitlabRunner{},
		&FeishuSettings{},
		&FeishuUserMapping{},
		&FeishuUserSession{},
		&NodeAnomaly{},
		&CacheEntry{},
		&AnsibleTask{},
		&AnsibleTemplate{},
		&AnsibleLog{},
		&AnsibleInventory{},
		&AnsibleSSHKey{},
		&AnsibleSchedule{},
		&AnsibleFavorite{},
		&AnsibleTaskHistory{},
		&SystemSSHKey{},
		&AnsibleTag{},
		&AnsibleTaskTag{},
		&AnsibleWorkflow{},
		&AnsibleWorkflowExecution{},
	}
}

// AutoMigrate 执行 GORM 自动迁移（保留用于向后兼容）
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(GetAllModels()...)
}

func SeedDefaultData(db *gorm.DB) error {
	var adminUser User
	result := db.Where("username = ?", "admin").First(&adminUser)
	if result.Error != nil && result.Error == gorm.ErrRecordNotFound {
		adminUser = User{
			Username: "admin",
			Email:    "admin@example.com",
			Role:     RoleAdmin,
			Status:   StatusActive,
		}
		if err := adminUser.HashPassword("admin123"); err != nil {
			return err
		}
		if err := db.Create(&adminUser).Error; err != nil {
			return err
		}
	}

	return nil
}
