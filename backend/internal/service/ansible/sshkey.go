package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/crypto"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// SSHKeyService SSH 密钥服务
type SSHKeyService struct {
	db        *gorm.DB
	logger    *logger.Logger
	encryptor *crypto.Encryptor
}

// NewSSHKeyService 创建新的 SSH 密钥服务实例
func NewSSHKeyService(db *gorm.DB, logger *logger.Logger, encryptor *crypto.Encryptor) *SSHKeyService {
	return &SSHKeyService{
		db:        db,
		logger:    logger,
		encryptor: encryptor,
	}
}

// Create 创建 SSH 密钥
func (s *SSHKeyService) Create(req model.SSHKeyCreateRequest, userID uint) (*model.SSHKeyResponse, error) {
	// 验证输入
	if req.Type == model.SSHKeyTypePrivateKey && req.PrivateKey == "" {
		return nil, fmt.Errorf("private_key is required when type is private_key")
	}
	if req.Type == model.SSHKeyTypePassword && req.Password == "" {
		return nil, fmt.Errorf("password is required when type is password")
	}

	// 设置默认端口
	if req.Port == 0 {
		req.Port = 22
	}

	// 加密敏感信息
	encryptedPrivateKey, err := s.encryptor.Encrypt(req.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	encryptedPassphrase, err := s.encryptor.Encrypt(req.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt passphrase: %w", err)
	}

	encryptedPassword, err := s.encryptor.Encrypt(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	// 如果设置为默认密钥，取消其他密钥的默认状态
	if req.IsDefault {
		if err := s.db.Model(&model.AnsibleSSHKey{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return nil, fmt.Errorf("failed to unset other default keys: %w", err)
		}
	}

	// 创建 SSH 密钥
	sshKey := &model.AnsibleSSHKey{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Username:    req.Username,
		PrivateKey:  encryptedPrivateKey,
		Passphrase:  encryptedPassphrase,
		Password:    encryptedPassword,
		Port:        req.Port,
		IsDefault:   req.IsDefault,
		CreatedBy:   userID,
	}

	if err := s.db.Create(sshKey).Error; err != nil {
		return nil, fmt.Errorf("failed to create ssh key: %w", err)
	}

	s.logger.Infof("Created SSH key: %s (ID: %d) by user %d", sshKey.Name, sshKey.ID, userID)

	return sshKey.ToResponse(), nil
}

// List 列出 SSH 密钥
func (s *SSHKeyService) List(req model.SSHKeyListRequest) ([]*model.SSHKeyResponse, int64, error) {
	var keys []model.AnsibleSSHKey
	var total int64

	query := s.db.Model(&model.AnsibleSSHKey{})

	// 搜索过滤
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 类型过滤
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	// 计数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count ssh keys: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 查询
	if err := query.Order("is_default DESC, created_at DESC").Find(&keys).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list ssh keys: %w", err)
	}

	// 转换为响应
	responses := make([]*model.SSHKeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = key.ToResponse()
	}

	return responses, total, nil
}

// GetByID 根据 ID 获取 SSH 密钥
func (s *SSHKeyService) GetByID(id uint) (*model.SSHKeyResponse, error) {
	var key model.AnsibleSSHKey
	if err := s.db.First(&key, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ssh key not found")
		}
		return nil, fmt.Errorf("failed to get ssh key: %w", err)
	}

	return key.ToResponse(), nil
}

// GetDecryptedByID 获取解密后的 SSH 密钥（用于内部使用）
func (s *SSHKeyService) GetDecryptedByID(id uint) (*model.AnsibleSSHKey, error) {
	var key model.AnsibleSSHKey
	if err := s.db.First(&key, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ssh key not found")
		}
		return nil, fmt.Errorf("failed to get ssh key: %w", err)
	}

	// 解密敏感信息
	decryptedPrivateKey, err := s.encryptor.Decrypt(key.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}
	key.PrivateKey = decryptedPrivateKey

	decryptedPassphrase, err := s.encryptor.Decrypt(key.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt passphrase: %w", err)
	}
	key.Passphrase = decryptedPassphrase

	decryptedPassword, err := s.encryptor.Decrypt(key.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}
	key.Password = decryptedPassword

	return &key, nil
}

// GetDefault 获取默认 SSH 密钥
func (s *SSHKeyService) GetDefault() (*model.AnsibleSSHKey, error) {
	var key model.AnsibleSSHKey
	if err := s.db.Where("is_default = ?", true).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no default ssh key configured")
		}
		return nil, fmt.Errorf("failed to get default ssh key: %w", err)
	}

	// 解密
	return s.GetDecryptedByID(key.ID)
}

// Update 更新 SSH 密钥
func (s *SSHKeyService) Update(id uint, req model.SSHKeyUpdateRequest, userID uint) (*model.SSHKeyResponse, error) {
	var key model.AnsibleSSHKey
	if err := s.db.First(&key, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ssh key not found")
		}
		return nil, fmt.Errorf("failed to get ssh key: %w", err)
	}

	// 更新基本信息
	if req.Name != "" {
		key.Name = req.Name
	}
	if req.Description != "" {
		key.Description = req.Description
	}
	if req.Username != "" {
		key.Username = req.Username
	}
	if req.Port > 0 {
		key.Port = req.Port
	}

	// 更新加密的敏感信息
	if req.PrivateKey != "" {
		encrypted, err := s.encryptor.Encrypt(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
		key.PrivateKey = encrypted
	}

	if req.Passphrase != "" {
		encrypted, err := s.encryptor.Encrypt(req.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt passphrase: %w", err)
		}
		key.Passphrase = encrypted
	}

	if req.Password != "" {
		encrypted, err := s.encryptor.Encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		key.Password = encrypted
	}

	// 如果更新为默认密钥，取消其他密钥的默认状态
	if req.IsDefault && !key.IsDefault {
		if err := s.db.Model(&model.AnsibleSSHKey{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false).Error; err != nil {
			return nil, fmt.Errorf("failed to unset other default keys: %w", err)
		}
		key.IsDefault = true
	} else if !req.IsDefault && key.IsDefault {
		key.IsDefault = false
	}

	// 保存更新
	if err := s.db.Save(&key).Error; err != nil {
		return nil, fmt.Errorf("failed to update ssh key: %w", err)
	}

	s.logger.Infof("Updated SSH key: %s (ID: %d) by user %d", key.Name, key.ID, userID)

	return key.ToResponse(), nil
}

// Delete 删除 SSH 密钥
func (s *SSHKeyService) Delete(id uint, userID uint) error {
	var key model.AnsibleSSHKey
	if err := s.db.First(&key, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("ssh key not found")
		}
		return fmt.Errorf("failed to get ssh key: %w", err)
	}

	// 检查是否有清单正在使用此密钥
	var count int64
	if err := s.db.Model(&model.AnsibleInventory{}).Where("ssh_key_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check inventory usage: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("ssh key is in use by %d inventories, cannot delete", count)
	}

	// 删除
	if err := s.db.Delete(&key).Error; err != nil {
		return fmt.Errorf("failed to delete ssh key: %w", err)
	}

	s.logger.Infof("Deleted SSH key: %s (ID: %d) by user %d", key.Name, key.ID, userID)

	return nil
}

// TestConnection 测试 SSH 连接（可选功能）
func (s *SSHKeyService) TestConnection(id uint, testHost string) error {
	key, err := s.GetDecryptedByID(id)
	if err != nil {
		return err
	}

	// TODO: 实现实际的 SSH 连接测试
	// 这里可以使用 golang.org/x/crypto/ssh 包来测试连接

	s.logger.Infof("Testing SSH connection to %s with key %s", testHost, key.Name)

	return nil
}

