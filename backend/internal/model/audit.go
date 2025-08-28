package model

import (
	"time"
)

type AuditLog struct {
	ID           uint         `json:"id" gorm:"primaryKey"`
	UserID       uint         `json:"user_id" gorm:"not null"`
	ClusterID    uint         `json:"cluster_id"`
	NodeName     string       `json:"node_name"`
	Action       AuditAction  `json:"action" gorm:"not null"`
	ResourceType ResourceType `json:"resource_type" gorm:"not null"`
	Details      string       `json:"details" gorm:"type:text"`
	Status       AuditStatus  `json:"status" gorm:"default:success"`
	ErrorMsg     string       `json:"error_msg"`
	IPAddress    string       `json:"ip_address"`
	UserAgent    string       `json:"user_agent"`
	CreatedAt    time.Time    `json:"created_at"`

	User    User     `json:"user" gorm:"foreignKey:UserID"`
	Cluster *Cluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
}

type AuditAction string

const (
	ActionCreate AuditAction = "create"
	ActionUpdate AuditAction = "update"
	ActionDelete AuditAction = "delete"
	ActionView   AuditAction = "view"
	ActionLogin  AuditAction = "login"
	ActionLogout AuditAction = "logout"
)

type ResourceType string

const (
	ResourceUser          ResourceType = "user"
	ResourceCluster       ResourceType = "cluster"
	ResourceNode          ResourceType = "node"
	ResourceLabel         ResourceType = "label"
	ResourceTaint         ResourceType = "taint"
	ResourceLabelTemplate ResourceType = "label_template"
	ResourceTaintTemplate ResourceType = "taint_template"
)

type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailed  AuditStatus = "failed"
)
