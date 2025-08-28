#!/bin/bash

# Kubernetes 清理脚本
set -e

NAMESPACE=${NAMESPACE:-default}

echo "🗑️  开始清理 kube-node-manager Kubernetes 资源..."

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

# 显示将要删除的资源
show_resources() {
    echo "📊 将要删除的资源:"
    echo ""
    
    echo "Pod:"
    kubectl get pods -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null || echo "  无"
    
    echo ""
    echo "StatefulSet:"
    kubectl get statefulset -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null || echo "  无"
    
    echo ""
    echo "Service:"
    kubectl get svc -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null || echo "  无"
    
    echo ""
    echo "Ingress:"
    kubectl get ingress -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null || echo "  无"
    
    echo ""
    echo "PVC:"
    kubectl get pvc -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null || echo "  无"
    
    echo ""
    echo "Secret:"
    kubectl get secret kube-node-manager-secret -n ${NAMESPACE} 2>/dev/null || echo "  无"
    kubectl get secret kube-node-manager-kubeconfig -n ${NAMESPACE} 2>/dev/null || echo "  无"
    
    echo ""
    echo "ConfigMap:"
    kubectl get configmap kube-node-manager-config -n ${NAMESPACE} 2>/dev/null || echo "  无"
}

# 确认删除
confirm_deletion() {
    echo ""
    read -p "❓ 确定要删除这些资源吗？这将删除所有数据！(y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "❌ 取消删除"
        exit 1
    fi
}

# 删除应用资源
delete_app_resources() {
    echo "🗑️  删除应用资源..."
    
    # 使用 kustomize 删除
    if kubectl delete -k deploy/k8s/ -n ${NAMESPACE} 2>/dev/null; then
        echo "✅ 应用资源删除完成"
    else
        echo "⚠️  应用资源删除失败或不存在"
    fi
}

# 删除 PVC（可选）
delete_persistent_volumes() {
    echo ""
    read -p "❓ 是否删除持久化数据（PVC）？这将永久删除数据！(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🗑️  删除 PVC..."
        kubectl delete pvc -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null || echo "⚠️  PVC 删除失败或不存在"
        echo "✅ PVC 删除完成"
    else
        echo "ℹ️  保留 PVC，数据将被保留"
    fi
}

# 删除 RBAC 资源
delete_rbac_resources() {
    echo "🗑️  删除 RBAC 资源..."
    
    # 删除 ClusterRoleBinding
    kubectl delete clusterrolebinding kube-node-manager 2>/dev/null || echo "⚠️  ClusterRoleBinding 不存在"
    
    # 删除 ClusterRole
    kubectl delete clusterrole kube-node-manager 2>/dev/null || echo "⚠️  ClusterRole 不存在"
    
    echo "✅ RBAC 资源清理完成"
}

# 验证清理结果
verify_cleanup() {
    echo "🔍 验证清理结果..."
    
    # 检查是否还有相关资源
    REMAINING_RESOURCES=""
    
    if kubectl get pods -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null | grep -q kube-node-manager; then
        REMAINING_RESOURCES="${REMAINING_RESOURCES}\n  - Pod"
    fi
    
    if kubectl get statefulset -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null | grep -q kube-node-manager; then
        REMAINING_RESOURCES="${REMAINING_RESOURCES}\n  - StatefulSet"
    fi
    
    if kubectl get svc -l app=kube-node-manager -n ${NAMESPACE} 2>/dev/null | grep -q kube-node-manager; then
        REMAINING_RESOURCES="${REMAINING_RESOURCES}\n  - Service"
    fi
    
    if [ -n "${REMAINING_RESOURCES}" ]; then
        echo "⚠️  以下资源可能仍然存在:${REMAINING_RESOURCES}"
        echo "   这可能是正常的，某些资源可能需要时间来完全删除"
    else
        echo "✅ 所有应用资源已成功删除"
    fi
}

# 显示清理完成信息
show_cleanup_info() {
    echo ""
    echo "🎉 清理完成！"
    echo ""
    echo "📍 清理摘要:"
    echo "   命名空间: ${NAMESPACE}"
    echo "   应用资源: 已删除"
    echo "   RBAC 资源: 已删除"
    echo ""
    echo "💡 提示:"
    echo "   如果需要重新部署，请运行:"
    echo "   ./deploy/scripts/k8s-deploy.sh"
    echo ""
    echo "   如果要删除整个命名空间:"
    echo "   kubectl delete namespace ${NAMESPACE}"
}

# 主函数
main() {
    echo "使用参数:"
    echo "  NAMESPACE=${NAMESPACE}"
    echo ""
    
    check_kubectl
    show_resources
    confirm_deletion
    delete_app_resources
    delete_persistent_volumes
    delete_rbac_resources
    verify_cleanup
    show_cleanup_info
}

# 执行清理
main "$@"