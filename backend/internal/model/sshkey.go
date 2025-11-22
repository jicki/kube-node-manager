package model

import (
	"time"

	"gorm.io/gorm"
)

// SSHKeyType SSH 密钥类型
type SSHKeyType string

const (
	SSHKeyTypePrivateKey SSHKeyType = "private_key" // 私钥认证
	SSHKeyTypePassword   SSHKeyType = "password"    // 密码认证
)

// SystemSSHKey 系统级 SSH 密钥模型 - 用于全局SSH认证配置
// 与原 AnsibleSSHKey 兼容，但作为系统级配置使用
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

// SSHKeyListRequest SSH 密钥列表请求
type SSHKeyListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Keyword  string `json:"keyword" form:"keyword"`
	Type     string `json:"type" form:"type"`
}

// SSHKeyCreateRequest SSH 密钥创建请求
type SSHKeyCreateRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	Type        SSHKeyType `json:"type" binding:"required,oneof=private_key password"`
	Username    string     `json:"username" binding:"required"`
	PrivateKey  string     `json:"private_key"`  // Type = private_key 时必填
	Passphrase  string     `json:"passphrase"`   // 可选
	Password    string     `json:"password"`     // Type = password 时必填
	Port        int        `json:"port"`         // 默认 22
	IsDefault   bool       `json:"is_default"`
}

// SSHKeyUpdateRequest SSH 密钥更新请求
type SSHKeyUpdateRequest struct {
	Name        *string     `json:"name"`
	Description *string     `json:"description"`
	Type        *SSHKeyType `json:"type"`
	Username    *string     `json:"username"`
	PrivateKey  *string     `json:"private_key"`
	Passphrase  *string     `json:"passphrase"`
	Password    *string     `json:"password"`
	Port        *int        `json:"port"`
	IsDefault   *bool       `json:"is_default"`
}

// SSHKeyResponse SSH 密钥响应（隐藏敏感信息）
type SSHKeyResponse struct {
	ID             uint       `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Type           SSHKeyType `json:"type"`
	Username       string     `json:"username"`
	Port           int        `json:"port"`
	IsDefault      bool       `json:"is_default"`
	HasPrivateKey  bool       `json:"has_private_key"`  // 是否配置了私钥
	HasPassphrase  bool       `json:"has_passphrase"`   // 是否配置了私钥密码
	HasPassword    bool       `json:"has_password"`     // 是否配置了密码
	CreatedBy      uint       `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
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

