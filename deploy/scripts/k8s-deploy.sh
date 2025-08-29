#!/bin/bash

# Kubernetes 部署脚本
set -e

NAMESPACE=${NAMESPACE:-kube-node-mgr}
DOMAIN=${DOMAIN:-kube-node-mgr.example.com}
IMAGE_TAG=${IMAGE_TAG:-latest}

echo "🚀 开始部署 kube-node-mgr 到 Kubernetes..."

# 检查 kubectl 命令
check_kubectl() {
    echo "📋 检查 kubectl..."
    if ! command -v kubectl &> /dev/null; then
        echo "❌ kubectl 未安装，请先安装 kubectl"
        exit 1
    fi
    
    # 检查集群连接
    if ! kubectl cluster-info &> /dev/null; then
        echo "❌ 无法连接到 Kubernetes 集群"
        exit 1
    fi
    
    echo "✅ kubectl 检查通过"
}

# 创建命名空间
create_namespace() {
    echo "📁 创建命名空间..."
    kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
    echo "✅ 命名空间 ${NAMESPACE} 准备就绪"
}

# 生成并应用 Secret
generate_secrets() {
    echo "🔑 生成 Secret..."
    
    # 生成 JWT Secret
    JWT_SECRET=$(openssl rand -base64 32)
    JWT_SECRET_B64=$(echo -n "${JWT_SECRET}" | base64)
    
    # 更新 StatefulSet 配置中的 Secret
    sed -i.bak "s/jwt-secret: .*/jwt-secret: ${JWT_SECRET_B64}/" deploy/k8s/k8s-statefulset.yaml
    rm -f deploy/k8s/k8s-statefulset.yaml.bak
    
    echo "✅ Secret 配置完成"
}

# 更新镜像标签和域名
update_config() {
    echo "⚙️  更新配置..."
    
    # 更新 Kustomization 中的镜像标签
    sed -i.bak "s/newTag: .*/newTag: ${IMAGE_TAG}/" deploy/k8s/kustomization.yaml
    rm -f deploy/k8s/kustomization.yaml.bak
    
    # 更新 Ingress 域名
    sed -i.bak "s/kube-node-mgr.example.com/${DOMAIN}/g" deploy/k8s/k8s-ingress.yaml
    rm -f deploy/k8s/k8s-ingress.yaml.bak
    
    echo "✅ 配置更新完成"
}

# 部署应用
deploy_app() {
    echo "🏗️  部署应用..."
    
    # 使用 kustomize 部署
    kubectl apply -k deploy/k8s/ -n ${NAMESPACE}
    
    echo "✅ 应用部署完成"
}

# 等待部署就绪
wait_for_deployment() {
    echo "⏳ 等待 Pod 就绪..."
    
    # 等待 StatefulSet 就绪
    kubectl wait --for=condition=ready pod -l app=kube-node-mgr -n ${NAMESPACE} --timeout=300s
    
    echo "✅ Pod 已就绪"
}

# 显示部署信息
show_deployment_info() {
    echo ""
    echo "🎉 部署完成！"
    echo ""
    echo "📍 部署信息:"
    echo "   命名空间: ${NAMESPACE}"
    echo "   域名: ${DOMAIN}"
    echo "   镜像标签: ${IMAGE_TAG}"
    echo ""
    
    # 显示 Pod 状态
    echo "📊 Pod 状态:"
    kubectl get pods -l app=kube-node-mgr -n ${NAMESPACE}
    
    echo ""
    echo "📊 Service 状态:"
    kubectl get svc -l app=kube-node-mgr -n ${NAMESPACE}
    
    echo ""
    echo "📊 Ingress 状态:"
    kubectl get ingress kube-node-mgr -n ${NAMESPACE} 2>/dev/null || echo "   Ingress 未配置"
    
    echo ""
    echo "🔗 访问地址:"
    if [ "${DOMAIN}" != "kube-node-mgr.example.com" ]; then
        echo "   https://${DOMAIN}"
    else
        echo "   请配置域名或使用 Port Forward:"
        echo "   kubectl port-forward svc/kube-node-mgr 8080:80 -n ${NAMESPACE}"
        echo "   然后访问: http://localhost:8080"
    fi
    
    echo ""
    echo "📚 管理命令:"
    echo "   查看日志: kubectl logs -l app=kube-node-mgr -n ${NAMESPACE} -f"
    echo "   重启应用: kubectl rollout restart statefulset/kube-node-mgr -n ${NAMESPACE}"
    echo "   删除应用: kubectl delete -k deploy/k8s/ -n ${NAMESPACE}"
}

# 主函数
main() {
    echo "使用参数:"
    echo "  NAMESPACE=${NAMESPACE}"
    echo "  DOMAIN=${DOMAIN}"
    echo "  IMAGE_TAG=${IMAGE_TAG}"
    echo ""
    
    check_kubectl
    create_namespace
    generate_secrets
    update_config
    deploy_app
    wait_for_deployment
    show_deployment_info
}

# 执行部署
main "$@"