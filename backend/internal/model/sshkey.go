package model

import (
	"time"

	"gorm.io/gorm"
)

// SystemSSHKey 系统级 SSH 密钥模型 - 用于全局SSH认证配置
// 与原 AnsibleSSHKey 兼容，但作为系统级配置使用
// 注意：SSHKeyType 和相关请求/响应结构在 ansible.go 中定义，两者共享
type SystemSSHKey struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null"` // 唯一索引由迁移文件创建
	Description string         `json:"description"`
	Type        SSHKeyType     `json:"type" gorm:"not null"`                  // private_key 或 password
	Username    string         `json:"username" gorm:"not null"`              // SSH 用户名
	PrivateKey  string         `json:"-" gorm:"type:text"`                    // 私钥内容（加密存储）
	Passphrase  string         `json:"-" gorm:"type:text"`                    // 私钥密码（加密存储）
	Password    string         `json:"-" gorm:"type:text"`                    // SSH 密码（加密存储）
	Port        int            `json:"port" gorm:"default:22"`                // SSH 端口
	IsDefault   bool           `json:"is_default" gorm:"default:false"`       // 是否为默认密钥
	CreatedBy   uint           `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (SystemSSHKey) TableName() string {
	return "system_ssh_keys"
}

// ToResponse 转换为响应格式（隐藏敏感信息）
func (k *SystemSSHKey) ToResponse() *SSHKeyResponse {
	return &SSHKeyResponse{
		ID:            k.ID,
		Name:          k.Name,
		Description:   k.Description,
		Type:          k.Type,
		Username:      k.Username,
		Port:          k.Port,
		IsDefault:     k.IsDefault,
		HasPrivateKey: k.PrivateKey != "",
		HasPassphrase: k.Passphrase != "",
		HasPassword:   k.Password != "",
		CreatedBy:     k.CreatedBy,
		CreatedAt:     k.CreatedAt,
		UpdatedAt:     k.UpdatedAt,
	}
}

// SSHKeyTestRequest SSH 密钥测试连接请求
type SSHKeyTestRequest struct {
	Host string `json:"host" binding:"required"` // 测试目标主机
	Port int    `json:"port"`                    // 目标端口，默认使用密钥配置的端口
}

