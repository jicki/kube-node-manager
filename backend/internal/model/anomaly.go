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

// NodeRoleStatistics 按节点角色聚合统计
type NodeRoleStatistics struct {
	Role            string  `json:"role"`             // 节点角色（master/worker/etcd等）
	ClusterID       *uint   `json:"cluster_id"`       // 集群ID（可选，用于多集群统计）
	ClusterName     string  `json:"cluster_name"`     // 集群名称
	TotalCount      int64   `json:"total_count"`      // 总异常次数
	ActiveCount     int64   `json:"active_count"`     // 活跃异常数
	ResolvedCount   int64   `json:"resolved_count"`   // 已恢复异常数
	AverageDuration float64 `json:"average_duration"` // 平均持续时长（秒）
	AffectedNodes   int64   `json:"affected_nodes"`   // 受影响节点数
	NodeCount       int64   `json:"node_count"`       // 该角色节点总数
}

// ClusterAggregateStatistics 按集群聚合统计
type ClusterAggregateStatistics struct {
	ClusterID       uint    `json:"cluster_id"`
	ClusterName     string  `json:"cluster_name"`
	TotalCount      int64   `json:"total_count"`      // 总异常次数
	ActiveCount     int64   `json:"active_count"`     // 活跃异常数
	ResolvedCount   int64   `json:"resolved_count"`   // 已恢复异常数
	AverageDuration float64 `json:"average_duration"` // 平均持续时长（秒）
	AffectedNodes   int64   `json:"affected_nodes"`   // 受影响节点数
	TotalNodes      int64   `json:"total_nodes"`      // 集群节点总数
	HealthScore     float64 `json:"health_score"`     // 集群健康度评分（0-100）
}

// NodeHistoryTrend 单节点历史趋势
type NodeHistoryTrend struct {
	Date          string  `json:"date"`           // 日期
	TotalCount    int64   `json:"total_count"`    // 当日异常次数
	ActiveCount   int64   `json:"active_count"`   // 当日活跃异常数
	ResolvedCount int64   `json:"resolved_count"` // 当日已恢复异常数
	AvgDuration   float64 `json:"avg_duration"`   // 当日平均持续时长
}

// MTTRStatistics 平均恢复时间统计
type MTTRStatistics struct {
	EntityType    string  `json:"entity_type"`    // 实体类型（node/cluster/role）
	EntityName    string  `json:"entity_name"`    // 实体名称
	MTTR          float64 `json:"mttr"`           // 平均恢复时间（秒）
	ResolvedCount int64   `json:"resolved_count"` // 已恢复异常数
	TotalDuration int64   `json:"total_duration"` // 累计恢复时长（秒）
	MinDuration   int64   `json:"min_duration"`   // 最短恢复时间（秒）
	MaxDuration   int64   `json:"max_duration"`   // 最长恢复时间（秒）
}

// SLAMetrics SLA 可用性指标
type SLAMetrics struct {
	EntityType       string    `json:"entity_type"`       // 实体类型（node/cluster）
	EntityName       string    `json:"entity_name"`       // 实体名称
	StartTime        time.Time `json:"start_time"`        // 统计开始时间
	EndTime          time.Time `json:"end_time"`          // 统计结束时间
	TotalTime        int64     `json:"total_time"`        // 总时间（秒）
	DowntimeDuration int64     `json:"downtime_duration"` // 异常时长（秒）
	UptimeDuration   int64     `json:"uptime_duration"`   // 正常时长（秒）
	Availability     float64   `json:"availability"`      // 可用性百分比（0-100）
	AnomalyCount     int64     `json:"anomaly_count"`     // 异常次数
}

// RecoveryMetrics 恢复率和复发率指标
type RecoveryMetrics struct {
	EntityType     string  `json:"entity_type"`     // 实体类型（node/cluster）
	EntityName     string  `json:"entity_name"`     // 实体名称
	TotalCount     int64   `json:"total_count"`     // 总异常次数
	ResolvedCount  int64   `json:"resolved_count"`  // 已恢复异常数
	ActiveCount    int64   `json:"active_count"`    // 活跃异常数
	RecoveryRate   float64 `json:"recovery_rate"`   // 恢复率（%）
	RecurringCount int64   `json:"recurring_count"` // 复发异常次数
	RecurrenceRate float64 `json:"recurrence_rate"` // 复发率（%）
}

// NodeHealthScore 节点健康度评分
type NodeHealthScore struct {
	NodeName        string             `json:"node_name"`
	ClusterID       uint               `json:"cluster_id"`
	ClusterName     string             `json:"cluster_name"`
	HealthScore     float64            `json:"health_score"`     // 综合健康度评分（0-100）
	ScoreLevel      string             `json:"score_level"`      // 评分等级（优秀/良好/一般/较差）
	TotalAnomalies  int64              `json:"total_anomalies"`  // 总异常次数
	ActiveAnomalies int64              `json:"active_anomalies"` // 活跃异常数
	AvgMTTR         float64            `json:"avg_mttr"`         // 平均恢复时间
	Availability    float64            `json:"availability"`     // 可用性（%）
	LastAnomaly     *time.Time         `json:"last_anomaly"`     // 最后一次异常时间
	Factors         map[string]float64 `json:"factors"`          // 影响因素分解
}

// GetScoreLevel 根据分数返回评分等级
func (s *NodeHealthScore) GetScoreLevel() string {
	if s.HealthScore >= 90 {
		return "优秀"
	} else if s.HealthScore >= 75 {
		return "良好"
	} else if s.HealthScore >= 60 {
		return "一般"
	}
	return "较差"
}

// HeatmapDataPoint 热力图数据点
type HeatmapDataPoint struct {
	Time     string `json:"time"`      // 时间点
	NodeName string `json:"node_name"` // 节点名称
	Value    int64  `json:"value"`     // 异常数量
}

// CalendarDataPoint 日历图数据点
type CalendarDataPoint struct {
	Date  string `json:"date"`  // 日期（YYYY-MM-DD）
	Value int64  `json:"value"` // 异常数量
}

