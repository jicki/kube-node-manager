#!/bin/bash
#
# Kubernetes 环境数据库迁移脚本
# 用途：在 K8s 环境中执行数据库迁移
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# 配置
NAMESPACE="${NAMESPACE:-kube-node-mgr}"
STATEFULSET="${STATEFULSET:-kube-node-mgr}"
POD_NAME="${POD_NAME:-${STATEFULSET}-0}"

# 检查 kubectl
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl 未安装"
        exit 1
    fi
    print_success "kubectl 已安装"
}

# 检查 Pod 状态
check_pod() {
    print_header "检查 Pod 状态"
    
    if ! kubectl get pod "$POD_NAME" -n "$NAMESPACE" &>/dev/null; then
        print_error "Pod $POD_NAME 不存在"
        print_info "可用的 Pods:"
        kubectl get pods -n "$NAMESPACE" -l app=kube-node-mgr
        exit 1
    fi
    
    local status=$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.phase}')
    if [ "$status" != "Running" ]; then
        print_error "Pod $POD_NAME 状态不是 Running (当前: $status)"
        exit 1
    fi
    
    print_success "Pod $POD_NAME 正在运行"
}

# 方案1: 在运行的 Pod 中执行迁移
run_in_pod() {
    print_header "方案1: 在运行的 Pod 中执行迁移"
    
    print_info "连接到 Pod: $POD_NAME"
    
    print_info "执行迁移命令..."
    kubectl exec -it "$POD_NAME" -n "$NAMESPACE" -- sh -c '
        echo "Current directory: $(pwd)"
        echo "Checking migration tool..."
        
        if [ -f "/app/backend/tools/migrate" ]; then
            echo "Using compiled migration tool"
            /app/backend/tools/migrate
        elif [ -f "/app/tools/migrate.go" ]; then
            echo "Using Go source migration tool"
            cd /app && go run tools/migrate.go
        else
            echo "ERROR: Migration tool not found"
            exit 1
        fi
    '
    
    if [ $? -eq 0 ]; then
        print_success "迁移执行成功"
    else
        print_error "迁移执行失败"
        exit 1
    fi
}

# 方案2: 使用 Job 执行迁移
run_with_job() {
    print_header "方案2: 使用 Kubernetes Job 执行迁移"
    
    local job_file="deploy/k8s/migration-job-sqlite.yaml"
    
    if [ ! -f "$job_file" ]; then
        print_error "Job 配置文件不存在: $job_file"
        exit 1
    fi
    
    print_info "删除旧的 Job (如果存在)..."
    kubectl delete job kube-node-mgr-migration -n "$NAMESPACE" --ignore-not-found=true
    
    print_info "创建迁移 Job..."
    kubectl apply -f "$job_file"
    
    print_info "等待 Job 完成..."
    kubectl wait --for=condition=complete job/kube-node-mgr-migration -n "$NAMESPACE" --timeout=120s
    
    print_info "查看 Job 日志..."
    kubectl logs -n "$NAMESPACE" job/kube-node-mgr-migration --tail=50
    
    print_success "迁移 Job 执行完成"
}

# 方案3: 重启 StatefulSet 触发自动迁移
restart_statefulset() {
    print_header "方案3: 重启 StatefulSet 触发自动迁移"
    
    print_warning "这将重启 StatefulSet，应用会短暂不可用"
    read -p "确认继续？(y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "已取消"
        exit 0
    fi
    
    print_info "重启 StatefulSet..."
    kubectl rollout restart statefulset "$STATEFULSET" -n "$NAMESPACE"
    
    print_info "等待重启完成..."
    kubectl rollout status statefulset "$STATEFULSET" -n "$NAMESPACE" --timeout=180s
    
    print_success "StatefulSet 重启完成"
    
    print_info "查看迁移日志..."
    sleep 5
    kubectl logs "$POD_NAME" -n "$NAMESPACE" --tail=50 | grep -i "migration\|migrate\|table"
}

# 查看数据库状态
check_database() {
    print_header "检查数据库状态"
    
    print_info "查询数据库表..."
    kubectl exec "$POD_NAME" -n "$NAMESPACE" -- sh -c '
        if [ -f "/app/data/kube-node-manager.db" ]; then
            echo "=== SQLite 数据库 ==="
            sqlite3 /app/data/kube-node-manager.db ".tables"
            echo ""
            echo "=== anomaly_report_configs 表 ==="
            sqlite3 /app/data/kube-node-manager.db "SELECT COUNT(*) as count FROM anomaly_report_configs;"
        elif command -v psql &> /dev/null; then
            echo "=== PostgreSQL 数据库 ==="
            psql -h $DATABASE_HOST -U $DATABASE_USER -d $DATABASE_NAME -c "\dt"
            echo ""
            echo "=== anomaly_report_configs 表 ==="
            psql -h $DATABASE_HOST -U $DATABASE_USER -d $DATABASE_NAME -c "SELECT COUNT(*) FROM anomaly_report_configs;"
        else
            echo "无法确定数据库类型"
        fi
    ' 2>/dev/null || print_warning "无法连接数据库"
}

# 显示使用帮助
show_help() {
    cat << EOF
Kubernetes 环境数据库迁移脚本

用法:
  $0 [选项] [方案]

方案:
  pod         在运行的 Pod 中执行迁移（推荐）
  job         使用 Kubernetes Job 执行迁移
  restart     重启 StatefulSet 触发自动迁移
  check       检查数据库状态

选项:
  -n, --namespace   指定 namespace（默认: kube-node-mgr）
  -s, --statefulset 指定 StatefulSet 名称（默认: kube-node-mgr）
  -p, --pod         指定 Pod 名称（默认: kube-node-mgr-0）
  -h, --help        显示帮助信息

示例:
  # 在默认 Pod 中执行迁移
  $0 pod
  
  # 指定 namespace 执行
  $0 -n my-namespace pod
  
  # 使用 Job 执行迁移
  $0 job
  
  # 检查数据库状态
  $0 check

环境变量:
  NAMESPACE    默认 namespace
  STATEFULSET  默认 StatefulSet 名称
  POD_NAME     默认 Pod 名称
EOF
}

# 解析参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -s|--statefulset)
                STATEFULSET="$2"
                POD_NAME="${2}-0"
                shift 2
                ;;
            -p|--pod)
                POD_NAME="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            pod|job|restart|check)
                METHOD="$1"
                shift
                ;;
            *)
                print_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 主函数
main() {
    print_header "Kubernetes 数据库迁移工具"
    
    # 检查环境
    check_kubectl
    check_pod
    
    # 如果没有指定方法，显示菜单
    if [ -z "$METHOD" ]; then
        echo "请选择迁移方案:"
        echo "  1) 在运行的 Pod 中执行（推荐，快速）"
        echo "  2) 使用 Kubernetes Job 执行"
        echo "  3) 重启 StatefulSet 触发自动迁移"
        echo "  4) 检查数据库状态"
        echo ""
        read -p "请选择 [1-4]: " choice
        
        case $choice in
            1) METHOD="pod" ;;
            2) METHOD="job" ;;
            3) METHOD="restart" ;;
            4) METHOD="check" ;;
            *)
                print_error "无效选择"
                exit 1
                ;;
        esac
    fi
    
    # 执行对应方案
    case "$METHOD" in
        pod)
            run_in_pod
            ;;
        job)
            run_with_job
            ;;
        restart)
            restart_statefulset
            ;;
        check)
            check_database
            exit 0
            ;;
        *)
            print_error "未知方案: $METHOD"
            show_help
            exit 1
            ;;
    esac
    
    # 检查结果
    echo ""
    check_database
    
    print_success "操作完成！"
}

# 解析参数并执行
parse_args "$@"
main

