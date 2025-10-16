package model

import (
	"time"

	"gorm.io/gorm"
)

// FeishuSettings stores Feishu (Lark) configuration
// Only one record should exist in the database
type FeishuSettings struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Enabled   bool           `json:"enabled" gorm:"default:false"`
	AppID     string         `json:"app_id" gorm:"type:varchar(255)"`
	AppSecret string         `json:"-" gorm:"type:text"` // Encrypted secret, not exposed in JSON
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// FeishuSettingsResponse is used for API responses (without sensitive data)
type FeishuSettingsResponse struct {
	ID           uint      `json:"id"`
	Enabled      bool      `json:"enabled"`
	AppID        string    `json:"app_id"`
	HasAppSecret bool      `json:"has_app_secret"` // Indicates if app_secret is configured
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ToResponse converts FeishuSettings to FeishuSettingsResponse
func (f *FeishuSettings) ToResponse() *FeishuSettingsResponse {
	return &FeishuSettingsResponse{
		ID:           f.ID,
		Enabled:      f.Enabled,
		AppID:        f.AppID,
		HasAppSecret: f.AppSecret != "",
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}
