package model

import (
	"time"

	"gorm.io/gorm"
)

// FeishuSettings stores Feishu (Lark) configuration
// Only one record should exist in the database
type FeishuSettings struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	Enabled    bool           `json:"enabled" gorm:"default:false"`
	AppID      string         `json:"app_id" gorm:"type:varchar(255)"`
	AppSecret  string         `json:"-" gorm:"type:text"`               // Encrypted secret, not exposed in JSON
	BotEnabled bool           `json:"bot_enabled" gorm:"default:false"` // 机器人功能启用状态（使用长连接模式）
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// FeishuSettingsResponse is used for API responses (without sensitive data)
type FeishuSettingsResponse struct {
	ID           uint      `json:"id"`
	Enabled      bool      `json:"enabled"`
	AppID        string    `json:"app_id"`
	HasAppSecret bool      `json:"has_app_secret"` // Indicates if app_secret is configured
	BotEnabled   bool      `json:"bot_enabled"`    // 机器人功能启用状态
	BotConnected bool      `json:"bot_connected"`  // 长连接状态
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
		BotEnabled:   f.BotEnabled,
		BotConnected: false, // 将由 Service 层设置实际的连接状态
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}

// FeishuUserMapping stores the mapping between Feishu users and system users
type FeishuUserMapping struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	FeishuUserID string         `json:"feishu_user_id" gorm:"type:varchar(255);uniqueIndex;not null"` // 飞书用户 open_id
	SystemUserID uint           `json:"system_user_id" gorm:"not null"`                               // 系统用户 ID
	Username     string         `json:"username" gorm:"type:varchar(100)"`
	FeishuName   string         `json:"feishu_name" gorm:"type:varchar(255)"` // 飞书用户名
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联的系统用户
	User User `json:"user,omitempty" gorm:"foreignKey:SystemUserID"`
}

// TableName specifies the table name for FeishuUserMapping
func (FeishuUserMapping) TableName() string {
	return "feishu_user_mappings"
}

// FeishuUserSession 存储用户的会话状态（如当前选择的集群）
type FeishuUserSession struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	FeishuUserID    string         `json:"feishu_user_id" gorm:"type:varchar(255);uniqueIndex;not null"` // 飞书用户 open_id
	CurrentCluster  string         `json:"current_cluster" gorm:"type:varchar(255)"`                     // 当前选择的集群名称
	LastCommandTime time.Time      `json:"last_command_time"`                                            // 最后一次命令时间
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for FeishuUserSession
func (FeishuUserSession) TableName() string {
	return "feishu_user_sessions"
}
