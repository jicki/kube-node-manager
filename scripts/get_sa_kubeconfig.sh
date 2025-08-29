#!/bin/bash

# simple-kubeconfig.sh
# 为现有 ServiceAccount 生成 KubeConfig 文件

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 显示使用帮助
show_help() {
    cat << EOF
用法: $0 [选项]

选项:
    -n, --name NAME           ServiceAccount 名称 (默认: kube-node-mgr)
    -s, --namespace NAMESPACE 命名空间 (默认: kube-node-mgr)
    -o, --output FILE         输出的 kubeconfig 文件名
    -i, --insecure            跳过 TLS 证书验证 (不安全，仅用于测试)
    -f, --force               强制重新创建 Secret
    -h, --help               显示此帮助信息

示例:
    $0 -n my-reader -s kube-system
    $0 --name app-reader --insecure --output app-config
EOF
}

# 默认参数
SERVICE_ACCOUNT_NAME="kube-node-mgr"
NAMESPACE="kube-node-mgr"
INSECURE=false
FORCE=false
OUTPUT_FILE=""

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            SERVICE_ACCOUNT_NAME="$2"
            shift 2
            ;;
        -s|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -i|--insecure)
            INSECURE=true
            shift
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 设置输出文件名
if [ -z "$OUTPUT_FILE" ]; then
    OUTPUT_FILE="${SERVICE_ACCOUNT_NAME}-kubeconfig"
fi

SECRET_NAME="${SERVICE_ACCOUNT_NAME}-token"

# 获取集群信息
CLUSTER_NAME=$(kubectl config current-context)
SERVER_URL=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')

log_info "配置信息:"
log_info "  ServiceAccount: $SERVICE_ACCOUNT_NAME"
log_info "  命名空间: $NAMESPACE"
log_info "  输出文件: $OUTPUT_FILE"
log_info "  跳过证书验证: $INSECURE"
log_info "  集群: $CLUSTER_NAME"
log_info "  API Server: $SERVER_URL"
echo ""

# 检查 ServiceAccount 是否存在
check_service_account() {
    log_info "检查 ServiceAccount..."
    if ! kubectl get serviceaccount "$SERVICE_ACCOUNT_NAME" -n "$NAMESPACE" >/dev/null 2>&1; then
        log_error "ServiceAccount $SERVICE_ACCOUNT_NAME 在命名空间 $NAMESPACE 中不存在"
        log_info "请先创建 ServiceAccount 或使用其他脚本"
        exit 1
    fi
    log_success "ServiceAccount $SERVICE_ACCOUNT_NAME 存在"
}

# 创建或检查 Secret
handle_secret() {
    log_info "处理 ServiceAccount Token Secret..."
    
    # 检查 Secret 是否存在
    if kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" >/dev/null 2>&1; then
        if [ "$FORCE" = true ]; then
            log_warning "Secret 已存在，强制重新创建..."
            kubectl delete secret "$SECRET_NAME" -n "$NAMESPACE"
        else
            log_info "Secret $SECRET_NAME 已存在"
            # 检查是否有有效的 token
            if kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.token}' | base64 --decode >/dev/null 2>&1; then
                local token_length=$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.token}' | base64 --decode | wc -c)
                if [ "$token_length" -gt 10 ]; then
                    log_success "Secret 包含有效的 token"
                    return 0
                fi
            fi
            log_warning "Secret 存在但 token 无效，重新创建..."
            kubectl delete secret "$SECRET_NAME" -n "$NAMESPACE"
        fi
    fi
    
    # 创建新的 Secret
    log_info "创建 ServiceAccount Token Secret..."
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: $SECRET_NAME
  namespace: $NAMESPACE
  annotations:
    kubernetes.io/service-account.name: $SERVICE_ACCOUNT_NAME
type: kubernetes.io/service-account-token
EOF
    
    log_success "Secret $SECRET_NAME 创建成功"
    
    # 等待 token 生成
    log_info "等待 token 生成..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        local token_data=$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.token}' 2>/dev/null || echo "")
        if [ -n "$token_data" ]; then
            local token=$(echo "$token_data" | base64 --decode 2>/dev/null || echo "")
            if [ ${#token} -gt 10 ]; then
                log_success "Token 生成完成 (尝试 $attempt/$max_attempts)"
                return 0
            fi
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo ""
    log_error "等待 token 生成超时"
    exit 1
}

# 生成 KubeConfig
generate_kubeconfig() {
    log_info "生成 KubeConfig 文件..."
    
    # 获取 Token
    local token=$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.token}' | base64 --decode)
    if [ -z "$token" ] || [ ${#token} -lt 10 ]; then
        log_error "无法获取有效的 token"
        exit 1
    fi
    
    # 生成 KubeConfig 文件
    if [ "$INSECURE" = true ]; then
        log_warning "生成不安全的 KubeConfig (跳过证书验证)"
        cat > "$OUTPUT_FILE" <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: $SERVER_URL
    insecure-skip-tls-verify: true
  name: $CLUSTER_NAME
contexts:
- context:
    cluster: $CLUSTER_NAME
    user: $SERVICE_ACCOUNT_NAME
    namespace: $NAMESPACE
  name: ${SERVICE_ACCOUNT_NAME}@${CLUSTER_NAME}
current-context: ${SERVICE_ACCOUNT_NAME}@${CLUSTER_NAME}
users:
- name: $SERVICE_ACCOUNT_NAME
  user:
    token: $token
EOF
    else
        # 获取 CA 证书
        local ca_cert=$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.ca\.crt}')
        if [ -z "$ca_cert" ]; then
            log_error "无法获取 CA 证书数据"
            exit 1
        fi
        
        cat > "$OUTPUT_FILE" <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: $ca_cert
    server: $SERVER_URL
  name: $CLUSTER_NAME
contexts:
- context:
    cluster: $CLUSTER_NAME
    user: $SERVICE_ACCOUNT_NAME
    namespace: $NAMESPACE
  name: ${SERVICE_ACCOUNT_NAME}@${CLUSTER_NAME}
current-context: ${SERVICE_ACCOUNT_NAME}@${CLUSTER_NAME}
users:
- name: $SERVICE_ACCOUNT_NAME
  user:
    token: $token
EOF
    fi
    
    log_success "KubeConfig 文件已生成: $OUTPUT_FILE"
}

# 测试生成的 KubeConfig
test_kubeconfig() {
    log_info "测试生成的 KubeConfig..."
    
    # 测试基本连接
    echo -n "  集群连接测试: "
    if kubectl --kubeconfig="$OUTPUT_FILE" cluster-info >/dev/null 2>&1; then
        echo -e "${GREEN}✓ 通过${NC}"
    else
        echo -e "${RED}✗ 失败${NC}"
        log_warning "如果是证书问题，可以使用 -i 参数重新生成"
        return 1
    fi
    
    # 测试权限
    echo -n "  基本权限测试: "
    if kubectl --kubeconfig="$OUTPUT_FILE" auth can-i get serviceaccounts >/dev/null 2>&1; then
        echo -e "${GREEN}✓ 有权限${NC}"
    else
        echo -e "${YELLOW}? 权限受限${NC}"
    fi
    
    # 尝试列出一些资源
    echo -n "  资源访问测试: "
    if kubectl --kubeconfig="$OUTPUT_FILE" get ns --no-headers >/dev/null 2>&1; then
        local ns_count=$(kubectl --kubeconfig="$OUTPUT_FILE" get ns --no-headers 2>/dev/null | wc -l)
        echo -e "${GREEN}✓ 成功 (可访问 $ns_count 个命名空间)${NC}"
    else
        echo -e "${YELLOW}? 访问受限${NC}"
    fi
    
    log_success "测试完成"
}

# 显示使用说明
show_usage() {
    echo ""
    log_info "使用方法:"
    echo "  # 测试连接"
    echo "  kubectl --kubeconfig=$OUTPUT_FILE cluster-info"
    echo ""
    echo "  # 查看权限"
    echo "  kubectl --kubeconfig=$OUTPUT_FILE auth can-i --list"
    echo ""
    echo "  # 列出 PVC (如果有权限)"
    echo "  kubectl --kubeconfig=$OUTPUT_FILE get pvc --all-namespaces"
    echo ""
    log_info "在 Go 程序中使用:"
    echo '  config, err := clientcmd.BuildConfigFromFlags("", "'$OUTPUT_FILE'")'
}

# 主函数
main() {
    log_info "开始为 ServiceAccount 生成 KubeConfig..."
    
    check_service_account
    handle_secret
    generate_kubeconfig
    
    echo ""
    if test_kubeconfig; then
        show_usage
        echo ""
        log_success "KubeConfig 生成成功！"
    else
        echo ""
        log_warning "KubeConfig 已生成但测试失败，可能需要检查权限或网络连接"
        show_usage
    fi
}

# 执行主函数
main
