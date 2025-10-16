package model

import (
	"time"
)

type AuditLog struct {
	ID           uint         `json:"id" gorm:"primaryKey"`
	UserID       uint         `json:"user_id" gorm:"not null"`
	ClusterID    *uint        `json:"cluster_id"` // 使用指针类型，允许 NULL
	NodeName     string       `json:"node_name"`
	Action       AuditAction  `json:"action" gorm:"not null"`
	ResourceType ResourceType `json:"resource_type" gorm:"not null"`
	Details      string       `json:"details" gorm:"type:text"`
	Reason       string       `json:"reason" gorm:"type:text"` // 操作原因，特别用于记录禁止调度的原因
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
	ActionTest   AuditAction = "test"   // 测试连接
	ActionQuery  AuditAction = "query"  // 查询
	ActionBind   AuditAction = "bind"   // 绑定
	ActionUnbind AuditAction = "unbind" // 解绑
)

type ResourceType string

const (
	ResourceUser           ResourceType = "user"
	ResourceCluster        ResourceType = "cluster"
	ResourceNode           ResourceType = "node"
	ResourceLabel          ResourceType = "label"
	ResourceTaint          ResourceType = "taint"
	ResourceLabelTemplate  ResourceType = "label_template"
	ResourceTaintTemplate  ResourceType = "taint_template"
	ResourceFeishuSettings ResourceType = "feishu_settings" // 飞书配置
	ResourceFeishuGroup    ResourceType = "feishu_group"    // 飞书群组
	ResourceFeishuUser     ResourceType = "feishu_user"     // 飞书用户
)

type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailed  AuditStatus = "failed"
)
