# Kubernetes Deployment

本目录包含 kube-node-manager 的 Kubernetes 部署配置。

## 文件说明

- `k8s-statefulset.yaml`: StatefulSet、ServiceAccount、RBAC 和 ConfigMap/Secret 配置
- `k8s-service.yaml`: Service 配置（ClusterIP 和 Headless）  
- `k8s-ingress.yaml`: Ingress 配置，支持 HTTPS
- `kustomization.yaml`: Kustomize 配置文件

## 部署步骤

### 1. 准备工作

确保已安装以下组件：
- Kubernetes 集群
- NGINX Ingress Controller
- 证书管理器（如 cert-manager，可选）

### 2. 修改配置

根据实际环境修改以下配置：

**修改 Ingress 域名:**
```yaml
# k8s-ingress.yaml
spec:
  tls:
  - hosts:
    - your-domain.com  # 修改为实际域名
  rules:
  - host: your-domain.com  # 修改为实际域名
```

**修改 Secret 配置:**
```bash
# 生成新的 JWT Secret（base64 编码）
echo -n "your-new-jwt-secret" | base64

# 更新 k8s-statefulset.yaml 中的 Secret 数据
```

### 3. 部署应用

```bash
# 使用 kubectl 直接部署
kubectl apply -f .

# 或使用 kustomize 部署
kubectl apply -k .
```

### 4. 验证部署

```bash
# 检查 Pod 状态
kubectl get pods -l app=kube-node-manager

# 检查 Service
kubectl get svc -l app=kube-node-manager

# 检查 Ingress
kubectl get ingress kube-node-manager

# 查看日志
kubectl logs -l app=kube-node-manager -f
```

## 配置说明

### 环境变量

应用通过以下环境变量进行配置：

**基础配置：**
- `JWT_SECRET`: JWT 密钥（来自 Secret）
- `LDAP_*`: LDAP 配置（来自 ConfigMap/Secret）
- `GIN_MODE`: 运行模式

**数据库配置（多副本环境）：**
- `DB_HOST`: PostgreSQL 主机地址（如：`postgres-service.default.svc.cluster.local`）
- `DB_PORT`: PostgreSQL 端口（默认：5432）
- `DB_USERNAME`: 数据库用户名
- `DB_PASSWORD`: 数据库密码（来自 Secret）
- `DB_DATABASE`: 数据库名称
- `DB_SSL_MODE`: SSL 模式（disable/require/verify-ca/verify-full）

**⚠️  多副本部署重要提示：**
1. 必须使用 PostgreSQL 数据库（SQLite 不支持多副本）
2. 确保所有环境变量正确设置，特别是 `DB_HOST`
3. PostgreSQL Listener 会使用这些环境变量建立独立连接
4. 建议配置文件中也设置相同的数据库参数以保持一致性

### 持久化存储

- 数据存储在 `/app/data` 目录
- 使用 PVC 提供持久化存储
- 默认存储大小：1Gi

### 权限配置

应用需要以下 Kubernetes 权限：
- 节点的读取、列出、监听、修改权限
- 节点状态的修改权限
- Metrics API 访问权限

## 故障排除

### Pod 启动失败

```bash
# 检查 Pod 事件
kubectl describe pod <pod-name>

# 检查日志
kubectl logs <pod-name>
```

### 权限问题

如果遇到节点封锁功能失败，提示权限错误：
```
pods is forbidden: User "system:serviceaccount:kube-node-mgr:kube-node-mgr" cannot list resource "pods" in API group "" at the cluster scope
```

**快速修复方法：**

1. **使用自动修复脚本（推荐）：**
```bash
# 运行权限修复脚本
../scripts/fix-rbac-permissions.sh

# 或指定命名空间
../scripts/fix-rbac-permissions.sh your-namespace
```

2. **手动修复权限：**
```bash
# 应用RBAC权限补丁
kubectl apply -f rbac-patch.yaml

# 重启应用以应用新权限
kubectl rollout restart statefulset/kube-node-mgr -n kube-node-mgr
```

**检查权限配置：**
```bash
# 检查 ServiceAccount
kubectl get sa kube-node-mgr -n kube-node-mgr

# 检查 RBAC
kubectl get clusterrole kube-node-mgr
kubectl get clusterrolebinding kube-node-mgr

# 查看详细权限配置
kubectl describe clusterrole kube-node-mgr
```

### 网络问题

```bash
# 检查 Service
kubectl get svc kube-node-manager -o yaml

# 检查 Ingress
kubectl get ingress kube-node-manager -o yaml

# 检查 Ingress Controller 日志
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller
```