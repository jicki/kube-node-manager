# Ansible SSH 密钥数据库存储实现

## 更新日期

2025-10-30

## 概述

将 Ansible 模块的 SSH 密钥认证方式从**文件系统挂载**改为**数据库加密存储**，提供更灵活、更安全的密钥管理方式。

## 变更摘要

### 之前的方式
- SSH 密钥通过 Docker Volume 挂载到容器
- 所有任务使用相同的 SSH 密钥
- 密钥文件位于 `/root/.ssh/`
- 需要手动管理多个密钥文件

### 现在的方式
- ✅ SSH 密钥加密存储在数据库中
- ✅ 每个主机清单可以关联不同的 SSH 密钥
- ✅ 支持多种认证方式（私钥/密码）
- ✅ 提供 Web UI 管理界面
- ✅ 自动加密/解密敏感信息

## 架构设计

### 1. 数据模型

#### AnsibleSSHKey (SSH 密钥表)

```sql
CREATE TABLE ansible_ssh_keys (
    id INTEGER PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,           -- 密钥名称
    description TEXT,                             -- 描述
    type VARCHAR(50) NOT NULL,                    -- private_key 或 password
    username VARCHAR(255) NOT NULL,               -- SSH 用户名
    private_key TEXT,                             -- 加密的私钥内容
    passphrase TEXT,                              -- 加密的密钥密码
    password TEXT,                                -- 加密的 SSH 密码
    port INTEGER DEFAULT 22,                      -- SSH 端口
    is_default BOOLEAN DEFAULT FALSE,             -- 是否为默认密钥
    created_by INTEGER NOT NULL,                  -- 创建用户 ID
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
```

#### 更新的 AnsibleInventory

```sql
ALTER TABLE ansible_inventories 
ADD COLUMN ssh_key_id INTEGER REFERENCES ansible_ssh_keys(id);
```

### 2. 加密机制

使用 **AES-256-GCM** 加密算法保护敏感信息：

- **加密内容**: 私钥、密钥密码、SSH 密码
- **加密密钥来源**: 
  1. 环境变量 `ANSIBLE_ENCRYPTION_KEY`
  2. 配置文件 `encryption_key`
  3. 默认值（仅开发环境）

**加密流程：**
```
明文 → AES-256-GCM 加密 → Base64 编码 → 存储到数据库
```

**解密流程：**
```
数据库 → Base64 解码 → AES-256-GCM 解密 → 明文（仅在内存中）
```

### 3. 组件架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend (Vue.js)                     │
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────┐│
│  │ SSH Key Manager │  │ Inventory Editor │  │ Task Center ││
│  └────────┬────────┘  └────────┬─────────┘  └──────┬──────┘│
└───────────┼────────────────────┼────────────────────┼───────┘
            │                    │                    │
            ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────┐
│                        Backend (Go)                          │
│  ┌──────────────┐  ┌─────────────────┐  ┌────────────────┐ │
│  │ SSH Key      │  │  Inventory      │  │  Task          │ │
│  │ Handler      │  │  Handler        │  │  Executor      │ │
│  └──────┬───────┘  └────────┬────────┘  └────────┬───────┘ │
│         │                   │                     │         │
│  ┌──────▼──────────────────┴─────────────────────▼──────┐  │
│  │              Ansible Service Layer                    │  │
│  │  ┌──────────────┐  ┌───────────────┐  ┌────────────┐ │  │
│  │  │ SSHKeyService│  │ InventoryService│  │  Executor  │ │  │
│  │  │ (加密/解密)  │  │                 │  │            │ │  │
│  │  └──────┬───────┘  └────────┬────────┘  └─────┬──────┘ │  │
│  └─────────┼────────────────────┼───────────────────┼──────┘  │
└────────────┼────────────────────┼───────────────────┼─────────┘
             │                    │                   │
             ▼                    ▼                   ▼
┌─────────────────────────────────────────────────────────────┐
│                     Database (SQLite/PostgreSQL)             │
│  ┌────────────────┐  ┌──────────────────┐  ┌──────────────┐│
│  │ ansible_ssh_   │  │ ansible_         │  │ ansible_     ││
│  │ keys (加密)    │  │ inventories      │  │ tasks        ││
│  └────────────────┘  └──────────────────┘  └──────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## API 接口

### SSH 密钥管理

#### 1. 列出 SSH 密钥
```http
GET /api/v1/ansible/ssh-keys?page=1&page_size=10
Authorization: Bearer <admin-token>

Response:
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "name": "production-key",
      "description": "生产环境 SSH 密钥",
      "type": "private_key",
      "username": "root",
      "port": 22,
      "is_default": true,
      "has_private_key": true,
      "has_passphrase": false,
      "has_password": false,
      "created_by": 1,
      "created_at": "2025-10-30T10:00:00Z"
    }
  ],
  "total": 1
}
```
**注意**: 响应中不包含实际的密钥内容，只显示是否存在。

#### 2. 创建 SSH 密钥
```http
POST /api/v1/ansible/ssh-keys
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "name": "production-key",
  "description": "生产环境 SSH 密钥",
  "type": "private_key",
  "username": "root",
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----",
  "passphrase": "optional-passphrase",
  "port": 22,
  "is_default": true
}

Response:
{
  "code": 0,
  "message": "SSH key created successfully",
  "data": {
    "id": 1,
    "name": "production-key",
    ...
  }
}
```

#### 3. 更新 SSH 密钥
```http
PUT /api/v1/ansible/ssh-keys/:id
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "name": "updated-name",
  "description": "更新后的描述"
}
```

#### 4. 删除 SSH 密钥
```http
DELETE /api/v1/ansible/ssh-keys/:id
Authorization: Bearer <admin-token>

Response:
{
  "code": 0,
  "message": "SSH key deleted successfully"
}
```
**注意**: 如果有主机清单正在使用此密钥，删除会失败。

#### 5. 测试 SSH 连接
```http
POST /api/v1/ansible/ssh-keys/:id/test
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "host": "192.168.1.100"
}

Response:
{
  "code": 0,
  "message": "Connection test successful"
}
```

### 主机清单关联 SSH 密钥

#### 创建主机清单时关联密钥
```http
POST /api/v1/ansible/inventories
Authorization: Bearer <admin-token>
Content-Type: application/json

{
  "name": "production-hosts",
  "source_type": "manual",
  "ssh_key_id": 1,           // ← 关联 SSH 密钥
  "content": "[webservers]\n192.168.1.100\n192.168.1.101"
}
```

## 使用流程

### 1. 初始化：设置加密密钥

**方式 A: 环境变量（推荐）**
```bash
export ANSIBLE_ENCRYPTION_KEY="your-very-secure-encryption-key-at-least-32-chars"
```

**方式 B: Docker Compose**
```yaml
services:
  kube-node-manager:
    image: your-registry/kube-node-manager:latest
    environment:
      - ANSIBLE_ENCRYPTION_KEY=${ANSIBLE_ENCRYPTION_KEY}
    # 不再需要挂载 SSH 密钥文件
```

**方式 C: Kubernetes Secret**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ansible-config
type: Opaque
data:
  ANSIBLE_ENCRYPTION_KEY: <base64-encoded-key>
---
apiVersion: v1
kind: Deployment
spec:
  containers:
  - name: app
    envFrom:
    - secretRef:
        name: ansible-config
```

### 2. 创建 SSH 密钥

**通过 Web UI：**
1. 登录系统（admin 账号）
2. 导航到 "Ansible" → "SSH 密钥管理"
3. 点击"创建密钥"
4. 填写信息：
   - 名称：例如 "生产环境密钥"
   - 类型：选择"私钥"或"密码"
   - 用户名：root（或其他）
   - 私钥：粘贴私钥内容
   - 设置为默认（可选）
5. 保存

**通过 API：**
```bash
curl -X POST http://localhost:8080/api/v1/ansible/ssh-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-key",
    "type": "private_key",
    "username": "root",
    "private_key": "'"$(cat ~/.ssh/id_rsa)"'",
    "port": 22,
    "is_default": true
  }'
```

### 3. 创建主机清单并关联密钥

```bash
curl -X POST http://localhost:8080/api/v1/ansible/inventories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-hosts",
    "source_type": "manual",
    "ssh_key_id": 1,
    "content": "[webservers]\n192.168.1.100\n192.168.1.101"
  }'
```

### 4. 执行任务

任务执行时会自动：
1. 从数据库读取关联的 SSH 密钥
2. 解密密钥内容
3. 创建临时密钥文件（权限 600）
4. 执行 `ansible-playbook --private-key /tmp/ssh-key-xxx.pem`
5. 删除临时密钥文件

```bash
curl -X POST http://localhost:8080/api/v1/ansible/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "系统更新",
    "template_id": 1,
    "inventory_id": 1,
    "cluster_id": 1
  }'
```

## 安全性

### 1. 加密强度

- **算法**: AES-256-GCM (Galois/Counter Mode)
- **密钥长度**: 256 bits (32 bytes)
- **认证加密**: 防止密文被篡改
- **随机 Nonce**: 每次加密使用唯一的 nonce

### 2. 密钥派生

使用 SHA-256 从用户提供的密码派生 256-bit 密钥：
```go
hash := sha256.Sum256([]byte(secretKey))
aesKey := hash[:]  // 32 bytes
```

### 3. 最佳实践

**DO ✅:**
- 使用至少 32 字符的强密码作为加密密钥
- 通过环境变量或 Secrets 管理加密密钥
- 定期轮换加密密钥（需要重新加密所有密钥）
- 限制 SSH 密钥的访问权限（仅 admin）
- 启用审计日志记录所有密钥操作

**DON'T ❌:**
- 不要在代码中硬编码加密密钥
- 不要将加密密钥提交到版本控制
- 不要在日志中记录解密后的密钥内容
- 不要在多个环境使用相同的加密密钥
- 不要通过 HTTP（非 HTTPS）传输密钥

### 4. 权限控制

- ✅ **所有 SSH 密钥操作都需要 admin 权限**
- ✅ API 响应永远不包含解密后的密钥内容
- ✅ 密钥只在任务执行时临时解密到内存
- ✅ 临时密钥文件权限设置为 600
- ✅ 临时文件在任务完成后立即删除

## 迁移指南

### 从文件系统迁移到数据库

如果你之前使用文件系统挂载 SSH 密钥，可以按以下步骤迁移：

#### 步骤 1: 备份现有密钥
```bash
# 从容器复制密钥文件
docker cp <container-id>:/root/.ssh/id_rsa ./backup/
docker cp <container-id>:/root/.ssh/id_rsa.pub ./backup/
```

#### 步骤 2: 创建数据库密钥记录
```bash
# 读取密钥内容
PRIVATE_KEY=$(cat ./backup/id_rsa)

# 创建密钥记录
curl -X POST http://localhost:8080/api/v1/ansible/ssh-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"migrated-key\",
    \"type\": \"private_key\",
    \"username\": \"root\",
    \"private_key\": \"$PRIVATE_KEY\",
    \"port\": 22,
    \"is_default\": true
  }"
```

#### 步骤 3: 更新现有清单
```bash
# 获取所有清单
INVENTORIES=$(curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:8080/api/v1/ansible/inventories")

# 为每个清单关联 SSH 密钥
curl -X PUT "http://localhost:8080/api/v1/ansible/inventories/<id>" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ssh_key_id": 1
  }'
```

#### 步骤 4: 移除文件系统挂载

**Docker Compose:**
```yaml
services:
  kube-node-manager:
    # volumes:
    #   - ~/.ssh:/root/.ssh:ro  # ← 移除这行
    environment:
      - ANSIBLE_ENCRYPTION_KEY=${ANSIBLE_ENCRYPTION_KEY}  # ← 添加这行
```

#### 步骤 5: 测试
```bash
# 创建测试任务
curl -X POST http://localhost:8080/api/v1/ansible/tasks \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "测试任务",
    "template_id": <template-id>,
    "inventory_id": <inventory-id>
  }'

# 查看任务日志确认成功
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "http://localhost:8080/api/v1/ansible/tasks/<task-id>/logs"
```

## 故障排查

### 1. 加密密钥错误

**现象**: 启动时看到警告
```
Using default encryption key for SSH credentials. Please set a secure encryption key in production!
```

**解决**:
```bash
# 设置环境变量
export ANSIBLE_ENCRYPTION_KEY="your-secure-key-at-least-32-characters-long"

# 或在配置文件中设置（如果支持）
echo "encryption_key: your-secure-key" >> config.yaml
```

### 2. 解密失败

**现象**: 任务执行失败，日志显示 "failed to decrypt private key"

**原因**: 加密密钥已更改

**解决**: 
- 恢复原加密密钥
- 或重新创建所有 SSH 密钥

### 3. SSH 连接失败

**现象**: 任务日志显示 "Permission denied (publickey)"

**排查步骤**:
```bash
# 1. 测试 SSH 密钥
curl -X POST "http://localhost:8080/api/v1/ansible/ssh-keys/<id>/test" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"host": "target-host"}'

# 2. 检查目标主机的 authorized_keys
docker exec <container-id> ssh root@target-host "cat ~/.ssh/authorized_keys"

# 3. 检查密钥内容（确认没有被截断）
# 在 Web UI 查看密钥的 "has_private_key" 字段
```

### 4. 权限被拒绝

**现象**: API 返回 403 Forbidden

**解决**: 确认当前用户是 admin 角色
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/auth/profile

# 如果不是 admin，请联系管理员升级权限
```

## 性能影响

### 加密/解密性能

| 操作 | 时间 | 说明 |
|------|------|------|
| 加密 1KB 数据 | ~0.1ms | 创建密钥时 |
| 解密 1KB 数据 | ~0.1ms | 任务执行时 |
| 创建临时文件 | ~1ms | 任务执行时 |

**结论**: 加密/解密开销可忽略不计。

### 数据库影响

- 每个 SSH 密钥约占用 5-10 KB 空间
- 100 个密钥约 0.5-1 MB
- 对数据库性能无明显影响

## 未来改进

1. **密钥轮换**
   - 自动轮换加密密钥
   - 无缝重新加密现有数据

2. **多因素认证**
   - SSH 密钥 + OTP
   - 临时密钥（有效期限制）

3. **密钥审计**
   - 记录密钥使用历史
   - 密钥访问统计

4. **外部密钥管理**
   - 集成 HashiCorp Vault
   - 集成 AWS Secrets Manager
   - 集成 Azure Key Vault

5. **密钥备份与恢复**
   - 导出/导入加密密钥
   - 灾难恢复流程

## 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Ansible 安全和部署更新](./ansible-security-and-deployment-update.md)
- [Ansible 快速参考](./ansible-quick-reference.md)

## 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|------|------|----------|------|
| 2025-10-30 | 1.0.0 | 初始版本：SSH 密钥数据库存储 | System |

