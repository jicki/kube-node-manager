<div align="center">

# 🚀 Kubernetes 节点管理器

[![Version](https://img.shields.io/badge/version-v2.27.0-blue.svg)](https://github.com/your-repo/kube-node-manager)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/vue-3.x-brightgreen.svg)](https://vuejs.org/)

**一个功能强大的企业级 Kubernetes 节点管理平台**

支持多集群管理、节点标签和污点批量操作、Ansible 自动化运维、飞书机器人集成、GitLab Runner 管理等功能

[功能特性](#-功能特性) • [快速开始](#-快速开始) • [部署方式](#-部署方式) • [文档](#-文档) • [贡献指南](#-贡献指南)

</div>

---

## 📖 目录

- [功能特性](#-功能特性)
  - [核心功能](#核心功能)
  - [Ansible 自动化运维](#-ansible-自动化运维)
  - [技术特性](#技术特性)
- [技术架构](#️-技术架构)
- [快速开始](#-快速开始)
- [部署方式](#-部署方式)
- [使用指南](#-使用指南)
  - [Ansible 自动化运维使用](#ansible-自动化运维使用)
  - [飞书机器人使用](#飞书机器人使用)
  - [kubectl 插件使用](#kubectl-插件使用)
- [项目结构](#-项目结构)
- [开发指南](#-开发指南)
- [文档](#-文档)
- [安全说明](#️-安全说明)
- [故障排除](#-故障排除)
- [贡献指南](#-贡献指南)
- [许可证](#-许可证)

---

## 🌟 功能特性

### 核心功能

#### 🎯 集群与节点管理
- **多集群支持** - 统一管理多个 Kubernetes 集群，支持动态添加和切换
- **节点可视化** - 实时展示节点状态、角色、资源使用情况和调度信息
- **节点调度控制** - 支持节点 Cordon/Uncordon 操作，带原因和时间记录
- **健康状态监控** - 自动检测节点健康状态，支持集群健康检查

#### 🏷️ 标签与污点管理
- **批量标签操作** - 支持批量添加、修改、删除节点标签
- **标签模板** - 预定义标签模板，快速应用常用标签配置
- **污点批量管理** - 批量管理节点污点（NoSchedule、PreferNoSchedule、NoExecute）
- **智能过滤** - 支持按标签、污点、节点名称等多维度搜索和过滤

#### 👥 用户与权限
- **RBAC 权限控制** - 基于角色的访问控制（Admin、User、Viewer）
- **LDAP 认证集成** - 支持 LDAP/AD 用户认证和自动同步
- **JWT Token 认证** - 安全的 Token 机制，支持自动刷新
- **操作审计日志** - 完整记录所有关键操作，支持审计追溯

#### 🤖 飞书机器人集成
- **命令式交互** - 通过飞书机器人执行节点管理操作
- **批量操作** - 支持批量 cordon/uncordon 节点
- **快捷命令** - 快速查看集群状态、问题节点和健康信息
- **交互式卡片** - 丰富的卡片交互界面，支持分页和搜索
- **会话管理** - 智能会话状态管理，支持多集群上下文切换
- **实时通知** - 操作结果实时推送，支持失败详情展示

#### 🦊 GitLab Runner 管理
- **Runner 配置** - 统一管理 GitLab Runner 配置和部署
- **Token 管理** - 安全管理 GitLab Runner Token
- **批量创建** - 支持批量创建和配置 Runner

#### 🤖 Ansible 自动化运维
- **任务管理** - 创建、执行、取消、重试 Ansible Playbook 任务
- **模板管理** - Playbook 模板复用，支持变量定义和必需参数验证
- **主机清单** - 手动创建或从 K8s 集群自动生成主机清单
- **SSH 密钥管理** - 统一管理 SSH 私钥和密码，支持加密存储
- **定时任务** - 基于 Cron 表达式的定时任务调度
- **工作流编排** - DAG 有向无环图工作流，支持任务依赖和条件执行
- **分批执行** - 按批次执行任务，支持暂停/继续/停止控制
- **前置检查** - 执行前的连接性、资源和配置检查
- **Dry Run 模式** - 检查模式运行，不实际执行变更
- **实时日志** - WebSocket 实时推送任务执行日志
- **任务队列** - 优先级队列管理，支持高/中/低优先级
- **执行可视化** - 任务执行时间线和主机状态可视化展示
- **收藏功能** - 收藏常用模板、清单和任务配置
- **快速重执行** - 基于历史记录快速创建新任务
- **标签管理** - 任务分类标签，支持批量标签操作
- **超时控制** - 任务执行超时自动终止
- **重试策略** - 失败自动重试和重试间隔配置
- **执行统计** - 任务执行统计和成功率分析

#### 🔧 kubectl 插件
- **kubectl-node_mgr 插件** - 扩展 kubectl 命令行工具
- **节点归属查看** - 快速查看节点归属标签
- **智能 Cordon** - 带详细说明的节点 cordon 操作
- **多格式输出** - 支持 table、JSON、YAML 等输出格式

### 技术特性

- ✨ **现代化 UI** - 响应式设计，支持桌面和移动端访问
- 🔍 **实时搜索** - 高性能的实时搜索和过滤功能
- ⚡ **批量处理** - 高效的批量操作，支持进度追踪
- 🔐 **安全加固** - 多层次的安全防护机制
- 📊 **数据持久化** - 支持 SQLite 和 PostgreSQL 数据库
- 🐳 **容器化部署** - 完整的 Docker 和 Kubernetes 部署方案
- 📈 **性能优化** - 缓存机制、连接池、异步处理
- 🔄 **高可用** - 支持多副本部署和滚动更新

## 🏗️ 技术架构

### 后端技术栈
```
Go 1.21+
├── Web 框架: Gin
├── 数据库: SQLite3 / PostgreSQL
├── ORM: GORM
├── 认证: JWT + LDAP
├── K8s 客户端: client-go
├── 飞书 SDK: lark-sdk-go
└── 日志: 结构化日志 (JSON)
```

### 前端技术栈
```
Vue 3.x
├── 组件库: Element Plus
├── 状态管理: Pinia
├── 路由: Vue Router 4
├── HTTP 客户端: Axios
├── 构建工具: Vite
└── 样式: CSS3 + Flexbox
```

### 部署架构
```
容器化部署
├── Docker: 多阶段构建
├── Docker Compose: 单机/开发环境
├── Kubernetes: 生产环境
│   ├── StatefulSet: 有状态应用
│   ├── Service: 服务发现
│   ├── Ingress: 外部访问
│   └── RBAC: 权限管理
├── Nginx: 反向代理 + 静态资源
└── 数据持久化: Volume / PVC
```

### 系统架构图
```
┌─────────────────────────────────────────────────────────┐
│                     用户层                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ Web UI   │  │飞书机器人 │  │ kubectl  │              │
│  │  浏览器   │  │   客户端  │  │  插件    │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└──────────┬───────────┬────────────┬────────────────────┘
           │           │            │
           │           │            │ gRPC/REST
           │           │            │
┌──────────┴───────────┴────────────┴────────────────────┐
│                   API 网关层                             │
│  ┌──────────────────────────────────────────────────┐  │
│  │           Nginx / Ingress                        │  │
│  │    (反向代理、负载均衡、HTTPS)                     │  │
│  └──────────────────────────────────────────────────┘  │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────────┐
│                   应用服务层                             │
│  ┌────────────────────────────────────────────────┐    │
│  │         kube-node-manager (Gin)                │    │
│  ├────────────────────────────────────────────────┤    │
│  │ ┌─────────┐ ┌─────────┐ ┌──────────┐          │    │
│  │ │认证鉴权  │ │节点管理  │ │标签污点  │          │    │
│  │ └─────────┘ └─────────┘ └──────────┘          │    │
│  │ ┌─────────┐ ┌─────────┐ ┌──────────┐          │    │
│  │ │用户管理  │ │集群管理  │ │飞书集成  │          │    │
│  │ └─────────┘ └─────────┘ └──────────┘          │    │
│  │ ┌─────────┐ ┌─────────┐ ┌──────────┐          │    │
│  │ │GitLab   │ │审计日志  │ │健康检查  │          │    │
│  │ └─────────┘ └─────────┘ └──────────┘          │    │
│  │ ┌──────────────────────────────────┐          │    │
│  │ │      Ansible 自动化运维           │          │    │
│  │ │  任务 | 模板 | 清单 | 工作流     │          │    │
│  │ └──────────────────────────────────┘          │    │
│  └────────────────────────────────────────────────┘    │
└───────────┬─────────────┬──────────────┬───────────────┘
            │             │              │
    ┌───────┴───┐   ┌─────┴──────┐  ┌───┴──────────┐
    │  数据库    │   │ Kubernetes │  │  LDAP/AD     │
    │ SQLite/   │   │  Clusters  │  │   服务器      │
    │ PostgreSQL│   │  (多集群)   │  │              │
    └───────────┘   └────────────┘  └──────────────┘
```

## 🚀 快速开始

### 前置要求

| 组件 | 版本要求 | 说明 |
|------|---------|------|
| Docker | 20.0+ | 容器运行环境 |
| Docker Compose | 2.0+ | 容器编排工具 |
| Kubernetes | 1.19+ | 至少一个可访问的集群 |
| Go | 1.21+ | 后端开发（可选） |
| Node.js | 16.0+ | 前端开发（可选） |

### 方式一：Docker Compose 快速部署（推荐新手）

最快 5 分钟完成部署！

```bash
# 1. 克隆项目
git clone <repository-url>
cd kube-node-manager

# 2. 准备 kubeconfig（确保你有集群访问权限）
mkdir -p ~/.kube
# 复制你的 kubeconfig 文件到 ~/.kube/config

# 3. 一键启动（使用默认配置）
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 查看日志
docker-compose logs -f
```

**访问应用**：
- 🌐 Web 界面：http://localhost:8080
- 📡 API 接口：http://localhost:8080/api/v1
- 🔑 默认账户：`admin` / `admin123`

### 方式二：Kubernetes 生产部署

适用于生产环境和高可用场景。

```bash
# 1. 克隆项目
git clone <repository-url>
cd kube-node-manager

# 2. 配置环境变量（可选）
export NAMESPACE=kube-system          # 部署命名空间
export DOMAIN=kube-mgr.example.com    # 访问域名

# 3. 部署到 Kubernetes
make k8s-deploy

# 4. 查看部署状态
kubectl get pods,svc,ingress -n $NAMESPACE -l app=kube-node-manager

# 5. 查看应用日志
make k8s-logs
```

详细部署文档请参考：[Kubernetes 部署指南](deploy/k8s/README.md)

### 方式三：本地开发环境

```bash
# 1. 启动后端
cd backend
go mod tidy
go run cmd/main.go

# 2. 启动前端（新终端）
cd frontend
npm install
npm run dev
```

### 首次登录配置

1. 使用默认管理员账户登录：`admin` / `admin123`
2. ⚠️ **重要**：立即修改默认密码
3. 添加你的第一个 Kubernetes 集群：
   - 进入 **集群管理** 页面
   - 点击 **添加集群**
   - 上传 kubeconfig 文件或粘贴内容
   - 点击 **测试连接** 验证
   - 保存配置

4. 开始管理节点：
   - 选择集群
   - 查看节点列表
   - 执行标签或污点操作

## 📁 项目结构

```
kube-node-manager/
├── backend/                    # Go 后端服务
│   ├── cmd/
│   │   └── main.go            # 应用入口
│   ├── internal/
│   │   ├── handler/           # HTTP 请求处理器
│   │   │   ├── auth/          # 认证处理
│   │   │   ├── cluster/       # 集群管理
│   │   │   ├── node/          # 节点管理
│   │   │   ├── label/         # 标签管理
│   │   │   ├── taint/         # 污点管理
│   │   │   ├── user/          # 用户管理
│   │   │   ├── ansible/       # Ansible 模块处理器
│   │   │   │   ├── handler.go          # 任务管理接口
│   │   │   │   ├── template.go         # 模板管理接口
│   │   │   │   ├── inventory.go        # 清单管理接口
│   │   │   │   ├── sshkey.go           # SSH 密钥接口
│   │   │   │   ├── schedule.go         # 定时任务接口
│   │   │   │   ├── workflow.go         # 工作流接口
│   │   │   │   ├── queue.go            # 队列管理接口
│   │   │   │   ├── favorite.go         # 收藏功能接口
│   │   │   │   ├── tag.go              # 标签管理接口
│   │   │   │   ├── estimation.go       # 执行估算接口
│   │   │   │   ├── visualization.go    # 可视化接口
│   │   │   │   └── websocket.go        # WebSocket 日志流
│   │   │   ├── feishu/        # 飞书集成
│   │   │   ├── gitlab/        # GitLab 集成
│   │   │   ├── audit/         # 审计日志
│   │   │   └── health/        # 健康检查
│   │   ├── service/           # 业务逻辑层
│   │   │   ├── auth/          # 认证服务
│   │   │   ├── cluster/       # 集群服务
│   │   │   ├── k8s/           # Kubernetes 客户端
│   │   │   ├── ansible/       # Ansible 自动化服务
│   │   │   │   ├── service.go          # 服务主入口
│   │   │   │   ├── executor.go         # 任务执行器
│   │   │   │   ├── template.go         # 模板管理
│   │   │   │   ├── inventory.go        # 主机清单管理
│   │   │   │   ├── sshkey.go           # SSH 密钥管理
│   │   │   │   ├── schedule.go         # 定时任务调度
│   │   │   │   ├── queue.go            # 任务队列管理
│   │   │   │   ├── workflow.go         # 工作流服务
│   │   │   │   ├── workflow_executor.go # 工作流执行器
│   │   │   │   ├── workflow_validator.go # DAG 验证器
│   │   │   │   ├── preflight.go        # 前置检查
│   │   │   │   ├── favorite.go         # 收藏功能
│   │   │   │   ├── tag.go              # 标签管理
│   │   │   │   ├── estimation.go       # 执行估算
│   │   │   │   └── visualization.go    # 可视化数据
│   │   │   ├── feishu/        # 飞书机器人服务
│   │   │   │   ├── bot.go              # 机器人核心
│   │   │   │   ├── command_*.go        # 命令处理器
│   │   │   │   ├── card_*.go           # 卡片构建器
│   │   │   │   └── event_client.go     # 事件客户端
│   │   │   ├── gitlab/        # GitLab 服务
│   │   │   ├── ldap/          # LDAP 服务
│   │   │   └── progress/      # 进度追踪
│   │   ├── model/             # 数据模型
│   │   │   ├── user.go        # 用户模型
│   │   │   ├── cluster.go     # 集群模型
│   │   │   ├── ansible.go     # Ansible 相关模型
│   │   │   │   # - AnsibleTask（任务）
│   │   │   │   # - AnsibleTemplate（模板）
│   │   │   │   # - AnsibleInventory（主机清单）
│   │   │   │   # - AnsibleSSHKey（SSH 密钥）
│   │   │   │   # - AnsibleSchedule（定时任务）
│   │   │   │   # - AnsibleWorkflow（工作流）
│   │   │   │   # - AnsibleWorkflowExecution（工作流执行）
│   │   │   │   # - AnsibleTag（标签）
│   │   │   │   # - AnsibleFavorite（收藏）
│   │   │   │   # - AnsibleTaskHistory（执行历史）
│   │   │   ├── audit.go       # 审计日志模型
│   │   │   └── migrate.go     # 数据库迁移
│   │   └── config/            # 配置管理
│   │       └── config.go      # 配置加载
│   ├── pkg/                   # 公共库
│   │   ├── database/          # 数据库连接
│   │   ├── logger/            # 日志工具
│   │   └── static/            # 静态资源
│   ├── configs/               # 配置文件模板
│   │   ├── config.yaml.example
│   │   └── config-postgres.yaml.example
│   ├── go.mod                 # Go 依赖管理
│   └── Dockerfile.dev         # 开发环境镜像
│
├── frontend/                  # Vue 3 前端
│   ├── src/
│   │   ├── views/            # 页面组件
│   │   │   ├── dashboard/    # 仪表盘
│   │   │   ├── clusters/     # 集群管理
│   │   │   ├── nodes/        # 节点管理
│   │   │   ├── labels/       # 标签管理
│   │   │   ├── taints/       # 污点管理
│   │   │   ├── users/        # 用户管理
│   │   │   ├── feishu/       # 飞书配置
│   │   │   ├── gitlab/       # GitLab 配置
│   │   │   ├── audit/        # 审计日志
│   │   │   └── login/        # 登录页面
│   │   ├── components/       # 公共组件
│   │   │   ├── common/       # 通用组件
│   │   │   └── layout/       # 布局组件
│   │   ├── api/              # API 接口封装
│   │   ├── router/           # 路由配置
│   │   ├── store/            # Pinia 状态管理
│   │   ├── utils/            # 工具函数
│   │   ├── App.vue           # 根组件
│   │   └── main.js           # 应用入口
│   ├── package.json          # 依赖配置
│   ├── vite.config.js        # Vite 配置
│   └── Dockerfile.dev        # 开发环境镜像
│
├── kubectl-plugin/            # kubectl 插件
│   ├── cmd/                  # 子命令实现
│   │   ├── root.go           # 根命令
│   │   ├── get.go            # 查看节点
│   │   ├── labels.go         # 标签管理
│   │   ├── cordon.go         # Cordon 操作
│   │   └── uncordon.go       # Uncordon 操作
│   ├── pkg/k8s/              # Kubernetes 客户端
│   ├── main.go               # 插件入口
│   ├── go.mod                # Go 依赖
│   ├── Makefile              # 构建脚本
│   └── README.md             # 插件文档
│
├── deploy/                   # 部署配置
│   ├── docker/               # Docker 部署
│   │   ├── docker-compose.yml          # 生产环境
│   │   ├── docker-compose.dev.yml      # 开发环境
│   │   ├── Dockerfile                  # 多阶段构建
│   │   └── nginx/                      # Nginx 配置
│   ├── k8s/                  # Kubernetes 部署
│   │   ├── k8s-statefulset.yaml       # 有状态部署
│   │   ├── k8s-service.yaml           # 服务配置
│   │   ├── k8s-ingress.yaml           # Ingress 配置
│   │   ├── configmap.yaml             # 配置映射
│   │   ├── rbac-patch.yaml            # RBAC 配置
│   │   ├── kustomization.yaml         # Kustomize 配置
│   │   └── README.md                  # 部署文档
│   ├── scripts/              # 部署脚本
│   │   ├── install.sh                 # Docker 安装
│   │   ├── k8s-deploy.sh              # K8s 部署
│   │   ├── backup.sh                  # 数据备份
│   │   └── k8s-cleanup.sh             # 清理脚本
│   └── README.md             # 部署总文档
│
├── docs/                     # 项目文档
│   ├── feishu-bot-*.md       # 飞书机器人文档
│   ├── gitlab-*.md           # GitLab 集成文档
│   ├── kubectl-plugin-*.md   # kubectl 插件文档
│   ├── database-*.md         # 数据库配置文档
│   └── batch-operations-*.md # 批量操作文档
│
├── scripts/                  # 运维脚本
│   ├── backup.sh             # 数据备份脚本
│   ├── migrate.sh            # 数据迁移脚本
│   ├── sqlite-to-postgres-v3.go  # SQLite 转 PostgreSQL
│   └── get_sa_kubeconfig.sh  # 获取 ServiceAccount kubeconfig
│
├── Dockerfile                # 生产环境镜像
├── Makefile                  # 项目构建脚本
├── VERSION                   # 版本文件
└── README.md                 # 项目说明文档
```

## 💡 使用指南

### Web 界面使用

#### 1. 集群管理
```
设置 → 集群管理 → 添加集群
- 上传 kubeconfig 文件或粘贴内容
- 测试连接验证
- 选择默认集群
```

#### 2. 节点管理
```
节点管理 → 查看节点列表
- 查看节点状态、角色、资源
- 执行 Cordon/Uncordon 操作
- 搜索和过滤节点
```

#### 3. 标签管理
```
标签管理 → 批量操作
- 选择多个节点
- 添加/删除标签
- 使用标签模板快速应用
```

#### 4. 污点管理
```
污点管理 → 批量操作
- 选择节点和污点效果（NoSchedule/PreferNoSchedule/NoExecute）
- 设置污点键值对
- 批量应用污点配置
```

### Ansible 自动化运维使用

#### 概述

Ansible 模块提供了完整的自动化运维能力，支持在 Kubernetes 节点上批量执行运维任务，包括系统配置、软件部署、健康检查等操作。

#### 1. SSH 密钥管理

在执行 Ansible 任务前，需要先配置 SSH 认证方式：

```
Ansible 管理 → SSH 密钥管理 → 添加密钥
```

**支持的认证方式**：
- **SSH 私钥认证**（推荐）：上传 SSH 私钥文件，支持密钥密码
- **密码认证**：使用 SSH 用户名和密码

**配置项**：
- 密钥名称：便于识别的名称
- SSH 用户名：目标主机的登录用户（如 root、ubuntu）
- SSH 端口：默认 22
- 私钥内容：RSA/Ed25519 等格式私钥
- 私钥密码：可选，如果私钥有密码保护
- 设为默认：新建清单时自动使用此密钥

**安全说明**：所有 SSH 凭据使用 AES-256 加密存储。

#### 2. 主机清单管理

主机清单定义了 Ansible 任务的执行目标主机。

**创建方式**：

##### 方式一：从 K8s 集群自动生成
```
Ansible 管理 → 主机清单 → 从集群生成
- 选择 K8s 集群
- 选择 SSH 密钥
- 可选：使用标签过滤节点（如 env=production）
- 自动获取节点内网 IP 作为主机地址
```

##### 方式二：手动创建清单
```
Ansible 管理 → 主机清单 → 新建清单
- 清单名称和描述
- 选择 SSH 密钥
- 编写 INI 或 YAML 格式的 Inventory 内容
```

**INI 格式示例**：
```ini
[webservers]
web1.example.com
web2.example.com

[databases]
db1.example.com
db2.example.com

[all:vars]
ansible_user=ubuntu
ansible_port=22
```

**YAML 格式示例**：
```yaml
all:
  children:
    webservers:
      hosts:
        web1.example.com:
        web2.example.com:
    databases:
      hosts:
        db1.example.com:
        db2.example.com:
  vars:
    ansible_user: ubuntu
    ansible_port: 22
```

#### 3. Playbook 模板管理

创建可复用的 Playbook 模板，提高运维效率。

```
Ansible 管理 → 模板管理 → 新建模板
```

**配置项**：
- **模板名称**：如"系统健康检查"、"Nginx 部署"
- **模板描述**：详细说明模板用途
- **Playbook 内容**：标准的 Ansible Playbook YAML
- **变量定义**：定义可配置的变量及默认值
- **必需变量**：标记必须提供的变量
- **风险等级**：low/medium/high，用于操作审批

**Playbook 示例**：
```yaml
---
- name: 系统资源状态检查
  hosts: all
  gather_facts: yes
  tasks:
    - name: 检查磁盘使用率
      shell: df -h
      register: disk_usage
    
    - name: 检查内存使用
      shell: free -h
      register: memory_usage
    
    - name: 检查 CPU 负载
      shell: uptime
      register: cpu_load
    
    - name: 显示结果
      debug:
        msg: |
          磁盘使用: {{ disk_usage.stdout }}
          内存使用: {{ memory_usage.stdout }}
          CPU 负载: {{ cpu_load.stdout }}
```

**带变量的 Playbook 示例**：
```yaml
---
- name: 部署 Nginx
  hosts: webservers
  become: yes
  vars:
    nginx_version: "{{ nginx_version | default('latest') }}"
    nginx_port: "{{ nginx_port | default(80) }}"
  tasks:
    - name: 安装 Nginx
      apt:
        name: "nginx={{ nginx_version }}"
        state: present
        update_cache: yes
    
    - name: 配置 Nginx 端口
      lineinfile:
        path: /etc/nginx/sites-available/default
        regexp: '^(\s*)listen'
        line: '    listen {{ nginx_port }};'
      notify: 重启 Nginx
    
  handlers:
    - name: 重启 Nginx
      service:
        name: nginx
        state: restarted
```

#### 4. 执行 Ansible 任务

##### 快速执行
```
Ansible 管理 → 任务管理 → 创建任务
```

**基本配置**：
1. **任务名称**：描述性名称
2. **选择模板**：使用已有模板或直接编写 Playbook
3. **选择清单**：指定目标主机清单
4. **关联集群**（可选）：关联 K8s 集群便于追踪

**高级选项**：

##### Dry Run 模式（检查模式）
```
☑ 启用 Dry Run 模式
- 不会实际执行变更操作
- 仅检查语法和预测可能的变更
- 适合生产环境操作前验证
```

##### 分批执行
```
☑ 启用分批执行
- 批次大小：固定数量（如 5 台）或百分比（如 20%）
- 批次间暂停：每批执行后需手动确认继续
- 失败阈值：超过 N 台主机失败则停止
- 单批失败率：超过百分比则停止（如 30%）
```

**应用场景**：大规模滚动更新、逐步灰度部署

##### 超时控制
```
超时时间：3600 秒
- 0 表示不限制
- 超时自动终止任务
```

##### 任务优先级
```
优先级：高/中/低
- 影响队列中的执行顺序
- 紧急任务使用高优先级
```

##### 变量传递
```json
{
  "nginx_version": "1.20.2",
  "nginx_port": 8080,
  "enable_ssl": true
}
```

#### 5. 任务监控与控制

##### 实时查看任务执行

**任务列表**：
- 查看所有任务的状态、进度、耗时
- 按状态筛选：运行中/成功/失败/已取消
- 按集群、模板筛选

**任务详情**：
- 实时日志输出（WebSocket）
- 执行统计：总主机数、成功/失败/跳过数
- 执行时间线：各阶段耗时可视化
- 主机状态列表：每台主机的详细执行情况

##### 任务控制操作

**取消任务**：
```
任务详情 → 取消任务
- 终止正在执行的任务
- 已完成的主机不会回滚
```

**暂停/继续批次**（分批执行时）：
```
任务详情 → 暂停批次
- 当前批次完成后暂停
- 人工检查后再继续下一批
```

**重试失败任务**：
```
任务详情 → 重试
- 仅在失败的主机上重新执行
- 保留原有配置和变量
```

#### 6. 定时任务调度

创建周期性执行的自动化任务。

```
Ansible 管理 → 定时任务 → 新建调度
```

**配置项**：
- **调度名称**：如"每日健康检查"
- **选择模板**：要执行的 Playbook 模板
- **选择清单**：目标主机清单
- **Cron 表达式**：定义执行周期
- **额外变量**：可选，覆盖模板默认值
- **启用状态**：开启/关闭调度

**Cron 表达式示例**：
```bash
# 每天凌晨 2 点执行
0 0 2 * * *

# 每小时执行
0 0 * * * *

# 每周一上午 9 点执行
0 0 9 * * 1

# 每 30 分钟执行
0 */30 * * * *
```

**调度管理**：
- 查看下次执行时间
- 查看历史执行记录
- 临时禁用/启用调度
- 查看执行统计

#### 7. 工作流编排（DAG）

使用有向无环图编排复杂的多步骤自动化流程。

```
Ansible 管理 → 工作流管理 → 新建工作流
```

**工作流组件**：
- **开始节点**：工作流入口（有且仅有一个）
- **任务节点**：执行 Ansible 任务（可配置模板、清单、变量）
- **结束节点**：工作流出口（有且仅有一个）
- **连接线**：定义任务执行顺序和依赖关系

**功能特性**：
- **依赖管理**：任务按依赖顺序执行
- **并行执行**：无依赖关系的任务可并行执行
- **条件执行**：根据前置任务结果决定是否执行
- **可视化编辑**：拖拽式 DAG 图编辑器
- **循环检测**：自动检测并禁止循环依赖

**应用场景**：
- 应用部署流程：拉取代码 → 构建 → 测试 → 部署 → 健康检查
- 系统初始化：安装基础软件 → 配置防火墙 → 安装监控 → 配置日志
- 故障恢复：检测故障 → 停止服务 → 恢复数据 → 重启服务 → 验证

**工作流执行**：
```
工作流管理 → 选择工作流 → 执行
- 查看实时执行状态
- 可视化显示当前执行节点
- 查看各任务的详细日志
- 失败自动停止后续依赖任务
```

#### 8. 前置检查

在执行任务前进行环境检查，降低执行风险。

```
任务创建 → 高级选项 → 执行前置检查
```

**检查项**：

| 类别 | 检查项 | 说明 |
|------|--------|------|
| **连接性检查** | SSH 连通性 | 验证能否连接所有目标主机 |
| | SSH 认证 | 验证 SSH 密钥或密码是否正确 |
| | 网络延迟 | 测试网络连接质量 |
| **资源检查** | 磁盘空间 | 检查可用磁盘空间是否充足 |
| | 内存资源 | 检查可用内存 |
| | CPU 负载 | 检查当前系统负载 |
| **配置检查** | Ansible 版本 | 验证目标主机 Ansible 版本兼容性 |
| | Python 版本 | 检查 Python 环境 |
| | 必需软件 | 检查依赖软件是否已安装 |

**检查结果**：
- ✅ **通过**：所有检查项通过，可以安全执行
- ⚠️ **警告**：部分检查项异常，建议修复后执行
- ❌ **失败**：严重问题，禁止执行任务

#### 9. 收藏与快速操作

**收藏功能**：
```
模板/清单/任务 → 收藏
- 收藏常用配置
- 快速访问收藏夹
- 一键创建相同配置的任务
```

**执行历史**：
```
任务管理 → 执行历史
- 查看历史执行记录
- 基于历史快速重建任务
- 保留原有变量和配置
- 一键重新执行
```

#### 10. 标签管理

使用标签对任务进行分类管理。

```
Ansible 管理 → 标签管理
```

**功能**：
- 创建自定义标签（支持颜色分类）
- 为任务批量添加/移除标签
- 按标签筛选任务
- 标签统计和分析

**应用场景**：
- 环境标签：dev、staging、production
- 类型标签：部署、配置、监控、备份
- 项目标签：project-a、project-b

#### 11. 任务队列与优先级

系统自动管理任务队列，支持优先级调度。

**优先级级别**：
- **高优先级**：紧急任务、故障恢复
- **中优先级**：常规运维任务（默认）
- **低优先级**：批量巡检、清理任务

**队列管理**：
```
Ansible 管理 → 任务队列
- 查看等待执行的任务
- 查看当前执行的任务
- 修改任务优先级
- 取消排队中的任务
```

#### 12. 执行统计与分析

查看 Ansible 模块的整体使用情况。

```
Ansible 管理 → 统计分析
```

**统计指标**：
- 总任务数、成功率、失败率
- 平均执行时长
- 最常用的模板和清单
- 各集群的任务分布
- 用户操作统计
- 每日/每周/每月趋势图

#### 安全与权限

**权限要求**：
- Ansible 模块仅限管理员（Admin）角色访问
- 所有操作记录审计日志
- SSH 凭据加密存储

**最佳实践**：
1. 使用 SSH 密钥认证，避免使用密码
2. 为不同环境创建独立的 SSH 密钥
3. 敏感操作前先执行 Dry Run
4. 使用分批执行进行大规模变更
5. 配置合理的超时时间
6. 定期清理历史任务日志
7. 使用工作流编排复杂流程
8. 充分利用模板复用 Playbook

### 飞书机器人使用

#### 配置机器人
1. 在飞书开放平台创建应用
2. 获取 App ID 和 App Secret
3. 在系统中配置飞书应用信息
4. 添加机器人到群聊

#### 常用命令

```bash
# 帮助信息
/help                          # 查看所有命令
/help cluster                  # 查看集群命令帮助
/help node                     # 查看节点命令帮助

# 集群管理
/cluster list                  # 查看所有集群
/cluster set <集群名>          # 切换当前集群
/cluster info                  # 查看当前集群信息

# 节点管理
/node list                     # 查看节点列表
/node get <节点名>             # 查看节点详情
/node cordon <节点名> [原因]   # 禁止调度节点
/node uncordon <节点名>        # 恢复调度节点

# 批量操作
/node batch cordon node1,node2,node3 维护升级
/node batch uncordon node1,node2,node3

# 快捷命令
/quick status                  # 查看集群状态概览
/quick nodes                   # 查看问题节点
/quick health                  # 所有集群健康检查

# 标签管理
/label list <节点名>           # 查看节点标签
/label add <节点名> key=value  # 添加标签
/label remove <节点名> key     # 删除标签

# 污点管理
/taint list <节点名>           # 查看节点污点
/taint add <节点名> key=value:NoSchedule  # 添加污点
/taint remove <节点名> key     # 删除污点

# 审计日志
/audit recent                  # 查看最近操作
/audit user <用户名>           # 查看指定用户操作
```

详细使用文档：[飞书机器人使用指南](docs/feishu-bot-batch-and-quick-commands.md)

### kubectl 插件使用

#### 安装插件
```bash
cd kubectl-plugin
make build-plugin
make install-plugin

# 验证安装
kubectl node_mgr --help
```

#### 常用命令
```bash
# 查看节点调度状态
kubectl node_mgr get
kubectl node_mgr get node1

# 查看节点归属
kubectl node_mgr labels
kubectl node_mgr labels node1

# Cordon 操作
kubectl node_mgr cordon node1 --reason "系统维护"
kubectl node_mgr cordon node1,node2,node3 --reason "批量维护"

# Uncordon 操作
kubectl node_mgr uncordon node1
kubectl node_mgr uncordon node1,node2,node3

# 查看已 cordon 的节点
kubectl node_mgr cordon list

# 多格式输出
kubectl node_mgr get -o json
kubectl node_mgr labels -o yaml
```

详细使用文档：[kubectl 插件文档](kubectl-plugin/README.md)

## 🔧 开发指南

### 后端开发

```bash
# 1. 进入后端目录
cd backend

# 2. 安装依赖
go mod tidy

# 3. 配置环境（复制并编辑配置文件）
cp configs/config.yaml.example configs/config.yaml

# 4. 启动开发服务器
go run cmd/main.go

# 5. 运行测试
go test ./...

# 6. 代码格式化
go fmt ./...

# 7. 代码检查
go vet ./...
```

#### 后端项目结构说明
- `cmd/`: 应用入口
- `internal/handler/`: HTTP 请求处理
- `internal/service/`: 业务逻辑实现
- `internal/model/`: 数据模型定义
- `pkg/`: 可复用的公共库

### 前端开发

```bash
# 1. 进入前端目录
cd frontend

# 2. 安装依赖
npm install

# 3. 启动开发服务器（默认 http://localhost:5173）
npm run dev

# 4. 构建生产版本
npm run build

# 5. 预览生产构建
npm run preview

# 6. 代码检查
npm run lint
```

#### 前端项目结构说明
- `src/views/`: 页面组件
- `src/components/`: 可复用组件
- `src/api/`: API 接口封装
- `src/store/`: 状态管理
- `src/router/`: 路由配置

### 代码规范

#### Go 代码规范
- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 所有公共函数必须添加注释
- 错误处理不能被忽略
- 使用有意义的变量和函数名

#### Vue 代码规范
- 遵循 Vue 3 官方风格指南
- 组件名使用 PascalCase
- Props 定义必须包含类型
- 使用 Composition API
- 合理拆分组件，保持单一职责

### 提交规范

使用语义化提交消息：

```
feat: 添加新功能
fix: 修复 bug
docs: 更新文档
style: 代码格式调整
refactor: 代码重构
perf: 性能优化
test: 添加测试
chore: 构建配置或辅助工具的变动
```

示例：
```bash
git commit -m "feat: 添加飞书机器人批量操作功能"
git commit -m "fix: 修复节点列表分页显示问题"
git commit -m "docs: 更新 README 安装说明"
```

## 🔐 权限说明

### 用户角色

| 角色 | 权限说明 | 可执行操作 |
|-----|---------|----------|
| **Admin（管理员）** | 完全控制权限 | • 管理用户和角色<br>• 添加/删除集群<br>• 所有节点操作<br>• 配置系统设置<br>• 查看审计日志 |
| **User（操作员）** | 操作权限 | • 查看集群信息<br>• 管理节点标签和污点<br>• 执行 Cordon/Uncordon<br>• 查看操作日志 |
| **Viewer（观察者）** | 只读权限 | • 查看集群信息<br>• 查看节点状态<br>• 查看标签和污点<br>• 无修改权限 |

### API 权限控制

```
认证流程：
用户登录 → 获取 JWT Token → 携带 Token 访问 API → 验证身份和权限 → 返回结果
```

**权限矩阵**：

| API 路径 | Admin | User | Viewer |
|---------|-------|------|--------|
| `/api/v1/users/*` | ✅ | ❌ | ❌ |
| `/api/v1/clusters/*` (POST/PUT/DELETE) | ✅ | ❌ | ❌ |
| `/api/v1/clusters/*` (GET) | ✅ | ✅ | ✅ |
| `/api/v1/nodes/*` (POST/PUT/DELETE) | ✅ | ✅ | ❌ |
| `/api/v1/nodes/*` (GET) | ✅ | ✅ | ✅ |
| `/api/v1/labels/*` (POST/PUT/DELETE) | ✅ | ✅ | ❌ |
| `/api/v1/labels/*` (GET) | ✅ | ✅ | ✅ |
| `/api/v1/taints/*` (POST/PUT/DELETE) | ✅ | ✅ | ❌ |
| `/api/v1/taints/*` (GET) | ✅ | ✅ | ✅ |
| `/api/v1/ansible/*` | ✅ | ❌ | ❌ |
| `/api/v1/audit/*` | ✅ | ✅ | ✅ |

## 🔌 LDAP 集成

### 配置 LDAP 认证

在 `backend/configs/config.yaml` 中配置：

```yaml
ldap:
  enabled: true                              # 启用 LDAP 认证
  host: "ldap.example.com"                   # LDAP 服务器地址
  port: 389                                  # LDAP 端口（389 或 636）
  use_ssl: false                             # 是否使用 SSL
  base_dn: "dc=example,dc=com"               # 基础 DN
  user_filter: "(uid=%s)"                    # 用户过滤器
  admin_dn: "cn=admin,dc=example,dc=com"     # 管理员 DN
  admin_password: "admin_password"           # 管理员密码
  
  # 用户属性映射
  attributes:
    username: "uid"
    email: "mail"
    display_name: "cn"
```

### LDAP 认证流程

```
1. 用户输入用户名和密码
   ↓
2. 系统使用管理员账户连接 LDAP
   ↓
3. 搜索用户 DN（根据 user_filter）
   ↓
4. 使用用户 DN 和密码验证
   ↓
5. 验证成功后创建/更新本地用户
   ↓
6. 返回 JWT Token
```

### 支持的 LDAP 服务器
- ✅ OpenLDAP
- ✅ Active Directory (AD)
- ✅ FreeIPA
- ✅ 其他兼容 LDAP v3 的服务器

## 📊 监控和日志

### 健康检查端点

| 端点 | 方法 | 说明 | 响应示例 |
|-----|------|------|---------|
| `/api/v1/health` | GET | 应用健康状态 | `{"status":"healthy","timestamp":"..."}` |
| `/api/v1/health/db` | GET | 数据库连接状态 | `{"status":"connected"}` |
| `/api/v1/health/k8s` | GET | Kubernetes 连接 | `{"status":"ok","clusters":["dev","prod"]}` |

### 日志系统

#### 日志级别
```
DEBUG   → 调试信息（开发环境）
INFO    → 一般操作日志
WARNING → 警告信息
ERROR   → 错误信息
FATAL   → 致命错误
```

#### 日志格式（JSON）
```json
{
  "time": "2024-10-22T10:30:00Z",
  "level": "INFO",
  "msg": "节点标签更新成功",
  "user": "admin",
  "cluster": "production",
  "node": "node-1",
  "action": "label_update",
  "duration": "125ms"
}
```

### 审计日志

系统自动记录以下操作的审计日志：

| 操作类型 | 记录内容 |
|---------|---------|
| 用户登录/登出 | 用户名、IP 地址、时间戳 |
| 集群管理 | 添加/删除/修改集群配置 |
| 节点操作 | Cordon/Uncordon、节点名称、原因 |
| 标签管理 | 添加/删除标签、节点、键值对 |
| 污点管理 | 添加/删除污点、节点、污点信息 |
| 用户管理 | 创建/修改/删除用户、角色变更 |
| 配置变更 | 系统配置修改记录 |

**审计日志查询**：
- Web 界面：审计日志页面
- 飞书机器人：`/audit recent` 或 `/audit user <用户名>`
- API：`GET /api/v1/audit`

## 🛡️ 安全说明

### 认证与授权

#### JWT Token 机制
- **Token 有效期**：24 小时（可配置）
- **刷新机制**：Token 过期前自动刷新
- **存储方式**：浏览器 LocalStorage（前端）/ Memory（后端）
- **传输方式**：HTTP Header `Authorization: Bearer <token>`

#### 密码安全
- **加密算法**：bcrypt（cost factor: 10）
- **密码策略**（可配置）：
  - 最小长度：8 字符
  - 必须包含：大小写字母、数字
  - 禁止常见弱密码
- **密码重置**：管理员可重置用户密码

### 数据安全

#### 敏感数据保护
- ✅ 数据库密码加密存储
- ✅ Kubeconfig 文件加密存储
- ✅ API 密钥加密存储
- ✅ LDAP 密码不在日志中显示

#### SQL 注入防护
- 使用 GORM 参数化查询
- 输入验证和清理
- 预编译语句

#### XSS 防护
- 前端输出转义
- CSP（Content Security Policy）策略
- HTTP 安全头部设置

### 网络安全

#### HTTPS 配置
```nginx
# Nginx 配置示例
server {
    listen 443 ssl http2;
    server_name kube-mgr.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    
    # 强制 HTTPS
    add_header Strict-Transport-Security "max-age=31536000" always;
    
    location / {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### CORS 配置
```go
// 允许的来源（配置文件）
AllowOrigins: []string{"https://kube-mgr.example.com"}
AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
AllowHeaders: []string{"Origin", "Authorization", "Content-Type"}
ExposeHeaders: []string{"Content-Length"}
AllowCredentials: true
MaxAge: 12 * time.Hour
```

#### 安全头部
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
```

### 安全最佳实践

#### 生产环境检查清单
- [ ] 修改默认管理员密码
- [ ] 配置强密码策略
- [ ] 启用 HTTPS
- [ ] 更改默认 JWT 密钥
- [ ] 配置防火墙规则
- [ ] 限制 API 访问速率
- [ ] 定期备份数据
- [ ] 监控审计日志
- [ ] 定期更新依赖
- [ ] 配置告警通知

#### 最小权限原则
```yaml
# Kubernetes RBAC 示例
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kube-node-manager
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list", "watch", "patch", "update"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

## 🐛 故障排除

### 常见问题及解决方案

#### 1. 无法连接 Kubernetes 集群

**问题表现**：
- 集群列表显示"连接失败"
- 节点列表无法加载
- 错误消息："connection refused" 或 "unauthorized"

**排查步骤**：
```bash
# 1. 验证 kubeconfig 文件有效性
kubectl cluster-info
kubectl get nodes

# 2. 检查集群访问权限
kubectl auth can-i get nodes

# 3. 查看应用日志
docker-compose logs backend | grep -i "kubernetes"

# 4. 测试网络连接
ping <kubernetes-api-server-host>
telnet <kubernetes-api-server-host> 6443
```

**解决方案**：
- 确保 kubeconfig 文件路径正确
- 验证 ServiceAccount 权限（Kubernetes 部署）
- 检查网络策略和防火墙规则
- 确认 API Server 证书有效

#### 2. 登录失败

**问题表现**：
- 登录页面提示"用户名或密码错误"
- LDAP 认证失败
- Token 验证失败

**排查步骤**：
```bash
# 1. 检查数据库中的用户
# SQLite
sqlite3 backend/data/kube-node-manager.db "SELECT username, role FROM users;"

# 2. 查看认证相关日志
docker-compose logs backend | grep -i "auth\|login"

# 3. 测试 LDAP 连接（如果启用）
ldapsearch -x -H ldap://<ldap-host>:<port> -D "<admin-dn>" -w "<password>" -b "<base-dn>"
```

**解决方案**：
- 使用默认账户：`admin` / `admin123`
- 重置管理员密码（数据库操作）
- 检查 LDAP 配置参数
- 验证 JWT 密钥配置

#### 3. 前端页面无法加载

**问题表现**：
- 白屏或空白页面
- 静态资源 404
- 浏览器控制台错误

**排查步骤**：
```bash
# 1. 检查服务状态
docker-compose ps

# 2. 查看前端日志
docker-compose logs frontend

# 3. 查看 Nginx 日志
docker-compose logs nginx

# 4. 测试后端 API
curl http://localhost:8080/api/v1/health
```

**解决方案**：
- 确保所有服务正常运行
- 检查 Nginx 配置和代理设置
- 清除浏览器缓存
- 查看浏览器控制台错误详情

#### 4. 数据库连接失败

**问题表现**：
- 应用启动失败
- 错误消息："database connection failed"

**排查步骤**：
```bash
# 1. 检查数据库文件权限
ls -la backend/data/

# 2. 测试数据库连接
sqlite3 backend/data/kube-node-manager.db ".tables"

# 3. 查看数据库配置
cat backend/configs/config.yaml | grep -A 5 database
```

**解决方案**：
- 确保数据目录存在且有写权限
- 检查数据库文件是否损坏
- PostgreSQL: 验证连接字符串和凭据

#### 5. 飞书机器人无响应

**问题表现**：
- 发送命令无反应
- 机器人离线
- 卡片无法显示

**排查步骤**：
```bash
# 1. 检查飞书配置
curl http://localhost:8080/api/v1/feishu/status

# 2. 查看飞书相关日志
docker-compose logs backend | grep -i "feishu\|lark"

# 3. 验证 webhook 配置
# 在飞书开放平台检查事件订阅和回调地址
```

**解决方案**：
- 验证 App ID 和 App Secret
- 检查回调 URL 是否可访问
- 确认事件订阅配置正确
- 重新添加机器人到群聊

#### 6. 性能问题

**问题表现**：
- 页面加载缓慢
- API 响应延迟
- 内存占用过高

**排查步骤**：
```bash
# 1. 查看容器资源使用
docker stats

# 2. 检查数据库性能
# 查看数据库大小和索引

# 3. 分析日志中的慢请求
docker-compose logs backend | grep -i "duration\|took"
```

**解决方案**：
- 增加容器资源限制
- 优化数据库查询和索引
- 启用缓存机制
- 清理过期审计日志

### 日志查看命令

```bash
# Docker Compose 部署
docker-compose logs -f                    # 所有服务日志
docker-compose logs -f backend            # 后端日志
docker-compose logs -f frontend           # 前端日志
docker-compose logs --tail=100 backend    # 最近 100 行

# Kubernetes 部署
kubectl logs -f deployment/kube-node-manager -n kube-system
kubectl logs -f <pod-name> -n kube-system
kubectl logs <pod-name> --previous        # 上一个容器的日志
```

### 调试模式

```bash
# 启用调试日志
export LOG_LEVEL=debug
docker-compose restart backend

# 前端开发模式
cd frontend
npm run dev

# 后端开发模式
cd backend
go run cmd/main.go
```

## 🔄 部署方式

### Docker Compose 部署（单机）

适用于开发环境和小规模部署：

```bash
# 生产环境
docker-compose -f deploy/docker/docker-compose.prod.yml up -d

# 开发环境
docker-compose -f deploy/docker/docker-compose.dev.yml up -d
```

详细文档：[Docker 部署指南](deploy/README.md)

### Kubernetes 部署（集群）

适用于生产环境和高可用场景：

```bash
# 使用 Makefile
make k8s-deploy

# 或使用 kubectl
kubectl apply -k deploy/k8s/

# 检查部署状态
kubectl get pods,svc,ingress -n kube-system -l app=kube-node-manager
```

详细文档：[Kubernetes 部署指南](deploy/k8s/README.md)

### 数据库配置

#### SQLite（默认）
- 适合：开发环境、小规模部署
- 配置：无需额外配置
- 数据文件：`backend/data/kube-node-manager.db`

#### PostgreSQL（推荐生产环境）
- 适合：生产环境、大规模部署
- 配置：参考 `backend/configs/config-postgres.yaml.example`
- 迁移工具：`scripts/sqlite-to-postgres-v3.go`

详细文档：[数据库配置指南](docs/database-configuration-guide.md)

## 📚 文档

### 功能文档

#### Ansible 自动化运维
- [Ansible 工作流 DAG 功能实现](docs/workflow-dag-implementation.md)
- [任务可视化与执行改进](docs/CHANGELOG-task-visualization-improvements.md)

#### 飞书机器人集成
- [飞书机器人批量操作和快捷命令](docs/feishu-bot-batch-and-quick-commands.md)
- [飞书机器人交互式卡片和命令解析](docs/feishu-bot-interactive-and-parser.md)
- [飞书机器人标签和污点管理](docs/feishu-bot-label-taint-implementation.md)
- [飞书机器人会话管理](docs/feishu-bot-session-management.md)
- [飞书机器人优化和性能提升](docs/feishu-bot-optimization-and-performance.md)

#### 其他功能
- [批量操作优化](docs/batch-operations-optimization.md)

### 部署与配置
- [数据库配置指南](docs/database-configuration-guide.md)
- [Docker 部署文档](deploy/README.md)
- [Kubernetes 部署文档](deploy/k8s/README.md)
- [数据库迁移指南](scripts/README_MIGRATION.md)

### 集成与插件
- [kubectl 插件实现](docs/kubectl-plugin-implementation.md)
- [kubectl 插件使用文档](kubectl-plugin/README.md)
- [GitLab Runner 配置](docs/gitlab-runner-configuration.md)
- [GitLab Runner Token 管理](docs/gitlab-runner-token-management.md)
- [GitLab 创建 Runner 指南](docs/gitlab-create-runner-guide.md)

### UI 优化
- [对话框尺寸和搜索修复](docs/dialog-size-and-search-fixes.md)

## 🔄 更新和维护

### 版本更新

```bash
# 1. 备份数据
./scripts/backup.sh

# 2. 拉取最新代码
git pull origin main

# 3. 查看版本变更
cat VERSION
git log --oneline -10

# 4. Docker Compose 更新
docker-compose down
docker-compose pull
docker-compose up --build -d

# 5. Kubernetes 更新
kubectl set image deployment/kube-node-manager \
  kube-node-manager=your-registry/kube-node-manager:v2.27.0 \
  -n kube-system

# 6. 验证更新
docker-compose ps  # 或
kubectl get pods -n kube-system
```

### 数据备份

```bash
# 自动备份脚本
./scripts/backup.sh

# 手动备份 SQLite
cp backend/data/kube-node-manager.db \
   backup/kube-node-manager-$(date +%Y%m%d-%H%M%S).db

# 手动备份 PostgreSQL
pg_dump -h <host> -U <user> <dbname> > backup.sql

# Kubernetes PVC 备份
kubectl cp kube-system/<pod-name>:/data/kube-node-manager.db \
  ./backup/kube-node-manager-$(date +%Y%m%d-%H%M%S).db
```

### 数据恢复

```bash
# SQLite 恢复
cp backup/kube-node-manager-YYYYMMDD.db backend/data/kube-node-manager.db
docker-compose restart backend

# PostgreSQL 恢复
psql -h <host> -U <user> <dbname> < backup.sql
```

### 清理维护

```bash
# 清理 Docker 资源
docker-compose down
docker system prune -a

# 清理日志（保留最近 7 天）
find backend/logs -name "*.log" -mtime +7 -delete

# 清理审计日志（数据库）
# 在应用中执行或直接操作数据库
```

## 🤝 贡献指南

感谢您考虑为 kube-node-manager 做出贡献！

### 贡献方式

1. **报告 Bug**
   - 使用 [Issue Tracker](../../issues)
   - 描述问题和复现步骤
   - 提供相关日志和环境信息

2. **功能建议**
   - 提交 Feature Request
   - 说明使用场景和预期效果
   - 参与讨论和方案设计

3. **代码贡献**
   - Fork 本仓库
   - 创建特性分支
   - 编写代码和测试
   - 提交 Pull Request

### 贡献流程

```bash
# 1. Fork 并克隆
git clone https://github.com/<your-username>/kube-node-manager.git
cd kube-node-manager

# 2. 创建特性分支
git checkout -b feature/amazing-feature

# 3. 进行开发
# 编写代码、测试、文档

# 4. 提交更改
git add .
git commit -m "feat: add amazing feature"

# 5. 推送到远程
git push origin feature/amazing-feature

# 6. 创建 Pull Request
# 在 GitHub 上创建 PR 并描述变更
```

### 代码审查标准

- ✅ 代码符合项目规范
- ✅ 包含必要的测试
- ✅ 通过所有 CI 检查
- ✅ 更新相关文档
- ✅ Commit 消息清晰

### 开发者社区

- 💬 讨论: [GitHub Discussions](../../discussions)
- 📧 邮件: 项目维护者邮箱
- 🤝 贡献者: 感谢所有贡献者！

## 📝 许可证

本项目采用 **MIT 许可证**。

这意味着您可以自由地：
- ✅ 商业使用
- ✅ 修改
- ✅ 分发
- ✅ 私有使用

详情请参见 [LICENSE](LICENSE) 文件。

## 📞 支持与联系

### 获取帮助

- 📖 **文档**: 查看 [docs](docs/) 目录
- 🐛 **Bug 报告**: [提交 Issue](../../issues/new?template=bug_report.md)
- 💡 **功能请求**: [提交 Feature Request](../../issues/new?template=feature_request.md)
- 💬 **讨论**: [GitHub Discussions](../../discussions)

### 项目信息

- **当前版本**: v2.27.0
- **最后更新**: 2025-11-04
- **维护状态**: 🟢 活跃维护中

### 鸣谢

感谢所有为这个项目做出贡献的开发者！

---

<div align="center">

**⚠️ 重要提醒**

在生产环境部署前，请务必：
- 🔐 修改默认管理员密码（`admin` / `admin123`）
- 🔑 更改默认 JWT 密钥
- 🛡️ 启用 HTTPS
- 📊 配置监控和告警
- 💾 设置定期数据备份

**Made with ❤️ by the kube-node-manager team**

[⬆ 回到顶部](#-kubernetes-节点管理器)

</div>