package model

import (
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Cluster{},
		&LabelTemplate{},
		&TaintTemplate{},
		&AuditLog{},
		&ProgressTask{},
		&ProgressMessage{},
		&GitlabSettings{},
		&GitlabRunner{},
	)
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
