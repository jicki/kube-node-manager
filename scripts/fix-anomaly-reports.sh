#!/bin/bash
#
# 异常报告配置修复脚本
# 用途：自动检测和修复异常报告配置相关的数据库问题
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

# 检查是否在项目根目录
check_project_root() {
    if [ ! -f "backend/cmd/main.go" ]; then
        print_error "请在项目根目录运行此脚本"
        exit 1
    fi
}

# 检测部署方式
detect_deployment() {
    print_header "检测部署方式"
    
    if kubectl get namespace kube-node-mgr &>/dev/null; then
        echo "kubernetes"
        print_info "检测到 Kubernetes 部署"
        return 0
    elif docker ps | grep -q kube-node-manager; then
        echo "docker"
        print_info "检测到 Docker 部署"
        return 0
    elif pgrep -f "kube-node-manager" &>/dev/null; then
        echo "local"
        print_info "检测到本地进程"
        return 0
    else
        echo "none"
        print_warning "未检测到正在运行的应用"
        return 1
    fi
}

# 检查 SQLite 数据库
check_sqlite_db() {
    print_header "检查 SQLite 数据库"
    
    local db_file="backend/data/kube-node-manager.db"
    
    if [ ! -f "$db_file" ]; then
        print_warning "SQLite 数据库文件不存在: $db_file"
        return 1
    fi
    
    print_success "找到数据库文件: $db_file"
    
    # 检查表是否存在
    local tables=$(sqlite3 "$db_file" ".tables" 2>/dev/null || echo "")
    
    if echo "$tables" | grep -q "anomaly_report_configs"; then
        print_success "表 'anomaly_report_configs' 已存在"
        
        # 检查记录数
        local count=$(sqlite3 "$db_file" "SELECT COUNT(*) FROM anomaly_report_configs;" 2>/dev/null || echo "0")
        print_info "当前记录数: $count"
        
        if [ "$count" -eq 0 ]; then
            print_info "表存在但为空，这是正常的（还未创建任何配置）"
        fi
        
        return 0
    else
        print_error "表 'anomaly_report_configs' 不存在"
        return 1
    fi
}

# 运行数据库迁移
run_migration() {
    print_header "运行数据库迁移"
    
    if [ ! -f "backend/cmd/migrate.go" ]; then
        print_error "迁移脚本不存在，请先运行 Cursor AI 创建迁移脚本"
        return 1
    fi
    
    print_info "开始执行数据库迁移..."
    cd backend
    
    if go run cmd/migrate.go; then
        print_success "数据库迁移完成"
        cd ..
        return 0
    else
        print_error "数据库迁移失败"
        cd ..
        return 1
    fi
}

# 重启 Kubernetes 应用
restart_kubernetes() {
    print_header "重启 Kubernetes 应用"
    
    print_info "重启 StatefulSet..."
    kubectl rollout restart statefulset kube-node-mgr -n kube-node-mgr
    
    print_info "等待重启完成..."
    kubectl rollout status statefulset kube-node-mgr -n kube-node-mgr --timeout=120s
    
    print_success "应用重启完成"
    
    # 显示最新日志
    print_info "查看最新日志（按 Ctrl+C 退出）..."
    sleep 2
    kubectl logs -f statefulset/kube-node-mgr -n kube-node-mgr --tail=50 | grep -i "migration\|anomaly\|database" || true
}

# 重启 Docker 应用
restart_docker() {
    print_header "重启 Docker 应用"
    
    print_info "重启容器..."
    docker-compose restart kube-node-manager
    
    print_success "应用重启完成"
    
    # 显示最新日志
    print_info "查看最新日志（按 Ctrl+C 退出）..."
    sleep 2
    docker-compose logs -f kube-node-manager --tail=50 | grep -i "migration\|anomaly\|database" || true
}

# 验证修复结果
verify_fix() {
    print_header "验证修复结果"
    
    print_info "等待服务就绪..."
    sleep 5
    
    # 尝试访问 API
    local api_url="http://localhost:8080/api/v1/health"
    
    if curl -s "$api_url" &>/dev/null; then
        print_success "API 服务正常"
        
        # 检查报告配置 API
        local report_api="http://localhost:8080/api/v1/anomaly-reports/configs"
        print_info "测试报告配置 API..."
        
        # 这里需要认证，所以可能失败，但我们检查返回码
        local status_code=$(curl -s -o /dev/null -w "%{http_code}" "$report_api" || echo "000")
        
        if [ "$status_code" = "401" ] || [ "$status_code" = "200" ]; then
            print_success "报告配置 API 可访问（状态码: $status_code）"
            print_info "401 是正常的（需要认证），请登录后使用前端测试"
        else
            print_warning "报告配置 API 返回异常状态码: $status_code"
        fi
    else
        print_warning "无法访问 API 服务，可能应用还在启动中"
    fi
    
    echo ""
    print_success "修复完成！"
    echo ""
    print_info "请按以下步骤验证："
    echo "  1. 打开浏览器访问应用"
    echo "  2. 进入 '系统管理' -> '异常分析' -> '报告配置'"
    echo "  3. 点击 '新增报告' 按钮"
    echo "  4. 检查 '目标集群' 下拉框是否有数据"
    echo "  5. 创建一个测试报告配置"
    echo ""
}

# 主函数
main() {
    print_header "异常报告配置修复工具"
    
    # 检查项目根目录
    check_project_root
    
    # 检测部署方式
    deployment_type=$(detect_deployment)
    
    # 检查数据库状态
    if check_sqlite_db; then
        print_info "数据库表已存在，无需修复"
        
        read -p "是否仍要重启应用？(y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "已取消"
            exit 0
        fi
    else
        print_warning "数据库表不存在或损坏，需要执行修复"
        
        # 运行迁移
        if ! run_migration; then
            print_error "迁移失败，退出"
            exit 1
        fi
    fi
    
    # 根据部署方式重启应用
    case "$deployment_type" in
        kubernetes)
            restart_kubernetes
            ;;
        docker)
            restart_docker
            ;;
        local)
            print_info "检测到本地进程，请手动停止并重启应用"
            print_info "或者直接运行迁移脚本即可："
            print_info "  cd backend && go run cmd/migrate.go"
            ;;
        none)
            print_warning "未检测到运行中的应用"
            print_info "数据库迁移已完成，请启动应用"
            ;;
    esac
    
    # 验证修复结果
    if [ "$deployment_type" != "none" ]; then
        verify_fix
    fi
}

# 执行主函数
main "$@"

