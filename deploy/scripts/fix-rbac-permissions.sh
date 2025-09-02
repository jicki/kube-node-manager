#!/bin/bash

# RBAC权限修复脚本
# 修复节点封锁功能所需的Pod权限问题
# 
# 使用方法: ./fix-rbac-permissions.sh [namespace]
# 默认命名空间: kube-node-mgr

set -e

NAMESPACE=${1:-kube-node-mgr}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="$(dirname "$SCRIPT_DIR")/k8s"

echo "🔧 修复 kube-node-manager RBAC权限问题"
echo "命名空间: $NAMESPACE"
echo ""

# 检查kubectl是否可用
if ! command -v kubectl &> /dev/null; then
    echo "❌ 错误: kubectl 命令未找到，请确保已安装kubectl"
    exit 1
fi

# 检查集群连接
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ 错误: 无法连接到Kubernetes集群"
    exit 1
fi

echo "📝 应用RBAC权限补丁..."
kubectl apply -f "$K8S_DIR/rbac-patch.yaml"

if [ $? -eq 0 ]; then
    echo "✅ RBAC权限更新成功"
else
    echo "❌ RBAC权限更新失败"
    exit 1
fi

echo ""
echo "🔄 重启应用Pod以应用新权限..."

# 获取Pod名称
POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l app=kube-node-mgr -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -z "$POD_NAME" ]; then
    echo "⚠️  警告: 未找到kube-node-mgr Pod，请手动重启应用"
    echo "   命令: kubectl rollout restart statefulset/kube-node-mgr -n $NAMESPACE"
else
    echo "重启Pod: $POD_NAME"
    kubectl delete pod "$POD_NAME" -n "$NAMESPACE"
    
    echo "等待Pod重新启动..."
    kubectl wait --for=condition=ready pod -l app=kube-node-mgr -n "$NAMESPACE" --timeout=120s
    
    if [ $? -eq 0 ]; then
        echo "✅ Pod重启成功"
    else
        echo "⚠️  警告: Pod重启超时，请检查Pod状态"
    fi
fi

echo ""
echo "🔍 验证权限配置..."

# 检查ClusterRole
if kubectl get clusterrole kube-node-mgr &> /dev/null; then
    echo "✅ ClusterRole 存在"
    
    # 检查pods权限
    if kubectl get clusterrole kube-node-mgr -o yaml | grep -A 10 -B 2 'resources.*pods' &> /dev/null; then
        echo "✅ Pod相关权限已配置"
    else
        echo "❌ Pod相关权限配置不完整"
    fi
else
    echo "❌ ClusterRole 不存在"
fi

# 检查ClusterRoleBinding
if kubectl get clusterrolebinding kube-node-mgr &> /dev/null; then
    echo "✅ ClusterRoleBinding 存在"
else
    echo "❌ ClusterRoleBinding 不存在"
fi

echo ""
echo "🎉 RBAC权限修复完成！"
echo ""
echo "现在可以测试节点封锁功能："
echo "1. 登录kube-node-manager Web界面"
echo "2. 进入节点管理页面"
echo "3. 尝试封锁一个测试节点"
echo ""
echo "如果仍有问题，请检查应用日志："
echo "kubectl logs -l app=kube-node-mgr -n $NAMESPACE -f"
