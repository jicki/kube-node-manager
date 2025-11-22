package model

import (
	"time"

	"gorm.io/gorm"
)

// NodeSettings 节点配置模型 - 用于存储节点特定的配置（如SSH连接信息）
type NodeSettings struct {
	ID             uint           `json:"id" gorm:"primarykey"`
	ClusterName    string         `json:"cluster_name" gorm:"not null;uniqueIndex:idx_cluster_node"`
	NodeName       string         `json:"node_name" gorm:"not null;uniqueIndex:idx_cluster_node"`
	SSHPort        int            `json:"ssh_port" gorm:"default:22"`
	SSHUser        string         `json:"ssh_user"` // 如果为空，使用密钥的用户
	SystemSSHKeyID *uint          `json:"system_ssh_key_id"` // SSH密钥ID，可以是system_ssh_keys或ansible_ssh_keys表的ID
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 注意：不定义 SystemSSHKey 关联字段，避免GORM尝试管理跨表关联
	// 密钥的加载在服务层手动处理，因为密钥可能来自不同的表(system_ssh_keys 或 ansible_ssh_keys)
}

// TableName 指定表名
func (NodeSettings) TableName() string {
	return "node_settings"
}
