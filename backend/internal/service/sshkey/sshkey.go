package sshkey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// Service SSH 密钥服务
type Service struct {
	db            *gorm.DB
	logger        *logger.Logger
	encryptionKey string // AES 加密密钥
}

// NewService 创建 SSH 密钥服务
func NewService(db *gorm.DB, logger *logger.Logger, encryptionKey string) *Service {
	// 如果未提供加密密钥，生成一个默认密钥（生产环境必须配置）
	if encryptionKey == "" {
		logger.Warning("SSH key encryption key not configured, using default key (NOT SECURE for production)")
		encryptionKey = "default-ssh-key-32-bytes-long!!" // 32 bytes for AES-256
	}
	
	return &Service{
		db:            db,
		logger:        logger,
		encryptionKey: encryptionKey,
	}
}

// List 获取 SSH 密钥列表（查询system_ssh_keys表，如果为空则回退到ansible_ssh_keys表）
func (s *Service) List(req model.SSHKeyListRequest) ([]model.SSHKeyResponse, int64, error) {
	var systemKeys []model.SystemSSHKey
	var total int64

	// 先查询 system_ssh_keys 表
	query := s.db.Model(&model.SystemSSHKey{})

	// 关键字搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 类型过滤
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 查询
	if err := query.Order("created_at DESC").Find(&systemKeys).Error; err != nil {
		return nil, 0, err
	}

	// 如果 system_ssh_keys 表为空，尝试从 ansible_ssh_keys 表读取（向后兼容）
	if len(systemKeys) == 0 {
		s.logger.Info("system_ssh_keys is empty, falling back to ansible_ssh_keys")
		var ansibleKeys []model.AnsibleSSHKey
		
		ansibleQuery := s.db.Model(&model.AnsibleSSHKey{})
		if req.Keyword != "" {
			ansibleQuery = ansibleQuery.Where("name LIKE ? OR description LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
		}
		if req.Type != "" {
			ansibleQuery = ansibleQuery.Where("type = ?", req.Type)
		}
		
		if err := ansibleQuery.Count(&total).Error; err != nil {
			return nil, 0, err
		}
		
		if req.Page > 0 && req.PageSize > 0 {
			offset := (req.Page - 1) * req.PageSize
			ansibleQuery = ansibleQuery.Offset(offset).Limit(req.PageSize)
		}
		
		if err := ansibleQuery.Order("created_at DESC").Find(&ansibleKeys).Error; err != nil {
			return nil, 0, err
		}
		
		// 转换 AnsibleSSHKey 为响应格式
		responses := make([]model.SSHKeyResponse, len(ansibleKeys))
		for i, key := range ansibleKeys {
			responses[i] = *key.ToResponse()
		}
		
		return responses, total, nil
	}

	// 转换 SystemSSHKey 为响应格式
	responses := make([]model.SSHKeyResponse, len(systemKeys))
	for i, key := range systemKeys {
		responses[i] = *key.ToResponse()
	}

	return responses, total, nil
}

// GetByID 根据 ID 获取 SSH 密钥
func (s *Service) GetByID(id uint) (*model.SSHKeyResponse, error) {
	var key model.SystemSSHKey
	if err := s.db.First(&key, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("SSH key not found")
		}
		return nil, err
	}

	return key.ToResponse(), nil
}

// GetDecryptedByID 获取解密后的 SSH 密钥（用于实际使用，先查system_ssh_keys，如无则查ansible_ssh_keys）
func (s *Service) GetDecryptedByID(id uint) (*model.SystemSSHKey, error) {
	var key model.SystemSSHKey
	
	// 先查询 system_ssh_keys 表
	err := s.db.First(&key, id).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	
	// 如果 system_ssh_keys 表没有该密钥，尝试从 ansible_ssh_keys 表读取
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Infof("Key %d not found in system_ssh_keys, checking ansible_ssh_keys", id)
		
		var ansibleKey model.AnsibleSSHKey
		if err := s.db.First(&ansibleKey, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("SSH key not found")
			}
			return nil, err
		}
		
		// 将 AnsibleSSHKey 转换为 SystemSSHKey 格式
		key = model.SystemSSHKey{
			ID:          ansibleKey.ID,
			Name:        ansibleKey.Name,
			Description: ansibleKey.Description,
			Type:        ansibleKey.Type,
			Username:    ansibleKey.Username,
			PrivateKey:  ansibleKey.PrivateKey,
			Passphrase:  ansibleKey.Passphrase,
			Password:    ansibleKey.Password,
			Port:        ansibleKey.Port,
			IsDefault:   ansibleKey.IsDefault,
			CreatedBy:   ansibleKey.CreatedBy,
			CreatedAt:   ansibleKey.CreatedAt,
			UpdatedAt:   ansibleKey.UpdatedAt,
		}
	}

	// 解密敏感信息
	if key.PrivateKey != "" {
		decrypted, err := s.decrypt(key.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}
		key.PrivateKey = decrypted
	}

	if key.Passphrase != "" {
		decrypted, err := s.decrypt(key.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt passphrase: %w", err)
		}
		key.Passphrase = decrypted
	}

	if key.Password != "" {
		decrypted, err := s.decrypt(key.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
		key.Password = decrypted
	}

	return &key, nil
}

// Create 创建 SSH 密钥
func (s *Service) Create(req model.SSHKeyCreateRequest, userID uint) (*model.SSHKeyResponse, error) {
	// 检查名称唯一性
	var count int64
	if err := s.db.Model(&model.SystemSSHKey{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("SSH key name already exists")
	}

	// 验证必填字段
	if req.Type == model.SSHKeyTypePrivateKey && req.PrivateKey == "" {
		return nil, fmt.Errorf("private key is required for private_key type")
	}
	if req.Type == model.SSHKeyTypePassword && req.Password == "" {
		return nil, fmt.Errorf("password is required for password type")
	}

	// 设置默认端口
	if req.Port == 0 {
		req.Port = 22
	}

	// 加密敏感信息
	key := &model.SystemSSHKey{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Username:    req.Username,
		Port:        req.Port,
		IsDefault:   req.IsDefault,
		CreatedBy:   userID,
	}

	if req.PrivateKey != "" {
		encrypted, err := s.encrypt(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
		key.PrivateKey = encrypted
	}

	if req.Passphrase != "" {
		encrypted, err := s.encrypt(req.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt passphrase: %w", err)
		}
		key.Passphrase = encrypted
	}

	if req.Password != "" {
		encrypted, err := s.encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		key.Password = encrypted
	}

	// 如果设置为默认，取消其他密钥的默认状态
	if key.IsDefault {
		if err := s.db.Model(&model.SystemSSHKey{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return nil, err
		}
	}

	// 创建密钥
	if err := s.db.Create(key).Error; err != nil {
		return nil, err
	}

	return key.ToResponse(), nil
}

// Update 更新 SSH 密钥
func (s *Service) Update(id uint, req model.SSHKeyUpdateRequest, userID uint) (*model.SSHKeyResponse, error) {
	var key model.SystemSSHKey
	if err := s.db.First(&key, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("SSH key not found")
		}
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})

	if req.Name != "" {
		// 检查名称唯一性
		var count int64
		if err := s.db.Model(&model.SystemSSHKey{}).Where("name = ? AND id != ?", req.Name, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("SSH key name already exists")
		}
		updates["name"] = req.Name
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if req.Username != "" {
		updates["username"] = req.Username
	}

	if req.Port > 0 {
		updates["port"] = req.Port
	}

	if req.PrivateKey != "" {
		encrypted, err := s.encrypt(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
		updates["private_key"] = encrypted
	}

	if req.Passphrase != "" {
		encrypted, err := s.encrypt(req.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt passphrase: %w", err)
		}
		updates["passphrase"] = encrypted
	}

	if req.Password != "" {
		encrypted, err := s.encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		updates["password"] = encrypted
	}

	if req.IsDefault {
		// 如果设置为默认，取消其他密钥的默认状态
		if err := s.db.Model(&model.SystemSSHKey{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false).Error; err != nil {
			return nil, err
		}
		updates["is_default"] = true
	}

	// 执行更新
	if len(updates) > 0 {
		if err := s.db.Model(&key).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// 重新查询返回最新数据
	if err := s.db.First(&key, id).Error; err != nil {
		return nil, err
	}

	return key.ToResponse(), nil
}

// Delete 删除 SSH 密钥（软删除）
func (s *Service) Delete(id uint) error {
	var key model.SystemSSHKey
	if err := s.db.First(&key, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("SSH key not found")
		}
		return err
	}

	// 检查是否有 Ansible Inventory 正在使用此密钥
	var count int64
	if err := s.db.Model(&model.AnsibleInventory{}).Where("ssh_key_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete SSH key: %d inventories are using it", count)
	}

	return s.db.Delete(&key).Error
}

// GetDefault 获取默认 SSH 密钥（先查system_ssh_keys，如无则查ansible_ssh_keys）
func (s *Service) GetDefault() (*model.SystemSSHKey, error) {
	var key model.SystemSSHKey
	
	// 先查询 system_ssh_keys 表
	err := s.db.Where("is_default = ?", true).First(&key).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	
	// 如果 system_ssh_keys 表没有默认密钥，尝试从 ansible_ssh_keys 表读取
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Info("No default key in system_ssh_keys, checking ansible_ssh_keys")
		
		var ansibleKey model.AnsibleSSHKey
		if err := s.db.Where("is_default = ?", true).First(&ansibleKey).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil // 没有默认密钥不是错误
			}
			return nil, err
		}
		
		// 将 AnsibleSSHKey 转换为 SystemSSHKey 格式
		key = model.SystemSSHKey{
			ID:          ansibleKey.ID,
			Name:        ansibleKey.Name,
			Description: ansibleKey.Description,
			Type:        ansibleKey.Type,
			Username:    ansibleKey.Username,
			PrivateKey:  ansibleKey.PrivateKey,
			Passphrase:  ansibleKey.Passphrase,
			Password:    ansibleKey.Password,
			Port:        ansibleKey.Port,
			IsDefault:   ansibleKey.IsDefault,
			CreatedBy:   ansibleKey.CreatedBy,
			CreatedAt:   ansibleKey.CreatedAt,
			UpdatedAt:   ansibleKey.UpdatedAt,
		}
	}

	// 解密敏感信息
	if key.PrivateKey != "" {
		decrypted, err := s.decrypt(key.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}
		key.PrivateKey = decrypted
	}

	if key.Passphrase != "" {
		decrypted, err := s.decrypt(key.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt passphrase: %w", err)
		}
		key.Passphrase = decrypted
	}

	if key.Password != "" {
		decrypted, err := s.decrypt(key.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
		key.Password = decrypted
	}

	return &key, nil
}

// encrypt 加密数据
func (s *Service) encrypt(plaintext string) (string, error) {
	// 使用 AES-256-GCM 加密
	key := []byte(s.encryptionKey)
	if len(key) != 32 {
		return "", fmt.Errorf("encryption key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 解密数据
func (s *Service) decrypt(ciphertext string) (string, error) {
	key := []byte(s.encryptionKey)
	if len(key) != 32 {
		return "", fmt.Errorf("encryption key must be 32 bytes for AES-256")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

