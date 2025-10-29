package automation

import (
	"testing"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 迁移测试表
	err = db.AutoMigrate(&model.SSHCredential{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestCredentialManager_StoreAndRetrievePassword(t *testing.T) {
	db := setupTestDB(t)
	log := logger.NewLogger("test", "debug")
	cm := NewCredentialManager(db, log, "test-key-32-bytes-long-exactly")

	credential := &model.SSHCredential{
		Name:        "test-credential",
		Description: "Test SSH Credential",
		Username:    "testuser",
		AuthType:    "password",
		Password:    "test-password-123",
		Port:        22,
	}

	// 存储凭据
	err := cm.StoreCredential(credential)
	assert.NoError(t, err)
	assert.NotEmpty(t, credential.ID)
	assert.NotEqual(t, "test-password-123", credential.Password, "Password should be encrypted")

	// 检索凭据
	retrieved, err := cm.GetCredential(credential.ID)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", retrieved.Username)
	assert.Equal(t, "test-password-123", retrieved.Password, "Password should be decrypted")
	assert.Equal(t, "password", retrieved.AuthType)
}

func TestCredentialManager_StoreAndRetrievePrivateKey(t *testing.T) {
	db := setupTestDB(t)
	log := logger.NewLogger("test", "debug")
	cm := NewCredentialManager(db, log, "test-key-32-bytes-long-exactly")

	privateKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----`

	credential := &model.SSHCredential{
		Name:        "test-key-credential",
		Description: "Test SSH Key Credential",
		Username:    "keyuser",
		AuthType:    "privatekey",
		PrivateKey:  privateKey,
		Port:        22,
	}

	// 存储凭据
	err := cm.StoreCredential(credential)
	assert.NoError(t, err)
	assert.NotEmpty(t, credential.ID)
	assert.NotEqual(t, privateKey, credential.PrivateKey, "Private key should be encrypted")

	// 检索凭据
	retrieved, err := cm.GetCredential(credential.ID)
	assert.NoError(t, err)
	assert.Equal(t, "keyuser", retrieved.Username)
	assert.Equal(t, privateKey, retrieved.PrivateKey, "Private key should be decrypted")
	assert.Equal(t, "privatekey", retrieved.AuthType)
}

func TestCredentialManager_DeleteCredential(t *testing.T) {
	db := setupTestDB(t)
	log := logger.NewLogger("test", "debug")
	cm := NewCredentialManager(db, log, "test-key-32-bytes-long-exactly")

	credential := &model.SSHCredential{
		Name:     "delete-test",
		Username: "testuser",
		AuthType: "password",
		Password: "password",
		Port:     22,
	}

	// 存储凭据
	err := cm.StoreCredential(credential)
	assert.NoError(t, err)

	// 删除凭据
	err = cm.DeleteCredential(credential.ID)
	assert.NoError(t, err)

	// 验证已删除
	_, err = cm.GetCredential(credential.ID)
	assert.Error(t, err)
}

func TestCredentialManager_EncryptionConsistency(t *testing.T) {
	db := setupTestDB(t)
	log := logger.NewLogger("test", "debug")
	cm := NewCredentialManager(db, log, "test-key-32-bytes-long-exactly")

	originalPassword := "my-secret-password"

	// 第一次加密
	encrypted1, err := cm.encrypt(originalPassword)
	assert.NoError(t, err)

	// 第二次加密（应该产生不同的结果，因为使用了随机 nonce）
	encrypted2, err := cm.encrypt(originalPassword)
	assert.NoError(t, err)
	assert.NotEqual(t, encrypted1, encrypted2, "Encrypted values should differ due to random nonce")

	// 两次加密都应该能解密回原始值
	decrypted1, err := cm.decrypt(encrypted1)
	assert.NoError(t, err)
	assert.Equal(t, originalPassword, decrypted1)

	decrypted2, err := cm.decrypt(encrypted2)
	assert.NoError(t, err)
	assert.Equal(t, originalPassword, decrypted2)
}

