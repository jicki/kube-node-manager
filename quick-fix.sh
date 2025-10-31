#!/bin/bash

# 快速修复脚本 - Monaco Editor & 定时任务数据库
# 日期: 2025-10-31

set -e

echo "================================"
echo "  快速修复脚本"
echo "================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查是否在项目根目录
if [ ! -f "VERSION" ]; then
    echo -e "${RED}❌ 错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

VERSION=$(cat VERSION | head -1)
echo -e "${GREEN}当前版本: $VERSION${NC}"
echo ""

# 询问部署方式
echo "请选择部署方式:"
echo "  1) Docker 部署（推荐）"
echo "  2) 仅执行数据库迁移"
echo "  3) 开发环境测试"
echo ""
read -p "请输入选项 [1-3]: " choice

case $choice in
    1)
        echo ""
        echo -e "${YELLOW}📦 开始 Docker 构建...${NC}"
        echo ""
        
        # 清理旧的 statik 文件
        rm -rf backend/statik/statik.go
        
        # 构建镜像
        echo "构建 Docker 镜像..."
        make docker-build
        
        echo ""
        echo -e "${GREEN}✅ Docker 镜像构建完成${NC}"
        echo ""
        
        # 询问是否重启容器
        read -p "是否现在重启容器? [y/N]: " restart
        if [[ $restart =~ ^[Yy]$ ]]; then
            echo ""
            echo "停止容器..."
            docker-compose down
            
            echo "启动容器..."
            docker-compose up -d
            
            echo ""
            echo -e "${GREEN}✅ 容器已重启${NC}"
            echo ""
            echo "查看日志:"
            echo "  docker-compose logs -f kube-node-manager"
        fi
        ;;
        
    2)
        echo ""
        echo -e "${YELLOW}🗄️  执行数据库迁移...${NC}"
        echo ""
        
        cd backend
        go run tools/migrate.go
        
        echo ""
        echo -e "${GREEN}✅ 数据库迁移完成${NC}"
        echo ""
        
        # 询问是否重启应用
        read -p "是否重启应用? [y/N]: " restart
        if [[ $restart =~ ^[Yy]$ ]]; then
            echo ""
            echo "重启应用..."
            docker-compose restart kube-node-manager
            
            echo ""
            echo -e "${GREEN}✅ 应用已重启${NC}"
        fi
        ;;
        
    3)
        echo ""
        echo -e "${YELLOW}🔧 启动开发环境...${NC}"
        echo ""
        
        # 检查依赖
        echo "检查前端依赖..."
        cd frontend
        if [ ! -d "node_modules" ]; then
            echo "安装前端依赖..."
            npm install
        fi
        
        echo ""
        echo -e "${GREEN}前端服务器启动命令:${NC}"
        echo "  cd frontend && npm run dev"
        echo ""
        echo -e "${GREEN}后端服务器启动命令:${NC}"
        echo "  cd backend && go run cmd/main.go"
        echo ""
        echo -e "${YELLOW}请在两个终端窗口分别运行上述命令${NC}"
        ;;
        
    *)
        echo -e "${RED}❌ 无效选项${NC}"
        exit 1
        ;;
esac

echo ""
echo "================================"
echo "  修复完成"
echo "================================"
echo ""
echo -e "${GREEN}📋 验证步骤:${NC}"
echo ""
echo "1. Monaco Editor 验证:"
echo "   - 访问: Ansible → 任务模板 → 创建模板"
echo "   - 检查: Playbook 编辑器应正常显示（深色主题，约 500px 高）"
echo "   - 测试: 可以正常输入和编辑 YAML 代码"
echo ""
echo "2. 定时任务验证:"
echo "   - 访问: Ansible → 定时任务"
echo "   - 检查: 页面正常加载，无错误提示"
echo "   - 测试: 可以创建、编辑、删除定时任务"
echo ""
echo -e "${YELLOW}💡 提示: 如果浏览器仍显示旧版本，请使用 Ctrl+Shift+R (或 Cmd+Shift+R) 强制刷新${NC}"
echo ""
echo -e "${GREEN}📖 详细文档: HOTFIX-2025-10-31.md${NC}"
echo ""

