# Ansible 主机清单 SSH 用户名优化

## 优化内容

从集群生成主机清单时，`ansible_user` 参数现在会自动从关联的 SSH 密钥中获取用户名，而不是硬编码为 `root`。

## 问题背景

### 之前的实现

在优化前，从 Kubernetes 集群生成主机清单时，所有主机的 `ansible_user` 都硬编码为 `root`：

```ini
[all]
node-1 ansible_host=10.0.0.1 ansible_user=root
node-2 ansible_host=10.0.0.2 ansible_user=root
node-3 ansible_host=10.0.0.3 ansible_user=root
```

这导致的问题：
- 如果实际使用的不是 `root` 用户，需要手动修改清单
- SSH 密钥中已经配置了正确的用户名，但生成清单时被忽略
- 不符合最小权限原则，实际可能只需要普通用户权限

## 优化实现

### 1. 代码修改

#### `GenerateFromK8s` 函数

在生成主机清单之前，先获取 SSH 密钥的用户名：

```go
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
```

#### `generateINIInventory` 函数

更新函数签名，接收 `ansibleUser` 参数：

```go
func (s *InventoryService) generateINIInventory(nodes []k8s.NodeInfo, clusterName string, ansibleUser string) string {
    // ...
    // 使用传入的用户名而不是硬编码的 "root"
    builder.WriteString(fmt.Sprintf("%s ansible_host=%s ansible_user=%s\n", node.Name, ip, ansibleUser))
    // ...
}
```

#### `generateHostsData` 函数

在结构化主机数据中也包含 `ansible_user` 信息：

```go
func (s *InventoryService) generateHostsData(nodes []k8s.NodeInfo, ansibleUser string) model.HostsData {
    // ...
    host := map[string]interface{}{
        "name":         node.Name,
        "ip":           ip,
        "internal_ip":  node.InternalIP,
        "external_ip":  node.ExternalIP,
        "roles":        node.Roles,
        "labels":       node.Labels,
        "version":      node.Version,
        "os":           node.OS,
        "ansible_user": ansibleUser, // 添加 ansible_user 信息
    }
    // ...
}
```

## 使用示例

### 场景 1: 使用指定 SSH 密钥

假设有一个 SSH 密钥配置为使用 `ubuntu` 用户：

```bash
# SSH 密钥信息
Name: Ubuntu Key
Username: ubuntu
Type: private_key
```

从集群生成主机清单时指定该 SSH 密钥：

```bash
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prod-cluster-inventory",
    "cluster_id": 1,
    "ssh_key_id": 5
  }'
```

生成的清单内容：

```ini
[all]
node-1 ansible_host=10.0.0.1 ansible_user=ubuntu
node-2 ansible_host=10.0.0.2 ansible_user=ubuntu
node-3 ansible_host=10.0.0.3 ansible_user=ubuntu

[all:vars]
ansible_python_interpreter=/usr/bin/python3
ansible_ssh_common_args='-o StrictHostKeyChecking=no'
```

### 场景 2: 不指定 SSH 密钥（使用默认）

如果不指定 SSH 密钥 ID，默认使用 `root` 用户：

```bash
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-cluster-inventory",
    "cluster_id": 1
  }'
```

生成的清单内容：

```ini
[all]
node-1 ansible_host=10.0.0.1 ansible_user=root
node-2 ansible_host=10.0.0.2 ansible_user=root
node-3 ansible_host=10.0.0.3 ansible_user=root

[all:vars]
ansible_python_interpreter=/usr/bin/python3
ansible_ssh_common_args='-o StrictHostKeyChecking=no'
```

### 场景 3: 不同环境使用不同用户

为不同的集群环境配置不同的 SSH 密钥：

```bash
# 生产环境 - 使用 ubuntu 用户
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -d '{"name": "prod-inv", "cluster_id": 1, "ssh_key_id": 1}'

# 测试环境 - 使用 admin 用户
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -d '{"name": "test-inv", "cluster_id": 2, "ssh_key_id": 2}'

# 开发环境 - 使用 root 用户
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -d '{"name": "dev-inv", "cluster_id": 3, "ssh_key_id": 3}'
```

## 前端界面调整建议

为了让用户更好地使用这个功能，建议在前端做以下调整：

### 1. 生成清单页面

在"从集群生成主机清单"表单中，SSH 密钥选择框应该：

```vue
<template>
  <el-form-item label="SSH 密钥" prop="ssh_key_id">
    <el-select 
      v-model="form.ssh_key_id" 
      placeholder="选择 SSH 密钥（留空使用 root）"
      clearable
    >
      <el-option
        v-for="key in sshKeys"
        :key="key.id"
        :label="`${key.name} (${key.username}@${key.port})`"
        :value="key.id"
      >
        <span>{{ key.name }}</span>
        <span style="float: right; color: #8492a6; font-size: 13px">
          用户: {{ key.username }}
        </span>
      </el-option>
    </el-select>
    <div class="el-form-item__description">
      选择 SSH 密钥后，将使用密钥配置的用户名作为 ansible_user
    </div>
  </el-form-item>
</template>
```

### 2. 清单详情页面

显示清单的 SSH 密钥和用户信息：

```vue
<el-descriptions-item label="SSH 密钥">
  <el-tag v-if="inventory.ssh_key">
    {{ inventory.ssh_key.name }}
  </el-tag>
  <span v-else>未配置</span>
</el-descriptions-item>

<el-descriptions-item label="Ansible 用户">
  <el-tag type="info">
    {{ inventory.ssh_key ? inventory.ssh_key.username : 'root' }}
  </el-tag>
</el-descriptions-item>
```

### 3. 清单预览

生成清单时实时预览 `ansible_user` 值：

```vue
<el-card header="清单预览" v-if="form.ssh_key_id">
  <pre>{{ previewInventory }}</pre>
</el-card>

<script>
computed: {
  previewInventory() {
    const user = this.selectedSshKey?.username || 'root';
    return `[all]
node-1 ansible_host=x.x.x.x ansible_user=${user}
node-2 ansible_host=x.x.x.x ansible_user=${user}
...`;
  }
}
</script>
```

## 优势

### 1. 自动化配置

✅ 用户无需手动修改生成的清单  
✅ SSH 密钥的用户名配置一次，自动应用到清单  
✅ 减少配置错误和不一致

### 2. 安全性提升

✅ 支持使用非 root 用户，符合最小权限原则  
✅ 不同环境可以使用不同的用户权限  
✅ 集中管理 SSH 凭据和用户配置

### 3. 灵活性

✅ 保持向后兼容，未指定密钥时默认使用 root  
✅ 支持多种用户名：root, ubuntu, admin, centos 等  
✅ 可以为不同集群配置不同的访问用户

## 日志示例

优化后，日志会显示使用的用户名：

```
[INFO] Using SSH key username: ubuntu
[INFO] Successfully generated inventory from K8s cluster prod-cluster: prod-inv (ID: 10, 5 nodes, user: ubuntu)
```

如果未指定 SSH 密钥：

```
[INFO] Successfully generated inventory from K8s cluster test-cluster: test-inv (ID: 11, 3 nodes, user: root)
```

如果 SSH 密钥不存在：

```
[WARN] SSH key 999 not found, using default user 'root'
[INFO] Successfully generated inventory from K8s cluster dev-cluster: dev-inv (ID: 12, 2 nodes, user: root)
```

## 测试验证

### 1. 单元测试

```go
func TestGenerateFromK8s_WithSSHKey(t *testing.T) {
    // 创建测试 SSH 密钥
    sshKey := &model.AnsibleSSHKey{
        Name:     "test-key",
        Username: "ubuntu",
        Type:     model.SSHKeyTypePrivateKey,
    }
    db.Create(sshKey)
    
    // 生成清单
    req := model.GenerateInventoryRequest{
        Name:      "test-inventory",
        ClusterID: 1,
        SSHKeyID:  &sshKey.ID,
    }
    
    inventory, err := inventorySvc.GenerateFromK8s(req, 1)
    
    // 验证 ansible_user 为 ubuntu
    assert.NoError(t, err)
    assert.Contains(t, inventory.Content, "ansible_user=ubuntu")
}

func TestGenerateFromK8s_WithoutSSHKey(t *testing.T) {
    // 生成清单（不指定 SSH 密钥）
    req := model.GenerateInventoryRequest{
        Name:      "test-inventory",
        ClusterID: 1,
    }
    
    inventory, err := inventorySvc.GenerateFromK8s(req, 1)
    
    // 验证 ansible_user 为 root（默认值）
    assert.NoError(t, err)
    assert.Contains(t, inventory.Content, "ansible_user=root")
}
```

### 2. 集成测试

```bash
# 1. 创建 SSH 密钥
curl -X POST http://localhost:8080/api/ansible/ssh-keys \
  -d '{"name":"ubuntu-key","username":"ubuntu","type":"private_key","private_key":"..."}'

# 2. 从集群生成清单（指定 SSH 密钥）
curl -X POST http://localhost:8080/api/ansible/inventories/generate \
  -d '{"name":"prod-inv","cluster_id":1,"ssh_key_id":1}'

# 3. 获取清单内容并验证
curl http://localhost:8080/api/ansible/inventories/1

# 验证清单内容包含 ansible_user=ubuntu
```

### 3. 功能测试清单

- [ ] 指定 SSH 密钥时，使用密钥的用户名
- [ ] 不指定 SSH 密钥时，默认使用 root
- [ ] SSH 密钥不存在时，回退到 root 并记录警告
- [ ] 不同 SSH 密钥生成不同的用户名
- [ ] 结构化主机数据也包含 ansible_user 字段
- [ ] 日志正确记录使用的用户名

## 兼容性说明

### 向后兼容

✅ 已有清单不受影响，保持原有配置  
✅ 未指定 SSH 密钥时行为不变（使用 root）  
✅ 数据库表结构无需修改  
✅ API 接口无需修改（ssh_key_id 本来就是可选字段）

### 数据迁移

不需要数据迁移，优化仅影响新生成的清单。

## 相关文件

- `backend/internal/service/ansible/inventory.go` - 主要修改文件
- `backend/internal/model/ansible.go` - 数据模型定义
- `frontend/src/views/ansible/InventoryList.vue` - 前端界面（建议优化）

## 版本历史

- **2025-10-31**: 初始版本 - 从 SSH 密钥获取 ansible_user

