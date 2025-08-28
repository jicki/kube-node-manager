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

- `JWT_SECRET`: JWT 密钥（来自 Secret）
- `LDAP_*`: LDAP 配置（来自 ConfigMap/Secret）
- `DATABASE_DSN`: 数据库路径
- `GIN_MODE`: 运行模式

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

```bash
# 检查 ServiceAccount
kubectl get sa kube-node-manager

# 检查 RBAC
kubectl get clusterrole kube-node-manager
kubectl get clusterrolebinding kube-node-manager
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