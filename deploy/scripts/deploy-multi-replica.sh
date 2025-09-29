#!/bin/bash

# 多副本部署脚本 - 解决进度条卡住问题
# 此脚本启用数据库模式的进度服务，支持多副本环境

set -e

NAMESPACE="kube-node-mgr"
APP_NAME="kube-node-mgr"

echo "🚀 开始部署 Kube Node Manager 多副本版本..."
echo "📝 此版本启用数据库模式，解决多副本环境下的进度条卡住问题"

# 检查kubectl命令
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl 命令未找到，请先安装 kubectl"
    exit 1
fi

# 创建命名空间
echo "📦 创建命名空间: $NAMESPACE"
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# 检查是否有旧的单副本部署
if kubectl get statefulset $APP_NAME -n $NAMESPACE &> /dev/null; then
    echo "⚠️  检测到已存在的部署"
    read -p "是否要删除现有部署并重新创建？ (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🗑️  删除现有部署..."
        kubectl delete statefulset $APP_NAME -n $NAMESPACE --force --grace-period=0
        echo "⏳ 等待Pod终止..."
        kubectl wait --for=delete pod -l app=$APP_NAME -n $NAMESPACE --timeout=60s || true
    else
        echo "❌ 部署取消"
        exit 1
    fi
fi

# 部署应用
echo "🚢 部署多副本版本..."
kubectl apply -f deploy/k8s/k8s-multi-replica.yaml

# 等待StatefulSet就绪
echo "⏳ 等待StatefulSet就绪..."
kubectl rollout status statefulset/$APP_NAME -n $NAMESPACE --timeout=300s

# 检查Pod状态
echo "📊 检查Pod状态..."
kubectl get pods -n $NAMESPACE -l app=$APP_NAME

# 检查服务状态
echo "🔗 检查服务状态..."
kubectl get service $APP_NAME -n $NAMESPACE

# 显示访问信息
echo ""
echo "✅ 部署完成！"
echo ""
echo "📋 部署信息："
echo "   - 副本数: 2"
echo "   - 数据库模式: 已启用"
echo "   - 会话亲和性: 已启用"
echo ""
echo "🔧 重要配置："
echo "   - PROGRESS_ENABLE_DATABASE=true (解决多副本进度同步问题)"
echo "   - sessionAffinity=ClientIP (WebSocket连接粘性)"
echo ""
echo "📱 访问应用："

# 获取服务访问方式
if kubectl get ingress $APP_NAME -n $NAMESPACE &> /dev/null; then
    INGRESS_HOST=$(kubectl get ingress $APP_NAME -n $NAMESPACE -o jsonpath='{.spec.rules[0].host}')
    echo "   - Ingress: http://$INGRESS_HOST"
elif kubectl get service $APP_NAME -n $NAMESPACE -o jsonpath='{.spec.type}' | grep -q LoadBalancer; then
    echo "   - LoadBalancer: 等待外部IP分配..."
    kubectl get service $APP_NAME -n $NAMESPACE
else
    echo "   - 端口转发: kubectl port-forward svc/$APP_NAME -n $NAMESPACE 8080:80"
fi

echo ""
echo "🔍 查看日志:"
echo "   kubectl logs -f statefulset/$APP_NAME -n $NAMESPACE"
echo ""
echo "📊 监控进度服务:"
echo "   kubectl logs -f statefulset/$APP_NAME -n $NAMESPACE | grep -E 'database|progress|WebSocket'"
echo ""
echo "✨ 多副本部署完成！进度条卡住问题已解决。"