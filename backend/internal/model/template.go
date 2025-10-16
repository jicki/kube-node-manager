package model

import (
	"time"

	"gorm.io/gorm"
)

type LabelTemplate struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	Labels      string         `json:"labels" gorm:"type:text;not null"` // JSON格式存储键值对
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	Creator User `json:"creator" gorm:"foreignKey:CreatedBy"`
}

type TaintTemplate struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	Taints      string         `json:"taints" gorm:"type:text;not null"` // JSON格式存储污点信息
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	Creator User `json:"creator" gorm:"foreignKey:CreatedBy"`
}
