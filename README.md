# Kubernetes 节点管理器

一个现代化的 Kubernetes 节点管理平台，支持多集群管理、节点标签和污点操作、用户权限管理等功能。

## 🌟 功能特性

### 核心功能
- **多集群管理** - 支持添加和管理多个 Kubernetes 集群
- **节点概览** - 实时显示节点状态、角色、资源使用情况
- **标签管理** - 支持批量添加/删除节点标签，提供标签模板功能
- **污点管理** - 支持批量管理节点污点（NoSchedule、PreferNoSchedule、NoExecute）
- **用户管理** - 基于角色的权限控制（Admin、User、Viewer）
- **LDAP集成** - 支持 LDAP 认证和用户同步
- **操作审计** - 完整的操作日志记录和查询

### 技术特性
- **响应式设计** - 支持桌面和移动端访问
- **实时搜索** - 支持按节点名称、标签等条件搜索过滤
- **批量操作** - 支持批量标签和污点管理
- **权限控制** - 细粒度的权限管理和访问控制
- **容器化部署** - 完整的 Docker 容器化方案

## 🏗️ 技术栈

### 后端
- **框架**: Go + Gin
- **数据库**: SQLite3
- **认证**: JWT + LDAP
- **Kubernetes**: client-go

### 前端
- **框架**: Vue 3 + Composition API
- **UI组件**: Element Plus
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **构建工具**: Vite

### 部署
- **容器化**: Docker + Docker Compose
- **反向代理**: Nginx
- **健康检查**: 内置健康检查端点

## 🚀 快速开始

### 前置要求
- Docker 20.0+
- Docker Compose 2.0+
- 有效的 Kubernetes 集群访问权限

### 1. 克隆项目
```bash
git clone <repository-url>
cd kube-node-manager
```

### 2. 配置环境变量
```bash
cp .env.example .env
# 编辑 .env 文件，配置JWT密钥等参数
```

### 3. 配置 Kubernetes 访问
```bash
# 确保有正确的 kubeconfig 文件
mkdir -p ~/.kube
# 复制你的 kubeconfig 文件到 ~/.kube/config
```

### 4. 启动服务
```bash
docker-compose up -d
```

### 5. 访问应用
- Web界面: http://localhost:8080
- API接口: http://localhost:8080/api/v1
- 默认账户: admin / admin123

## 📁 项目结构

```
kube-node-manager/
├── backend/                 # Go后端
│   ├── cmd/                # 应用入口
│   ├── internal/           # 内部业务逻辑
│   │   ├── handler/        # HTTP处理器
│   │   ├── service/        # 业务服务层
│   │   ├── model/          # 数据模型
│   │   ├── middleware/     # 中间件
│   │   └── config/         # 配置管理
│   └── pkg/                # 公共包
├── frontend/               # Vue3前端
│   ├── src/
│   │   ├── components/     # 组件
│   │   ├── views/          # 页面
│   │   ├── store/          # 状态管理
│   │   ├── router/         # 路由配置
│   │   └── api/            # API服务
│   └── public/
├── Dockerfile              # 多阶段构建
├── docker-compose.yml      # Docker编排
├── .env.example           # 环境变量模板
└── README.md
```

## 🔧 开发指南

### 后端开发

#### 启动开发服务器
```bash
cd backend
go mod tidy
go run cmd/main.go
```

#### 运行测试
```bash
go test ./...
```

### 前端开发

#### 安装依赖
```bash
cd frontend
npm install
```

#### 启动开发服务器
```bash
npm run dev
```

#### 构建生产版本
```bash
npm run build
```

## 🔐 权限说明

### 用户角色
- **Admin（管理员）**: 拥有所有权限，可以管理用户、集群、节点
- **User（操作员）**: 可以管理节点标签和污点，查看所有信息
- **Viewer（观察者）**: 只能查看信息，无法进行任何修改操作

### API权限
- 所有API都需要JWT认证
- 基于用户角色进行权限控制
- 管理员才能访问用户管理接口

## 🔌 LDAP 集成

支持 LDAP 认证，配置示例：

```yaml
ldap:
  enabled: true
  host: "ldap.example.com"
  port: 389
  base_dn: "dc=example,dc=com"
  user_filter: "(uid=%s)"
  admin_dn: "cn=admin,dc=example,dc=com"
  admin_pass: "admin_password"
```

## 📊 监控和日志

### 健康检查
- 后端: `GET /api/v1/health`
- 前端: `GET /health`

### 日志级别
- INFO: 一般操作日志
- WARNING: 警告信息
- ERROR: 错误信息

### 审计日志
所有关键操作都会记录审计日志，包括：
- 用户登录/登出
- 节点标签/污点修改
- 用户管理操作
- 集群配置变更

## 🛡️ 安全特性

### 认证安全
- JWT Token 认证
- Token 过期时间控制
- 刷新Token机制

### 数据安全
- 密码哈希存储
- 敏感信息加密
- SQL注入防护

### 网络安全
- HTTPS 支持
- CORS 配置
- 安全头部设置

## 🐛 故障排除

### 常见问题

#### 1. 无法连接 Kubernetes 集群
- 检查 kubeconfig 文件是否正确
- 确认网络连接
- 验证集群凭据

#### 2. 登录失败
- 检查用户名密码
- 确认JWT密钥配置
- 查看后端日志

#### 3. 前端页面无法加载
- 检查后端服务是否启动
- 确认代理配置
- 查看浏览器控制台错误

### 日志查看
```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
docker-compose logs -f frontend
```

## 🔄 更新和维护

### 更新应用
```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker-compose up --build -d
```

### 数据备份
```bash
# 备份数据库
cp data/kube-node-manager.db backup/kube-node-manager-$(date +%Y%m%d).db
```

### 清理数据
```bash
# 停止服务
docker-compose down

# 清理数据卷
docker-compose down -v
```

## 📝 许可证

本项目采用 MIT 许可证。详情请参见 [LICENSE](LICENSE) 文件。

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request 来改进这个项目。

### 贡献步骤
1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📞 支持

如果您有任何问题或建议，请：
- 提交 [Issue](issues)
- 
---

**注意**: 请在生产环境中修改默认密码和JWT密钥！