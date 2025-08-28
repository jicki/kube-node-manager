# Kubernetes 节点管理器 Makefile

.PHONY: help dev build start stop clean install test docker-build docker-push

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
	@echo "构建应用..."
	docker build -t kube-node-manager:latest .

build-compose: ## 使用docker-compose构建
	@echo "使用docker-compose构建..."
	docker-compose -f deploy/docker/docker-compose.yml build

build-backend: ## 构建后端
	@echo "构建后端..."
	cd backend && go build -o main cmd/main.go

build-frontend: ## 构建前端
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
docker-build: ## 构建 Docker 镜像（多阶段构建）
	@echo "构建 Docker 镜像..."
	docker build -t kube-node-manager:latest .

docker-build-dev: ## 构建开发环境镜像
	@echo "构建开发环境镜像..."
	docker build -t kube-node-manager/backend:dev -f backend/Dockerfile.dev backend/
	docker build -t kube-node-manager/frontend:dev -f frontend/Dockerfile.dev frontend/

docker-push: ## 推送 Docker 镜像
	@echo "推送 Docker 镜像..."
	docker push kube-node-manager:latest

docker-tag: ## 给镜像打标签
	@echo "给镜像打标签..."
	@if [ -z "$(TAG)" ]; then echo "错误: 请指定标签，例如: make docker-tag TAG=v1.0.0"; exit 1; fi
	docker tag kube-node-manager:latest kube-node-manager:$(TAG)

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
	@echo "Git 提交: $$(git rev-parse HEAD)"
	@echo "Git 分支: $$(git branch --show-current)"
	@echo "构建时间: $$(date)"

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

# 默认变量
BACKUP ?= ""
VERSION ?= ""
REPLICAS ?= ""

# 帮助信息
.DEFAULT_GOAL := help