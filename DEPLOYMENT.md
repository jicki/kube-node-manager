# 部署指南

本文档详细介绍如何部署 Kubernetes 节点管理器到不同环境。

## 📋 部署清单

### 系统要求
- **操作系统**: Linux/macOS/Windows
- **Docker**: 20.0 或更高版本
- **Docker Compose**: 2.0 或更高版本
- **内存**: 最少 2GB，推荐 4GB
- **磁盘**: 最少 10GB 可用空间
- **网络**: 能够访问 Kubernetes 集群

### 端口要求
- **3000**: 前端Web界面
- **8080**: 后端API服务
- **443**: HTTPS（可选）

## 🚀 快速部署

### 方式一：使用安装脚本（推荐）

```bash
# 下载项目
git clone <repository-url>
cd kube-node-manager

# 运行安装脚本
./scripts/install.sh
```

### 方式二：手动部署

```bash
# 1. 创建环境配置
cp .env.example .env
# 编辑 .env 文件

# 2. 创建必要目录
mkdir -p data logs

# 3. 启动服务
docker-compose up -d

# 4. 检查服务状态
docker-compose ps
```

## 🏗️ 生产环境部署

### 1. 环境配置优化

```bash
# .env 配置示例
PORT=8080
GIN_MODE=release
JWT_SECRET=your-super-secure-jwt-secret-here
JWT_EXPIRE_TIME=86400
DATABASE_DSN=./data/kube-node-manager.db

# LDAP配置（可选）
LDAP_ENABLED=true
LDAP_HOST=ldap.company.com
LDAP_PORT=636
LDAP_BASE_DN=dc=company,dc=com
LDAP_USER_FILTER=(uid=%s)
```

### 2. 安全配置

#### SSL/TLS 配置
```yaml
# docker-compose.prod.yml
version: '3.8'
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - frontend
```

#### Nginx 配置示例
```nginx
server {
    listen 443 ssl http2;
    server_name your-domain.com;
    
    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;
    
    location / {
        proxy_pass http://frontend:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 3. 数据持久化

```yaml
# docker-compose.prod.yml
volumes:
  app_data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /opt/kube-node-manager/data
```

### 4. 监控和日志

```yaml
# docker-compose.monitoring.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
  
  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

## 🔒 安全最佳实践

### 1. 认证安全
- 使用强JWT密钥
- 设置合理的Token过期时间
- 启用HTTPS加密传输

### 2. 网络安全
```yaml
# 网络隔离配置
networks:
  frontend:
    driver: bridge
    internal: false
  backend:
    driver: bridge
    internal: true
```

### 3. 容器安全
```dockerfile
# 使用非root用户
RUN addgroup -g 1001 appgroup && \
    adduser -u 1001 -G appgroup -s /bin/sh -D appuser
USER appuser
```

### 4. 文件权限
```bash
# 设置正确的文件权限
chmod 600 .env
chmod 700 data/
chmod 644 configs/*.yaml
```

## 🌐 负载均衡部署

### HAProxy 配置
```
# /etc/haproxy/haproxy.cfg
global
    daemon

defaults
    mode http
    timeout connect 5000ms
    timeout client 50000ms
    timeout server 50000ms

frontend web_frontend
    bind *:80
    default_backend web_servers

backend web_servers
    balance roundrobin
    server web1 node1:3000 check
    server web2 node2:3000 check
    server web3 node3:3000 check
```

### Kubernetes 部署
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-node-manager
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kube-node-manager
  template:
    metadata:
      labels:
        app: kube-node-manager
    spec:
      containers:
      - name: backend
        image: kube-node-manager/backend:latest
        ports:
        - containerPort: 8080
      - name: frontend
        image: kube-node-manager/frontend:latest
        ports:
        - containerPort: 80
```

## 📊 监控配置

### Prometheus 监控
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'kube-node-manager'
    static_configs:
      - targets: ['backend:8080']
    metrics_path: '/metrics'
```

### 健康检查
```bash
# 检查服务健康状态
curl -f http://localhost:8080/api/v1/health
curl -f http://localhost:3000/health
```

## 🔄 升级和维护

### 1. 应用升级
```bash
# 停止服务
docker-compose down

# 备份数据
./scripts/backup.sh

# 拉取新版本
git pull

# 重新构建并启动
docker-compose up --build -d
```

### 2. 数据库维护
```bash
# 数据库备份
sqlite3 data/kube-node-manager.db ".backup backup.db"

# 数据库优化
sqlite3 data/kube-node-manager.db "VACUUM;"
```

### 3. 日志轮转
```bash
# logrotate 配置
/opt/kube-node-manager/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    copytruncate
}
```

## 🐛 故障排除

### 1. 服务无法启动
```bash
# 检查端口占用
netstat -tlnp | grep :3000
netstat -tlnp | grep :8080

# 查看服务日志
docker-compose logs backend
docker-compose logs frontend
```

### 2. 数据库问题
```bash
# 检查数据库文件
ls -la data/
file data/kube-node-manager.db

# 修复数据库
sqlite3 data/kube-node-manager.db "PRAGMA integrity_check;"
```

### 3. Kubernetes 连接问题
```bash
# 验证 kubeconfig
kubectl cluster-info
kubectl get nodes

# 检查权限
kubectl auth can-i get nodes
kubectl auth can-i patch nodes
```

## 📈 性能优化

### 1. 数据库优化
```sql
-- 创建索引
CREATE INDEX IF NOT EXISTS idx_audit_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);
```

### 2. 缓存配置
```yaml
# Redis 缓存（可选）
services:
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
```

### 3. 资源限制
```yaml
# docker-compose.yml
services:
  backend:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
```

## 🔐 备份和恢复

### 自动备份脚本
```bash
#!/bin/bash
# /etc/cron.daily/kube-node-manager-backup

cd /opt/kube-node-manager
./scripts/backup.sh

# 清理旧备份（保留30天）
find backups/ -name "*.db" -mtime +30 -delete
```

### 恢复数据
```bash
# 停止服务
docker-compose down

# 恢复数据库
cp backups/backup_20240101_120000.db data/kube-node-manager.db

# 恢复配置
tar -xzf backups/backup_20240101_120000_configs.tar.gz

# 重启服务
docker-compose up -d
```

---

**注意事项**:
- 生产环境部署前请仔细阅读安全配置
- 定期备份数据和配置文件
- 监控系统资源使用情况
- 及时更新到最新版本