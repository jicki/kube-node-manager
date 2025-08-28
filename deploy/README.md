# 部署文档

本目录包含 kube-node-manager 的部署配置和脚本。

## 目录结构

```
deploy/
├── docker/              # Docker 部署配置
│   ├── docker-compose.yml       # 生产环境配置
│   ├── docker-compose.dev.yml   # 开发环境配置
│   ├── docker-compose.prod.yml  # 生产环境配置
│   ├── Dockerfile               # 主 Dockerfile
│   └── nginx/                   # Nginx 配置
├── k8s/                 # Kubernetes 部署配置
│   ├── k8s-statefulset.yaml    # StatefulSet 和 RBAC
│   ├── k8s-service.yaml         # Service 配置
│   ├── k8s-ingress.yaml         # Ingress 配置
│   ├── kustomization.yaml       # Kustomize 配置
│   └── README.md                # K8s 部署文档
├── scripts/             # 部署脚本
│   ├── install.sh               # Docker 安装脚本
│   ├── backup.sh                # 数据备份脚本
│   ├── k8s-deploy.sh           # K8s 部署脚本
│   └── k8s-cleanup.sh          # K8s 清理脚本
└── README.md           # 本文档
```

## 部署方式

### 1. Docker 部署

适用于单机部署或开发环境：

```bash
# 开发环境
make dev

# 生产环境
make deploy

# 使用脚本安装
./deploy/scripts/install.sh
```

#### Docker 部署特点

- 单一镜像多阶段构建
- 前后端集成部署
- 包含 SQLite 数据库
- 支持数据持久化
- 内置健康检查

### 2. Kubernetes 部署

适用于集群环境或生产环境：

```bash
# 快速部署
make k8s-deploy

# 或使用脚本部署
NAMESPACE=kube-system DOMAIN=your-domain.com ./deploy/scripts/k8s-deploy.sh

# 查看状态
make k8s-status

# 查看日志
make k8s-logs
```

#### Kubernetes 部署特点

- StatefulSet 确保数据持久化
- 完整的 RBAC 权限配置
- ConfigMap/Secret 管理配置
- Ingress 支持 HTTPS 访问
- 支持水平扩展

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `PORT` | 服务端口 | `8080` |
| `GIN_MODE` | 运行模式 | `release` |
| `DATABASE_DSN` | 数据库连接 | `./data/kube-node-manager.db` |
| `JWT_SECRET` | JWT 密钥 | - |
| `JWT_EXPIRE_TIME` | JWT 过期时间 | `86400` |
| `LDAP_ENABLED` | 启用 LDAP | `false` |
| `LDAP_*` | LDAP 配置 | - |

### 数据持久化

- **Docker**: 使用 volume 挂载 `./data` 目录
- **Kubernetes**: 使用 PVC 提供持久化存储

### 网络配置

- **Docker**: 通过 `8080` 端口访问
- **Kubernetes**: 通过 Ingress 访问，支持自定义域名

## 安全配置

### 1. JWT 密钥

生产环境必须更改默认的 JWT 密钥：

```bash
# 生成随机密钥
openssl rand -base64 32
```

### 2. LDAP 配置

如需集成 LDAP 认证：

```yaml
LDAP_ENABLED: "true"
LDAP_HOST: "ldap.example.com"
LDAP_PORT: "389"
LDAP_BASE_DN: "dc=example,dc=com"
LDAP_USER_FILTER: "(uid=%s)"
LDAP_ADMIN_DN: "cn=admin,dc=example,dc=com"
LDAP_ADMIN_PASS: "admin_password"
```

### 3. HTTPS 配置

Kubernetes 部署支持 HTTPS，需要配置证书：

```bash
# 创建 TLS Secret
kubectl create secret tls kube-node-manager-tls \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key \
  -n default
```

## 监控和维护

### 健康检查

应用提供健康检查接口：
- **端点**: `/api/v1/health`
- **方法**: `GET`
- **响应**: JSON 状态信息

### 日志管理

```bash
# Docker 环境
make logs

# Kubernetes 环境
make k8s-logs
```

### 数据备份

```bash
# 备份数据
make backup

# 恢复数据
make restore BACKUP=backup_filename
```

### 更新部署

```bash
# Docker 更新
make deploy-update

# Kubernetes 更新
make k8s-restart
```

## 故障排除

### 常见问题

1. **端口冲突**
   - 检查 8080 端口是否被占用
   - 修改 docker-compose.yml 中的端口映射

2. **数据库连接失败**
   - 检查数据目录权限
   - 确认 SQLite 文件路径正确

3. **Kubernetes 权限问题**
   - 检查 ServiceAccount 配置
   - 验证 RBAC 权限设置

4. **Ingress 访问失败**
   - 检查 Ingress Controller 状态
   - 验证域名 DNS 解析
   - 检查证书配置

### 诊断命令

```bash
# Docker 环境诊断
docker-compose ps
docker-compose logs app

# Kubernetes 环境诊断
kubectl get pods,svc,ingress -l app=kube-node-manager
kubectl describe pod <pod-name>
kubectl logs <pod-name> -f
```

## 性能优化

### 资源配置

根据实际使用情况调整资源限制：

```yaml
# Kubernetes 资源配置
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 扩展建议

- 生产环境建议使用外部数据库（PostgreSQL/MySQL）
- 启用日志聚合和监控
- 配置备份策略
- 使用 CDN 加速静态资源

## 支持

如遇到问题，请：
1. 查看相关日志
2. 检查配置文件
3. 参考项目文档
4. 提交 Issue