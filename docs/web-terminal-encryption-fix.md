# Web Terminal SSH密钥加密兼容性修复

## 问题描述

Web Terminal功能在连接节点时出现SSH密钥解密失败的错误：

```
ERROR: Failed to decrypt private key for key 1: encryption key must be 32 bytes for AES-256
```

## 根本原因

系统中存在两个不同的SSH密钥服务，它们使用了不兼容的加密方式：

### 1. Ansible Service (`internal/service/ansible`)
- 创建SSH密钥时使用
- 使用 `pkg/crypto/crypto.go` 的 `NewEncryptor`
- **自动使用SHA256哈希**将任意长度的密钥转换为32字节
- 默认密钥: `"default-encryption-key-change-in-production"` (43字节)
- 经过SHA256哈希后变为32字节

### 2. System SSH Key Service (`internal/service/sshkey`)
- 解密SSH密钥时使用
- **直接使用原始密钥字符串**，要求必须精确32字节
- 默认密钥: `"default-ssh-key-32-bytes-long!!"` (32字节)
- 不进行哈希处理

### 问题场景

1. 用户通过Ansible接口创建SSH密钥
2. Ansible Service使用SHA256哈希后的密钥加密
3. 密钥存储在 `ansible_ssh_keys` 表中
4. Web Terminal尝试连接节点
5. System SSH Key Service从 `ansible_ssh_keys` 回退读取
6. 尝试用不同的默认密钥解密 → **失败**

## 解决方案

修改 `internal/service/sshkey/sshkey.go`，使其与Ansible Service使用相同的加密方式：

### 1. 添加SHA256导入

```go
import (
    "crypto/sha256"
    // ... other imports
)
```

### 2. 修改NewService方法

```go
func NewService(db *gorm.DB, logger *logger.Logger, encryptionKey string) *Service {
    if encryptionKey == "" {
        logger.Warning("SSH key encryption key not configured, using default key (NOT SECURE for production)")
        // 使用与 ansible.Service 相同的默认密钥
        encryptionKey = "default-encryption-key-change-in-production"
    }
    
    // 使用 SHA256 哈希生成32字节密钥，与 pkg/crypto/crypto.go 保持一致
    hash := sha256.Sum256([]byte(encryptionKey))
    hashedKey := string(hash[:])
    
    return &Service{
        db:            db,
        logger:        logger,
        encryptionKey: hashedKey,
    }
}
```

### 3. 统一默认密钥

- Ansible Service: `"default-encryption-key-change-in-production"`
- System SSH Key Service: `"default-encryption-key-change-in-production"`
- 两者经过SHA256哈希后得到**相同的32字节密钥**

## 测试验证

重启服务后，Web Terminal应该能够：

1. ✅ 成功读取 `ansible_ssh_keys` 表中的密钥
2. ✅ 使用相同的哈希算法解密私钥
3. ✅ 建立SSH连接到节点

## 生产环境建议

在生产环境中，应该通过环境变量设置自定义加密密钥：

```bash
export SSH_ENCRYPTION_KEY="your-super-secret-key-at-least-32-chars-long"
```

或者（向后兼容）：

```bash
export ANSIBLE_ENCRYPTION_KEY="your-super-secret-key-at-least-32-chars-long"
```

这样，Ansible Service和System SSH Key Service都会使用相同的自定义密钥，经过SHA256哈希后保证兼容性。

## 相关文件

- `backend/internal/service/sshkey/sshkey.go` - System SSH Key Service
- `backend/internal/service/ansible/service.go` - Ansible Service初始化
- `backend/pkg/crypto/crypto.go` - 加密工具（使用SHA256）
- `backend/internal/service/services.go` - 服务初始化，读取环境变量

## 修复时间

- 2025-11-22 18:47:04
- Version: 修复前端UI和后端SSH密钥解密兼容性

