# Ansible 主机清单自动使用 SSH 密钥用户名

## 功能说明

在从 K8s 集群生成 Ansible 主机清单时，系统会自动从关联的 SSH 密钥中获取用户名，并将其设置为 `ansible_user`。

## 实现逻辑

### 1. 用户名获取规则

```
如果指定了 SSH 密钥 → 使用 SSH 密钥的 Username 字段
否则 → 使用默认值 "root"
```

### 2. 代码实现位置

**文件**: `backend/internal/service/ansible/inventory.go`

#### 2.1 从 SSH 密钥获取用户名

```go
// GenerateFromK8s 从 K8s 集群动态生成主机清单
func (s *InventoryService) GenerateFromK8s(req model.GenerateInventoryRequest, userID uint) (*model.AnsibleInventory, error) {
    // 获取 SSH 密钥的用户名（如果指定了 SSH 密钥）
    ansibleUser := "root" // 默认用户名
    if req.SSHKeyID != nil {
        var sshKey model.AnsibleSSHKey
        if err := s.db.First(&sshKey, *req.SSHKeyID).Error; err != nil {
            if err == gorm.ErrRecordNotFound {
                s.logger.Warningf("SSH key %d not found, using default user 'root'", *req.SSHKeyID)
            } else {
                s.logger.Errorf("Failed to get SSH key %d: %v", *req.SSHKeyID, err)
            }
        } else {
            ansibleUser = sshKey.Username
            s.logger.Infof("Using SSH key username: %s", ansibleUser)
        }
    }
    
    // ... 生成 inventory
}
```

#### 2.2 生成 INI 格式清单

```go
func (s *InventoryService) generateINIInventory(nodes []k8s.NodeInfo, clusterName string, ansibleUser string) string {
    var builder strings.Builder
    builder.WriteString("[all]\n")
    
    for _, node := range nodes {
        ip := node.InternalIP
        if ip == "" {
            ip = node.ExternalIP
        }
        
        // 使用从 SSH 密钥获取的用户名
        builder.WriteString(fmt.Sprintf("%s ansible_host=%s ansible_user=%s\n", 
            node.Name, ip, ansibleUser))
    }
    
    // 写入全局变量
    builder.WriteString("\n[all:vars]\n")
    builder.WriteString("ansible_python_interpreter=/usr/bin/python3\n")
    builder.WriteString("ansible_ssh_common_args='-o StrictHostKeyChecking=no'\n")
    
    return builder.String()
}
```

#### 2.3 生成结构化主机数据

```go
func (s *InventoryService) generateHostsData(nodes []k8s.NodeInfo, ansibleUser string) model.HostsData {
    hostsData := make(model.HostsData)
    hosts := make([]map[string]interface{}, 0, len(nodes))
    
    for _, node := range nodes {
        host := map[string]interface{}{
            "name":         node.Name,
            "ip":           ip,
            "ansible_user": ansibleUser, // 添加 ansible_user 信息
            // ... 其他字段
        }
        hosts = append(hosts, host)
    }
    
    hostsData["hosts"] = hosts
    hostsData["total"] = len(hosts)
    
    return hostsData
}
```

## 使用示例

### 示例 1: 使用 SSH 密钥（推荐）

#### 步骤 1: 创建 SSH 密钥

```bash
curl -X POST http://localhost:8080/api/ansible/ssh-keys \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ubuntu-key",
    "description": "Ubuntu 用户密钥",
    "type": "private_key",
    "username": "ubuntu",
    "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----",
    "port": 22
  }'
```

响应：
```json
{
  "id": 1,
  "name": "ubuntu-key",
  "username": "ubuntu"
}
```

#### 步骤 2: 从集群生成清单（指定 SSH 密钥）

```bash
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-nodes",
    "description": "生产环境节点",
    "cluster_id": 1,
    "ssh_key_id": 1,
    "node_labels": {
      "env": "production"
    }
  }'
```

生成的清单内容：
```ini
[all]
node-1 ansible_host=10.0.1.10 ansible_user=ubuntu
node-2 ansible_host=10.0.1.11 ansible_user=ubuntu
node-3 ansible_host=10.0.1.12 ansible_user=ubuntu

[all:vars]
ansible_python_interpreter=/usr/bin/python3
ansible_ssh_common_args='-o StrictHostKeyChecking=no'
```

### 示例 2: 不指定 SSH 密钥（使用默认 root）

```bash
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -H "Content-Type: application/json" \
  -d '{
    "name": "all-nodes",
    "description": "所有节点",
    "cluster_id": 1,
    "node_labels": {}
  }'
```

生成的清单内容（使用默认 root 用户）：
```ini
[all]
node-1 ansible_host=10.0.1.10 ansible_user=root
node-2 ansible_host=10.0.1.11 ansible_user=root
node-3 ansible_host=10.0.1.12 ansible_user=root

[all:vars]
ansible_python_interpreter=/usr/bin/python3
ansible_ssh_common_args='-o StrictHostKeyChecking=no'
```

### 示例 3: 不同用户的 SSH 密钥

#### CentOS/RHEL 节点（root 用户）
```bash
# 创建 root 密钥
curl -X POST http://localhost:8080/api/ansible/ssh-keys \
  -d '{
    "name": "centos-root-key",
    "username": "root",
    "type": "private_key",
    "private_key": "..."
  }'

# 生成清单
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -d '{
    "name": "centos-nodes",
    "cluster_id": 1,
    "ssh_key_id": 2,
    "node_labels": {"os": "centos"}
  }'
```

#### Ubuntu 节点（ubuntu 用户）
```bash
# 创建 ubuntu 密钥
curl -X POST http://localhost:8080/api/ansible/ssh-keys \
  -d '{
    "name": "ubuntu-key",
    "username": "ubuntu",
    "type": "private_key",
    "private_key": "..."
  }'

# 生成清单
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -d '{
    "name": "ubuntu-nodes",
    "cluster_id": 1,
    "ssh_key_id": 1,
    "node_labels": {"os": "ubuntu"}
  }'
```

## API 接口

### 从集群生成主机清单

**端点**: `POST /api/ansible/inventories/generate`

**请求体**:
```json
{
  "name": "清单名称",
  "description": "清单描述",
  "cluster_id": 1,
  "ssh_key_id": 1,          // 可选，指定 SSH 密钥 ID
  "node_labels": {          // 可选，过滤节点标签
    "key": "value"
  }
}
```

**响应**:
```json
{
  "id": 1,
  "name": "清单名称",
  "source_type": "k8s",
  "cluster_id": 1,
  "ssh_key_id": 1,
  "content": "[all]\nnode-1 ansible_host=10.0.1.10 ansible_user=ubuntu\n...",
  "hosts_data": {
    "hosts": [
      {
        "name": "node-1",
        "ip": "10.0.1.10",
        "ansible_user": "ubuntu",
        "roles": ["worker"],
        "labels": {"env": "prod"},
        "version": "v1.28.0"
      }
    ],
    "total": 1
  }
}
```

## 数据模型

### AnsibleSSHKey
```go
type AnsibleSSHKey struct {
    ID          uint       `json:"id"`
    Name        string     `json:"name"`
    Username    string     `json:"username"`     // SSH 用户名
    Type        SSHKeyType `json:"type"`         // private_key 或 password
    PrivateKey  string     `json:"-"`            // 私钥内容（加密存储）
    Password    string     `json:"-"`            // SSH 密码（加密存储）
    Port        int        `json:"port"`         // SSH 端口，默认 22
    // ...
}
```

### GenerateInventoryRequest
```go
type GenerateInventoryRequest struct {
    Name        string            `json:"name" binding:"required"`
    Description string            `json:"description"`
    ClusterID   uint              `json:"cluster_id" binding:"required"`
    SSHKeyID    *uint             `json:"ssh_key_id"`         // SSH 密钥 ID（可选）
    NodeLabels  map[string]string `json:"node_labels"`        // 节点标签过滤
}
```

## 工作流程

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 用户创建 SSH 密钥                                           │
│    - 指定用户名（如 ubuntu, root, admin 等）                    │
│    - 保存私钥或密码                                            │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. 从集群生成清单                                              │
│    - 指定集群 ID                                              │
│    - 指定 SSH 密钥 ID（可选）                                  │
│    - 指定节点标签过滤条件（可选）                               │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. 系统获取 SSH 密钥用户名                                      │
│    - 如果指定了 SSH 密钥 → 从密钥获取用户名                      │
│    - 如果未指定 → 使用默认值 "root"                             │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. 生成 Inventory                                            │
│    - INI 格式：ansible_user=<username>                        │
│    - JSON 格式：hosts_data.hosts[].ansible_user               │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. 执行 Ansible 任务                                          │
│    - 使用清单关联的 SSH 密钥                                   │
│    - 使用清单中指定的 ansible_user                             │
└─────────────────────────────────────────────────────────────┘
```

## 最佳实践

### 1. 为不同操作系统创建不同的 SSH 密钥

```bash
# Ubuntu/Debian 节点
{
  "name": "ubuntu-nodes-key",
  "username": "ubuntu",
  ...
}

# CentOS/RHEL 节点
{
  "name": "centos-nodes-key",
  "username": "root",
  ...
}

# 自定义用户
{
  "name": "app-user-key",
  "username": "appuser",
  ...
}
```

### 2. 使用标签过滤创建不同的清单

```bash
# 生产环境（ubuntu 用户）
POST /api/ansible/inventories/generate
{
  "name": "prod-ubuntu-nodes",
  "cluster_id": 1,
  "ssh_key_id": 1,  // ubuntu-key
  "node_labels": {
    "env": "production",
    "os": "ubuntu"
  }
}

# 开发环境（root 用户）
POST /api/ansible/inventories/generate
{
  "name": "dev-centos-nodes",
  "cluster_id": 1,
  "ssh_key_id": 2,  // centos-root-key
  "node_labels": {
    "env": "development",
    "os": "centos"
  }
}
```

### 3. 在 Playbook 中使用

生成的清单已经包含 `ansible_user`，在 Playbook 中可以直接使用：

```yaml
---
- name: 配置 K8s 节点
  hosts: all
  become: yes  # 如果需要提升权限
  tasks:
    - name: 更新系统包
      apt:
        update_cache: yes
        upgrade: dist
      when: ansible_user == "ubuntu"
    
    - name: 安装必要软件
      yum:
        name: 
          - vim
          - htop
        state: present
      when: ansible_user == "root"
```

## 日志输出

系统会记录用户名获取过程：

```
[INFO] Using SSH key username: ubuntu
[INFO] Successfully generated inventory from K8s cluster production: prod-nodes (ID: 1, 5 nodes, user: ubuntu)
```

如果 SSH 密钥未找到：
```
[WARN] SSH key 99 not found, using default user 'root'
[INFO] Successfully generated inventory from K8s cluster production: prod-nodes (ID: 1, 5 nodes, user: root)
```

## 注意事项

1. **SSH 密钥必须与目标节点匹配**
   - 确保指定的用户名在目标节点上存在
   - 确保 SSH 密钥已添加到目标节点的 authorized_keys

2. **默认使用 root 用户**
   - 如果不指定 SSH 密钥，默认使用 root 用户
   - 适用于传统的 Kubernetes 部署

3. **权限考虑**
   - 某些操作可能需要 sudo 权限
   - 在 Playbook 中使用 `become: yes` 提升权限

4. **多用户场景**
   - 为不同的操作系统/环境创建不同的清单
   - 每个清单关联相应的 SSH 密钥

## 验证配置

### 测试 SSH 连接

```bash
# 获取生成的清单
curl http://localhost:8080/api/ansible/inventories/1

# 手动测试 SSH 连接
ssh -i /path/to/private_key ubuntu@10.0.1.10

# 使用 Ansible 测试
ansible all -i inventory.ini -m ping
```

### 查看生成的清单内容

```bash
curl http://localhost:8080/api/ansible/inventories/1 | jq '.content'
```

输出示例：
```ini
[all]
node-1 ansible_host=10.0.1.10 ansible_user=ubuntu
node-2 ansible_host=10.0.1.11 ansible_user=ubuntu
node-3 ansible_host=10.0.1.12 ansible_user=ubuntu

[all:vars]
ansible_python_interpreter=/usr/bin/python3
ansible_ssh_common_args='-o StrictHostKeyChecking=no'
```

## 版本历史

- **2025-10-31**: 初始实现 - 支持从 SSH 密钥获取用户名
- 默认用户名：root
- 支持自定义用户名

## 相关文档

- [Ansible SSH 密钥管理](./ansible-ssh-key-database-storage.md)
- [Ansible 主机清单实现](./ansible-task-center-implementation.md)
- [Ansible 故障排查](./ansible-troubleshooting.md)

