#!/bin/bash

# Kubernetes 节点管理器安装脚本
set -e

echo "🚀 开始安装 Kubernetes 节点管理器..."

# 检查依赖
check_dependencies() {
    echo "📋 检查系统依赖..."
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        echo "❌ Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    echo "✅ 依赖检查通过"
}

# 创建必要目录
create_directories() {
    echo "📁 创建项目目录..."
    
    mkdir -p data
    mkdir -p logs
    mkdir -p configs
    
    echo "✅ 目录创建完成"
}

# 配置环境变量
setup_env() {
    echo "⚙️  配置环境变量..."
    
    if [ ! -f .env ]; then
        cp .env.example .env
        
        # 生成随机JWT密钥
        JWT_SECRET=$(openssl rand -base64 32)
        sed -i.bak "s/your-jwt-secret-change-in-production/${JWT_SECRET}/" .env
        rm -f .env.bak
        
        echo "✅ 环境变量配置完成"
        echo "⚠️  请编辑 .env 文件配置 LDAP 等其他参数"
    else
        echo "✅ .env 文件已存在，跳过创建"
    fi
}

# 检查 Kubernetes 配置
check_kube_config() {
    echo "🔍 检查 Kubernetes 配置..."
    
    if [ ! -f ~/.kube/config ]; then
        echo "⚠️  未找到 Kubernetes 配置文件"
        echo "   请将您的 kubeconfig 文件复制到 ~/.kube/config"
        echo "   或者在启动后通过 Web 界面添加集群"
    else
        echo "✅ Kubernetes 配置文件存在"
    fi
}

# 构建和启动服务
start_services() {
    echo "🏗️  构建和启动服务..."
    
    # 构建单一镜像（多阶段构建）
    echo "正在构建 Docker 镜像（多阶段构建）..."
    docker build -t kube-node-manager:latest .
    
    # 启动服务
    echo "正在启动服务..."
    docker-compose -f deploy/docker/docker-compose.yml up -d
    
    echo "✅ 服务启动完成"
}

# 等待服务就绪
wait_for_services() {
    echo "⏳ 等待服务就绪..."
    
    # 等待应用服务（前后端集成）
    echo "等待应用服务启动..."
    for i in {1..60}; do
        if curl -f http://localhost:8080/api/v1/health > /dev/null 2>&1; then
            echo "✅ 应用服务就绪"
            break
        fi
        sleep 2
        echo -n "."
    done
}

# 显示访问信息
show_access_info() {
    echo ""
    echo "🎉 安装完成！"
    echo ""
    echo "📍 访问地址:"
    echo "   Web界面: http://localhost:8080"
    echo "   API接口: http://localhost:8080/api/v1"
    echo ""
    echo "👤 默认账户:"
    echo "   用户名: admin"
    echo "   密码:   admin123"
    echo ""
    echo "⚠️  重要提醒:"
    echo "   1. 请及时修改默认密码"
    echo "   2. 请配置 .env 文件中的 JWT_SECRET"
    echo "   3. 如需 LDAP 认证，请配置相关参数"
    echo ""
    echo "📚 更多信息请查看 README.md"
}

# 主函数
main() {
    check_dependencies
    create_directories
    setup_env
    check_kube_config
    start_services
    wait_for_services
    show_access_info
}

# 执行安装
main "$@"