package model

import (
	"time"

	"gorm.io/gorm"
)

type Cluster struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"uniqueIndex;not null"`
	Description string     `json:"description"`
	KubeConfig  string     `json:"kube_config" gorm:"type:text;not null"`
	Status      ClusterStatus `json:"status" gorm:"default:active"`
	Version     string     `json:"version"`
	NodeCount   int        `json:"node_count" gorm:"default:0"`
	LastSync    *time.Time `json:"last_sync"`
	CreatedBy   uint       `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	Creator     User       `json:"creator" gorm:"foreignKey:CreatedBy"`
}

type ClusterStatus string

const (
	ClusterStatusActive     ClusterStatus = "active"
	ClusterStatusInactive   ClusterStatus = "inactive"
	ClusterStatusError      ClusterStatus = "error"
	ClusterStatusMaintenance ClusterStatus = "maintenance"
)