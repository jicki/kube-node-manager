# kubectl-nodemanager 插件 Makefile

# 版本信息
VERSION := v1.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# 构建参数
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# 二进制文件名
BINARY_NAME := kubectl-node_mgr

# 默认目标
.PHONY: all
all: build

# 构建二进制文件
.PHONY: build
build:
	@echo "构建 $(BINARY_NAME)..."
	@CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) main.go

# 构建多平台二进制文件
.PHONY: build-all
build-all:
	@echo "构建多平台二进制文件..."
	@mkdir -p dist
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 main.go
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 main.go
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 main.go
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 main.go
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe main.go

# 安装插件到本地
.PHONY: install
install: build
	@echo "安装 $(BINARY_NAME) 到 /usr/local/bin/..."
	@sudo mv $(BINARY_NAME) /usr/local/bin/
	@echo "安装完成！可以使用 'kubectl node_mgr --help' 来验证安装"

# 安装到用户本地 bin 目录
.PHONY: install-user
install-user: build
	@echo "安装 $(BINARY_NAME) 到 ~/bin/..."
	@mkdir -p ~/bin
	@mv $(BINARY_NAME) ~/bin/
	@echo "安装完成！请确保 ~/bin 在您的 PATH 中"
	@echo "可以使用 'kubectl node_mgr --help' 来验证安装"

# 卸载插件
.PHONY: uninstall
uninstall:
	@echo "卸载 $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@rm -f ~/bin/$(BINARY_NAME)
	@echo "卸载完成！"

# 清理构建文件
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
	@echo "清理完成！"

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	@go test ./...

# 检查代码格式
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	@go fmt ./...

# 代码检查
.PHONY: vet
vet:
	@echo "代码检查..."
	@go vet ./...

# 下载依赖
.PHONY: deps
deps:
	@echo "下载依赖..."
	@go mod download
	@go mod tidy

# 显示版本信息
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# 帮助信息
.PHONY: help
help:
	@echo "可用的 make 目标："
	@echo "  build        - 构建二进制文件"
	@echo "  build-all    - 构建多平台二进制文件"
	@echo "  install      - 安装到 /usr/local/bin/"
	@echo "  install-user - 安装到 ~/bin/"
	@echo "  uninstall    - 卸载插件"
	@echo "  clean        - 清理构建文件"
	@echo "  test         - 运行测试"
	@echo "  fmt          - 格式化代码"
	@echo "  vet          - 代码检查"
	@echo "  deps         - 下载依赖"
	@echo "  version      - 显示版本信息"
	@echo "  help         - 显示此帮助信息"
