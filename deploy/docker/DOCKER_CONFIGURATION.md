# Docker 部署配置指南

本文档介绍如何使用环境变量灵活配置 Docker 部署，避免硬编码配置。

## 配置文件说明

### 开发环境配置

创建 `deploy/docker/.env.dev` 文件：

```bash
# 开发环境Docker配置
GIN_MODE=debug
DATABASE_DSN=./data/kube-node-manager.db

# 端口配置 
FRONTEND_PORT=3000
BACKEND_PORT=8080

# JWT配置 (开发环境)
JWT_SECRET=development-jwt-secret-not-for-production
JWT_EXPIRE_TIME=86400

# 前端配置
VITE_API_BASE_URL=
VITE_API_TARGET=http://backend:8080
VITE_ENABLE_LDAP=false

# LDAP配置 (默认禁用)
LDAP_ENABLED=false
```

### 生产环境配置

创建 `deploy/docker/.env.prod` 文件：

```bash
# 生产环境Docker配置
IMAGE_TAG=latest
GIN_MODE=release
DATABASE_DSN=./data/kube-node-manager.db

# 容器名称
APP_CONTAINER_NAME=kube-node-manager-prod
NGINX_CONTAINER_NAME=kube-node-manager-nginx

# 端口配置
APP_PORT=8080
HOST_PORT=8080
HTTP_PORT=80
HTTPS_PORT=443

# JWT配置 (请修改密码!)
JWT_SECRET=YOUR-STRONG-SECRET-HERE
JWT_EXPIRE_TIME=86400

# 前端配置
VITE_API_BASE_URL=
VITE_ENABLE_LDAP=false

# LDAP配置 (根据需要启用)
LDAP_ENABLED=false
# LDAP_HOST=ldap.your-company.com
# LDAP_PORT=389
# LDAP_BASE_DN=dc=your-company,dc=com

# SSL和日志路径
SSL_CERT_PATH=./nginx/ssl
NGINX_LOG_PATH=./logs/nginx
```

## 使用方式

### 开发环境启动

```bash
# 进入Docker配置目录
cd deploy/docker

# 使用开发环境配置启动
docker-compose -f docker-compose.dev.yml --env-file .env.dev up

# 或者使用环境变量直接启动
VITE_API_TARGET=https://prod-nodemgr.srv.deeproute.cn \
docker-compose -f docker-compose.dev.yml up
```

### 生产环境启动

```bash
# 构建镜像（如果需要）
docker build -t kube-node-manager:latest .

# 使用生产环境配置启动
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

### 连接到生产环境API进行开发

```bash
# 方式1: 通过环境变量
VITE_API_TARGET=https://prod-nodemgr.srv.deeproute.cn \
docker-compose -f docker-compose.dev.yml up

# 方式2: 修改 .env.dev 文件
# VITE_API_TARGET=https://prod-nodemgr.srv.deeproute.cn
```

## 环境变量说明

### 基础配置

| 变量名 | 说明 | 默认值 | 示例 |
|--------|------|--------|------|
| `IMAGE_TAG` | 镜像标签 | `latest` | `v1.0.0` |
| `GIN_MODE` | 运行模式 | `release` | `debug` |
| `DATABASE_DSN` | 数据库连接 | `./data/kube-node-manager.db` | - |

### 端口配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `FRONTEND_PORT` | 前端端口 | `3000` |
| `BACKEND_PORT` | 后端端口 | `8080` |
| `HOST_PORT` | 主机端口 | `8080` |
| `HTTP_PORT` | HTTP端口 | `80` |
| `HTTPS_PORT` | HTTPS端口 | `443` |

### JWT配置

| 变量名 | 说明 | 必填 |
|--------|------|------|
| `JWT_SECRET` | JWT密钥 | 生产环境必填 |
| `JWT_EXPIRE_TIME` | 过期时间(秒) | 否 |

### 前端配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `VITE_API_BASE_URL` | API基础URL | `""` |
| `VITE_API_TARGET` | 代理目标 | `http://backend:8080` |
| `VITE_ENABLE_LDAP` | 启用LDAP | `false` |

### LDAP配置

| 变量名 | 说明 | 必填 |
|--------|------|------|
| `LDAP_ENABLED` | 启用LDAP | 否 |
| `LDAP_HOST` | LDAP服务器 | 启用时必填 |
| `LDAP_PORT` | LDAP端口 | 否 |
| `LDAP_BASE_DN` | 基础DN | 启用时必填 |

## 最佳实践

### 1. 安全性

- 生产环境必须使用强JWT密钥
- 不要在代码中提交包含敏感信息的`.env`文件
- 使用Docker secrets管理敏感配置

### 2. 灵活性

- 所有硬编码的配置都已改为环境变量
- 支持通过命令行参数临时覆盖配置
- 不同环境使用不同的配置文件

### 3. 维护性

- 配置文件包含详细注释
- 提供默认值避免配置遗漏
- 将相关配置分组管理

## 常见用例

### 本地开发连接生产API

```bash
# 临时连接生产环境进行开发
VITE_API_TARGET=https://prod-nodemgr.srv.deeproute.cn \
docker-compose -f docker-compose.dev.yml up frontend
```

### 多环境部署

```bash
# 测试环境
docker-compose -f docker-compose.prod.yml --env-file .env.test up -d

# 预生产环境  
docker-compose -f docker-compose.prod.yml --env-file .env.staging up -d

# 生产环境
docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

### 自定义端口部署

```bash
# 使用自定义端口
HOST_PORT=9080 HTTP_PORT=8080 \
docker-compose -f docker-compose.prod.yml up -d
```
