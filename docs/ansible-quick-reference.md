# Ansible 任务中心快速参考

## 🚀 快速开始

### 1. 权限要求
```
✅ 必须使用 admin 角色的用户账号
❌ user 和 viewer 角色无法访问 Ansible 模块
```

### 2. 构建 Docker 镜像
```bash
docker build -t your-registry/kube-node-manager:latest .
```

### 3. 运行容器（Docker Compose）
```yaml
version: '3.8'
services:
  kube-node-manager:
    image: your-registry/kube-node-manager:latest
    volumes:
      - ~/.ssh:/root/.ssh:ro  # SSH 密钥（只读）
      - ./data:/app/data       # 数据持久化
    ports:
      - "8080:8080"
    environment:
      - GIN_MODE=release
      - DATABASE_DSN=./data/kube-node-manager.db
```

### 4. 访问模块
```bash
# 获取 admin 用户 token
TOKEN=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}' | jq -r '.token')

# 列出任务
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/ansible/tasks
```

## 📋 常用操作

### 创建模板
```bash
curl -X POST http://localhost:8080/api/v1/ansible/templates \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "系统更新",
    "description": "更新系统软件包",
    "playbook_content": "---\n- hosts: all\n  tasks:\n    - name: 更新包\n      yum:\n        name: '*'\n        state: latest",
    "tags": ["system", "update"]
  }'
```

### 从 K8s 生成清单
```bash
curl -X POST http://localhost:8080/api/v1/ansible/inventories/generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": 1,
    "name": "生产环境节点",
    "label_selector": "node-role.kubernetes.io/worker=true"
  }'
```

### 创建并执行任务
```bash
curl -X POST http://localhost:8080/api/v1/ansible/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "系统更新任务",
    "template_id": 1,
    "inventory_id": 1,
    "cluster_id": 1
  }'
```

## 🔒 权限错误处理

### 错误响应
```json
{
  "error": "Only administrators can access Ansible module"
}
```
**HTTP 状态码：** 403 Forbidden

### 解决方法
1. 确认当前用户角色：
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/auth/profile
```

2. 升级为管理员（需要现有 admin 操作）：
```bash
curl -X PUT http://localhost:8080/api/v1/users/<user-id> \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}'
```

## 🐳 Docker 命令

### 验证 Ansible 安装
```bash
# 检查版本
docker exec <container-id> ansible --version

# 测试连接
docker exec <container-id> ansible all -i "host," -m ping -u root
```

### 查看日志
```bash
# Docker
docker logs -f <container-id>

# Kubernetes
kubectl logs -f deployment/kube-node-manager
```

### SSH 密钥管理
```bash
# 生成密钥
ssh-keygen -t rsa -b 4096 -f ~/.ssh/ansible_id_rsa

# 复制到目标主机
ssh-copy-id -i ~/.ssh/ansible_id_rsa.pub root@target-host

# 验证密钥权限
ls -la ~/.ssh/ansible_id_rsa  # 应该是 -rw------- (600)
```

## 📊 监控任务

### 查看任务状态
```bash
# 列出所有任务
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/ansible/tasks?page=1&page_size=10"

# 查看特定任务
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/ansible/tasks/1"

# 获取任务日志
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/ansible/tasks/1/logs"
```

### WebSocket 实时日志
```javascript
const ws = new WebSocket(`ws://localhost:8080/api/v1/ansible/tasks/${taskId}/ws`);

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`[${data.log_type}] ${data.content}`);
};
```

## 🛠️ 故障排查

### 1. Ansible 命令未找到
```bash
# 检查 Ansible
docker exec <container-id> which ansible

# 重新构建镜像
docker build --no-cache -t your-registry/kube-node-manager:latest .
```

### 2. SSH 连接失败
```bash
# 检查密钥挂载
docker exec <container-id> ls -la /root/.ssh/

# 手动测试 SSH
docker exec <container-id> ssh -i /root/.ssh/id_rsa root@target-host "echo OK"
```

### 3. 任务执行失败
```bash
# 查看详细日志
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/ansible/tasks/<task-id>/logs?limit=1000"

# 检查 Playbook 语法
curl -X POST http://localhost:8080/api/v1/ansible/templates/validate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"playbook_content": "..."}'
```

## 📚 相关文档

- [完整实施文档](./ansible-task-center-implementation.md)
- [安全和部署更新](./ansible-security-and-deployment-update.md)
- [部署指南](../deploy/README.md)

## 💡 最佳实践

1. **模板管理**
   - 为常用操作创建模板
   - 使用标签组织模板
   - 定期审查和更新模板

2. **主机清单**
   - K8s 集群使用动态清单
   - 手动主机使用静态清单
   - 定期刷新 K8s 清单

3. **任务执行**
   - 先在测试环境验证 Playbook
   - 使用 `--check` 模式进行干运行
   - 监控任务日志

4. **安全**
   - 仅授予必要的管理员权限
   - 定期轮换 SSH 密钥
   - 审查所有 Playbook 更改

## 🆘 获取帮助

遇到问题？
1. 查看[故障排查指南](./ansible-security-and-deployment-update.md#5-故障排查)
2. 检查容器日志
3. 联系开发团队

