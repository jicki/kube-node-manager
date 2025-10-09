package model

import (
	"time"

	"gorm.io/gorm"
)

// GitlabSettings stores GitLab configuration
// Only one record should exist in the database
type GitlabSettings struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Enabled   bool           `json:"enabled" gorm:"default:false"`
	Domain    string         `json:"domain" gorm:"type:varchar(255)"`
	Token     string         `json:"-" gorm:"type:text"` // Encrypted token, not exposed in JSON
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// GitlabSettingsResponse is used for API responses (without sensitive data)
type GitlabSettingsResponse struct {
	ID        uint      `json:"id"`
	Enabled   bool      `json:"enabled"`
	Domain    string    `json:"domain"`
	HasToken  bool      `json:"has_token"` // Indicates if token is configured
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts GitlabSettings to GitlabSettingsResponse
func (g *GitlabSettings) ToResponse() *GitlabSettingsResponse {
	return &GitlabSettingsResponse{
		ID:        g.ID,
		Enabled:   g.Enabled,
		Domain:    g.Domain,
		HasToken:  g.Token != "",
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
}

// GitlabRunner stores GitLab runner information created by the platform
type GitlabRunner struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	RunnerID    int            `json:"runner_id" gorm:"uniqueIndex;not null"` // GitLab Runner ID
	Token       string         `json:"-" gorm:"type:text;not null"`           // Runner registration token (encrypted)
	Description string         `json:"description" gorm:"type:varchar(255)"`
	RunnerType  string         `json:"runner_type" gorm:"type:varchar(50)"`
	CreatedBy   string         `json:"created_by" gorm:"type:varchar(100)"` // Username who created this runner
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName specifies the table name for GitlabRunner
func (GitlabRunner) TableName() string {
	return "gitlab_runners"
}
