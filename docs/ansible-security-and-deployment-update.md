# Ansible 任务中心安全和部署更新

## 更新日期

2025-10-30

## 更新摘要

本次更新主要包括两个方面的改进：
1. **权限管理强化** - 限制 Ansible 模块只允许 admin 角色访问
2. **Docker 镜像增强** - 在容器中集成 Ansible 命令行工具

## 1. 权限管理强化

### 变更说明

为了确保系统安全，所有 Ansible 模块的 API 接口现在都需要管理员权限才能访问。

### 实现细节

#### 1.1 权限检查函数

在 `backend/internal/handler/ansible/handler.go` 中新增了权限检查辅助函数：

```go
// checkAdminPermission 检查管理员权限
func checkAdminPermission(c *gin.Context) bool {
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can access Ansible module"})
		return false
	}
	return true
}
```

#### 1.2 受保护的接口

所有 Ansible 相关的 API 接口都添加了权限检查：

**任务管理（handler.go）:**
- ✅ `GET /api/v1/ansible/tasks` - 列出任务
- ✅ `GET /api/v1/ansible/tasks/:id` - 获取任务详情
- ✅ `POST /api/v1/ansible/tasks` - 创建并执行任务
- ✅ `POST /api/v1/ansible/tasks/:id/cancel` - 取消任务
- ✅ `POST /api/v1/ansible/tasks/:id/retry` - 重试任务
- ✅ `GET /api/v1/ansible/tasks/:id/logs` - 获取任务日志
- ✅ `POST /api/v1/ansible/tasks/:id/refresh` - 刷新任务状态
- ✅ `GET /api/v1/ansible/statistics` - 获取统计信息

**模板管理（template.go）:**
- ✅ `GET /api/v1/ansible/templates` - 列出模板
- ✅ `GET /api/v1/ansible/templates/:id` - 获取模板详情
- ✅ `POST /api/v1/ansible/templates` - 创建模板
- ✅ `PUT /api/v1/ansible/templates/:id` - 更新模板
- ✅ `DELETE /api/v1/ansible/templates/:id` - 删除模板
- ✅ `POST /api/v1/ansible/templates/validate` - 验证模板

**主机清单管理（inventory.go）:**
- ✅ `GET /api/v1/ansible/inventories` - 列出清单
- ✅ `GET /api/v1/ansible/inventories/:id` - 获取清单详情
- ✅ `POST /api/v1/ansible/inventories` - 创建清单
- ✅ `PUT /api/v1/ansible/inventories/:id` - 更新清单
- ✅ `DELETE /api/v1/ansible/inventories/:id` - 删除清单
- ✅ `POST /api/v1/ansible/inventories/generate` - 从集群生成清单
- ✅ `POST /api/v1/ansible/inventories/:id/refresh` - 刷新清单

### 权限验证流程

```
请求 → AuthMiddleware → 提取 user_role → checkAdminPermission
                                              ↓
                                        role == "admin"?
                                        ↙            ↘
                                      是             否
                                      ↓              ↓
                                  继续处理         403 Forbidden
```

### 错误响应

当非管理员用户尝试访问 Ansible 模块时，会收到以下响应：

```json
{
  "error": "Only administrators can access Ansible module"
}
```

HTTP 状态码：`403 Forbidden`

## 2. Docker 镜像 Ansible 集成

### 变更说明

在 Docker 镜像中集成 Ansible 命令行工具，使得容器可以直接执行 Ansible Playbook，无需额外的 Ansible 节点。

### Dockerfile 更新

#### 2.1 安装的软件包

```dockerfile
RUN apk --no-cache add \
    ansible \          # Ansible 核心工具
    python3 \          # Python3 运行时（Ansible 依赖）
    py3-pip \          # Python 包管理器
    openssh-client \   # SSH 客户端（连接目标主机）
    sshpass \          # SSH 密码认证工具
    ca-certificates \  # SSL 证书
    tzdata             # 时区数据
```

#### 2.2 Ansible 配置

自动创建 Ansible 配置文件 `/etc/ansible/ansible.cfg`，包含以下默认设置：

```ini
[defaults]
host_key_checking = False    # 禁用主机密钥检查（适用于动态环境）
timeout = 30                 # 连接超时时间（秒）
gather_timeout = 30          # Facts 收集超时时间（秒）
```

### 镜像大小影响

添加 Ansible 后，镜像大小预计增加约 **50-80 MB**：

- ansible: ~30 MB
- python3: ~20 MB
- 其他依赖: ~20-30 MB

### 使用说明

#### 2.3.1 SSH 密钥配置

为了让容器能够连接到目标主机，需要挂载 SSH 密钥：

**Docker Compose 示例:**

```yaml
services:
  kube-node-manager:
    image: your-registry/kube-node-manager:latest
    volumes:
      - ./ssh-keys:/root/.ssh:ro   # 挂载 SSH 密钥（只读）
      - ./data:/app/data            # 数据持久化
    environment:
      - ANSIBLE_HOST_KEY_CHECKING=False
```

**Kubernetes 示例:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ansible-ssh-key
type: Opaque
data:
  id_rsa: <base64-encoded-private-key>
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-node-manager
spec:
  containers:
  - name: app
    image: your-registry/kube-node-manager:latest
    volumeMounts:
    - name: ssh-key
      mountPath: /root/.ssh/id_rsa
      subPath: id_rsa
      readOnly: true
  volumes:
  - name: ssh-key
    secret:
      secretName: ansible-ssh-key
      defaultMode: 0600
```

#### 2.3.2 SSH 密钥权限

确保 SSH 密钥文件权限正确：

```bash
chmod 600 /path/to/id_rsa
chmod 644 /path/to/id_rsa.pub
chmod 700 /path/to/ssh-keys-directory
```

#### 2.3.3 测试 Ansible 安装

在容器中验证 Ansible 安装：

```bash
# 进入容器
docker exec -it <container-id> sh

# 检查 Ansible 版本
ansible --version

# 测试 Ansible 连接
ansible all -i "target-host," -m ping -u root --private-key=/root/.ssh/id_rsa
```

## 3. 安全建议

### 3.1 SSH 密钥管理

1. **使用专用密钥**
   - 为 Ansible 创建专用的 SSH 密钥对
   - 不要复用个人或其他系统的密钥

2. **最小权限原则**
   - 目标主机上的 Ansible 用户应仅具有必要的权限
   - 考虑使用 `sudo` 进行权限提升而非直接使用 root

3. **密钥轮换**
   - 定期轮换 SSH 密钥
   - 记录密钥使用历史

### 3.2 网络隔离

1. **限制访问**
   - 使用防火墙规则限制哪些主机可以被 Ansible 访问
   - 考虑使用 VPN 或专用网络

2. **审计日志**
   - 启用目标主机的 SSH 登录日志
   - 记录所有 Ansible 操作

### 3.3 Playbook 安全

1. **代码审查**
   - 所有 Playbook 模板都经过审查后才能使用
   - 实施危险命令检测（已在代码中实现）

2. **版本控制**
   - 将 Playbook 模板存储在版本控制系统中
   - 追踪所有变更历史

## 4. 升级步骤

### 4.1 备份数据

```bash
# 备份数据库
docker exec <container-id> sqlite3 /app/data/kube-node-manager.db ".backup '/app/data/backup.db'"

# 复制备份文件到主机
docker cp <container-id>:/app/data/backup.db ./backup-$(date +%Y%m%d).db
```

### 4.2 构建新镜像

```bash
# 构建镜像
docker build -t your-registry/kube-node-manager:latest .

# 推送到镜像仓库
docker push your-registry/kube-node-manager:latest
```

### 4.3 更新部署

**Docker Compose:**

```bash
# 停止旧容器
docker-compose down

# 拉取新镜像
docker-compose pull

# 启动新容器
docker-compose up -d
```

**Kubernetes:**

```bash
# 更新部署
kubectl set image deployment/kube-node-manager \
  kube-node-manager=your-registry/kube-node-manager:latest

# 查看更新状态
kubectl rollout status deployment/kube-node-manager
```

### 4.4 验证部署

```bash
# 检查容器日志
docker logs <container-id>

# 或 Kubernetes
kubectl logs -f deployment/kube-node-manager

# 验证 Ansible 可用性
docker exec <container-id> ansible --version

# 验证权限管理
curl -H "Authorization: Bearer <non-admin-token>" \
  http://localhost:8080/api/v1/ansible/tasks
# 预期：403 Forbidden
```

## 5. 故障排查

### 5.1 Ansible 命令未找到

**问题：** 容器中找不到 `ansible` 命令

**解决方案：**
```bash
# 检查 Ansible 是否安装
docker exec <container-id> which ansible

# 如果未安装，重新构建镜像
docker build --no-cache -t your-registry/kube-node-manager:latest .
```

### 5.2 SSH 连接失败

**问题：** Ansible 无法连接到目标主机

**排查步骤：**
```bash
# 1. 检查 SSH 密钥是否挂载
docker exec <container-id> ls -la /root/.ssh/

# 2. 检查密钥权限
docker exec <container-id> stat -c "%a %n" /root/.ssh/id_rsa

# 3. 手动测试 SSH 连接
docker exec <container-id> ssh -i /root/.ssh/id_rsa -o StrictHostKeyChecking=no root@target-host "echo OK"

# 4. 检查网络连通性
docker exec <container-id> ping -c 3 target-host
```

### 5.3 权限被拒绝

**问题：** 普通用户无法访问 Ansible 模块（按预期工作）

**确认：**
```bash
# 检查用户角色
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/auth/profile

# 升级用户为管理员（需要现有管理员操作）
curl -X PUT \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}' \
  http://localhost:8080/api/v1/users/<user-id>
```

## 6. 性能影响

### 6.1 镜像构建时间

- 增加约 **2-3 分钟**（首次构建）
- 后续构建可利用 Docker 缓存

### 6.2 启动时间

- Ansible 安装不影响应用启动时间
- 容器启动时间增加约 **1-2 秒**（包加载）

### 6.3 运行时内存

- Ansible idle 状态：~10 MB
- Ansible 执行任务时：~50-100 MB（取决于 playbook 复杂度）

## 7. 后续改进建议

1. **Ansible Galaxy 集成**
   - 支持从 Ansible Galaxy 安装 roles 和 collections
   - 提供常用 roles 的模板

2. **任务队列优化**
   - 实现更复杂的任务调度
   - 支持任务优先级

3. **多 Ansible 节点**
   - 分离 Ansible 执行节点
   - 支持横向扩展

4. **增强的审计**
   - 记录所有 Playbook 执行历史
   - 提供详细的审计报告

## 8. 相关文档

- [Ansible 任务中心实施总结](./ansible-task-center-implementation.md)
- [Kubernetes 部署指南](../deploy/k8s/README.md)
- [Docker 部署指南](../deploy/docker/README.md)

## 变更记录

| 日期 | 版本 | 变更内容 | 作者 |
|------|------|----------|------|
| 2025-10-30 | 1.0.0 | 初始版本：权限管理和 Docker Ansible 集成 | System |

