package automation

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"regexp"

	"gorm.io/gorm"
)

// CredentialManager 凭据管理器
type CredentialManager struct {
	db            *gorm.DB
	logger        *logger.Logger
	encryptionKey []byte // AES-256 需要 32 字节密钥
}

// NewCredentialManager 创建凭据管理器
func NewCredentialManager(db *gorm.DB, logger *logger.Logger, encryptionKey string) *CredentialManager {
	// 使用 SHA-256 从密钥字符串生成固定长度的密钥
	hash := sha256.Sum256([]byte(encryptionKey))

	return &CredentialManager{
		db:            db,
		logger:        logger,
		encryptionKey: hash[:],
	}
}

// Encrypt 使用 AES-256-GCM 加密数据
func (cm *CredentialManager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 创建随机 nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据（nonce + 加密数据）
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 使用 AES-256-GCM 解密数据
func (cm *CredentialManager) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// CreateCredential 创建新凭据
func (cm *CredentialManager) CreateCredential(credential *model.SSHCredential) error {
	// 验证凭据数据
	if err := cm.validateCredential(credential); err != nil {
		return err
	}

	// 加密敏感数据
	if credential.Password != "" {
		encrypted, err := cm.Encrypt(credential.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		credential.Password = encrypted
	}

	if credential.PrivateKey != "" {
		encrypted, err := cm.Encrypt(credential.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt private key: %w", err)
		}
		credential.PrivateKey = encrypted
	}

	if credential.Passphrase != "" {
		encrypted, err := cm.Encrypt(credential.Passphrase)
		if err != nil {
			return fmt.Errorf("failed to encrypt passphrase: %w", err)
		}
		credential.Passphrase = encrypted
	}

	// 检查名称是否已存在
	var existing model.SSHCredential
	if err := cm.db.Where("name = ?", credential.Name).First(&existing).Error; err == nil {
		return fmt.Errorf("credential with name '%s' already exists", credential.Name)
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	// 如果设置为默认凭据，取消其他默认凭据
	if credential.IsDefault {
		if err := cm.db.Model(&model.SSHCredential{}).
			Where("cluster_name = ? AND is_default = ?", credential.ClusterName, true).
			Update("is_default", false).Error; err != nil {
			return err
		}
	}

	// 保存到数据库
	if err := cm.db.Create(credential).Error; err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}

	cm.logger.Infof("Created SSH credential: %s", credential.Name)
	return nil
}

// GetCredential 获取凭据并解密
func (cm *CredentialManager) GetCredential(id uint) (*model.SSHCredential, error) {
	var credential model.SSHCredential
	if err := cm.db.First(&credential, id).Error; err != nil {
		return nil, err
	}

	// 解密敏感数据
	if credential.Password != "" {
		decrypted, err := cm.Decrypt(credential.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
		credential.Password = decrypted
	}

	if credential.PrivateKey != "" {
		decrypted, err := cm.Decrypt(credential.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}
		credential.PrivateKey = decrypted
	}

	if credential.Passphrase != "" {
		decrypted, err := cm.Decrypt(credential.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt passphrase: %w", err)
		}
		credential.Passphrase = decrypted
	}

	return &credential, nil
}

// ListCredentials 列出凭据（不包含敏感数据）
func (cm *CredentialManager) ListCredentials(clusterName string) ([]model.SSHCredential, error) {
	var credentials []model.SSHCredential
	query := cm.db.Model(&model.SSHCredential{})

	if clusterName != "" {
		query = query.Where("cluster_name = ?", clusterName)
	}

	if err := query.Find(&credentials).Error; err != nil {
		return nil, err
	}

	// 清除敏感数据（用于列表显示）
	for i := range credentials {
		credentials[i].Password = ""
		credentials[i].PrivateKey = ""
		credentials[i].Passphrase = ""
	}

	return credentials, nil
}

// UpdateCredential 更新凭据
func (cm *CredentialManager) UpdateCredential(id uint, updates *model.SSHCredential) error {
	var existing model.SSHCredential
	if err := cm.db.First(&existing, id).Error; err != nil {
		return err
	}

	// 验证更新数据
	if err := cm.validateCredential(updates); err != nil {
		return err
	}

	// 加密新的敏感数据
	if updates.Password != "" {
		encrypted, err := cm.Encrypt(updates.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		updates.Password = encrypted
	} else {
		updates.Password = existing.Password // 保持原有密码
	}

	if updates.PrivateKey != "" {
		encrypted, err := cm.Encrypt(updates.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt private key: %w", err)
		}
		updates.PrivateKey = encrypted
	} else {
		updates.PrivateKey = existing.PrivateKey // 保持原有私钥
	}

	if updates.Passphrase != "" {
		encrypted, err := cm.Encrypt(updates.Passphrase)
		if err != nil {
			return fmt.Errorf("failed to encrypt passphrase: %w", err)
		}
		updates.Passphrase = encrypted
	} else {
		updates.Passphrase = existing.Passphrase // 保持原有 passphrase
	}

	// 如果设置为默认凭据，取消其他默认凭据
	if updates.IsDefault && !existing.IsDefault {
		if err := cm.db.Model(&model.SSHCredential{}).
			Where("cluster_name = ? AND is_default = ? AND id != ?", updates.ClusterName, true, id).
			Update("is_default", false).Error; err != nil {
			return err
		}
	}

	// 更新数据库
	if err := cm.db.Model(&existing).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update credential: %w", err)
	}

	cm.logger.Infof("Updated SSH credential: %s", updates.Name)
	return nil
}

// DeleteCredential 删除凭据
func (cm *CredentialManager) DeleteCredential(id uint) error {
	var credential model.SSHCredential
	if err := cm.db.First(&credential, id).Error; err != nil {
		return err
	}

	if err := cm.db.Delete(&credential).Error; err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	cm.logger.Infof("Deleted SSH credential: %s", credential.Name)
	return nil
}

// GetCredentialByName 根据名称获取凭据
func (cm *CredentialManager) GetCredentialByName(name string) (*model.SSHCredential, error) {
	var credential model.SSHCredential
	if err := cm.db.Where("name = ?", name).First(&credential).Error; err != nil {
		return nil, err
	}

	return cm.GetCredential(credential.ID)
}

// GetDefaultCredential 获取指定集群的默认凭据
func (cm *CredentialManager) GetDefaultCredential(clusterName string) (*model.SSHCredential, error) {
	var credential model.SSHCredential
	if err := cm.db.Where("cluster_name = ? AND is_default = ?", clusterName, true).First(&credential).Error; err != nil {
		return nil, err
	}

	return cm.GetCredential(credential.ID)
}

// GetCredentialForNode 根据节点名称获取匹配的凭据
func (cm *CredentialManager) GetCredentialForNode(clusterName, nodeName string) (*model.SSHCredential, error) {
	var credentials []model.SSHCredential
	if err := cm.db.Where("cluster_name = ?", clusterName).Find(&credentials).Error; err != nil {
		return nil, err
	}

	// 查找匹配节点模式的凭据
	for _, cred := range credentials {
		if cred.NodePattern != "" {
			matched, err := regexp.MatchString(cred.NodePattern, nodeName)
			if err != nil {
				cm.logger.Errorf("Invalid node pattern regex '%s': %v", cred.NodePattern, err)
				continue
			}
			if matched {
				return cm.GetCredential(cred.ID)
			}
		}
	}

	// 如果没有匹配的，返回默认凭据
	return cm.GetDefaultCredential(clusterName)
}

// validateCredential 验证凭据数据
func (cm *CredentialManager) validateCredential(cred *model.SSHCredential) error {
	if cred.Name == "" {
		return errors.New("credential name is required")
	}

	if cred.Username == "" {
		return errors.New("username is required")
	}

	if cred.AuthType != "password" && cred.AuthType != "privatekey" {
		return errors.New("auth_type must be 'password' or 'privatekey'")
	}

	if cred.AuthType == "password" && cred.Password == "" {
		return errors.New("password is required for password auth")
	}

	if cred.AuthType == "privatekey" && cred.PrivateKey == "" {
		return errors.New("private key is required for privatekey auth")
	}

	if cred.Port == 0 {
		cred.Port = 22 // 默认端口
	}

	// 验证节点模式正则表达式
	if cred.NodePattern != "" {
		if _, err := regexp.Compile(cred.NodePattern); err != nil {
			return fmt.Errorf("invalid node pattern regex: %w", err)
		}
	}

	return nil
}
