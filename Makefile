# Kubernetes 节点管理器 Makefile

# ================================
# 默认配置变量
# ================================

# 备份文件名（用于数据备份恢复）
BACKUP ?= ""

# 版本号配置（从VERSION文件读取，可通过环境变量或命令行覆盖）
ifneq ($(VERSION),)
    # 使用命令行或环境变量提供的版本
    VERSION_TAG := $(VERSION)
else ifeq ($(wildcard VERSION),VERSION)
    # 从VERSION文件读取版本号
    VERSION_TAG := $(shell cat VERSION 2>/dev/null | tr -d '\n' | tr -d ' ')
else
    # 使用git标签或默认版本
    VERSION_TAG := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
endif

# 确保版本标签不为空
ifeq ($(strip $(VERSION_TAG)),)
    VERSION_TAG := dev
endif

# Pod副本数（用于Kubernetes扩缩容）
REPLICAS ?= ""

# 镜像仓库前缀配置，可通过环境变量或命令行覆盖
# 示例: harbor.example.com/project, docker.io/username, registry.local/namespace
ifeq ($(strip $(REGISTRY)),)
    REGISTRY := reg.deeproute.ai/deeproute-public/zzh
endif

# 前端构建时环境变量配置
# VITE_API_BASE_URL: 前端API基础URL，留空使用相对路径
VITE_API_BASE_URL ?= 
# VITE_ENABLE_LDAP: 是否启用LDAP登录功能
VITE_ENABLE_LDAP ?= false

# ================================
# Makefile 配置
# ================================

.PHONY: help dev build start stop clean install test docker-build docker-build-only docker-push version update-version build-plugin install-plugin

# 默认目标
help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# 开发环境
dev: ## 启动开发环境
	@echo "启动开发环境..."
	docker-compose -f deploy/docker/docker-compose.dev.yml up --build

dev-backend: ## 启动后端开发服务器
	@echo "启动后端开发服务器..."
	cd backend && go run cmd/main.go

dev-frontend: ## 启动前端开发服务器
	@echo "启动前端开发服务器..."
	cd frontend && npm run dev

# 构建
build: ## 构建应用（多阶段构建单一镜像）
	@echo "构建应用 [版本: $(VERSION_TAG)]..."
	@echo "清理 statik 文件..."
	rm -rf backend/statik/statik.go
	docker build \
		--build-arg VITE_API_BASE_URL="$(VITE_API_BASE_URL)" \
		--build-arg VITE_ENABLE_LDAP="$(VITE_ENABLE_LDAP)" \
		-t $(REGISTRY)/kube-node-manager:$(VERSION_TAG) .
	docker tag $(REGISTRY)/kube-node-manager:$(VERSION_TAG) $(REGISTRY)/kube-node-manager:latest

build-compose: ## 使用docker-compose构建
	@echo "使用docker-compose构建..."
	docker-compose -f deploy/docker/docker-compose.yml build

build-backend: ## 构建后端
	@echo "构建后端..."
	cd backend && go build -o main cmd/main.go

build-frontend: ## 构建前端
	@echo "清理 statik 文件..."
	rm -rf backend/statik/statik.go
	@echo "构建前端..."
	cd frontend && npm run build

build-statik: ## 生成静态文件嵌入代码
	@echo "生成静态文件嵌入代码..."
	cd frontend && npm run build
	cd backend && statik -src=../frontend/dist -dest=. -f

# 服务管理
start: ## 启动所有服务
	@echo "启动服务..."
	docker-compose -f deploy/docker/docker-compose.yml up -d

stop: ## 停止所有服务
	@echo "停止服务..."
	docker-compose -f deploy/docker/docker-compose.yml down

restart: ## 重启服务
	@echo "重启服务..."
	docker-compose -f deploy/docker/docker-compose.yml restart

logs: ## 查看日志
	docker-compose -f deploy/docker/docker-compose.yml logs -f

status: ## 查看服务状态
	docker-compose -f deploy/docker/docker-compose.yml ps

# 安装和初始化
install: ## 运行安装脚本
	@echo "运行安装脚本..."
	./deploy/scripts/install.sh

init-db: ## 初始化数据库
	@echo "初始化数据库..."
	cd backend && go run cmd/main.go migrate

seed-data: ## 填充种子数据
	@echo "填充种子数据..."
	cd backend && go run cmd/main.go seed

# 测试
test: ## 运行所有测试
	@echo "运行测试..."
	$(MAKE) test-backend
	$(MAKE) test-frontend

test-backend: ## 运行后端测试
	@echo "运行后端测试..."
	cd backend && go test ./...

test-frontend: ## 运行前端测试
	@echo "运行前端测试..."
	cd frontend && npm run test

lint: ## 代码检查
	@echo "运行代码检查..."
	$(MAKE) lint-backend
	$(MAKE) lint-frontend

lint-backend: ## 后端代码检查
	@echo "后端代码检查..."
	cd backend && go fmt ./...
	cd backend && go vet ./...

lint-frontend: ## 前端代码检查
	@echo "前端代码检查..."
	cd frontend && npm run lint

# Docker 相关
docker-build: ## 构建 Docker 镜像（多阶段构建）并推送
	@echo "清理 statik 文件..."
	rm -rf backend/statik/statik.go
	@echo "构建 Docker 镜像 [版本: $(VERSION_TAG)]..."
	docker build \
		--build-arg VITE_API_BASE_URL="$(VITE_API_BASE_URL)" \
		--build-arg VITE_ENABLE_LDAP="$(VITE_ENABLE_LDAP)" \
		--build-arg CACHEBUST=$$(date +%s) \
		-t $(REGISTRY)/kube-node-manager:$(VERSION_TAG) .
	docker tag $(REGISTRY)/kube-node-manager:$(VERSION_TAG) $(REGISTRY)/kube-node-manager:latest
	@echo "镜像构建成功，开始推送..."
	@$(MAKE) docker-push

docker-build-no-cache: ## 强制无缓存构建 Docker 镜像（用于前端更新）并推送
	@echo "⚠️  警告: 使用无缓存构建，将花费较长时间（约10分钟）"
	@echo "清理 statik 文件..."
	rm -rf backend/statik/statik.go
	@echo "强制无缓存构建 Docker 镜像 [版本: $(VERSION_TAG)]..."
	docker build --no-cache \
		--build-arg VITE_API_BASE_URL="$(VITE_API_BASE_URL)" \
		--build-arg VITE_ENABLE_LDAP="$(VITE_ENABLE_LDAP)" \
		-t $(REGISTRY)/kube-node-manager:$(VERSION_TAG) .
	docker tag $(REGISTRY)/kube-node-manager:$(VERSION_TAG) $(REGISTRY)/kube-node-manager:latest
	@echo "镜像构建成功，开始推送..."
	@$(MAKE) docker-push
	@echo "✅ 无缓存构建完成！前端代码已完全更新。"

docker-build-only: ## 只构建 Docker 镜像，不推送
	@echo "清理 statik 文件..."
	rm -rf backend/statik/statik.go
	@echo "构建 Docker 镜像 [版本: $(VERSION_TAG)]..."
	docker build \
		--build-arg VITE_API_BASE_URL="$(VITE_API_BASE_URL)" \
		--build-arg VITE_ENABLE_LDAP="$(VITE_ENABLE_LDAP)" \
		--build-arg CACHEBUST=$$(date +%s) \
		-t $(REGISTRY)/kube-node-manager:$(VERSION_TAG) .
	docker tag $(REGISTRY)/kube-node-manager:$(VERSION_TAG) $(REGISTRY)/kube-node-manager:latest
	@echo "镜像构建完成（未推送）"

docker-build-only-no-cache: ## 只构建 Docker 镜像（无缓存），不推送
	@echo "⚠️  警告: 使用无缓存构建，将花费较长时间"
	@echo "清理 statik 文件..."
	rm -rf backend/statik/statik.go
	@echo "强制无缓存构建 Docker 镜像 [版本: $(VERSION_TAG)]..."
	docker build --no-cache \
		--build-arg VITE_API_BASE_URL="$(VITE_API_BASE_URL)" \
		--build-arg VITE_ENABLE_LDAP="$(VITE_ENABLE_LDAP)" \
		-t $(REGISTRY)/kube-node-manager:$(VERSION_TAG) .
	docker tag $(REGISTRY)/kube-node-manager:$(VERSION_TAG) $(REGISTRY)/kube-node-manager:latest
	@echo "✅ 无缓存构建完成（未推送）"

docker-build-dev: ## 构建开发环境镜像
	@echo "清理 statik 文件..."
	rm -rf backend/statik/statik.go
	@echo "构建开发环境镜像 [版本: $(VERSION_TAG)]..."
	docker build -t $(REGISTRY)/kube-node-manager/backend:$(VERSION_TAG)-dev -f backend/Dockerfile.dev backend/
	docker build -t $(REGISTRY)/kube-node-manager/frontend:$(VERSION_TAG)-dev -f frontend/Dockerfile.dev frontend/
	docker tag $(REGISTRY)/kube-node-manager/backend:$(VERSION_TAG)-dev $(REGISTRY)/kube-node-manager/backend:dev
	docker tag $(REGISTRY)/kube-node-manager/frontend:$(VERSION_TAG)-dev $(REGISTRY)/kube-node-manager/frontend:dev

docker-push: ## 推送 Docker 镜像
	@echo "推送 Docker 镜像 [版本: $(VERSION_TAG)]..."
	docker push $(REGISTRY)/kube-node-manager:$(VERSION_TAG)
	docker push $(REGISTRY)/kube-node-manager:latest

docker-tag: ## 给镜像打标签
	@echo "给镜像打标签 [源版本: $(VERSION_TAG)]..."
	@if [ -z "$(TAG)" ]; then echo "错误: 请指定标签，例如: make docker-tag TAG=v1.0.0"; exit 1; fi
	docker tag $(REGISTRY)/kube-node-manager:$(VERSION_TAG) $(REGISTRY)/kube-node-manager:$(TAG)

docker-push-dev: ## 推送开发环境镜像
	@echo "推送开发环境镜像 [版本: $(VERSION_TAG)]..."
	docker push $(REGISTRY)/kube-node-manager/backend:$(VERSION_TAG)-dev
	docker push $(REGISTRY)/kube-node-manager/frontend:$(VERSION_TAG)-dev
	docker push $(REGISTRY)/kube-node-manager/backend:dev
	docker push $(REGISTRY)/kube-node-manager/frontend:dev

docker-config: ## 显示Docker构建配置信息
	@echo "Docker构建配置:"
	@echo "  镜像仓库: $(REGISTRY)"
	@echo "  版本标签: $(VERSION_TAG)"
	@echo "  前端API地址: $(VITE_API_BASE_URL)"
	@echo "  启用LDAP: $(VITE_ENABLE_LDAP)"
	@echo ""
	@echo "完整镜像名: $(REGISTRY)/kube-node-manager:$(VERSION_TAG)"
	@echo ""
	@echo "自定义构建示例:"
	@echo "  make docker-build-only VITE_API_BASE_URL=https://api.example.com"
	@echo "  make docker-build VITE_ENABLE_LDAP=true REGISTRY=your.registry.com"
	@echo "  export VITE_API_BASE_URL=https://prod-api.example.com && make build"

docker-registry: ## 显示镜像仓库配置信息
	@echo "镜像仓库配置信息:"
	@REGISTRY_VAR="$(strip $(REGISTRY))"; \
	VERSION_VAR="$(VERSION_TAG)"; \
	DEFAULT_REGISTRY="reg.deeproute.ai/deeproute-public/zzh"; \
	echo "  当前仓库前缀: $$REGISTRY_VAR"; \
	echo "  当前版本标签: $$VERSION_VAR"; \
	echo "  完整镜像名: $$REGISTRY_VAR/kube-node-manager:$$VERSION_VAR"; \
	echo "  Latest 镜像名: $$REGISTRY_VAR/kube-node-manager:latest"; \
	echo ""; \
	if [ "$$REGISTRY_VAR" = "$$DEFAULT_REGISTRY" ]; then \
		echo "正在使用默认仓库配置"; \
		echo ""; \
		echo "自定义仓库示例:"; \
		echo "  make docker-build REGISTRY=harbor.example.com/project"; \
		echo "  make docker-push REGISTRY=your.registry.com/namespace"; \
		echo "  export REGISTRY=registry.local/myproject && make docker-build"; \
	else \
		echo "正在使用自定义仓库配置"; \
		echo ""; \
		echo "恢复默认配置:"; \
		echo "  unset REGISTRY && make docker-build"; \
	fi

docker-clean: ## 清理 Docker 镜像和容器
	@echo "清理 Docker..."
	docker-compose -f deploy/docker/docker-compose.yml down -v
	docker image prune -f
	docker container prune -f

# 数据管理
backup: ## 备份数据
	@echo "备份数据..."
	./deploy/scripts/backup.sh

restore: ## 恢复数据 (需要指定备份文件)
	@echo "恢复数据..."
	@echo "使用方法: make restore BACKUP=backup_file"
	@if [ -z "$(BACKUP)" ]; then echo "错误: 请指定备份文件"; exit 1; fi
	cp backups/$(BACKUP).db data/kube-node-manager.db

# 清理
clean: ## 清理构建文件
	@echo "清理构建文件..."
	rm -rf backend/main
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	docker-compose -f deploy/docker/docker-compose.yml down -v

clean-all: clean docker-clean ## 清理所有文件（包括Docker）

# 部署
deploy: ## 部署到生产环境
	@echo "部署到生产环境..."
	docker-compose -f deploy/docker/docker-compose.prod.yml up -d --build

deploy-update: ## 更新生产环境
	@echo "更新生产环境..."
	git pull
	$(MAKE) backup
	docker-compose -f deploy/docker/docker-compose.prod.yml up -d --build

# 维护
health-check: ## 健康检查
	@echo "执行健康检查..."
	@curl -f http://localhost:8080/api/v1/health || echo "后端服务异常"
	@curl -f http://localhost:3000/health || echo "前端服务异常"

update-deps: ## 更新依赖
	@echo "更新后端依赖..."
	cd backend && go mod tidy
	@echo "更新前端依赖..."
	cd frontend && npm update

# 版本管理
version: ## 显示版本信息
	@echo "项目版本信息:"
	@echo "当前版本: $(VERSION_TAG)"
	@if [ -f VERSION ]; then echo "VERSION文件: $$(cat VERSION)"; fi
	@echo "Git 提交: $$(git rev-parse HEAD 2>/dev/null || echo 'N/A')"
	@echo "Git 分支: $$(git branch --show-current 2>/dev/null || echo 'N/A')"
	@echo "Git 标签: $$(git describe --tags --always 2>/dev/null || echo 'N/A')"
	@echo "构建时间: $$(date)"

update-version: ## 更新VERSION文件
	@if [ -z "$(VERSION)" ]; then echo "错误: 请指定版本号，例如: make update-version VERSION=v1.0.1"; exit 1; fi
	@echo "更新版本到: $(VERSION)"
	@echo "$(VERSION)" > VERSION
	@echo "VERSION文件已更新"

release: ## 创建发布版本
	@echo "创建发布版本..."
	@if [ -z "$(VERSION)" ]; then echo "错误: 请指定版本号，例如: make release VERSION=v1.0.0"; exit 1; fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "版本 $(VERSION) 已创建并推送"

# 监控
monitor: ## 启动监控服务
	@echo "启动监控服务..."
	docker-compose -f deploy/docker/docker-compose.monitoring.yml up -d

# Kubernetes 部署
k8s-deploy: ## 部署到 Kubernetes
	@echo "部署到 Kubernetes..."
	kubectl apply -k deploy/k8s/

k8s-delete: ## 从 Kubernetes 删除
	@echo "从 Kubernetes 删除..."
	kubectl delete -k deploy/k8s/

k8s-status: ## 查看 Kubernetes 部署状态
	@echo "查看 Kubernetes 部署状态..."
	kubectl get pods,svc,ingress -l app=kube-node-manager

k8s-logs: ## 查看 Kubernetes Pod 日志
	@echo "查看 Pod 日志..."
	kubectl logs -l app=kube-node-manager -f --tail=100

k8s-restart: ## 重启 Kubernetes Pod
	@echo "重启 Pod..."
	kubectl rollout restart statefulset/kube-node-manager

k8s-scale: ## 扩缩容 Pod (需要指定副本数)
	@echo "扩缩容 Pod..."
	@if [ -z "$(REPLICAS)" ]; then echo "错误: 请指定副本数，例如: make k8s-scale REPLICAS=3"; exit 1; fi
	kubectl scale statefulset/kube-node-manager --replicas=$(REPLICAS)

k8s-update: ## 更新 Kubernetes 部署（使用新版本镜像）
	@echo "更新 Kubernetes 部署 [版本: $(VERSION_TAG)]..."
	kubectl set image statefulset/kube-node-mgr kube-node-mgr=$(REGISTRY)/kube-node-manager:$(VERSION_TAG) -n kube-node-mgr
	kubectl rollout status statefulset/kube-node-mgr -n kube-node-mgr

k8s-deploy-full: docker-build-no-cache k8s-update ## 🚀 完整部署流程（无缓存构建 + 推送 + K8s更新）
	@echo ""
	@echo "✅ 完整部署流程完成！"
	@echo ""
	@echo "📝 请完成以下浏览器端验证步骤："
	@echo "  1. 清除浏览器缓存（Ctrl+Shift+Delete）"
	@echo "  2. 强制刷新页面（Ctrl+F5 或 Cmd+Shift+R）"
	@echo "  3. 或使用无痕模式打开"
	@echo "  4. 检查报告配置页面是否显示数据"
	@echo ""
	@echo "🔍 快速验证命令："
	@echo "  # 查看 Pod 日志"
	@echo "  kubectl logs -f kube-node-mgr-0 -n kube-node-mgr"
	@echo ""
	@echo "  # 查看 Pod 状态"
	@echo "  kubectl get pods -n kube-node-mgr"
	@echo ""
	@echo "  # 测试 API"
	@echo "  kubectl exec kube-node-mgr-0 -n kube-node-mgr -- wget -qO- http://localhost:8080/api/v1/anomaly-reports/configs"
	@echo ""

# kubectl 插件相关
build-plugin: ## 构建 kubectl 插件
	@echo "构建 kubectl 插件..."
	cd kubectl-plugin && make build

install-plugin: ## 安装 kubectl 插件
	@echo "安装 kubectl 插件..."
	cd kubectl-plugin && make install

install-plugin-user: ## 安装 kubectl 插件到用户目录
	@echo "安装 kubectl 插件到用户目录..."
	cd kubectl-plugin && make install-user

uninstall-plugin: ## 卸载 kubectl 插件
	@echo "卸载 kubectl 插件..."
	cd kubectl-plugin && make uninstall

test-plugin: ## 测试 kubectl 插件
	@echo "测试 kubectl 插件..."
	cd kubectl-plugin && make test

clean-plugin: ## 清理 kubectl 插件构建文件
	@echo "清理 kubectl 插件构建文件..."
	cd kubectl-plugin && make clean

# 帮助信息
.DEFAULT_GOAL := help