package model

import (
	"time"

	"gorm.io/gorm"
)

// AnomalyType 异常类型
type AnomalyType string

const (
	AnomalyTypeNotReady           AnomalyType = "NotReady"
	AnomalyTypeMemoryPressure     AnomalyType = "MemoryPressure"
	AnomalyTypeDiskPressure       AnomalyType = "DiskPressure"
	AnomalyTypePIDPressure        AnomalyType = "PIDPressure"
	AnomalyTypeNetworkUnavailable AnomalyType = "NetworkUnavailable"
)

// AnomalyStatus 异常状态
type AnomalyStatus string

const (
	AnomalyStatusActive   AnomalyStatus = "Active"   // 进行中
	AnomalyStatusResolved AnomalyStatus = "Resolved" // 已恢复
)

// NodeAnomaly 节点异常记录
type NodeAnomaly struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	ClusterID   uint           `json:"cluster_id" gorm:"not null;index:idx_cluster_node"`
	ClusterName string         `json:"cluster_name" gorm:"not null"`
	NodeName    string         `json:"node_name" gorm:"not null;index:idx_cluster_node"`
	AnomalyType AnomalyType    `json:"anomaly_type" gorm:"not null;index:idx_anomaly_type"`
	Status      AnomalyStatus  `json:"status" gorm:"default:Active;index:idx_status"`
	StartTime   time.Time      `json:"start_time" gorm:"not null;index:idx_start_time"`
	EndTime     *time.Time     `json:"end_time,omitempty"`
	Duration    int64          `json:"duration" gorm:"default:0"` // 持续时长（秒）
	Reason      string         `json:"reason" gorm:"type:text"`
	Message     string         `json:"message" gorm:"type:text"`
	LastCheck   time.Time      `json:"last_check" gorm:"not null"` // 最后检查时间
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	Cluster Cluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
}

// TableName 指定表名
func (NodeAnomaly) TableName() string {
	return "node_anomalies"
}

// CalculateDuration 计算持续时长（秒）
func (a *NodeAnomaly) CalculateDuration() int64 {
	if a.Status == AnomalyStatusResolved && a.EndTime != nil {
		return int64(a.EndTime.Sub(a.StartTime).Seconds())
	}
	// 对于活跃状态，计算到当前时间的持续时长
	return int64(time.Since(a.StartTime).Seconds())
}

// AnomalyStatistics 异常统计数据
type AnomalyStatistics struct {
	Date            string  `json:"date"`             // 日期（YYYY-MM-DD 或 YYYY-WW）
	TotalCount      int64   `json:"total_count"`      // 总异常次数
	ActiveCount     int64   `json:"active_count"`     // 活跃异常数
	ResolvedCount   int64   `json:"resolved_count"`   // 已恢复异常数
	AverageDuration float64 `json:"average_duration"` // 平均持续时长（秒）
	AffectedNodes   int64   `json:"affected_nodes"`   // 受影响节点数
}

// AnomalyTypeStatistics 按异常类型统计
type AnomalyTypeStatistics struct {
	AnomalyType AnomalyType `json:"anomaly_type"`
	TotalCount  int64       `json:"total_count"` // 总异常次数
}
